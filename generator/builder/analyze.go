// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"go.thesmos.sh/testkit/generator"
)

// Analyze produces a [Data] from the loaded package and the
// user-supplied list of struct type names.
//
// The defaults factory lookup runs in two phases — first against
// the source package, then against the test package the generator
// emits into. The two-phase lookup solves the chicken-and-egg
// where the factory typically lives next to the generated builder
// (sibling test pkg), so on first generation the test pkg may not
// even exist yet; the source-pkg fallback covers callers that
// declare the factory inline with the type.
func Analyze(pkg *generator.Package, args []string, cfg generator.Config, opts generator.Options) (*Data, error) {
	if len(args) == 0 {
		return nil, generator.Errorf(token.Position{}, "builder: no types specified")
	}

	ctx, err := generator.BuildOutputCtx(pkg, cfg, opts)
	if err != nil {
		return nil, err
	}
	tracker := ctx.Tracker
	qualifier := tracker.AddPath(pkg.Path())

	structs := make([]StructData, 0, len(args))
	for _, name := range args {
		sd, err := analyzeStruct(pkg, name, qualifier, tracker, opts)
		if err != nil {
			return nil, err
		}
		structs = append(structs, sd)
	}

	return &Data{
		PackageName:    ctx.PackageName,
		Imports:        tracker.Imports(),
		Structs:        structs,
		ImplImportPath: ctx.OutputImportPath,
	}, nil
}

// analyzeStruct produces the full [StructData] for a single named
// type. Returns an error from [Package.Struct] if the type is
// missing or isn't a struct — those failures should already have
// been caught by the pipeline's KindStruct validation, but we
// re-check defensively.
func analyzeStruct(
	pkg *generator.Package, name, qualifier string,
	tracker *generator.ImportTracker, opts generator.Options,
) (StructData, error) {
	s, err := pkg.Struct(name)
	if err != nil {
		return StructData{}, err
	}

	typeParamDecl := s.TypeParamDecl(tracker)
	typeParamArgs := s.TypeParamArgs()
	isGeneric := typeParamDecl != ""

	qualifiedType := generator.QualifyType(qualifier, name) + typeParamArgs
	testTypeArgs := ""
	testQualifiedType := qualifiedType
	var typeParamMap map[string]generator.ConcreteType
	if isGeneric {
		testTypeArgs = defaultTestTypeArgs(s.TypeParams)
		testQualifiedType = generator.QualifyType(qualifier, name) + testTypeArgs
		typeParamMap = buildTypeParamMap(s.TypeParams)
	}

	sd := StructData{
		Name:                name,
		BuilderName:         name + "Builder",
		QualifiedType:       qualifiedType,
		HasUnexportedFields: generator.HasUnexportedFields(s.Type),
		TypeParamDecl:       typeParamDecl,
		TypeParamArgs:       typeParamArgs,
		IsGeneric:           isGeneric,
		TestTypeArgs:        testTypeArgs,
		TestQualifiedType:   testQualifiedType,
	}

	resolveDefaultsFactory(pkg, &sd, qualifier, opts)

	for i := range s.Fields {
		f := &s.Fields[i]
		if !f.Exported {
			continue
		}
		fd := analyzeField(f, tracker, isGeneric, typeParamMap)
		applyFieldDirective(pkg, name, &sd, &fd)
		sd.Fields = append(sd.Fields, fd)
	}

	sort.Slice(sd.Fields, func(i, j int) bool {
		return sd.Fields[i].Name < sd.Fields[j].Name
	})
	return sd, nil
}

