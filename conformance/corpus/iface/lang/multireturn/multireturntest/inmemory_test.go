// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package multireturntest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/multireturn"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/multireturn/multireturntest"
)

// A method returning several values beside its error owes the zero for each.
func TestWideContract(t *testing.T) {
	t.Parallel()

	multireturntest.AssertWideContract(t,
		multireturntest.WideSubject("in-memory", func() multireturn.Wide {
			return multireturntest.NewInMemory()
		}),
	)
}

// A subject zeroing only its first slot must fail, or the generated check is
// reading one of three and reporting on all of them.
//
// This is what running the fixture adds over compiling it: a template emitting
// one comparison instead of three compiles identically, and only a subject that
// zeroes some slots and not others can tell the two apart.
func TestQuadZeroOnErrorReadsEverySlot(t *testing.T) {
	t.Parallel()

	f := testkit.NewFailableTB()
	multireturntest.AssertWideQuadZeroOnError(f,
		multireturntest.PartialZero{InMemory: multireturntest.NewInMemory()}, "missing")

	testkit.True(t, f.Failed(),
		"a non-zero slot beside an error must fail, whichever slot it is")
}

func TestWideContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	// The values the run itself uses, read rather than replaced: nothing here
	// passes WideWithFixture, so the derivation stands.
	fixture := multireturntest.DefaultWideFixture()

	multireturntest.AssertWideContract(t,
		multireturntest.WideSubject("seeded", func() multireturn.Wide {
			return multireturntest.NewInMemory()
		}),
		multireturntest.WideSeed(func(_ context.Context, subject multireturn.Wide) error {
			subject.(*multireturntest.InMemory).Put(fixture.ID, "found")
			return nil
		}),
		multireturntest.WideOnQuad("returns every slot for a hit", func(
			tb testing.TB, subject multireturn.Wide, id string,
		) {
			tb.Helper()
			v, n, ok, err := subject.Quad(tb.Context(), id)
			testkit.NoError(tb, err, "a seeded identifier is found")
			testkit.Equal(tb, v, "found", "the value slot carries it")
			testkit.Equal(tb, n, len("found"), "the derived slot agrees")
			testkit.True(tb, ok, "and the flag says so")
		}),
		multireturntest.WideOnTriple("reports absence through its flag", func(
			tb testing.TB, subject multireturn.Wide, id string,
		) {
			tb.Helper()
			_, _, ok := subject.Triple(tb.Context(), "absent")
			testkit.False(tb, ok, "a method with no error slot says so through the flag")
		}),
		multireturntest.WideOnNoError("returns the zero for an absent identifier", func(
			tb testing.TB, subject multireturn.Wide, id string,
		) {
			tb.Helper()
			v, n := subject.NoError(tb.Context(), "absent")
			testkit.Equal(tb, v, "", "the value slot is zero")
			testkit.Equal(tb, n, 0, "and so is the derived one")
		}),
		multireturntest.WideWithout("Triple/smoke"),
		multireturntest.WideWithoutDouble(),
	)
}
