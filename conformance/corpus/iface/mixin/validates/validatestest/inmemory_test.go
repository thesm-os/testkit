// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package validatestest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/validates"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/validates/validatestest"
)

// The whole wiring a consumer writes: one subject, and the checks the generator
// has no classification to derive.
//
// No fixture and no seed. Both are derived — every value from the parameter's
// own type, the write from the method the annotator classified writer — and
// passing either here would be a derivation that had not been done. So is the
// run through the double, which comes from the //testkit:stub on the same
// interface.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	validatestest.AssertMixedContract(t,
		validatestest.MixedSubject("in-memory", func() validates.Mixed {
			return validatestest.NewInMemory()
		}),
		// The model tier: random sequences against the derived reference,
		// reporting under "model" beside the per-method checks.
		validatestest.MixedModel(),
		validatestest.MixedOnStore("refuses what its own validator refuses", func(
			tb testing.TB, subject validates.Mixed, v validates.Payload,
		) {
			tb.Helper()
			// The mixin's own law, written by hand until the generator reads
			// the classification: a payload with no key is one Validate
			// rejects, and Store must not take it.
			invalid := validates.Payload{Body: v.Body}
			testkit.ErrorIs(tb, subject.Validate(invalid), validatestest.ErrInvalid,
				"a payload with no key does not validate")
			testkit.ErrorIs(tb, subject.Store(tb.Context(), invalid), validatestest.ErrInvalid,
				"and Store refuses it for the same reason")
		}),
		validatestest.MixedOnRead("returns what Store wrote", func(
			tb testing.TB, subject validates.Mixed, key string,
		) {
			tb.Helper()
			// Writes its own precondition rather than reading the seed's. A
			// check that assumed what the seed left behind would break the
			// moment a subject supplied its own, and every argument it needs is
			// already handed to it.
			want := validates.Payload{Key: key, Body: "read-after-write"}
			testkit.NoError(tb, subject.Store(tb.Context(), want),
				"a valid payload stores under its own key")
			got, err := subject.Read(tb.Context(), key)
			testkit.NoError(tb, err, "and is found under it")
			testkit.Equal(tb, got, want, "and comes back whole")
		}),
	)
}

// Declining the double is a separate decision from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	validatestest.AssertMixedContract(t,
		validatestest.MixedSubject("in-memory", func() validates.Mixed {
			return validatestest.NewInMemory()
		}),
		validatestest.MixedWithout("Validate/smoke"),
		validatestest.MixedWithoutDouble(),
	)
}

// Overriding the two derived inputs, which is the shape of the escape hatch
// rather than something this subject needs.
//
// validates.Payload accepts any string, so the derivation is fine here. The
// override earns its place where a type validates — an identifier that must be
// UUID-shaped, an address that must hold an `@` — and corpus/integration/validated
// is that case, where the source declares an AccountDefaults the fixture reads
// instead.
func TestMixedContractWithSuppliedInputs(t *testing.T) {
	t.Parallel()

	fixture := validatestest.DefaultMixedFixture()
	fixture.V = validates.Payload{Key: "supplied-key", Body: "supplied-body"}

	validatestest.AssertMixedContract(t,
		validatestest.MixedSubject("in-memory", func() validates.Mixed {
			return validatestest.NewInMemory()
		}),
		validatestest.MixedWithFixture(fixture),
		validatestest.MixedSeed(func(ctx context.Context, subject validates.Mixed) error {
			return subject.Store(ctx, fixture.V)
		}),
	)
}
