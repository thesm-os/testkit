// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"fmt"
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

// ContractDoc is a one-line description of a single conformance
// subtest emitted by the suite generator. The Name mirrors the t.Run
// name (or its template, when the runtime parameterizes it) and the
// Description explains what behavior the subtest verifies. Used to
// populate the per-method docblock so an engineer reading the
// generated file sees the full contract surface inline rather than
// having to grep the runtime helpers in suite/<shape>.go.
type ContractDoc struct {
	Name        string
	Description string
}

// Subtest names common across many shapes. Kept as constants so the
// docblock stays in lockstep with the runtime t.Run names; renaming a
// universal primitive flips both at once.
const (
	subtestSmoke         = "smoke"
	subtestConcurrent    = "concurrent safe"
	subtestRespectsCtx   = "respects context"
	subtestRespectsCtxSt = "respects context (structural)"
	subtestConsistent    = "consistent reads"
	subtestReturnsForKey = "returns for key"
)

// Description fragments shared across shapes for the same primitive
// behavior. Same rationale as the subtest-name constants above.
const (
	descSmokeBare    = "fail-fast bare invocation"
	descConcurrent10 = "4 workers × 10 iterations under -race"
	descCtxCanceled  = "ctx.Done() surfaces context.Canceled"
	descConsistent3  = "three sequential calls yield equal results"
	descRejectInvOpt = "rejects invalid (when InvalidFactory option supplied)"
)

// Shape-name constants for the switch statements below. Keeping them
// here lets static analysis cross-check the dispatcher's coverage of
// the suite's known shapes.
const (
	shapePure            = "Pure"
	shapePredicate       = "Predicate"
	shapeAggregator      = "Aggregator"
	shapeMultiAggregator = "MultiAggregator"
	shapeReader          = "Reader"
	shapeReaderNoError   = "ReaderNoError"
	shapeReaderWithBool  = "ReaderWithBool"
	shapePointerReader   = "PointerReader"
	shapeLookup          = "Lookup"
	shapeMultiReader     = "MultiReader"
	shapeBatchReader     = "BatchReader"
	shapeWriter          = "Writer"
	shapeCompositeWriter = "CompositeWriter"
	shapeMultiArgWriter  = "MultiArgWriter"
	shapeDeleter         = "Deleter"
	shapeMutator         = "Mutator"
	shapeLifecycle       = "Lifecycle"
	shapeVoidLifecycle   = "VoidLifecycle"
	shapePoisonAccessor  = "PoisonAccessor"
	shapeStreamReader    = "StreamReader"
	shapeStreamConsumer  = "StreamConsumer"
)

