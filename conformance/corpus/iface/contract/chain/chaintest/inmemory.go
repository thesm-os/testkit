// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package chaintest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/chain], and the
// in-memory subject they are run against — scaffolding for the run, so it lives
// beside the harness rather than in the package stating the shape.
package chaintest

import (
	"context"
	"errors"
	"slices"
	"sync"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/chain"
)

// ErrBroken is what Verify reports for a chain whose links do not compute.
var ErrBroken = errors.New("chaintest: chain digest does not match")

// BreakKey is the lever a check pulls to reach the broken state.
//
// Through a correct interface a chain cannot break: Append updates the entry
// list and the digest together, so Verify's failure arm is unreachable and
// would be dead code. Rather than delete the arm — it is the whole point of
// the verify role — appending under this key records the entry and leaves the
// digest behind, which is exactly the divergence a tampered log has.
const BreakKey = "chaintest-break"

// InMemory is the implementation the generated conformance harness is run
// against.
type InMemory struct {
	mu      sync.Mutex
	entries []chain.Entry
	digest  string
}

var _ chain.Contract = (*InMemory)(nil)

// NewInMemory returns an empty chain.
func NewInMemory() *InMemory { return &InMemory{} }

// Append adds one entry and extends the digest that links it to the last.
func (s *InMemory) Append(ctx context.Context, e chain.Entry) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, e)
	if e.Key != BreakKey {
		s.digest = link(s.digest, e)
	}
	return nil
}

// Replay reports the entries in append order.
//
// A copy rather than the slice itself: a replay walked while the log is still
// growing would otherwise observe an append mid-walk.
func (s *InMemory) Replay(ctx context.Context) ([]chain.Entry, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.entries), nil
}

// Verify recomputes the chain from its entries and compares.
func (s *InMemory) Verify(ctx context.Context) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var want string
	for _, e := range s.entries {
		want = link(want, e)
	}
	if want != s.digest {
		return ErrBroken
	}
	return nil
}

// link folds one entry into the running digest. A stand-in for a real hash:
// the fixture states the shape of the contract, and a cryptographic digest
// would say nothing more about it.
func link(prev string, e chain.Entry) string {
	return prev + "|" + e.Key + "=" + e.Body
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("chaintest: nil context")
	}
	return ctx.Err()
}
