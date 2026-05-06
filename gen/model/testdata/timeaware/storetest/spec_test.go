// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/gen/model/testdata/timeaware"
	"go.thesmos.sh/testkit/gen/model/testdata/timeaware/storetest"
)

func TestStoreModel(t *testing.T) {
	t.Parallel()

	t.Run("tier 0 with clock factory TTL behavior", func(t *testing.T) {
		t.Parallel()
		// ClockFactory injects a TestClock into both SUT and ref.
		// AdvanceClock actions advance both clocks in sync. When
		// virtual time advances past DefaultTTL (10m), both SUT and
		// ref expire items identically.
		storetest.AssertStoreModel(t,
			func() timeaware.Store {
				return timeaware.NewInMemoryStore(clock.RealClock())
			},
			storetest.StoreModelClockFactory(func(c clock.Clock) timeaware.Store {
				return timeaware.NewInMemoryStore(c)
			}),
			storetest.StoreModelMaxAdvance(15*time.Minute),
		)
	})

	t.Run("catches broken TTL that ignores injected clock", func(t *testing.T) {
		t.Parallel()
		// Negative: BrokenTTLStore uses time.Now() for expiry checks
		// instead of the injected clock. Under TestClock (origin=epoch),
		// time.Now() (2026) is far past epoch+10m, so every Get
		// returns ErrNotFound even immediately after Put. The ref
		// (correct TTL via TestClock) keeps items alive until the
		// TestClock advances past TTL.
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			storetest.AssertStoreModel(ft,
				func() timeaware.Store {
					return timeaware.NewInMemoryStore(clock.RealClock())
				},
				storetest.StoreModelClockFactory(func(c clock.Clock) timeaware.Store {
					return timeaware.NewBrokenTTLStore(c)
				}),
			)
		}()
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			t.Fatal("negative TTL test timed out")
		}
		if !ft.Failed() {
			t.Fatal("framework should have caught broken TTL behavior")
		}
	})

	t.Run("tier 0 without clock factory", func(t *testing.T) {
		t.Parallel()
		// Without ClockFactory: standard model testing, no
		// AdvanceClock action added. TTL never fires because
		// real time doesn't advance 10 minutes during the test.
		storetest.AssertStoreModel(t,
			func() timeaware.Store {
				return timeaware.NewInMemoryStore(clock.RealClock())
			},
		)
	})
}
