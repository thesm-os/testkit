// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package readerwithbooltest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/readerwithbool"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/readerwithbool/readerwithbooltest"
)

// The comma-ok read: absence is a flag, not a failure, so no error slot exists
// and the signature earns only the check that asks about crashing.
//
// The claim worth making — that the flag and the value agree — is stated here.
// A subject returning a populated value beside false, or the zero beside true,
// satisfies every generated check and is broken.
func TestReaderWithBoolContract(t *testing.T) {
	t.Parallel()

	fx := readerwithbooltest.DefaultReaderWithBoolFixture()

	readerwithbooltest.RunReaderWithBool(t,
		readerwithbooltest.ReaderWithBoolHarness[*readerwithbooltest.InMemory]{
			Name: "in-memory",
			New: func() *readerwithbooltest.InMemory {
				s := readerwithbooltest.NewInMemory()
				s.Put(readerwithbool.Value{Key: fx.Key(), Body: "seeded"})
				return s
			},
		},
		readerwithbooltest.ReaderWithBoolChecks{
			{
				Method: "Load",
				Name:   "flag-agrees-on-a-hit",
				Claim:  "Load agrees with its own flag on a hit",
				Run: func(tb testing.TB, s readerwithbool.ReaderWithBool, fx readerwithbooltest.ReaderWithBoolFixture) {
					tb.Helper()
					got, ok := s.Load(tb.Context(), fx.Key())
					testkit.True(tb, ok, "a seeded key is present")
					testkit.Equal(tb, got.Body, "seeded", "and the value comes with the flag")
				},
			},
			{
				Method: "Load",
				Name:   "flag-agrees-on-a-miss",
				Claim:  "Load agrees with its own flag on a miss",
				Run: func(tb testing.TB, s readerwithbool.ReaderWithBool, fx readerwithbooltest.ReaderWithBoolFixture) {
					tb.Helper()
					// Both halves, because either alone is satisfied by a
					// broken subject: false beside a populated value, or true
					// beside the zero.
					got, ok := s.Load(tb.Context(), fx.KeyOther())
					testkit.False(tb, ok, "an unwritten key is absent")
					testkit.Equal(tb, got, readerwithbool.Value{}, "and the value slot is the zero")
				},
			},
		},
	)
}

// Dropping a check is written against the typed index rather than a string, so
// a check that is renamed or stops being emitted breaks this compile instead of
// silently declining nothing.
func TestReaderWithBoolContractWithoutSmoke(t *testing.T) {
	t.Parallel()

	readerwithbooltest.RunReaderWithBool(
		t,
		readerwithbooltest.ReaderWithBoolHarness[*readerwithbooltest.InMemory]{
			Name: "in-memory",
			New:  readerwithbooltest.NewInMemory,
		},
		readerwithbooltest.ReaderWithBoolSuite.Without(readerwithbooltest.ReaderWithBoolSuite.Checks.Load.Smoke()),
	)
}
