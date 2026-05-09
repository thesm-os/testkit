// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/errors"
)

// Name is the generator's CLI subcommand name. Exported so other
// packages reference suite by typed constant rather than the raw
// "suite" string.
const Name = "suite"

// Data is the top-level template input for one suite generation.
// The renderer emits one file containing the contract drivers
// (`Assert<Iface>Contract` + `Assert<Iface>ContractAcrossImpls`)
// and per-method subtests against any factory-produced impl.
//
// Data wraps [spec.Data] so suite reads the shared analysis output —
// methods, shape classifications, directive payloads — through one
// pointer. Suite-specific naming + per-method projection live on
// the wrapper.
type Data struct {
	// Spec is the shared analysis result.
	Spec *spec.Data

	// PackageName / Imports / GenQualifier / ImplImportPath are the
	// standard output-file fields. The contract driver lives in the
	// external _test package alongside the impl's tests.
	PackageName    string
	Imports        []generator.Import
	ImplImportPath string
	GenQualifier   string

	// IfaceName is the bare interface name ("Store", "Cache") —
	// drives the public driver names below.
	IfaceName string

	// DriverName is the public single-impl driver — `Assert<Iface>Contract`.
	DriverName string

	// AcrossImplsName is the public multi-impl driver —
	// `Assert<Iface>ContractAcrossImpls`. The consumer supplies one
	// factory per implementation; the driver runs the full contract
	// once per impl with `t.Run(name, ...)`.
	AcrossImplsName string

	// FactoryTypeName is the helper alias for the factory function
	// signature — `<Iface>Factory`.
	FactoryTypeName string

	// NamedFactoryTypeName is the helper alias for the multi-impl
	// (name, factory) tuple — `<Iface>NamedFactory`.
	NamedFactoryTypeName string

	// HasErrorMethod is true when at least one non-skipped method
	// returns an error. Drives the driver's `errTest` declaration —
	// suite's per-sentinel and fault-helper-validation subtests
	// reference a shared canonical test error.
	HasErrorMethod bool

	// Methods is one [MethodView] per interface method, populated
	// by the Pipeline's PostEnrich step.
	Methods []MethodView
}

// HasContent reports whether the interface has at least one method
// worth verifying. Reads [Spec] because the Pipeline invokes
// HasContent right after Analyze, before [Project] populates
// [Methods].
func (d *Data) HasContent() bool { return d.Spec != nil && len(d.Spec.Methods) > 0 }

// QualifiedTypeForTest delegates to [spec.Data.QualifiedTypeForTest]
// — the wrapper exists so templates can call .QualifiedTypeForTest
// off the suite view without reaching through .Spec.
func (d *Data) QualifiedTypeForTest() string {
	return d.Spec.QualifiedTypeForTest()
}

// FirstNonSkipMethod returns the first non-skipped method — used by
// driver-level subtests that only need a representative method
// (e.g. "factory yields a non-nil value").
func (d *Data) FirstNonSkipMethod() *MethodView {
	for i := range d.Methods {
		if !d.Methods[i].IsIntegrationOnly() {
			return &d.Methods[i]
		}
	}
	return nil
}

// MethodView wraps a [*spec.Method] with the suite-specific naming
// conventions. Templates call directly into the embedded spec.Method
// for signature-driven helpers (ParamList, ResultDecls, ZeroArgs,
// IsIntegrationOnly, …) and onto MethodView's own methods for
// suite-specific concerns (subtest naming, sample-driven invocation).
type MethodView struct {
	// Method is the underlying analyzed method — embedded for
	// promotion so templates can call its receiver methods directly
	// (e.g. {{.HasContext}}, {{.IsIntegrationOnly}}, {{.ZeroArgs .Tracker}}).
	*spec.Method

	// ifaceName is captured at projection time so per-method
	// renderers stay self-contained when emitting subtests qualified
	// by interface name.
	ifaceName string

	// typeArgs captured at projection time — the test-time concrete
	// instantiation suffix for generic interfaces. Empty for
	// non-generic. Mirrors stub's MethodView.typeArgs pattern.
	typeArgs string

	// substitute rewrites type-parameter names to their concrete
	// test-instantiation forms (e.g. "V" → "string"). Identity for
	// non-generic interfaces; bound by [asTestView] for generics so
	// every Shape* accessor lands the concrete type. Without this,
	// the generated driver would mix abstract V (in shape type
	// args) with concrete `Holder[string]` (in QualifiedTypeForTest)
	// and fail to compile.
	substitute func(string) string
}

