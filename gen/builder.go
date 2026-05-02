// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"fmt"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
)

const (
	builderGenerator  = "builder"
	builderFileSuffix = "_builder"

	builderImplTmplFile = "builder.go.tmpl"
	builderTestTmplFile = "builder_test.go.tmpl"
)

var emptyPos token.Position

// GenerateBuilder produces a fluent builder with one With* setter per
// exported field for each named struct. Returns a [Result] with the
// implementation file and its companion test file.
func GenerateBuilder(pkg *Package, typeNames []string, cfg Config, opts Options) (*Result, error) {
	errs := ValidateTypes(pkg, typeNames, KindStruct)
	if len(errs) > 0 {
		return nil, errs[0]
	}

	// Determine output paths.
	implPath := opts.Output
	if implPath == "" {
		implPath = defaultBuilderPath(typeNames[0], cfg)
	}
	testPath := testPathFrom(implPath)
	implPkgName := DerivePackageName(implPath, pkg.Pkg.Name(), cfg)

	// Compute the output package's import path so the tracker knows
	// when to suppress qualifiers (same package → no qualifier).
	outputImportPath, err := OutputImportPath(implPath, pkg)
	if err != nil {
		return nil, err
	}
	tracker := NewImportTracker(outputImportPath)
	srcQualifier := tracker.Add(pkg.Pkg)

	var structs []builderData
	for _, name := range typeNames {
		info, lookupErr := pkg.Struct(name)
		if lookupErr != nil {
			return nil, lookupErr
		}
		structs = append(structs, newBuilderData(name, srcQualifier, info, tracker))
	}

	header := Header{
		Subcommand: builderGenerator,
		Args:       builderGenerator + " " + strings.Join(typeNames, " "),
	}

	// Render implementation.
	implBytes, err := Render(templateFile(builderImplTmplFile), builderTemplateData{
		PackageName: implPkgName,
		Imports:     tracker.Imports(),
		Structs:     structs,
	}, header)
	if err != nil {
		return nil, WrapErr(emptyPos, err, "render builder implementation")
	}

	// Build test imports. Tests always use external test package (_test
	// suffix) and import both testkit and the generated package (which
	// re-exports the source types via the builder).
	testPkgName := implPkgName + testPkgSuffix
	testTracker := NewImportTracker("")
	testTracker.AddPath("testing")
	testTracker.AddPath("go.thesmos.sh/testkit")
	// Import the generated package so tests can access builders.
	testTracker.AddPath(outputImportPath)
	// Import source package so tests can reference source types.
	testTracker.Add(pkg.Pkg)

	// For test struct data, we need the qualifier the test file will use
	// for source types and builders.
	testSrcQualifier := testTracker.Add(pkg.Pkg)
	testGenQualifier := testTracker.AddPath(outputImportPath)

	// Build test-specific struct data with correct qualifiers.
	var testStructs []builderData
	for _, name := range typeNames {
		info, lookupErr2 := pkg.Struct(name)
		if lookupErr2 != nil {
			return nil, lookupErr2
		}
		testStructs = append(testStructs, newBuilderData(name, testSrcQualifier, info, testTracker))
	}

	// Render tests.
	testBytes, err := Render(templateFile(builderTestTmplFile), builderTestTemplateData{
		PackageName:  testPkgName,
		Imports:      testTracker.Imports(),
		Structs:      testStructs,
		GenQualifier: testGenQualifier,
	}, header)
	if err != nil {
		return nil, WrapErr(emptyPos, err, "render builder tests")
	}

	return &Result{
		Files: []OutputFile{
			{Path: implPath, Content: implBytes},
			{Path: testPath, Content: testBytes},
		},
	}, nil
}

// --- data types ---

type builderData struct {
	TypeName      string // "User"
	BuilderName   string // "UserBuilder"
	QualifiedType string // "store.User" or "User" if same package
	HasUnexported bool   // true if struct has unexported fields
	Fields        []builderFieldData
}

// fieldTestKind determines how the generated test handles the field.
type fieldTestKind int

const (
	// fieldTestDirect: sample is a value, assert got.Field == sample
	fieldTestDirect fieldTestKind = iota
	// fieldTestPointer: sample is the pointed-to value, pass &sample, assert *got.Field == sample
	fieldTestPointer
	// fieldTestNilOnly: no meaningful sample possible (func, chan, interface), just verify setter compiles
	fieldTestNilOnly
)

type builderFieldData struct {
	Name        string        // "ID"
	TypeStr     string        // "string", "time.Time", etc.
	SampleValue string        // `"test-id"`, `42`, `true`, etc.
	ElemTypeStr string        // for pointers: the element type ("string" for *string)
	TestKind    fieldTestKind // how to test this field
}

type builderTemplateData struct {
	PackageName string
	Imports     []Import
	Structs     []builderData
}

type builderTestTemplateData struct {
	PackageName  string
	Imports      []Import
	Structs      []builderData
	GenQualifier string // package qualifier for generated builders (e.g. "buildertest")
}

// --- helpers ---

