// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package spec

import (
	"fmt"
	"go/types"
	"strconv"
	"strings"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/shape"
)

// errVarName is the receiver-side variable name carrying a method's
// trailing error result. Used by every helper that names result vars
// in dispatch-time render output.
const errVarName = "err"

// Data is the analyzed view of one interface, consumed by stub,
// suite, bench, and model generators. Populated by [Analyze].
//
// Each generator wraps Data via composition rather than extending
// it:
//
//	type Data struct {
//	    *spec.Data
//	    BenchSpecificField string
//	}
//
// Embedding keeps the shared model immutable from the generator's
// view; generator-specific state lives on the wrapper.
type Data struct {
	// PackageName is the package declaration the output file uses,
	// computed via [generator.DerivePackageName]. The test-view
	// renderer reshapes this via [generator.BuildTestFileInfo] when
	// emitting an external _test file.
	PackageName string

	// Imports is the resolved import set for the output file,
	// computed from the [Tracker] after analysis (which records
	// every package referenced by the rendered method signatures).
	Imports []generator.Import

	// ImplImportPath is the import path of the impl file's
	// package, computed via [generator.BuildOutputCtx]. The
	// test-view renderer uses it to derive the test file's
	// GenQualifier.
	ImplImportPath string

	// Package is the loaded source package. Generators query the
	// package for type lookups, position information, and import
	// resolution during enrichment.
	Package *generator.Package

	// Interface is the fully populated [generator.InterfaceInfo] —
	// name, type parameters, methods (including embedded), doc,
	// directives at the type level, and source position.
	Interface generator.InterfaceInfo

	// QualifiedType is the source-package-qualified interface
	// reference, suffixed with type-args for generics. Examples:
	//
	//	non-generic:        "basic.Store"
	//	generic Cache:      "generics.Cache[K, V]"
	QualifiedType string

	// TypeParamDecl renders the interface's type-parameter
	// declaration when generic, e.g. "[K comparable, V any]".
	// Empty for non-generic interfaces.
	TypeParamDecl string

	// TypeParamArgs renders just the type-parameter names for
	// instantiation, e.g. "[K, V]". Empty for non-generic.
	TypeParamArgs string

	// IsGeneric is true when [TypeParamDecl] is non-empty.
	IsGeneric bool

	// Methods is one [Method] per interface method, in the order
	// returned by the loader (sorted alphabetically). Holds shape
	// classification plus per-method directive payloads.
	Methods []Method

	// Tracker is the [generator.ImportTracker] used to render type
	// names in generated output. Owned by Data; methods and
	// templates reference it via this single instance to keep the
	// import alias map consistent across all rendering.
	Tracker *generator.ImportTracker

	// Loader is the [generator.Loader] used by consumers that need
	// to resolve symbols in packages other than [Package] — e.g.
	// the sample directive consumer when a directive arg names a
	// fully-qualified function in a fixtures package. Owned by Data
	// so cache hits accumulate across all consumers in one Enrich
	// pass.
	Loader *generator.Loader

	// Args carries the original CLI argv (e.g. ["Store"] or
	// ["Cache", "--variant=fast"]) — used for source attribution
	// and by some enrichers that consult flags.
	Args []string

	// Directives are the //testkit: package-level directives
	// consumed by spec, pre-rendered as source lines for the header
	// partial. Empty when none apply.
	Directives []string
}

// Method is the analyzed view of one interface method. Embeds
// [generator.MethodInfo] so generators can promote Name, Signature,
// Doc, Directives, and Pos directly.
//
// The Attachments map carries directive-driven payloads populated
// by the consumer pass (enrichment) and the emitter pass (mixin
// emissions). Both passes use the directive name as the key, so any
// given method exposes a flat namespace of `directive name → payload`.
//
// The map is nil for fresh [Method] values; use [Set] (which lazily
// allocates) to write.
type Method struct {
	// MethodInfo is the underlying loader output. Embedded for
	// promotion: m.Name, m.Signature, m.Doc, m.Directives, m.Pos.
	generator.MethodInfo

	// Shape is the detected shape from [shape.Classify]. Set during
	// the analysis pass before enrichment runs.
	Shape shape.Info

	// Attachments carries per-directive payloads keyed by directive
	// name. Both enrichers (errors → fault-helper data, sample →
	// bench inputs) and mixin emitters (atomic → assertion suite,
	// roundtrip → assertion suite) write into this single map.
	//
	// The directive [Category] (Mixin vs Enrichment vs ...) is
	// queryable via the directive registry when callers need to
	// distinguish emit-time vs read-time payloads.
	//
	// Generators read via [Get]; consumers and emitters write via
	// [Set].
	Attachments map[string]any
}

