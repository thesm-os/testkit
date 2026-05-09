// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/orderafter"
)

// Analyze produces a [*Data] for one interface. Calls [spec.Analyze]
// for the shared analysis (signature + shape + type-param handling),
// then attaches the stub-specific top-level fields. Per-method
// projection runs in [Project] (the Pipeline's PostEnrich slot)
// after directive consumers populate Attachments.
func Analyze(pkg *generator.Package, args []string, cfg generator.Config, opts generator.Options) (*Data, error) {
	base, err := spec.Analyze(pkg, args, cfg, opts)
	if err != nil {
		return nil, err
	}
	// Register the runtime imports the stub-impl emits unconditionally
	// — adding them up front prevents goimports from picking
	// "math/rand/v2" when our templates reference rand.Source.
	base.Tracker.AddPath("testing")
	base.Tracker.AddPath("go.thesmos.sh/testkit")
	base.Tracker.AddPath("go.thesmos.sh/testkit/stub")
	base.Tracker.AddPath("go.thesmos.sh/testkit/clock")
	base.Tracker.AddPath("go.thesmos.sh/testkit/rand")

	return &Data{
		Spec:           base,
		PackageName:    base.PackageName,
		ImplImportPath: base.ImplImportPath,
		StubName:       base.Interface.Name + "Stub",
		TestTypeArgs:   generator.TestTypeArgs(base.Interface.TypeParams),
	}, nil
}

// Enrich runs the shared directive-consumer pass via [spec.Enrich].
// Stub adds no consumers of its own — every directive it cares
// about lives under spec/<directive>.
func Enrich(d *Data, pkg *generator.Package) error {
	return spec.Enrich(d.Spec, pkg)
}

// Project populates [Data.Methods] using the embedded spec.Method
// + stub-specific naming. Computes the interface-level flags
// (HasOrderConstraint, HasErrorMethod) by walking enriched
// payloads. Finalizes the import set last — directive consumers
// register cross-package imports during Enrich, so the import list
// captures here.
func Project(d *Data, _ *generator.Package) error {
	d.Methods = make([]MethodView, len(d.Spec.Methods))
	for i := range d.Spec.Methods {
		d.Methods[i] = MethodView{
			Method:   &d.Spec.Methods[i],
			stubName: d.StubName,
			typeArgs: d.Spec.TypeParamArgs,
		}
		if d.Methods[i].ReturnsError() {
			d.HasErrorMethod = true
		}
		if orderafter.Has(d.Methods[i].Method) {
			d.HasOrderConstraint = true
		}
	}
	d.Spec.FinalizeImports()
	d.Imports = d.Spec.Imports
	return nil
}
