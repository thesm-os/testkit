// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref

import (
	"context"
	"sync"
)

// KeyedStore is the keyed-put counterpart of [MapStore]: the reference for
// interfaces whose writer takes the key beside the value — Put(ctx, k, v) —
// rather than a value that carries its own key.
//
// The two models cannot share an implementation surface: MapStore's Put
// derives the key from the value through a projection, and a keyed put has no
// projection to derive — the key is an argument. Everything else mirrors
// MapStore, method for method, so a generated adapter forwards a shape's
// parameters in order and changes nothing but the name.
//
// Thread-safe via mutex, like every reference: the concurrent checkers drive
// them from many goroutines. An oracle, not production code.
type KeyedStore[K comparable, V any] struct {
	mu       sync.Mutex
	data     map[K]V
	notFound error
}

// NewKeyedStore returns an empty store reporting notFound for a missing key.
func NewKeyedStore[K comparable, V any](notFound error) *KeyedStore[K, V] {
	return &KeyedStore[K, V]{data: map[K]V{}, notFound: notFound}
}

// Put stores v under k, replacing what was there.
func (s *KeyedStore[K, V]) Put(_ context.Context, k K, v V) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[k] = v
	return nil
}

// Get returns the value under k, or the not-found sentinel.
func (s *KeyedStore[K, V]) Get(_ context.Context, k K) (V, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[k]
	if !ok {
		var zero V
		return zero, s.notFound
	}
	return v, nil
}

// Delete removes k. Deleting an absent key succeeds: the postcondition — the
// key is gone — already holds, and a subject is held to the same reading by
// the delete laws.
func (s *KeyedStore[K, V]) Delete(_ context.Context, k K) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, k)
	return nil
}

// Count reports how many keys hold values.
func (s *KeyedStore[K, V]) Count(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data), nil
}
