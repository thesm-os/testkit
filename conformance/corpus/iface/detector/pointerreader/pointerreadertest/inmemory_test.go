// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package pointerreadertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/pointerreader"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/pointerreader/pointerreadertest"
)

// The miss lives in nil, so the whole contract the signature can state is that
// dereferencing never has to happen: two checks, both about not crashing.
//
// This is the shape where the smoke check earns the most. A subject returning a
// pointer into its own map passes every generated check and hands the caller a
// handle on its state — which is the failure this fixture's own check is about.
func TestPointerReaderContract(t *testing.T) {
	t.Parallel()

	// The values the run itself uses, read rather than replaced: nothing here
	// passes PointerReaderWithFixture, so the derivation stands.
	fixture := pointerreadertest.DefaultPointerReaderFixture()

	pointerreadertest.AssertPointerReaderContract(t,
		pointerreadertest.PointerReaderSubject("in-memory", func() pointerreader.PointerReader {
			return pointerreadertest.NewInMemory()
		}),
		pointerreadertest.PointerReaderSeed(func(_ context.Context, subject pointerreader.PointerReader) error {
			// A seed may reach for the concrete subject: it runs before the
			// double wraps it and sees what the factory made. A check may not.
			subject.(*pointerreadertest.InMemory).Put(
				pointerreader.Value{Key: fixture.Key, Body: "seeded"},
			)
			return nil
		}),
		pointerreadertest.PointerReaderOnFind("returns a pointer to what was seeded", func(
			tb testing.TB, subject pointerreader.PointerReader, key string,
		) {
			tb.Helper()
			// Only the hit. That an absent key reads as nil — the only signal
			// this shape has — is the pointerreader classification's own check
			// and is generated, with nil derived as the pointer's zero.
			got := subject.Find(tb.Context(), key)
			testkit.True(tb, got != nil, "a present key reads as a pointer")
			testkit.Equal(tb, got.Body, "seeded", "to what was written")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestPointerReaderContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	pointerreadertest.AssertPointerReaderContract(t,
		pointerreadertest.PointerReaderSubject("in-memory", func() pointerreader.PointerReader {
			return pointerreadertest.NewInMemory()
		}),
		pointerreadertest.PointerReaderWithout("Find/smoke"),
		pointerreadertest.PointerReaderWithoutDouble(),
	)
}
