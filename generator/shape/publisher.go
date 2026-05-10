// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// publisherDetector promotes a method to Publisher when
// `//testkit:publisher <Subscribe>` names the sibling subscribe
// method. The contract is "every active subscriber sees every
// publish, with delivery semantics governed by the
// `//testkit:delivery` mixin."
//
// Detection requires an InterfaceContext: the named Subscribe
// sibling must exist on the same interface.
type publisherDetector struct{}

func (publisherDetector) Name() string  { return "Publisher" }
func (publisherDetector) Priority() int { return PriorityContractPublisher }

func (publisherDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.Publisher)
	if !ok || d.Off || len(d.Args) == 0 {
		return Info{}, false
	}
	if _, ok := s.Interface.Methods[d.Args[0]]; !ok {
		return Info{}, false
	}
	carrier := s.Interface.Shapes[s.Method.Name]
	return Info{
		Shape:   Publisher,
		KeyType: carrier.KeyType,
		ValType: carrier.ValType,
	}, true
}
