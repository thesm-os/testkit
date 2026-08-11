// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package streamreadertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/streamreader], and
// the in-memory subject they are run against — scaffolding for the run, so it
// lives beside the harness rather than in the package stating the shape.
package streamreadertest

import (
	"context"
	"iter"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/streamreader"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values []streamreader.Value
}

var _ streamreader.StreamReader = (*InMemory)(nil)

// NewInMemory returns an empty stream.
func NewInMemory() *InMemory { return &InMemory{} }

// List yields lazily and stops when the consumer does, which is the shape's own
// law: a consumer may break out of the range, so the implementation must not
// assume the sequence is drained.
//
// The snapshot is taken before the sequence is returned rather than read inside
// it. A generated smoke check calls List and never ranges it, which is exactly
// how a lazy implementation that captures a lock ends up holding one nobody
// releases.
//
// The context is checked once, here, for the same reason: with the error
// carried per element there is nowhere to report a cancelled context except the
// first yield, and a caller who never ranges would never see it.
func (s *InMemory) List(ctx context.Context) iter.Seq2[streamreader.Value, error] {
	if ctx == nil || ctx.Err() != nil {
		err := context.Canceled
		if ctx != nil {
			err = ctx.Err()
		}
		return func(yield func(streamreader.Value, error) bool) {
			yield(streamreader.Value{}, err)
		}
	}

	s.mu.Lock()
	snapshot := make([]streamreader.Value, len(s.values))
	copy(snapshot, s.values)
	s.mu.Unlock()

	return func(yield func(streamreader.Value, error) bool) {
		for _, v := range snapshot {
			if !yield(v, nil) {
				return
			}
		}
	}
}

// Put appends a value, so a test can seed an interface that declares no writer
// to seed itself through.
func (s *InMemory) Put(v streamreader.Value) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = append(s.values, v)
}
