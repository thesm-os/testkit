// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package stub implements the stub generator for testkit. It produces
// configurable test doubles with recording, fault injection, and strict
// mode from Go interface definitions.
package stub

import (
	"strconv"
	"strings"

	"go.thesmos.sh/testkit/gen"
)

const errVarName = "err" // default variable name for error results in generated test expressions

// Data is the top-level template data for a stub generation run.
type Data struct {
	PackageName  string
	Imports      []gen.Import
	Interfaces   []InterfaceData
	GenQualifier string // package qualifier for test file (e.g. "storetest." or "")
}

// InterfaceData holds one interface being stubbed.
type InterfaceData struct {
	Name          string // "Store"
	StubName      string // "StoreStub"
	TypeName      string // "Store" (for method naming prefixes)
	QualifiedType string // "store.Store"
	Methods       []*MethodData

	sourcePkgPath string // source package import path (for qualifying sentinels)
}

// FirstContextMethod returns the first non-skipped method that has a
// context.Context parameter, or nil if none exists. Used by test templates
// for clock-aware tests gated on HasContext.
func (d *InterfaceData) FirstContextMethod() *MethodData {
	for _, m := range d.Methods {
		if !m.Skip && m.HasContext() {
			return m
		}
	}
	return nil
}

// FirstErrorMethod returns the first non-skipped method that returns error,
// or nil if none exists.
func (d *InterfaceData) FirstErrorMethod() *MethodData {
	for _, m := range d.Methods {
		if !m.Skip && m.ReturnsError() {
			return m
		}
	}
	return nil
}

// HasOrderConstraint reports whether any method has an order-after directive.
func (d *InterfaceData) HasOrderConstraint() bool {
	for _, m := range d.Methods {
		if !m.Skip && m.OrderAfter != "" {
			return true
		}
	}
	return false
}

// HasErrorMethod reports whether any method in this interface returns error.
// Used by test templates to conditionally declare errTest.
func (d *InterfaceData) HasErrorMethod() bool {
	for _, m := range d.Methods {
		if m.ReturnsError() {
			return true
		}
	}
	return false
}

// MethodData holds one method with base fields and directive-enriched
// fields. Template helper methods delegate to gen.MethodInfo with the
// stored ImportTracker.
type MethodData struct {
	gen.MethodInfo

	CallType   string // "StoreGetCall"
	StubType   string // "StoreGetStub"
	ReturnType string // "storeGetReturn" (unexported)

	Params  []gen.FieldData
	Results []gen.FieldData

	tracker *gen.ImportTracker
	iface   *InterfaceData

	// Directive-driven fields — zero-value means not applicable.
	Skip       bool           // integration-only: skip method emission
	Deprecated string         // deprecated: replacement method name
	Sentinels  []SentinelInfo // errors: fault helper methods
	RetryN     int            // retry-succeeds-on-attempt: succeed on Nth call
	Partition  *PartitionInfo // partition: per-field fault targeting
	OrderAfter string         // order-after: must be called after this method
}

// PartitionInfo describes a partition field for per-key fault targeting.
type PartitionInfo struct {
	FieldPath string // "Req.RunID" — dot-separated path from call struct root
	FieldName string // "RunID" — leaf field name, used in helper method names
	FieldType string // "string" — Go type string for the partition key parameter
}

// SentinelInfo describes one sentinel error for fault helpers.
type SentinelInfo struct {
	VarName   string // "ErrNotFound"
	ShortName string // "NotFound"
	Qualified string // "store.ErrNotFound"
}

// --- Template helper methods ---

// ParamList renders the parameter list as Go source.
//
//	"ctx context.Context, id string"
func (m *MethodData) ParamList() string {
	return m.MethodInfo.ParamList(m.tracker)
}

// ResultList renders the result type list as Go source.
//
//	"(basic.Item, error)" or "" for void
func (m *MethodData) ResultList() string {
	return m.MethodInfo.ResultList(m.tracker)
}

// FuncTypeStr renders the function type signature without name.
//
//	"func(context.Context, string) (basic.Item, error)"
//	"func(context.Context)" for void methods
func (m *MethodData) FuncTypeStr() string {
	return m.FuncType(m.tracker)
}

// CallExpr renders a forwarding call expression. Variadic methods
// spread the last parameter.
//
//	"recv.Get(ctx, id)"
func (m *MethodData) CallExpr(recv string) string {
	return m.CallForward(recv)
}

