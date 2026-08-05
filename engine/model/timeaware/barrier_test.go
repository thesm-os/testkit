// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package timeaware_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model/timeaware"
)

func TestBarrier(t *testing.T) {
	t.Parallel()

	t.Run("multiple Ops run concurrently", func(t *testing.T) {
		t.Parallel()
		b := timeaware.NewBarrier()
		var inFlight atomic.Int32
		var peakConcurrency atomic.Int32
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				release := b.Op()
				defer release()
				cur := inFlight.Add(1)
				if peakConcurrency.Load() < cur {
					peakConcurrency.Store(cur)
				}
				time.Sleep(time.Millisecond)
				inFlight.Add(-1)
			})
		}
		wg.Wait()
		testkit.True(t, peakConcurrency.Load() > 1, "more than one op observed concurrently")
	})

	t.Run("Advance blocks until in-flight Ops release", func(t *testing.T) {
		t.Parallel()
		b := timeaware.NewBarrier()
		release := b.Op()
		advanced := make(chan struct{})
		var observedOpStillHeld atomic.Bool
		go func() {
			b.Advance(func() {
				// When Advance runs, the read-lock must already be free.
				observedOpStillHeld.Store(false)
				close(advanced)
			})
		}()
		// Hold the op for a beat to confirm Advance blocks.
		time.Sleep(5 * time.Millisecond)
		select {
		case <-advanced:
			t.Fatal("Advance ran while Op was still in-flight")
		default:
		}
		observedOpStillHeld.Store(true)
		release()
		select {
		case <-advanced:
			// expected: Advance ran after release
		case <-time.After(time.Second):
			t.Fatal("Advance did not run within 1s of Op release")
		}
	})

	t.Run("nil advance is a synchronization fence", func(t *testing.T) {
		t.Parallel()
		b := timeaware.NewBarrier()
		release := b.Op()
		done := make(chan struct{})
		go func() {
			b.Advance(nil)
			close(done)
		}()
		time.Sleep(2 * time.Millisecond)
		select {
		case <-done:
			t.Fatal("Advance(nil) returned while Op was in-flight")
		default:
		}
		release()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("Advance(nil) did not complete after Op release")
		}
	})

	t.Run("release is idempotent", func(t *testing.T) {
		t.Parallel()
		b := timeaware.NewBarrier()
		release := b.Op()
		release()
		release() // second call must not panic or double-unlock
		// If we reached here, the release was idempotent.
		// Verify the barrier is reusable.
		b.Advance(nil)
	})
}
