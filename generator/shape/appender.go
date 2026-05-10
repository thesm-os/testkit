// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"go.thesmos.sh/testkit/generator/directive"
)

// appenderDetector promotes a Writer-with-result to Appender when
// `//testkit:appender` is present. The contract is "monotonic,
// gap-free offsets": the returned offset never decreases across
// calls, and observers see a contiguous sequence.
//
// The detector accepts any single-input + single-non-error-result +
// error signature (same shape as Reader); the directive
// disambiguates from Reader at codegen time.
type appenderDetector struct{}

func (appenderDetector) Name() string  { return "Appender" }
func (appenderDetector) Priority() int { return PriorityContractAppender }

func (appenderDetector) Detect(s Signature) (Info, bool) {
	if s.Interface == nil {
		return Info{}, false
	}
	d, ok := findDirective(s.Directives, directive.Appender)
	if !ok || d.Off {
		return Info{}, false
	}
	if !singleInputSingleResult(s) {
		return Info{}, false
	}
	return Info{
		Shape:   Appender,
		ValType: s.keyType(), // input value (the appended entry)
		RetType: s.valType(), // returned offset
	}, true
}
