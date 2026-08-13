// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package publisherexactlyoncetest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-exactlyonce], and the
// in-memory subject they are run against.
package publisherexactlyoncetest

import (
	"context"
	"errors"
	"sync"

	publisherexactlyonce "go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-exactlyonce"
)

// subscriberBuffer is how many messages a subscriber's channel holds.
//
// Small enough that a check can fill it: a bound nothing can reach is a report
// nothing proves.
const subscriberBuffer = 8

// ErrFull reports a subscriber too far behind to take the message.
var ErrFull = errors.New("publisherexactlyoncetest: a subscriber is too far behind to take the message")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// No backlog, deliberately. `outbox` carries the same two methods and keeps
// every record until somebody reads it; a publisher delivers to whoever is
// listening at the time, and a subject that replayed history to a late
// subscriber would satisfy both contracts and be neither.
type InMemory struct {
	mu          sync.Mutex
	subscribers []chan publisherexactlyonce.Value
}

var _ publisherexactlyonce.Contract = (*InMemory)(nil)

// NewInMemory returns a publisher with no subscribers.
func NewInMemory() *InMemory { return &InMemory{} }

// Publish delivers a message to every current subscriber.
//
// A publish with nobody listening succeeds and delivers nothing. That is the
// contract rather than a silent loss: a caller wanting the message held until
// somebody reads it wants `outbox`.
func (s *InMemory) Publish(ctx context.Context, v publisherexactlyonce.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.subscribers {
		select {
		case ch <- v:
		default:
			return ErrFull
		}
	}
	return nil
}

// Subscribe attaches a listener for everything published after this call.
//
// The channel is never closed, because nothing here says when the publisher is
// done. A real implementation would take a lifetime and close on it.
func (s *InMemory) Subscribe(ctx context.Context) (<-chan publisherexactlyonce.Value, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan publisherexactlyonce.Value, subscriberBuffer)
	s.subscribers = append(s.subscribers, ch)
	return ch, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("publisherexactlyoncetest: nil context")
	}
	return ctx.Err()
}
