// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go/token"
	"go/types"
	"sort"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
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
	if isGeneric {
		testTypeArgs = generator.TestTypeArgs(s.TypeParams)
		testQualifiedType = generator.QualifyType(qualifier, name) + testTypeArgs
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
		fd := analyzeField(f, tracker, isGeneric, s.TypeParams)
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
	isGeneric bool, params []generator.TypeParamInfo,
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
		fd.IsBasicComparable = generator.IsBasicComparableKind(ut.Kind())
	}
	if _, ok := f.Type.(*types.Pointer); ok {
		fd.IsPointer = true
	}
	if isGeneric && len(params) > 0 {
		fd.TestTypeStr = generator.SubstituteTypeParams(fd.TypeStr, params)
		fd.TestSample = generator.SampleForConcreteType(fd.TestTypeStr, fd.Name)
		// For generic map fields, map-key and map-val samples must
		// resolve through the type-parameter map too — otherwise
		// the generic test path emits `WithEntriesEntry(nil, nil)`
		// against a `map[K]V` whose K/V are unresolved type
		// parameters.
		if fd.IsMap {
			rk := generator.SubstituteTypeParams(fd.MapKeyTypeStr, params)
			rv := generator.SubstituteTypeParams(fd.MapValTypeStr, params)
			fd.TestMapKeySample = generator.SampleForConcreteType(rk, "Key")
			fd.TestMapValSample = generator.SampleForConcreteType(rv, "Val")
		}
		// A type-parameter-typed field whose test instantiation
		// resolves to a basic-comparable concrete type IS basic-
		// comparable in the test view — its non-zero sample drives
		// a meaningful `!=` assertion. Without this, generic
		// structs whose only "scalar" fields are type parameters
		// (Pair[A, B any]) lose their Mutate / Clone subtests
		// because FirstComparableField returns nil.
		if !fd.IsBasicComparable && generator.IsBasicComparableTypeName(fd.TestTypeStr) {
			fd.IsBasicComparable = true
		}
	}
	return fd
}

// applyFieldDirective records a `//testkit:default <value>` field
// directive as the field's literal default. Skipped entirely when
// HasDefaults is true — the factory wins.
func applyFieldDirective(pkg *generator.Package, structName string, sd *StructData, fd *FieldData) {
	if sd.HasDefaults {
		return
	}
	if d, ok := generator.FieldDirective(pkg, structName, fd.Name, directive.Default); ok && len(d.Args) > 0 {
		fd.DefaultValue = d.Args[0]
		sd.HasFieldDefaults = true
	}
}

// resolveDefaultsFactory looks for a `<Type>Defaults() <Type>`
// factory function via [generator.LookupCompanionFunc] (source pkg
// first, then sibling output pkg).
//
// When the factory lives in the source package, qualifier prefixes
// the call so the emitted code references it cross-package; output-
// pkg matches leave it bare since the impl lands in that same
// package, and the test-view Transform prepends GenQualifier for the
// external _test file.
func resolveDefaultsFactory(
	pkg *generator.Package, sd *StructData, qualifier string,
	opts generator.Options,
) {
	defaultsFunc := sd.Name + "Defaults"
	check := generator.DefaultsFuncSig(sd.Name)
	found, fromOutput := generator.LookupCompanionFunc(pkg, opts, defaultsFunc, check)
	if !found {
		return
	}
	sd.HasDefaults = true
	if fromOutput {
		sd.DefaultsFunc = defaultsFunc
		sd.DefaultsFromOutput = true
		return
	}
	sd.DefaultsFunc = generator.QualifyType(qualifier, defaultsFunc)
}