// ZeroReturn renders zero values for all result types, comma-separated.
//
//	"basic.Item{}, nil" or "nil" or "" for void
func (m *MethodData) ZeroReturn() string {
	return m.ZeroResults(m.tracker)
}

// ParamFieldAssign renders struct field assignments from parameter names.
//
//	"Ctx: ctx, ID: id, "
func (m *MethodData) ParamFieldAssign() string {
	names := m.ParamNameList()
	var b strings.Builder
	for i, p := range m.Params {
		if i < len(names) {
			b.WriteString(p.FieldName)
			b.WriteString(": ")
			b.WriteString(names[i])
			b.WriteString(", ")
		}
	}
	return b.String()
}

// ResultFieldAssignVars renders struct field assignments from result
// variable names (r0, r1, ...).
//
//	"Result: r0, Err: r1, "
func (m *MethodData) ResultFieldAssignVars() string {
	var b strings.Builder
	for i, r := range m.Results {
		b.WriteString(r.FieldName)
		b.WriteString(": r")
		b.WriteString(strconv.Itoa(i))
		b.WriteString(", ")
	}
	return b.String()
}

// ResultFieldAssignFallback renders struct field assignments from the
// fallback struct fields.
//
//	"Result: f.Result, Err: f.Err, "
func (m *MethodData) ResultFieldAssignFallback() string {
	var b strings.Builder
	for _, r := range m.Results {
		b.WriteString(r.FieldName)
		b.WriteString(": f.")
		b.WriteString(r.FieldName)
		b.WriteString(", ")
	}
	return b.String()
}

// CallResultStampVars renders statements that stamp result variable values
// onto an existing call struct.
//
//	"call.Result = r0\n\tcall.Err = r1"
func (m *MethodData) CallResultStampVars() string {
	var b strings.Builder
	for i, r := range m.Results {
		if i > 0 {
			b.WriteString("\n\t")
		}
		b.WriteString("call.")
		b.WriteString(r.FieldName)
		b.WriteString(" = r")
		b.WriteString(strconv.Itoa(i))
	}
	return b.String()
}

// CallResultStampFallback renders statements that stamp fallback values
// onto an existing call struct.
//
//	"call.Result = f.Result\n\tcall.Err = f.Err"
func (m *MethodData) CallResultStampFallback() string {
	var b strings.Builder
	for i, r := range m.Results {
		if i > 0 {
			b.WriteString("\n\t")
		}
		b.WriteString("call.")
		b.WriteString(r.FieldName)
		b.WriteString(" = f.")
		b.WriteString(r.FieldName)
	}
	return b.String()
}

// ReturnParams renders result values as function parameters for the
// Returns() method.
//
//	"result basic.Item, err error"
func (m *MethodData) ReturnParams() string {
	parts := make([]string, len(m.Results))
	for i, r := range m.Results {
		name := strings.ToLower(r.FieldName[:1]) + r.FieldName[1:]
		parts[i] = name + " " + r.TypeStr
	}
	return strings.Join(parts, ", ")
}

// ReturnFieldAssign renders struct field assignments from Returns()
// parameter names.
//
//	"Result: result, Err: err"
func (m *MethodData) ReturnFieldAssign() string {
	parts := make([]string, len(m.Results))
	for i, r := range m.Results {
		name := strings.ToLower(r.FieldName[:1]) + r.FieldName[1:]
		parts[i] = r.FieldName + ": " + name
	}
	return strings.Join(parts, ", ")
}

// ReturnFromFallback renders return values from the fallback struct.
//
//	"f.Result, f.Err"
func (m *MethodData) ReturnFromFallback() string {
	parts := make([]string, len(m.Results))
	for i, r := range m.Results {
		parts[i] = "f." + r.FieldName
	}
	return strings.Join(parts, ", ")
}

// ResultVarDecl renders the variable declaration for capturing fn results.
//
//	"r0, r1 := " for multi-return
//	"r0 := " for single return
//	"" for void
func (m *MethodData) ResultVarDecl() string {
	n := len(m.Results)
	if n == 0 {
		return ""
	}
	names := make([]string, n)
	for i := range n {
		names[i] = "r" + strconv.Itoa(i)
	}
	return strings.Join(names, ", ") + " := "
}

// ResultVarNames renders the variable names for returning fn results.
//
//	"r0, r1" or "" for void
func (m *MethodData) ResultVarNames() string {
	n := len(m.Results)
	if n == 0 {
		return ""
	}
	names := make([]string, n)
	for i := range n {
		names[i] = "r" + strconv.Itoa(i)
	}
	return strings.Join(names, ", ")
}

