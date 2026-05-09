// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench

import (
	"fmt"
	"go/types"
	"strconv"
	"strings"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/allocs"
	"go.thesmos.sh/testkit/generator/spec/latency"
	"go.thesmos.sh/testkit/generator/spec/percentiles"
	"go.thesmos.sh/testkit/generator/spec/sample"
)

// Name is the generator's CLI subcommand name. Exported so other
// packages reference bench by typed constant rather than the raw
// "bench" string.
const Name = "bench"

// Data is the top-level template input for one bench generation.
// The renderer emits one file containing the bench driver
// (`Benchmark<Iface>Contract`), the per-method bench helpers, and
// the option-pattern scaffolding that lets consumers plug in
// per-method primitives + a prePopulate seeder.
//
// Data wraps [spec.Data] so bench reads the shared analysis output —
// methods, shape classifications, directive payloads — through one
// pointer. Bench-specific naming + per-method projection live on the
// wrapper.
type Data struct {
	// Spec is the shared analysis result.
	Spec *spec.Data

	// PackageName / Imports / GenQualifier / ImplImportPath are the
	// standard output-file fields. The bench driver lives in the
	// external _test package alongside the impl's tests.
	PackageName    string
	Imports        []generator.Import
	ImplImportPath string
	GenQualifier   string

	// IfaceName is the bare interface name ("Store", "Cache") —
	// drives the public driver names below.
	IfaceName string

	// LowerIfaceName is the camelCase interface name used for
	// unexported helpers (`storeBenchConfig`, `storeBenchCustomSub`).
	LowerIfaceName string

	// DriverName is the public driver name — `Benchmark<Iface>Contract`.
	DriverName string

	// OptionTypeName is the public option function-type name —
	// `<Iface>BenchOption`. Consumers compose options from this set.
	OptionTypeName string

	// ConfigTypeName is the unexported config-struct name —
	// `<lowerIface>BenchConfig`. The driver folds options into it
	// via [NewConfigFunc].
	ConfigTypeName string

	// CustomSubTypeName is the unexported (name, fn) tuple name —
	// `<lowerIface>BenchCustomSub`. Backs the `<Iface>BenchCustom`
	// escape hatch.
	CustomSubTypeName string

	// PrePopulateName is the public option constructor name —
	// `<Iface>BenchPrePopulate`. Consumers call it once to seed
	// every freshly-factory'd impl before per-method benchmarks run.
	PrePopulateName string

	// CustomName is the public option constructor name —
	// `<Iface>BenchCustom`. Consumers use it to drop in arbitrary
	// per-impl benchmarks the shape vocabulary doesn't cover.
	CustomName string

	// NewConfigFunc is the unexported constructor name —
	// `new<Iface>BenchConfig`. The driver calls it to fold options.
	NewConfigFunc string

	// Methods is one [MethodView] per interface method, populated
	// by the Pipeline's PostEnrich step.
	Methods []MethodView
}

// HasContent reports whether the interface has at least one method
// worth benchmarking. Reads [Spec] because the Pipeline invokes
// HasContent right after Analyze, before [Project] populates
// [Methods].
func (d *Data) HasContent() bool { return d.Spec != nil && len(d.Spec.Methods) > 0 }

// QualifiedTypeForTest delegates to [spec.Data.QualifiedTypeForTest]
// — the wrapper exists so templates can call .QualifiedTypeForTest
// off the bench view without reaching through .Spec.
func (d *Data) QualifiedTypeForTest() string {
	return d.Spec.QualifiedTypeForTest()
}

// CoveredMethodCount returns the number of non-skip methods (i.e.
// every method that produces a per-method bench helper). Used by
// the package-doc summary so consumers know exactly how much of the
// interface gets benchmarked.
func (d *Data) CoveredMethodCount() int {
	n := 0
	for _, m := range d.Methods {
		if !m.IsIntegrationOnly() {
			n++
		}
	}
	return n
}

// SkippedMethodCount returns the number of methods skipped via
// `//testkit:integration-only`. Drives the
// `Methods: N (M skipped)` line in the package-doc summary.
func (d *Data) SkippedMethodCount() int {
	n := 0
	for _, m := range d.Methods {
		if m.IsIntegrationOnly() {
			n++
		}
	}
	return n
}

