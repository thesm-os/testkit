// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package streamreflectsmutationstest holds the generated harness and double
// for [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/streamreflectsmutations],
// and the in-memory subject they are run against.
package streamreflectsmutationstest

import (
	"context"
	"errors"
	"iter"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/streamreflectsmutations"
)

// InMemory yields live rather than from a snapshot, which is the mixin: an item
// added while a consumer is mid-range is one that consumer sees.
//
// The distinction is invisible to any check that ranges the stream to
// completion before mutating — which is every check a signature can imply.
type InMemory struct {
	mu    sync.Mutex
	items []string
}

var _ streamreflectsmutations.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty subject.
func NewInMemory() *InMemory { return &InMemory{} }

// Add appends an item, and is the partner the mixin names through `mutate=Add`.
func (s *InMemory) Add(ctx context.Context, item string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, item)
	return nil
}

// Stream reads the backing slice by index on each step rather than copying it
// up front, so an Add landing mid-range is yielded. A snapshot would satisfy
// every generated check and lose the property the mixin names.
//
// The lock is taken per step and released before the yield: holding it across
// the consumer's body would deadlock a consumer that adds, which is exactly the
// consumer this mixin is about.
func (s *InMemory) Stream(ctx context.Context) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		if err := contextErr(ctx); err != nil {
			yield("", err)
			return
		}
		for i := 0; ; i++ {
			s.mu.Lock()
			if i >= len(s.items) {
				s.mu.Unlock()
				return
			}
			item := s.items[i]
			s.mu.Unlock()

			if !yield(item, nil) {
				return
			}
		}
	}
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("streamreflectsmutationstest: nil context")
	}
	return ctx.Err()
}
