// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

// readerWithBoolDetector matches `func(ctx?, K) (V, bool)` —
// two results with bool last, no error. The Go idiom for
// "value, ok" lookups.
type readerWithBoolDetector struct{}

func (readerWithBoolDetector) Name() string  { return "ReaderWithBool" }
func (readerWithBoolDetector) Priority() int { return PriorityReaderWithBool }

func (readerWithBoolDetector) Detect(s Signature) (Info, bool) {
	if s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 1 {
		return Info{}, false
	}
	if s.HasError || len(s.AllResults) != 2 {
		return Info{}, false
	}
	if !isBool(s.AllResults[1].Type()) {
		return Info{}, false
	}
	return Info{
		Shape:   ReaderWithBool,
		KeyType: s.keyType(),
		ValType: s.valType(),
	}, true
}
