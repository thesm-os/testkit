// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package streamreflectsmutationstest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/streamreflectsmutations"
	sm "go.thesmos.sh/testkit/conformance/corpus/iface/mixin/streamreflectsmutations/streamreflectsmutationstest"
)

// streamreflectsmutations is the model tier's — AUTO-STREAM-REFLECTS-MUTATIONS
// states it — so the suite generates the signature family alone, even though
// eidos now lets the mixin name its mutator through `mutate=Add`.
//
// Stream returns a function, so what the signature can promise ends at the
// call: two checks, both about not crashing. Everything the mixin is about
// happens while someone is mid-range.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	sm.AssertMixedContract(t,
		sm.MixedSubject("in-memory", func() streamreflectsmutations.Mixed {
			return sm.NewInMemory()
		}),
	)
}

// An item added mid-range is yielded to the consumer already ranging. A subject
// snapshotting at the call satisfies every generated check and misses it.
func TestStreamYieldsAnItemAddedMidRange(t *testing.T) {
	t.Parallel()

	s := sm.NewInMemory()
	ctx := t.Context()
	testkit.NoError(t, s.Add(ctx, "first"), "seeding succeeds")

	var seen []string
	for item, err := range s.Stream(ctx) {
		testkit.NoError(t, err, "a healthy stream yields no per-element error")
		seen = append(seen, item)
		if len(seen) == 1 {
			// The mutation the mixin is named for, from inside the range —
			// which is also why Stream must not hold its lock across the yield.
			testkit.NoError(t, s.Add(ctx, "second"), "adding mid-range succeeds")
		}
	}

	testkit.Equal(t, seen, []string{"first", "second"},
		"the stream reflects what was added while it was being read")
}

// A consumer may stop early, so the implementation must not assume the sequence
// is drained.
func TestStreamStopsWhenTheConsumerDoes(t *testing.T) {
	t.Parallel()

	s := sm.NewInMemory()
	ctx := t.Context()
	testkit.NoError(t, s.Add(ctx, "first"), "seeding succeeds")
	testkit.NoError(t, s.Add(ctx, "second"), "and again")

	var seen int
	for range s.Stream(ctx) {
		seen++
		break
	}
	testkit.Equal(t, seen, 1, "the range stopped after one element")
}

// A cancelled context is reported on the first yield, which is the only place
// a per-element error can carry it — a caller who never ranges never sees it,
// and that is the shape rather than a defect.
func TestStreamReportsACancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var seen int
	for _, err := range sm.NewInMemory().Stream(ctx) {
		seen++
		testkit.ErrorIs(t, err, context.Canceled,
			"a cancelled stream says so through its element error")
	}
	testkit.Equal(t, seen, 1, "and yields nothing else")
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	sm.AssertMixedContract(t,
		sm.MixedSubject("in-memory", func() streamreflectsmutations.Mixed {
			return sm.NewInMemory()
		}),
		sm.MixedWithout("Add/smoke"),
		sm.MixedWithoutDouble(),
	)
}
