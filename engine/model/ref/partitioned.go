// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package ref

import (
	"context"
	"fmt"
	"iter"
	"sort"
	"sync"
)

// PartitionedAppendOnly is a per-partition wrapper around [AppendOnly].
// Each partition maintains its own independent hash chain. Used as the
// auto-synthesized reference for chain interfaces with per-partition
// Replay shapes.
type PartitionedAppendOnly[K comparable, Entry any] struct {
	mu     sync.Mutex
	chains map[K]*AppendOnly[Entry]
	hash   ChainHashFunc[Entry]
	keyOf  func(Entry) K
}

// NewPartitionedAppendOnly creates a [PartitionedAppendOnly]. If hash is nil,
// [DefaultChainHash] is used.
func NewPartitionedAppendOnly[K comparable, Entry any](
	keyOf func(Entry) K,
	hash ChainHashFunc[Entry],
) *PartitionedAppendOnly[K, Entry] {
	if hash == nil {
		hash = DefaultChainHash[Entry]()
	}
	return &PartitionedAppendOnly[K, Entry]{
		chains: make(map[K]*AppendOnly[Entry]),
		hash:   hash,
		keyOf:  keyOf,
	}
}

func (p *PartitionedAppendOnly[K, Entry]) getOrCreate(k K) *AppendOnly[Entry] {
	c, ok := p.chains[k]
	if !ok {
		c = NewAppendOnly[Entry](p.hash)
		p.chains[k] = c
	}
	return c
}

// Append adds an entry to the partition determined by keyOf(e).
func (p *PartitionedAppendOnly[K, Entry]) Append(ctx context.Context, e Entry) error {
	p.mu.Lock()
	c := p.getOrCreate(p.keyOf(e))
	p.mu.Unlock()
	return c.Append(ctx, e)
}

// Replay returns entries for a single partition.
func (p *PartitionedAppendOnly[K, Entry]) Replay(ctx context.Context, partKey K) iter.Seq2[Entry, error] {
	p.mu.Lock()
	c, ok := p.chains[partKey]
	p.mu.Unlock()
	if !ok {
		return func(func(Entry, error) bool) {}
	}
	return c.Replay(ctx)
}

// Verify recomputes hash chains across all partitions in deterministic order.
func (p *PartitionedAppendOnly[K, Entry]) Verify(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, k := range p.sortedKeys() {
		err := p.chains[k].Verify(ctx)
		if err != nil {
			return fmt.Errorf("partition %v: %w", k, err)
		}
	}
	return nil
}

// Err always reports healthy, mirroring [AppendOnly.Err]: every partition is a
// reference chain, and a reference is correct by construction. Per-partition
// integrity is reported by [PartitionedAppendOnly.Verify].
func (*PartitionedAppendOnly[K, Entry]) Err() error { return nil }

// sortedKeys returns partition keys in deterministic order. Must be
// called under p.mu.
func (p *PartitionedAppendOnly[K, Entry]) sortedKeys() []K {
	keys := make([]K, 0, len(p.chains))
	for k := range p.chains {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j])
	})
	return keys
}

// Partitions returns all partition keys in deterministic order.
func (p *PartitionedAppendOnly[K, Entry]) Partitions() []K {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sortedKeys()
}
