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
	"time"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/watcher"
)

// watcherBuffer is how many changes a subscription holds.
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

// subscription is the handle Watch answers: one key's pending changes,
// read with a bounded wait and ended by Stop.
type subscription struct {
	ch   chan watcher.Value
	stop func()
	once sync.Once
}

// Next answers the next change within the wait, or reports that none arrived.
func (s *subscription) Next(timeout time.Duration) (watcher.Value, bool) {
	select {
	case v := <-s.ch:
		return v, true
	case <-time.After(timeout):
		return watcher.Value{}, false
	}
}

// Stop ends the subscription; stopping twice is the deferred double-stop a
// harness performs, so it is safe.
func (s *subscription) Stop() { s.once.Do(s.stop) }

// InMemory is the implementation the generated conformance harness is run
// against.
//
// Watchers are kept per key rather than in one list, because that is the
// contract: `AUTO-WATCHER-RETURNS-ON-CHANGE` is about the key that changed, and
// a subject notifying everybody wakes every caller on every write.
type InMemory struct {
	mu       sync.Mutex
	watchers map[string][]*subscription
}

var _ watcher.Contract = (*InMemory)(nil)

// NewInMemory returns a store with nothing watched.
func NewInMemory() *InMemory {
	return &InMemory{watchers: map[string][]*subscription{}}
}

// Watch returns a subscription to one key's changes.
func (s *InMemory) Watch(ctx context.Context, key string) (watcher.Subscription, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if key == "" {
		return nil, ErrUnwatchable
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sub := &subscription{ch: make(chan watcher.Value, watcherBuffer)}
	sub.stop = func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		held := s.watchers[key]
		for i, candidate := range held {
			if candidate == sub {
				s.watchers[key] = append(held[:i], held[i+1:]...)
				break
			}
		}
	}
	s.watchers[key] = append(s.watchers[key], sub)
	return sub, nil
}

// Trigger records a change and wakes everybody watching that key.
//
// A trigger with nobody watching succeeds and wakes nothing. There is no
// backlog here: a watcher asks about changes from now on, and replaying history
// to a late watcher is `outbox`.
func (s *InMemory) Trigger(ctx context.Context, key string, v watcher.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sub := range s.watchers[key] {
		select {
		case sub.ch <- v:
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
