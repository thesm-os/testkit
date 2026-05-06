// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/model/testdata/leakcheck"
	"go.thesmos.sh/testkit/gen/model/testdata/leakcheck/storetest"
)

// Tests are NOT parallel — goroutine leak detection is process-wide
// and parallel subtests interfere with each other's goroutine counts.
func TestStoreModel(t *testing.T) {
	t.Run("no leak with correct implementation", func(t *testing.T) {
		// InMemoryStore doesn't leak goroutines. With leak check
		// enabled, the test should pass.
		storetest.AssertStoreModel(t,
			func() leakcheck.Store { return leakcheck.NewInMemoryStore() },
			storetest.StoreModelGoroutineLeakCheck(),
		)
	})

	t.Run("catches leaked goroutine", func(t *testing.T) {
		// LeakyStore leaks a goroutine on every Put. The wrapper-level
		// leak check detects goroutines that outlive rapid.Check.
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			storetest.AssertStoreModel(ft,
				func() leakcheck.Store { return leakcheck.NewLeakyStore() },
				storetest.StoreModelGoroutineLeakCheck(),
			)
		}()
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			t.Fatal("leak check test timed out")
		}
		if !ft.Failed() {
			t.Fatal("should have caught goroutine leak from LeakyStore")
		}
	})
}