// HasDirective reports whether the method's [Attachments] map
// holds a payload for the named directive — i.e. the directive
// fired during [Enrich] and a consumer attached its result. The
// check is name-keyed and zero-allocation; generators use it as
// a presence test when they don't need the typed payload (e.g.
// `m.HasDirective(directive.IntegrationOnly)` is enough to gate
// dispatch emission, while `errors.Get(m)` is needed to enumerate
// the sentinels).
func (m *Method) HasDirective(name string) bool {
	_, ok := m.Attachments[name]
	return ok
}

// IsIntegrationOnly reports whether the method carries
// `//testkit:integration-only`. Generators emit no contract subtests
// for these methods — the consumer's integration test owns those
// invariants. Equivalent to `HasDirective(directive.IntegrationOnly)`
// with a typed name.
func (m *Method) IsIntegrationOnly() bool {
	return m.HasDirective(directive.IntegrationOnly)
}

// NonCtxParamCount returns the number of parameters excluding a
// leading [context.Context]. Useful to consumers whose arg-shape
// rule is "one per non-ctx param" (sample, hooks, validates).
func (m *Method) NonCtxParamCount() int {
	n := m.Signature.Params().Len()
	if m.HasContext() {
		n--
	}
	return n
}

// NonCtxParamAt returns the type of the i-th non-ctx parameter.
// Pairs with [NonCtxParamCount] for consumers iterating directive
// args alongside the corresponding parameter types.
func (m *Method) NonCtxParamAt(i int) types.Type {
	offset := 0
	if m.HasContext() {
		offset = 1
	}
	return m.Signature.Params().At(offset + i).Type()
}

// HasResults reports whether the method has any results (including
// the error result). Equivalent to NumResults > 0; provided as a
// boolean for template ergonomics.
func (m *Method) HasResults() bool { return m.NumResults() > 0 }

// HasNonErrorResults reports whether the method has at least one
// result that isn't the trailing error. Drives template branches
// that distinguish error-only methods (just "err") from value-
// returning methods (need to stamp results onto the Call struct
// after dispatch).
func (m *Method) HasNonErrorResults() bool {
	n := m.NumResults()
	return n > 1 || (n == 1 && !m.ReturnsError())
}

// HasNonContextParam reports whether the method has at least one
// parameter that isn't the leading context.
func (m *Method) HasNonContextParam() bool { return m.NonCtxParamCount() > 0 }

// ParamTypeAt returns the rendered Go type of the i-th non-ctx
// parameter (0-indexed), qualified through the import tracker.
// Generators emitting per-shape subtests use this when a shape's
// detector doesn't populate Shape.KeyType / KeyType2 / ValType
// (e.g. MultiArgWriter, where each param's role is positional only).
// Empty when i is out of range.
func (m *Method) ParamTypeAt(i int, t *generator.ImportTracker) string {
	if i < 0 || i >= m.NonCtxParamCount() {
		return ""
	}
	return generator.TypeStr(m.NonCtxParamAt(i), t)
}

// ResultTypeAt returns the rendered Go type of the i-th result
// (0-indexed, raw position — the trailing error is at the last
// index). Used by per-directive subtests that need the impl's
// return type at template time (e.g. pagination's Page struct).
// Empty when i is out of range.
func (m *Method) ResultTypeAt(i int, t *generator.ImportTracker) string {
	results := m.Signature.Results()
	if i < 0 || i >= results.Len() {
		return ""
	}
	return generator.TypeStr(results.At(i).Type(), t)
}

// SampleParamAt returns the rendered sample literal for the i-th
// non-ctx parameter (0-indexed). Generators emitting per-shape
// subtests use this to populate the K/V/P slots a shape's assertion
// primitives consume. The seed name for string samples is the
// source param name when available (so a `key string` parameter
// samples as "test-key", not "test-p0"). Empty when i is out of
// range.
func (m *Method) SampleParamAt(i int, t *generator.ImportTracker) string {
	if i < 0 || i >= m.NonCtxParamCount() {
		return ""
	}
	offset := 0
	if m.HasContext() {
		offset = 1
	}
	p := m.Signature.Params().At(offset + i)
	name := p.Name()
	if name == "" {
		name = generator.ParamName(i)
	}
	return generator.SampleValueOf(p.Type(), name, t)
}

// ZeroParamAt returns the rendered zero literal for the i-th non-ctx
// parameter. Used as the "unknown" key for RejectInvalid-style
// assertions where the test calls with a value the impl is not
// expected to recognize. Empty when i is out of range.
func (m *Method) ZeroParamAt(i int, t *generator.ImportTracker) string {
	if i < 0 || i >= m.NonCtxParamCount() {
		return ""
	}
	return generator.ZeroValueOf(m.NonCtxParamAt(i), t)
}

