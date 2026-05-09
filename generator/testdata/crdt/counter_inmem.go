// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package crdt

import (
	"context"
	"sync"
)

// additiveCounter is the AdditiveCounter companion. State is one
// integer + a mutex for goroutine safety. Merge sums; Value reads.
//
// Reflective state-equality (the CRDT-merge contract's default
// comparator falls back to reflect.DeepEqual) compares the embedded
// sum across two instances; mutexes in their idle state compare
// equal under DeepEqual.
type additiveCounter struct {
	mu  sync.Mutex
	sum int
}

// NewAdditiveCounter returns a fresh counter at zero.
func NewAdditiveCounter() *additiveCounter {
	return &additiveCounter{}
}

// Merge implements [AdditiveCounter].
func (c *additiveCounter) Merge(ctx context.Context, n int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sum += n
	return nil
}

// Value implements [AdditiveCounter].
func (c *additiveCounter) Value(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sum, nil
}
