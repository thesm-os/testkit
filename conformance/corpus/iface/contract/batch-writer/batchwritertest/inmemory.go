// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package batchwritertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/batch-writer], and
// the in-memory subject they are run against.
package batchwritertest

import (
	"context"
	"errors"
	"sync"

	batchwriter "go.thesmos.sh/testkit/conformance/corpus/iface/contract/batch-writer"
)

// ErrUnkeyed reports a value with nothing to file it under.
//
// The subject's one way to fail, and it exists so `mode=atomic` has something
// to be about: a write that cannot fail leaves "nothing was partially applied"
// with no case to observe.
var ErrUnkeyed = errors.New("batchwritertest: a value needs a key")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// Two structures updated together — the values and the order they arrived in —
// because that is what makes atomicity visible at all. A store with one map has
// no partial state to leave behind, so it satisfies `mode=atomic` by having
// nothing to get wrong.
type InMemory struct {
	mu     sync.Mutex
	values map[string]batchwriter.Value
	order  []string
}

var _ batchwriter.Contract = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{values: map[string]batchwriter.Value{}}
}

// Put applies a value, or leaves the store exactly as it found it.
//
// Everything that can be refused is refused before anything is written. That
// ordering is the whole of `mode=atomic`: a version validating between the two
// updates would leave the index naming a value the store does not hold.
func (s *InMemory) Put(ctx context.Context, v batchwriter.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if v.Key == "" {
		return ErrUnkeyed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, present := s.values[v.Key]; !present {
		s.order = append(s.order, v.Key)
	}
	s.values[v.Key] = v
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
		return errors.New("batchwritertest: nil context")
	}
	return ctx.Err()
}
