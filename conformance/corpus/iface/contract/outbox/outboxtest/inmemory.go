// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package outboxtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/outbox], and the
// in-memory subject they are run against.
package outboxtest

import (
	"context"
	"errors"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/outbox"
)

// subscriberBuffer is how many records a subscriber's channel holds.
//
// Small enough that a check can fill it, which is the point: an outbox that
// dropped a record to avoid blocking would stop being an outbox, so the drop
// has to be reported — and a bound nothing can reach is a report nothing
// proves. A conformance run appends a handful, so the ordinary checks never
// come near it.
const subscriberBuffer = 8

// ErrFull reports a subscriber that fell far enough behind to lose a record.
//
// Reported rather than swallowed. An outbox exists so a record survives until
// it is delivered, so a silent drop would be the one failure this whole fixture
// is about — better a loud one than a subject that quietly stops being an
// outbox under load.
var ErrFull = errors.New("outboxtest: a subscriber is too far behind to take the record")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// Records are kept as well as fanned out, which is the contract rather than an
// optimisation: a subscriber attaching after an append still has to receive it,
// so the log is what a late subscriber is replayed from.
type InMemory struct {
	mu          sync.Mutex
	records     []outbox.Value
	subscribers []chan outbox.Value
}

var _ outbox.Contract = (*InMemory)(nil)

// NewInMemory returns an empty outbox.
func NewInMemory() *InMemory { return &InMemory{} }

// Append records a value and hands it to every attached subscriber.
//
// The record is kept whether or not anyone is listening. That is the whole of
// what separates this contract from `publisher`, which carries the same two
// methods and may drop what nobody was waiting for.
func (s *InMemory) Append(ctx context.Context, v outbox.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, v)
	for _, ch := range s.subscribers {
		select {
		case ch <- v:
		default:
			return ErrFull
		}
	}
	return nil
}

// Subscribe returns a stream carrying the backlog and everything appended after
// it.
//
// The backlog is loaded under the same lock the registration takes, so a record
// appended between the two is delivered once rather than twice or not at all.
//
// The channel is never closed, because nothing here says when the outbox is
// done. A subscriber that stops reading leaks a buffer, which is what
// [subscriberBuffer] bounds; a real implementation would take a lifetime and
// close on it.
func (s *InMemory) Subscribe(ctx context.Context) (<-chan outbox.Value, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ch := make(chan outbox.Value, subscriberBuffer)
	for _, v := range s.records {
		select {
		case ch <- v:
		default:
			return nil, ErrFull
		}
	}
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
		return errors.New("outboxtest: nil context")
	}
	return ctx.Err()
}
