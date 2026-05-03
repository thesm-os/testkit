// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package servicetest_test

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/gen/model/testdata/allshapes"
	"go.thesmos.sh/testkit/gen/model/testdata/allshapes/servicetest"
	"go.thesmos.sh/testkit/model/action"
	"go.thesmos.sh/testkit/model/law"
)

func TestInMemoryServiceModel(t *testing.T) {
	t.Parallel()

	factory := func() allshapes.Service { return allshapes.NewInMemoryService() }

	t.Run("tier 1 consumer reference", func(t *testing.T) {
		t.Parallel()
		// Tier 1: allshapes has Pure/Predicate/Stream/Lifecycle methods
		// that refmap.MapStore can't satisfy. Consumer supplies reference.
		servicetest.AssertServiceModel(t, factory,
			servicetest.ServiceModelReference(factory),
		)
	})

	t.Run("extra actions for coverage", func(t *testing.T) {
		t.Parallel()
		// ExtraActions supplements auto-derived actions without replacing them.
		// Here we add a "GetAndCheck" action that reads a key and verifies
		// the result is either an item or ErrNotFound.
		servicetest.AssertServiceModel(t, factory,
			servicetest.ServiceModelReference(factory),
			servicetest.ServiceModelExtraActions(
				action.Unknown("GetAndCheck",
					func(rt *rapid.T, sut, ref allshapes.Service) {
						key := rapid.SampledFrom([]string{"a", "b", "c"}).Draw(rt, "key")
						_, err := sut.Get(rt.Context(), key)
						if err != nil && err != allshapes.ErrNotFound {
							rt.Fatalf("unexpected error: %v", err)
						}
					},
				),
			),
		)
	})

	t.Run("tier 2 custom law", func(t *testing.T) {
		t.Parallel()
		// Tier 2: custom domain invariant. Describe() must be stable
		// (calling it twice yields the same result with no state change).
		servicetest.AssertServiceModel(t, factory,
			servicetest.ServiceModelReference(factory),
			servicetest.ServiceModelLaw(describeStable{}),
		)
	})

	t.Run("skip auto law", func(t *testing.T) {
		t.Parallel()
		// Opt out of ReadAfterWrite — useful when the SUT has eventual
		// consistency semantics that the auto-law doesn't model.
		servicetest.AssertServiceModel(t, factory,
			servicetest.ServiceModelReference(factory),
			servicetest.ServiceModelSkipLaw("AUTO-READ-AFTER-WRITE"),
		)
	})
}

// describeStable checks that Describe() is deterministic: two calls
// with no intervening state change must return the same value.
type describeStable struct{}

func (describeStable) ID() string    { return "CUSTOM-DESCRIBE-STABLE" }
func (describeStable) REQID() string { return "" }

func (describeStable) Check(_ *rapid.T, sut, _ allshapes.Service) error {
	a := sut.Describe()
	b := sut.Describe()
	if a != b {
		return fmt.Errorf("Describe() not stable: %q then %q", a, b)
	}
	return nil
}

var _ law.Law[allshapes.Service] = describeStable{}
