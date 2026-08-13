// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package compositewritertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/compositewriter], and
// the in-memory subject they are run against — scaffolding for the run, so it
// lives beside the harness rather than in the package stating the shape.
package compositewritertest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/compositewriter"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]compositewriter.Value
}

var _ compositewriter.CompositeWriter = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{values: map[string]compositewriter.Value{}}
}

// Set stores the value under the key beside it — the composite pair whole,
// which is the shape the fixture's directory names.
func (s *InMemory) Set(ctx context.Context, key string, v compositewriter.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = v
	return nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("compositewritertest: nil context")
	}
	return ctx.Err()
}
