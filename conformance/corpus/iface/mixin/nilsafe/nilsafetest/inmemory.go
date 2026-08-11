// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package nilsafetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/mixin/nilsafe], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package nilsafetest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/nilsafe"
)

// ErrNilPayload is what Store refuses a nil pointer with.
//
// The whole content of the mixin: refusing is a failed request, and
// dereferencing is an outage. A subject that returned nil here would pass every
// check except the one that matters.
var ErrNilPayload = errors.New("nilsafetest: nil payload")

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu     sync.Mutex
	values map[string]nilsafe.Payload
}

var _ nilsafe.Mixed = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory {
	return &InMemory{values: map[string]nilsafe.Payload{}}
}

// Store takes a pointer, which is what makes nil expressible, and reports it
// rather than dereferencing it.
func (s *InMemory) Store(ctx context.Context, v *nilsafe.Payload) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if v == nil {
		return ErrNilPayload
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = *v
	return nil
}

// Stored reports what was written under a key, which the interface exposes no
// way to observe.
func (s *InMemory) Stored(key string) (nilsafe.Payload, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	return v, ok
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("nilsafetest: nil context")
	}
	return ctx.Err()
}
