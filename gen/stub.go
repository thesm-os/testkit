// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"go/types"
	"path/filepath"
	"strings"
)

const (
	stubGenerator    = "stub"
	stubFileSuffix   = "in_memory_"
	stubImplTmplFile = "stub.go.tmpl"
	stubTestTmplFile = "stub_test.go.tmpl"

	errFieldName    = "Err"
	resultFieldName = "Result"
)

// GenerateStub produces a three-tier test double for each named
// interface: recording, function override, and fault injection via
// [testkit.MethodStub]. Returns a [Result] with impl + test files.
func GenerateStub(pkg *Package, typeNames []string, cfg Config, opts Options) (*Result, error) {
	errs := ValidateTypes(pkg, typeNames, KindInterface)
	if len(errs) > 0 {
		return nil, errs[0]
	}

	implPath := opts.Output
	if implPath == "" {
		implPath = defaultStubPath(typeNames[0], cfg)
	}
	testPath := testPathFrom(implPath)
	implPkgName := DerivePackageName(implPath, pkg.Pkg.Name(), cfg)

	outputImportPath, err := OutputImportPath(implPath, pkg)
	if err != nil {
		return nil, err
	}
	tracker := NewImportTracker(outputImportPath)
	srcQualifier := tracker.Add(pkg.Pkg)

	// Always need testkit and testing.
	tracker.AddPath("go.thesmos.sh/testkit")
	tracker.AddPath("testing")

	var ifaces []stubIfaceData
	for _, name := range typeNames {
		info, lookupErr := pkg.Interface(name)
		if lookupErr != nil {
			return nil, lookupErr
		}
		ifaces = append(ifaces, buildStubIfaceData(name, srcQualifier, info, tracker))
	}

	header := Header{
		Subcommand: stubGenerator,
		Args:       stubGenerator + " " + strings.Join(typeNames, " "),
	}

	implBytes, err := Render(templateFile(stubImplTmplFile), stubTemplateData{
		PackageName: implPkgName,
		Imports:     tracker.Imports(),
		Interfaces:  ifaces,
	}, header)
	if err != nil {
		return nil, WrapErr(emptyPos, err, "render stub implementation")
	}

	// Test file.
	testPkgName := implPkgName + testPkgSuffix
	testTracker := NewImportTracker("")
	testTracker.AddPath("testing")
	testTracker.AddPath("go.thesmos.sh/testkit")
	testTracker.AddPath(outputImportPath)
	testTracker.Add(pkg.Pkg)

	testSrcQualifier := testTracker.Add(pkg.Pkg)
	testGenQualifier := testTracker.AddPath(outputImportPath)

	var testIfaces []stubIfaceData
	for _, name := range typeNames {
		info, lookupErr := pkg.Interface(name)
		if lookupErr != nil {
			return nil, lookupErr
		}
		testIfaces = append(testIfaces, buildStubIfaceData(name, testSrcQualifier, info, testTracker))
	}

	testBytes, err := Render(templateFile(stubTestTmplFile), stubTestTemplateData{
		PackageName:  testPkgName,
		Imports:      testTracker.Imports(),
		Interfaces:   testIfaces,
		GenQualifier: testGenQualifier,
	}, header)
	if err != nil {
		return nil, WrapErr(emptyPos, err, "render stub tests")
	}

	return &Result{
		Files: []OutputFile{
			{Path: implPath, Content: implBytes},
			{Path: testPath, Content: testBytes},
		},
	}, nil
}

// --- data types ---

type stubMethodData struct {
	Name string // "Get"
	Doc  string // interface method doc, each line prefixed with "// "

	// Type names for generated types.
	CallType    string // "GetCall"
	StubType    string // "GetStub"
	MatcherType string // "getMatcher"
	ReturnType  string // "getReturn"
	WhenType    string // "getWhen"

	// Rendered signature parts.
	FuncTypeStr      string // "func(context.Context, string) (store.Item, error)"
	ParamListStr     string // "ctx context.Context, id string"
	ResultListStr    string // "(store.Item, error)"
	ParamNames       string // "ctx, id" — for recording/struct literals
	ParamNamesSpread string // "ctx, id..." — for forwarding variadic calls

	// Fields for call/return types.
	Params  []stubFieldData
	Results []stubFieldData

	// Error handling.
	ReturnsError bool
	ErrFieldName string // "Err" — the result field that is error

	// Rendering helpers for the template.
	ReturnParams      string // "result store.Item, err error" — for Returns()
	ReturnFieldAssign string // "Result: result, Err: err"
	FaultReturn       string // "store.Item{}, faultErr"
	ZeroReturn        string // "store.Item{}, nil"

	// Call recording helpers.
	ParamFieldAssign          string // "Ctx: ctx, ID: id, "
	ResultVarDecl             string // "r0, r1 := " or "" if no results
	ResultVarNames            string // "r0, r1" or ""
	ResultFieldAssignVars     string // "Result: r0, Err: r1"
	ResultFieldAssignFallback string // "Result: f.Result, Err: f.Err"
	ReturnFromFallback        string // "f.Result, f.Err"

	// Test template helpers.
	ZeroParamValues      string
	ErrVarName           string
	NonErrResultVars     string
	IgnoredCallExpr      string
	ZeroAssertCallExpr   string // call + assert all returns are zero
	FaultTestCallExpr    string
	ReturnsTestCallExpr  string
	FuncOverrideTestExpr string
}

