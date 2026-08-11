// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package paginationtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/pagination], and the
// in-memory subject they are run against.
package paginationtest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/pagination"
)

// ErrNotFound reports a key the store does not hold.
var ErrNotFound = errors.New("paginationtest: no value under that key")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// A plain keyed store, because that is what the fixture declares. The
// contract's `cursor=Cursor` names a page token the interface has no parameter
// for, so nothing here can page — which is a fact about the fixture rather than
// about the subject, and it is why the paging claims are stated against
// composite/paginated-reader instead.
type InMemory struct {
	mu     sync.Mutex
	values map[string]pagination.Value
}

var _ pagination.Contract = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{values: map[string]pagination.Value{}} }

// Store puts a value in, so a test can seed an interface declaring no writer.
func (s *InMemory) Store(v pagination.Value) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
}

// Get returns the zero value alongside every error it reports, so a caller who
// checks the error and one who checks the value do not disagree about whether
// the call succeeded.
func (s *InMemory) Get(ctx context.Context, key string) (pagination.Value, error) {
	if err := contextErr(ctx); err != nil {
		return pagination.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, present := s.values[key]
	if !present {
		return pagination.Value{}, ErrNotFound
	}
	return v, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("paginationtest: nil context")
	}
	return ctx.Err()
}
