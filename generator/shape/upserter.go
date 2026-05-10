// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// upserterDetector promotes a Writer or CompositeWriter to Upserter
// when `//testkit:upserter <Reader>` names a sibling Reader. The
// contract is "idempotent insert-or-update": repeated calls with the
// same input produce the same observable state.
type upserterDetector struct{}

func (upserterDetector) Name() string  { return "Upserter" }
func (upserterDetector) Priority() int { return PriorityContractUpserter }

func (upserterDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.Upserter)
	if !ok || d.Off || len(d.Args) == 0 {
		return Info{}, false
	}
	carrier := s.Interface.Shapes[s.Method.Name]
	if carrier.Shape != Writer && carrier.Shape != CompositeWriter {
		return Info{}, false
	}
	sibling, ok := s.Interface.Shapes[d.Args[0]]
	if !ok || !isReaderClass(sibling.Shape) {
		return Info{}, false
	}
	return Info{
		Shape:    Upserter,
		KeyType:  sibling.KeyType,
		KeyType2: carrier.KeyType2,
		ValType:  carrier.ValType,
	}, true
}
