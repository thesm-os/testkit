// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go/types"

	"go.thesmos.sh/testkit/generator"
)

// batchReaderDetector matches `func(ctx?, ...K) ([]V, error)`.
// Variadic plus a slice-typed first non-error result is
// unambiguous; fires above Reader/Writer.
type batchReaderDetector struct{}

func (batchReaderDetector) Name() string  { return "BatchReader" }
func (batchReaderDetector) Priority() int { return PriorityBatchReader }

func (batchReaderDetector) Detect(s Signature) (Info, bool) {
	if s.Variadic == nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 0 {
		return Info{}, false
	}
	if !s.HasError || len(s.NonErrResults) != 1 {
		return Info{}, false
	}
	slice, ok := s.NonErrResults[0].Type().Underlying().(*types.Slice)
	if !ok {
		return Info{}, false
	}
	variadicElem, ok := s.Variadic.Type().(*types.Slice)
	if !ok {
		return Info{}, false
	}
	return Info{
		Shape:   BatchReader,
		KeyType: generator.TypeStr(variadicElem.Elem(), s.Tracker),
		ValType: generator.TypeStr(slice.Elem(), s.Tracker),
	}, true
}