// analyzeField classifies one field into its shape and renders the
// strings every template branch needs. Field-shape detection is
// mutually exclusive across IsSlice / IsMap / IsBytes / IsStruct
// (Go's underlying-type taxonomy is partition-shaped); IsPointer
// is computed from the un-underlied type so it can co-occur with
// e.g. `*[]string` (pointer to slice).
func analyzeField(
	f *generator.FieldInfo, tracker *generator.ImportTracker,
	isGeneric bool, typeParamMap map[string]generator.ConcreteType,
) FieldData {
	fd := FieldData{
		Name:        f.Name,
		TypeStr:     types.TypeString(f.Type, tracker.Qualifier()),
		SampleValue: generator.SampleValueOf(f.Type, f.Name, tracker),
	}
	switch ut := f.Type.Underlying().(type) {
	case *types.Slice:
		elemStr := types.TypeString(ut.Elem(), tracker.Qualifier())
		if elemStr == "byte" {
			fd.IsBytes = true
		} else {
			fd.IsSlice = true
			fd.ElemTypeStr = elemStr
		}
	case *types.Map:
		fd.IsMap = true
		fd.MapKeyTypeStr = types.TypeString(ut.Key(), tracker.Qualifier())
		fd.MapValTypeStr = types.TypeString(ut.Elem(), tracker.Qualifier())
		fd.MapKeySample = generator.SampleValueOf(ut.Key(), "Key", tracker)
		fd.MapValSample = generator.SampleValueOf(ut.Elem(), "Val", tracker)
	case *types.Struct:
		fd.IsStruct = true
	case *types.Basic:
		// Basic-comparable kinds drive FirstComparableField
		// selection: their non-zero samples produce a meaningful
		// `!=` assertion in Mutate / Clone subtests. Interface,
		// func, and chan kinds reach the Underlying() default
		// branch and stay un-flagged, so generated subtests
		// correctly skip them.
		fd.IsBasicComparable = isComparableBasicKind(ut.Kind())
	}
	if _, ok := f.Type.(*types.Pointer); ok {
		fd.IsPointer = true
	}
	if isGeneric && typeParamMap != nil {
		fd.TestTypeStr = resolveTypeStr(fd.TypeStr, typeParamMap)
		fd.TestSample = sampleForResolvedType(fd.TestTypeStr, fd.Name)
		// For generic map fields, map-key and map-val samples must
		// resolve through the type-parameter map too — otherwise
		// the generic test path emits `WithEntriesEntry(nil, nil)`
		// against a `map[K]V` whose K/V are unresolved type
		// parameters.
		if fd.IsMap {
			rk := resolveTypeStr(fd.MapKeyTypeStr, typeParamMap)
			rv := resolveTypeStr(fd.MapValTypeStr, typeParamMap)
			fd.TestMapKeySample = sampleForResolvedType(rk, "Key")
			fd.TestMapValSample = sampleForResolvedType(rv, "Val")
		}
		// A type-parameter-typed field whose test instantiation
		// resolves to a basic-comparable concrete type IS basic-
		// comparable in the test view — its non-zero sample drives
		// a meaningful `!=` assertion. Without this, generic
		// structs whose only "scalar" fields are type parameters
		// (Pair[A, B any]) lose their Mutate / Clone subtests
		// because FirstComparableField returns nil.
		if !fd.IsBasicComparable && isResolvedBasicComparable(fd.TestTypeStr) {
			fd.IsBasicComparable = true
		}
	}
	return fd
}

// applyFieldDirective walks the field's //testkit: directives and,
// when a `default` directive is present and the struct doesn't
// already use a factory, records the directive's argument as the
// field's literal default. Field directives are skipped entirely
// when HasDefaults is true — the factory wins.
func applyFieldDirective(pkg *generator.Package, structName string, sd *StructData, fd *FieldData) {
	if sd.HasDefaults {
		return
	}
	for _, d := range pkg.FieldDirectives(structName, fd.Name) {
		if d.Name == "default" && len(d.Args) > 0 {
			fd.DefaultValue = d.Args[0]
			sd.HasFieldDefaults = true
		}
	}
}

// resolveDefaultsFactory looks for a `<Type>Defaults() <Type>`
// factory function. The lookup runs against the source package
// first, then (if absent) the sibling output package — handling
// the typical case where the factory lives next to the generated
// builder.
//
// When the factory lives in the source package but the output
// lands in a sibling package, qualifier prefixes the call so the
// emitted code references it cross-package; same-package matches
// (sibling-pkg branch, or source==output) leave it bare.
//
// On first generation the output package may not exist yet; that's
// not an error, just "no factory found". The function silently
// falls through to the directive-based or zero-seed mechanisms.
func resolveDefaultsFactory(
	pkg *generator.Package, sd *StructData, qualifier string,
	opts generator.Options,
) {
	defaultsFunc := sd.Name + "Defaults"
	check := generator.DefaultsFuncSig(sd.Name)
	if generator.HasFunc(pkg, defaultsFunc, check) {
		sd.HasDefaults = true
		// Source-pkg factory: bake the source qualifier into the
		// call expression so the impl renders correctly. Test view
		// consumes the same form (the test pkg also imports source).
		sd.DefaultsFunc = generator.QualifyType(qualifier, defaultsFunc)
		return
	}
	if outputPkg := tryLoadOutputPackage(opts); outputPkg != nil {
		if generator.HasFunc(outputPkg, defaultsFunc, check) {
			sd.HasDefaults = true
			// Output-pkg factory: bare for impl (same package).
			// Test view's Transform prepends GenQualifier so the
			// test file's external _test pkg can reach it.
			sd.DefaultsFunc = defaultsFunc
			sd.DefaultsFromOutput = true
		}
	}
}