// FaultReturn renders return values for the fault path. All positions
// are zero except the error position which gets faultErr.
//
//	"basic.Item{}, faultErr"
func (m *MethodData) FaultReturn() string {
	parts := make([]string, len(m.Results))
	for i, r := range m.Results {
		if r.IsError {
			parts[i] = "faultErr"
		} else {
			parts[i] = r.ZeroValue
		}
	}
	return strings.Join(parts, ", ")
}

// ErrFieldName returns the name of the error field in the call type,
// or "" if the method does not return an error.
func (m *MethodData) ErrFieldName() string {
	for _, r := range m.Results {
		if r.IsError {
			return r.FieldName
		}
	}
	return ""
}

// HasResults reports whether the method has any return values.
func (m *MethodData) HasResults() bool {
	return len(m.Results) > 0
}

// Note: ReturnsError(), HasContext(), ParamNames(), ParamNamesSpread(),
// IsVariadic(), NumParams(), NumResults() are promoted from the
// embedded gen.MethodInfo and available to templates directly.

// SampleReturn renders non-zero sample values for all results, comma-separated.
// Used by test templates to set Returns with values that can be asserted.
// Error positions use errTest.
//
//	"basic.Item{ID: \"test-result\"}, errTest"
func (m *MethodData) SampleReturn() string {
	parts := make([]string, len(m.Results))
	for i, r := range m.Results {
		if r.IsError {
			parts[i] = "errTest"
		} else {
			parts[i] = gen.SampleValueOf(m.Signature.Results().At(i).Type(), r.FieldName, m.tracker)
		}
	}
	return strings.Join(parts, ", ")
}

// SampleReturnNoError renders non-zero sample values for all results,
// comma-separated. Error positions use nil. Used by "Returns fixed value"
// tests where we don't want to trigger error assertions.
func (m *MethodData) SampleReturnNoError() string {
	parts := make([]string, len(m.Results))
	for i, r := range m.Results {
		if r.IsError {
			parts[i] = "nil"
		} else {
			parts[i] = gen.SampleValueOf(m.Signature.Results().At(i).Type(), r.FieldName, m.tracker)
		}
	}
	return strings.Join(parts, ", ")
}

// --- Test expression helpers (used by test.go.tmpl) ---

// ZeroAssertCallExpr renders a call with zero-value args and assertions
// on the zero-value results.
func (m *MethodData) ZeroAssertCallExpr() string {
	var b strings.Builder
	// Build zero param values.
	paramZeros := m.buildZeroParamValues()
	if len(m.Results) > 0 {
		names := m.resultNames()
		b.WriteString(strings.Join(names, ", "))
		b.WriteString(" := s.")
		b.WriteString(m.Name)
		b.WriteString("(")
		b.WriteString(strings.Join(paramZeros, ", "))
		b.WriteString(")\n")
		for _, r := range m.Results {
			name := strings.ToLower(r.FieldName[:1]) + r.FieldName[1:]
			if r.IsError {
				b.WriteString("\ttestkit.NoError(t, ")
				b.WriteString(name)
				b.WriteString(`, "`)
				b.WriteString(m.Name)
				b.WriteString(` must not error")`)
			} else {
				b.WriteString("\ttestkit.Equal(t, ")
				b.WriteString(name)
				b.WriteString(", ")
				b.WriteString(r.ZeroValue)
				b.WriteString(`, "`)
				b.WriteString(m.Name)
				b.WriteString(` must return zero")`)
			}
			b.WriteString("\n")
		}
	} else {
		b.WriteString("s.")
		b.WriteString(m.Name)
		b.WriteString("(")
		b.WriteString(strings.Join(paramZeros, ", "))
		b.WriteString(")")
	}
	return strings.TrimSpace(b.String())
}

// FaultTestCallExpr renders a call that expects a fault error.
// Non-error results use blank identifiers to avoid unused variable errors.
func (m *MethodData) FaultTestCallExpr() string {
	var b strings.Builder
	paramZeros := m.buildZeroParamValues()

	// Build result names: _ for non-error, named for error.
	resultNames := make([]string, len(m.Results))
	errName := ""
	for i, r := range m.Results {
		if r.IsError {
			name := strings.ToLower(r.FieldName[:1]) + r.FieldName[1:]
			resultNames[i] = name
			errName = name
		} else {
			resultNames[i] = "_"
		}
	}

	b.WriteString(strings.Join(resultNames, ", "))
	b.WriteString(" := s.")
	b.WriteString(m.Name)
	b.WriteString("(")
	b.WriteString(strings.Join(paramZeros, ", "))
	b.WriteString(")\n")
	if errName != "" {
		b.WriteString("\ttestkit.ErrorIs(t, ")
		b.WriteString(errName)
		b.WriteString(`, errTest, "must return injected fault")`)
	}
	return strings.TrimSpace(b.String())
}

