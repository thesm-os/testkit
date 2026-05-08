// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

// predicateDetector matches `func() bool` — no ctx, no params,
// exactly one bool result. Pure-by-definition; fires above [Pure]
// to claim the bool case.
type predicateDetector struct{}

func (predicateDetector) Name() string  { return "Predicate" }
func (predicateDetector) Priority() int { return PriorityPredicate }

func (predicateDetector) Detect(s Signature) (Info, bool) {
	if s.HasCtx || s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 0 {
		return Info{}, false
	}
	if s.HasError || len(s.AllResults) != 1 {
		return Info{}, false
	}
	if !isBool(s.AllResults[0].Type()) {
		return Info{}, false
	}
	return Info{Shape: Predicate}, true
}
