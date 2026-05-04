// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package refchain provides a generic in-memory append-only chain
// reference implementation for model-based testing. [AppendOnly]
// covers chain-shaped interfaces (audit logs, event streams,
// write-ahead logs); the generator's Tier 0 reference synthesis
// emits one line:
//
//	func() AuditLog { return refchain.New[Entry](nil) }
package refchain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/gob"
	"errors"
	"fmt"
	"iter"
	"sync"
)

// Hash is a 32-byte chain hash (SHA-256).
type Hash [32]byte

// HashFunc computes the next chain hash from the previous hash and
// an entry. The default ([DefaultHash]) uses SHA-256 over
// gob-encoded (prevHash || entry).
type HashFunc[Entry any] func(prev Hash, e Entry) Hash

// DefaultHash returns a SHA-256 hash function suitable for any
// gob-encodable Entry type. Deterministic across runs.
func DefaultHash[Entry any]() HashFunc[Entry] {
	return func(prev Hash, e Entry) Hash {
		var buf bytes.Buffer
		buf.Write(prev[:])
		enc := gob.NewEncoder(&buf)
		_ = enc.Encode(e)
		return sha256.Sum256(buf.Bytes())
	}
}

// ErrChainIntegrity is returned by [AppendOnly.Verify] when the
// hash chain is broken.
var ErrChainIntegrity = errors.New("chain integrity violation")

// AppendOnly is a generic in-memory append-only chain with hash
// linkage. Thread-safe via mutex.
type AppendOnly[Entry any] struct {
	mu      sync.Mutex
	entries []Entry
	hashes  []Hash
	hash    HashFunc[Entry]
	err     error
}

// New creates an [AppendOnly] chain. If hash is nil, [DefaultHash]
// is used.
func New[Entry any](hash HashFunc[Entry]) *AppendOnly[Entry] {
	if hash == nil {
		hash = DefaultHash[Entry]()
	}
	return &AppendOnly[Entry]{hash: hash}
}

// Append adds an entry to the chain and extends the hash chain.
func (c *AppendOnly[Entry]) Append(_ context.Context, e Entry) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	var prev Hash
	if len(c.hashes) > 0 {
		prev = c.hashes[len(c.hashes)-1]
	}
	c.entries = append(c.entries, e)
	c.hashes = append(c.hashes, c.hash(prev, e))
	return nil
}

// Replay returns all entries in append order.
func (c *AppendOnly[Entry]) Replay(_ context.Context) iter.Seq2[Entry, error] {
	return func(yield func(Entry, error) bool) {
		c.mu.Lock()
		// Copy under lock, then yield without holding the lock.
		entries := make([]Entry, len(c.entries))
		copy(entries, c.entries)
		c.mu.Unlock()
		for _, e := range entries {
			if !yield(e, nil) {
				return
			}
		}
	}
}

// Verify recomputes the hash chain from scratch and compares
// against stored hashes. Returns nil if the chain is intact.
func (c *AppendOnly[Entry]) Verify(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var prev Hash
	for i, e := range c.entries {
		computed := c.hash(prev, e)
		if computed != c.hashes[i] {
			return fmt.Errorf("%w: entry %d hash mismatch", ErrChainIntegrity, i)
		}
		prev = computed
	}
	return nil
}

// Err returns the poison error, or nil if healthy.
func (c *AppendOnly[Entry]) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// Len returns the number of entries in the chain.
func (c *AppendOnly[Entry]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