// ShapeBreakdown returns a one-line summary of which shapes get
// benchmarked and which methods classify into each — e.g.:
//
//	"Reader (Get), Writer (Put, LegacyPut), Lifecycle (Init)"
//
// Methods marked `//testkit:integration-only` are excluded. Shapes
// are ordered by first appearance so the list is stable across
// regenerations.
func (d *Data) ShapeBreakdown() string {
	type entry struct {
		shape   string
		methods []string
	}
	var order []string
	groups := map[string]*entry{}
	for _, m := range d.Methods {
		if m.IsIntegrationOnly() {
			continue
		}
		s := m.ShapeName()
		if _, ok := groups[s]; !ok {
			groups[s] = &entry{shape: s}
			order = append(order, s)
		}
		groups[s].methods = append(groups[s].methods, m.Name)
	}
	parts := make([]string, 0, len(order))
	for _, s := range order {
		e := groups[s]
		parts = append(parts, fmt.Sprintf("%s (%s)", e.shape, strings.Join(e.methods, ", ")))
	}
	return strings.Join(parts, ", ")
}

// PlugInPoints returns the list of `<Iface>BenchOn<Method>` option
// constructor names for non-skip methods, in source order. Drives
// the package-doc plug-in summary.
func (d *Data) PlugInPoints() []string {
	out := make([]string, 0, len(d.Methods))
	for _, m := range d.Methods {
		if m.IsIntegrationOnly() {
			continue
		}
		out = append(out, m.OnOptionName())
	}
	return out
}

// MethodView wraps a [*spec.Method] with the bench-specific naming
// conventions. Templates call directly into the embedded spec.Method
// for signature-driven helpers (ParamList, ResultDecls, ZeroArgs,
// IsIntegrationOnly, …) and onto MethodView's own methods for
// bench-specific concerns (per-method helper naming, budget gates,
// type-param substitution for the test-time monomorphization).
type MethodView struct {
	// Method is the underlying analyzed method — embedded for
	// promotion so templates can call its receiver methods directly.
	*spec.Method

	// ifaceName is captured at projection time so per-method
	// renderers stay self-contained when emitting subtests qualified
	// by interface name.
	ifaceName string

	// lowerIfaceName mirrors [Data.LowerIfaceName] so per-method
	// templates can name unexported helpers without reaching back
	// through the parent dict.
	lowerIfaceName string

	// substitute rewrites type-parameter names to their concrete
	// test-instantiation forms (e.g. "V" → "string"). Identity for
	// non-generic interfaces; bound by [asTestView] for generics so
	// every Shape* accessor lands the concrete type.
	substitute func(string) string
}

func (m MethodView) sub(s string) string {
	if m.substitute == nil {
		return s
	}
	return m.substitute(s)
}

// IfaceName returns the source interface's bare name.
func (m MethodView) IfaceName() string { return m.ifaceName }

// SignatureSummary renders the method's signature for inclusion in
// per-method docblocks. The leading `func` is stripped so the
// caller can prepend the method name to form a familiar
// declaration-style summary:
//
//	method.SignatureSummary(t) → "(ctx context.Context, key string) (basic.Item, error)"
//
// Types are qualified through the tracker so cross-package types
// appear with the right alias. Param names come from the source
// declaration when available.
func (m MethodView) SignatureSummary(t *generator.ImportTracker) string {
	return strings.TrimPrefix(types.TypeString(m.Signature, t.Qualifier()), "func")
}

// DirectiveLines renders this method's `//testkit:` directives back
// to their canonical source form, one per element. Off=true entries
// are skipped (they're meta-directives that don't reach the
// emission layer); ordering matches source declaration so docblocks
// stay stable across regenerations.
//
//	["//testkit:errors ErrNotFound", "//testkit:atomic"]
//
// Returns an empty slice when the method carries no directives —
// templates render `(none)` in that case.
func (m MethodView) DirectiveLines() []string {
	out := make([]string, 0, len(m.Directives))
	for _, d := range m.Directives {
		if d.Off {
			continue
		}
		line := "//testkit:" + d.Name
		if len(d.Args) > 0 {
			line += " " + strings.Join(d.Args, " ")
		}
		out = append(out, line)
	}
	return out
}

// LowerIfaceName returns the camelCase interface name — used by
// per-method partials when emitting unexported helpers.
func (m MethodView) LowerIfaceName() string { return m.lowerIfaceName }

// HelperName returns the unexported per-method helper function name
// — `bench<Iface><Method>`. The driver delegates each method's
// HotPath / ConcurrentThroughput / opt-in budget gates / plug-in
// dispatch to this helper to keep the driver body small.
func (m MethodView) HelperName() string {
	return "bench" + m.ifaceName + m.Name
}

