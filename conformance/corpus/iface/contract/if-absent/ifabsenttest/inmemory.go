// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package ifabsenttest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/if-absent], and the
// in-memory subject they are run against.
package ifabsenttest

import (
	"context"
	"errors"
	"fmt"
	"sync"

	ifabsent "go.thesmos.sh/testkit/conformance/corpus/iface/contract/if-absent"
)

// ErrPresent reports a write for a key the store already holds.
//
// Named rather than anonymous because refusal is the contract: a caller
// distinguishing "somebody else got there first" from "the store is down"
// cannot do it against an unlabelled error.
// Wrapping the declaration's own conflict sentinel: the sharpened check
// asserts errors.Is against ifabsent.ErrExists, and a subject spelling its
// refusal through the chain is the shape consumers ship.
var ErrPresent = fmt.Errorf("ifabsenttest: key already present: %w", ifabsent.ErrExists)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// A map behind a mutex. The lock is what makes the check-then-write one
// operation — an if-absent store that looked and then wrote would admit two
// concurrent writers for one key, which is the failure the contract exists to
// name.
type InMemory struct {
	mu     sync.Mutex
	values map[string]ifabsent.Value
}

var _ ifabsent.Contract = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{values: map[string]ifabsent.Value{}} }

// Put stores a value under its key, and refuses a key the store already holds.
//
// The context is consulted first, before the key is looked at. A caller who
// cancelled wants the call abandoned rather than answered, and a store
// reporting ErrPresent for a cancelled write has told them something about
// state they no longer asked about.
func (s *InMemory) Put(ctx context.Context, v ifabsent.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, present := s.values[v.Key]; present {
		return ErrPresent
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
		return errors.New("ifabsenttest: nil context")
	}
	return ctx.Err()
}