type stubFieldData struct {
	FieldName string // "Ctx", "ID", "Result", "Err"
	TypeStr   string // "context.Context", "string"
	ZeroValue string // "\"\"", "0", "nil", "Item{}"
}

type stubIfaceData struct {
	TypeName      string // "Store"
	StubName      string // "StoreStub"
	QualifiedType string // "store.Store"
	Methods       []stubMethodData
}

type stubTemplateData struct {
	PackageName string
	Imports     []Import
	Interfaces  []stubIfaceData
}

type stubTestTemplateData struct {
	PackageName  string
	Imports      []Import
	Interfaces   []stubIfaceData
	GenQualifier string
}

// --- builders ---

func buildStubIfaceData(name, srcQualifier string, info *InterfaceInfo, tracker *ImportTracker) stubIfaceData {
	qualified := qualifyType(srcQualifier, name)
	methods := make([]stubMethodData, 0, len(info.Methods))
	for _, m := range info.Methods {
		methods = append(methods, buildStubMethodData(name, m, tracker))
	}
	return stubIfaceData{
		TypeName:      name,
		StubName:      name + "Stub",
		QualifiedType: qualified,
		Methods:       methods,
	}
}

func buildStubMethodData(ifaceName string, m MethodInfo, tracker *ImportTracker) stubMethodData {
	prefix := ifaceName + m.Name
	lowerPrefix := strings.ToLower(prefix[:1]) + prefix[1:]
	sig := m.Signature

	// Build param fields.
	params := buildParamFields(sig.Params(), tracker)
	results := buildResultFields(sig.Results(), tracker)

	returnsError := m.ReturnsError()
	errFieldName := ""
	if returnsError {
		errFieldName = results[len(results)-1].FieldName
	}

	// Build rendered strings.
	funcTypeStr := m.FuncType(tracker)
	paramListStr := m.ParamList(tracker)
	paramNames := m.ParamNames()
	paramNamesSpread := paramNames
	if m.IsVariadic() && sig.Params().Len() > 0 {
		paramNamesSpread = paramNames + "..."
	}
	resultListStr := m.ResultList(tracker)

	// ReturnParams for Returns() method: "result store.Item, err error"
	returnParams := buildReturnParams(results)

	// ReturnFieldAssign: "Result: result, Err: err"
	returnFieldAssign := buildReturnFieldAssign(results)

	// FaultReturn: "store.Item{}, faultErr"
	faultReturn := buildFaultReturn(results, tracker, sig.Results())

	// ZeroReturn: "store.Item{}, nil"
	zeroReturn := m.ZeroResults(tracker)

	// ParamFieldAssign: "Ctx: ctx, ID: id, "
	paramFieldAssign := buildFieldAssignFromParams(params, sig.Params())

	// Result var handling.
	resultVarDecl, resultVarNames := buildResultVarDecl(results)
	resultFieldAssignVars := buildResultFieldAssignVars(results)
	resultFieldAssignFallback := buildResultFieldAssignPrefix(results, "f")
	returnFromFallback := buildReturnFromPrefix(results, "f")

	// Test helpers.
	zeroParamValues := buildZeroParamValues(sig.Params(), tracker, m.IsVariadic())
	errVarName, nonErrResultVars := buildErrVarNames(results, returnsError)
	ignoredCallExpr := buildIgnoredCallExpr(m.Name, zeroParamValues, len(results))
	zeroAssertCallExpr := buildZeroAssertCallExpr(m.Name, zeroParamValues, results, sig.Results(), tracker)
	faultTestCallExpr := buildFaultTestCallExpr(
		m.Name, zeroParamValues, errVarName,
		nonErrResultVars, results, returnsError,
	)
	returnsTestCallExpr := buildReturnsTestExpr(
		m.Name, zeroParamValues, results, returnsError, errVarName,
	)
	funcOverrideTestExpr := buildFuncOverrideTestExpr(
		m.Name, zeroParamValues, m.FuncType(tracker), results,
		returnsError, errVarName,
	)

	return stubMethodData{
		Name:        m.Name,
		Doc:         formatDocComment(m.Doc),
		CallType:    prefix + "Call",
		StubType:    prefix + "Stub",
		MatcherType: lowerPrefix + "Matcher",
		ReturnType:  lowerPrefix + "Return",
		WhenType:    lowerPrefix + "When",

		FuncTypeStr:      funcTypeStr,
		ParamListStr:     paramListStr,
		ResultListStr:    resultListStr,
		ParamNames:       paramNames,
		ParamNamesSpread: paramNamesSpread,

		Params:  params,
		Results: results,

		ReturnsError: returnsError,
		ErrFieldName: errFieldName,

		ReturnParams:              returnParams,
		ReturnFieldAssign:         returnFieldAssign,
		FaultReturn:               faultReturn,
		ZeroReturn:                zeroReturn,
		ParamFieldAssign:          paramFieldAssign,
		ResultVarDecl:             resultVarDecl,
		ResultVarNames:            resultVarNames,
		ResultFieldAssignVars:     resultFieldAssignVars,
		ResultFieldAssignFallback: resultFieldAssignFallback,
		ReturnFromFallback:        returnFromFallback,
		ZeroParamValues:           zeroParamValues,
		ErrVarName:                errVarName,
		NonErrResultVars:          nonErrResultVars,
		IgnoredCallExpr:           ignoredCallExpr,
		ZeroAssertCallExpr:        zeroAssertCallExpr,
		FaultTestCallExpr:         faultTestCallExpr,
		ReturnsTestCallExpr:       returnsTestCallExpr,
		FuncOverrideTestExpr:      funcOverrideTestExpr,
	}
}

