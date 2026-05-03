// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder

import (
	"go/types"

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

		for _, f := range s.Fields {
			if !f.Exported {
				continue
			}
			sd.Fields = append(sd.Fields, FieldData{
				Name:        f.Name,
				TypeStr:     types.TypeString(f.Type, tracker.Qualifier()),
				SampleValue: gen.SampleValueOf(f.Type, f.Name, tracker),
			})
		}

		structs = append(structs, sd)
	}

	pkgName := gen.DerivePackageName(opts.Output, pkg.Pkg.Name(), cfg)

	return &Data{
		PackageName: pkgName,
		Imports:     tracker.Imports(),
		Structs:     structs,
	}, nil
}
