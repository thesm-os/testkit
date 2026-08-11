// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package errorstest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/errors], and the
// in-memory subject they are run against.
package errorstest

import (
	"context"
	stderrors "errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/errors"
)

// gone is the key this subject reports ErrGone for, so both declared sentinels
// are reachable rather than only the miss.
const gone = "gone"

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu    sync.Mutex
	items map[string]string
}

var _ errors.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{items: map[string]string{}} }

// Get reports one of the sentinels the source declares, which is the mixin's
// claim: a caller distinguishes absence from removal by comparing against a
// named error rather than by reading a message.
func (s *InMemory) Get(ctx context.Context, key string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	if key == gone {
		return "", errors.ErrGone
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.items[key]
	if !ok {
		return "", errors.ErrNotFound
	}
	return v, nil
}

// Put seeds the store, which the interface cannot do.
func (s *InMemory) Put(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = value
}

// GoneKey is the key Get reports ErrGone for.
func GoneKey() string { return gone }

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return stderrors.New("errorstest: nil context")
	}
	return ctx.Err()
}
