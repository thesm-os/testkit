// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// updaterDetector promotes a Writer or CompositeWriter to Updater
// when `//testkit:updater <Reader>` names a sibling Reader. The
// contract is "this writer replaces by key" — distinct from a generic
// Writer in that the per-key invariant is "last-write-wins."
//
// Detection requires an InterfaceContext: the named sibling must
// resolve to a Reader-class shape.
type updaterDetector struct{}

func (updaterDetector) Name() string  { return "Updater" }
func (updaterDetector) Priority() int { return PriorityContractUpdater }

func (updaterDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.Updater)
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
		Shape:    Updater,
		KeyType:  sibling.KeyType, // lookup key inherited from the Reader
		KeyType2: carrier.KeyType2,
		ValType:  carrier.ValType,
	}, true
}
