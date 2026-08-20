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

	fx := multireadertest.DefaultMultiReaderFixture()

	multireadertest.RunMultiReader(t,
		multireadertest.MultiReaderHarness[*multireadertest.InMemory]{
			Name: "in-memory",
			// MultiReader declares no writer, so the hit path is unreachable
			// without a seeded constructor.
			New: func() *multireadertest.InMemory {
				s := multireadertest.NewInMemory()
				s.Put(
					multireader.Value{Key: fx.Key(), Body: "seeded"},
					multireader.Meta{Revision: 7},
				)
				return s
			},
		},
		multireadertest.MultiReaderChecks{
			{
				Method: "GetWithMeta",
				Name:   "both-slots-for-a-hit",
				Claim:  "GetWithMeta returns both slots for a hit",
				Run: func(tb testing.TB, s multireader.MultiReader, fx multireadertest.MultiReaderFixture) {
					tb.Helper()
					v, m, err := s.GetWithMeta(tb.Context(), fx.Key())
					testkit.NoError(tb, err, "a seeded key is found")
					testkit.Equal(tb, v.Body, "seeded", "the value slot carries what was written")
					testkit.Equal(tb, m.Revision, 7, "and the metadata slot agrees")
				},
			},
		},
	)
}

// A subject zeroing only its first slot must fail, or the generated check reads
// one of two and reports on both.
//
// The check is reached as data rather than by name: the assertion functions are
// unexported now, and Suite is the seam a runner of your own would use.
func TestGetWithMetaZeroesEverySlot(t *testing.T) {
	t.Parallel()

	fx := multireadertest.DefaultMultiReaderFixture()
	want := multireadertest.MultiReaderSuite.Checks.GetWithMeta.ZeroOnError()

	var zeroOnError func(tb testing.TB, s multireader.MultiReader)
	for _, c := range multireadertest.MultiReaderSuite.Suite(fx).Checks {
		if c.ID == want {
			zeroOnError = c.Run
		}
	}
	testkit.True(t, zeroOnError != nil, "the run emits the check this test is about")

	f := testkit.NewFailableTB()
	zeroOnError(f, partialZero{})

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

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestMultiReaderContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	multireadertest.RunMultiReader(
		t,
		multireadertest.MultiReaderHarness[*multireadertest.InMemory]{
			Name: "in-memory",
			New:  multireadertest.NewInMemory,
		},
		multireadertest.MultiReaderSuite.Without(multireadertest.MultiReaderSuite.Checks.GetWithMeta.Smoke()),
	)
}
