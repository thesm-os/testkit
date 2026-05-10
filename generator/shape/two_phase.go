// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// twoPhaseDetector promotes a method to TwoPhase when
// `//testkit:two-phase <Commit> <Rollback>` names the sibling pair.
// The contract is mutex-of-Commit-or-Rollback (one wins) and
// rollback-after-commit is rejected.
//
// Detection requires an InterfaceContext: both named siblings must
// exist on the same interface. The directive is on the Begin method
// (or whichever method returns the transaction handle).
type twoPhaseDetector struct{}

func (twoPhaseDetector) Name() string  { return "TwoPhase" }
func (twoPhaseDetector) Priority() int { return PriorityCompositeTwoPhase }

func (twoPhaseDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.TwoPhase)
	if !ok || d.Off || len(d.Args) < 2 {
		return Info{}, false
	}
	if _, ok := s.Interface.Methods[d.Args[0]]; !ok {
		return Info{}, false
	}
	if _, ok := s.Interface.Methods[d.Args[1]]; !ok {
		return Info{}, false
	}
	carrier := s.Interface.Shapes[s.Method.Name]
	return Info{
		Shape:   TwoPhase,
		ValType: carrier.ValType,
	}, true
}
