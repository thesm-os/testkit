// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package readernoerrortest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/readernoerror"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/readernoerror/readernoerrortest"
)

// A read with no error slot earns two checks, and the missing three are the
// point of the fixture rather than a gap.
//
// Cancellation and an expired deadline are claims about what a method *reports*,
// and this one has nowhere to report them; the zero-value check compares a
// result against an error that does not exist. Only the smoke call and the
// nil-context tolerance survive, and both are about not crashing.
func TestReaderNoErrorContract(t *testing.T) {
	t.Parallel()

	// The values the run itself uses, read rather than replaced: nothing here
	// passes ReaderNoErrorWithFixture, so the derivation stands.
	fixture := readernoerrortest.DefaultReaderNoErrorFixture()

	readernoerrortest.AssertReaderNoErrorContract(t,
		readernoerrortest.ReaderNoErrorSubject("in-memory", func() readernoerror.ReaderNoError {
			return readernoerrortest.NewInMemory()
		}),
		readernoerrortest.ReaderNoErrorSeed(func(_ context.Context, subject readernoerror.ReaderNoError) error {
			// A seed may reach for the concrete subject: it runs before the
			// double wraps it and sees what the factory made. A check may not.
			subject.(*readernoerrortest.InMemory).Put(
				readernoerror.Value{Key: fixture.Key, Body: "seeded"},
			)
			return nil
		}),
		readernoerrortest.ReaderNoErrorOnLookup("reads back what was seeded", func(
			tb testing.TB, subject readernoerror.ReaderNoError, key string,
		) {
			tb.Helper()
			// Only the hit. That an absent key reads as the zero — the whole of
			// what this shape can say about absence — is the readernoerror
			// classification's own check and is generated.
			testkit.Equal(tb, subject.Lookup(tb.Context(), key).Body, "seeded",
				"a present key reads as what was written")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestReaderNoErrorContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	readernoerrortest.AssertReaderNoErrorContract(t,
		readernoerrortest.ReaderNoErrorSubject("in-memory", func() readernoerror.ReaderNoError {
			return readernoerrortest.NewInMemory()
		}),
		readernoerrortest.ReaderNoErrorWithout("Lookup/smoke"),
		readernoerrortest.ReaderNoErrorWithoutDouble(),
	)
}
