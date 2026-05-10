// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// sagaDetector promotes a method to Saga when
// `//testkit:saga <Steps>...` names every step method on the
// interface. The contract is "either all steps complete in order, or
// the partial-completion runs full compensation in reverse." Saga is
// the only composite-tier shape where the directive is required —
// auto-detection across N free-form chained methods is too
// unreliable to ship.
//
// Detection requires an InterfaceContext: every named step must
// exist on the same interface.
type sagaDetector struct{}

func (sagaDetector) Name() string  { return "Saga" }
func (sagaDetector) Priority() int { return PriorityCompositeSaga }

func (sagaDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.Saga)
	if !ok || d.Off || len(d.Args) == 0 {
		return Info{}, false
	}
	for _, step := range d.Args {
		if _, ok := s.Interface.Methods[step]; !ok {
			return Info{}, false
		}
	}
	carrier := s.Interface.Shapes[s.Method.Name]
	return Info{
		Shape:   Saga,
		ValType: carrier.ValType,
	}, true
}
