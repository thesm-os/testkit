// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package paginationtest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/pagination], and the
// in-memory subject they are run against.
package paginationtest

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/pagination"
)

// PageSize is how many entries one page carries.
//
// Small on purpose: the walk laws are about page boundaries, and a page size
// the seeded store never fills is a paginator that is never seen paginating.
const PageSize = 2

// InMemory is the implementation the generated conformance harness is run
// against.
//
// Entries are served in key order, and the cursor is the last key a page
// emitted: resuming means "strictly after this key", which stays correct when
// entries are inserted between walks — a numeric offset would shift and
// re-emit or skip.
type InMemory struct {
	mu     sync.Mutex
	values map[string]pagination.Value
}

var _ pagination.Contract = (*InMemory)(nil)

// NewInMemory returns an empty store.
func NewInMemory() *InMemory { return &InMemory{values: map[string]pagination.Value{}} }

// Put stores an entry under its own key, replacing what was there.
func (s *InMemory) Put(ctx context.Context, v pagination.Value) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[v.Key] = v
	return nil
}

// Page answers the entries strictly after the cursor, in key order, up to
// PageSize of them.
func (s *InMemory) Page(
	ctx context.Context, cur pagination.Cursor,
) (items []pagination.Value, next pagination.Cursor, more bool, err error) {
	if err := contextErr(ctx); err != nil {
		return nil, "", false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	keys := slices.Sorted(maps.Keys(s.values))
	from, _ := slices.BinarySearch(keys, string(cur))
	if from < len(keys) && keys[from] == string(cur) {
		from++
	}

	for _, k := range keys[from:] {
		if len(items) == PageSize {
			return items, next, true, nil
		}
		items = append(items, s.values[k])
		next = pagination.Cursor(k)
	}
	return items, next, false, nil
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("paginationtest: nil context")
	}
	return ctx.Err()
}
