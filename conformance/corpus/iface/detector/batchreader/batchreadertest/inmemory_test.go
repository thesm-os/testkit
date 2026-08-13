// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package batchreadertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/batchreader"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/batchreader/batchreadertest"
)

// secondKey is a present key this package names for itself.
//
// The fixture's Keys and KeysOther are a hit and a miss, and the generated
// checks depend on that: taking KeysOther for a second present key would make
// the miss check succeed and assert nothing.
const secondKey = "second-key"

// A variadic read witnesses one key per generated check, which is the narrowing
// the generated check type and fixture field both state.
//
// So everything a *batch* read is for — order, arity, the empty call, all-or-
// nothing on a partial miss — is written here. None of it is a property one
// derived value can reach, and the fixture holds one value per parameter.
func TestBatchReaderContract(t *testing.T) {
	t.Parallel()

	// The values the run itself uses, read rather than replaced: nothing here
	// passes BatchReaderWithFixture, so the derivation stands.
	fixture := batchreadertest.DefaultBatchReaderFixture()

	batchreadertest.AssertBatchReaderContract(t,
		batchreadertest.BatchReaderModel(),
		batchreadertest.BatchReaderSubject("in-memory", func() batchreader.BatchReader {
			return batchreadertest.NewInMemory()
		}),
		batchreadertest.BatchReaderSeed(func(_ context.Context, subject batchreader.BatchReader) error {
			// Keys is seeded and KeysOther deliberately is not. The alternate is
			// the fixture's "a value that should not be found", and the
			// generated miss check calls GetAll with it — so a seed that stored
			// it would make that check succeed and assert nothing.
			//
			// A test needing a second *present* key names its own, below.
			//
			// A seed may reach for the concrete subject: it runs before the
			// double wraps it and sees what the factory made. A check may not.
			s := subject.(*batchreadertest.InMemory)
			s.Put(batchreader.Value{Key: fixture.Keys, Body: "first"})
			s.Put(batchreader.Value{Key: secondKey, Body: "second"})
			return nil
		}),
		batchreadertest.BatchReaderOnGetAll("answers several keys in the order asked", func(
			tb testing.TB, subject batchreader.BatchReader, keys string,
		) {
			tb.Helper()
			got, err := subject.GetAll(tb.Context(), fixture.Keys, secondKey)
			testkit.NoError(tb, err, "a batch of held keys succeeds")
			testkit.Equal(tb, got, []batchreader.Value{
				{Key: fixture.Keys, Body: "first"},
				{Key: secondKey, Body: "second"},
			}, "and comes back in the order it was asked for")

			reversed, err := subject.GetAll(tb.Context(), secondKey, fixture.Keys)
			testkit.NoError(tb, err, "so does the same batch reversed")
			testkit.Equal(tb, reversed[0].Key, secondKey,
				"and the answer follows the question rather than the store")
		}),
		batchreadertest.BatchReaderOnGetAll("returns nothing rather than a partial answer", func(
			tb testing.TB, subject batchreader.BatchReader, keys string,
		) {
			tb.Helper()
			// The failure mode a batch read has and a single read does not: a
			// caller cannot tell a short result from a complete one without
			// comparing lengths, so dropping the absent key silently is worse
			// than failing.
			//
			// The absent key is named here rather than taken from the fixture's
			// alternate. A check runs against every declared subject, and one
			// of them holds both derived keys — so a claim resting on the
			// alternate being absent is a claim about one subject rather than
			// about the shape.
			got, err := subject.GetAll(tb.Context(), fixture.Keys, "held-by-nobody")
			testkit.ErrorIs(tb, err, batchreadertest.ErrNotFound,
				"one absent key fails the batch")
			testkit.True(tb, got == nil, "and nothing is returned beside the error")
		}),
		batchreadertest.BatchReaderOnGetAll("succeeds on the empty call", func(
			tb testing.TB, subject batchreader.BatchReader, keys string,
		) {
			tb.Helper()
			// The call no derivation reaches: a fixture holds one value per
			// parameter, so a generated check always passes exactly one.
			got, err := subject.GetAll(tb.Context())
			testkit.NoError(tb, err, "asking for nothing is not a failure")
			testkit.Len(tb, got, 0, "and answers with nothing")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestBatchReaderContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	batchreadertest.AssertBatchReaderContract(t,
		batchreadertest.BatchReaderSubject("in-memory", func() batchreader.BatchReader {
			return batchreadertest.NewInMemory()
		}),
		batchreadertest.BatchReaderWithout("GetAll/smoke"),
		batchreadertest.BatchReaderWithoutDouble(),
	)
}

// The count claim needs both derived keys present; the miss claims need the
// alternate absent. One fixture cannot be both, so this is a second run rather
// than a second subject.
//
// Which is the option API doing what it exists for: the same statement of the
// contract, against a subject in a state the first run cannot put it in, with
// the checks that contradict that state dropped by name rather than by
// abandoning the harness.
func TestBatchReaderAnswersPerKeyWhenItHoldsThemAll(t *testing.T) {
	t.Parallel()

	fixture := batchreadertest.DefaultBatchReaderFixture()

	batchreadertest.AssertBatchReaderContract(t,
		batchreadertest.BatchReaderSubject("in-memory, holding both keys", func() batchreader.BatchReader {
			s := batchreadertest.NewInMemory()
			s.Put(batchreader.Value{Key: fixture.Keys})
			s.Put(batchreader.Value{Key: fixture.KeysOther})
			return s
		}),
		batchreadertest.BatchReaderWithout(
			"GetAll/reports a miss",
			"GetAll/returns nothing rather than a partial answer",
		),
	)
}