// SampleResultAt returns the rendered sample literal for the i-th
// result (0-indexed, raw position — the trailing error result is at
// the last index). Used by per-shape subtests to populate the
// expected-value slot of Returns-style assertions. Empty when i is
// out of range.
//
// The fieldName seed encodes the result index ("Result0", "Result1",
// …) so multi-result signatures (MultiReader, MultiAggregator,
// Lookup) sample distinct values across slots. Without the index,
// two same-typed results would collide on the same default literal
// and the contract couldn't catch a tuple-swap bug. Named results
// take precedence over the default seed.
func (m *Method) SampleResultAt(i int, t *generator.ImportTracker) string {
	results := m.Signature.Results()
	if i < 0 || i >= results.Len() {
		return ""
	}
	name := results.At(i).Name()
	if name == "" {
		name = fmt.Sprintf("Result%d", i)
	}
	return generator.SampleValueOf(results.At(i).Type(), name, t)
}

// ZeroResultAt returns the rendered zero literal for the i-th
// result. Empty when i is out of range.
func (m *Method) ZeroResultAt(i int, t *generator.ImportTracker) string {
	results := m.Signature.Results()
	if i < 0 || i >= results.Len() {
		return ""
	}
	return generator.ZeroValueOf(results.At(i).Type(), t)
}

// HasTrailingBool reports whether the method's final result is a
// builtin bool that follows at least one non-bool result with no
// error in the list — the ReaderWithBool `(V, bool)` and Lookup
// `(V0, V1, bool)` signature pattern where the bool encodes
// presence (true → found, false → miss). Generators emit a
// FaultMiss helper for these methods so consumers configure the
// "miss" outcome ergonomically without re-spelling the zero values
// for every other result.
//
// A single-bool result (Predicate, e.g. `IsHealthy() bool`) does
// NOT qualify — the bool there is the value itself, not a presence
// flag, and FaultMiss would conflate a "fault" with the legitimate
// `false` outcome. Predicate consumers wanting the false outcome
// use `Returns(false)` directly.
func (m *Method) HasTrailingBool() bool {
	if m.ReturnsError() {
		return false
	}
	results := m.Signature.Results()
	n := results.Len()
	if n < 2 {
		return false
	}
	b, ok := results.At(n - 1).Type().(*types.Basic)
	return ok && b.Kind() == types.Bool
}

// MissResultsTuple renders the comma-joined "miss" return list for
// a [HasTrailingBool] method: zero values for every non-bool result
// followed by `false` for the trailing bool. Empty for methods that
// don't qualify. Used by templates that document FaultMiss's
// observable outcome.
//
//	(V, bool)         → "<zero V>, false"
//	(V, M, bool)      → "<zero V>, <zero M>, false"
func (m *Method) MissResultsTuple(t *generator.ImportTracker) string {
	if !m.HasTrailingBool() {
		return ""
	}
	results := m.Signature.Results()
	n := results.Len()
	parts := make([]string, n)
	for i := range n - 1 {
		parts[i] = generator.ZeroValueOf(results.At(i).Type(), t)
	}
	parts[n-1] = "false"
	return strings.Join(parts, ", ")
}

// ParamFieldAssign renders the per-method Call struct's parameter
// field initialiser — "Field: var, Field: var" — used inside the
// `call := XCall{ ... }` literal at dispatch time. Only param
// fields are stamped here; results stamp later after the OnX/
// fallback path runs.
func (m *Method) ParamFieldAssign(t *generator.ImportTracker) string {
	fields := m.CallStructFields(t)
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		if f.IsResult {
			continue
		}
		parts = append(parts, f.Name+": "+f.VarName)
	}
	return strings.Join(parts, ", ")
}

// CallResultStampVars renders the post-dispatch result-var-to-Call-
// field assignment block — "call.Result = v0\n\tcall.Err = err".
// Empty when the method has no results.
//
// Templates emit this after the OnX.fn invocation so the recorded
// Call carries every result the consumer set. Non-comparable
// results (function values, channels) are skipped — the Call
// struct doesn't carry those fields, so stamping them would be a
// reference to a missing field.
func (m *Method) CallResultStampVars(t *generator.ImportTracker) string {
	fields := m.CallStructFields(t)
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		if !f.IsResult || f.NotComparable {
			continue
		}
		parts = append(parts, "call."+f.Name+" = "+f.VarName)
	}
	return strings.Join(parts, "\n\t")
}

