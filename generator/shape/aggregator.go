// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

// aggregatorDetector matches `func(ctx?) (T, error)` and the
// no-error form `func(ctx?) T`. Zero non-ctx parameters, exactly
// one non-error result. Models snapshot, count, summary, head
// readouts (ANALYSIS.md G28-extended).
//
// Requires ctx OR error so the no-ctx no-error case (`func() T`)
// stays with [Pure] above. Predicate (priority 820) and
// PoisonAccessor (priority 830) claim their special cases first.
type aggregatorDetector struct{}

func (aggregatorDetector) Name() string  { return "Aggregator" }
func (aggregatorDetector) Priority() int { return PriorityAggregator }

func (aggregatorDetector) Detect(s Signature) (Info, bool) {
	if s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 0 {
		return Info{}, false
	}
	if len(s.NonErrResults) != 1 {
		return Info{}, false
	}
	if !s.HasCtx && !s.HasError {
		return Info{}, false
	}
	return Info{
		Shape:   Aggregator,
		ValType: s.valType(),
	}, true
}
