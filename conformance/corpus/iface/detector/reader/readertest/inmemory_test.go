// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package readertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/reader"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/reader/readertest"
)

// Reader declares no writer, so nothing is derived to seed through and the
// hit path is unreachable without a seeded constructor. Which error a miss
// reports is the reader shape's own law and no signature says it, so the
// generated check asks only that the value beside it be the zero; the row
// below is what pins the sentinel.
func TestReaderContract(t *testing.T) {
	t.Parallel()

	// The values the run itself uses, read rather than replaced.
	fx := readertest.DefaultReaderFixture()

	readertest.RunReader(t,
		readertest.ReaderHarness[*readertest.InMemory]{
			Name: "in-memory",
			// The seed folded into the constructor, which is where a
			// seeded subject is built now: a factory may make any
			// starting state, and it runs before anything wraps it.
			New: func() *readertest.InMemory {
				s := readertest.NewInMemory()
				s.Put(reader.Value{Key: fx.Key(), Body: "seeded"})
				return s
			},
		},
		// Get/miss is generated and asserts the sentinel, because the read
		// declares one. The hand-written row that used to pin it is gone
		// with the drop that used to excuse it.
		readertest.ReaderChecks{
			{
				Method: "Get",
				Name:   "hit-returns-seeded",
				Claim:  "Get returns what was seeded under a key that was written",
				Run: func(tb testing.TB, s reader.Reader, fx readertest.ReaderFixture) {
					tb.Helper()
					got, err := s.Get(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "a seeded key is found")
					testkit.Equal(tb, got.Body, "seeded", "and carries what was written")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string,
// so a check that is renamed or stops being emitted breaks this compile
// instead of silently declining nothing.
func TestReaderContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	readertest.RunReader(t,
		readertest.ReaderHarness[*readertest.InMemory]{
			Name: "in-memory",
			New:  readertest.NewInMemory,
		},
		readertest.ReaderSuite.Without(readertest.ReaderSuite.Checks.Get.Smoke()),
	)
}