// CallResultStampFallback is the fallback-path counterpart of
// [CallResultStampVars] — sources from the per-method return
// record's fields ("call.Result = f.Result\n\tcall.Err = f.Err").
// The template binds `f` to the dereferenced fallback pointer.
// Non-comparable results are skipped for the same reason as
// [CallResultStampVars] — they're not in the Call struct.
func (m *Method) CallResultStampFallback(t *generator.ImportTracker) string {
	fields := m.CallStructFields(t)
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		if !f.IsResult || f.NotComparable {
			continue
		}
		parts = append(parts, "call."+f.Name+" = f."+f.Name)
	}
	return strings.Join(parts, "\n\t")
}

// ErrFieldName returns the Call struct field that carries the
// trailing error ("Err"), or "" when the method doesn't return an
// error. Templates use this to stamp injected fault errors into
// the recorded call.
func (m *Method) ErrFieldName() string {
	if !m.ReturnsError() {
		return ""
	}
	return "Err"
}

// ReturnParams renders the parameter list for a [Returns]-style
// configurator: one (varName typeStr) per result field.
//
//	(Item, error) → "result Item, err error"
//	() error      → "err error"
//	()            → ""
//
// Used by stub's per-method Returns() helper and by any future
// generator that emits a "configure expected return values" method.
func (m *Method) ReturnParams(t *generator.ImportTracker) string {
	fields := m.CallStructFields(t)
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		if !f.IsResult {
			continue
		}
		parts = append(parts, f.VarName+" "+f.TypeStr)
	}
	return strings.Join(parts, ", ")
}

// ReturnFieldAssign renders the struct-literal field initialiser
// matching [ReturnParams]:
//
//	"Result: result, Err: err"
//
// Used inside the Returns() body's fallback-record literal.
func (m *Method) ReturnFieldAssign(t *generator.ImportTracker) string {
	fields := m.CallStructFields(t)
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		if !f.IsResult {
			continue
		}
		parts = append(parts, f.Name+": "+f.VarName)
	}
	return strings.Join(parts, ", ")
}

// ResultsFromFallback renders the comma-separated `f.<Field>`
// accessor list — used by the fallback path's `return f.Result,
// f.Err` statement. Empty when the method has no results.
func (m *Method) ResultsFromFallback(t *generator.ImportTracker) string {
	fields := m.CallStructFields(t)
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		if !f.IsResult {
			continue
		}
		parts = append(parts, "f."+f.Name)
	}
	return strings.Join(parts, ", ")
}

// CallField is one field on a per-method Call recording struct.
// Generators emitting test doubles (stub today, suite/bench/model in
// the future) all need the same per-method struct shape — params
// first, then results — and the same field-naming convention.
type CallField struct {
	// Name is the title-cased Go field name ("Key", "Result", "Err").
	Name string

	// VarName is the receiver-side variable name in dispatch
	// ("key", "v0", "err"). Used when populating the Call struct
	// after dispatch.
	VarName string

	// TypeStr is the rendered Go type, qualified through the
	// package's import tracker.
	TypeStr string

	// IsResult is true for non-error result fields (drives stamp-
	// after-dispatch rendering vs stamp-before for params).
	IsResult bool

	// IsError is true for the trailing error result.
	IsError bool

	// NotComparable is true when the field's type can't pass through
	// [cmp.Diff] without panicking — function and channel types.
	// Tests reading this field cannot use [testkit.Equal] against
	// it; consumers must inspect the value's structure (e.g. iterate
	// an iter.Seq) rather than equality-asserting it. Generators
	// emit a doc note on the field so manual readers don't trip on
	// the limitation.
	NotComparable bool
}

