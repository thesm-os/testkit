// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package pointerreadertest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/pointerreader"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/pointerreader/pointerreadertest"
)

// The miss lives in nil, so the whole contract the signature can state is that
// dereferencing never has to happen: one generated check, about not crashing.
//
// This is the shape where the smoke check earns the most. A subject returning a
// pointer into its own map passes every generated check and hands the caller a
// handle on its state — which is the failure this fixture's own check is about.
func TestPointerReaderContract(t *testing.T) {
	t.Parallel()

	fx := pointerreadertest.DefaultPointerReaderFixture()

	pointerreadertest.RunPointerReader(t,
		pointerreadertest.PointerReaderHarness[*pointerreadertest.InMemory]{
			Name: "in-memory",
			New: func() *pointerreadertest.InMemory {
				s := pointerreadertest.NewInMemory()
				s.Put(pointerreader.Value{Key: fx.Key(), Body: "seeded"})
				return s
			},
		},
		pointerreadertest.PointerReaderChecks{
			{
				Method: "Find",
				Name:   "hit-returns-a-pointer",
				Claim:  "Find returns a pointer to what was seeded",
				Run: func(tb testing.TB, s pointerreader.PointerReader, fx pointerreadertest.PointerReaderFixture) {
					tb.Helper()
					got := s.Find(tb.Context(), fx.Key())
					testkit.True(tb, got != nil, "a present key reads as a pointer")
					testkit.Equal(tb, got.Body, "seeded", "to what was written")
				},
			},
			{
				Method: "Find",
				Name:   "miss-reads-as-nil",
				Claim:  "Find reads as nil for a key nothing wrote",
				Run: func(tb testing.TB, s pointerreader.PointerReader, fx pointerreadertest.PointerReaderFixture) {
					tb.Helper()
					// The only signal this shape has. Nothing on the interface
					// writes, so the generator refuses the miss and this is
					// where it is stated.
					testkit.True(tb, s.Find(tb.Context(), fx.KeyOther()) == nil,
						"an absent key reads as nil rather than as a zero value")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestPointerReaderContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	pointerreadertest.RunPointerReader(
		t,
		pointerreadertest.PointerReaderHarness[*pointerreadertest.InMemory]{
			Name: "in-memory",
			New:  pointerreadertest.NewInMemory,
		},
		pointerreadertest.PointerReaderSuite.Without(pointerreadertest.PointerReaderSuite.Checks.Find.Smoke()),
	)
}
