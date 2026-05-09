// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/spec"
)

// Analyze produces a [*Data] for one interface. Calls [spec.Analyze]
// for the shared analysis (signature + shape + type-param handling),
// then attaches the bench-specific top-level fields. Per-method
// projection runs in [Project] after directive consumers populate
// Attachments.
func Analyze(pkg *generator.Package, args []string, cfg generator.Config, opts generator.Options) (*Data, error) {
	base, err := spec.Analyze(pkg, args, cfg, opts)
	if err != nil {
		return nil, err
	}
	// Register the runtime imports the bench driver always emits.
	// Adding them up front prevents goimports from resolving to the
	// wrong "testing"/"context" package alias.
	base.Tracker.AddPath("testing")
	base.Tracker.AddPath("context")
	base.Tracker.AddPath("go.thesmos.sh/testkit/bench")
	base.Tracker.AddPath("go.thesmos.sh/testkit/bindings")

	ifaceName := base.Interface.Name
	lower := generator.LowerCamelCase(ifaceName)
	return &Data{
		Spec:              base,
		PackageName:       base.PackageName,
		ImplImportPath:    base.ImplImportPath,
		IfaceName:         ifaceName,
		LowerIfaceName:    lower,
		DriverName:        "Benchmark" + ifaceName + "Contract",
		OptionTypeName:    ifaceName + "BenchOption",
		ConfigTypeName:    lower + "BenchConfig",
		CustomSubTypeName: lower + "BenchCustomSub",
		PrePopulateName:   ifaceName + "BenchPrePopulate",
		CustomName:        ifaceName + "BenchCustom",
		NewConfigFunc:     "new" + ifaceName + "BenchConfig",
	}, nil
}

// Enrich runs the shared directive-consumer pass via [spec.Enrich].
// Bench adds no consumers of its own — every directive it cares
// about (sample, allocs, latency, integration-only) lives under
// generator/spec/<directive>.
func Enrich(d *Data, pkg *generator.Package) error {
	return spec.Enrich(d.Spec, pkg)
}

// Project populates [Data.Methods] using the embedded spec.Method +
// bench-specific naming. Finalizes the import set last — directive
// consumers register cross-package imports during Enrich, so the
// import list captures here.
//
// Conditional imports (added to the tracker only when at least one
// method actually uses them, to avoid unused-import errors):
//
//   - `time`         — when any method has //testkit:latency or
//     //testkit:percentiles (both render time.Duration literals)
//   - `iter`         — when any method is a StreamReader (Call
//     closure returns iter.Seq2[V, error])
//   - `bytes` + `io` — when any StreamConsumer takes io.Reader
//     (default streamFactory uses bytes.NewReader)
func Project(d *Data, _ *generator.Package) error {
	d.Methods = make([]MethodView, 0, len(d.Spec.Methods))
	var needsTime, needsIter, needsIOReader bool
	for i := range d.Spec.Methods {
		mv := MethodView{
			Method:         &d.Spec.Methods[i],
			ifaceName:      d.IfaceName,
			lowerIfaceName: d.LowerIfaceName,
		}
		d.Methods = append(d.Methods, mv)
		if mv.IsIntegrationOnly() {
			continue
		}
		if mv.HasLatencyBudget() || mv.HasPercentiles() {
			needsTime = true
		}
		switch mv.ShapeName() {
		case "StreamReader":
			needsIter = true
		case "StreamConsumer":
			if mv.ShapeKeyType() == "io.Reader" {
				needsIOReader = true
			}
		}
	}
	if needsTime {
		d.Spec.Tracker.AddPath("time")
	}
	if needsIter {
		d.Spec.Tracker.AddPath("iter")
	}
	if needsIOReader {
		d.Spec.Tracker.AddPath("bytes")
		d.Spec.Tracker.AddPath("io")
	}
	d.Spec.FinalizeImports()
	d.Imports = d.Spec.Imports
	return nil
}
