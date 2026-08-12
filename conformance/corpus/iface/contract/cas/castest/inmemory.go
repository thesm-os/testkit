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

// ErrEmpty is what Get reports for a cell nothing has written.
var ErrEmpty = errors.New("castest: the cell holds nothing yet")

// InMemory is the implementation the generated conformance harness is run
// against: one slot, guarded by the version the value carries in the field
// `version=Version` names.
//
// A write is accepted exactly when its embedded version matches the cell's —
// zero against a fresh cell — and the cell then advances by one. The stored
// value is answered verbatim: the version a reader sees is the one the
// writer sent, which is the same dialect the derived VersionedCell oracle
// speaks, so the two agree or the subject is wrong.
type InMemory struct {
	mu      sync.Mutex
	value   cas.Value
	current int64
	present bool
}

var _ cas.Contract = (*InMemory)(nil)

// NewInMemory returns an empty cell.
func NewInMemory() *InMemory { return &InMemory{} }

// Put accepts v exactly when its version matches the cell's, and advances
// the cell by one.
func (s *InMemory) Put(ctx context.Context, v cas.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if v.Version != s.current {
		return cas.ErrMismatch
	}
	s.value = v
	s.current++
	s.present = true
	return nil
}

// Get answers the stored value verbatim, or reports an empty cell.
func (s *InMemory) Get(ctx context.Context) (cas.Value, error) {
	if err := contextErr(ctx); err != nil {
		return cas.Value{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.present {
		return cas.Value{}, ErrEmpty
	}
	return s.value, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("castest: nil context")
	}
	return ctx.Err()
}