// ShapeDescription returns a one-paragraph description of the
// method's detected shape — its signature category, what guarantees
// the shape claims, and the kind of behavior the runtime baseline
// verifies. Grounding for an engineer who hasn't memorized the shape
// vocabulary; ignored when ShapeName is unrecognized.
func (m MethodView) ShapeDescription() string {
	switch m.ShapeName() {
	case shapePure:
		return "func() R — pure function. No context, no error. Deterministic across calls; no observable side effects."
	case shapePredicate:
		return "func() bool — predicate. No inputs; returns true/false. Race-free under concurrent access."
	case shapeAggregator:
		return "func(ctx) (R, error) — scalar aggregator. Reads internal state through a context-aware ctor; returns one value plus an error. Consistent across calls when state hasn't changed."
	case shapeMultiAggregator:
		return "func(ctx) (V1, V2, error) — multi-value aggregator. Same shape as Aggregator but returns two values."
	case shapeReader:
		return "func(ctx, K) (V, error) — keyed reader. Maps a key to a value or surfaces a sentinel error for unknown keys. Reads must be consistent (same input → same output) and free of observable side effects."
	case shapeReaderNoError:
		return "func(ctx, K) V — keyed reader without error. Returns the zero V for unknown keys; no sentinel discipline."
	case shapeReaderWithBool:
		return "func(ctx, K) (V, bool) — keyed reader with presence bool. ok=false replaces error/sentinel for missing keys."
	case shapePointerReader:
		return "func(ctx, K) *V — keyed reader returning a pointer. nil signals missing; non-nil is the value."
	case shapeLookup:
		return "func(ctx, K) (V, R, bool) — keyed lookup with a secondary result and presence bool."
	case shapeMultiReader:
		return "func(ctx, K) (V1, V2, error) — keyed reader returning two values."
	case shapeBatchReader:
		return "func(ctx, ...K) ([]V, error) — batch reader. Variadic key input, slice value output, parallel arity."
	case shapeWriter:
		return "func(ctx, V) error — value writer. Idempotent on repeated identical writes; honors ctx cancellation."
	case shapeCompositeWriter:
		return "func(ctx, K1, V) error — composite-keyed writer. Same idempotency contract as Writer with a paired key+value input."
	case shapeMultiArgWriter:
		return "func(ctx, ...) error — multi-arg writer. Arity-agnostic via variadic dispatch; runtime restores typed args at the boundary."
	case shapeDeleter:
		return "func(ctx, K) error — keyed deleter. Idempotent on repeated deletes of the same key."
	case shapeMutator:
		return "func(ctx, V) — void mutator. Mutates internal state on each call; no return."
	case shapeLifecycle:
		return "func(ctx) error — lifecycle method (typically Open/Close). Idempotent across repeated invocations."
	case shapeVoidLifecycle:
		return "func(ctx) — void lifecycle method. Same shape as Lifecycle but no error return."
	case shapePoisonAccessor:
		return "func() error — poison-state accessor. Returns nil on a fresh impl, returns the configured sentinel after a poisoned-factory builds it."
	case shapeStreamReader:
		return "func(ctx) iter.Seq[V] / iter.Seq2[V, error] — stream reader. Must complete, respect ctx cancellation, support break, and be re-entrant."
	case shapeStreamConsumer:
		return "func(ctx, S) (V, error) — stream consumer. Reads from S (typically io.Reader) and returns a parsed value."
	}
	return ""
}