// CallStructFields builds the per-method Call struct's field list:
// params first (variadic last param recorded as a slice), then
// non-error results, then Err if the method returns an error.
//
// The naming convention for non-error results when there is more
// than one: "Result0", "Result1", ... — single non-error result
// names "Result". Mirrors the legacy stub layout so consumers'
// LastCall(t).Result/Err lookups stay stable.
//
// Non-comparable result types (function values, channels — common
// for iter.Seq) are flagged via [CallField.NotComparable] but
// retained in the slice. The Call struct template omits them from
// the recording struct (recording a function pointer is dead
// weight); the return-record template keeps them so the fallback
// dispatch path has somewhere to store the value before returning
// it. Splitting at consumer time keeps the analysis layer
// agnostic to per-template policy.
func (m *Method) CallStructFields(t *generator.ImportTracker) []CallField {
	out := make([]CallField, 0, m.NumParams()+m.NumResults())

	params := m.Signature.Params()
	pn := params.Len()
	for i := range pn {
		p := params.At(i)
		name := p.Name()
		if name == "" {
			name = generator.ParamName(i)
		}
		typeStr := types.TypeString(p.Type(), t.Qualifier())
		if m.Signature.Variadic() && i == pn-1 {
			if s, ok := p.Type().(*types.Slice); ok {
				typeStr = "[]" + types.TypeString(s.Elem(), t.Qualifier())
			}
		}
		out = append(out, CallField{
			Name:          generator.Title(name),
			VarName:       name,
			TypeStr:       typeStr,
			NotComparable: !isAssertable(p.Type()),
		})
	}

	results := m.Signature.Results()
	rn := results.Len()
	hasErr := m.ReturnsError()
	nonErrCount := rn
	if hasErr {
		nonErrCount--
	}
	nonErr := 0
	for i := range rn {
		r := results.At(i)
		typeStr := types.TypeString(r.Type(), t.Qualifier())
		if hasErr && i == rn-1 {
			out = append(out, CallField{
				Name:     "Err",
				VarName:  errVarName,
				TypeStr:  typeStr,
				IsResult: true,
				IsError:  true,
			})
			continue
		}
		out = append(out, CallField{
			Name:          resultFieldName(nonErrCount, nonErr),
			VarName:       "v" + strconv.Itoa(nonErr),
			TypeStr:       typeStr,
			IsResult:      true,
			NotComparable: !isAssertable(r.Type()),
		})
		nonErr++
	}

	return out
}

// resultFieldName produces the Call struct's non-error result
// field name. Single non-error result → "Result"; multiple →
// "Result0", "Result1", ...
func resultFieldName(nonErrCount, idx int) string {
	if nonErrCount == 1 {
		return "Result"
	}
	return "Result" + strconv.Itoa(idx)
}

// ResultNames returns the comma-separated receiver-side variable
// list templates use to pre-declare result vars before dispatch:
//
//	results=(string)         → "v0"
//	results=(string, error)  → "v0, err"
//	results=(error)          → "err"
//	results=()               → ""
//
// The matching declarations come from [ResultDecls].
func (m *Method) ResultNames() string {
	results := m.Signature.Results()
	n := results.Len()
	if n == 0 {
		return ""
	}
	hasErr := m.ReturnsError()
	parts := make([]string, n)
	nonErr := 0
	for i := range n {
		if hasErr && i == n-1 {
			parts[i] = errVarName
			continue
		}
		parts[i] = "v" + strconv.Itoa(nonErr)
		nonErr++
	}
	return strings.Join(parts, ", ")
}

// ResultDecls returns the var-declaration lines pairing the names
// returned by [ResultNames] with their types. Each line is one
// declaration; templates emit these immediately before the dispatch
// block so the post-dispatch path can stamp into them.
//
//	void           → ""
//	(string)       → "var v0 string"
//	(string, error)→ "var v0 string\n\tvar err error"
func (m *Method) ResultDecls(t *generator.ImportTracker) string {
	results := m.Signature.Results()
	n := results.Len()
	if n == 0 {
		return ""
	}
	hasErr := m.ReturnsError()
	parts := make([]string, 0, n)
	nonErr := 0
	for i := range n {
		typeStr := types.TypeString(results.At(i).Type(), t.Qualifier())
		if hasErr && i == n-1 {
			parts = append(parts, "var err error")
			continue
		}
		parts = append(parts, "var v"+strconv.Itoa(nonErr)+" "+typeStr)
		nonErr++
	}
	return strings.Join(parts, "\n\t")
}

// FaultReturn renders a fault-helper return-value list: non-error
// results stay zero, the trailing error slot carries [sentinelExpr].
// Methods without an error return fall back to the zero-results
// list (the helper's behavioral effect is then a no-op replacing
// the call with zeros).
//
//	(Item, error) + sentinel "ErrNotFound"
//	  → "Item{}, ErrNotFound"
//	error + sentinel "ErrNotFound"
//	  → "ErrNotFound"
func (m *Method) FaultReturn(t *generator.ImportTracker, sentinelExpr string) string {
	results := m.Signature.Results()
	n := results.Len()
	if n == 0 {
		return ""
	}
	if !m.ReturnsError() {
		return m.ZeroResults(t)
	}
	parts := make([]string, n)
	for i := range n - 1 {
		parts[i] = generator.ZeroValueOf(results.At(i).Type(), t)
	}
	parts[n-1] = sentinelExpr
	return strings.Join(parts, ", ")
}

