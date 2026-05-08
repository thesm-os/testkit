// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go/types"

	"go.thesmos.sh/testkit/generator"
)

// pointerReaderDetector matches `func(ctx?, K) *V` — single
// non-ctx parameter and a pointer-typed non-error result. Models
// the nil-on-miss idiom common in lookup APIs that return
// `*Item` instead of `(Item, error)` or `(Item, bool)`
// (ANALYSIS.md G28).
//
// Fires above ReaderNoError so the pointer case gets the
// dedicated AssertReturnsNilForMissing primitive instead of
// generic ReaderNoError treatment.
type pointerReaderDetector struct{}

func (pointerReaderDetector) Name() string  { return "PointerReader" }
func (pointerReaderDetector) Priority() int { return PriorityPointerReader }

func (pointerReaderDetector) Detect(s Signature) (Info, bool) {
	if s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 1 {
		return Info{}, false
	}
	if s.HasError || len(s.NonErrResults) != 1 {
		return Info{}, false
	}
	ptr, ok := s.NonErrResults[0].Type().(*types.Pointer)
	if !ok {
		return Info{}, false
	}
	return Info{
		Shape:   PointerReader,
		KeyType: s.keyType(),
		ValType: generator.TypeStr(ptr.Elem(), s.Tracker),
	}, true
}
