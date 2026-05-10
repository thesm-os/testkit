// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// watcherDetector promotes a method to Watcher when
// `//testkit:watcher <Trigger>` names a sibling whose calls cause
// the watcher to deliver an event. The contract is "the watcher
// returns once the trigger fires" — used by config systems, event
// streams, and pub/sub-style subscribers.
//
// Detection requires an InterfaceContext: the named Trigger sibling
// must exist on the same interface. Carrier is typically Lifecycle,
// StreamReader, or a Reader returning a channel; the structural
// shape isn't constrained here so the detector accepts any signature
// with the directive present.
type watcherDetector struct{}

func (watcherDetector) Name() string  { return "Watcher" }
func (watcherDetector) Priority() int { return PriorityContractWatcher }

func (watcherDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.Watcher)
	if !ok || d.Off || len(d.Args) == 0 {
		return Info{}, false
	}
	if _, ok := s.Interface.Methods[d.Args[0]]; !ok {
		return Info{}, false
	}
	carrier := s.Interface.Shapes[s.Method.Name]
	return Info{
		Shape:   Watcher,
		KeyType: carrier.KeyType,
		ValType: carrier.ValType,
	}, true
}
