// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package history provides a partition-aware per-iteration trace of
// consumer-attempted appends for chain-shaped model tests. The runner
// resets the trace at iteration boundaries; chain action helpers
// record successful appends keyed by partition; chain laws iterate
// the partitions and cross-check against per-partition replays to
// detect dropped writes.
//
// Non-partitioned chains use K = struct{} — same code path, no
// special-casing.
package history

import (
	"fmt"
	"sort"
	"sync"
)

// IndexedEntry pairs an entry with its global append index and
// partition key. Used by [CausalOrdering] to walk entries in
// global append order rather than partition-name order.
type IndexedEntry[K comparable, Entry any] struct {
	Index   int
	PartKey K
	Entry   Entry
}

// History records entries the consumer asked the action helper to
// append AND that succeeded (helper observed err == nil from both
// SUT and ref). Partition-aware: each Record carries a partition key K.
// Each entry gets a monotonically increasing global index for
// cross-partition causal-ordering checks.
type History[K comparable, Entry any] struct {
	mu      sync.Mutex
	entries map[K][]Entry
	global  []IndexedEntry[K, Entry]
	nextIdx int
}

// New creates an empty History.
func New[K comparable, Entry any]() *History[K, Entry] {
	return &History[K, Entry]{entries: make(map[K][]Entry)}
}

// Record appends an entry to the given partition and the global
// timeline. Thread-safe.
func (h *History[K, Entry]) Record(partKey K, e Entry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries[partKey] = append(h.entries[partKey], e)
	h.global = append(h.global, IndexedEntry[K, Entry]{
		Index: h.nextIdx, PartKey: partKey, Entry: e,
	})
	h.nextIdx++
}

// Reset clears all partitions and the global timeline.
func (h *History[K, Entry]) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	clear(h.entries)
	h.global = h.global[:0]
	h.nextIdx = 0
}

// Snapshot returns a copy of entries for one partition.
func (h *History[K, Entry]) Snapshot(partKey K) []Entry {
	h.mu.Lock()
	defer h.mu.Unlock()
	src := h.entries[partKey]
	cp := make([]Entry, len(src))
	copy(cp, src)
	return cp
}

// SnapshotByPartition returns a defensive copy of all partitions.
func (h *History[K, Entry]) SnapshotByPartition() map[K][]Entry {
	h.mu.Lock()
	defer h.mu.Unlock()
	cp := make(map[K][]Entry, len(h.entries))
	for k, v := range h.entries {
		entries := make([]Entry, len(v))
		copy(entries, v)
		cp[k] = entries
	}
	return cp
}

// GlobalTimeline returns all entries in global append order.
// Used by CausalOrdering for cross-partition dependency checking.
func (h *History[K, Entry]) GlobalTimeline() []IndexedEntry[K, Entry] {
	h.mu.Lock()
	defer h.mu.Unlock()
	cp := make([]IndexedEntry[K, Entry], len(h.global))
	copy(cp, h.global)
	return cp
}

// Partitions returns all partition keys in deterministic order
// (sorted by fmt.Sprint).
func (h *History[K, Entry]) Partitions() []K {
	h.mu.Lock()
	defer h.mu.Unlock()
	keys := make([]K, 0, len(h.entries))
	for k := range h.entries {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j])
	})
	return keys
}

// Len returns the number of entries in one partition.
func (h *History[K, Entry]) Len(partKey K) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.entries[partKey])
}

// TotalLen returns the total number of entries across all partitions.
func (h *History[K, Entry]) TotalLen() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.global)
}