func newBuilderData(name, srcQualifier string, info *StructInfo, tracker *ImportTracker) builderData {
	qualified := name
	if srcQualifier != "" {
		qualified = srcQualifier + "." + name
	}

	var fields []builderFieldData
	hasUnexported := false
	for _, f := range info.Fields {
		if !f.Exported {
			hasUnexported = true
			continue
		}
		fields = append(fields, newBuilderFieldData(f, tracker))
	}

	return builderData{
		TypeName:      name,
		BuilderName:   name + "Builder",
		QualifiedType: qualified,
		HasUnexported: hasUnexported,
		Fields:        fields,
	}
}

func newBuilderFieldData(f FieldInfo, tracker *ImportTracker) builderFieldData {
	typeStr := types.TypeString(f.Type, tracker.Qualifier())

	fd := builderFieldData{
		Name:    f.Name,
		TypeStr: typeStr,
	}

	switch u := f.Type.Underlying().(type) {
	case *types.Pointer:
		elem := u.Elem()
		elemStr := types.TypeString(elem, tracker.Qualifier())
		fd.ElemTypeStr = elemStr
		// Pointer to struct with unexported fields → can't deep compare.
		if hasUnexportedFields(elem) {
			fd.TestKind = fieldTestNilOnly
			fd.SampleValue = zeroNil
		} else {
			fd.TestKind = fieldTestPointer
			fd.SampleValue = sampleValueOf(elem, f.Name, tracker)
		}
	case *types.Signature, *types.Chan:
		fd.TestKind = fieldTestNilOnly
		fd.SampleValue = zeroNil
	case *types.Interface:
		fd.TestKind = fieldTestNilOnly
		fd.SampleValue = zeroNil
	default:
		fd.TestKind = fieldTestDirect
		fd.SampleValue = sampleValueOf(f.Type, f.Name, tracker)
	}

	return fd
}

// sampleValueOf returns a non-zero Go literal for a type, suitable for
// use in generated test assertions. The value is deterministic and
// distinct from the zero value so tests can verify setters work.
func sampleValueOf(typ types.Type, fieldName string, tracker *ImportTracker) string {
	// Handle named types — wrap the underlying sample with a type conversion.
	if named, ok := typ.(*types.Named); ok {
		qualifiedName := types.TypeString(typ, tracker.Qualifier())
		switch named.Underlying().(type) {
		case *types.Struct:
			return qualifiedName + zeroSuffix
		case *types.Basic:
			// Named basic type: ID("test-id"), Status(42), etc.
			innerSample := sampleValueOf(named.Underlying(), fieldName, tracker)
			return qualifiedName + "(" + innerSample + ")"
		default:
			return sampleValueOf(named.Underlying(), fieldName, tracker)
		}
	}

	switch u := typ.(type) {
	case *types.Basic:
		return sampleBasicValue(u, fieldName)
	case *types.Slice:
		elemStr := types.TypeString(u.Elem(), tracker.Qualifier())
		sample := sampleValueOf(u.Elem(), fieldName, tracker)
		return fmt.Sprintf("[]%s{%s}", elemStr, sample)
	case *types.Array:
		return types.TypeString(typ, tracker.Qualifier()) + "{1}"
	case *types.Map:
		keyStr := types.TypeString(u.Key(), tracker.Qualifier())
		valStr := types.TypeString(u.Elem(), tracker.Qualifier())
		keySample := sampleValueOf(u.Key(), fieldName, tracker)
		valSample := sampleValueOf(u.Elem(), fieldName, tracker)
		return fmt.Sprintf("map[%s]%s{%s: %s}", keyStr, valStr, keySample, valSample)
	case *types.Struct:
		// Anonymous struct (struct{}).
		return types.TypeString(typ, tracker.Qualifier()) + zeroSuffix
	case *types.Pointer:
		return zeroNil
	case *types.Signature, *types.Chan, *types.Interface:
		return zeroNil
	}
	return zeroNil
}

func sampleBasicValue(b *types.Basic, fieldName string) string {
	switch {
	case b.Info()&types.IsString != 0:
		return fmt.Sprintf(`"test-%s"`, strings.ToLower(fieldName))
	case b.Info()&types.IsBoolean != 0:
		return "true"
	case b.Info()&types.IsInteger != 0:
		return "42"
	case b.Info()&types.IsFloat != 0:
		return "3.14"
	case b.Info()&types.IsUnsigned != 0:
		return "7"
	default:
		return "0"
	}
}

// hasUnexportedFields reports whether the type (or its named underlying
// struct) contains unexported fields, which prevents cmp.Diff comparison.
func hasUnexportedFields(typ types.Type) bool {
	// Unwrap named types.
	if named, ok := typ.(*types.Named); ok {
		typ = named.Underlying()
	}
	strct, ok := typ.(*types.Struct)
	if !ok {
		return false
	}
	for field := range strct.Fields() {
		if !field.Exported() {
			return true
		}
	}
	return false
}

func defaultBuilderPath(typeName string, cfg Config) string {
	return filepath.Join(
		cfg.TestPackageSuffix,
		strings.ToLower(typeName)+builderFileSuffix+cfg.GeneratedSuffix,
	)
}

func testPathFrom(implPath string) string {
	ext := filepath.Ext(implPath)
	return strings.TrimSuffix(implPath, ext) + testFileSuffix
}
