// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package multireturntest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/multireturn"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/multireturn/multireturntest"
)

// A method returning several values beside its error owes the zero for each.
func TestWideContract(t *testing.T) {
	t.Parallel()

	fx := multireturntest.DefaultWideFixture()

	multireturntest.RunWide(t,
		multireturntest.WideHarness[*multireturntest.InMemory]{
			Name: "in-memory",
			New: func() *multireturntest.InMemory {
				s := multireturntest.NewInMemory()
				s.Put(fx.ID(), "found")
				return s
			},
		},
		multireturntest.WideChecks{
			{
				Method: "Quad",
				Name:   "every-slot-for-a-hit",
				Claim:  "Quad returns every slot for a hit",
				Run: func(tb testing.TB, s multireturn.Wide, fx multireturntest.WideFixture) {
					tb.Helper()
					v, n, ok, err := s.Quad(tb.Context(), fx.ID())
					testkit.NoError(tb, err, "a seeded identifier is found")
					testkit.Equal(tb, v, "found", "the value slot carries it")
					testkit.Equal(tb, n, len("found"), "the derived slot agrees")
					testkit.True(tb, ok, "and the flag says so")
				},
			},
			{
				Method: "Triple",
				Name:   "absence-through-the-flag",
				Claim:  "Triple reports absence through its flag",
				Run: func(tb testing.TB, s multireturn.Wide, fx multireturntest.WideFixture) {
					tb.Helper()
					_, _, ok := s.Triple(tb.Context(), fx.IDOther())
					testkit.False(tb, ok, "a method with no error slot says so through the flag")
				},
			},
			{
				Method: "NoError",
				Name:   "zero-for-an-absent-identifier",
				Claim:  "NoError returns the zero for an absent identifier",
				Run: func(tb testing.TB, s multireturn.Wide, fx multireturntest.WideFixture) {
					tb.Helper()
					v, n := s.NoError(tb.Context(), fx.IDOther())
					testkit.Equal(tb, v, "", "the value slot is zero")
					testkit.Equal(tb, n, 0, "and so is the derived one")
				},
			},
		},
	)
}

// A subject zeroing only its first slot must fail, or the generated check is
// reading one of three and reporting on all of them.
//
// This is what running the fixture adds over compiling it: a template emitting
// one comparison instead of three compiles identically, and only a subject that
// zeroes some slots and not others can tell the two apart. The check is reached
// as data, since the assertion functions are unexported.
func TestQuadZeroOnErrorReadsEverySlot(t *testing.T) {
	t.Parallel()

	fx := multireturntest.DefaultWideFixture()
	want := multireturntest.WideSuite.Checks.Quad.ZeroOnError()

	var zeroOnError func(tb testing.TB, s multireturn.Wide)
	for _, c := range multireturntest.WideSuite.Suite(fx).Checks {
		if c.ID == want {
			zeroOnError = c.Run
		}
	}
	testkit.True(t, zeroOnError != nil, "the run emits the check this test is about")

	f := testkit.NewFailableTB()
	zeroOnError(f, multireturntest.PartialZero{InMemory: multireturntest.NewInMemory()})

	testkit.True(t, f.Failed(),
		"a non-zero slot beside an error must fail, whichever slot it is")
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestWideContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	multireturntest.RunWide(t,
		multireturntest.WideHarness[*multireturntest.InMemory]{Name: "in-memory", New: multireturntest.NewInMemory},
		multireturntest.WideSuite.Without(multireturntest.WideSuite.Checks.Triple.Smoke()),
	)
}