// SampleResults renders the comma-joined non-zero literal list for
// the method's results, suitable as the args to a per-method
// Returns(...) helper. Non-error results sample via
// [generator.SampleValueOf]; the trailing error slot is nil.
//
//	(Item, error)             → "basic.Item{ID: \"test-id\"}, nil"
//	(string, int, error)      → "\"test-result\", 42, nil"
//	(error)                   → "nil"
//	()                        → ""
//
// Used by auto-tests to drive Returns(...) with a recognizable
// non-zero set, then assert each captured result equals that same
// literal — both via [SampleResultAsserts].
func (m *Method) SampleResults(t *generator.ImportTracker) string {
	results := m.Signature.Results()
	n := results.Len()
	if n == 0 {
		return ""
	}
	hasErr := m.ReturnsError()
	parts := make([]string, n)
	nonErr := 0
	for i := range n {
		if hasErr && i == n-1 {
			parts[i] = "nil"
			continue
		}
		parts[i] = generator.SampleValueOf(results.At(i).Type(),
			"Result"+strconv.Itoa(nonErr), t)
		nonErr++
	}
	return strings.Join(parts, ", ")
}

// SampleResultAsserts renders one [testkit.Equal] assertion per
// non-error result, comparing the captured result variable to its
// SAMPLE literal (not zero — the assertion paired with Returns(
// SampleResults(...))). Empty for methods with no non-error
// results — Returns(nil) on an error-only method has nothing to
// observe beyond NoError.
func (m *Method) SampleResultAsserts(t *generator.ImportTracker, label string) string {
	results := m.Signature.Results()
	n := results.Len()
	if n == 0 {
		return ""
	}
	hasErr := m.ReturnsError()
	var parts []string
	nonErr := 0
	for i := range n {
		if hasErr && i == n-1 {
			continue
		}
		rt := results.At(i).Type()
		if !isAssertable(rt) {
			nonErr++
			continue
		}
		sample := generator.SampleValueOf(rt,
			"Result"+strconv.Itoa(nonErr), t)
		parts = append(parts,
			fmt.Sprintf(`testkit.Equal(t, v%d, %s, %q)`,
				nonErr, sample,
				fmt.Sprintf("%s: v%d must match the configured Returns value", label, nonErr)))
		nonErr++
	}
	return strings.Join(parts, "\n\t\t")
}

// ErrCaptureLHS renders the LHS of an assignment that captures
// only the trailing error result, using blank placeholders for the
// non-error results. Empty for methods that don't return an error
// (callers should test [ReturnsError] first).
//
//	(error)               → "err"
//	(Item, error)         → "_, err"
//	(string, int, error)  → "_, _, err"
//
// Used by subtests asserting on injected fault errors that need
// `err` but don't care about the other returns.
func (m *Method) ErrCaptureLHS() string {
	if !m.ReturnsError() {
		return ""
	}
	n := m.NumResults()
	parts := make([]string, n)
	for i := range n - 1 {
		parts[i] = "_"
	}
	parts[n-1] = errVarName
	return strings.Join(parts, ", ")
}

// BlankResults renders a comma-joined list of "_" placeholders
// matching the method's result arity, suitable for the LHS of a
// blank-discard assignment when the caller wants to invoke the
// method but not inspect any of its returns. Empty for void
// methods — callers should branch on [HasResults] before placing
// this on an assignment.
//
//	(Item, error) → "_, _"
//	(error)       → "_"
//	()            → ""
func (m *Method) BlankResults() string {
	n := m.NumResults()
	if n == 0 {
		return ""
	}
	parts := make([]string, n)
	for i := range n {
		parts[i] = "_"
	}
	return strings.Join(parts, ", ")
}

// isAssertable reports whether typ can be passed to [testkit.Equal]
// without panicking inside go-cmp. Function and channel types are
// excluded — cmp.Diff panics on those. Used by the result-assert
// renderers to silently drop non-comparable result fields.
func isAssertable(typ types.Type) bool {
	switch typ.Underlying().(type) {
	case *types.Signature, *types.Chan:
		return false
	}
	return true
}

// HasAssertableNonErrorResults reports whether the method has at
// least one non-error result whose type is comparable via go-cmp.
// Drives template branches that capture result vars only when
// there's something the assertion templates will compare against
// — methods returning only `iter.Seq2[T, error]` (function value)
// have HasNonErrorResults true but no comparable result, so the
// capture would otherwise produce an unused-var compile error.
func (m *Method) HasAssertableNonErrorResults() bool {
	results := m.Signature.Results()
	n := results.Len()
	if n == 0 {
		return false
	}
	hasErr := m.ReturnsError()
	for i := range n {
		if hasErr && i == n-1 {
			continue
		}
		if isAssertable(results.At(i).Type()) {
			return true
		}
	}
	return false
}