// SentinelFaultTestCallExpr renders a call that expects a specific sentinel error.
func (m *MethodData) SentinelFaultTestCallExpr(qualifiedSentinel string) string {
	var b strings.Builder
	paramZeros := m.buildZeroParamValues()

	resultNames := make([]string, len(m.Results))
	errName := ""
	for i, r := range m.Results {
		if r.IsError {
			name := strings.ToLower(r.FieldName[:1]) + r.FieldName[1:]
			resultNames[i] = name
			errName = name
		} else {
			resultNames[i] = "_"
		}
	}

	b.WriteString(strings.Join(resultNames, ", "))
	b.WriteString(" := s.")
	b.WriteString(m.Name)
	b.WriteString("(")
	b.WriteString(strings.Join(paramZeros, ", "))
	b.WriteString(")\n")
	if errName != "" {
		b.WriteString("\ttestkit.ErrorIs(t, ")
		b.WriteString(errName)
		b.WriteString(", ")
		b.WriteString(qualifiedSentinel)
		b.WriteString(`, "must return `)
		b.WriteString(qualifiedSentinel)
		b.WriteString(`")`)
	}
	return strings.TrimSpace(b.String())
}

// FuncOverrideTestExpr renders a test that sets a Func override,
// calls the method, and asserts the custom return values come through.
func (m *MethodData) FuncOverrideTestExpr() string {
	var b strings.Builder
	b.WriteString("called := false\n")
	b.WriteString("\ts.On")
	b.WriteString(m.Name)
	b.WriteString(".Func(")
	b.WriteString(m.FuncTypeStr())
	b.WriteString(" {\n")
	b.WriteString("\t\tcalled = true\n")
	if len(m.Results) > 0 {
		b.WriteString("\t\treturn ")
		b.WriteString(m.SampleReturnNoError())
		b.WriteString("\n")
	}
	b.WriteString("\t})\n")

	paramZeros := m.buildZeroParamValues()
	if len(m.Results) > 0 {
		names := m.resultNames()
		b.WriteString("\t")
		b.WriteString(strings.Join(names, ", "))
		b.WriteString(" := s.")
		b.WriteString(m.Name)
		b.WriteString("(")
		b.WriteString(strings.Join(paramZeros, ", "))
		b.WriteString(")\n")
		b.WriteString(`	testkit.True(t, called, "Func must be called")`)
		b.WriteString("\n")
		for i, r := range m.Results {
			name := names[i]
			if r.IsError {
				b.WriteString("\ttestkit.NoError(t, ")
				b.WriteString(name)
				b.WriteString(`, "Func must not error")`)
			} else {
				sample := gen.SampleValueOf(m.Signature.Results().At(i).Type(), r.FieldName, m.tracker)
				b.WriteString("\ttestkit.Equal(t, ")
				b.WriteString(name)
				b.WriteString(", ")
				b.WriteString(sample)
				b.WriteString(`, "Func must return set value")`)
			}
			b.WriteString("\n")
		}
	} else {
		b.WriteString("\ts.")
		b.WriteString(m.Name)
		b.WriteString("(")
		b.WriteString(strings.Join(paramZeros, ", "))
		b.WriteString(")\n")
		b.WriteString(`	testkit.True(t, called, "Func must be called")`)
	}
	return strings.TrimSpace(b.String())
}

