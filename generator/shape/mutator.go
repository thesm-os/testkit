// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import "go.thesmos.sh/testkit/generator/directive"

// mutatorDetector matches `func(ctx?, V)` — exactly one non-ctx
// parameter and a void return. Auto-detected from the signature
// (ANALYSIS.md G32); the legacy //testkit:mutator directive is
// optional confirmation.
//
// Two opt-out forms are honored:
//
//   - //testkit:not-mutator
//   - //testkit:mutator off
//
// Either suppresses Mutator detection and lets the method fall
// through to lower-priority detectors (typically Unknown).
type mutatorDetector struct{}

func (mutatorDetector) Name() string  { return "Mutator" }
func (mutatorDetector) Priority() int { return PriorityMutator }

func (mutatorDetector) Detect(s Signature) (Info, bool) {
	if s.Variadic != nil {
		return Info{}, false
	}
	if len(s.NonCtxParams) != 1 {
		return Info{}, false
	}
	if len(s.AllResults) != 0 {
		return Info{}, false
	}
	if isMutatorOptOut(s) {
		return Info{}, false
	}
	return Info{
		Shape:   Mutator,
		ValType: s.keyType(),
	}, true
}

func isMutatorOptOut(s Signature) bool {
	for _, d := range s.Directives {
		if d.Name == directive.NotMutator {
			return true
		}
		if d.Name == directive.Mutator && d.Off {
			return true
		}
	}
	return false
}
