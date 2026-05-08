// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

// pureDetector matches `func() T` — no ctx, no params, exactly
// one non-error result. Pure-by-definition: deterministic
// function of nothing.
//
// Predicate (bool result) and PoisonAccessor (error result) fire
// above this and claim those special cases first.
type pureDetector struct{}

func (pureDetector) Name() string  { return "Pure" }
func (pureDetector) Priority() int { return PriorityPure }

func (pureDetector) Detect(s Signature) (Info, bool) {
	if s.HasCtx || s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 0 {
		return Info{}, false
	}
	if s.HasError || len(s.NonErrResults) != 1 {
		return Info{}, false
	}
	return Info{
		Shape:   Pure,
		ValType: s.valType(),
	}, true
}
