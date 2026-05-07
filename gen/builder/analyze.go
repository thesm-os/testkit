// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"go.thesmos.sh/testkit/gen"
)

// Analyze builds a Data model from a loaded package and type args.
func Analyze(
	pkg *gen.Package,
	args []string,
	cfg gen.Config,
	opts gen.Options,
) (*Data, error) {
	outputImportPath, err := gen.OutputImportPath(opts.Output, pkg, opts)
	if err != nil {
		return nil, err
	}

	tracker := gen.NewImportTracker(outputImportPath)

	var structs []StructData

	for _, name := range args {
		s, err := pkg.Struct(name)
		if err != nil {
			return nil, err
		}

		qualifier := tracker.AddPath(pkg.Pkg.Path())
		typeParamDecl := s.TypeParamDecl(tracker)
		typeParamArgs := s.TypeParamArgs()
		isGeneric := typeParamDecl != ""

		qualifiedType := gen.QualifyType(qualifier, name) + typeParamArgs

		// For generics, compute concrete test instantiation types
		// and build a mapping from type param names to concrete types.
		testTypeArgs := ""
		testQualifiedType := qualifiedType
		var typeParamMap map[string]string
		if isGeneric {
			testTypeArgs = defaultTestTypeArgs(s.TypeParams)
			testQualifiedType = gen.QualifyType(qualifier, name) + testTypeArgs
			typeParamMap = buildTypeParamMap(s.TypeParams)
		}

		sd := StructData{
			Name:                name,
			BuilderName:         name + "Builder",
			QualifiedType:       qualifiedType,
			HasUnexportedFields: gen.HasUnexportedFields(s.Type),
			TypeParamDecl:       typeParamDecl,
			TypeParamArgs:       typeParamArgs,
			IsGeneric:           isGeneric,
			TestTypeArgs:        testTypeArgs,
			TestQualifiedType:   testQualifiedType,
		}

		// Check for convention-based defaults: <Type>Defaults() <Type>
		// First check the source package, then try loading the output
		// package (for defaults functions in the test package).
		defaultsFunc := name + "Defaults"
		if hasDefaultsFunc(pkg, defaultsFunc, name) {
			sd.HasDefaults = true
			sd.DefaultsFunc = defaultsFunc
		} else if outputPkg := tryLoadOutputPackage(opts); outputPkg != nil {
			if hasDefaultsFuncInScope(outputPkg.Pkg.Scope(), defaultsFunc, name) {
				sd.HasDefaults = true
				sd.DefaultsFunc = defaultsFunc
			}
		}

		for _, f := range s.Fields {
			if !f.Exported {
				continue
			}
			fd := FieldData{
				Name:        f.Name,
				TypeStr:     types.TypeString(f.Type, tracker.Qualifier()),
				SampleValue: gen.SampleValueOf(f.Type, f.Name, tracker),
			}

			// Detect field type shapes for specialized setters.
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
				fd.MapKeySample = gen.SampleValueOf(ut.Key(), "Key", tracker)
				fd.MapValSample = gen.SampleValueOf(ut.Elem(), "Val", tracker)
			case *types.Struct:
				fd.IsStruct = true
			}

			// Check the original type (not underlying) for pointers.
			if _, ok := f.Type.(*types.Pointer); ok {
				fd.IsPointer = true
			}

			// Check for //testkit:default directive on the field.
			if !sd.HasDefaults {
				dirs := pkg.FieldDirectives(name, f.Name)
				for _, d := range dirs {
					if d.Name == "default" && len(d.Args) > 0 {
						fd.DefaultValue = d.Args[0]
						sd.HasFieldDefaults = true
					}
				}
			}

			// For generic structs, resolve type parameter names to
			// concrete types for test assertions.
			if isGeneric && typeParamMap != nil {
				fd.TestTypeStr = resolveTypeStr(fd.TypeStr, typeParamMap)
				fd.TestSample = sampleForConcreteType(fd.TestTypeStr, fd.Name)
			}

			sd.Fields = append(sd.Fields, fd)
		}

		// Sort fields alphabetically for diff-stable output.
		sort.Slice(sd.Fields, func(i, j int) bool {
			return sd.Fields[i].Name < sd.Fields[j].Name
		})

		structs = append(structs, sd)
	}

	pkgName := gen.DerivePackageName(opts.Output, pkg.Pkg.Name(), cfg, opts)

	return &Data{
		PackageName: pkgName,
		Imports:     tracker.Imports(),
		Structs:     structs,
	}, nil
}