// ReturnsTestCallExpr renders a test that sets Returns with sample
// values, calls the method, and asserts the returned values match.
func (m *MethodData) ReturnsTestCallExpr() string {
	paramZeros := m.buildZeroParamValues()
	var b strings.Builder

	if len(m.Results) == 0 {
		// Void method — no Returns(), just call.
		b.WriteString("s.")
		b.WriteString(m.Name)
		b.WriteString("(")
		b.WriteString(strings.Join(paramZeros, ", "))
		b.WriteString(")")
		return b.String()
	}

	b.WriteString("s.On")
	b.WriteString(m.Name)
	b.WriteString(".Returns(")
	b.WriteString(m.SampleReturnNoError())
	b.WriteString(")\n")

	if len(m.Results) > 0 {
		names := m.resultNames()
		b.WriteString("\t")
		b.WriteString(strings.Join(names, ", "))
		b.WriteString(" := s.")
		b.WriteString(m.Name)
		b.WriteString("(")
		b.WriteString(strings.Join(paramZeros, ", "))
		b.WriteString(")\n")
		for i, r := range m.Results {
			name := names[i]
			if r.IsError {
				b.WriteString("\ttestkit.NoError(t, ")
				b.WriteString(name)
				b.WriteString(`, "Returns must not error")`)
			} else {
				sample := gen.SampleValueOf(m.Signature.Results().At(i).Type(), r.FieldName, m.tracker)
				b.WriteString("\ttestkit.Equal(t, ")
				b.WriteString(name)
				b.WriteString(", ")
				b.WriteString(sample)
				b.WriteString(`, "Returns must return set value")`)
			}
			b.WriteString("\n")
		}
	} else {
		b.WriteString("\ts.")
		b.WriteString(m.Name)
		b.WriteString("(")
		b.WriteString(strings.Join(paramZeros, ", "))
		b.WriteString(")")
	}
	return strings.TrimSpace(b.String())
}

// IgnoredCallExpr renders a call expression that discards results.
// Used for recording tests and Reset tests.
func (m *MethodData) IgnoredCallExpr() string {
	paramZeros := m.buildZeroParamValues()
	var b strings.Builder
	if len(m.Results) > 0 {
		// Use blank identifiers for all results.
		blanks := make([]string, len(m.Results))
		for i := range blanks {
			blanks[i] = "_"
		}
		b.WriteString(strings.Join(blanks, ", "))
		b.WriteString(" = s.")
	} else {
		b.WriteString("s.")
	}
	b.WriteString(m.Name)
	b.WriteString("(")
	b.WriteString(strings.Join(paramZeros, ", "))
	b.WriteString(")")
	return b.String()
}

// RetryScheduleTestExpr renders a test that verifies the retry-succeeds-on-attempt
// pattern: first N-1 calls fail, Nth call succeeds.
func (m *MethodData) RetryScheduleTestExpr() string {
	var b strings.Builder
	paramZeros := m.buildZeroParamValues()
	callArgs := strings.Join(paramZeros, ", ")

	// Generate N calls. First N-1 assert error, last asserts success.
	for call := 1; call <= m.RetryN; call++ {
		resultNames := make([]string, len(m.Results))
		for i, r := range m.Results {
			if r.IsError {
				resultNames[i] = "err" + strconv.Itoa(call)
			} else {
				resultNames[i] = "_"
			}
		}
		b.WriteString("\t")
		b.WriteString(strings.Join(resultNames, ", "))
		b.WriteString(" := s.")
		b.WriteString(m.Name)
		b.WriteString("(")
		b.WriteString(callArgs)
		b.WriteString(")\n")
	}

	// Assert first N-1 fail, Nth succeeds.
	for call := 1; call < m.RetryN; call++ {
		b.WriteString("\ttestkit.ErrorIs(t, err")
		b.WriteString(strconv.Itoa(call))
		b.WriteString(`, errTest, "attempt `)
		b.WriteString(strconv.Itoa(call))
		b.WriteString(` must fail")`)
		b.WriteString("\n")
	}
	b.WriteString("\ttestkit.NoError(t, err")
	b.WriteString(strconv.Itoa(m.RetryN))
	b.WriteString(`, "attempt `)
	b.WriteString(strconv.Itoa(m.RetryN))
	b.WriteString(` must succeed")`)

	return strings.TrimSpace(b.String())
}

