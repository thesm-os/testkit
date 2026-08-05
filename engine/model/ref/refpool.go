// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// provides the [BalancedPool] reference for the
// Pool composite-tier shape. Every Get balances with a Put;
// double-Put is rejected; the pool tracks lifetime acquire and
// release counts for leak-free verification.

package ref

import (
	"context"
	"sync"
)

// BalancedPool is a pool of R values where the consumer supplies
// the constructor for new resources. Construct with [NewBalancedPool].
// Thread-safe.
type BalancedPool[R any] struct {
	mu          sync.Mutex
	ctor        func() R
	free        []R
	outstanding int
	gets        int
	puts        int
	doublePut   error
	identity    func(R) any
	heldKeys    map[any]struct{}
}

// NewBalancedPool constructs a [BalancedPool].
//
// ctor is the resource constructor. identity returns a
// comparable handle per R (typically a pointer-as-any or a
// stable ID) used to detect double-Put — when nil, double-Put
// detection is disabled. doublePut is returned by Put when the
// resource is not currently outstanding.
func NewBalancedPool[R any](
	ctor func() R,
	identity func(R) any,
	doublePut error,
) *BalancedPool[R] {
	return &BalancedPool[R]{
		ctor:      ctor,
		identity:  identity,
		doublePut: doublePut,
		heldKeys:  make(map[any]struct{}),
	}
}

// Get hands a resource to the caller — reusing a freed one when
// available, otherwise calling ctor().
func (p *BalancedPool[R]) Get(_ context.Context) (R, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gets++
	p.outstanding++
	if n := len(p.free); n > 0 {
		r := p.free[n-1]
		p.free = p.free[:n-1]
		if p.identity != nil {
			p.heldKeys[p.identity(r)] = struct{}{}
		}
		return r, nil
	}
	r := p.ctor()
	if p.identity != nil {
		p.heldKeys[p.identity(r)] = struct{}{}
	}
	return r, nil
}

// Put returns the resource to the pool. Returns the configured
// doublePut error when identity is set and the resource is not
// currently outstanding (double-Put or unknown resource).
func (p *BalancedPool[R]) Put(_ context.Context, r R) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.identity != nil {
		key := p.identity(r)
		if _, held := p.heldKeys[key]; !held {
			return p.doublePut
		}
		delete(p.heldKeys, key)
	}
	p.free = append(p.free, r)
	p.outstanding--
	p.puts++
	return nil
}

// Balanced reports whether Gets and Puts have balanced across
// the pool's lifetime. Used by leak-free verification: a sequence
// of Get→Put cycles ending with Balanced=true demonstrates no
// resource leaks.
func (p *BalancedPool[R]) Balanced() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.outstanding == 0
}

// Stats returns the lifetime acquire and release counts plus the
// current outstanding (held) count. Useful for stats-based law
// assertions.
func (p *BalancedPool[R]) Stats() (gets, puts, outstanding int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.gets, p.puts, p.outstanding
}