// tryLoadOutputPackage attempts to load the package the generator
// emits into. Returns nil for any failure mode — package not yet
// existing on first generation is the load-bearing case, not an
// error to propagate.
func tryLoadOutputPackage(opts generator.Options) *generator.Package {
	if opts.Output == "" || opts.WorkDir == "" {
		return nil
	}
	outputDir := filepath.Dir(opts.Output)
	if outputDir == "." {
		// Same directory as source — already covered by the
		// source-pkg lookup; loading would be redundant.
		return nil
	}
	loader := generator.NewLoader()
	pkg, err := loader.Load(".", filepath.Join(opts.WorkDir, outputDir))
	if err != nil {
		return nil
	}
	return pkg
}

// concreteFor picks a concrete instantiation for one type
// parameter at position idx. Constraint-aware: walks
// [generator.DefaultConcreteTypes] from the rotated start so any
// `any`/`comparable`-constrained position gets a distinct candidate
// while narrow constraints (Numeric, ~int64, etc.) fall through to
// the first satisfying candidate.
//
// Falls back to the round-robin candidate when no candidate
// satisfies — better to render a possibly-wrong concrete type and
// let the Go compiler complain than emit a placeholder the user
// has to chase down.
func concreteFor(p generator.TypeParamInfo, idx int) generator.ConcreteType {
	if ct := generator.SelectConcreteType(p.Constraint, generator.DefaultConcreteTypes, idx); ct != nil {
		return *ct
	}
	n := len(generator.DefaultConcreteTypes)
	return generator.DefaultConcreteTypes[((idx%n)+n)%n]
}

// defaultTestTypeArgs renders the concrete-instantiation suffix
// for the test file (e.g. `[string]`, `[string, int]`,
// `[int]` for Stat[T Numeric]). Each position is selected
// independently via [concreteFor] so constraint-driven types win
// over position-driven defaults.
func defaultTestTypeArgs(params []generator.TypeParamInfo) string {
	names := make([]string, len(params))
	for i, p := range params {
		names[i] = concreteFor(p, i).Name
	}
	return "[" + strings.Join(names, ", ") + "]"
}

// buildTypeParamMap maps each type parameter name to the concrete
// type chosen for it (used by [resolveTypeStr] to substitute "T"
// → "string" / "K" → "int" when rendering field types in the
// test view) plus the per-field Sample function so analyzeField
// can produce constraint-correct sample literals.
func buildTypeParamMap(params []generator.TypeParamInfo) map[string]generator.ConcreteType {
	m := make(map[string]generator.ConcreteType, len(params))
	for i, p := range params {
		m[p.Name] = concreteFor(p, i)
	}
	return m
}

// resolveTypeStr replaces every type-parameter name in typeStr
// with its concrete type. "T" → "string", "[]T" → "[]string",
// "map[K]V" → "map[string]int" when paramMap maps {K:string,
// V:int}. Naive textual replacement — adequate for built-in basic
// names but won't survive a type parameter named "int" or similar
// shadowing accident (rare; user error).
func resolveTypeStr(typeStr string, paramMap map[string]generator.ConcreteType) string {
	out := typeStr
	for param, concrete := range paramMap {
		out = strings.ReplaceAll(out, param, concrete.Name)
	}
	return out
}

// isResolvedBasicComparable reports whether typeStr names a known
// basic-comparable concrete type (i.e. matches one of the names in
// [generator.DefaultConcreteTypes]). Used by analyzeField to
// propagate IsBasicComparable through type-parameter resolution.
func isResolvedBasicComparable(typeStr string) bool {
	for _, c := range generator.DefaultConcreteTypes {
		if c.Name == typeStr {
			return true
		}
	}
	return false
}

// sampleForResolvedType produces a sample literal for the rendered
// concrete typeStr. Walks [generator.DefaultConcreteTypes] for an
// exact-name match and uses that candidate's Sample. Falls through
// to a typed-zero-literal expression when the type isn't a known
// basic (e.g. resolved to a slice/map of basics, or an unknown
// custom name).
func sampleForResolvedType(typeStr, fieldName string) string {
	for _, c := range generator.DefaultConcreteTypes {
		if c.Name == typeStr {
			return c.Sample(fieldName)
		}
	}
	if elemType, ok := strings.CutPrefix(typeStr, "[]"); ok {
		return typeStr + "{" + sampleForResolvedType(elemType, fieldName) + "}"
	}
	return typeStr + "{}"
}

// isComparableBasicKind reports whether kind is one of the basic
// types whose non-zero samples drive a meaningful `!=` assertion
// in the generated Mutate / Clone independence subtests.
//
// Excluded: untyped kinds (not reachable as field types), complex
// (rare as fixture fields with `!=`-assertable samples), and
// UnsafePointer (samples would be non-trivial). Strings,
// signed/unsigned integers, booleans, and floats are in.
func isComparableBasicKind(kind types.BasicKind) bool {
	switch kind {
	case types.String,
		types.Bool,
		types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr,
		types.Float32, types.Float64:
		// types.Rune and types.Byte are aliases for Int32 / Uint8
		// and are covered by the integer cases above.
		return true
	default:
		return false
	}
}
