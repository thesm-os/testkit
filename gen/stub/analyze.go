// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"strings"

	"go.thesmos.sh/testkit/gen"
)

// Analyze builds a Data model from a loaded package and type args.
// The returned data has all base fields populated but no directive
// enrichment — that happens in a separate step.
func Analyze(pkg *gen.Package, args []string, cfg gen.Config, opts gen.Options) (*Data, error) {
	outputImportPath, err := gen.OutputImportPath(opts.Output, pkg)
	if err != nil {
		return nil, err
	}

	tracker := gen.NewImportTracker(outputImportPath)

	// Always needed imports.
	tracker.AddPath("testing")
	tracker.AddPath("go.thesmos.sh/testkit")

	var interfaces []InterfaceData

	for _, name := range args {
		iface, err := pkg.Interface(name)
		if err != nil {
			return nil, err
		}

		ifaceData := InterfaceData{
			Name:          name,
			StubName:      name + cfg.Stub.TypeSuffix,
			TypeName:      name,
			QualifiedType: gen.QualifyType(tracker.AddPath(pkg.Pkg.Path()), name),
			sourcePkgPath: pkg.Pkg.Path(),
		}

		methods := make([]*MethodData, 0, len(iface.Methods))
		for _, m := range iface.Methods {
			lowerIface := strings.ToLower(name[:1]) + name[1:]
			md := &MethodData{
				MethodInfo: m,
				CallType:   name + m.Name + "Call",
				StubType:   name + m.Name + cfg.Stub.TypeSuffix,
				ReturnType: lowerIface + m.Name + "Return",
				Params:     gen.BuildParamFields(m.Signature.Params(), tracker),
				Results:    gen.BuildResultFields(m.Signature.Results(), tracker),
				tracker:    tracker,
				iface:      &ifaceData,
			}
			// Attach directives from source AST.
			md.Directives = pkg.EffectiveMethodDirectives(name, m.Name)
			methods = append(methods, md)
		}
		ifaceData.Methods = methods
		interfaces = append(interfaces, ifaceData)
	}

	pkgName := gen.DerivePackageName(opts.Output, pkg.Pkg.Name(), cfg)

	return &Data{
		PackageName: pkgName,
		Imports:     tracker.Imports(),
		Interfaces:  interfaces,
	}, nil
}
