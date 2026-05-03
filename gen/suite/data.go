// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"fmt"
	"go/types"
	"strings"

	"go.thesmos.sh/testkit/gen"
)

const errVarName = "err"

// SpecData is the top-level template data for a spec generation run.
type SpecData struct {
	PackageName   string
	Imports       []gen.Import
	InterfaceName string
	QualifiedType string
	Methods       []*SpecMethodData
}

// LowerInterfaceName returns the interface name with the first letter lowered.
func (d *SpecData) LowerInterfaceName() string {
	if d.InterfaceName == "" {
		return ""
	}
	return strings.ToLower(d.InterfaceName[:1]) + d.InterfaceName[1:]
}

// HasContent reports whether there are any methods with directives.
func (d *SpecData) HasContent() bool {
	for _, m := range d.Methods {
		if m.hasDirectives() {
			return true
		}
	}
	return false
}

// ShapeSummary returns a one-line summary of detected shapes, e.g.
// "Reader (Get), Writer (Put, Delete), Lifecycle (Ping), Unknown (Count)".
func (d *SpecData) ShapeSummary() string {
	groups := make(map[gen.MethodShape][]string)
	order := []gen.MethodShape{
		gen.ShapeReader, gen.ShapeWriter, gen.ShapeDeleter,
		gen.ShapeAggregator, gen.ShapeStreamReader, gen.ShapeLifecycle,
		gen.ShapePure, gen.ShapePredicate, gen.ShapeUnknown,
	}
	for _, m := range d.Methods {
		if m.Skip {
			continue
		}
		groups[m.Shape.Shape] = append(groups[m.Shape.Shape], m.Name)
	}
	var parts []string
	for _, shape := range order {
		names := groups[shape]
		if len(names) == 0 {
			continue
		}
		parts = append(parts, shape.String()+" ("+strings.Join(names, ", ")+")")
	}
	return strings.Join(parts, ", ")
}

