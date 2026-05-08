// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

// readerDetector matches `func(ctx?, K) (V, error)` — single
// non-ctx parameter, one non-error result, error return. The
// canonical "fetch by key" shape.
//
// StreamConsumer (priority 900) claims interface-typed K first;
// PointerReader (priority 450) claims `*V` results before this
// detector runs.
type readerDetector struct{}

func (readerDetector) Name() string  { return "Reader" }
func (readerDetector) Priority() int { return PriorityReader }

func (readerDetector) Detect(s Signature) (Info, bool) {
	if s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 1 {
		return Info{}, false
	}
	if !s.HasError || len(s.NonErrResults) != 1 {
		return Info{}, false
	}
	return Info{
		Shape:   Reader,
		KeyType: s.keyType(),
		ValType: s.valType(),
	}, true
}
