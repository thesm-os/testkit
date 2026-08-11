// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package boundedtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/bounded], and the
// in-memory subject they are run against.
package boundedtest

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/bounded"
)

// Limit is the ceiling the source declares through `//testkit:mixin bounded
// limit=100`, restated here because the subject has to honour it and nothing
// generated reads it: bounded is the model tier's under ADR-0018.
const Limit = 100

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu    sync.Mutex
	items []string
}

var _ bounded.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty collection.
func NewInMemory() *InMemory { return &InMemory{} }

// List never returns more than Limit, which is the whole of the mixin and no
// part of the signature — `List(ctx) ([]string, error)` is the same shape
// bounded or not.
func (s *InMemory) List(ctx context.Context) ([]string, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := min(len(s.items), Limit)
	out := make([]string, n)
	copy(out, s.items[:n])
	return out, nil
}

// Fill adds n items, so a test can push the collection past its ceiling.
func (s *InMemory) Fill(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range n {
		s.items = append(s.items, strconv.Itoa(i))
	}
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("boundedtest: nil context")
	}
	return ctx.Err()
}
