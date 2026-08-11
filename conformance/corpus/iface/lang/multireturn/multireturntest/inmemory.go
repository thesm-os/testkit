// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package multireturntest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/lang/multireturn], and the in-memory
// subject they are run against.
package multireturntest

import (
	"context"
	"errors"

	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/multireturn"
)

// ErrNotFound is what Quad reports for an identifier nothing holds.
var ErrNotFound = errors.New("multireturntest: not found")

// InMemory returns the zero of every slot alongside its error, which is what a
// method with several results owes: a caller who checks the error and one who
// checks any value must not disagree about whether the call succeeded.
//
// It lives beside the harness rather than in the package declaring the
// interface: a fixture package states a shape, and the subject a conformance
// run holds to it is scaffolding for the run.
type InMemory struct{ items map[string]string }

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{items: map[string]string{}} }

// Put is not part of the interface. It exists so a test can reach Quad's hit
// path, which no generated check does.
func (s *InMemory) Put(id, value string) { s.items[id] = value }

// Quad zeroes every slot beside its error.
func (s *InMemory) Quad(ctx context.Context, id string) (string, int, bool, error) {
	if err := contextErr(ctx); err != nil {
		return "", 0, false, err
	}
	v, ok := s.items[id]
	if !ok {
		return "", 0, false, ErrNotFound
	}
	return v, len(v), true, nil
}

// Triple returns no error, so nothing can disagree about success.
func (s *InMemory) Triple(_ context.Context, id string) (string, int, bool) {
	v, ok := s.items[id]
	return v, len(v), ok
}

// NoError is the two-slot form of the same.
func (s *InMemory) NoError(_ context.Context, id string) (string, int) {
	v := s.items[id]
	return v, len(v)
}

// PartialZero returns the zero for its first slot and values for the rest,
// which is the defect a check reading one slot of three would pass.
//
// Exported because the demonstration is the point: it is driven through a
// stand-in and asserted to fail.
type PartialZero struct{ *InMemory }

// Quad zeroes only the first slot.
func (PartialZero) Quad(context.Context, string) (string, int, bool, error) {
	return "", 7, true, ErrNotFound
}

// contextErr reports a cancelled or expired context and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("multireturntest: nil context")
	}
	return ctx.Err()
}

// Compile-time proof for both.
var (
	_ multireturn.Wide = (*InMemory)(nil)
	_ multireturn.Wide = PartialZero{}
)
