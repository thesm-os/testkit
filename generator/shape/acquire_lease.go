// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// acquireLeaseDetector promotes a method to AcquireLease when
// `//testkit:acquire <Release>` names the sibling release method.
// The contract is "double-acquire blocks or errors; release returns
// the lease to availability; cancel/panic releases automatically."
//
// Detection requires an InterfaceContext: the named Release sibling
// must exist with Lifecycle, VoidLifecycle, or PoisonAccessor shape.
type acquireLeaseDetector struct{}

func (acquireLeaseDetector) Name() string  { return "AcquireLease" }
func (acquireLeaseDetector) Priority() int { return PriorityContractAcquireLease }

func (acquireLeaseDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.Acquire)
	if !ok || d.Off || len(d.Args) == 0 {
		return Info{}, false
	}
	sibling, ok := s.Interface.Shapes[d.Args[0]]
	if !ok || !isLifecycleClass(sibling.Shape) {
		return Info{}, false
	}
	carrier := s.Interface.Shapes[s.Method.Name]
	return Info{
		Shape:   AcquireLease,
		KeyType: carrier.KeyType,
		ValType: carrier.ValType,
	}, true
}

// isLifecycleClass reports whether the given shape is a lifecycle-
// style finisher (releases a resource, closes a handle, signals a
// terminal event). Used by AcquireLease to validate the named
// release sibling.
func isLifecycleClass(s Shape) bool {
	switch s {
	case Lifecycle, VoidLifecycle, PoisonAccessor:
		return true
	default:
		return false
	}
}