// OnOptionName returns the public option constructor for this
// method's plug-in primitives — `<Iface>BenchOn<Method>`. Consumers
// pass typed bench primitives (e.g. `bench.Reader[T, K, V]`) through
// this option to extend the per-method bench beyond the always-emit
// HotPath / ConcurrentThroughput.
func (m MethodView) OnOptionName() string {
	return m.ifaceName + "BenchOn" + m.Name
}

// OnFieldName returns the unexported config-struct field that
// accumulates this method's plug-in primitives — `on<Method>`.
func (m MethodView) OnFieldName() string {
	return "on" + m.Name
}

// HasAllocsBudget reports whether the method declares
// //testkit:allocs N. Drives whether the per-method helper emits an
// [bench.AllocsWithin] gate.
func (m MethodView) HasAllocsBudget() bool { return allocs.Has(m.Method) }

// AllocsBudget returns the parsed allocation ceiling. Templates
// emit this as the integer budget literal. Returns 0 when no
// directive is present — pair with [HasAllocsBudget] to gate the
// emission, since 0 is also a valid budget (alloc-free assertion).
func (m MethodView) AllocsBudget() int {
	if p, ok := allocs.Get(m.Method); ok {
		return p.Max
	}
	return 0
}

// HasLatencyBudget reports whether the method declares
// //testkit:latency D. Drives whether the per-method helper emits a
// [bench.LatencyWithin] gate.
func (m MethodView) HasLatencyBudget() bool { return latency.Has(m.Method) }

// LatencyBudgetRaw returns the directive's original duration arg
// (e.g. "100us") for inclusion in docblocks. Pair with
// [LatencyBudgetExpr] which renders the Go literal for emission.
// Returns the empty string when no latency directive is present.
func (m MethodView) LatencyBudgetRaw() string {
	if p, ok := latency.Get(m.Method); ok {
		return p.Raw
	}
	return ""
}

// HasPercentiles reports whether the method declares
// `//testkit:percentiles pXX=Dur ...`. Drives whether the per-method
// helper emits a [bench.LatencyPercentilesWithin] gate.
func (m MethodView) HasPercentiles() bool { return percentiles.Has(m.Method) }

// PercentilesBudgetsExpr renders the per-percentile budgets as a
// `map[float64]time.Duration{...}` Go literal suitable for
// embedding directly in the [bench.LatencyPercentilesWithin] call.
// Each entry carries the original directive arg as a trailing
// comment so durations round-trip stably without locale-sensitive
// formatting (matching the [LatencyBudgetExpr] convention).
//
// Returns the empty string when no percentiles directive is present
// — pair with [HasPercentiles] to gate emission.
//
//	method: //testkit:percentiles p50=10us p99=100us
//
//	→  map[float64]time.Duration{
//	       0.50: time.Duration(10000) /* p50=10us */,
//	       0.99: time.Duration(100000) /* p99=100us */,
//	   }
func (m MethodView) PercentilesBudgetsExpr() string {
	p, ok := percentiles.Get(m.Method)
	if !ok {
		return ""
	}
	var b strings.Builder
	b.WriteString("map[float64]time.Duration{\n")
	for _, budget := range p.Budgets {
		b.WriteString("\t\t\t")
		b.WriteString(strconv.FormatFloat(float64(budget.Percentile)/100.0, 'f', 2, 64))
		b.WriteString(": time.Duration(")
		b.WriteString(strconv.FormatInt(budget.Max.Nanoseconds(), 10))
		b.WriteString(") /* ")
		b.WriteString(budget.Raw)
		b.WriteString(" */,\n")
	}
	b.WriteString("\t\t}")
	return b.String()
}

// PercentilesBudgetsLines returns the directive's args verbatim
// (e.g. ["p50=10us", "p99=100us"]) for inclusion in per-method
// docblocks. Empty when no percentiles directive is present.
func (m MethodView) PercentilesBudgetsLines() []string {
	p, ok := percentiles.Get(m.Method)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(p.Budgets))
	for _, b := range p.Budgets {
		out = append(out, b.Raw)
	}
	return out
}

// HasSampleDirective reports whether the method carries
// `//testkit:sample <Func>...`. Used to decide whether the per-
// method helper emits a hot-path-may-be-measuring-the-wrong-path
// warning at run time.
func (m MethodView) HasSampleDirective() bool { return sample.Has(m.Method) }

// NeedsSampleWarning reports whether the per-method helper should
// emit a `b.Logf` warning above the hot-path. True when the method
// has non-ctx params AND no //testkit:sample directive — the
// generator falls back to synthesized literals (e.g. `"test-key"`)
// that require the factory to pre-seed matching values for the
// hot-path to land on the success path. The warning surfaces that
// contract expectation at run time so consumers reading benchmark
// output don't silently measure the not-found error path.
func (m MethodView) NeedsSampleWarning() bool {
	return m.NonCtxParamCount() > 0 && !m.HasSampleDirective()
}

