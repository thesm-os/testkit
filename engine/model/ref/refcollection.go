// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref

import (
	"context"
	"slices"
	"sync"
)

// Collection is the append-and-drain reference: the store model behind the
// stream mixins, where a writer adds values and a collector returns them all.
//
// It keeps insertion order and every duplicate, deliberately: which of those
// an implementation may collapse or reorder is exactly what the stream laws
// state per mixin, and an oracle that guessed would hide the difference the
// laws exist to catch. A subject that deduplicates diverges from this oracle
// on the second identical add — and whether that divergence is a defect is
// the noduplicates mixin's to say, not this type's.
//
// Thread-safe via mutex, like every reference. An oracle, not production
// code.
type Collection[V any] struct {
	mu   sync.Mutex
	data []V
}

// NewCollection returns an empty collection.
func NewCollection[V any]() *Collection[V] {
	return &Collection[V]{}
}

// Add appends v, keeping order and duplicates.
func (c *Collection[V]) Add(_ context.Context, v V) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = append(c.data, v)
	return nil
}

// Items returns everything added, in insertion order. A copy, so a caller
// ranging it cannot disturb the oracle.
func (c *Collection[V]) Items(_ context.Context) ([]V, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.data), nil
}

// SetCollection is [Collection] under the no-duplicates claim: Add collapses
// repeats, which is the store model a deduplicating subject diverges from a
// plain log on — by design, at the second identical add.
//
// A separate type rather than a mode because the element must be comparable
// to dedupe, a constraint the plain collection does not carry.
type SetCollection[V comparable] struct {
	mu   sync.Mutex
	data []V
	seen map[V]struct{}
}

// NewSetCollection returns an empty deduplicating collection.
func NewSetCollection[V comparable]() *SetCollection[V] {
	return &SetCollection[V]{seen: map[V]struct{}{}}
}

// Add appends v unless an equal value is already held.
func (c *SetCollection[V]) Add(_ context.Context, v V) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, dup := c.seen[v]; dup {
		return nil
	}
	c.seen[v] = struct{}{}
	c.data = append(c.data, v)
	return nil
}

// Items returns everything held, first-insertion order, as a copy.
func (c *SetCollection[V]) Items(_ context.Context) ([]V, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.data), nil
}
