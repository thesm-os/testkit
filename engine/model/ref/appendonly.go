// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// provides a generic in-memory append-only chain
// reference implementation for model-based testing. [AppendOnly]
// covers chain-shaped interfaces (audit logs, event streams,
// write-ahead logs); the generator's Tier 0 reference synthesis
// emits one line:
//
//	func() AuditLog { return New[Entry](nil) }

package ref

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

// ChainHash is a 32-byte chain hash (SHA-256).
type ChainHash [32]byte

// ChainHashFunc computes the next chain hash from the previous hash and
// an entry. The default ([DefaultChainHash]) uses SHA-256 over
// gob-encoded (prevHash || entry).
type ChainHashFunc[Entry any] func(prev ChainHash, e Entry) ChainHash

// DefaultChainHash returns a SHA-256 hash function suitable for any
// gob-encodable Entry type. Deterministic across runs.
func DefaultChainHash[Entry any]() ChainHashFunc[Entry] {
	return func(prev ChainHash, e Entry) ChainHash {
		var buf bytes.Buffer
		buf.Write(prev[:])
		enc := gob.NewEncoder(&buf)
		_ = enc.Encode(e)
		return sha256.Sum256(buf.Bytes())
	}
}

// ErrChainIntegrity is returned by [AppendOnly.Verify] when the
// hash chain is broken.
var ErrChainIntegrity = errors.New("ref: chain integrity violation")

// AppendOnly is a generic in-memory append-only chain with hash
// linkage. Thread-safe via mutex.
type AppendOnly[Entry any] struct {
	mu      sync.Mutex
	entries []Entry
	hashes  []ChainHash
	hash    ChainHashFunc[Entry]
}

// NewAppendOnly creates an [AppendOnly] chain. If hash is nil, [DefaultChainHash]
// is used.
func NewAppendOnly[Entry any](hash ChainHashFunc[Entry]) *AppendOnly[Entry] {
	if hash == nil {
		hash = DefaultChainHash[Entry]()
	}
	return &AppendOnly[Entry]{hash: hash}
}

// Append adds an entry to the chain and extends the hash chain.
//
// A cancelled context refuses the write: the oracle stands opposite subjects
// whose lifecycle-shaped methods are held to context respect, and a reference
// that ignored cancellation would fail the very law its adapter is driven
// through.
func (c *AppendOnly[Entry]) Append(ctx context.Context, e Entry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var prev ChainHash
	if len(c.hashes) > 0 {
		prev = c.hashes[len(c.hashes)-1]
	}
	c.entries = append(c.entries, e)
	c.hashes = append(c.hashes, c.hash(prev, e))
	return nil
}

// Replay returns all entries in append order, or yields the context's own
// error where the caller has already given up.
func (c *AppendOnly[Entry]) Replay(ctx context.Context) iter.Seq2[Entry, error] {
	return func(yield func(Entry, error) bool) {
		if err := ctx.Err(); err != nil {
			var zero Entry
			yield(zero, err)
			return
		}
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
// against stored hashes. Returns nil if the chain is intact, and the
// context's own error where the caller has already given up — the shape the
// lifecycle law holds every context-taking operation to.
func (c *AppendOnly[Entry]) Verify(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var prev ChainHash
	for i, e := range c.entries {
		computed := c.hash(prev, e)
		if computed != c.hashes[i] {
			return fmt.Errorf("%w: entry %d hash mismatch", ErrChainIntegrity, i)
		}
		prev = computed
	}
	return nil
}

// Err always reports healthy, and that is the contract rather than a stub.
//
// A reference implementation is correct by construction, so it never latches a
// poisoned state. The method exists so the reference can stand opposite a
// PoisonAccessor-shaped subject in [action.PoisonCheck], which compares the two
// error states: a subject that poisons while the oracle does not is exactly the
// divergence that check exists to surface.
//
// Integrity failures the chain can actually detect are reported by [Verify],
// which recomputes the hash chain rather than reading a latched flag.
func (*AppendOnly[Entry]) Err() error { return nil }

// Len returns the number of entries in the chain.
func (c *AppendOnly[Entry]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
