// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package readertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/reader"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/reader/readertest"
)

// The reader shape is the commonest in any store-like interface, and its whole
// contract here comes from the signature: a key in, a value and an error out.
//
// The miss check is the one that needs the pair. It calls Get with KeyOther —
// derived to differ from Key — so a subject seeded under Key still has to
// report a miss, and a subject that returns something for every key fails.
func TestReaderContract(t *testing.T) {
	t.Parallel()

	// The values the run itself uses, read rather than replaced: nothing here
	// passes ReaderWithFixture, so the derivation stands.
	fixture := readertest.DefaultReaderFixture()

	readertest.AssertReaderContract(t,
		readertest.ReaderModel(),
		readertest.ReaderSubject("in-memory", func() reader.Reader {
			return readertest.NewInMemory()
		}),
		readertest.ReaderSeed(func(_ context.Context, subject reader.Reader) error {
			// Reader declares no writer, so nothing is derived and the hit path
			// is unreachable without this. A seed may reach for the concrete
			// subject: it runs before the double wraps it and sees what the
			// factory made. A check may not.
			subject.(*readertest.InMemory).Put(reader.Value{Key: fixture.Key, Body: "seeded"})
			return nil
		}),
		readertest.ReaderOnGet("reports the miss sentinel for a key nothing holds", func(
			tb testing.TB, subject reader.Reader, key string,
		) {
			tb.Helper()
			// Which error a miss reports is the reader shape's own law, and no
			// signature says it. The generated check asks only that the value
			// beside it be the zero.
			_, err := subject.Get(tb.Context(), fixture.KeyOther)
			testkit.ErrorIs(tb, err, reader.ErrNotFound,
				"an absent key is a miss rather than an unlabelled failure")
		}),
		readertest.ReaderOnGet("returns what was seeded", func(
			tb testing.TB, subject reader.Reader, key string,
		) {
			tb.Helper()
			got, err := subject.Get(tb.Context(), key)
			testkit.NoError(tb, err, "a seeded key is found")
			testkit.Equal(tb, got.Body, "seeded", "and carries what was written")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestReaderContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	readertest.AssertReaderContract(t,
		readertest.ReaderSubject("in-memory", func() reader.Reader {
			return readertest.NewInMemory()
		}),
		readertest.ReaderWithout("Get/smoke"),
		readertest.ReaderWithoutDouble(),
	)
}
