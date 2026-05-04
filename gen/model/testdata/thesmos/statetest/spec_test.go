// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package statetest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/model/testdata/thesmos"
	"go.thesmos.sh/testkit/gen/model/testdata/thesmos/statetest"
)

func TestStateModel(t *testing.T) {
	t.Parallel()

	t.Run("map SUT vs slice ref", func(t *testing.T) {
		t.Parallel()
		// Different impls, pre-populated so Get/Has have data to read.
		statetest.AssertStateModel(t,
			func() thesmos.State {
				s := thesmos.NewInMemoryState()
				seedState(s)
				return s
			},
			statetest.StateModelReference(func() thesmos.State {
				s := thesmos.NewSliceState()
				seedState(s)
				return s
			}),
		)
	})

	t.Run("catches broken Get", func(t *testing.T) {
		t.Parallel()
		// Negative: BrokenGetState corrupts TurnID on Get.
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			statetest.AssertStateModel(ft,
				func() thesmos.State {
					s := thesmos.NewBrokenGetState()
					seedState(&s.InMemoryState)
					return s
				},
				statetest.StateModelReference(func() thesmos.State {
					s := thesmos.NewSliceState()
					seedState(s)
					return s
				}),
			)
		}()
		<-done
		if !ft.Failed() {
			t.Fatal("framework should have caught corrupted Get TurnID")
		}
	})
}

type seeder interface {
	Put(thesmos.StateKey, thesmos.StateEntry)
}

func seedState(s seeder) {
	keys := []thesmos.StateKey{"a", "b", "c", "d", "e"}
	for i, k := range keys {
		s.Put(k, thesmos.StateEntry{Value: []byte(k), TurnID: i + 1, Region: "us"})
	}
}
