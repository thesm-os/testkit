// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen

import (
	"go/types"

	"go.thesmos.sh/testkit/gen/directives"
)

// MethodShape classifies an interface method by its signature pattern.
// The generator uses the shape to emit type-safe On<Method> options
// accepting only primitives matching the detected shape.
//
// Observer alignment: plug-in primitives for each shape receive
// context.Context in their closures if and only if the method itself
// takes context.Context. Pure and Predicate methods are ctx-free by
// definition (rules 2-3 require !hasCtx), so their observers
// (PureContext, PredicateContext) are also ctx-free. This is
// intentional, not an omission.
type MethodShape int

// Method shape constants.
const (
	ShapeUnknown        MethodShape = iota
	ShapeReader                     // func(ctx, K) (V, error)
	ShapeReaderWithBool             // func(ctx, K) (V, bool) or func(K) (V, bool)
	ShapeLookup                     // func(K) (R1, R2, bool) or func(ctx, K) (R1, R2, bool)
	ShapeWriter                     // func(ctx, V) error or func(ctx, V) (R, error)
	ShapeMutator                    // func(ctx, V) — no return; requires //testkit:mutator directive
	ShapeDeleter                    // func(ctx, K) error — requires //testkit:deleter directive
	ShapeAggregator                 // func(ctx) (T, error)
	ShapeStreamReader               // returns iter.Seq[V] or iter.Seq2[V, error]
	ShapeLifecycle                  // func(ctx) error
	ShapePure                       // no error return, no ctx
	ShapePredicate                  // returns bool only, no ctx
	ShapePoisonAccessor             // func() error — no ctx, no params, returns error only
)

// ShapeInfo holds the detected shape of a method plus the extracted
// type parameters for that shape.
type ShapeInfo struct {
	Shape    MethodShape
	KeyType  string      // qualified type for K (Reader/Deleter)
	ValType  string      // qualified type for V
	RetType  string      // qualified type for R (Writer with result)
	IterInfo IterSeqInfo // for StreamReader
}

