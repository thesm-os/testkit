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
	"strconv"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/compositewriter"
)

// ErrNoKey is what Store refuses a value with no key.
var ErrNoKey = errors.New("compositewritertest: value has no key")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	rev    int
	values map[string]compositewriter.Value
}

var _ compositewriter.CompositeWriter = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{values: map[string]compositewriter.Value{}}
}

// Store returns a receipt beside its error, which is what takes this out of the
// writer shape: a write reporting only failure has nothing to hold to the zero,
// and one returning what it wrote does.
//
// It refuses a value with no key so the receipt has a failure path at all. The
// harness reaches it through the alternate the fixture supplies, since every
// derived value has a key and would succeed.
func (s *InMemory) Store(
	ctx context.Context, v compositewriter.Value,
) (compositewriter.Value, error) {
	if err := contextErr(ctx); err != nil {
		return compositewriter.Value{}, err
	}
	if v.Key == "" {
		return compositewriter.Value{}, ErrNoKey
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rev++
	v.Rev = strconv.Itoa(s.rev)
	s.values[v.Key] = v
	return v, nil
}

// No out-of-band observer, unlike the other writers here. Store returns the
// receipt, so what it wrote is already on the interface — a Stored method would
// be a second way to ask a question the shape answers itself.

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
