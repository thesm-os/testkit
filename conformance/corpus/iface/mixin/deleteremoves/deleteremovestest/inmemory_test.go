// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package deleteremovestest_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deleteremoves"
	"go.thesmos.sh/testkit/conformance/corpus/iface/mixin/deleteremoves/deleteremovestest"
	"go.thesmos.sh/testkit/engine/model"
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
		// The model tier: random sequences against the derived reference,
		// reporting under "model" beside the per-method checks.
		deleteremovestest.MixedModel(),
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

// The saturation prover: every bound law must be able to fail as itself,
// a defect worn on its own methods reddening the run by name.
func TestMixedSaturation(t *testing.T) {
	t.Parallel()
	deleteremovestest.MixedModelSaturation(t, func() deleteremoves.Mixed {
		return deleteremovestest.NewInMemory()
	})
}

// privateMiss is the audit's deleteremoves defect made flesh: it stores
// faithfully and misses under its own error where the declaration stamped
// ErrGone. Presence-only comparison reads that as agreement.
type privateMiss struct{ deleteremoves.Mixed }

func (p privateMiss) Read(ctx context.Context, key string) (string, error) {
	v, err := p.Mixed.Read(ctx, key)
	if errors.Is(err, deleteremoves.ErrGone) {
		return v, errors.New("deleteremovestest: private miss")
	}
	return v, err
}

// The wrong miss identity cannot survive the property: the routed sentinel
// arms the namesake law's guard, so the law catches it at the first step —
// and the sequences behind it carry the same identity through
// action.WithSentinel for every read the law never makes. The probe pins
// the first defense by its own message; the second is engine-proven.
func TestMixedPropertyCatchesAWrongMissIdentity(t *testing.T) {
	t.Parallel()

	got := testkit.Rejects(t, "a miss answered under a private error",
		func(tb testing.TB) {
			tb.Helper()
			model.Check(tb, deleteremovestest.MixedModelProperty(func() deleteremoves.Mixed {
				return privateMiss{deleteremovestest.NewInMemory()}
			}))
		})
	testkit.Assert(t, got).Contains("ref returned sentinel",
		"and rejects it for the identity, not for some unrelated divergence")
}
