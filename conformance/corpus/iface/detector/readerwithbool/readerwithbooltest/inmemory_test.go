// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package readerwithbooltest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/readerwithbool"
	"go.thesmos.sh/testkit/conformance/corpus/iface/detector/readerwithbool/readerwithbooltest"
)

// The comma-ok read: absence is a flag, not a failure, so no error slot exists
// and the signature earns only the two checks that ask about crashing.
//
// The claim worth making — that the flag and the value agree — is stated here.
// A subject returning a populated value beside false, or the zero beside true,
// satisfies every generated check and is broken.
func TestReaderWithBoolContract(t *testing.T) {
	t.Parallel()

	// The values the run itself uses, read rather than replaced: nothing here
	// passes ReaderWithBoolWithFixture, so the derivation stands.
	fixture := readerwithbooltest.DefaultReaderWithBoolFixture()

	readerwithbooltest.AssertReaderWithBoolContract(t,
		readerwithbooltest.ReaderWithBoolSubject("in-memory", func() readerwithbool.ReaderWithBool {
			return readerwithbooltest.NewInMemory()
		}),
		readerwithbooltest.ReaderWithBoolSeed(func(_ context.Context, subject readerwithbool.ReaderWithBool) error {
			// A seed may reach for the concrete subject: it runs before the
			// double wraps it and sees what the factory made. A check may not.
			subject.(*readerwithbooltest.InMemory).Put(
				readerwithbool.Value{Key: fixture.Key, Body: "seeded"},
			)
			return nil
		}),
		readerwithbooltest.ReaderWithBoolOnLoad("agrees with its own flag on a hit", func(
			tb testing.TB, subject readerwithbool.ReaderWithBool, key string,
		) {
			tb.Helper()
			// Only the hit. The miss — false beside the zero value — is the
			// readerwithbool classification's own check and is generated, so
			// restating it here would be one property asserted twice, drifting
			// apart the moment either changes.
			got, ok := subject.Load(tb.Context(), key)
			testkit.True(tb, ok, "a seeded key is present")
			testkit.Equal(tb, got.Body, "seeded", "and the value comes with the flag")
		}),
	)
}

// Declining the double is separate from dropping a check.
func TestReaderWithBoolContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	readerwithbooltest.AssertReaderWithBoolContract(t,
		readerwithbooltest.ReaderWithBoolSubject("in-memory", func() readerwithbool.ReaderWithBool {
			return readerwithbooltest.NewInMemory()
		}),
		readerwithbooltest.ReaderWithBoolWithout("Load/smoke"),
		readerwithbooltest.ReaderWithBoolWithoutDouble(),
	)
}
