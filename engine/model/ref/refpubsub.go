// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// provides three reference implementations of
// the Publisher contract paired with the delivery mixin:
//
//   - [AtLeastOnce]: every subscriber receives every message, but
//     may receive duplicates after a redelivery.
//   - [AtMostOnce]: every subscriber receives a message at most
//     once; the broker drops on subscriber-side back-pressure.
//   - [ExactlyOnce]: each message reaches each subscriber exactly
//     once via per-subscriber dedup IDs.

package ref

import (
	"context"
	"sync"
)

// AtLeastOnce delivers every published message to every subscriber
// at least once. Construct with [NewAtLeastOnce]. Thread-safe.
type AtLeastOnce[T any] struct {
	mu          sync.Mutex
	subscribers [][]T
}

// NewAtLeastOnce constructs an empty broker.
func NewAtLeastOnce[T any]() *AtLeastOnce[T] {
	return &AtLeastOnce[T]{}
}

// Subscribe registers a new subscriber and returns its index.
func (b *AtLeastOnce[T]) Subscribe(_ context.Context) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	idx := len(b.subscribers)
	b.subscribers = append(b.subscribers, nil)
	return idx, nil
}

// Publish appends the message to every subscriber's queue.
func (b *AtLeastOnce[T]) Publish(_ context.Context, msg T) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.subscribers {
		b.subscribers[i] = append(b.subscribers[i], msg)
	}
	return nil
}

// Drain returns every message queued for the named subscriber and
// clears the queue. At-least-once: the consumer may re-Publish a
// drained message to model redelivery.
func (b *AtLeastOnce[T]) Drain(_ context.Context, sub int) ([]T, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := b.subscribers[sub]
	b.subscribers[sub] = nil
	return out, nil
}

// AtMostOnce delivers each published message to each subscriber
// at most once. The broker has a per-subscriber buffer; when full,
// further publications are dropped silently (the at-most-once
// guarantee allows loss but forbids duplicates).
type AtMostOnce[T any] struct {
	mu          sync.Mutex
	capacity    int
	subscribers [][]T
	dropped     []int
}

// NewAtMostOnce constructs a broker with the given per-subscriber
// buffer capacity. capacity < 0 is treated as unlimited.
func NewAtMostOnce[T any](capacity int) *AtMostOnce[T] {
	return &AtMostOnce[T]{capacity: capacity}
}

// Subscribe registers a new subscriber.
func (b *AtMostOnce[T]) Subscribe(_ context.Context) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	idx := len(b.subscribers)
	b.subscribers = append(b.subscribers, nil)
	b.dropped = append(b.dropped, 0)
	return idx, nil
}

// Publish appends msg to each subscriber whose buffer isn't full.
// Subscribers at capacity record a drop instead.
func (b *AtMostOnce[T]) Publish(_ context.Context, msg T) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.subscribers {
		if b.capacity >= 0 && len(b.subscribers[i]) >= b.capacity {
			b.dropped[i]++
			continue
		}
		b.subscribers[i] = append(b.subscribers[i], msg)
	}
	return nil
}

// Drain returns the buffered messages for the named subscriber
// plus the count of messages dropped due to back-pressure since
// the last Drain.
func (b *AtMostOnce[T]) Drain(_ context.Context, sub int) ([]T, int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := b.subscribers[sub]
	dropped := b.dropped[sub]
	b.subscribers[sub] = nil
	b.dropped[sub] = 0
	return out, dropped, nil
}

// ExactlyOnce delivers each (publisher, message-id) pair to each
// subscriber exactly once. Construct with [NewExactlyOnce]. The
// broker assigns a monotonic ID per Publish; subscribers see each
// ID at most once even on republish.
type ExactlyOnce[T any] struct {
	mu          sync.Mutex
	next        int64
	subscribers []*onceSubscriber[T]
}

type onceSubscriber[T any] struct {
	seen   map[int64]struct{}
	queued []T
}

// NewExactlyOnce constructs an empty broker.
func NewExactlyOnce[T any]() *ExactlyOnce[T] {
	return &ExactlyOnce[T]{}
}

// Subscribe registers a new subscriber.
func (b *ExactlyOnce[T]) Subscribe(_ context.Context) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	idx := len(b.subscribers)
	b.subscribers = append(b.subscribers, &onceSubscriber[T]{
		seen: make(map[int64]struct{}),
	})
	return idx, nil
}

// Publish enqueues msg under a fresh monotonic ID. Returns the ID
// so the consumer can replay it later via [Replay] without causing
// duplicate delivery.
func (b *ExactlyOnce[T]) Publish(_ context.Context, msg T) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.next
	b.next++
	for _, sub := range b.subscribers {
		sub.seen[id] = struct{}{}
		sub.queued = append(sub.queued, msg)
	}
	return id, nil
}

// Replay re-enqueues an already-published ID. Subscribers that
// have seen the ID drop the redelivery silently — exactly-once.
func (b *ExactlyOnce[T]) Replay(_ context.Context, id int64, msg T) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, sub := range b.subscribers {
		if _, dup := sub.seen[id]; dup {
			continue
		}
		sub.seen[id] = struct{}{}
		sub.queued = append(sub.queued, msg)
	}
	return nil
}

// Drain returns and clears the named subscriber's queue.
func (b *ExactlyOnce[T]) Drain(_ context.Context, sub int) ([]T, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := b.subscribers[sub].queued
	b.subscribers[sub].queued = nil
	return out, nil
}