// FaultForPartitionTestExpr renders a test that verifies partition-targeted
// faults fire for the matching key and pass for others.
func (m *MethodData) FaultForPartitionTestExpr() string {
	var b strings.Builder
	paramZeros := m.buildZeroParamValues()

	// Build a sample call with the partition field set.
	sampleKey := `"target-partition"`

	b.WriteString("s.On")
	b.WriteString(m.Name)
	b.WriteString(".FaultForPartition(")
	b.WriteString(sampleKey)
	b.WriteString(", errTest, 1)\n")

	// Call with matching key — should fault.
	resultNames := make([]string, len(m.Results))
	errName := ""
	for i, r := range m.Results {
		if r.IsError {
			resultNames[i] = errVarName
			errName = errVarName
		} else {
			resultNames[i] = "_"
		}
	}
	// Build a call with the partition field set to the matching key.
	b.WriteString("\t")
	b.WriteString(strings.Join(resultNames, ", "))
	b.WriteString(" := s.")
	b.WriteString(m.Name)
	b.WriteString("(")
	// Replace the param that contains the partition field.
	for i, pz := range paramZeros {
		if i > 0 {
			b.WriteString(", ")
		}
		isPartitionParam := i < len(m.Params) && m.Partition != nil &&
			strings.HasPrefix(m.Partition.FieldPath, m.Params[i].FieldName+".")
		if isPartitionParam {
			// This is the struct param containing the partition field.
			// Emit a struct literal with the partition field set.
			b.WriteString(m.Params[i].TypeStr)
			b.WriteString("{")
			b.WriteString(m.Partition.FieldName)
			b.WriteString(": ")
			b.WriteString(sampleKey)
			b.WriteString("}")
		} else {
			b.WriteString(pz)
		}
	}
	b.WriteString(")\n")
	if errName != "" {
		b.WriteString("\ttestkit.ErrorIs(t, ")
		b.WriteString(errName)
		b.WriteString(`, errTest, "must fault for matching partition key")`)
	}

	return strings.TrimSpace(b.String())
}

// FaultForOtherPartitionsTestExpr renders a test that verifies
// FaultForOtherPartitions faults when the key does NOT match.
func (m *MethodData) FaultForOtherPartitionsTestExpr() string {
	var b strings.Builder
	paramZeros := m.buildZeroParamValues()

	protectedKey := `"protected-partition"`
	otherKey := `"other-partition"`

	b.WriteString("s.On")
	b.WriteString(m.Name)
	b.WriteString(".FaultForOtherPartitions(")
	b.WriteString(protectedKey)
	b.WriteString(", errTest, 1)\n")

	// Call with non-matching key — should fault.
	resultNames := make([]string, len(m.Results))
	errName := ""
	for i, r := range m.Results {
		if r.IsError {
			resultNames[i] = errVarName
			errName = errVarName
		} else {
			resultNames[i] = "_"
		}
	}
	b.WriteString("\t")
	b.WriteString(strings.Join(resultNames, ", "))
	b.WriteString(" := s.")
	b.WriteString(m.Name)
	b.WriteString("(")
	for i, pz := range paramZeros {
		if i > 0 {
			b.WriteString(", ")
		}
		isPartitionParam := i < len(m.Params) && m.Partition != nil &&
			strings.HasPrefix(m.Partition.FieldPath, m.Params[i].FieldName+".")
		if isPartitionParam {
			b.WriteString(m.Params[i].TypeStr)
			b.WriteString("{")
			b.WriteString(m.Partition.FieldName)
			b.WriteString(": ")
			b.WriteString(otherKey)
			b.WriteString("}")
		} else {
			b.WriteString(pz)
		}
	}
	b.WriteString(")\n")
	if errName != "" {
		b.WriteString("\ttestkit.ErrorIs(t, ")
		b.WriteString(errName)
		b.WriteString(`, errTest, "must fault for non-matching partition key")`)
	}

	return strings.TrimSpace(b.String())
}

// FaultsForExpiredCallExpr renders a call after the fault window has expired,
// asserting that the error is nil. Uses = (not :=) for the error variable,
// since it's already declared by the preceding FaultTestCallExpr. Non-error
// results use _ (same as FaultTestCallExpr).
func (m *MethodData) FaultsForExpiredCallExpr() string {
	var b strings.Builder
	paramZeros := m.buildZeroParamValues()

	// Match FaultTestCallExpr's variable pattern: _ for non-error, named for error.
	resultNames := make([]string, len(m.Results))
	errName := ""
	for i, r := range m.Results {
		if r.IsError {
			name := strings.ToLower(r.FieldName[:1]) + r.FieldName[1:]
			resultNames[i] = name
			errName = name
		} else {
			resultNames[i] = "_"
		}
	}

	if len(resultNames) > 0 {
		b.WriteString(strings.Join(resultNames, ", "))
		b.WriteString(" = s.")
	} else {
		b.WriteString("s.")
	}
	b.WriteString(m.Name)
	b.WriteString("(")
	b.WriteString(strings.Join(paramZeros, ", "))
	b.WriteString(")\n")
	if errName != "" {
		b.WriteString("\ttestkit.NoError(t, ")
		b.WriteString(errName)
		b.WriteString(`, "`)
		b.WriteString(m.Name)
		b.WriteString(` must succeed after window expires")`)
	}
	return strings.TrimSpace(b.String())
}

