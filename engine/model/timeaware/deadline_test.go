// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package timeaware_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/timeaware"
)

func TestDeadlineRespecting(t *testing.T) {
	t.Parallel()

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