func (m MethodView) sub(s string) string {
	if m.substitute == nil {
		return s
	}
	return m.substitute(s)
}

// IfaceName returns the source interface's bare name. Suite emits
// subtest names like "Store/Get/smoke" where the iface name is
// implicit in the driver but useful for shape-specific assertion
// references (e.g. `suite.AssertReturnsForKey[basic.Store, ...]`).
func (m MethodView) IfaceName() string { return m.ifaceName }

// FirstSentinel returns the qualified expression of the first
// sentinel declared via //testkit:errors on this method, or empty
// when the method carries no errors directive. Drives the
// RejectInvalid baseline emission — shapes whose RejectInvalid
// primitive needs a sentinel skip the subtest when no sentinel is
// available (the contract isn't expressible without one).
func (m MethodView) FirstSentinel() string {
	if p, ok := errors.Get(m.Method); ok && len(p.Sentinels) > 0 {
		return p.Sentinels[0].Qualified
	}
	return ""
}

// HasFirstSentinel reports whether [FirstSentinel] returns a non-
// empty value. Template-friendly predicate for gating
// RejectInvalid emission per method.
func (m MethodView) HasFirstSentinel() bool { return m.FirstSentinel() != "" }

// ShapeName returns the detected shape's display name as a string —
// "Reader", "Writer", "StreamReader", etc. Templates dispatch on
// this to pick the per-shape subtest partial.
func (m MethodView) ShapeName() string { return m.Shape.Shape.String() }

// ShapeKeyType returns the rendered type for the shape's primary
// key/input slot. Empty when the shape has no key.
func (m MethodView) ShapeKeyType() string { return m.sub(m.Shape.KeyType) }

// ShapeKeyType2 returns the rendered type for the shape's second
// key (CompositeWriter only).
func (m MethodView) ShapeKeyType2() string { return m.sub(m.Shape.KeyType2) }

// ShapeValType returns the rendered type for the shape's primary
// value/output slot.
func (m MethodView) ShapeValType() string { return m.sub(m.Shape.ValType) }

// ShapeValType2 returns the rendered type for the shape's second
// value (MultiReader, MultiAggregator).
func (m MethodView) ShapeValType2() string { return m.sub(m.Shape.ValType2) }

// ShapeRetType returns the rendered type for the optional non-error
// result of a Writer-with-result.
func (m MethodView) ShapeRetType() string { return m.sub(m.Shape.RetType) }

// ShapeIterValType returns the rendered element type for an
// iter.Seq / iter.Seq2 stream — the V in `iter.Seq[V]` or the
// first arg in `iter.Seq2[V, error]`. Empty for non-stream shapes.
func (m MethodView) ShapeIterValType() string { return m.sub(m.Shape.Iter.ValType) }

// ParamTypeAt returns the rendered type for the i-th non-context
// parameter, with type-parameter substitution applied for the test
// view. Templates call this when the shape didn't capture the type
// in its KeyType/ValType slots (MultiArgWriter's positional params).
func (m MethodView) ParamTypeAt(i int, t *generator.ImportTracker) string {
	return m.sub(m.Method.ParamTypeAt(i, t))
}

// SampleParamAt overrides the embedded [spec.Method.SampleParamAt]
// to apply type-parameter substitution. The spec emits `*new(V)`
// for generic params; the test view needs `*new(string)` so the
// generated assertion compiles against the monomorphized driver.
func (m MethodView) SampleParamAt(i int, t *generator.ImportTracker) string {
	return m.sub(m.Method.SampleParamAt(i, t))
}

// ZeroParamAt mirrors [SampleParamAt] for the zero-literal form.
func (m MethodView) ZeroParamAt(i int, t *generator.ImportTracker) string {
	return m.sub(m.Method.ZeroParamAt(i, t))
}

// SampleResultAt mirrors [SampleParamAt] for results.
func (m MethodView) SampleResultAt(i int, t *generator.ImportTracker) string {
	return m.sub(m.Method.SampleResultAt(i, t))
}

// ZeroResultAt mirrors [SampleParamAt] for the zero-literal form
// of the i-th non-error result.
func (m MethodView) ZeroResultAt(i int, t *generator.ImportTracker) string {
	return m.sub(m.Method.ZeroResultAt(i, t))
}
