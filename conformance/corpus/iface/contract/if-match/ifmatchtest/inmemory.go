// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package ifmatchtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/if-match], and the
// in-memory subject they are run against.
package ifmatchtest

import (
	"context"
	"errors"
	"sync"

	ifmatch "go.thesmos.sh/testkit/conformance/corpus/iface/contract/if-match"
)

// ErrAbsent reports a predicate asked about a key the store does not hold.
//
// A verdict rather than a refusal: "does the stored value match" has no answer
// when nothing is stored, and returning false would say the values differ.
var ErrAbsent = errors.New("ifmatchtest: no value under that key")

// ErrMismatch reports a write the predicate declined.
var ErrMismatch = errors.New("ifmatchtest: stored value does not match")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// One map behind one mutex, and both methods take it. The predicate and the
// write have to see the same state or the contract says nothing: a caller that
// matched and then wrote through a lock taken twice is the lost-update the
// conditional write exists to prevent.
type InMemory struct {
	mu     sync.Mutex
	values map[string]ifmatch.Value
}

var _ ifmatch.Contract = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{values: map[string]ifmatch.Value{}} }

// Put establishes a key nothing holds, and otherwise writes only what Match
// admits.
//
// Establishing rather than refusing, because a conditional write over an empty
// store would have no first writer — every call would fail the predicate it was
// waiting for. The condition is about replacing state, which is what the
// contract is for.
func (s *InMemory) Put(ctx context.Context, v ifmatch.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, present := s.values[v.Key]; present && !matches(current, v) {
		return ErrMismatch
	}
	s.values[v.Key] = v
	return nil
}

// Match reports whether the stored value is the one the caller expects.
//
// The context is consulted first. A cancelled caller wants the call abandoned
// rather than answered, and a verdict returned for a cancelled question is one
// they will act on after deciding not to ask.
func (s *InMemory) Match(ctx context.Context, v ifmatch.Value) (bool, error) {
	if err := contextErr(ctx); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, present := s.values[v.Key]
	if !present {
		return false, ErrAbsent
	}
	return matches(current, v), nil
}

// matches is the predicate itself, in one place so Put and Match cannot come to
// different answers about one pair.
func matches(current, want ifmatch.Value) bool { return current.Body == want.Body }

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("ifmatchtest: nil context")
	}
	return ctx.Err()
}
