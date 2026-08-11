// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package receivercollisiontest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/receivercollision"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/receivercollision/receivercollisiontest"
)

// Every method here names a parameter `s`, one at Session and one at string.
//
// The fixture keys on the name *and* the type, so the checks are handed PutS
// and GetS rather than one value the other method could not take. The author's
// own rule is that the two agree: what Put stored under a session's identifier
// is what Get returns for it.
func TestStoreContract(t *testing.T) {
	t.Parallel()

	// Neither the fixture nor the seed is supplied: Put is classified writer, so
	// the run already writes fixture.PutS through it before every check. Reading
	// the same derivation here is what lets the check below name the identifier
	// that was stored.
	fixture := receivercollisiontest.DefaultStoreFixture()

	receivercollisiontest.AssertStoreContract(t,
		receivercollisiontest.StoreSubject("in-memory", func() receivercollision.Store {
			return receivercollisiontest.NewInMemory()
		}),
		receivercollisiontest.StoreOnGet("returns what Put stored under that identifier", func(
			tb testing.TB, subject receivercollision.Store, id string,
		) {
			tb.Helper()
			got, err := subject.Get(tb.Context(), fixture.PutS.ID)
			testkit.NoError(tb, err, "a stored session is found by its own identifier")
			testkit.Equal(tb, got, fixture.PutS, "and comes back whole")
		}),
	)
}

// Touch records something rather than only returning, which the contract cannot
// state: observing it needs Touched, and Touched is not on the interface.
func TestTouchRecords(t *testing.T) {
	t.Parallel()

	s := receivercollisiontest.NewInMemory()
	sess := receivercollisiontest.DefaultStoreFixture().PutS
	s.Touch(t.Context(), sess)
	testkit.True(t, s.Touched(sess.ID), "Touch does something observable")
}

// A method returning nothing cannot report a context that is done, so it takes
// the only other option open to it and does no work.
func TestTouchDeclinesADoneContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	s := receivercollisiontest.NewInMemory()
	sess := receivercollisiontest.DefaultStoreFixture().PutS
	s.Touch(ctx, sess)
	testkit.False(t, s.Touched(sess.ID), "a cancelled Touch records nothing")
}

// Suppression, against the same subject: what is under test is the harness
// declining what it was told to.
func TestStoreContractSuppression(t *testing.T) {
	t.Parallel()

	receivercollisiontest.AssertStoreContract(t,
		receivercollisiontest.StoreSubject("in-memory", func() receivercollision.Store {
			return receivercollisiontest.NewInMemory()
		}),
		receivercollisiontest.StoreWithout("Touch/smoke"),
		receivercollisiontest.StoreWithoutDouble(),
	)
}
