// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package timeaware_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/law"
	"go.thesmos.sh/testkit/engine/model/timeaware"
)

func TestDeadlineRespecting(t *testing.T) {
	t.Parallel()

	t.Run("an Op that ignores its deadline is caught", func(t *testing.T) {
		t.Parallel()
		// The implementation the claim exists to catch: it never reads the
		// context, outlives the deadline, and then answers nil as if nothing
		// had happened. Only the error tells it apart from an Op that
		// honoured the deadline — and only the elapsed time tells it apart
		// from an Op that finished before there was a deadline to honour.
		l := timeaware.DeadlineRespecting[struct{}]{
			Op: func(context.Context, struct{}) error {
				time.Sleep(20 * time.Millisecond)
				return nil
			},
			Deadline: 5 * time.Millisecond,
			Advance:  func(time.Duration) {},
			AwaitFor: 500 * time.Millisecond,
		}
		err := l.Check(nil, struct{}{}, struct{}{})
		testkit.False(t, errors.Is(err, law.Vacuous),
			"outliving the deadline engages the claim; this is a verdict, not a refusal")
		testkit.Assert(t, err.Error()).Contains("never saw one",
			"and the verdict names what the subject failed to observe")
	})

	// The complement, and the reason the elapsed-time gate exists: finishing
	// at once is the most complete way to satisfy "within a budget". Failing
	// it told the consumer with the perfect implementation that the tool was
	// broken, which is the one verdict a conformance tool cannot afford.
	t.Run("an Op finishing inside its budget engages nothing", func(t *testing.T) {
		t.Parallel()
		l := timeaware.DeadlineRespecting[struct{}]{
			Op:       func(context.Context, struct{}) error { return nil },
			Deadline: time.Second,
			Advance:  func(time.Duration) {},
			AwaitFor: 50 * time.Millisecond,
		}
		err := l.Check(nil, struct{}{}, struct{}{})
		testkit.True(t, errors.Is(err, law.Vacuous),
			"the deadline never fired, so there was no expiry behaviour to judge")
	})

	t.Run("an Op failing for its own reasons is not a deadline", func(t *testing.T) {
		t.Parallel()
		// Outliving the deadline and then failing for some unrelated reason
		// is not evidence the deadline was respected — the claim names a
		// context error, and a disk fault is not one.
		l := timeaware.DeadlineRespecting[struct{}]{
			Op: func(context.Context, struct{}) error {
				time.Sleep(20 * time.Millisecond)
				return errors.New("disk on fire")
			},
			Deadline: 5 * time.Millisecond,
			Advance:  func(time.Duration) {},
			AwaitFor: 500 * time.Millisecond,
		}
		err := l.Check(nil, struct{}{}, struct{}{})
		testkit.False(t, errors.Is(err, law.Vacuous), "the claim was engaged")
		testkit.Assert(t, err.Error()).Contains("not a context error",
			"the verdict names why the error does not answer the claim")
	})

	t.Run("compliant Op returns when ctx is cancelled", func(t *testing.T) {
		t.Parallel()
		l := timeaware.DeadlineRespecting[struct{}]{
			Op: func(ctx context.Context, _ struct{}) error {
				<-ctx.Done()
				return ctx.Err()
			},
			Deadline: 50 * time.Millisecond,
			Advance:  func(_ time.Duration) {}, // real-time deadline; advance is a no-op
			AwaitFor: 500 * time.Millisecond,
		}
		if err := l.Check(nil, struct{}{}, struct{}{}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Op that ignores the deadline is flagged", func(t *testing.T) {
		t.Parallel()
		var done atomic.Bool
		l := timeaware.DeadlineRespecting[struct{}]{
			Op: func(_ context.Context, _ struct{}) error {
				// Sleep past the deadline plus the AwaitFor window
				// without checking ctx; the law's await timer should
				// fire first.
				time.Sleep(2 * time.Second)
				done.Store(true)
				return nil
			},
			Deadline: 10 * time.Millisecond,
			Advance:  func(_ time.Duration) {},
			AwaitFor: 50 * time.Millisecond,
		}
		err := l.Check(nil, struct{}{}, struct{}{})
		if err == nil {
			t.Fatal("expected deadline violation")
		}
		testkit.Assert(t, err.Error()).Contains("did not return", "diagnostic")
		// Don't wait for the rogue goroutine to finish; let it leak
		// for the duration of the test (rapid suite isolation handles
		// this naturally).
	})

	// AwaitFor ≤ 0 must not mean "wait no time at all", which would flag
	// every compliant Op as ignoring its deadline.
	t.Run("a non-positive await window falls back to the default", func(t *testing.T) {
		t.Parallel()
		l := timeaware.DeadlineRespecting[struct{}]{
			Op: func(ctx context.Context, _ struct{}) error {
				<-ctx.Done()
				return ctx.Err()
			},
			Deadline: 50 * time.Millisecond,
			Advance:  func(time.Duration) {},
		}
		if err := l.Check(nil, struct{}{}, struct{}{}); err != nil {
			t.Fatalf("a compliant Op must pass under the default window: %v", err)
		}
	})

	// The law's identity is load-bearing: the runner keys skips, ran/fired
	// counters and REQ traceability off it.
	t.Run("the law identifies itself", func(t *testing.T) {
		t.Parallel()
		var l timeaware.DeadlineRespecting[struct{}]
		testkit.Equal(t, l.ID(), "AUTO-DEADLINE-RESPECTING", "stable law ID")
		testkit.Equal(t, l.REQID(), "", "auto-derived laws carry no REQ tag")
	})
}
