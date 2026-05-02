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

// Note: ReturnsError(), HasContext(), ParamNames(), ParamNamesSpread(),
// IsVariadic(), NumParams(), NumResults() are promoted from the
// embedded gen.MethodInfo and available to templates directly.

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

// FuncOverrideTestExpr renders a test that sets a Func override and
// verifies it was called.
func (m *MethodData) FuncOverrideTestExpr() string {
	var b strings.Builder
	b.WriteString("called := false\n")
	b.WriteString("\ts.On")
	b.WriteString(m.Name)
	b.WriteString(".Func(")
	b.WriteString(m.FuncTypeStr())
	b.WriteString(" {\n")
	b.WriteString("\t\tcalled = true\n")
	b.WriteString("\t\treturn ")
	b.WriteString(m.ZeroReturn())
	b.WriteString("\n\t})\n")
	b.WriteString("\t")
	b.WriteString(m.IgnoredCallExpr())
	b.WriteString("\n")
	b.WriteString(`	testkit.True(t, called, "Func must be called")`)
	return b.String()
}

// ReturnsTestCallExpr renders a test that sets Returns and verifies
// the fixed values are returned.
func (m *MethodData) ReturnsTestCallExpr() string {
	var b strings.Builder
	b.WriteString("s.On")
	b.WriteString(m.Name)
	b.WriteString(".Returns(")
	b.WriteString(m.ZeroReturn())
	b.WriteString(")\n")
	b.WriteString("\t")
	b.WriteString(m.IgnoredCallExpr())
	return b.String()
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

// buildZeroParamValues returns zero-value expressions for each parameter.
func (m *MethodData) buildZeroParamValues() []string {
	params := m.Signature.Params()
	values := make([]string, params.Len())
	for i := range params.Len() {
		typ := params.At(i).Type()
		if m.Signature.Variadic() && i == params.Len()-1 {
			values[i] = "nil"
		} else {
			values[i] = gen.ZeroValueOf(typ, m.tracker)
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
