// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package multiargwritertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/detector/multiargwriter], and
// the in-memory subject they are run against — scaffolding for the run, so it
// lives beside the harness rather than in the package stating the shape.
package multiargwritertest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/multiargwriter"
)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]string
}

var _ multiargwriter.MultiArgWriter = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{values: map[string]string{}} }

// Set spreads the value across three parameters rather than taking a struct —
// past the composite pair, into the boundary the fixture's directory names.
// The harness seeds through it either way, because arity is not something a
// seed has to know.
func (s *InMemory) Set(ctx context.Context, key, body, mime string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = mime + ":" + body
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
		return errors.New("multiargwritertest: nil context")
	}
	return ctx.Err()
}
