// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import "go.thesmos.sh/testkit/generator"

// compositeWriterDetector matches `func(ctx?, K1, V) error` — two
// non-ctx parameters and an error-only return. Models namespaced
// stores, tagged caches, and two-key indexes.
//
// Three-or-more-key Writers route to MultiArgWriter (priority
// 750) when ctx is present, or fall to Unknown without ctx.
type compositeWriterDetector struct{}

func (compositeWriterDetector) Name() string  { return "CompositeWriter" }
func (compositeWriterDetector) Priority() int { return PriorityCompositeWriter }

func (compositeWriterDetector) Detect(s Signature) (Info, bool) {
	if s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 2 {
		return Info{}, false
	}
	if !s.HasError || len(s.NonErrResults) != 0 {
		return Info{}, false
	}
	return Info{
		Shape:    CompositeWriter,
		KeyType:  generator.TypeStr(s.NonCtxParams[0].Type(), s.Tracker),
		KeyType2: generator.TypeStr(s.NonCtxParams[1].Type(), s.Tracker),
		ValType:  generator.TypeStr(s.NonCtxParams[1].Type(), s.Tracker),
	}, true
}
