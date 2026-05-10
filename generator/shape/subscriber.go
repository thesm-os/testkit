// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// subscriberDetector promotes a method to Subscriber when
// `//testkit:subscribe` is present. Subscriber sits opposite
// Publisher in the pub/sub pair: it returns a channel or accepts a
// callback that fires on each Publisher publication.
//
// The detector accepts any signature with the directive; the
// generator's spec consumer validates the channel-or-callback shape.
type subscriberDetector struct{}

func (subscriberDetector) Name() string  { return "Subscriber" }
func (subscriberDetector) Priority() int { return PriorityContractSubscriber }

func (subscriberDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.Subscribe)
	if !ok || d.Off {
		return Info{}, false
	}
	carrier := s.Interface.Shapes[s.Method.Name]
	return Info{
		Shape:   Subscriber,
		KeyType: carrier.KeyType,
		ValType: carrier.ValType,
	}, true
}
