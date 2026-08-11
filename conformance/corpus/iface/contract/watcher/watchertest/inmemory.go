// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package watchertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/watcher], and the
// in-memory subject they are run against.
package watchertest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/watcher"
)

// watcherBuffer is how many changes a watcher's channel holds.
//
// Small enough that a check can fill it: a bound nothing can reach is a report
// nothing proves.
const watcherBuffer = 8

// ErrUnwatchable reports a key nothing can be watched under.
//
// An empty key is the case: a watcher over nothing would receive every change
// or none, and neither is what the caller asked for. Refusing is what lets the
// generated "an error carries the zero value" check run at all — a watcher that
// accepted every key could never be made to fail.
var ErrUnwatchable = errors.New("watchertest: a watch needs a key")

// ErrFull reports a watcher too far behind to take the change.
var ErrFull = errors.New("watchertest: a watcher is too far behind to take the change")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// Watchers are kept per key rather than in one list, because that is the
// contract: `AUTO-WATCHER-RETURNS-ON-CHANGE` is about the key that changed, and
// a subject notifying everybody wakes every caller on every write.
type InMemory struct {
	mu       sync.Mutex
	watchers map[string][]chan watcher.Value
}

var _ watcher.Contract = (*InMemory)(nil)

// NewInMemory returns a store with nothing watched.
func NewInMemory() *InMemory {
	return &InMemory{watchers: map[string][]chan watcher.Value{}}
}

// Watch returns a stream of changes to one key.
//
// The channel is never closed, because nothing here says when watching ends. A
// real implementation would close on the context; this one is driven by a
// harness that builds a fresh subject per check, so the lifetime is the check's.
func (s *InMemory) Watch(ctx context.Context, key string) (<-chan watcher.Value, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if key == "" {
		return nil, ErrUnwatchable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan watcher.Value, watcherBuffer)
	s.watchers[key] = append(s.watchers[key], ch)
	return ch, nil
}

// Trigger records a change and wakes everybody watching that key.
//
// A trigger with nobody watching succeeds and wakes nothing. There is no
// backlog here: a watcher asks about changes from now on, and replaying history
// to a late watcher is `outbox`.
func (s *InMemory) Trigger(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.watchers[key] {
		select {
		case ch <- watcher.Value{Key: key, Body: "changed"}:
		default:
			return ErrFull
		}
	}
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
		return errors.New("watchertest: nil context")
	}
	return ctx.Err()
}