// FaultPriorityTestExpr renders a test that configures both Func (which panics)
// and Faults, calls the method, and asserts the fault wins. Only valid for
// methods that return error.
func (m *MethodData) FaultPriorityTestExpr() string {
	var b strings.Builder
	paramZeros := m.buildZeroParamValues()

	// Set a Func that panics — must not be called.
	b.WriteString("s.On")
	b.WriteString(m.Name)
	b.WriteString(".Func(")
	b.WriteString(m.FuncTypeStr())
	b.WriteString(" {\n")
	b.WriteString("\t\tpanic(\"Func must not be called when Faults is configured\")\n")
	b.WriteString("\t})\n")
	b.WriteString("\ts.On")
	b.WriteString(m.Name)
	b.WriteString(".Faults(errTest, 1)\n")

	// Call and assert fault error.
	resultNames := make([]string, len(m.Results))
	errName := ""
	for i, r := range m.Results {
		if r.IsError {
			name := strings.ToLower(r.FieldName[:1]) + r.FieldName[1:]
			resultNames[i] = name
			errName = name
		} else {
			resultNames[i] = "_"
		}
	}
	b.WriteString("\t")
	b.WriteString(strings.Join(resultNames, ", "))
	b.WriteString(" := s.")
	b.WriteString(m.Name)
	b.WriteString("(")
	b.WriteString(strings.Join(paramZeros, ", "))
	b.WriteString(")\n")
	if errName != "" {
		b.WriteString("\ttestkit.ErrorIs(t, ")
		b.WriteString(errName)
		b.WriteString(`, errTest, "Faults must win over Func")`)
	}
	return strings.TrimSpace(b.String())
}

// CountedFaultTestExpr renders a test that configures Faults(errTest, 3),
// calls the method 3 times, and asserts calls 1-2 succeed while call 3 fails.
// Only valid for methods that return error.
func (m *MethodData) CountedFaultTestExpr() string {
	var b strings.Builder
	paramZeros := m.buildZeroParamValues()
	callArgs := strings.Join(paramZeros, ", ")

	b.WriteString("s.On")
	b.WriteString(m.Name)
	b.WriteString(".Faults(errTest, 3)\n")

	// Three calls with numbered error variables.
	for call := 1; call <= 3; call++ {
		resultNames := make([]string, len(m.Results))
		for i, r := range m.Results {
			if r.IsError {
				resultNames[i] = "err" + strconv.Itoa(call)
			} else {
				resultNames[i] = "_"
			}
		}
		b.WriteString("\t")
		b.WriteString(strings.Join(resultNames, ", "))
		b.WriteString(" := s.")
		b.WriteString(m.Name)
		b.WriteString("(")
		b.WriteString(callArgs)
		b.WriteString(")\n")
	}

	// Assert calls 1-2 succeed, call 3 fails.
	b.WriteString("\ttestkit.NoError(t, err1, \"call 1 must succeed\")\n")
	b.WriteString("\ttestkit.NoError(t, err2, \"call 2 must succeed\")\n")
	b.WriteString("\ttestkit.ErrorIs(t, err3, errTest, \"call 3 must fault\")")

	return strings.TrimSpace(b.String())
}

