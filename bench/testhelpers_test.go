// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"context"
	"errors"
	"iter"
)

// Test fixtures for bench tests. These are local to package bench_test
// and mirror the fixtures in suite_test. Each shape needs a minimal
// concrete type to exercise its bench primitives.

var errNotFound = errors.New("not found")

// --- Reader fixture ---

type mapReader struct {
	data map[string]string
}

func newMapReader(data map[string]string) *mapReader {
	return &mapReader{data: data}
}

func (r *mapReader) Get(_ context.Context, key string) (string, error) {
	v, ok := r.data[key]
	if !ok {
		return "", errNotFound
	}
	return v, nil
}

// --- Writer fixture ---

type mapWriter struct {
	data map[string]string
}

func newMapWriter() *mapWriter {
	return &mapWriter{data: make(map[string]string)}
}

type entry struct {
	Key   string
	Value string
}

func (w *mapWriter) Put(_ context.Context, e entry) error {
	w.data[e.Key] = e.Value
	return nil
}

// --- Deleter fixture ---

type delStore struct {
	data map[string]bool
}

func newDelStore() *delStore {
	return &delStore{data: map[string]bool{"existing": true}}
}

func (s *delStore) Delete(_ context.Context, key string) error {
	if !s.data[key] {
		return errNotFound
	}
	delete(s.data, key)
	return nil
}

// --- Aggregator fixture ---

type itemCounter struct{ n int }

func newItemCounter(n int) *itemCounter { return &itemCounter{n: n} }

func (c *itemCounter) Count(_ context.Context) (int, error) { return c.n, nil }

// --- Lifecycle fixture ---

type lifecycle struct{ opened bool }

func newLifecycle() *lifecycle { return &lifecycle{} }

func (l *lifecycle) Open(_ context.Context) error {
	l.opened = true
	return nil
}

// --- Pure fixture ---

type counter struct{ n int }

func newCounter() *counter { return &counter{n: 42} }

func (c *counter) Value() int { return c.n }

// --- Predicate fixture ---

type validator struct{ valid bool }

func newValidator(v bool) *validator { return &validator{valid: v} }

func (v *validator) IsValid() bool { return v.valid }

// --- Stream fixture ---

type listStore struct {
	items []string
}

func (s *listStore) List() iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		for _, item := range s.items {
			if !yield(item, nil) {
				return
			}
		}
	}
}
