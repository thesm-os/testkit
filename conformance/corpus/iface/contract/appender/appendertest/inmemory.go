// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package appendertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/appender], and the
// in-memory subject they are run against.
package appendertest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/appender"
)

// InMemory is the implementation the generated conformance harness is run
// against.
//
// A slice and a mutex. Nothing removes from it, which is the contract rather
// than a missing feature: `AUTO-APPEND-ONLY-GROWS` is the claim that the log
// only ever gets longer, and a subject offering a delete would be a different
// one.
type InMemory struct {
	mu      sync.Mutex
	entries []appender.Value
}

var _ appender.Contract = (*InMemory)(nil)

// NewInMemory returns an empty log.
func NewInMemory() *InMemory { return &InMemory{} }

// Run appends the value and answers its offset — the position it landed at,
// which only ever grows.
//
// The context is consulted first, so a cancelled caller's entry is not written.
// An append-only log cannot take one back.
func (s *InMemory) Run(ctx context.Context, v appender.Value) (int64, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, v)
	return int64(len(s.entries) - 1), nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("appendertest: nil context")
	}
	return ctx.Err()
}
