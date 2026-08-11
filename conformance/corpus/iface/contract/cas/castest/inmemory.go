// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package castest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/cas], and the
// in-memory subject they are run against.
package castest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/cas"
)

// versioned is a stored value and the revision it was written at.
type versioned struct {
	value   cas.Value
	version int
}

// InMemory is the implementation the generated conformance harness is run
// against.
//
// The version is kept beside every value because that is what `version=Version`
// names — the field a compare-and-set compares. The fixture's Put takes no
// version to compare against, so this subject can only maintain the counter;
// what an implementation does with a stale one is the law's question, and
// `AUTO-CAS-ATOMIC-ONE-WINNER` needs accumulated state to ask it.
type InMemory struct {
	mu     sync.Mutex
	values map[string]versioned
}

var _ cas.Contract = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{values: map[string]versioned{}} }

// Put writes a value and advances its version.
//
// One increment per accepted write, under the lock that performs it. A counter
// advanced outside the lock is a counter two writers can share, which is the
// lost update the whole contract exists to detect.
func (s *InMemory) Put(ctx context.Context, v cas.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.values[v.Key]
	s.values[v.Key] = versioned{value: v, version: current.version + 1}
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
		return errors.New("castest: nil context")
	}
	return ctx.Err()
}
