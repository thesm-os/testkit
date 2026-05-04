// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"errors"

	"go.thesmos.sh/testkit/gen"
)

// Analyze builds a SpecData model from a loaded package and interface name.
// The returned data has base fields populated but no directive enrichment —
// that happens in a separate Enrich step.
func Analyze(pkg *gen.Package, args []string, cfg gen.Config, opts gen.Options) (*SpecData, error) {
	if len(args) != 1 {
		return nil, errors.New("suite generator requires exactly one interface argument")
	}
	typeName := args[0]

	outputImportPath, err := gen.OutputImportPath(opts.Output, pkg)
	if err != nil {
		return nil, err
	}

	tracker := gen.NewImportTracker(outputImportPath)
	tracker.AddPath("testing")
	tracker.AddPath("go.thesmos.sh/testkit")
	tracker.AddPath("go.thesmos.sh/testkit/suite")
	tracker.AddPath("go.thesmos.sh/testkit/bindings")

	iface, err := pkg.Interface(typeName)
	if err != nil {
		return nil, err
	}

	qualifiedType := gen.QualifyType(tracker.AddPath(pkg.Pkg.Path()), typeName)

	var methods []*SpecMethodData
	for _, m := range iface.Methods {
		md := &SpecMethodData{
			MethodInfo:    m,
			InterfaceName: typeName,
			QualifiedType: qualifiedType,
			tracker:       tracker,
		}
		md.Directives = pkg.EffectiveMethodDirectives(iface.OriginName, m.Name)

		// Detect context, params, and method shape.
		md.HasCtx = gen.HasContextParam(m.Signature)
		md.ParamOnly = gen.NonCtxParamCount(m.Signature) > 0
		md.Shape = gen.DetectShape(m, tracker, md.Directives)
		if md.Shape.Shape == gen.ShapeStreamReader {
			md.Iter = md.Shape.IterInfo
			tracker.AddPath("iter")
		}

		methods = append(methods, md)
	}

	pkgName := gen.DerivePackageName(opts.Output, pkg.Pkg.Name(), cfg)

	return &SpecData{
		PackageName:   pkgName,
		Imports:       tracker.Imports(),
		InterfaceName: typeName,
		QualifiedType: qualifiedType,
		Methods:       methods,
	}, nil
}