// SampleInputsRendered returns the rendered sample expressions for
// each non-ctx parameter in source order — the values the always-
// emitted hot-path benchmark passes when invoking the method. This
// is the contract surface consumers must align their factory's seed
// against: e.g. a Reader hot-path calling `Get(ctx, "test-key")`
// only measures the happy path when the factory pre-seeds
// `"test-key"` to a real value.
//
// Templates emit this list under a `Sample inputs:` line in the
// per-method docblock so consumers can see exactly what to seed
// without reading the generated body. Empty slice means the method
// takes no non-ctx params (Aggregator/Lifecycle/Pure shapes).
//
// Override the synthesized literals via `//testkit:sample` —
// SampleParamAt resolves directive-supplied call expressions ahead
// of the type-driven defaults.
func (m MethodView) SampleInputsRendered(t *generator.ImportTracker) []string {
	n := m.NonCtxParamCount()
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, m.SampleParamAt(i, t))
	}
	return out
}

// LatencyBudgetExpr returns a compile-stable Go expression for the
// declared latency budget — `time.Duration(<ns>)` carrying the
// parsed nanosecond count, with the original directive arg as a
// trailing comment for readability:
//
//	time.Duration(100000) /* 100us */
//
// The directive parser validates the duration; the rendered literal
// captures the exact value without depending on locale-sensitive
// formatting (time.Duration.String() emits "100µs" with a non-ASCII
// rune, which gofmt trips on in some environments).
//
// Returns the empty string when the method carries no latency
// directive — pair with [HasLatencyBudget] to gate emission.
func (m MethodView) LatencyBudgetExpr() string {
	p, ok := latency.Get(m.Method)
	if !ok {
		return ""
	}
	return fmt.Sprintf("time.Duration(%d) /* %s */", p.Max.Nanoseconds(), p.Raw)
}

// ShapeName returns the detected shape's display name as a string —
// "Reader", "Writer", "StreamReader", etc. Templates dispatch on
// this to pick the per-shape body partial.
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
// result slot (Lookup's R, Writer-with-result's V).
func (m MethodView) ShapeRetType() string { return m.sub(m.Shape.RetType) }

// ShapeIterValType returns the rendered element type for an
// iter.Seq / iter.Seq2 stream — the V in `iter.Seq[V]` or the
// first arg in `iter.Seq2[V, error]`. Empty for non-stream shapes.
func (m MethodView) ShapeIterValType() string { return m.sub(m.Shape.Iter.ValType) }

// HasShapePrimitive reports whether the method's shape maps to a
// typed primitive in the [bench] package. False only for Unknown
// shape (3+ non-error returns, fallthrough cases) — those methods
// receive only an inline hot-path measurement and the raw
// `func(*testing.B, T)` plug-in slot.
func (m MethodView) HasShapePrimitive() bool {
	return m.ShapeName() != "Unknown"
}

// EmitsConcurrent reports whether the per-method bench helper
// should emit ConcurrentThroughput by default. Lifecycle,
// VoidLifecycle, and Unknown shapes default to no — concurrent
// invocation of Init/Reset/Close on a shared impl is rarely safe,
// and Unknown shapes have no typed primitive that can guarantee
// safe concurrent dispatch. Every other shape defaults to yes.
//
// Consumers needing concurrent measurement on a default-skip
// method supply a typed primitive through the per-method
// `<Iface>BenchOn<Method>` plug-in slot, or a free-form
// sub-benchmark via `<Iface>BenchCustom`.
func (m MethodView) EmitsConcurrent() bool {
	switch m.ShapeName() {
	case "Lifecycle", "VoidLifecycle", "Unknown":
		return false
	}
	return true
}

