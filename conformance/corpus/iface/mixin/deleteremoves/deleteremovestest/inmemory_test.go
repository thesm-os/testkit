// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package deleteremovestest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deleteremoves"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deleteremoves/deleteremovestest"
)

// deleteremoves is the model tier's — AUTO-DELETE-RETURNS-NOT-FOUND states it —
// so the suite generates the signature family alone, even though eidos now lets
// the mixin name its reader through `read=Read`.
//
// Naming the partner is what makes the law bindable; stating it needs a
// reference to compare against, which is the line ADR-0018 draws.
func TestMixedContract(t *testing.T) {
	t.Parallel()

	fixture := deleteremovestest.DefaultMixedFixture()

	deleteremovestest.AssertMixedContract(t,
		deleteremovestest.MixedSubject("in-memory", func() deleteremoves.Mixed {
			return deleteremovestest.NewInMemory()
		}),
		deleteremovestest.MixedSeed(func(ctx context.Context, subject deleteremoves.Mixed) error {
			return subject.Put(ctx, fixture.Key, fixture.Value)
		}),
		deleteremovestest.MixedOnRead("returns what was written", func(
			tb testing.TB, subject deleteremoves.Mixed, key string,
		) {
			tb.Helper()
			got, err := subject.Read(tb.Context(), key)
			testkit.NoError(tb, err, "a written key is found")
			testkit.Equal(tb, got, fixture.Value, "and carries what was written")
		}),
	)
}

// A deleted key reads as absent rather than as empty, which is the difference
// between removing and tombstoning — and invisible to Delete, which reports
// only whether it failed.
func TestDeleteMakesTheKeyAbsent(t *testing.T) {
	t.Parallel()

	s := deleteremovestest.NewInMemory()
	testkit.NoError(t, s.Put(t.Context(), "k", "v"), "the write succeeds")
	testkit.NoError(t, s.Delete(t.Context(), "k"), "and so does the delete")

	_, err := s.Read(t.Context(), "k")
	testkit.ErrorIs(t, err, deleteremovestest.ErrNotFound,
		"the key reads as absent rather than as an empty value")
}

// Declining the double is separate from dropping a check.
func TestMixedContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	deleteremovestest.AssertMixedContract(t,
		deleteremovestest.MixedSubject("in-memory", func() deleteremoves.Mixed {
			return deleteremovestest.NewInMemory()
		}),
		deleteremovestest.MixedWithout("Delete/smoke"),
		deleteremovestest.MixedWithoutDouble(),
	)
}
