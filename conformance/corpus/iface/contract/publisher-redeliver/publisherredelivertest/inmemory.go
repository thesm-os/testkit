// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package publisherredelivertest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-redeliver], and the
// in-memory subject they are run against.
package publisherredelivertest

import (
	"context"
	"errors"
	"sync"

	publisherredeliver "go.thesmos.sh/testkit/conformance/corpus/iface/contract/publisher-redeliver"
)

// subscriberBuffer is how many messages a subscriber's channel holds.
//
// Small enough that a check can fill it: a bound nothing can reach is a report
// nothing proves.
const subscriberBuffer = 8

// ErrFull reports a subscriber too far behind to take the message.
var ErrFull = errors.New("publisherredelivertest: a subscriber is too far behind to take the message")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// No backlog, deliberately, like its unarmed siblings: a publisher delivers
// to whoever is listening at the time. What this subject adds is the
// redelivery itself — Republish re-sends without asking whether anyone
// already received the message, which is at-least-once conduct rather than
// an oversight.
type InMemory struct {
	mu          sync.Mutex
	subscribers []chan publisherredeliver.Value
}

var _ publisherredeliver.Contract = (*InMemory)(nil)

// NewInMemory returns a publisher with no subscribers.
func NewInMemory() *InMemory { return &InMemory{} }

// Publish delivers a message to every current subscriber.
//
// A publish with nobody listening succeeds and delivers nothing. That is the
// contract rather than a silent loss: a caller wanting the message held until
// somebody reads it wants `outbox`.
func (s *InMemory) Publish(ctx context.Context, v publisherredeliver.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	return s.deliver(v)
}

// Republish re-offers a message to every current subscriber.
//
// No dedupe, on purpose: at-least-once permits the duplicate, and this
// subject produces it honestly — a subscriber that took the original takes
// the copy too. The exactly-once sibling is the one that suppresses it.
func (s *InMemory) Republish(ctx context.Context, v publisherredeliver.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	return s.deliver(v)
}

// Subscribe attaches a listener for everything published after this call.
//
// The channel is never closed, because nothing here says when the publisher is
// done. A real implementation would take a lifetime and close on it.
func (s *InMemory) Subscribe(ctx context.Context) (<-chan publisherredeliver.Value, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan publisherredeliver.Value, subscriberBuffer)
	s.subscribers = append(s.subscribers, ch)
	return ch, nil
}

// deliver fans one message out to every current subscriber, reporting the
// first one too far behind to take it.
func (s *InMemory) deliver(v publisherredeliver.Value) error {
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

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("publisherredelivertest: nil context")
	}
	return ctx.Err()
}
