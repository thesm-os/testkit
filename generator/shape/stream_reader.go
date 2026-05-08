// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape

// streamReaderDetector matches methods that return iter.Seq[V] or
// iter.Seq2[V, error]. It fires before every other detector
// because an iter return type unambiguously identifies a stream.
type streamReaderDetector struct{}

func (streamReaderDetector) Name() string  { return "StreamReader" }
func (streamReaderDetector) Priority() int { return PriorityStreamReader }

func (streamReaderDetector) Detect(s Signature) (Info, bool) {
	if !s.Iter.IsSeq && !s.Iter.IsSeq2 {
		return Info{}, false
	}
	return Info{
		Shape:   StreamReader,
		ValType: s.Iter.ValType,
		Iter:    s.Iter,
	}, true
}