// BaselineSubtests returns the subtests the per-shape AssertXBaseline
// runs unconditionally against this method. Each entry mirrors the
// runtime t.Run name (or its parameterized template) so a failure
// path from `go test -v` maps directly back to the corresponding
// runtime helper in suite/<shape>.go.
func (m MethodView) BaselineSubtests() []ContractDoc {
	switch m.ShapeName() {
	case shapePure:
		return []ContractDoc{
			{subtestSmoke, "fail-fast bare invocation; catches panics from a broken Factory or panic-on-call"},
			{"returns expected", "value matches the configured sample"},
			{
				subtestRespectsCtxSt,
				"Pure has no ctx parameter — structural smoke proves no goroutine-local ctx subverts the no-ctx contract",
			},
			{"deterministic", descConsistent3},
			{
				"rejects invalid (structural — no inputs)",
				"Pure has no input — total over its empty domain; structural smoke",
			},
			{subtestConcurrent, descConcurrent10},
		}
	case shapePredicate:
		return []ContractDoc{
			{subtestSmoke, descSmokeBare},
			{"predicate returns expected", "result matches the configured value"},
			{subtestRespectsCtxSt, "Predicate has no ctx — structural smoke"},
			{"predicate consistent", descConsistent3},
			{"rejects invalid (structural — no inputs)", "Predicate has no input — structural smoke"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeAggregator:
		return []ContractDoc{
			{subtestSmoke, descSmokeBare},
			{"aggregator returns expected", "value matches the configured sample with no error"},
			{subtestRespectsCtx, descCtxCanceled},
			{"aggregator consistent", descConsistent3},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeMultiAggregator:
		return []ContractDoc{
			{subtestSmoke, descSmokeBare},
			{"multi-aggregator returns expected", "both values match their configured samples with no error"},
			{subtestRespectsCtx, descCtxCanceled},
			{"multi-aggregator consistent", descConsistent3},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeReader:
		return []ContractDoc{
			{subtestSmoke, "fail-fast bare invocation with the sample key"},
			{"returns for key <sample>", "happy-path read against the configured sample"},
			{subtestRespectsCtx, descCtxCanceled},
			{subtestConsistent, "three sequential reads of the same key yield equal results"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeReaderNoError:
		return []ContractDoc{
			{subtestSmoke, descSmokeBare},
			{subtestReturnsForKey, "happy-path read against the configured sample"},
			{subtestRespectsCtx, "ctx.Done() surfaces a degraded/zero return without panic"},
			{subtestConsistent, "three sequential reads yield equal results"},
			{"zero on unknown key", "missing key returns the zero V (no sentinel discipline)"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeReaderWithBool:
		return []ContractDoc{
			{subtestSmoke, descSmokeBare},
			{"returns for key (ok=true)", "known key returns ok=true with the expected value"},
			{subtestRespectsCtx, "ctx.Done() returns ok=false rather than blocking"},
			{subtestConsistent, "three sequential reads yield equal (value, ok) pairs"},
			{"missing key (ok=false)", "unknown key returns ok=false"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapePointerReader:
		return []ContractDoc{
			{subtestSmoke, descSmokeBare},
			{subtestReturnsForKey, "known key returns non-nil with the expected value"},
			{subtestRespectsCtx, "ctx.Done() returns nil rather than blocking"},
			{subtestConsistent, "three sequential reads return equal *V"},
			{"nil on unknown key", "unknown key returns nil"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeLookup:
		return []ContractDoc{
			{subtestSmoke, descSmokeBare},
			{subtestReturnsForKey, "known key returns ok=true with the expected (value, R)"},
			{subtestRespectsCtx, "ctx.Done() surfaces an honest miss"},
			{"consistent lookups", "three sequential lookups yield equal results"},
			{"missing key", "unknown key returns ok=false"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeMultiReader:
		return []ContractDoc{
			{subtestSmoke, descSmokeBare},
			{subtestReturnsForKey, "known key returns both values with no error"},
			{subtestRespectsCtx, descCtxCanceled},
			{subtestConsistent, "three sequential reads yield equal value pairs"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeBatchReader:
		return []ContractDoc{
			{subtestSmoke, "fail-fast bare invocation with the sample keys"},
			{"returns all (batch)", "result slice matches the configured sample"},
			{subtestRespectsCtx, descCtxCanceled},
			{"batch consistent", "three sequential batch reads yield equal slices"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeWriter:
		return []ContractDoc{
			{subtestSmoke, descSmokeBare},
			{"write succeeds", "happy-path write against the configured sample"},
			{subtestRespectsCtx, "ctx.Done() surfaces context.Canceled before mutation"},
			{"idempotent", "writing the same value twice returns nil twice"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeCompositeWriter:
		return []ContractDoc{
			{subtestSmoke, descSmokeBare},
			{"composite write succeeds", "happy-path write against the (key, value) pair"},
			{subtestRespectsCtx, descCtxCanceled},
			{"composite idempotent", "writing the same pair twice returns nil twice"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeMultiArgWriter:
		return []ContractDoc{
			{subtestSmoke, "fail-fast bare invocation with the sample args"},
			{"multi-arg write succeeds", "happy-path write against the configured args"},
			{subtestRespectsCtx, descCtxCanceled},
			{"multi-arg idempotent", "writing the same args twice returns nil twice"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeDeleter:
		return []ContractDoc{
			{subtestSmoke, "fail-fast bare invocation with the sample key"},
			{"delete succeeds", "happy-path delete returns nil"},
			{subtestRespectsCtx, "ctx.Done() surfaces context.Canceled before mutation"},
			{"delete idempotent", "deleting the same key twice returns nil twice"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeMutator:
		return []ContractDoc{
			{subtestSmoke, descSmokeBare},
			{"mutator succeeds", "happy-path mutation completes without panic"},
			{subtestRespectsCtx, "ctx.Done() short-circuits the mutation"},
			{"mutator idempotent", "applying the same mutation twice does not panic"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeLifecycle:
		return []ContractDoc{
			{subtestSmoke, descSmokeBare},
			{"lifecycle succeeds", "first call returns nil"},
			{subtestRespectsCtx, descCtxCanceled},
			{"lifecycle idempotent", "second call returns nil too"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeVoidLifecycle:
		return []ContractDoc{
			{subtestSmoke, descSmokeBare},
			{"void-lifecycle succeeds", "first call completes without panic"},
			{subtestRespectsCtx, "ctx.Done() short-circuits the call"},
			{"void-lifecycle idempotent", "second call also completes"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapePoisonAccessor:
		return []ContractDoc{
			{subtestSmoke, descSmokeBare},
			{"nil on fresh", "fresh impl returns nil — poison only surfaces post-poisoning"},
			{subtestRespectsCtxSt, "PoisonAccessor has no ctx — structural smoke"},
			{"poison consistent", "repeated calls return the same nil result on a fresh impl"},
			{subtestConcurrent, descConcurrent10},
		}
	case shapeStreamReader:
		return []ContractDoc{
			{subtestSmoke, "fail-fast iter drain on a fresh impl"},
			{"completes", "iter terminates without producing an unbounded stream"},
			{subtestRespectsCtx, "ctx.Done() halts iteration; no further yields"},
			{"reentrant", "two iterators against the same impl yield equal sequences"},
			{"respects break", "early break from the for-range halts production cleanly"},
			{"concurrent safe", "4 workers iterate independently under -race"},
		}
	case shapeStreamConsumer:
		return []ContractDoc{
			{"smoke (default sample)", "fail-fast invocation with bytes.NewReader([]byte(\"test-data\"))"},
			{"succeeds (default sample)", "default sample produces the expected V"},
			{"respects context (default sample)", "ctx.Done() surfaces context.Canceled"},
			{"concurrent safe (default sample)", "4 workers × 10 iterations, fresh stream per call, under -race"},
		}
	}
	return nil
}

// OptionalBaselineExtras returns baseline extras that fire only when
// a signature- or option-driven gate is satisfied — e.g.
// AssertReturnsSentinel only emits when the method declares
// //testkit:errors with a nameable sentinel and the sample key
// differs from the zero key. Returns nil when no extras apply.
func (m MethodView) OptionalBaselineExtras() []ContractDoc {
	var out []ContractDoc
	switch m.ShapeName() {
	case shapeReader:
		if m.HasFirstSentinel() {
			out = append(out, ContractDoc{
				"returns sentinel for unknown key",
				fmt.Sprintf("//testkit:errors declared %s — surfaces it on lookup of the zero key", m.FirstSentinel()),
			})
		}
	case shapeBatchReader, shapeMultiReader:
		if m.HasFirstSentinel() {
			out = append(out, ContractDoc{
				"returns sentinel for unknown key",
				fmt.Sprintf("//testkit:errors declared %s — surfaces it on lookup of the zero key", m.FirstSentinel()),
			})
		}
	case shapeWriter, shapeCompositeWriter, shapeMultiArgWriter:
		if m.HasFirstSentinel() {
			out = append(out, ContractDoc{
				"rejects invalid input",
				fmt.Sprintf("//testkit:errors declared %s — writing the zero value returns it", m.FirstSentinel()),
			})
		}
	case shapeDeleter:
		if m.HasFirstSentinel() {
			out = append(out, ContractDoc{
				"returns not-found on unknown key",
				fmt.Sprintf("//testkit:errors declared %s — deleting the zero key returns it", m.FirstSentinel()),
			})
		}
	case shapeLifecycle:
		out = append(out, ContractDoc{
			descRejectInvOpt,
			"under suite.WithInvalidFactory the lifecycle method must error on a misconfigured impl",
		})
	case shapeVoidLifecycle:
		out = append(out, ContractDoc{
			descRejectInvOpt,
			"under suite.WithInvalidFactory the void method must complete without panic on a misconfigured impl",
		})
	case shapeMutator:
		out = append(out, ContractDoc{
			descRejectInvOpt,
			"under suite.WithInvalidFactory the mutator must complete without panic on a misconfigured impl",
		})
	case shapeMultiAggregator:
		if m.HasFirstSentinel() {
			out = append(out, ContractDoc{
				"returns sentinel (when InvalidFactory option supplied)",
				fmt.Sprintf(
					"under suite.WithInvalidFactory + //testkit:errors %s, the misconfigured impl returns the sentinel",
					m.FirstSentinel(),
				),
			})
		}
	case shapePoisonAccessor:
		if m.HasFirstSentinel() {
			out = append(out, ContractDoc{
				"rejects invalid (when PoisonedFactory option supplied)",
				fmt.Sprintf(
					"under suite.WithPoisonedFactory + //testkit:errors %s, the poisoned impl surfaces the sentinel",
					m.FirstSentinel(),
				),
			})
		}
	case shapeAggregator:
		if m.HasBounded() {
			out = append(out, ContractDoc{
				"aggregator bounded (when AggregatorBounds option supplied)",
				fmt.Sprintf(
					"//testkit:bounded declares range [%s..%s]; result must lie within",
					m.BoundedMin(),
					m.BoundedMax(),
				),
			})
		}
	}
	return out
}

// DirectiveContracts returns one entry per directive attached to
// this method that drives an extra subtest below the baseline. The
// dispatcher emits each directive's partial inside the same per-method
// t.Run, so failures appear under the same path as the baseline.
func (m MethodView) DirectiveContracts() []ContractDoc {
	var out []ContractDoc
	if m.IsDeprecated() {
		out = append(out, ContractDoc{
			"deprecated smoke",
			"//testkit:deprecated — the method still answers; the marker is documentation, not a runtime gate",
		})
	}
	if m.HasRetrySucceeds() {
		out = append(out, ContractDoc{
			fmt.Sprintf("retry succeeds within %d attempts", m.RetryN()),
			"//testkit:retry-succeeds-on-attempt — the method must succeed within N invocations under the runtime's retry harness",
		})
	}
	if m.HasOrderAfter() {
		out = append(out, ContractDoc{
			"order after",
			fmt.Sprintf(
				"//testkit:order-after %s — invoking %s before this method changes its observed result",
				m.OrderAfterTarget(),
				m.OrderAfterTarget(),
			),
		})
	}
	if m.HasPartition() {
		out = append(out, ContractDoc{
			"partition isolation",
			"//testkit:partition — concurrent invocations on disjoint partition keys must not contend",
		})
	}
	if m.HasWrappedVia() {
		out = append(out, ContractDoc{
			"wrapped via",
			fmt.Sprintf(
				"//testkit:wrapped-via %s — internal errors must wrap %s so callers see the canonical type",
				m.WrappedViaTarget(),
				m.WrappedViaTarget(),
			),
		})
	}
	if m.IsIdempotent() {
		out = append(out, ContractDoc{
			"idempotent (second call)",
			"//testkit:idempotent — a second identical call does not error or alter observable state beyond the first",
		})
	}
	if m.IsPure() {
		out = append(out, ContractDoc{
			"pure (impl-independent)",
			"//testkit:pure — output depends only on inputs; multiple impls must agree given the same input",
		})
	}
	if m.IsCacheable() {
		out = append(out, ContractDoc{
			"cacheable (repeated reads)",
			"//testkit:cacheable — three sequential calls return equal values; the impl is free to cache without divergence",
		})
	}
	if m.IsMonotonic() {
		out = append(out, ContractDoc{
			"monotonic non-decreasing",
			"//testkit:monotonic — the result type satisfies cmp.Ordered and never decreases across sequential calls",
		})
	}
	if m.IsConcurrent() {
		out = append(out, ContractDoc{
			"concurrent strict (16×25)",
			"//testkit:concurrent — 16 workers × 25 iterations under -race; explicit strict-concurrency guarantee beyond the baseline",
		})
	}
	if m.IsConcurrentReaders() {
		out = append(out, ContractDoc{
			"concurrent readers parallel (32 readers)",
			"//testkit:concurrent-readers — 32 simultaneous readers; verifies reader-side contention is non-blocking",
		})
	}
	if m.IsNilSafe() {
		out = append(out, ContractDoc{
			"nil-safe (no panic)",
			"//testkit:nil-safe — the method handles nil-bearing inputs without panic",
		})
	}
	if m.IsAtomic() {
		out = append(out, ContractDoc{
			"atomic (no observable trace on failure)",
			"//testkit:atomic — when the method errors, observable state is unchanged",
		})
	}
	if m.HasBounded() {
		out = append(out, ContractDoc{
			"bounded range",
			fmt.Sprintf(
				"//testkit:bounded %s..%s — return value lies within the declared inclusive range",
				m.BoundedMin(),
				m.BoundedMax(),
			),
		})
	}
	if m.HasTimeout() {
		out = append(out, ContractDoc{
			"timeout",
			fmt.Sprintf(
				"//testkit:timeout %s — the method completes within the declared deadline",
				m.TimeoutDuration(),
			),
		})
	}
	if m.HasSideEffect() {
		out = append(out, ContractDoc{
			"side-effect observable",
			fmt.Sprintf(
				"//testkit:side-effect Method=%s — the named observation method reflects the effect",
				m.SideEffectMethod(),
			),
		})
	}
	if m.HasValidates() {
		out = append(out, ContractDoc{
			"validates input (zero rejected)",
			fmt.Sprintf(
				"//testkit:validates %s — invocation with the zero value of %s returns a non-nil error",
				m.ValidatesField(),
				m.ValidatesField(),
			),
		})
	}
	if m.HasHooks() {
		out = append(out, ContractDoc{
			"hooks fire",
			fmt.Sprintf(
				"//testkit:hooks %s — the named hooks fire in declared order during the method's execution",
				strings.Join(m.HookNames(), ", "),
			),
		})
	}
	if m.HasEventually() {
		out = append(out, ContractDoc{
			"eventually converges",
			fmt.Sprintf(
				"//testkit:eventually %s — the method's observable state converges to the expected value within the declared timeout (polled, no time.Sleep)",
				m.EventuallyTimeout(),
			),
		})
	}
	if m.HasScope() {
		out = append(out, ContractDoc{
			"scope auth required",
			fmt.Sprintf(
				"//testkit:scope %s — invocation without the named scope context returns ErrUnauthorized",
				m.ScopeName(),
			),
		})
	}
	if m.HasPagination() {
		out = append(out, ContractDoc{
			"paginates",
			fmt.Sprintf(
				"//testkit:pagination %s — repeated calls with the cursor field eventually drain the corpus and the cursor terminates",
				m.PaginationCursor(),
			),
		})
	}
	if m.HasLease() {
		out = append(out, ContractDoc{
			"lease acquire/release",
			fmt.Sprintf(
				"//testkit:lease Release=%s — acquiring with this method, releasing via %s, releases the lease for re-acquisition",
				m.LeaseRelease(),
				m.LeaseRelease(),
			),
		})
	}
	if m.HasReadAfterWrite() {
		out = append(out, ContractDoc{
			"read after write",
			fmt.Sprintf(
				"//testkit:read-after-write Reader=%s — a write of (key, value) followed by a read returns the written value",
				m.ReadAfterWriteReader(),
			),
		})
	}
	if m.HasDeleteRemoves() {
		out = append(out, ContractDoc{
			"delete removes",
			fmt.Sprintf(
				"//testkit:delete-removes Reader=%s — a delete of key followed by a read surfaces the missing-key sentinel",
				m.DeleteRemovesReader(),
			),
		})
	}
	if m.HasStreamReflectsMutations() {
		out = append(out, ContractDoc{
			"stream reflects mutations",
			fmt.Sprintf(
				"//testkit:stream-reflects-mutations Stream=%s — values written via this method appear in the named stream",
				m.StreamReflectsMutationsStream(),
			),
		})
	}
	if m.HasLifecycleAfterClose() {
		out = append(out, ContractDoc{
			"lifecycle after close",
			fmt.Sprintf(
				"//testkit:lifecycle-after-close Reader=%s — calls to %s after this method returns ErrClosed (or equivalent)",
				m.LifecycleAfterCloseReader(),
				m.LifecycleAfterCloseReader(),
			),
		})
	}
	if m.HasCRDTMerge() {
		out = append(out, ContractDoc{
			"crdt merge",
			fmt.Sprintf(
				"//testkit:crdt-merge Other=%s — paired with %s, this method's output is invariant under merge order",
				m.CRDTMergeOther(),
				m.CRDTMergeOther(),
			),
		})
	}
	return out
}
