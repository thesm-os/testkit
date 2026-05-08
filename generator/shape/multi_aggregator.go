// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import "go.thesmos.sh/testkit/generator"

// multiAggregatorDetector matches `func(ctx?) (V1, V2, error)` —
// no non-ctx params, two non-error results plus error. Models
// `Stats(ctx) (count, total, error)` and similar 2-tuple
// aggregations (ANALYSIS.md G26).
//
// Requires ctx OR error to disambiguate from a hypothetical
// no-ctx no-error 2-tuple Pure (which falls to Unknown).
type multiAggregatorDetector struct{}

func (multiAggregatorDetector) Name() string  { return "MultiAggregator" }
func (multiAggregatorDetector) Priority() int { return PriorityMultiAggregator }

func (multiAggregatorDetector) Detect(s Signature) (Info, bool) {
	if s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 0 {
		return Info{}, false
	}
	if len(s.NonErrResults) != 2 {
		return Info{}, false
	}
	if !s.HasCtx && !s.HasError {
		return Info{}, false
	}
	return Info{
		Shape:    MultiAggregator,
		ValType:  generator.TypeStr(s.NonErrResults[0].Type(), s.Tracker),
		ValType2: generator.TypeStr(s.NonErrResults[1].Type(), s.Tracker),
	}, true
}
