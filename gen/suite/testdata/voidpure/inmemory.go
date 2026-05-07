// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package voidpure

// InMemoryStream implements [Stream] for testing.
type InMemoryStream struct {
	data   []byte
	closed bool
}

func NewInMemoryStream() *InMemoryStream {
	return &InMemoryStream{}
}

func (s *InMemoryStream) Sum() Digest {
	var d Digest
	copy(d[:], s.data)
	return d
}

func (s *InMemoryStream) Reset() {
	s.data = nil
}

func (s *InMemoryStream) Close() {
	s.closed = true
}
