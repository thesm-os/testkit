// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import "go.thesmos.sh/testkit/generator/directive"

// deleterDetector matches `func(ctx?, K) error` annotated with
// //testkit:deleter. Without the directive the same signature is
// a Writer; the directive elevates it to Deleter so the suite
// can emit delete-removes invariants and the bench can isolate
// the delete-by-key cost.
type deleterDetector struct{}

func (deleterDetector) Name() string  { return "Deleter" }
func (deleterDetector) Priority() int { return PriorityDeleter }

func (deleterDetector) Detect(s Signature) (Info, bool) {
	if !s.HasDirective(directive.Deleter) {
		return Info{}, false
	}
	if s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 1 {
		return Info{}, false
	}
	if !s.HasError || len(s.NonErrResults) != 0 {
		return Info{}, false
	}
	return Info{
		Shape:   Deleter,
		KeyType: s.keyType(),
	}, true
}