// defaultConcreteTypes are the concrete types used to instantiate
// generic type parameters in generated tests.
var defaultConcreteTypes = []string{"string", "int", "bool", "float64"}

// defaultTestTypeArgs maps type parameters to concrete types for tests.
func defaultTestTypeArgs(params []gen.TypeParamInfo) string {
	names := make([]string, len(params))
	for i := range params {
		names[i] = defaultConcreteTypes[i%len(defaultConcreteTypes)]
	}
	return "[" + strings.Join(names, ", ") + "]"
}

// buildTypeParamMap maps type parameter names to default concrete types.
func buildTypeParamMap(params []gen.TypeParamInfo) map[string]string {
	m := make(map[string]string, len(params))
	for i, p := range params {
		m[p.Name] = defaultConcreteTypes[i%len(defaultConcreteTypes)]
	}
	return m
}

// resolveTypeStr replaces type parameter names with concrete types.
// "T" → "string", "[]T" → "[]string", etc.
func resolveTypeStr(typeStr string, paramMap map[string]string) string {
	result := typeStr
	for param, concrete := range paramMap {
		result = strings.ReplaceAll(result, param, concrete)
	}
	return result
}

// sampleForConcreteType returns a sample value for a resolved concrete type.
func sampleForConcreteType(concreteType, fieldName string) string {
	switch concreteType {
	case "string":
		return `"test-` + strings.ToLower(fieldName) + `"`
	case "int":
		return "42"
	case "bool":
		return "true"
	case "float64":
		return "3.14"
	default:
		if elemType, ok := strings.CutPrefix(concreteType, "[]"); ok {
			inner := sampleForConcreteType(elemType, fieldName)
			return concreteType + "{" + inner + "}"
		}
		return concreteType + "{}"
	}
}

// hasDefaultsFunc checks if a function named funcName exists in the
// package and returns the named type.
func hasDefaultsFunc(pkg *gen.Package, funcName, typeName string) bool {
	return hasDefaultsFuncInScope(pkg.Pkg.Scope(), funcName, typeName)
}

// hasDefaultsFuncInScope checks a specific scope for the defaults function.
func hasDefaultsFuncInScope(scope *types.Scope, funcName, typeName string) bool {
	obj := scope.Lookup(funcName)
	if obj == nil {
		return false
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}
	sig := fn.Type().(*types.Signature)
	if sig.Params().Len() != 0 || sig.Results().Len() != 1 {
		return false
	}
	named, ok := sig.Results().At(0).Type().(*types.Named)
	if !ok {
		return false
	}
	return named.Obj().Name() == typeName
}

// tryLoadOutputPackage attempts to load the output package. Returns nil
// if the package doesn't exist yet (first generation) or can't be loaded.
func tryLoadOutputPackage(opts gen.Options) *gen.Package {
	if opts.Output == "" || opts.WorkDir == "" {
		return nil
	}
	outputDir := filepath.Dir(opts.Output)
	if outputDir == "." {
		return nil // same directory — already checked via source package
	}
	loader := gen.NewLoader()
	pkg, err := loader.Load(".", filepath.Join(opts.WorkDir, outputDir))
	if err != nil {
		return nil // output package doesn't exist yet — that's fine
	}
	return pkg
}
