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

	iface, err := pkg.Interface(typeName)
	if err != nil {
		return nil, err
	}

	qualifiedType := gen.QualifyType(tracker.AddPath(pkg.Pkg.Path()), typeName)

	var methods []*SpecMethodData
	for _, m := range iface.Methods {
		md := &SpecMethodData{
			MethodInfo:    m,
			QualifiedType: qualifiedType,
			tracker:       tracker,
		}
		md.Directives = pkg.EffectiveMethodDirectives(typeName, m.Name)

		// Auto-detect iter.Seq / iter.Seq2 returns.
		for r := range m.Signature.Results().Variables() {
			info := gen.AnalyzeIterReturn(r.Type(), tracker)
			if info.IsSeq || info.IsSeq2 {
				md.Iter = info
				tracker.AddPath("iter")
				break
			}
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
