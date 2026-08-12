// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package multireadertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/multireader"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/multireader/multireadertest"
)

// A read returning value and metadata owes the zero for both, not for the first.
//
// That is the whole difference from the plain reader shape, and it is invisible
// in a harness that compiles: a check comparing one slot renders identically to
// one comparing two. Only a subject that zeroes some and not others tells them
// apart, which is what TestGetWithMetaZeroesEverySlot is for.
func TestMultiReaderContract(t *testing.T) {
	t.Parallel()

	// The values the run itself uses, read rather than replaced: nothing here
	// passes MultiReaderWithFixture, so the derivation stands.
	fixture := multireadertest.DefaultMultiReaderFixture()

	multireadertest.AssertMultiReaderContract(t,
		multireadertest.MultiReaderModel(),
		multireadertest.MultiReaderSubject("in-memory", func() multireader.MultiReader {
			return multireadertest.NewInMemory()
		}),
		multireadertest.MultiReaderSeed(func(_ context.Context, subject multireader.MultiReader) error {
			// MultiReader declares no writer, so nothing is derived and the hit
			// path is unreachable without this. A seed may reach for the
			// concrete subject: it runs before the double wraps it. A check may
			// not.
			subject.(*multireadertest.InMemory).Put(
				multireader.Value{Key: fixture.Key, Body: "seeded"},
				multireader.Meta{Revision: 7},
			)
			return nil
		}),
		multireadertest.MultiReaderOnGetWithMeta("returns both slots for a hit", func(
			tb testing.TB, subject multireader.MultiReader, key string,
		) {
			tb.Helper()
			v, m, err := subject.GetWithMeta(tb.Context(), key)
			testkit.NoError(tb, err, "a seeded key is found")
			testkit.Equal(tb, v.Body, "seeded", "the value slot carries what was written")
			testkit.Equal(tb, m.Revision, 7, "and the metadata slot agrees")
		}),
	)
}

// A subject zeroing only its first slot must fail, or the generated check reads
// one of two and reports on both.
func TestGetWithMetaZeroesEverySlot(t *testing.T) {
	t.Parallel()

	f := testkit.NewFailableTB()
	multireadertest.AssertMultiReaderGetWithMetaZeroOnError(f, partialZero{}, "absent")

	testkit.True(t, f.Failed(),
		"metadata leaked beside an error must fail, whichever slot it is")
}

// partialZero reports a miss and leaks its metadata, which is the one violation
// a single-slot check cannot see.
type partialZero struct{}

func (partialZero) GetWithMeta(
	context.Context, string,
) (multireader.Value, multireader.Meta, error) {
	return multireader.Value{}, multireader.Meta{Revision: 9}, multireader.ErrNotFound
}

// Declining the double is separate from dropping a check.
func TestMultiReaderContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	multireadertest.AssertMultiReaderContract(t,
		multireadertest.MultiReaderSubject("in-memory", func() multireader.MultiReader {
			return multireadertest.NewInMemory()
		}),
		multireadertest.MultiReaderWithout("GetWithMeta/smoke"),
		multireadertest.MultiReaderWithoutDouble(),
	)
}
