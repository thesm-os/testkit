// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package spec_test

import (
	"go/types"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/shape"
	"go.thesmos.sh/testkit/generator/spec"
)

func TestData(t *testing.T) {
	t.Parallel()

	t.Run("Method embeds MethodInfo and exposes promoted fields", func(t *testing.T) {
		t.Parallel()
		m := spec.Method{
			MethodInfo: generator.MethodInfo{Name: "Get"},
			Shape:      shape.Info{Shape: shape.Reader, KeyType: "string", ValType: "Item"},
		}
		// Promotion check: Name is reachable directly from spec.Method.
		testkit.Equal(t, m.Name, "Get", "MethodInfo promotion")
		testkit.Equal(t, m.Shape.Shape, shape.Reader, "Shape attached")
	})

	t.Run("Attachments map starts nil and tolerates Set", func(t *testing.T) {
		t.Parallel()
		var m spec.Method
		testkit.True(t, m.Attachments == nil, "Attachments starts nil")
		spec.Set(&m.Attachments, "any", 42)
		testkit.True(t, spec.Has(m.Attachments, "any"), "Set populated the map")
	})

	t.Run("Data composes Package, Interface, Methods, Tracker", func(t *testing.T) {
		t.Parallel()
		// Smoke test: a freshly-allocated Data has the expected zero values
		// and accepts assignment without panic.
		d := &spec.Data{
			Args:    []string{"Store"},
			Tracker: generator.NewImportTracker("p"),
		}
		testkit.Len(t, d.Args, 1, "Args attached")
		testkit.True(t, d.Tracker != nil, "Tracker attached")
		testkit.Equal(t, d.Tracker.LocalPkg(), "p", "Tracker carries local pkg")
	})

	t.Run("NonCtxParamCount + NonCtxParamAt skip leading ctx", func(t *testing.T) {
		t.Parallel()
		// Apply(ctx, key string, item Item) error → 2 non-ctx params.
		_, byName := analyzeFixture(t, "basic", "Sampler")
		apply := byName["Apply"]
		testkit.Equal(t, apply.NonCtxParamCount(), 2, "Apply has 2 non-ctx params")
		testkit.Equal(t, apply.NonCtxParamAt(0).String(), "string", "param 0 is string")
		testkit.Equal(t, apply.NonCtxParamAt(1).String(),
			"go.thesmos.sh/testkit/generator/testdata/basic.Item",
			"param 1 is basic.Item")
	})

	t.Run("ParamFieldAsserts skips chan/func-typed params", func(t *testing.T) {
		t.Parallel()
		// Synthetic method (ch chan int) — chan is non-assertable, so
		// the only non-ctx param is dropped, yielding empty asserts.
		chanSig := types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(0, nil, "ch", types.NewChan(types.SendRecv, types.Typ[types.Int]))),
			nil, false)
		m := spec.Method{MethodInfo: generator.MethodInfo{
			Signature: chanSig,
		}}
		tracker := generator.NewImportTracker("p")
		testkit.Equal(t, m.ParamFieldAsserts(tracker, "label", "call"), "",
			"chan param is non-assertable")
	})

	t.Run("NonCtxParamCount returns NumParams when no ctx is present", func(t *testing.T) {
		t.Parallel()
		// Build a synthetic Method with no context to exercise the
		// false branch of HasContext().
		var noCtxSig types.Type = types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewVar(0, nil, "x", types.Typ[types.String])),
			nil, false)
		m := spec.Method{MethodInfo: generator.MethodInfo{
			Signature: noCtxSig.(*types.Signature),
		}}
		testkit.Equal(t, m.NonCtxParamCount(), 1, "all params count when no ctx")
		testkit.Equal(t, m.NonCtxParamAt(0).String(), "string", "first param is x")
	})
}