// ZeroResultAsserts renders one [testkit.Equal] assertion per
// non-error result, comparing the captured result variable to its
// zero literal. The result variable names match the receiver-side
// names produced by [ResultNames] (`v0`, `v1`, ...). Returns the
// empty string when the method has no non-error results — callers
// should test that explicitly via [HasNonErrorResults] or guard the
// rendered string for emptiness.
//
// Function- and channel-typed results are skipped silently — both
// trip cmp.Diff's panic-on-uncomparable path. The result var still
// gets a `v<N>` slot in [ResultNames] / [ResultDecls]; only the
// assertion is omitted.
//
//	(Item, error) + "Get default"
//	  → `testkit.Equal(t, v0, basic.Item{}, "Get default: v0 must be zero")`
//	(string, int, error) + "Pair default"
//	  → `testkit.Equal(t, v0, "", "Pair default: v0 must be zero")` +
//	    "\n\t\t" +
//	    `testkit.Equal(t, v1, 0, "Pair default: v1 must be zero")`
//
// Lines join with "\n\t\t" so the rendered block sits inside a
// subtest body (function inside t.Run) at two-tab indent.
func (m *Method) ZeroResultAsserts(t *generator.ImportTracker, label string) string {
	results := m.Signature.Results()
	n := results.Len()
	if n == 0 {
		return ""
	}
	hasErr := m.ReturnsError()
	var parts []string
	nonErr := 0
	for i := range n {
		if hasErr && i == n-1 {
			continue
		}
		rt := results.At(i).Type()
		if !isAssertable(rt) {
			nonErr++
			continue
		}
		zero := generator.ZeroValueOf(rt, t)
		parts = append(parts,
			fmt.Sprintf(`testkit.Equal(t, v%d, %s, %q)`,
				nonErr, zero,
				fmt.Sprintf("%s: v%d must be zero", label, nonErr)))
		nonErr++
	}
	return strings.Join(parts, "\n\t\t")
}

// ResultFieldAsserts renders one [testkit.Equal] assertion per
// non-error result, comparing the captured Call struct's per-result
// field to the same sample literal [SampleResults] returned from
// the configured Returns. The recv argument is the receiver-side
// call variable name.
//
// Function- and channel-typed results are skipped (cmp.Diff panics).
// Empty for methods with no non-error results — the post-dispatch
// stamping has nothing to observe beyond Err.
func (m *Method) ResultFieldAsserts(t *generator.ImportTracker, label, recv string) string {
	results := m.Signature.Results()
	n := results.Len()
	if n == 0 {
		return ""
	}
	hasErr := m.ReturnsError()
	nonErrCount := n
	if hasErr {
		nonErrCount--
	}
	var parts []string
	nonErr := 0
	for i := range n {
		if hasErr && i == n-1 {
			continue
		}
		rt := results.At(i).Type()
		if !isAssertable(rt) {
			nonErr++
			continue
		}
		field := resultFieldName(nonErrCount, nonErr)
		sample := generator.SampleValueOf(rt,
			"Result"+strconv.Itoa(nonErr), t)
		parts = append(parts,
			fmt.Sprintf(`testkit.Equal(t, %s.%s, %s, %q)`,
				recv, field, sample,
				fmt.Sprintf("%s: %s must be stamped on the Call", label, field)))
		nonErr++
	}
	return strings.Join(parts, "\n\t\t")
}

// SampleArgs is the sample-value counterpart to [ZeroArgs] — a
// leading [context.Context] resolves to `t.Context()`, every other
// parameter resolves to its [generator.SampleValueOf] sample
// literal. The variadic last param is dropped — `f()` is a valid
// call.
//
//	(ctx context.Context, key string)            → "t.Context(), \"test-key\""
//	(ctx context.Context, item Item)             → "t.Context(), basic.Item{ID: \"test-id\"}"
//	(ctx context.Context, ids ...string)         → "t.Context()"
//
// Used by auto-tests verifying parameter recording: the values
// land in the Call struct's per-param fields.
func (m *Method) SampleArgs(t *generator.ImportTracker) string {
	return strings.Join(generator.SampleParamExprs(m.Signature, t), ", ")
}

