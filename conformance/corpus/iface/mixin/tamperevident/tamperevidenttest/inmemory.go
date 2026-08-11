// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package tamperevidenttest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/tamperevident], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package tamperevidenttest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/tamperevident"
)

// ErrTampered is what Verify reports for a value altered since it was stored.
var ErrTampered = errors.New("tamperevidenttest: stored value does not match its digest")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	body   string
	digest string
}

var _ tamperevident.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{} }

// Store records the value beside a digest of it.
func (s *InMemory) Store(ctx context.Context, body string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.body, s.digest = body, digestOf(body)
	return nil
}

// Corrupt alters the body and leaves the digest alone, which is exactly the
// state a tamper-evident store must notice.
func (s *InMemory) Corrupt(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.body += "!"
	return nil
}

// Verify recomputes the digest and compares.
func (s *InMemory) Verify(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if digestOf(s.body) != s.digest {
		return ErrTampered
	}
	return nil
}

// digestOf is a stand-in for a real hash: the fixture states the shape of the
// claim, and a cryptographic digest would say nothing more about it.
func digestOf(body string) string {
	return body + "#" + string(rune(len(body)))
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("tamperevidenttest: nil context")
	}
	return ctx.Err()
}