// DirectiveSummary returns a one-line summary of consumed directives, e.g.
// "errors (Get→ErrNotFound), nilsafe (Put), timeout (Ping: 5s)".
func (d *SpecData) DirectiveSummary() string {
	var parts []string
	for _, m := range d.Methods {
		if m.Skip {
			continue
		}
		for _, s := range m.Sentinels {
			parts = append(parts, fmt.Sprintf("errors (%s→%s)", m.Name, s.VarName))
		}
		if m.NilSafe {
			parts = append(parts, fmt.Sprintf("nilsafe (%s)", m.Name))
		}
		if m.Timeout != "" {
			parts = append(parts, fmt.Sprintf("timeout (%s: %s)", m.Name, m.Timeout))
		}
		if m.Pure {
			parts = append(parts, fmt.Sprintf("pure (%s)", m.Name))
		}
		if m.BoundedMin != "" {
			parts = append(parts, fmt.Sprintf("bounded (%s: %s..%s)", m.Name, m.BoundedMin, m.BoundedMax))
		}
		if m.Deprecated != "" {
			parts = append(parts, fmt.Sprintf("deprecated (%s→%s)", m.Name, m.Deprecated))
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

// PluginPoints returns the list of plug-in option function names.
func (d *SpecData) PluginPoints() string {
	var points []string
	for _, m := range d.Methods {
		if m.Skip {
			continue
		}
		points = append(points, d.InterfaceName+"On"+m.Name)
	}
	points = append(points, d.InterfaceName+"OnAll", d.InterfaceName+"Custom")
	return strings.Join(points, ", ")
}

// DefaultSubtestCount returns the number of auto-generated subtests.
func (d *SpecData) DefaultSubtestCount() int {
	count := 0
	for _, m := range d.Methods {
		if m.Skip {
			continue
		}
		count++ // smoke — always
		if m.HasContext() && m.ReturnsError() {
			count += 3 // ctx cancellation, ctx deadline, nil context
		}
		count += len(m.nilableParams()) // nil-input per param
		if m.Iter.IsSeq {
			count += 3 // empty iteration, break, double
		}
		if m.Iter.Seq2Error {
			count += 3 // iterate no error, break, double
		}
		count += len(m.Sentinels)
		if m.NilSafe {
			count++
		}
		if m.Timeout != "" {
			count++
		}
		if m.Pure {
			count++
		}
		if m.BoundedMin != "" {
			count++
		}
		if m.Deprecated != "" {
			count++
		}
		count += len(m.Validates)
	}
	return count
}

// BenchPluginPoints returns the list of bench plug-in option function names.
func (d *SpecData) BenchPluginPoints() string {
	var points []string
	for _, m := range d.Methods {
		if m.Skip {
			continue
		}
		points = append(points, d.InterfaceName+"BenchOn"+m.Name)
	}
	points = append(points, d.InterfaceName+"BenchCustom")
	return strings.Join(points, ", ")
}

// ActiveMethodCount returns the number of non-skipped methods.
func (d *SpecData) ActiveMethodCount() int {
	count := 0
	for _, m := range d.Methods {
		if !m.Skip {
			count++
		}
	}
	return count
}

// SpecMethodData holds one method with directive-enriched fields.
type SpecMethodData struct {
	gen.MethodInfo

	// Base fields populated by Analyze.
	InterfaceName string // "Store" — for naming generated functions
	QualifiedType string // "store.Store" — for factory type
	tracker       *gen.ImportTracker

	// Directive-driven fields — zero-value means not applicable.
	Sentinels  []SentinelInfo // errors: per-sentinel ErrorIs subtests
	NilSafe    bool           // nilsafe: AssertNilSafe subtest
	Ctx        bool           // ctx: AssertCtxCancellation subtest (requires error return)
	Timeout    string         // timeout: AssertTimeout subtest (duration string, e.g. "5s")
	Pure       bool           // pure: AssertPure subtest
	Validates  []string       // validates: per-field validation subtests
	BoundedMin string         // bounded: min value (parsed as return-type literal)
	BoundedMax string         // bounded: max value
	Deprecated string         // deprecated: skip with warning
	Skip       bool           // integration-only: skip entirely

	// Auto-detected from signature — no directive needed.
	Iter  gen.IterSeqInfo // iter.Seq[T] or iter.Seq2[K, V] return info
	Shape gen.ShapeInfo   // method shape (Reader/Writer/Stream/etc.)
}

// hasDirectives reports whether this method has any spec-relevant directives.
func (m *SpecMethodData) hasDirectives() bool {
	return len(m.Sentinels) > 0 ||
		m.NilSafe || m.Ctx || m.Timeout != "" || m.Pure ||
		len(m.Validates) > 0 || m.BoundedMin != "" ||
		m.Deprecated != ""
}

// SentinelInfo describes one sentinel error for ErrorIs subtests.
type SentinelInfo struct {
	VarName   string // "ErrNotFound"
	ShortName string // "NotFound"
	Qualified string // "store.ErrNotFound"
}

// LowerInterfaceName returns the interface name with first letter lowered.
func (m *SpecMethodData) LowerInterfaceName() string {
	if m.InterfaceName == "" {
		return ""
	}
	return strings.ToLower(m.InterfaceName[:1]) + m.InterfaceName[1:]
}

// IsReader reports whether the method has Reader shape.
func (m *SpecMethodData) IsReader() bool { return m.Shape.Shape == gen.ShapeReader }

// IsWriter reports whether the method has Writer shape.
func (m *SpecMethodData) IsWriter() bool { return m.Shape.Shape == gen.ShapeWriter }

// IsStreamReader reports whether the method has StreamReader shape.
func (m *SpecMethodData) IsStreamReader() bool { return m.Shape.Shape == gen.ShapeStreamReader }

// IsLifecycle reports whether the method has Lifecycle shape.
func (m *SpecMethodData) IsLifecycle() bool { return m.Shape.Shape == gen.ShapeLifecycle }

// IsPure reports whether the method has Pure shape.
func (m *SpecMethodData) IsPure() bool { return m.Shape.Shape == gen.ShapePure }

// IsAggregator reports whether the method has Aggregator shape.
func (m *SpecMethodData) IsAggregator() bool { return m.Shape.Shape == gen.ShapeAggregator }

// IsDeleter reports whether the method has Deleter shape.
func (m *SpecMethodData) IsDeleter() bool { return m.Shape.Shape == gen.ShapeDeleter }

// IsPredicate reports whether the method has Predicate shape.
func (m *SpecMethodData) IsPredicate() bool { return m.Shape.Shape == gen.ShapePredicate }

// OnMethodAssertionType renders the Go type expression for the On<Method>
// assertion parameter. For Reader: testkit.ReaderAssertion[T, K, V].
// For Unknown/untyped: func(*testing.T, T).
func (m *SpecMethodData) OnMethodAssertionType() string {
	switch m.Shape.Shape {
	case gen.ShapeReader:
		return "testkit.ReaderAssertion[" + m.QualifiedType + ", " + m.Shape.KeyType + ", " + m.Shape.ValType + "]"
	case gen.ShapeWriter:
		return "testkit.WriterAssertion[" + m.QualifiedType + ", " + m.Shape.ValType + "]"
	case gen.ShapeDeleter:
		return "testkit.DeleterAssertion[" + m.QualifiedType + ", " + m.Shape.KeyType + "]"
	case gen.ShapeStreamReader:
		return "testkit.StreamAssertion[" + m.QualifiedType + ", " + m.Shape.IterInfo.ElemType + "]"
	case gen.ShapeAggregator:
		return "testkit.AggregatorAssertion[" + m.QualifiedType + ", " + m.Shape.ValType + "]"
	case gen.ShapeLifecycle:
		return "testkit.LifecycleAssertion[" + m.QualifiedType + "]"
	case gen.ShapePure:
		return "testkit.PureAssertion[" + m.QualifiedType + ", " + m.Shape.ValType + "]"
	case gen.ShapePredicate:
		return "testkit.PredicateAssertion[" + m.QualifiedType + "]"
	default:
		return "func(*testing.T, " + m.QualifiedType + ")"
	}
}

// OnMethodBenchType renders the Go type expression for the BenchOn<Method>
// assertion parameter. For Reader: testkit.BenchReaderAssertion[T, K, V].
// For unsupported shapes: func(*testing.B, T).
func (m *SpecMethodData) OnMethodBenchType() string {
	switch m.Shape.Shape {
	case gen.ShapeReader:
		return "testkit.BenchReader[" + m.QualifiedType + ", " + m.Shape.KeyType + ", " + m.Shape.ValType + "]"
	default:
		return "func(*testing.B, " + m.QualifiedType + ")"
	}
}

// ZeroCallArgs renders zero-value arguments for calling this method.
// Context params use t.Context(), others use gen.ZeroValueOf.
func (m *SpecMethodData) ZeroCallArgs() string {
	return gen.ZeroCallArgs(m.Signature, m.tracker)
}

// PureCallArgs renders zero-value arguments for Pure-shaped methods.
// Pure methods have no context, so all params get zero values.
func (m *SpecMethodData) PureCallArgs() string {
	params := m.Signature.Params()
	n := params.Len()
	if m.Signature.Variadic() {
		n--
	}
	parts := make([]string, n)
	for i := range n {
		parts[i] = gen.ZeroValueOf(params.At(i).Type(), m.tracker)
	}
	return strings.Join(parts, ", ")
}

// StreamCallArgs renders arguments for calling this method inside a stream
// dispatch closure where context.Context is available as "ctx".
func (m *SpecMethodData) StreamCallArgs() string {
	params := m.Signature.Params()
	n := params.Len()
	if m.Signature.Variadic() {
		n--
	}
	parts := make([]string, n)
	for i := range n {
		if gen.IsContextType(params.At(i).Type()) {
			parts[i] = "ctx"
		} else {
			parts[i] = gen.ZeroValueOf(params.At(i).Type(), m.tracker)
		}
	}
	return strings.Join(parts, ", ")
}

// --- Template helper methods ---

// IterCallExpr renders a call on recv that returns the iterator result.
func (m *SpecMethodData) IterCallExpr(recv string) string {
	return recv + "." + m.Name + "(" + gen.ZeroCallArgs(m.Signature, m.tracker) + ")"
}

// NilCtxCallExpr renders a call with nil context, capturing the error result.
// Used by the nil-context auto-detected test.
func (m *SpecMethodData) NilCtxCallExpr() string {
	var b strings.Builder
	results := m.Signature.Results()
	names := make([]string, results.Len())
	for i := range results.Len() {
		if gen.IsErrorType(results.At(i).Type()) {
			names[i] = errVarName
		} else {
			names[i] = "_"
		}
	}
	if results.Len() > 0 {
		b.WriteString(strings.Join(names, ", "))
		b.WriteString(" := ")
	}
	b.WriteString("factory().")
	b.WriteString(m.Name)
	b.WriteString("(")
	// Build args — context param gets nil, others get zero.
	params := m.Signature.Params()
	n := params.Len()
	if m.Signature.Variadic() {
		n--
	}
	for i := range n {
		if i > 0 {
			b.WriteString(", ")
		}
		if gen.IsContextType(params.At(i).Type()) {
			b.WriteString("nil") //nolint:govet // intentional nil context for testing
		} else {
			b.WriteString(gen.ZeroValueOf(params.At(i).Type(), m.tracker))
		}
	}
	b.WriteString(")\n")
	b.WriteString("\t\t\treturn err")
	return b.String()
}

// FreshCallAssertNoError renders a call on a fresh factory instance and
// asserts no error. Used by the setup-then-read auto-detected test.
func (m *SpecMethodData) FreshCallAssertNoError() string {
	var b strings.Builder
	results := m.Signature.Results()
	names := make([]string, results.Len())
	errName := ""
	for i := range results.Len() {
		if gen.IsErrorType(results.At(i).Type()) {
			names[i] = errVarName
			errName = errVarName
		} else {
			names[i] = "_"
		}
	}
	b.WriteString(strings.Join(names, ", "))
	b.WriteString(" := s.")
	b.WriteString(m.Name)
	b.WriteString("(")
	b.WriteString(gen.ZeroCallArgs(m.Signature, m.tracker))
	b.WriteString(")\n")
	if errName != "" {
		b.WriteString("\t\ttestkit.NoError(t, ")
		b.WriteString(errName)
		b.WriteString(`, "`)
		b.WriteString(m.Name)
		b.WriteString(` must not error after setup")`)
	}
	return b.String()
}

// HasNilableParams reports whether the method has any pointer, slice, map,
// or interface-typed parameters (excluding context.Context).
func (m *SpecMethodData) HasNilableParams() bool {
	return len(m.nilableParams()) > 0
}

// NilableParams returns info about each nilable parameter for nil-input tests.
func (m *SpecMethodData) NilableParams() []NilableParam {
	return m.nilableParams()
}

// NilableParam describes a parameter that can be nil.
type NilableParam struct {
	Index     int
	FieldName string
	TypeStr   string
}

func (m *SpecMethodData) nilableParams() []NilableParam {
	params := m.Signature.Params()
	n := params.Len()
	if m.Signature.Variadic() {
		n--
	}
	var result []NilableParam
	for i := range n {
		p := params.At(i)
		if gen.IsContextType(p.Type()) {
			continue
		}
		switch p.Type().Underlying().(type) {
		case *types.Pointer, *types.Slice, *types.Map, *types.Interface:
			name := p.Name()
			if name == "" {
				name = fmt.Sprintf("param%d", i)
			}
			result = append(result, NilableParam{
				Index:     i,
				FieldName: name,
				TypeStr:   gen.TypeStr(p.Type(), m.tracker),
			})
		}
	}
	return result
}

// NilParamCallExpr renders a call where the specified param index is nil
// and all others are zero-valued. Returns the full call as a string.
func (m *SpecMethodData) NilParamCallExpr(paramIdx int) string {
	var b strings.Builder
	results := m.Signature.Results()
	if results.Len() > 0 {
		blanks := make([]string, results.Len())
		for i := range blanks {
			blanks[i] = "_"
		}
		b.WriteString(strings.Join(blanks, ", "))
		b.WriteString(" = ")
	}
	b.WriteString("factory().")
	b.WriteString(m.Name)
	b.WriteString("(")
	params := m.Signature.Params()
	n := params.Len()
	if m.Signature.Variadic() {
		n--
	}
	for i := range n {
		if i > 0 {
			b.WriteString(", ")
		}
		if i == paramIdx {
			b.WriteString("nil")
		} else if gen.IsContextType(params.At(i).Type()) {
			b.WriteString("t.Context()")
		} else {
			b.WriteString(gen.ZeroValueOf(params.At(i).Type(), m.tracker))
		}
	}
	b.WriteString(")")
	return b.String()
}

// NeedsUnknownID reports whether the errors subtest for this method
// uses cfg.unknownID() — true when the first non-context param is string.
func (m *SpecMethodData) NeedsUnknownID() bool {
	params := m.Signature.Params()
	n := params.Len()
	if m.Signature.Variadic() {
		n--
	}
	for i := range n {
		if gen.IsContextType(params.At(i).Type()) {
			continue
		}
		b, ok := params.At(i).Type().Underlying().(*types.Basic)
		return ok && b.Kind() == types.String
	}
	return false
}

// ErrCallExpr renders a call that captures the error result, using
// cfg.unknownID() for the first string-typed non-context param.
// If no string param exists, uses zero values for all params.
// Non-error results use blank identifiers.
//
//	_, err := s.Get(t.Context(), cfg.unknownID())
func (m *SpecMethodData) ErrCallExpr(inputExpr string) string {
	// Only substitute unknownID if the first non-context param is string.
	params := m.Signature.Params()
	n := params.Len()
	if m.Signature.Variadic() {
		n--
	}
	hasStringParam := false
	for i := range n {
		if gen.IsContextType(params.At(i).Type()) {
			continue
		}
		if b, ok := params.At(i).Type().Underlying().(*types.Basic); ok && b.Kind() == types.String {
			hasStringParam = true
		}
		break // only check the first non-context param
	}
	if hasStringParam {
		return m.buildCallExpr("s", inputExpr, true)
	}
	return m.buildCallExpr("s", "", true)
}

// BenchIgnoredCallExpr renders a call on recv that discards all results,
// using b.Context() for context parameters (bench-compatible).
func (m *SpecMethodData) BenchIgnoredCallExpr(recv string) string {
	var b strings.Builder
	results := m.Signature.Results()
	if results.Len() > 0 {
		blanks := make([]string, results.Len())
		for i := range blanks {
			blanks[i] = "_"
		}
		b.WriteString(strings.Join(blanks, ", "))
		b.WriteString(" = ")
	}
	b.WriteString(recv)
	b.WriteString(".")
	b.WriteString(m.Name)
	b.WriteString("(")
	b.WriteString(gen.ZeroCallArgsWithCtx(m.Signature, m.tracker, "b.Context()"))
	b.WriteString(")")
	return b.String()
}

// IgnoredCallExpr renders a call on recv that discards all results.
func (m *SpecMethodData) IgnoredCallExpr(recv string) string {
	var b strings.Builder
	results := m.Signature.Results()
	if results.Len() > 0 {
		blanks := make([]string, results.Len())
		for i := range blanks {
			blanks[i] = "_"
		}
		b.WriteString(strings.Join(blanks, ", "))
		b.WriteString(" = ")
	}
	b.WriteString(recv)
	b.WriteString(".")
	b.WriteString(m.Name)
	b.WriteString("(")
	b.WriteString(gen.ZeroCallArgs(m.Signature, m.tracker))
	b.WriteString(")")
	return b.String()
}

// CallExpr renders a call on recv that returns all results.
//
//	s.Get(t.Context(), "")
func (m *SpecMethodData) CallExpr(recv string) string {
	return recv + "." + m.Name + "(" + gen.ZeroCallArgs(m.Signature, m.tracker) + ")"
}

// CtxCallExpr renders a call that passes the ctx parameter from the
// closure and captures the error result. Used by ctx and timeout subtests.
//
//	_, err := factory().Get(ctx, "")
//	return err
func (m *SpecMethodData) CtxCallExpr() string {
	var b strings.Builder
	results := m.Signature.Results()

	// Build result capture.
	names := make([]string, results.Len())
	errIdx := -1
	for i := range results.Len() {
		if gen.IsErrorType(results.At(i).Type()) {
			names[i] = errVarName
			errIdx = i
		} else {
			names[i] = "_"
		}
	}
	if errIdx >= 0 {
		b.WriteString(strings.Join(names, ", "))
		b.WriteString(" := ")
	}

	b.WriteString("factory().")
	b.WriteString(m.Name)
	b.WriteString("(")
	// Build args — first context param uses "ctx", others zero.
	params := m.Signature.Params()
	n := params.Len()
	if m.Signature.Variadic() {
		n--
	}
	for i := range n {
		if i > 0 {
			b.WriteString(", ")
		}
		if gen.IsContextType(params.At(i).Type()) {
			b.WriteString("ctx")
		} else {
			b.WriteString(gen.ZeroValueOf(params.At(i).Type(), m.tracker))
		}
	}
	b.WriteString(")\n")
	if errIdx >= 0 {
		b.WriteString("\t\t\treturn err")
	}
	return b.String()
}

// ResultTypeStr renders the result type tuple for use as a return type.
// For single result: "Item". For multi: "(Item, error)".
func (m *SpecMethodData) ResultTypeStr() string {
	return m.ResultList(m.tracker)
}

// PureObserveType renders the return type for the pure observer function.
// For single non-error return, this is the type itself. For multi-return
// with error, only the non-error types are included.
func (m *SpecMethodData) PureObserveType() string {
	// Return the first non-error result type. AssertPure[S] requires a
	// single type parameter — multi-value returns use only the primary result.
	for r := range m.Signature.Results().Variables() {
		if !gen.IsErrorType(r.Type()) {
			return gen.TypeStr(r.Type(), m.tracker)
		}
	}
	return "any"
}

// PureObserveCallExpr renders the observer call body — calls the method,
// discards error, returns non-error values.
func (m *SpecMethodData) PureObserveCallExpr(recv string) string {
	results := m.Signature.Results()
	var b strings.Builder

	// Capture only the first non-error result; discard the rest.
	names := make([]string, results.Len())
	resultName := ""
	for i := range results.Len() {
		if gen.IsErrorType(results.At(i).Type()) {
			names[i] = "_"
		} else if resultName == "" {
			names[i] = "v"
			resultName = "v"
		} else {
			names[i] = "_"
		}
	}

	if len(names) > 0 {
		b.WriteString(strings.Join(names, ", "))
		b.WriteString(" := ")
	}
	b.WriteString(recv)
	b.WriteString(".")
	b.WriteString(m.Name)
	b.WriteString("(")
	b.WriteString(gen.ZeroCallArgs(m.Signature, m.tracker))
	b.WriteString(")\n")
	b.WriteString("\t\t\t\treturn ")
	b.WriteString(resultName)
	return b.String()
}

// TimeoutDuration renders the timeout directive argument as a Go duration
// expression. Parses common patterns: "5s" → "5 * time.Second",
// "100ms" → "100 * time.Millisecond", "2m" → "2 * time.Minute".
// Falls back to raw string for complex durations.
func (m *SpecMethodData) TimeoutDuration() string {
	s := m.Timeout
	// Handle simple single-unit durations.
	for _, unit := range []struct {
		suffix string
		goUnit string
	}{
		{"ms", "time.Millisecond"},
		{"s", "time.Second"},
		{"m", "time.Minute"},
		{"h", "time.Hour"},
	} {
		if num, ok := strings.CutSuffix(s, unit.suffix); ok {
			if num != "" && num[0] >= '0' && num[0] <= '9' {
				return num + " * " + unit.goUnit
			}
		}
	}
	// Fallback: emit as a parsed duration.
	return "func() time.Duration { d, _ := time.ParseDuration(\"" + s + "\"); return d }()"
}

// BoundedResultType renders the type of the first non-error result.
func (m *SpecMethodData) BoundedResultType() string {
	for r := range m.Signature.Results().Variables() {
		if !gen.IsErrorType(r.Type()) {
			return gen.TypeStr(r.Type(), m.tracker)
		}
	}
	return "int" // fallback
}

// BoundedCallExpr renders a call on recv that returns the first non-error result.
func (m *SpecMethodData) BoundedCallExpr(recv string) string {
	results := m.Signature.Results()
	if results.Len() == 1 {
		return recv + "." + m.Name + "(" + gen.ZeroCallArgs(m.Signature, m.tracker) + ")"
	}
	// Multi-return — capture and return first non-error.
	var b strings.Builder
	names := make([]string, results.Len())
	resultName := ""
	for i := range results.Len() {
		if gen.IsErrorType(results.At(i).Type()) {
			names[i] = "_"
		} else if resultName == "" {
			names[i] = "v"
			resultName = "v"
		} else {
			names[i] = "_"
		}
	}
	b.WriteString(strings.Join(names, ", "))
	b.WriteString(" := ")
	b.WriteString(recv)
	b.WriteString(".")
	b.WriteString(m.Name)
	b.WriteString("(")
	b.WriteString(gen.ZeroCallArgs(m.Signature, m.tracker))
	b.WriteString(")\n\t\t\treturn ")
	b.WriteString(resultName)
	return b.String()
}

// ValidatesCallExpr renders a validates test body — creates a sample,
// zeros the named field, calls the method, asserts error.
func (*SpecMethodData) ValidatesCallExpr(field string) string {
	// This is a placeholder — validates needs field resolution similar
	// to the stub's partition enricher. For now emit a comment.
	return "// TODO: validates " + field + " — needs field resolution"
}

// buildCallExpr builds a call expression with optional input substitution.
func (m *SpecMethodData) buildCallExpr(recv, inputExpr string, captureErr bool) string {
	var b strings.Builder
	results := m.Signature.Results()

	if captureErr && results.Len() > 0 {
		names := make([]string, results.Len())
		for i := range results.Len() {
			if gen.IsErrorType(results.At(i).Type()) {
				names[i] = errVarName
			} else {
				names[i] = "_"
			}
		}
		b.WriteString(strings.Join(names, ", "))
		b.WriteString(" := ")
	}

	b.WriteString(recv)
	b.WriteString(".")
	b.WriteString(m.Name)
	b.WriteString("(")

	params := m.Signature.Params()
	n := params.Len()
	if m.Signature.Variadic() {
		n--
	}
	usedInput := false
	for i := range n {
		if i > 0 {
			b.WriteString(", ")
		}
		if gen.IsContextType(params.At(i).Type()) {
			b.WriteString("t.Context()")
		} else if !usedInput && inputExpr != "" {
			b.WriteString(inputExpr)
			usedInput = true
		} else {
			b.WriteString(gen.ZeroValueOf(params.At(i).Type(), m.tracker))
		}
	}
	b.WriteString(")")
	return b.String()
}
