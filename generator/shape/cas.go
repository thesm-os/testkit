// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// casDetector promotes a Writer to CompareAndSwap when
// `//testkit:cas <VersionField>` names the version-bearing field on
// the input value. The contract is "exactly one concurrent writer
// wins; losers see a version-mismatch error."
//
// VersionField resolution against the input type happens in the
// generator's spec consumer; this detector only verifies the
// directive is present with an arg.
type casDetector struct{}

func (casDetector) Name() string  { return "CompareAndSwap" }
func (casDetector) Priority() int { return PriorityContractCAS }

func (casDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.CAS)
	if !ok || d.Off || len(d.Args) == 0 {
		return Info{}, false
	}
	carrier := s.Interface.Shapes[s.Method.Name]
	if carrier.Shape != Writer && carrier.Shape != CompositeWriter {
		return Info{}, false
	}
	return Info{
		Shape:    CompareAndSwap,
		KeyType:  carrier.KeyType,
		KeyType2: carrier.KeyType2,
		ValType:  carrier.ValType,
	}, true
}
