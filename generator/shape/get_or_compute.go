// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// getOrComputeDetector promotes a method to GetOrCompute when
// `//testkit:singleflight` is present. The contract is "N concurrent
// calls with the same key invoke the supplied compute function
// exactly once; all callers see the same result" — classic
// Go single-flight semantics.
//
// Carrier signature is typically `func(ctx, K, func() V) (V, error)`,
// but the detector only requires the directive plus the parsed
// signature having one or more non-ctx params. Argument-shape
// validation against the func()V slot lives in the spec consumer,
// which has access to the typed parameter list.
type getOrComputeDetector struct{}

func (getOrComputeDetector) Name() string  { return "GetOrCompute" }
func (getOrComputeDetector) Priority() int { return PriorityContractGetOrCompute }

func (getOrComputeDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.Singleflight)
	if !ok || d.Off {
		return Info{}, false
	}
	if len(s.NonCtxParams) == 0 {
		return Info{}, false
	}
	carrier := s.Interface.Shapes[s.Method.Name]
	return Info{
		Shape:   GetOrCompute,
		KeyType: carrier.KeyType,
		ValType: carrier.ValType,
	}, true
}