func buildParamFields(tuple *types.Tuple, tracker *ImportTracker) []stubFieldData {
	fields := make([]stubFieldData, 0, tuple.Len())
	for i := range tuple.Len() {
		v := tuple.At(i)
		name := v.Name()
		if name == "" {
			name = paramName(i)
		}
		fields = append(fields, stubFieldData{
			FieldName: title(name),
			TypeStr:   types.TypeString(v.Type(), tracker.Qualifier()),
		})
	}
	return fields
}

func buildResultFields(tuple *types.Tuple, tracker *ImportTracker) []stubFieldData {
	fields := make([]stubFieldData, 0, tuple.Len())
	for i := range tuple.Len() {
		v := tuple.At(i)
		name := v.Name()
		if name == "" {
			// Derive name from type: error → Err, otherwise Result/Result0/Result1/etc.
			if isErrorType(v.Type()) {
				name = errFieldName
			} else if tuple.Len() == 1 || (tuple.Len() == 2 && isErrorType(tuple.At(1).Type())) {
				// Single non-error result, or (Result, error) pair.
				name = resultFieldName
			} else {
				name = resultFieldName + string(rune('0'+i))
			}
		} else {
			name = title(name)
		}
		fields = append(fields, stubFieldData{
			FieldName: name,
			TypeStr:   types.TypeString(v.Type(), tracker.Qualifier()),
			ZeroValue: zeroValueOf(v.Type(), tracker),
		})
	}
	return fields
}

func buildReturnParams(results []stubFieldData) string {
	parts := make([]string, len(results))
	for i, r := range results {
		parts[i] = strings.ToLower(r.FieldName[:1]) + r.FieldName[1:] + " " + r.TypeStr
	}
	return strings.Join(parts, ", ")
}

func buildReturnFieldAssign(results []stubFieldData) string {
	parts := make([]string, len(results))
	for i, r := range results {
		lowerName := strings.ToLower(r.FieldName[:1]) + r.FieldName[1:]
		parts[i] = r.FieldName + ": " + lowerName
	}
	return strings.Join(parts, ", ")
}

func buildFaultReturn(results []stubFieldData, tracker *ImportTracker, tuple *types.Tuple) string {
	parts := make([]string, len(results))
	for i := range results {
		if isErrorType(tuple.At(i).Type()) {
			parts[i] = "faultErr"
		} else {
			parts[i] = zeroValueOf(tuple.At(i).Type(), tracker)
		}
	}
	return strings.Join(parts, ", ")
}

