// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package embeddedforeigntest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/lang/embeddedforeign], and the
// in-memory subject they are run against.
package embeddedforeigntest

import (
	"context"
	"errors"
	"io"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/embeddedforeign"
)

// ErrNotFound is what Read reports for a key nothing holds.
var ErrNotFound = errors.New("embeddedforeigntest: not found")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// It lives beside the harness rather than in the package declaring the
// interface, because that is what it is for: a fixture package states a shape,
// and the subject a conformance run holds to it is scaffolding for the run.
type InMemory struct {
	mu     sync.Mutex
	items  map[string]string
	closed bool
}

// Compile-time proof that the subject satisfies both the interface and the
// standard-library one it embeds.
//
// The second is what makes this fixture's claim observable: Close has no
// declaration in embeddedforeign, so a method set that stopped at the run's own
// source would have generated a harness that never mentioned it — and this line
// would still compile, which is why the assertion that matters is the generated
// AssertStreamCloseSmoke rather than either of these.
var (
	_ embeddedforeign.Stream = (*InMemory)(nil)
	_ io.Closer              = (*InMemory)(nil)
)

// NewInMemory returns an open, empty stream.
func NewInMemory() *InMemory { return &InMemory{items: map[string]string{}} }

// Read returns the zero value alongside every error it reports, which is the
// property the generated check is about: a caller who checks the error and one
// who checks the value must not disagree about whether the call succeeded.
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

// Close is idempotent, which io.Closer does not require and a caller with a
// deferred close plus an explicit one relies on anyway.
//
// It takes no context, so the generated family for it is a smoke call and
// nothing else — every other signature-derived check is a claim about a
// parameter this method does not have.
func (s *InMemory) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// Put stores a value, so a test can seed a stream whose interface declares no
// writer to seed itself through.
func (s *InMemory) Put(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = value
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("embeddedforeigntest: nil context")
	}
	return ctx.Err()
}
