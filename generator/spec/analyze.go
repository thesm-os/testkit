// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package spec

import (
	"go/token"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/shape"
)

// Analyze produces a [*Data] for one interface. Every conformance
// generator (stub, suite, bench, model) calls this from its own
// Analyze function, then wraps the result with generator-specific
// fields:
//
//	func Analyze(pkg *generator.Package, args []string, cfg generator.Config, opts generator.Options) (*Data, error) {
//	    base, err := spec.Analyze(pkg, args, cfg, opts)
//	    if err != nil {
//	        return nil, err
//	    }
//	    return &Data{Data: base, /* generator-specific */}, nil
//	}
//
// What Analyze does NOT do:
//
//   - It does not run directive consumers — those land via [Enrich]
//     (called by the consuming generator's own Enrich after
//     spec.Analyze returns).
//   - It does not validate composition — the Pipeline's
//     CompositionValidator slot owns that.
//   - It does not run mixin emitters — those run via [Enrich] in the
//     same pass as consumers (mixins and enrichers share the
//     Attachments namespace).
//
// Errors are returned with the canonical [generator.Errorf] /
// [generator.Errorf]-shaped wrapping so callers can preserve source
// positions through the pipeline.
func Analyze(pkg *generator.Package, args []string, cfg generator.Config, opts generator.Options) (*Data, error) {
	if len(args) == 0 {
		return nil, generator.Errorf(token.Position{}, "spec: no interface specified")
	}
	if len(args) > 1 {
		return nil, generator.Errorf(token.Position{},
			"spec: expected exactly one interface, got %d (%v)", len(args), args)
	}

	ctx, err := generator.BuildOutputCtx(pkg, cfg, opts)
	if err != nil {
		return nil, err
	}

	name := args[0]
	iface, err := pkg.Interface(name)
	if err != nil {
		return nil, err
	}

	tracker := ctx.Tracker
	qualifier := tracker.AddPath(pkg.Path())

	typeParamDecl := iface.TypeParamDecl(tracker)
	typeParamArgs := iface.TypeParamArgs()
	qualifiedType := generator.QualifyType(qualifier, name) + typeParamArgs

	methods := make([]Method, 0, len(iface.Methods))
	for _, m := range iface.Methods {
		info := shape.Classify(m, tracker, m.Directives)
		methods = append(methods, Method{
			MethodInfo: m,
			Shape:      info,
		})
	}

	return &Data{
		PackageName:    ctx.PackageName,
		ImplImportPath: ctx.OutputImportPath,
		Package:        pkg,
		Interface:      *iface,
		QualifiedType:  qualifiedType,
		TypeParamDecl:  typeParamDecl,
		TypeParamArgs:  typeParamArgs,
		IsGeneric:      typeParamDecl != "",
		Methods:        methods,
		Tracker:        tracker,
		Loader:         generator.NewLoader(),
		Args:           args,
	}, nil
}

// FinalizeImports captures the resolved import list from the tracker
// into [Data.Imports]. Call this AFTER all enrichment / template-data
// derivation finishes — any code path that adds imports through the
// tracker must run before this. The Pipeline's PostEnrich slot or a
// renderer's Transform hook is the natural call site.
//
// Separate from [Analyze] because some consumers add imports during
// enrichment (e.g. an enricher that rewrites a method's sample
// expression to use a helper from a different package). Calling this
// at Analyze time would freeze the import set before those imports
// land.
func (d *Data) FinalizeImports() {
	d.Imports = d.Tracker.Imports()
}