// DetectShape classifies a method by its signature pattern.
//
// Detection rules (first match wins):
//  1. Returns iter.Seq[T] or iter.Seq2[T, error] → StreamReader
//  2. Returns bool only → Predicate
//  3. No error return → Pure
//  4. ctx + one non-ctx param + (V, error) return where V is not error → Reader
//  5. ctx + one non-ctx param + error-only return → Writer (default; //testkit:deleter overrides)
//  6. ctx + one non-ctx param + (R, error) return → Writer (with result)
//  7. ctx only + (T, error) return → Aggregator
//  8. ctx only + error return → Lifecycle
//  9. Otherwise → Unknown
func DetectShape(m MethodInfo, tracker *ImportTracker, dirs []Directive) ShapeInfo {
	sig := m.Signature
	params := sig.Params()
	results := sig.Results()

	// Count context and non-context params (excluding variadic).
	paramCount := params.Len()
	if sig.Variadic() {
		paramCount--
	}
	ctxIdx := -1
	var nonCtxParams []*types.Var
	for i := range paramCount {
		p := params.At(i)
		if IsContextType(p.Type()) {
			ctxIdx = i
		} else {
			nonCtxParams = append(nonCtxParams, p)
		}
	}
	hasCtx := ctxIdx >= 0

	// Analyze returns.
	resCount := results.Len()
	hasError := false
	errIdx := -1
	for i := range resCount {
		if IsErrorType(results.At(i).Type()) {
			hasError = true
			errIdx = i
		}
	}

	// Rule 1: iter.Seq or iter.Seq2 return → StreamReader.
	for i := range resCount {
		info := AnalyzeIterReturn(results.At(i).Type(), tracker)
		if info.IsSeq || info.IsSeq2 {
			return ShapeInfo{
				Shape:    ShapeStreamReader,
				IterInfo: info,
			}
		}
	}

	// Rule 2: returns bool only, no ctx → Predicate.
	if !hasCtx && resCount == 1 {
		if b, ok := results.At(0).Type().Underlying().(*types.Basic); ok && b.Kind() == types.Bool {
			return ShapeInfo{Shape: ShapePredicate}
		}
	}

	// Rule 2.5: ReaderWithBool — func(K) (V, bool) or func(ctx, K) (V, bool).
	// Must fire BEFORE Pure (rule 3) which would swallow (V, bool) as no-error.
	if len(nonCtxParams) == 1 && resCount == 2 && !hasError {
		if b, ok := results.At(1).Type().Underlying().(*types.Basic); ok && b.Kind() == types.Bool {
			p := nonCtxParams[0]
			return ShapeInfo{
				Shape:   ShapeReaderWithBool,
				KeyType: TypeStr(p.Type(), tracker),
				ValType: TypeStr(results.At(0).Type(), tracker),
			}
		}
	}

	// Rule 2.6: Lookup — func(K) (R1, R2, bool) or func(ctx, K) (R1, R2, bool).
	// Must fire BEFORE Pure (rule 3) which would swallow (R1, R2, bool) as no-error.
	if len(nonCtxParams) == 1 && resCount == 3 && !hasError {
		if b, ok := results.At(2).Type().Underlying().(*types.Basic); ok && b.Kind() == types.Bool {
			p := nonCtxParams[0]
			return ShapeInfo{
				Shape:   ShapeLookup,
				KeyType: TypeStr(p.Type(), tracker),
				ValType: TypeStr(results.At(0).Type(), tracker),
				RetType: TypeStr(results.At(1).Type(), tracker),
			}
		}
	}

	// Rule 3: no error return, no ctx, with return value → Pure.
	// Void methods (no return) fall through to Unknown.
	if !hasCtx && !hasError && resCount > 0 {
		return ShapeInfo{
			Shape:   ShapePure,
			ValType: TypeStr(results.At(0).Type(), tracker),
		}
	}

	// From here, method returns error.
	nonErrResults := resCount
	if hasError {
		nonErrResults--
	}

	// Rule 3.5: no ctx, no params, error-only return → PoisonAccessor.
	if !hasCtx && len(nonCtxParams) == 0 && hasError && nonErrResults == 0 {
		return ShapeInfo{Shape: ShapePoisonAccessor}
	}

	// Check for //testkit:deleter and //testkit:mutator directives.
	isDeleter := false
	isMutator := false
	for _, d := range dirs {
		switch d.Name {
		case directives.Deleter:
			isDeleter = true
		case directives.Mutator:
			isMutator = true
		}
	}

	// Rules 4-6: ctx + one non-ctx param.
	if hasCtx && len(nonCtxParams) == 1 {
		p := nonCtxParams[0]
		keyType := TypeStr(p.Type(), tracker)

		// Rule 4: (V, error) with V != error → Reader.
		if nonErrResults == 1 {
			valIdx := 0
			if errIdx == 0 {
				valIdx = 1
			}
			valType := TypeStr(results.At(valIdx).Type(), tracker)
			return ShapeInfo{
				Shape:   ShapeReader,
				KeyType: keyType,
				ValType: valType,
			}
		}

		// Rule 4.5: no return at all + //testkit:mutator → Mutator.
		if resCount == 0 && isMutator {
			return ShapeInfo{
				Shape:   ShapeMutator,
				ValType: keyType,
			}
		}

		// Rule 5: error-only return → Writer (or Deleter with directive).
		if nonErrResults == 0 {
			if isDeleter {
				return ShapeInfo{
					Shape:   ShapeDeleter,
					KeyType: keyType,
				}
			}
			return ShapeInfo{
				Shape:   ShapeWriter,
				ValType: keyType,
			}
		}

		// Rule 6: (R, error) with R != error → Writer with result.
		if nonErrResults == 1 {
			retIdx := 0
			if errIdx == 0 {
				retIdx = 1
			}
			retType := TypeStr(results.At(retIdx).Type(), tracker)
			return ShapeInfo{
				Shape:   ShapeWriter,
				ValType: keyType,
				RetType: retType,
			}
		}
	}

	// Rule 6.5: ctx + one non-ctx param + no return + no directive → Unknown
	// but we also check for Mutator without ctx:
	// func(V) with no return + //testkit:mutator → Mutator.
	if !hasCtx && len(nonCtxParams) == 1 && resCount == 0 && isMutator {
		return ShapeInfo{
			Shape:   ShapeMutator,
			ValType: TypeStr(nonCtxParams[0].Type(), tracker),
		}
	}

	// Rules 7-8: ctx only (no non-ctx params).
	if hasCtx && len(nonCtxParams) == 0 {
		// Rule 7: (T, error) → Aggregator (exactly one non-error result + error).
		if hasError && nonErrResults == 1 {
			retIdx := 0
			if errIdx == 0 {
				retIdx = 1
			}
			valType := TypeStr(results.At(retIdx).Type(), tracker)
			return ShapeInfo{
				Shape:   ShapeAggregator,
				ValType: valType,
			}
		}

		// Rule 8: error only (no non-error results) → Lifecycle.
		if nonErrResults == 0 {
			return ShapeInfo{Shape: ShapeLifecycle}
		}
	}

	// Rule 9: fallback.
	return ShapeInfo{Shape: ShapeUnknown}
}

// String returns the shape name for debugging.
func (s MethodShape) String() string {
	switch s {
	case ShapeReader:
		return "Reader"
	case ShapeReaderWithBool:
		return "ReaderWithBool"
	case ShapeLookup:
		return "Lookup"
	case ShapeWriter:
		return "Writer"
	case ShapeMutator:
		return "Mutator"
	case ShapeDeleter:
		return "Deleter"
	case ShapeAggregator:
		return "Aggregator"
	case ShapeStreamReader:
		return "StreamReader"
	case ShapeLifecycle:
		return "Lifecycle"
	case ShapePure:
		return "Pure"
	case ShapePredicate:
		return "Predicate"
	case ShapePoisonAccessor:
		return "PoisonAccessor"
	default:
		return "Unknown"
	}
}