// buildFieldAssignFromParams: "Ctx: ctx, ID: id, " — trailing comma+space for composability.
func buildFieldAssignFromParams(fields []stubFieldData, tuple *types.Tuple) string {
	parts := make([]string, 0, len(fields))
	for i, f := range fields {
		paramName := tuple.At(i).Name()
		if paramName == "" {
			paramName = "p" + string(rune('0'+i))
		}
		parts = append(parts, f.FieldName+": "+paramName+", ")
	}
	return strings.Join(parts, "")
}

// buildResultVarDecl: "r0, r1 := " or "" if no results.
func buildResultVarDecl(results []stubFieldData) (decl, names string) {
	if len(results) == 0 {
		return "", ""
	}
	vars := make([]string, len(results))
	for i := range results {
		vars[i] = "r" + string(rune('0'+i))
	}
	names = strings.Join(vars, ", ")
	decl = names + " := "
	return decl, names
}

// buildResultFieldAssignVars: "Result: r0, Err: r1"
func buildResultFieldAssignVars(results []stubFieldData) string {
	parts := make([]string, len(results))
	for i, r := range results {
		parts[i] = r.FieldName + ": r" + string(rune('0'+i))
	}
	return strings.Join(parts, ", ")
}

// buildResultFieldAssignPrefix: "Result: f.Result, Err: f.Err"
func buildResultFieldAssignPrefix(results []stubFieldData, prefix string) string {
	parts := make([]string, len(results))
	for i, r := range results {
		parts[i] = r.FieldName + ": " + prefix + "." + r.FieldName
	}
	return strings.Join(parts, ", ")
}

// buildReturnFromPrefix: "f.Result, f.Err"
func buildReturnFromPrefix(results []stubFieldData, prefix string) string {
	parts := make([]string, len(results))
	for i, r := range results {
		parts[i] = prefix + "." + r.FieldName
	}
	return strings.Join(parts, ", ")
}

// buildZeroParamValues: "t.Context(), \"\"" — zero values for test calls.
// Variadic params are omitted (pass zero args).
func buildZeroParamValues(tuple *types.Tuple, tracker *ImportTracker, variadic bool) string {
	count := tuple.Len()
	if variadic && count > 0 {
		count-- // omit variadic param — pass zero args
	}
	parts := make([]string, count)
	for i := range count {
		typ := tuple.At(i).Type()
		if isContextType(typ) {
			parts[i] = "t.Context()"
		} else {
			parts[i] = zeroValueOf(typ, tracker)
		}
	}
	return strings.Join(parts, ", ")
}

// buildErrVarNames returns the error var name and non-error var names.
func buildErrVarNames(results []stubFieldData, returnsError bool) (errVar, nonErrVars string) {
	if len(results) == 0 {
		return "", ""
	}
	var nonErr []string
	for i := range results {
		varName := "r" + string(rune('0'+i))
		if returnsError && i == len(results)-1 {
			errVar = varName
		} else {
			nonErr = append(nonErr, varName)
		}
	}
	nonErrVars = strings.Join(nonErr, ", ")
	return errVar, nonErrVars
}

// buildZeroAssertCallExpr calls the method and asserts each return is zero.
func buildZeroAssertCallExpr(
	methodName, zeroParams string, results []stubFieldData,
	tuple *types.Tuple, tracker *ImportTracker,
) string {
	if len(results) == 0 {
		return "s." + methodName + "(" + zeroParams + ")"
	}
	vars := make([]string, len(results))
	for i := range results {
		vars[i] = "r" + string(rune('0'+i))
	}
	var b strings.Builder
	b.WriteString(strings.Join(vars, ", ") + " := s." + methodName + "(" + zeroParams + ")")
	for i, r := range results {
		zero := zeroValueOf(tuple.At(i).Type(), tracker)
		if r.FieldName == errFieldName {
			b.WriteString("\n\t\ttestkit.NoError(t, " + vars[i] +
				", \"default " + methodName + " must not error\")")
		} else {
			b.WriteString("\n\t\ttestkit.Equal(t, " + vars[i] + ", " + zero +
				", \"default " + methodName + " " + r.FieldName + " must be zero\")")
		}
	}
	return b.String()
}

// buildIgnoredCallExpr: "s.List(t.Context())" or "_, _ = s.Get(t.Context(), \"\")"
func buildIgnoredCallExpr(methodName, zeroParams string, numResults int) string {
	call := "s." + methodName + "(" + zeroParams + ")"
	if numResults == 0 {
		return call
	}
	blanks := make([]string, numResults)
	for i := range blanks {
		blanks[i] = "_"
	}
	return strings.Join(blanks, ", ") + " = " + call
}

