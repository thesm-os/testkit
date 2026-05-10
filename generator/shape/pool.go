// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// poolDetector promotes a method to Pool when `//testkit:pool <Put>`
// names the sibling release method on the same interface. The
// contract is "every Get balances with a matching Put; the pool is
// leak-free across cycles; double-Put is rejected."
//
// Detection requires an InterfaceContext: the named Put sibling must
// exist. Composite-tier shapes intercept at priority 2000+, before
// any contract- or signature-tier detector sees the carrier.
type poolDetector struct{}

func (poolDetector) Name() string  { return "Pool" }
func (poolDetector) Priority() int { return PriorityCompositePool }

func (poolDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.Pool)
	if !ok || d.Off || len(d.Args) == 0 {
		return Info{}, false
	}
	if _, ok := s.Interface.Methods[d.Args[0]]; !ok {
		return Info{}, false
	}
	carrier := s.Interface.Shapes[s.Method.Name]
	return Info{
		Shape:   Pool,
		KeyType: carrier.KeyType,
		ValType: carrier.ValType,
	}, true
}
