// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package ratelimittest holds the generated harness and double for
// [go.thesmos.sh/testkit/conformance/corpus/iface/contract/rate-limit], and the
// in-memory subject they are run against.
package ratelimittest

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.thesmos.sh/testkit/clock"
	ratelimit "go.thesmos.sh/testkit/conformance/corpus/iface/contract/rate-limit"
)

// ErrLimited reports a call refused because the bucket is empty.
var ErrLimited = errors.New("ratelimittest: rate limit exceeded")

// InMemory is the implementation the generated conformance harness is run
// against.
//
// A token bucket on an injected [clock.Clock], never on the wall clock. A
// limiter that read time.Now would have the machine it runs on as part of its
// behaviour, and a test of it would pass or fail on how loaded the box was —
// which is the whole reason this classification is owned by no tier: stating
// "the rate is enforced" needs controlled time, and a fixed sequence against
// one subject has none.
type InMemory struct {
	mu     sync.Mutex
	clock  clock.Clock
	burst  int
	period time.Duration
	tokens int
	filled time.Time
}

var _ ratelimit.Contract = (*InMemory)(nil)

// NewInMemory returns a limiter starting full, refilling one token per period.
//
// The clock is a parameter rather than a default, because a subject and the
// run measuring it have to share one: build this with the same
// [clock.TestClock] the test advances and "the rate is enforced" becomes a
// claim settled exactly rather than approximately.
func NewInMemory(clk clock.Clock, burst int, period time.Duration) *InMemory {
	return &InMemory{
		clock:  clk,
		burst:  burst,
		period: period,
		tokens: burst,
		filled: clk.Now(),
	}
}

// Run spends a token, or reports that there are none.
func (s *InMemory) Run(ctx context.Context, key string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refill()
	if s.tokens == 0 {
		return ErrLimited
	}
	s.tokens--
	return nil
}

// refill credits whole periods elapsed since the last refill, and carries the
// remainder forward.
//
// The remainder matters: advancing the fill mark to now would discard the part
// of a period already served, so a caller polling faster than the period would
// never accumulate a token at all.
func (s *InMemory) refill() {
	elapsed := s.clock.Now().Sub(s.filled)
	if elapsed < s.period {
		return
	}
	earned := int(elapsed / s.period)
	s.filled = s.filled.Add(time.Duration(earned) * s.period)
	s.tokens = min(s.tokens+earned, s.burst)
}

// contextErr reports a cancelled or expired context, and tolerates a nil one.
//
// Nil is not a legal context and reaches production anyway, through a caller
// that forgot one. Returning an error is a failed request; dereferencing it is
// an outage, which is why the generated check asks only that this does not
// panic.
func contextErr(ctx context.Context) error {
	if ctx == nil {
		return errors.New("ratelimittest: nil context")
	}
	return ctx.Err()
}