func TestMethodTemplateHelpers(t *testing.T) {
	t.Parallel()

	data, byName := analyzeFixture(t, "interfaces", "AllShapes")
	tracker := data.Tracker

	t.Run("HasResults / HasNonErrorResults / HasNonContextParam", func(t *testing.T) {
		t.Parallel()
		// Reset() — no params, no results.
		reset := byName["Reset"]
		testkit.False(t, reset.HasResults(), "Reset has no results")
		testkit.False(t, reset.HasNonErrorResults(), "Reset has no non-error results")
		testkit.False(t, reset.HasNonContextParam(), "Reset has no non-ctx params")

		// Init(ctx) error — error-only result, ctx-only param.
		init := byName["Init"]
		testkit.True(t, init.HasResults(), "Init has results (error)")
		testkit.False(t, init.HasNonErrorResults(), "Init has no non-error results")
		testkit.False(t, init.HasNonContextParam(), "Init has only ctx param")

		// Get(ctx, key) (Record, error) — full shape.
		get := byName["Get"]
		testkit.True(t, get.HasResults(), "Get has results")
		testkit.True(t, get.HasNonErrorResults(), "Get has Record before error")
		testkit.True(t, get.HasNonContextParam(), "Get has key param")

		// Lookup(ctx, key) Record — single non-error result, no error.
		lookup := byName["Lookup"]
		testkit.True(t, lookup.HasNonErrorResults(), "Lookup has non-error result")
	})

	t.Run("ErrFieldName toggles on error return", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, byName["Get"].ErrFieldName(), "Err", "Get returns error")
		testkit.Equal(t, byName["Reset"].ErrFieldName(), "", "Reset has no results")
		testkit.Equal(t, byName["Lookup"].ErrFieldName(), "", "Lookup returns no error")
	})

	t.Run("ParamFieldAssign skips results, lists params", func(t *testing.T) {
		t.Parallel()
		// Get(ctx, key string): two params, no result fields in the assignment.
		got := byName["Get"].ParamFieldAssign(tracker)
		testkit.Equal(t, got, "Ctx: ctx, Key: key", "param fields only")

		// Reset(): empty.
		testkit.Equal(t, byName["Reset"].ParamFieldAssign(tracker), "", "no params, empty")
	})

	t.Run("CallResultStampVars and CallResultStampFallback render result block", func(t *testing.T) {
		t.Parallel()
		get := byName["Get"]
		testkit.Equal(t,
			get.CallResultStampVars(tracker),
			"call.Result = v0\n\tcall.Err = err",
			"single non-error → Result, error → Err")
		testkit.Equal(t,
			get.CallResultStampFallback(tracker),
			"call.Result = f.Result\n\tcall.Err = f.Err",
			"fallback sources from f.<Field>")

		// Method without results yields empty stamps.
		reset := byName["Reset"]
		testkit.Equal(t, reset.CallResultStampVars(tracker), "", "empty when no results")
		testkit.Equal(t, reset.CallResultStampFallback(tracker), "", "empty when no results")
	})

	t.Run("ReturnParams / ReturnFieldAssign / ResultsFromFallback", func(t *testing.T) {
		t.Parallel()
		get := byName["Get"]
		testkit.Assert(t, get.ReturnParams(tracker)).
			Contains("v0 ", "first non-error var name").
			HasSuffix(", err error", "trailing error param").
			Contains("Record", "value type rendered through tracker")
		testkit.Equal(t,
			get.ReturnFieldAssign(tracker),
			"Result: v0, Err: err",
			"struct-literal field-to-var binding")
		testkit.Equal(t,
			get.ResultsFromFallback(tracker),
			"f.Result, f.Err",
			"fallback accessor list")

		// Multi-non-error result indexes the field name.
		fetch := byName["Fetch"] // (ctx, key) (Record, string, error)
		testkit.Equal(t,
			fetch.ResultsFromFallback(tracker),
			"f.Result0, f.Result1, f.Err",
			"multiple non-error results pluralise")

		// Touch(ctx, key) — void; everything empty.
		touch := byName["Touch"]
		testkit.Equal(t, touch.ReturnParams(tracker), "", "empty when no results")
		testkit.Equal(t, touch.ReturnFieldAssign(tracker), "", "empty when no results")
		testkit.Equal(t, touch.ResultsFromFallback(tracker), "", "empty when no results")
	})

	t.Run("CallStructFields covers params, results, variadic, and Err", func(t *testing.T) {
		t.Parallel()

		// Get(ctx, key) (Record, error) — Ctx, Key, Result, Err.
		fields := byName["Get"].CallStructFields(tracker)
		testkit.Len(t, fields, 4, "Get has 4 Call fields")
		testkit.Equal(t, fields[0].Name, "Ctx", "field 0 is Ctx")
		testkit.False(t, fields[0].IsResult, "Ctx is a param")
		testkit.Equal(t, fields[1].Name, "Key", "field 1 is Key")
		testkit.Equal(t, fields[2].Name, "Result", "single non-error result names Result")
		testkit.True(t, fields[2].IsResult, "Result is a result")
		testkit.False(t, fields[2].IsError, "Result is not the error")
		testkit.Equal(t, fields[3].Name, "Err", "trailing error becomes Err")
		testkit.True(t, fields[3].IsError, "Err carries IsError")

		// Many(ctx, keys ...string) ([]Record, error) — variadic last
		// param renders as a slice type, not a slice-of-slice.
		manyFields := byName["Many"].CallStructFields(tracker)
		var keys spec.CallField
		for _, f := range manyFields {
			if f.Name == "Keys" {
				keys = f
			}
		}
		testkit.Equal(t, keys.TypeStr, "[]string", "variadic recorded as slice")

		// Fetch(ctx, key) (Record, string, error) — Result0, Result1, Err.
		fetchFields := byName["Fetch"].CallStructFields(tracker)
		testkit.Equal(t, fetchFields[2].Name, "Result0", "first non-error pluralises")
		testkit.Equal(t, fetchFields[3].Name, "Result1", "second non-error pluralises")
		testkit.Equal(t, fetchFields[4].Name, "Err", "trailing error after non-errors")

		// Inspect(ctx, key) (Record, string, bool) — no error; the
		// trailing bool is a regular non-error result.
		inspectFields := byName["Inspect"].CallStructFields(tracker)
		testkit.Equal(t, inspectFields[len(inspectFields)-1].Name, "Result2",
			"no-error trio names the bool as Result2")
		testkit.False(t, inspectFields[len(inspectFields)-1].IsError, "bool is not an error")
	})

	t.Run("ResultNames and ResultDecls render dispatch-time vars", func(t *testing.T) {
		t.Parallel()
		// Touch is void.
		testkit.Equal(t, byName["Touch"].ResultNames(), "", "void method")
		testkit.Equal(t, byName["Touch"].ResultDecls(tracker), "", "void method")

		// Init returns just error.
		testkit.Equal(t, byName["Init"].ResultNames(), "err", "error-only")
		testkit.Equal(t, byName["Init"].ResultDecls(tracker), "var err error", "error-only decl")

		// Get returns (Record, error).
		testkit.Equal(t, byName["Get"].ResultNames(), "v0, err", "value-and-error names")
		testkit.Assert(t, byName["Get"].ResultDecls(tracker)).
			HasPrefix("var v0 ", "decl starts with var v0").
			HasSuffix("\n\tvar err error", "trailing error decl").
			Contains("Record", "value type rendered")

		// Inspect returns (Record, string, bool) — no error; three vNN.
		testkit.Equal(t, byName["Inspect"].ResultNames(), "v0, v1, v2", "three non-error vars")
	})

	t.Run("FaultReturn renders fault helper return list", func(t *testing.T) {
		t.Parallel()
		// Get(ctx, key) (Record, error) — zero-of-Record, sentinel.
		got := byName["Get"].FaultReturn(tracker, "ErrNotFound")
		testkit.Assert(t, got).
			HasSuffix(", ErrNotFound", "trailing sentinel").
			Contains("Record", "Record zero-value rendered")

		// Init(ctx) error — single error result.
		testkit.Equal(t,
			byName["Init"].FaultReturn(tracker, "ErrNotFound"),
			"ErrNotFound",
			"error-only methods return just the sentinel")

		// Touch(ctx, key) — void; nothing to return.
		testkit.Equal(t,
			byName["Touch"].FaultReturn(tracker, "ErrNotFound"), "",
			"void method renders empty fault return")

		// Inspect(ctx, key) (Record, string, bool) — no error; falls
		// back to ZeroResults so the helper degenerates to the
		// no-fault zero list.
		inspectFault := byName["Inspect"].FaultReturn(tracker, "ErrNotFound")
		testkit.Assert(t, inspectFault).
			IsNotEmpty("non-empty zero list").
			NotContains("ErrNotFound", "no-error method ignores sentinel")
	})

	t.Run("IterReturn detects iter.Seq and iter.Seq2 returns", func(t *testing.T) {
		t.Parallel()
		// All(ctx) iter.Seq[Record].
		all := byName["All"].IterReturn(tracker)
		testkit.True(t, all.IsSeq, "All returns iter.Seq")

		// Scan(ctx) iter.Seq2[Record, error].
		scan := byName["Scan"].IterReturn(tracker)
		testkit.True(t, scan.IsSeq2, "Scan returns iter.Seq2")
		testkit.True(t, scan.Seq2Error, "Scan's second arg is error")

		// Get is not an iter return.
		none := byName["Get"].IterReturn(tracker)
		testkit.False(t, none.IsSeq, "Get is not iter.Seq")
		testkit.False(t, none.IsSeq2, "Get is not iter.Seq2")

		// Statistics has multiple results — IterReturn requires a
		// single-result method, so the early-out branch fires.
		stats := byName["Statistics"].IterReturn(tracker)
		testkit.False(t, stats.IsSeq, "multi-result method short-circuits to zero IterSeqInfo")
	})

	t.Run("SubstituteTypeParams is a no-op for non-generic Data", func(t *testing.T) {
		t.Parallel()
		got := data.SubstituteTypeParams("[]V")
		testkit.Equal(t, got, "[]V", "non-generic data passes input through")
	})

	t.Run("ErrCaptureLHS renders blank discards then err", func(t *testing.T) {
		t.Parallel()
		// Init(ctx) error — single error, no leading blanks.
		testkit.Equal(t, byName["Init"].ErrCaptureLHS(), "err", "single error result")
		// Get(ctx, key) (Record, error) — one blank then err.
		testkit.Equal(t, byName["Get"].ErrCaptureLHS(), "_, err", "one blank then err")
		// Fetch(ctx, key) (Record, string, error) — two blanks then err.
		testkit.Equal(t, byName["Fetch"].ErrCaptureLHS(), "_, _, err", "two blanks then err")
		// Reset() — no error → empty string.
		testkit.Equal(t, byName["Reset"].ErrCaptureLHS(), "", "no error → empty")
		// Lookup(ctx, key) Record — non-error result, no error → empty.
		testkit.Equal(t, byName["Lookup"].ErrCaptureLHS(), "", "non-error-only → empty")
	})

	t.Run("BlankResults renders one underscore per result", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, byName["Reset"].BlankResults(), "", "void method")
		testkit.Equal(t, byName["Init"].BlankResults(), "_", "single error")
		testkit.Equal(t, byName["Get"].BlankResults(), "_, _", "two results")
		testkit.Equal(t, byName["Inspect"].BlankResults(), "_, _, _", "three results")
	})

	t.Run("HasTrailingBool detects ReaderWithBool / Lookup signatures", func(t *testing.T) {
		t.Parallel()
		// Get (Record, error): error-bearing → false.
		testkit.False(t, byName["Get"].HasTrailingBool(), "error-bearing → false")
		// Load(ctx, key) (Record, bool): trailing bool after a value → true.
		testkit.True(t, byName["Load"].HasTrailingBool(), "Load is ReaderWithBool")
		// Inspect(ctx, key) (Record, string, bool): trailing bool after values → true.
		testkit.True(t, byName["Inspect"].HasTrailingBool(), "Inspect is Lookup")
		// Lookup(ctx, key) Record (no bool): false.
		testkit.False(t, byName["Lookup"].HasTrailingBool(), "ReaderNoError has no trailing bool")
		// Reset() (void): false.
		testkit.False(t, byName["Reset"].HasTrailingBool(), "void method")
		// IsHealthy() bool — single-bool result is a Predicate, not a
		// presence flag; FaultMiss would be semantically wrong.
		testkit.False(t, byName["IsHealthy"].HasTrailingBool(),
			"single-bool Predicate is excluded")
	})

	t.Run("MissResultsTuple renders zeros + trailing false for HasTrailingBool", func(t *testing.T) {
		t.Parallel()
		// Load (Record, bool) — one zero before the false.
		testkit.Assert(t, byName["Load"].MissResultsTuple(tracker)).
			HasSuffix(", false", "trailing false").
			Contains("Record", "Record zero rendered")
		// Inspect (Record, string, bool) — two zeros before the false.
		got := byName["Inspect"].MissResultsTuple(tracker)
		testkit.Assert(t, got).
			HasSuffix(", false", "trailing false").
			Contains("Record", "Record zero").
			Contains(`""`, "string zero")
		// Get (Record, error) — error-bearing → empty.
		testkit.Equal(t, byName["Get"].MissResultsTuple(tracker), "",
			"error-bearing has no miss tuple")
		// IsHealthy() bool — single-bool Predicate → empty.
		testkit.Equal(t, byName["IsHealthy"].MissResultsTuple(tracker), "",
			"Predicate has no miss tuple")
	})

	t.Run("HasAssertableNonCtxParams filters by ctx / variadic / type", func(t *testing.T) {
		t.Parallel()
		// Reset() — no params.
		testkit.False(t, byName["Reset"].HasAssertableNonCtxParams(), "no params")
		// Init(ctx) — only ctx.
		testkit.False(t, byName["Init"].HasAssertableNonCtxParams(), "ctx-only")
		// Many(ctx, keys ...string) — variadic-only, dropped.
		testkit.False(t, byName["Many"].HasAssertableNonCtxParams(), "variadic-only")
		// Get(ctx, key) — has assertable Key.
		testkit.True(t, byName["Get"].HasAssertableNonCtxParams(), "Get has Key")
		// Schedule(ctx, key, value, priority) — multiple assertable params.
		testkit.True(t, byName["Schedule"].HasAssertableNonCtxParams(), "Schedule has key/value/priority")
	})

	t.Run("SampleArgs renders the smallest non-zero argument list", func(t *testing.T) {
		t.Parallel()
		// Get(ctx, key string) — ctx → t.Context(), key → "test-key".
		testkit.Equal(t, byName["Get"].SampleArgs(tracker), `t.Context(), "test-key"`,
			"ctx + sampled string")
		// Reset() — empty.
		testkit.Equal(t, byName["Reset"].SampleArgs(tracker), "", "void method")
		// Many(ctx, keys ...string) — variadic dropped.
		testkit.Equal(t, byName["Many"].SampleArgs(tracker), "t.Context()",
			"variadic last excluded")
	})

	t.Run("ParamFieldAsserts emits one Equal per non-ctx assertable param", func(t *testing.T) {
		t.Parallel()
		// Reset / Init — no asserts.
		testkit.Equal(t, byName["Reset"].ParamFieldAsserts(tracker, "label", "call"), "",
			"void method")
		testkit.Equal(t, byName["Init"].ParamFieldAsserts(tracker, "label", "call"), "",
			"ctx-only method")
		// Get(ctx, key string) — one assert on call.Key.
		got := byName["Get"].ParamFieldAsserts(tracker, "Get records args", "call")
		testkit.Assert(t, got).
			HasPrefix("testkit.Equal(t, call.Key,", "asserts call.Key").
			Contains(`"test-key"`, "sample string").
			Contains("Get records args: Key must be recorded", "label propagated")
		// Many(ctx, keys ...string) — variadic-only dropped.
		testkit.Equal(t, byName["Many"].ParamFieldAsserts(tracker, "label", "call"), "",
			"variadic-only method")
		// Schedule(ctx, key, value, priority) — three asserts joined by indent.
		got = byName["Schedule"].ParamFieldAsserts(tracker, "label", "call")
		testkit.Assert(t, got).
			Contains("call.Key,", "Key").
			Contains("call.Value,", "Value").
			Contains("call.Priority,", "Priority").
			Contains("\n\t\t", "indent-joined")
	})

	t.Run("ResultFieldAsserts emits one Equal per non-error assertable result", func(t *testing.T) {
		t.Parallel()
		// Reset / Init — no non-error results.
		testkit.Equal(t, byName["Reset"].ResultFieldAsserts(tracker, "label", "call"), "",
			"void method")
		testkit.Equal(t, byName["Init"].ResultFieldAsserts(tracker, "label", "call"), "",
			"error-only")
		// Get (Record, error) — single assert on call.Result.
		got := byName["Get"].ResultFieldAsserts(tracker, "Get returns", "call")
		testkit.Assert(t, got).
			HasPrefix("testkit.Equal(t, call.Result,", "asserts call.Result").
			Contains("Record", "Record sample").
			Contains("Get returns: Result must be stamped on the Call", "label")
		// Fetch (Record, string, error) — call.Result0 + call.Result1.
		got = byName["Fetch"].ResultFieldAsserts(tracker, "label", "call")
		testkit.Assert(t, got).
			Contains("call.Result0,", "Result0").
			Contains("call.Result1,", "Result1").
			Contains("\n\t\t", "indent-joined")
		// All(ctx) iter.Seq[Record] — non-assertable result skipped.
		testkit.Equal(t, byName["All"].ResultFieldAsserts(tracker, "label", "call"), "",
			"function-typed sole result skipped")
	})

	t.Run("HasAssertableNonErrorResults filters function/channel results", func(t *testing.T) {
		t.Parallel()
		// Reset (void): false.
		testkit.False(t, byName["Reset"].HasAssertableNonErrorResults(),
			"void method has no non-error results")
		// Init (error-only): false.
		testkit.False(t, byName["Init"].HasAssertableNonErrorResults(),
			"error-only has no non-error results")
		// Get (Record, error): true — Record is comparable.
		testkit.True(t, byName["Get"].HasAssertableNonErrorResults(),
			"Record is assertable")
		// All(ctx) iter.Seq[Record] — sole result is iter.Seq, a
		// function type → skipped by isAssertable.
		testkit.False(t, byName["All"].HasAssertableNonErrorResults(),
			"iter.Seq function value isn't assertable")
	})

	t.Run("ZeroResultAsserts emits one Equal per non-error result", func(t *testing.T) {
		t.Parallel()
		// Reset is void → empty string.
		testkit.Equal(t, byName["Reset"].ZeroResultAsserts(tracker, "label"), "",
			"void method emits no asserts")
		// Init returns just error → no non-error results → empty.
		testkit.Equal(t, byName["Init"].ZeroResultAsserts(tracker, "label"), "",
			"error-only emits no asserts")
		// Get returns (Record, error) → exactly one assert on v0.
		got := byName["Get"].ZeroResultAsserts(tracker, "Get default")
		testkit.Assert(t, got).
			HasPrefix("testkit.Equal(t, v0, ", "starts with v0 assert").
			Contains("Get default: v0 must be zero", "label propagated").
			Contains("Record", "zero of Record rendered through tracker")
		// Fetch returns (Record, string, error) → two asserts joined by indent.
		got = byName["Fetch"].ZeroResultAsserts(tracker, "label")
		testkit.Assert(t, got).
			Contains("v0,", "first assert names v0").
			Contains("v1,", "second assert names v1").
			Contains("\n\t\t", "lines join with two-tab indent")
		// All(ctx) iter.Seq[Record] — non-assertable result is skipped
		// silently; the rendered block is empty.
		testkit.Equal(t, byName["All"].ZeroResultAsserts(tracker, "label"), "",
			"function-typed sole result skipped")
	})

	t.Run("SampleResults renders one sample per result with trailing nil for error", func(t *testing.T) {
		t.Parallel()
		// Reset is void → empty.
		testkit.Equal(t, byName["Reset"].SampleResults(tracker), "", "void method")
		// Init returns just error → "nil".
		testkit.Equal(t, byName["Init"].SampleResults(tracker), "nil", "error-only method")
		// Get(ctx, key) (Record, error) — Record sample then nil.
		testkit.Assert(t, byName["Get"].SampleResults(tracker)).
			HasSuffix(", nil", "trailing nil for error slot").
			Contains("Record", "Record sample rendered through tracker")
		// Lookup(ctx, key) Record — single non-error result, no trailing nil.
		testkit.Assert(t, byName["Lookup"].SampleResults(tracker)).
			NotContains("nil", "no error → no nil suffix").
			Contains("Record", "Record sample present")
	})

	t.Run("SampleResultAsserts emits one Equal per non-error result", func(t *testing.T) {
		t.Parallel()
		// Reset (void) and Init (error-only) → empty.
		testkit.Equal(t, byName["Reset"].SampleResultAsserts(tracker, "label"), "",
			"void method emits no asserts")
		testkit.Equal(t, byName["Init"].SampleResultAsserts(tracker, "label"), "",
			"error-only emits no asserts")
		// Get → one assert on v0 with the SAMPLE (not zero).
		got := byName["Get"].SampleResultAsserts(tracker, "Get returns")
		testkit.Assert(t, got).
			HasPrefix("testkit.Equal(t, v0, ", "first assert names v0").
			Contains("Get returns: v0 must match the configured Returns value", "label propagated").
			Contains("Record", "Record sample rendered")
		// Fetch (Record, string, error) → two asserts joined by indent.
		got = byName["Fetch"].SampleResultAsserts(tracker, "label")
		testkit.Assert(t, got).
			Contains("v0,", "first assert names v0").
			Contains("v1,", "second assert names v1").
			Contains("\n\t\t", "lines join with two-tab indent")
		// All(ctx) iter.Seq[Record] — non-assertable result is skipped.
		testkit.Equal(t, byName["All"].SampleResultAsserts(tracker, "label"), "",
			"function-typed sole result skipped")
	})

	t.Run("ZeroArgs renders the smallest valid call argument list", func(t *testing.T) {
		t.Parallel()
		// Get(ctx, key string) → ctx becomes t.Context(), key zero is "".
		testkit.Equal(t, byName["Get"].ZeroArgs(tracker), `t.Context(), ""`,
			"ctx + string zero")
		// Reset() → empty.
		testkit.Equal(t, byName["Reset"].ZeroArgs(tracker), "", "void method")
		// Many(ctx, keys ...string) → variadic dropped; ctx remains.
		testkit.Equal(t, byName["Many"].ZeroArgs(tracker), "t.Context()",
			"variadic last param dropped")
	})
}

func TestSubstituteTypeParamsGeneric(t *testing.T) {
	t.Parallel()
	// Holder[V any] has one type parameter; substitution must replace
	// every standalone occurrence of "V" with the concrete chosen by
	// generator.ConcreteFor.
	data, _ := analyzeFixture(t, "generics", "Holder")
	testkit.True(t, data.IsGeneric, "Holder is generic")
	got := data.SubstituteTypeParams("func() []V")
	testkit.NotContains(t, got, " V", "type-param V replaced")
	testkit.NotContains(t, got, "[]V", "no leftover []V")
}
