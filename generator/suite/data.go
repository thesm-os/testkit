// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"strings"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/atomic"
	"go.thesmos.sh/testkit/generator/spec/bounded"
	"go.thesmos.sh/testkit/generator/spec/cacheable"
	"go.thesmos.sh/testkit/generator/spec/concurrent"
	"go.thesmos.sh/testkit/generator/spec/concurrentreaders"
	"go.thesmos.sh/testkit/generator/spec/crdtmerge"
	"go.thesmos.sh/testkit/generator/spec/deleteremoves"
	"go.thesmos.sh/testkit/generator/spec/deprecated"
	"go.thesmos.sh/testkit/generator/spec/errors"
	"go.thesmos.sh/testkit/generator/spec/eventually"
	"go.thesmos.sh/testkit/generator/spec/hooks"
	"go.thesmos.sh/testkit/generator/spec/idempotent"
	"go.thesmos.sh/testkit/generator/spec/lease"
	"go.thesmos.sh/testkit/generator/spec/lifecycleafterclose"
	"go.thesmos.sh/testkit/generator/spec/monotonic"
	"go.thesmos.sh/testkit/generator/spec/nilsafe"
	"go.thesmos.sh/testkit/generator/spec/orderafter"
	"go.thesmos.sh/testkit/generator/spec/pagination"
	"go.thesmos.sh/testkit/generator/spec/partition"
	"go.thesmos.sh/testkit/generator/spec/pure"
	"go.thesmos.sh/testkit/generator/spec/readafterwrite"
	"go.thesmos.sh/testkit/generator/spec/retrysucceeds"
	"go.thesmos.sh/testkit/generator/spec/sample"
	"go.thesmos.sh/testkit/generator/spec/scope"
	"go.thesmos.sh/testkit/generator/spec/sideeffect"
	"go.thesmos.sh/testkit/generator/spec/streamreflectsmutations"
	"go.thesmos.sh/testkit/generator/spec/timeout"
	"go.thesmos.sh/testkit/generator/spec/validates"
	"go.thesmos.sh/testkit/generator/spec/wrappedvia"
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

// ResultTypeAt returns the rendered type for the i-th result, with
// type-parameter substitution applied. Used by pagination and
// other directive subtests that need the impl's return type at
// emit time.
func (m MethodView) ResultTypeAt(i int, t *generator.ImportTracker) string {
	return m.sub(m.Method.ResultTypeAt(i, t))
}

// SampleParamAt overrides the embedded [spec.Method.SampleParamAt]
// to consult `//testkit:sample` first, then apply type-parameter
// substitution. When the method carries `//testkit:sample <Func>...`,
// the i-th param resolves to `<Func>()` (the consumer's constructor
// invocation) instead of the default literal — letting consumers
// drive contract assertions with values their impl actually returns.
// Falls back to the default (with substitute applied) when no sample
// directive is present.
func (m MethodView) SampleParamAt(i int, t *generator.ImportTracker) string {
	if p, ok := sample.Get(m.Method); ok && i >= 0 && i < len(p.Calls) {
		return p.Calls[i] + "()"
	}
	return m.sub(m.Method.SampleParamAt(i, t))
}

