// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package readernoerrortest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/readernoerror"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/readernoerror/readernoerrortest"
)

// A read with no error slot earns one check, and the missing four are the
// point of the fixture rather than a gap.
//
// Cancellation and an expired deadline are claims about what a method *reports*,
// and this one has nowhere to report them; the zero-value check compares a
// result against an error that does not exist. Nothing on the interface writes
// either, so the miss is refused too — leaving the smoke call, which is about
// not crashing, and the two rows below.
func TestReaderNoErrorContract(t *testing.T) {
	t.Parallel()

	fx := readernoerrortest.DefaultReaderNoErrorFixture()

	readernoerrortest.RunReaderNoError(t,
		readernoerrortest.ReaderNoErrorHarness[*readernoerrortest.InMemory]{
			Name: "in-memory",
			New: func() *readernoerrortest.InMemory {
				s := readernoerrortest.NewInMemory()
				s.Put(readernoerror.Value{Key: fx.Key(), Body: "seeded"})
				return s
			},
		},
		readernoerrortest.ReaderNoErrorChecks{
			{
				Method: "Lookup",
				Name:   "hit-reads-what-was-seeded",
				Claim:  "Lookup reads back what was seeded",
				Run: func(tb testing.TB, s readernoerror.ReaderNoError, fx readernoerrortest.ReaderNoErrorFixture) {
					tb.Helper()
					testkit.Equal(tb, s.Lookup(tb.Context(), fx.Key()).Body, "seeded",
						"a present key reads as what was written")
				},
			},
			{
				Method: "Lookup",
				Name:   "miss-reads-as-the-zero",
				Claim:  "Lookup reads as the zero for a key nothing wrote",
				Run: func(tb testing.TB, s readernoerror.ReaderNoError, fx readernoerrortest.ReaderNoErrorFixture) {
					tb.Helper()
					// The whole of what this shape can say about absence, and
					// a claim only a seeding consumer can make: without a
					// writer the generator cannot tell an unwritten key from
					// any other.
					testkit.Equal(tb, s.Lookup(tb.Context(), fx.KeyOther()), readernoerror.Value{},
						"an absent key reads as the zero rather than as anything held")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestReaderNoErrorContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	readernoerrortest.RunReaderNoError(
		t,
		readernoerrortest.ReaderNoErrorHarness[*readernoerrortest.InMemory]{
			Name: "in-memory",
			New:  readernoerrortest.NewInMemory,
		},
		readernoerrortest.ReaderNoErrorSuite.Without(readernoerrortest.ReaderNoErrorSuite.Checks.Lookup.Smoke()),
	)
}
