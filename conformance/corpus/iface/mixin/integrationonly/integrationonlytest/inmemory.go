// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package integrationonlytest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/integrationonly], and
// the in-memory subject they are run against.
package integrationonlytest

import (
	"context"
	"errors"
	"strings"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/integrationonly"
)

// ErrBadDSN is what Connect refuses an unparseable target with.
var ErrBadDSN = errors.New("integrationonlytest: malformed dsn")

// InMemory stands in for the external infrastructure the mixin says Connect
// needs, which is the point of having it: the real subject cannot run under the
// default gate, and a fixture that also could not would leave the harness
// unexercised.
type InMemory struct {
	mu        sync.Mutex
	connected bool
}

var _ integrationonly.Mixed = (*InMemory)(nil)

// NewInMemory returns a disconnected subject.
func NewInMemory() *InMemory { return &InMemory{} }

// Connect validates the target and records the connection. It reaches nothing
// outside the process, so the mixin's gate — which the suite does not generate,
// because it is a build tag rather than an assertion — does not apply here.
func (s *InMemory) Connect(ctx context.Context, dsn string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if !strings.Contains(dsn, "://") {
		return ErrBadDSN
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connected = true
	return nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("integrationonlytest: nil context")
	}
	return ctx.Err()
}
