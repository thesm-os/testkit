// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package deleteremovestest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deleteremoves], and the
// in-memory subject they are run against.
package deleteremovestest

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deleteremoves"
)

// ErrNotFound answers to the contract's own sentinel, so a caller checking
// [deleteremoves.ErrGone] and one checking this agree. It wraps rather than
// aliases — a subject may say more than the contract, never less.
//
// Previously a bare local: the contract gained its own sentinel when the
// delete law started comparing against it, and a subject reporting a miss the
// contract cannot recognise fails that law — correctly.
var ErrNotFound = fmt.Errorf("deleteremovestest: %w", deleteremoves.ErrGone)

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu    sync.Mutex
	items map[string]string
}

var _ deleteremoves.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{items: map[string]string{}} }

// Put writes a value.
func (s *InMemory) Put(ctx context.Context, key, value string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = value
	return nil
}

// Delete removes rather than tombstoning, so the subsequent read is a miss and
// not an empty value. The mixin names Read through `read=Read`, which is what
// makes the claim bindable at all.
func (s *InMemory) Delete(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
	return nil
}

// Read reports the miss sentinel for a deleted key, which is the observable
// half of the mixin — Delete reports only whether it failed.
func (s *InMemory) Read(ctx context.Context, key string) (string, error) {
	if err := contextErr(ctx); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.items[key]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("deleteremovestest: nil context")
	}
	return ctx.Err()
}
