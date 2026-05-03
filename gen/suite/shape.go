// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"go/types"

	"go.thesmos.sh/testkit/gen"
	"go.thesmos.sh/testkit/gen/directive"
)

// MethodShape classifies an interface method by its signature pattern.
// The generator uses the shape to emit type-safe On<Method> options
// accepting only primitives matching the detected shape.
type MethodShape int

// Method shape constants.
const (
	ShapeUnknown      MethodShape = iota
	ShapeReader                   // func(ctx, K) (V, error)
	ShapeWriter                   // func(ctx, V) error or func(ctx, V) (R, error)
	ShapeDeleter                  // func(ctx, K) error — requires //testkit:deleter directive
	ShapeAggregator               // func(ctx) (T, error)
	ShapeStreamReader             // returns iter.Seq[V] or iter.Seq2[V, error]
	ShapeLifecycle                // func(ctx) error
	ShapePure                     // no error return
	ShapePredicate                // returns bool only
)

// ShapeInfo holds the detected shape of a method plus the extracted
// type parameters for that shape.
type ShapeInfo struct {
	Shape    MethodShape
	KeyType  string          // qualified type for K (Reader/Deleter)
	ValType  string          // qualified type for V
	RetType  string          // qualified type for R (Writer with result)
	IterInfo gen.IterSeqInfo // for StreamReader
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
func DetectShape(m gen.MethodInfo, tracker *gen.ImportTracker, directives []gen.Directive) ShapeInfo {
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
		if gen.IsContextType(p.Type()) {
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
		if gen.IsErrorType(results.At(i).Type()) {
			hasError = true
			errIdx = i
		}
	}

	// Rule 1: iter.Seq or iter.Seq2 return → StreamReader.
	for i := range resCount {
		info := gen.AnalyzeIterReturn(results.At(i).Type(), tracker)
		if info.IsSeq || info.IsSeq2 {
			return ShapeInfo{
				Shape:    ShapeStreamReader,
				IterInfo: info,
			}
		}
	}

	// Rule 2: returns bool only → Predicate.
	if resCount == 1 {
		if b, ok := results.At(0).Type().Underlying().(*types.Basic); ok && b.Kind() == types.Bool {
			return ShapeInfo{Shape: ShapePredicate}
		}
	}

	// Rule 3: no error return → Pure.
	if !hasError {
		return ShapeInfo{Shape: ShapePure}
	}

	// From here, method returns error.
	nonErrResults := resCount
	if hasError {
		nonErrResults--
	}

	// Check for //testkit:deleter directive.
	isDeleter := false
	for _, d := range directives {
		if d.Name == directive.DirDeleter {
			isDeleter = true
			break
		}
	}

	// Rules 4-6: ctx + one non-ctx param.
	if hasCtx && len(nonCtxParams) == 1 {
		p := nonCtxParams[0]
		keyType := gen.TypeStr(p.Type(), tracker)

		// Rule 4: (V, error) with V != error → Reader.
		if nonErrResults == 1 {
			valIdx := 0
			if errIdx == 0 {
				valIdx = 1
			}
			valType := gen.TypeStr(results.At(valIdx).Type(), tracker)
			return ShapeInfo{
				Shape:   ShapeReader,
				KeyType: keyType,
				ValType: valType,
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
		if nonErrResults >= 1 {
			retIdx := 0
			if errIdx == 0 {
				retIdx = 1
			}
			retType := gen.TypeStr(results.At(retIdx).Type(), tracker)
			return ShapeInfo{
				Shape:   ShapeWriter,
				ValType: keyType,
				RetType: retType,
			}
		}
	}

	// Rules 7-8: ctx only (no non-ctx params).
	if hasCtx && len(nonCtxParams) == 0 {
		// Rule 7: (T, error) → Aggregator.
		if nonErrResults >= 1 {
			retIdx := 0
			if errIdx == 0 {
				retIdx = 1
			}
			valType := gen.TypeStr(results.At(retIdx).Type(), tracker)
			return ShapeInfo{
				Shape:   ShapeAggregator,
				ValType: valType,
			}
		}

		// Rule 8: error only → Lifecycle.
		return ShapeInfo{Shape: ShapeLifecycle}
	}

	// Rule 9: fallback.
	return ShapeInfo{Shape: ShapeUnknown}
}

// String returns the shape name for debugging.
func (s MethodShape) String() string {
	switch s {
	case ShapeReader:
		return "Reader"
	case ShapeWriter:
		return "Writer"
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
	default:
		return "Unknown"
	}
}
