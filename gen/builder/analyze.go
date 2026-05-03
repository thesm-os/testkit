// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go/types"
	"path/filepath"
	"sort"

	"go.thesmos.sh/testkit/gen"
)

// Analyze builds a Data model from a loaded package and type args.
func Analyze(
	pkg *gen.Package,
	args []string,
	cfg gen.Config,
	opts gen.Options,
) (*Data, error) {
	outputImportPath, err := gen.OutputImportPath(opts.Output, pkg)
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
		sd := StructData{
			Name:                name,
			BuilderName:         name + "Builder",
			QualifiedType:       gen.QualifyType(qualifier, name),
			HasUnexportedFields: gen.HasUnexportedFields(s.Type),
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

			// Detect slice types for variadic setters.
			if sl, ok := f.Type.Underlying().(*types.Slice); ok {
				fd.IsSlice = true
				fd.ElemTypeStr = types.TypeString(sl.Elem(), tracker.Qualifier())
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

			sd.Fields = append(sd.Fields, fd)
		}

		// Sort fields alphabetically for diff-stable output.
		sort.Slice(sd.Fields, func(i, j int) bool {
			return sd.Fields[i].Name < sd.Fields[j].Name
		})

		structs = append(structs, sd)
	}

	pkgName := gen.DerivePackageName(opts.Output, pkg.Pkg.Name(), cfg)

	return &Data{
		PackageName: pkgName,
		Imports:     tracker.Imports(),
		Structs:     structs,
	}, nil
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
