// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package batchreadertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/batchreader"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/batchreader/batchreadertest"
)

// secondKey is a present key this package names for itself.
//
// The fixture's Keys and KeysOther are a hit and a miss, and the rows below
// depend on that: taking KeysOther for a second present key would make the
// miss row assert nothing.
const secondKey = "second-key"

// A variadic read witnesses one key per generated check, which is the narrowing
// the generated check type and fixture field both state.
//
// So everything a *batch* read is for — order, arity, the empty call, all-or-
// nothing on a partial miss — is written here. None of it is a property one
// derived value can reach, and the fixture holds one value per parameter.
func TestBatchReaderContract(t *testing.T) {
	t.Parallel()

	fx := batchreadertest.DefaultBatchReaderFixture()

	batchreadertest.RunBatchReader(t,
		batchreadertest.BatchReaderHarness[*batchreadertest.InMemory]{
			Name: "in-memory",
			// Keys is seeded and KeysOther deliberately is not, so the
			// all-or-nothing row below has a real absence to fail on.
			New: func() *batchreadertest.InMemory {
				s := batchreadertest.NewInMemory()
				s.Put(batchreader.Value{Key: fx.Keys(), Body: "first"})
				s.Put(batchreader.Value{Key: secondKey, Body: "second"})
				return s
			},
		},
		batchreadertest.BatchReaderChecks{
			{
				Method: "GetAll",
				Name:   "answers-in-the-order-asked",
				Claim:  "GetAll answers several keys in the order asked",
				Run: func(tb testing.TB, s batchreader.BatchReader, fx batchreadertest.BatchReaderFixture) {
					tb.Helper()
					got, err := s.GetAll(tb.Context(), fx.Keys(), secondKey)
					testkit.NoError(tb, err, "a batch of held keys succeeds")
					testkit.Equal(tb, got, []batchreader.Value{
						{Key: fx.Keys(), Body: "first"},
						{Key: secondKey, Body: "second"},
					}, "and comes back in the order it was asked for")

					reversed, err := s.GetAll(tb.Context(), secondKey, fx.Keys())
					testkit.NoError(tb, err, "so does the same batch reversed")
					testkit.Equal(tb, reversed[0].Key, secondKey,
						"and the answer follows the question rather than the store")
				},
			},
			{
				Method: "GetAll",
				Name:   "nothing-rather-than-a-partial-answer",
				Claim:  "GetAll returns nothing rather than a partial answer",
				Run: func(tb testing.TB, s batchreader.BatchReader, fx batchreadertest.BatchReaderFixture) {
					tb.Helper()
					// The failure mode a batch read has and a single read does
					// not: a caller cannot tell a short result from a complete
					// one without comparing lengths, so dropping the absent key
					// silently is worse than failing.
					got, err := s.GetAll(tb.Context(), fx.Keys(), "held-by-nobody")
					testkit.ErrorIs(tb, err, batchreadertest.ErrNotFound,
						"one absent key fails the batch")
					testkit.True(tb, got == nil, "and nothing is returned beside the error")
				},
			},
			{
				Method: "GetAll",
				Name:   "empty-call-succeeds",
				Claim:  "GetAll succeeds on the empty call",
				Run: func(tb testing.TB, s batchreader.BatchReader, fx batchreadertest.BatchReaderFixture) {
					tb.Helper()
					// The call no derivation reaches: a fixture holds one value
					// per parameter, so a generated check always passes exactly
					// one.
					got, err := s.GetAll(tb.Context())
					testkit.NoError(tb, err, "asking for nothing is not a failure")
					testkit.Len(tb, got, 0, "and answers with nothing")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestBatchReaderContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	batchreadertest.RunBatchReader(
		t,
		batchreadertest.BatchReaderHarness[*batchreadertest.InMemory]{
			Name: "in-memory",
			New:  batchreadertest.NewInMemory,
		},
		batchreadertest.BatchReaderSuite.Without(batchreadertest.BatchReaderSuite.Checks.GetAll.Smoke()),
	)
}

// A subject holding BOTH derived keys, which the run above cannot be in: the
// row that asks for a partial miss needs KeysOther absent.
func TestBatchReaderAnswersPerKeyWhenItHoldsThemAll(t *testing.T) {
	t.Parallel()

	fx := batchreadertest.DefaultBatchReaderFixture()

	batchreadertest.RunBatchReader(t,
		batchreadertest.BatchReaderHarness[*batchreadertest.InMemory]{
			Name: "in-memory, holding both keys",
			New: func() *batchreadertest.InMemory {
				s := batchreadertest.NewInMemory()
				s.Put(batchreader.Value{Key: fx.Keys()})
				s.Put(batchreader.Value{Key: fx.KeysOther()})
				return s
			},
		},
	)
}