// OnMethodBenchType returns the rendered Go type expression for the
// `On<Method>` option's plug-in slot, qualified with `iface`. The
// type maps the method's shape to the corresponding [bench.<Shape>]
// closure alias:
//
//	Reader    → bench.Reader[Iface, K, V]
//	Aggregator → bench.Aggregator[Iface, R]
//	...
//
// MultiArgWriter requires the three positional non-ctx param types,
// resolved through the import tracker. Shapes whose typed primitive
// can't be expressed (Unknown, MultiArgWriter with arity ≠ 3) fall
// back to the raw `func(*testing.B, Iface)` form so consumers can
// still drop in inline benchmarks via the option pattern.
func (m MethodView) OnMethodBenchType(iface string, t *generator.ImportTracker) string {
	switch m.ShapeName() {
	case "Reader":
		return "bench.Reader[" + iface + ", " + m.ShapeKeyType() + ", " + m.ShapeValType() + "]"
	case "ReaderNoError":
		return "bench.ReaderNoError[" + iface + ", " + m.ShapeKeyType() + ", " + m.ShapeValType() + "]"
	case "ReaderWithBool":
		return "bench.ReaderWithBool[" + iface + ", " + m.ShapeKeyType() + ", " + m.ShapeValType() + "]"
	case "Lookup":
		return "bench.Lookup[" + iface + ", " + m.ShapeKeyType() + ", " + m.ShapeValType() + ", " + m.ShapeRetType() + "]"
	case "PointerReader":
		return "bench.PointerReader[" + iface + ", " + m.ShapeKeyType() + ", " + m.ShapeValType() + "]"
	case "MultiReader":
		return "bench.MultiReader[" + iface + ", " + m.ShapeKeyType() + ", " + m.ShapeValType() + ", " + m.ShapeValType2() + "]"
	case "BatchReader":
		return "bench.BatchReader[" + iface + ", " + m.ShapeKeyType() + ", " + m.ShapeValType() + "]"
	case "Writer":
		return "bench.Writer[" + iface + ", " + m.ShapeValType() + "]"
	case "CompositeWriter":
		return "bench.CompositeWriter[" + iface + ", " + m.ShapeKeyType() + ", " + m.ShapeValType() + "]"
	case "Mutator":
		return "bench.Mutator[" + iface + ", " + m.ShapeValType() + "]"
	case "Deleter":
		return "bench.Deleter[" + iface + ", " + m.ShapeKeyType() + "]"
	case "MultiArgWriter":
		// bench.MultiArgWriter is parametrized over exactly three
		// positional params (P1, P2, P3). Methods with 4+ params
		// can't be wired to the typed primitive — the option's
		// plug-in slot collapses to the raw inline form so consumers
		// can still benchmark them.
		if m.NonCtxParamCount() != 3 {
			return "func(*testing.B, " + iface + ")"
		}
		return "bench.MultiArgWriter[" + iface + ", " +
			m.ParamTypeAt(0, t) + ", " +
			m.ParamTypeAt(1, t) + ", " +
			m.ParamTypeAt(2, t) + "]"
	case "Aggregator":
		return "bench.Aggregator[" + iface + ", " + m.ShapeValType() + "]"
	case "MultiAggregator":
		return "bench.MultiAggregator[" + iface + ", " + m.ShapeValType() + ", " + m.ShapeValType2() + "]"
	case "StreamReader":
		return "bench.Stream[" + iface + ", " + m.ShapeIterValType() + "]"
	case "StreamConsumer":
		return "bench.StreamConsumer[" + iface + ", " + m.ShapeKeyType() + ", " + m.ShapeValType() + "]"
	case "Pure":
		return "bench.Pure[" + iface + ", " + m.ShapeValType() + "]"
	case "Predicate":
		return "bench.Predicate[" + iface + "]"
	case "PoisonAccessor":
		return "bench.PoisonAccessor[" + iface + "]"
	case "Lifecycle":
		return "bench.Lifecycle[" + iface + "]"
	case "VoidLifecycle":
		return "bench.VoidLifecycle[" + iface + "]"
	}
	// Unknown shape: raw inline-bench callback.
	return "func(*testing.B, " + iface + ")"
}

// ParamTypeAt returns the rendered type for the i-th non-context
// parameter, with type-parameter substitution applied for the test
// view.
func (m MethodView) ParamTypeAt(i int, t *generator.ImportTracker) string {
	return m.sub(m.Method.ParamTypeAt(i, t))
}

// SampleParamAt overrides the embedded [spec.Method.SampleParamAt]
// to apply type-parameter substitution and to honor the
// `//testkit:sample <Func>...` directive when present. With a
// sample directive, the i-th param resolves to `<Func>()` (the
// consumer's constructor invocation) — the same call expression
// the consumer's factory uses when seeding the impl, so the
// hot-path measurement lands on the success path. Falls back to
// the synthesized literal (with substitute applied) when no
// directive is present.
func (m MethodView) SampleParamAt(i int, t *generator.ImportTracker) string {
	if p, ok := sample.Get(m.Method); ok && i >= 0 && i < len(p.Calls) {
		return p.Calls[i] + "()"
	}
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
