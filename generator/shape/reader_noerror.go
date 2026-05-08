// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

// readerNoErrorDetector matches `func(ctx?, K) V` — single
// non-ctx parameter, one non-error result, no error return.
// Models infallible lookups against in-memory state (caches,
// gauges, stable mappings) (ANALYSIS.md G28).
//
// PointerReader (priority 450) claims `*V` results first;
// ReaderWithBool (priority 840) claims `(V, bool)` first.
type readerNoErrorDetector struct{}

func (readerNoErrorDetector) Name() string  { return "ReaderNoError" }
func (readerNoErrorDetector) Priority() int { return PriorityReaderNoError }

func (readerNoErrorDetector) Detect(s Signature) (Info, bool) {
	if s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 1 {
		return Info{}, false
	}
	if s.HasError || len(s.NonErrResults) != 1 {
		return Info{}, false
	}
	return Info{
		Shape:   ReaderNoError,
		KeyType: s.keyType(),
		ValType: s.valType(),
	}, true
}