// buildFaultTestCallExpr produces the full test snippet for a fault test.
func buildFaultTestCallExpr(
	methodName, zeroParams, errVarName, nonErrVars string,
	results []stubFieldData, returnsError bool,
) string {
	if !returnsError {
		return ""
	}
	call := "s." + methodName + "(" + zeroParams + ")"

	// Build assignment.
	vars := make([]string, len(results))
	for i := range results {
		vars[i] = "r" + string(rune('0'+i))
	}
	assign := strings.Join(vars, ", ") + " := " + call

	// Assertion on the error var.
	assertion := "testkit.ErrorIs(t, " + errVarName + ", errTest, \"fault must fire on " + methodName + "\")"

	// Ignore non-error vars.
	var ignore string
	if nonErrVars != "" {
		parts := strings.Split(nonErrVars, ", ")
		blanks := make([]string, len(parts))
		for i, p := range parts {
			blanks[i] = "_ = " + p
		}
		ignore = "\n\t\t" + strings.Join(blanks, "\n\t\t")
	}

	return assign + "\n\t\t" + assertion + ignore
}

// buildReturnsTestExpr produces the Returns() test snippet.
// For error-returning: s.OnGet.Returns(Item{}, errTest) → assert errTest.
// For non-error: s.OnList.Returns([]Item{{}}) → assert len == 1.
func buildReturnsTestExpr(
	methodName, zeroParams string, results []stubFieldData,
	returnsError bool, errVarName string,
) string {
	if len(results) == 0 {
		return "s." + methodName + "(" + zeroParams + ")"
	}

	// Build Returns() args — use zero values for non-error, errTest for error.
	retArgs := make([]string, len(results))
	for i, r := range results {
		if r.FieldName == errFieldName {
			retArgs[i] = "errTest"
		} else {
			retArgs[i] = r.ZeroValue
		}
	}

	var b strings.Builder
	b.WriteString("s.On" + methodName + ".Returns(" + strings.Join(retArgs, ", ") + ")\n\t\t")

	// Call and assign.
	vars := make([]string, len(results))
	for i := range results {
		vars[i] = "r" + string(rune('0'+i))
	}
	b.WriteString(strings.Join(vars, ", ") + " := s." + methodName + "(" + zeroParams + ")")

	// Assert.
	if returnsError {
		assertion := "\n\t\ttestkit.ErrorIs(t, " + errVarName +
			", errTest, \"Returns must set error on " + methodName + "\")"
		b.WriteString(assertion)
	}
	// Ignore non-error vars.
	for i, r := range results {
		if r.FieldName != errFieldName {
			b.WriteString("\n\t\t_ = " + vars[i])
		}
	}
	return b.String()
}

// buildFuncOverrideTestExpr produces the Func() override test snippet.
func buildFuncOverrideTestExpr(
	methodName, zeroParams, funcType string, results []stubFieldData,
	returnsError bool, errVarName string,
) string {
	var b strings.Builder

	// Build the override function that returns errTest.
	b.WriteString("s.On" + methodName + ".Func(" + funcType + " {\n\t\t\treturn ")
	retVals := make([]string, len(results))
	for i, r := range results {
		if r.FieldName == errFieldName {
			retVals[i] = "errTest"
		} else {
			retVals[i] = r.ZeroValue
		}
	}
	b.WriteString(strings.Join(retVals, ", "))
	b.WriteString("\n\t\t})\n\t\t")

	// Call and assign.
	vars := make([]string, len(results))
	for i := range results {
		vars[i] = "r" + string(rune('0'+i))
	}
	if len(vars) > 0 {
		b.WriteString(strings.Join(vars, ", ") + " := ")
	}
	b.WriteString("s." + methodName + "(" + zeroParams + ")")

	// Assert.
	if returnsError {
		assertion := "\n\t\ttestkit.ErrorIs(t, " + errVarName +
			", errTest, \"Func must override " + methodName + "\")"
		b.WriteString(assertion)
	}
	for i, r := range results {
		if r.FieldName != errFieldName {
			b.WriteString("\n\t\t_ = " + vars[i])
		}
	}
	return b.String()
}

// formatDocComment prefixes each line of doc with "// " so it can be
// pasted directly into generated Go source.
func formatDocComment(doc string) string {
	if doc == "" {
		return ""
	}
	lines := strings.Split(doc, "\n")
	for i, line := range lines {
		if line == "" {
			lines[i] = "//"
		} else {
			lines[i] = "// " + line
		}
	}
	return strings.Join(lines, "\n")
}

func defaultStubPath(typeName string, cfg Config) string {
	return filepath.Join(
		cfg.TestPackageSuffix,
		stubFileSuffix+strings.ToLower(typeName)+cfg.GeneratedSuffix,
	)
}
