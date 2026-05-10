// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// cursorDetector promotes a method to Cursor when
// `//testkit:cursor <Close>` names the sibling Close method. The
// contract is "Next yields each element exactly once until
// exhaustion; Close is idempotent; Next-after-Close returns the
// not-found sentinel."
//
// Detection requires an InterfaceContext: the named Close sibling
// must exist with Lifecycle/VoidLifecycle/PoisonAccessor shape.
type cursorDetector struct{}

func (cursorDetector) Name() string  { return "Cursor" }
func (cursorDetector) Priority() int { return PriorityCompositeCursor }

func (cursorDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.Cursor)
	if !ok || d.Off || len(d.Args) == 0 {
		return Info{}, false
	}
	sibling, ok := s.Interface.Shapes[d.Args[0]]
	if !ok || !isLifecycleClass(sibling.Shape) {
		return Info{}, false
	}
	carrier := s.Interface.Shapes[s.Method.Name]
	return Info{
		Shape:   Cursor,
		ValType: carrier.ValType,
	}, true
}