// SampleArgs overrides the embedded [spec.Method.SampleArgs] to
// consult `//testkit:sample` first. Returns the comma-joined call
// list with a leading `t.Context()` when the method takes a context
// — matching spec's default rendering — or falls back to the
// default when no sample directive is present.
func (m MethodView) SampleArgs(t *generator.ImportTracker) string {
	if p, ok := sample.Get(m.Method); ok && len(p.Calls) == m.NonCtxParamCount() {
		args := make([]string, 0, len(p.Calls)+1)
		if m.HasContext() {
			args = append(args, "t.Context()")
		}
		for _, call := range p.Calls {
			args = append(args, call+"()")
		}
		return strings.Join(args, ", ")
	}
	return m.sub(m.Method.SampleArgs(t))
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

// SampleEqualsZero reports whether the rendered sample literal for
// the i-th non-ctx param equals the zero literal. True for generic
// Reader where both render as `*new(K)`. The Reader contract uses
// it to skip the [AssertReturnsSentinel] emission when it would
// conflict with [AssertReturnsForKey] on the same key.
func (m MethodView) SampleEqualsZero(i int, t *generator.ImportTracker) bool {
	return m.SampleParamAt(i, t) == m.ZeroParamAt(i, t)
}

// --- Per-directive typed accessors ---
//
// Each pair (`HasX` predicate + `XPayload` getter) projects one
// registered consumer's payload through suite's preferred names.
// Templates dispatch on the predicate; partials read the payload
// fields when emitting their assertion. Mirror of stub's pattern.

// IsAtomic reports whether the method carries //testkit:atomic.
func (m MethodView) IsAtomic() bool { return atomic.Has(m.Method) }

// IsIdempotent reports whether the method carries //testkit:idempotent.
func (m MethodView) IsIdempotent() bool { return idempotent.Has(m.Method) }

// IsPure reports whether the method carries //testkit:pure.
func (m MethodView) IsPure() bool { return pure.Has(m.Method) }

// IsCacheable reports whether the method carries //testkit:cacheable.
func (m MethodView) IsCacheable() bool { return cacheable.Has(m.Method) }

// IsMonotonic reports whether the method carries //testkit:monotonic.
func (m MethodView) IsMonotonic() bool { return monotonic.Has(m.Method) }

// IsConcurrent reports whether the method carries //testkit:concurrent.
func (m MethodView) IsConcurrent() bool { return concurrent.Has(m.Method) }

// IsConcurrentReaders reports whether the method carries
// //testkit:concurrent-readers.
func (m MethodView) IsConcurrentReaders() bool { return concurrentreaders.Has(m.Method) }

// IsNilSafe reports whether the method carries //testkit:nilsafe.
func (m MethodView) IsNilSafe() bool { return nilsafe.Has(m.Method) }

// IsDeprecated reports whether the method carries //testkit:deprecated.
func (m MethodView) IsDeprecated() bool { return deprecated.Has(m.Method) }

// DeprecatedReplacement returns the replacement-method name from
// //testkit:deprecated, or "" when absent.
func (m MethodView) DeprecatedReplacement() string {
	if p, ok := deprecated.Get(m.Method); ok {
		return p.Replacement
	}
	return ""
}

// HasRetrySucceeds reports whether the method carries
// //testkit:retry-succeeds-on-attempt.
func (m MethodView) HasRetrySucceeds() bool { return retrysucceeds.Has(m.Method) }

// RetryN returns the retry-succeeds N from
// //testkit:retry-succeeds-on-attempt, or 0 when absent.
func (m MethodView) RetryN() int {
	if p, ok := retrysucceeds.Get(m.Method); ok {
		return p.N
	}
	return 0
}

// HasOrderAfter reports whether the method carries //testkit:order-after.
func (m MethodView) HasOrderAfter() bool { return orderafter.Has(m.Method) }

// OrderAfterTarget returns the prerequisite-method name from
// //testkit:order-after, or "" when absent.
func (m MethodView) OrderAfterTarget() string {
	if p, ok := orderafter.Get(m.Method); ok {
		return p.Method
	}
	return ""
}

// HasPartition reports whether the method carries //testkit:partition.
func (m MethodView) HasPartition() bool { return partition.Has(m.Method) }

// PartitionField returns the field name from //testkit:partition,
// or "" when absent.
func (m MethodView) PartitionField() string {
	if p, ok := partition.Get(m.Method); ok {
		return p.FieldName
	}
	return ""
}

// HasWrappedVia reports whether the method carries //testkit:wrapped-via.
func (m MethodView) HasWrappedVia() bool { return wrappedvia.Has(m.Method) }

// WrappedViaTarget returns the qualified wrap-target sentinel from
// //testkit:wrapped-via, or "" when absent.
func (m MethodView) WrappedViaTarget() string {
	if p, ok := wrappedvia.Get(m.Method); ok {
		return p.Qualified
	}
	return ""
}

// HasBounded reports whether the method carries //testkit:bounded.
func (m MethodView) HasBounded() bool { return bounded.Has(m.Method) }

// BoundedMin returns the rendered lower bound from //testkit:bounded,
// or "" when absent.
func (m MethodView) BoundedMin() string {
	if p, ok := bounded.Get(m.Method); ok {
		return p.Min
	}
	return ""
}

// BoundedMax returns the rendered upper bound from //testkit:bounded,
// or "" when absent.
func (m MethodView) BoundedMax() string {
	if p, ok := bounded.Get(m.Method); ok {
		return p.Max
	}
	return ""
}

// HasTimeout reports whether the method carries //testkit:timeout.
func (m MethodView) HasTimeout() bool { return timeout.Has(m.Method) }

// TimeoutDuration returns the verbatim duration string from
// //testkit:timeout, or "" when absent.
func (m MethodView) TimeoutDuration() string {
	if p, ok := timeout.Get(m.Method); ok {
		return p.Duration
	}
	return ""
}

// HasSideEffect reports whether the method carries //testkit:sideeffect.
func (m MethodView) HasSideEffect() bool { return sideeffect.Has(m.Method) }

// SideEffectMethod returns the paired observation-method name from
// //testkit:sideeffect, or "" when absent.
func (m MethodView) SideEffectMethod() string {
	if p, ok := sideeffect.Get(m.Method); ok {
		return p.Method
	}
	return ""
}

// HasValidates reports whether the method carries //testkit:validates.
func (m MethodView) HasValidates() bool { return validates.Has(m.Method) }

// ValidatesField returns the field name from //testkit:validates,
// or "" when absent.
func (m MethodView) ValidatesField() string {
	if p, ok := validates.Get(m.Method); ok {
		return p.Field
	}
	return ""
}

// HasHooks reports whether the method carries //testkit:hooks.
func (m MethodView) HasHooks() bool { return hooks.Has(m.Method) }

// HookNames returns the declared hook names from //testkit:hooks,
// or nil when absent.
func (m MethodView) HookNames() []string {
	if p, ok := hooks.Get(m.Method); ok {
		return p.Names
	}
	return nil
}

// HasEventually reports whether the method carries //testkit:eventually.
func (m MethodView) HasEventually() bool { return eventually.Has(m.Method) }

// EventuallyTimeout returns the verbatim convergence duration from
// //testkit:eventually, or "" when absent.
func (m MethodView) EventuallyTimeout() string {
	if p, ok := eventually.Get(m.Method); ok {
		return p.Duration
	}
	return ""
}

// HasScope reports whether the method carries //testkit:scope.
func (m MethodView) HasScope() bool { return scope.Has(m.Method) }

// ScopeName returns the required scope name from //testkit:scope,
// or "" when absent.
func (m MethodView) ScopeName() string {
	if p, ok := scope.Get(m.Method); ok {
		return p.Name
	}
	return ""
}

// HasPagination reports whether the method carries //testkit:pagination.
func (m MethodView) HasPagination() bool { return pagination.Has(m.Method) }

// PaginationCursor returns the cursor field name from
// //testkit:pagination, or "" when absent.
func (m MethodView) PaginationCursor() string {
	if p, ok := pagination.Get(m.Method); ok {
		return p.CursorField
	}
	return ""
}

// HasLease reports whether the method carries //testkit:lease.
func (m MethodView) HasLease() bool { return lease.Has(m.Method) }

// LeaseRelease returns the paired release-method name from
// //testkit:lease, or "" when absent.
func (m MethodView) LeaseRelease() string {
	if p, ok := lease.Get(m.Method); ok {
		return p.Release
	}
	return ""
}

// HasReadAfterWrite reports whether the method carries
// //testkit:read-after-write.
func (m MethodView) HasReadAfterWrite() bool { return readafterwrite.Has(m.Method) }

// ReadAfterWriteReader returns the paired reader-method name from
// //testkit:read-after-write, or "" when absent.
func (m MethodView) ReadAfterWriteReader() string {
	if p, ok := readafterwrite.Get(m.Method); ok {
		return p.Reader
	}
	return ""
}

// HasDeleteRemoves reports whether the method carries
// //testkit:delete-removes.
func (m MethodView) HasDeleteRemoves() bool { return deleteremoves.Has(m.Method) }

// DeleteRemovesReader returns the paired reader-method name from
// //testkit:delete-removes, or "" when absent.
func (m MethodView) DeleteRemovesReader() string {
	if p, ok := deleteremoves.Get(m.Method); ok {
		return p.Reader
	}
	return ""
}

// HasStreamReflectsMutations reports whether the method carries
// //testkit:stream-reflects-mutations.
func (m MethodView) HasStreamReflectsMutations() bool {
	return streamreflectsmutations.Has(m.Method)
}

// StreamReflectsMutationsStream returns the paired stream-method name
// from //testkit:stream-reflects-mutations, or "" when absent.
func (m MethodView) StreamReflectsMutationsStream() string {
	if p, ok := streamreflectsmutations.Get(m.Method); ok {
		return p.Stream
	}
	return ""
}

// HasLifecycleAfterClose reports whether the method carries
// //testkit:lifecycle-after-close.
func (m MethodView) HasLifecycleAfterClose() bool { return lifecycleafterclose.Has(m.Method) }

// LifecycleAfterCloseReader returns the paired reader-method name from
// //testkit:lifecycle-after-close, or "" when absent.
func (m MethodView) LifecycleAfterCloseReader() string {
	if p, ok := lifecycleafterclose.Get(m.Method); ok {
		return p.Reader
	}
	return ""
}

// HasCRDTMerge reports whether the method carries //testkit:crdt-merge.
func (m MethodView) HasCRDTMerge() bool { return crdtmerge.Has(m.Method) }

// CRDTMergeOther returns the paired counterpart-method name from
// //testkit:crdt-merge, or "" when absent.
func (m MethodView) CRDTMergeOther() string {
	if p, ok := crdtmerge.Get(m.Method); ok {
		return p.Other
	}
	return ""
}
