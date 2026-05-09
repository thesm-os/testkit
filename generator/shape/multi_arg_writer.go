// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

// multiArgWriterDetector matches `func(ctx, p1, p2, p3, ...) error`
// where there are 3+ non-ctx parameters and an error return. It
// gives ctx-respect coverage to method shapes with arity beyond
// what CompositeWriter can model.
//
// Requires ctx — without it, the method has no cancellation
// surface and falls to Unknown.
type multiArgWriterDetector struct{}

func (multiArgWriterDetector) Name() string  { return "MultiArgWriter" }
func (multiArgWriterDetector) Priority() int { return PriorityMultiArgWriter }

func (multiArgWriterDetector) Detect(s Signature) (Info, bool) {
	if !s.HasCtx || s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) < 3 {
		return Info{}, false
	}
	if !s.HasError {
		return Info{}, false
	}
	// Allow either error-only return or (R, error) — the shape's
	// contract is "ctx-respecting multi-arg call." Multi-non-error
	// results route to MultiReader/MultiAggregator above.
	if len(s.NonErrResults) > 1 {
		return Info{}, false
	}
	return Info{Shape: MultiArgWriter}, true
}