// CallAndAssertSampleExpr renders a call that asserts the returned values
// match the expected non-zero sample values, WITHOUT setting Returns.
// The caller is responsible for ensuring Returns (or DelegateTo) is
// configured before this expression runs. Messages are context-neutral
// so this works for both "Reset preserves config" and "DelegateTo
// surfaces return values" tests.
func (m *MethodData) CallAndAssertSampleExpr() string {
	paramZeros := m.buildZeroParamValues()
	var b strings.Builder

	if len(m.Results) == 0 {
		return ""
	}

	names := m.resultNames()
	b.WriteString(strings.Join(names, ", "))
	b.WriteString(" := s.")
	b.WriteString(m.Name)
	b.WriteString("(")
	b.WriteString(strings.Join(paramZeros, ", "))
	b.WriteString(")\n")
	for i, r := range m.Results {
		name := names[i]
		if r.IsError {
			b.WriteString("\ttestkit.NoError(t, ")
			b.WriteString(name)
			b.WriteString(`, "`)
			b.WriteString(m.Name)
			b.WriteString(` must not error")`)
		} else {
			sample := gen.SampleValueOf(m.Signature.Results().At(i).Type(), r.FieldName, m.tracker)
			b.WriteString("\ttestkit.Equal(t, ")
			b.WriteString(name)
			b.WriteString(", ")
			b.WriteString(sample)
			b.WriteString(`, "`)
			b.WriteString(m.Name)
			b.WriteString(` must return configured value")`)
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

// SampleCallAndAssert renders a call with sample (non-zero) args,
// captures the return in LastCall, and asserts each param field matches.
// Used by TEST-3 (LastCall arg round-trip).
func (m *MethodData) SampleCallAndAssert() string {
	var b strings.Builder
	paramSamples := m.buildSampleParamValues()

	// Call with sample values, discard results.
	if len(m.Results) > 0 {
		blanks := make([]string, len(m.Results))
		for i := range blanks {
			blanks[i] = "_"
		}
		b.WriteString(strings.Join(blanks, ", "))
		b.WriteString(" = s.")
	} else {
		b.WriteString("s.")
	}
	b.WriteString(m.Name)
	b.WriteString("(")
	b.WriteString(strings.Join(paramSamples, ", "))
	b.WriteString(")\n")

	// Capture via LastCall and assert each param field.
	b.WriteString("\tcall := s.On")
	b.WriteString(m.Name)
	b.WriteString(".LastCall(t)\n")
	params := m.Signature.Params()
	n := params.Len()
	if m.Signature.Variadic() {
		n-- // skip variadic
	}
	for i := range n {
		p := m.Params[i]
		if gen.IsContextType(params.At(i).Type()) {
			// Don't assert context equality — it's not comparable via Equal.
			continue
		}
		b.WriteString("\ttestkit.Equal(t, call.")
		b.WriteString(p.FieldName)
		b.WriteString(", ")
		b.WriteString(paramSamples[i])
		b.WriteString(`, "`)
		b.WriteString(p.FieldName)
		b.WriteString(` must be recorded"`)
		b.WriteString(")\n")
	}
	return strings.TrimSpace(b.String())
}

// HasNonContextParam reports whether the method has any parameter that
// is not context.Context. Used to gate TEST-3 — methods with only context
// have no interesting args to round-trip.
func (m *MethodData) HasNonContextParam() bool {
	params := m.Signature.Params()
	n := params.Len()
	if m.Signature.Variadic() {
		n--
	}
	for i := range n {
		if !gen.IsContextType(params.At(i).Type()) {
			return true
		}
	}
	return false
}

// buildSampleParamValues returns sample (non-zero) values for each
// parameter. Context gets t.Context(), other params get SampleValueOf.
func (m *MethodData) buildSampleParamValues() []string {
	params := m.Signature.Params()
	n := params.Len()
	if m.Signature.Variadic() {
		n--
	}
	values := make([]string, n)
	for i := range n {
		if gen.IsContextType(params.At(i).Type()) {
			values[i] = "t.Context()"
		} else {
			values[i] = gen.SampleValueOf(params.At(i).Type(), m.Params[i].FieldName, m.tracker)
		}
	}
	return values
}

// buildZeroParamValues returns zero-value expressions for each parameter.
// Variadic parameters are omitted entirely — callers should invoke the
// method without passing any variadic args (e.g. s.Find(ctx) not s.Find(ctx, nil)).
func (m *MethodData) buildZeroParamValues() []string {
	params := m.Signature.Params()
	n := params.Len()
	if m.Signature.Variadic() {
		n-- // skip the variadic param
	}
	values := make([]string, n)
	for i := range n {
		if gen.IsContextType(params.At(i).Type()) {
			values[i] = "t.Context()"
		} else {
			values[i] = gen.ZeroValueOf(params.At(i).Type(), m.tracker)
		}
	}
	return values
}

// resultNames returns lowercased result variable names.
func (m *MethodData) resultNames() []string {
	names := make([]string, len(m.Results))
	for i, r := range m.Results {
		names[i] = strings.ToLower(r.FieldName[:1]) + r.FieldName[1:]
	}
	return names
}

// NewMethodDataForTest constructs a MethodData for testing.
// Not intended for production use — exported only for package tests.
func NewMethodDataForTest(
	info gen.MethodInfo,
	callType, stubType, returnType string,
	params, results []gen.FieldData,
	tracker *gen.ImportTracker,
	ifaceName, sourcePkgPath string,
) *MethodData {
	iface := &InterfaceData{
		Name:          ifaceName,
		sourcePkgPath: sourcePkgPath,
	}
	return &MethodData{
		MethodInfo: info,
		CallType:   callType,
		StubType:   stubType,
		ReturnType: returnType,
		Params:     params,
		Results:    results,
		tracker:    tracker,
		iface:      iface,
	}
}
