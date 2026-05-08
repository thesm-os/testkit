// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import "go.thesmos.sh/testkit/generator"

// multiReaderDetector matches `func(ctx?, K) (V1, V2, error)` —
// one non-ctx parameter and two non-error results plus error.
// Models "get the entity + metadata" idioms (ANALYSIS.md G26).
//
// Three-or-more non-error results fall to Unknown — too rare to
// type usefully.
type multiReaderDetector struct{}

func (multiReaderDetector) Name() string  { return "MultiReader" }
func (multiReaderDetector) Priority() int { return PriorityMultiReader }

func (multiReaderDetector) Detect(s Signature) (Info, bool) {
	if s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 1 {
		return Info{}, false
	}
	if !s.HasError || len(s.NonErrResults) != 2 {
		return Info{}, false
	}
	return Info{
		Shape:    MultiReader,
		KeyType:  s.keyType(),
		ValType:  generator.TypeStr(s.NonErrResults[0].Type(), s.Tracker),
		ValType2: generator.TypeStr(s.NonErrResults[1].Type(), s.Tracker),
	}, true
}