// ParamFieldAsserts renders one [testkit.Equal] assertion per
// non-ctx, non-variadic parameter, comparing the captured Call
// struct's field to the same sample literal [SampleArgs] passed
// to invocation. The first argument is the receiver-side call
// variable name (typically "call" for `call := s.OnX.LastCall(t)`).
//
// Function- and channel-typed params are skipped (cmp.Diff panics).
// Empty when the method has no non-ctx params.
//
//	(ctx, key string) + label="Get records args" + recv="call"
//	  → `testkit.Equal(t, call.Key, "test-key", "Get records args: Key must be recorded")`
//	(ctx, item Item, n int) + label="Insert records args"
//	  → two assertions joined by "\n\t\t".
func (m *Method) ParamFieldAsserts(t *generator.ImportTracker, label, recv string) string {
	params := m.Signature.Params()
	pn := params.Len()
	if m.Signature.Variadic() && pn > 0 {
		pn--
	}
	var parts []string
	for i := range pn {
		p := params.At(i)
		if i == 0 && generator.IsContextType(p.Type()) {
			continue
		}
		if !isAssertable(p.Type()) {
			continue
		}
		name := p.Name()
		if name == "" {
			name = generator.ParamName(i)
		}
		field := generator.Title(name)
		sample := generator.SampleValueOf(p.Type(), name, t)
		parts = append(parts,
			fmt.Sprintf(`testkit.Equal(t, %s.%s, %s, %q)`,
				recv, field, sample,
				fmt.Sprintf("%s: %s must be recorded", label, field)))
	}
	return strings.Join(parts, "\n\t\t")
}

// HasAssertableNonCtxParams reports whether the method has at
// least one non-ctx, non-variadic parameter whose type is
// comparable via go-cmp. Drives the records-call-args subtest's
// emit gate — methods with no observable params skip the subtest
// entirely.
func (m *Method) HasAssertableNonCtxParams() bool {
	params := m.Signature.Params()
	pn := params.Len()
	if m.Signature.Variadic() && pn > 0 {
		pn--
	}
	for i := range pn {
		p := params.At(i)
		if i == 0 && generator.IsContextType(p.Type()) {
			continue
		}
		if isAssertable(p.Type()) {
			return true
		}
	}
	return false
}

// ZeroArgs renders the comma-joined argument list for invoking the
// method on a stub from inside a `t *testing.T` test body. A leading
// [context.Context] parameter resolves to `t.Context()`; every
// remaining parameter resolves to its zero literal via
// [generator.ZeroValueOf]. The variadic last param is dropped — `f()`
// is a valid zero call for variadic Go.
//
//	(ctx context.Context, key string)            → "t.Context(), \"\""
//	(ctx context.Context, item Item)             → "t.Context(), basic.Item{}"
//	(ctx context.Context, ids ...string)         → "t.Context()"
//	()                                            → ""
//
// Used by generated auto-tests to invoke each method with the
// smallest valid call — paired with [ZeroResults] for the receive
// side, the rendered subtest body never has to know the method's
// exact shape.
func (m *Method) ZeroArgs(t *generator.ImportTracker) string {
	return strings.Join(generator.ZeroParamExprs(m.Signature, t), ", ")
}

// IterReturn returns the iter.Seq / iter.Seq2 detection result for
// the method's return position. Equivalent to calling
// [generator.AnalyzeIterReturn] on the first result type when the
// method has exactly one return; otherwise the zero
// [generator.IterSeqInfo].
//
// Pairs with [generator.IterSeqInfo.IsSeq] / Seq2Error /
// Seq2Generic to drive iter-aware template branches in stub /
// suite / bench.
func (m *Method) IterReturn(t *generator.ImportTracker) generator.IterSeqInfo {
	results := m.Signature.Results()
	if results.Len() != 1 {
		return generator.IterSeqInfo{}
	}
	return generator.AnalyzeIterReturn(results.At(0).Type(), t)
}

// SubstituteTypeParams returns s with every type-parameter name
// replaced by its concrete instantiation, derived from
// [Data.Interface.TypeParams] via [generator.ConcreteFor]. Empty
// substitution (non-generic interface) returns s unchanged.
//
// Used by generators emitting auto-tests against concrete-
// instantiated generic stubs. Wraps [generator.SubstituteTypeParams]
// with the spec-side type-param list so templates call this method
// without threading the slice through.
func (d *Data) SubstituteTypeParams(s string) string {
	return generator.SubstituteTypeParams(s, d.Interface.TypeParams)
}

// QualifiedTypeForTest returns the source-package-qualified
// interface reference suffixed with the test-time concrete type-args.
// Non-generic interfaces return [QualifiedType] unchanged. Generics
// strip the type-parameter-name suffix and append the constraint-
// driven concrete instantiation from [generator.TestTypeArgs] —
// every test-emitting generator (stub auto-test, suite contract
// driver, bench harness, model runner) needs the same form for
// compile-time guards and concrete instantiation references.
//
//	non-generic:        "basic.Store"
//	generic Cache[V]:   "generics.Cache[int]"
//	KeyMap[K, V]:       "generics.KeyMap[string, int]"
func (d *Data) QualifiedTypeForTest() string {
	if !d.IsGeneric {
		return d.QualifiedType
	}
	base := strings.TrimSuffix(d.QualifiedType, d.TypeParamArgs)
	return base + generator.TestTypeArgs(d.Interface.TypeParams)
}
