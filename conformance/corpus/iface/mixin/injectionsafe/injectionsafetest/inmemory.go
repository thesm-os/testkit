// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package injectionsafetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/injectionsafe], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package injectionsafetest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/injectionsafe"
)

// ErrMissing is what Load reports for a key nothing stored.
var ErrMissing = errors.New("injectionsafetest: nothing stored under the key")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]string
}

var _ injectionsafe.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{values: map[string]string{}} }

// Store keeps the value verbatim: interpreting either slot is the defect this
// mixin exists to name, so the subject deliberately does no parsing at all.
func (s *InMemory) Store(ctx context.Context, key, value string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
	return nil
}

// Load answers the stored data, byte for byte, or reports a miss.
func (s *InMemory) Load(ctx context.Context, key string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	value, held := s.values[key]
	if !held {
		return "", ErrMissing
	}
	return value, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("injectionsafetest: nil context")
	}
	return ctx.Err()
}
