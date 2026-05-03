// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/gen/model/testdata/basic"
	"go.thesmos.sh/testkit/gen/model/testdata/basic/storetest"
	"go.thesmos.sh/testkit/model/law"
)

func TestInMemoryStoreModel(t *testing.T) {
	t.Parallel()

	t.Run("tier 0 zero code", func(t *testing.T) {
		t.Parallel()
		// Tier 0: auto-synthesized reference (refmap.MapStore),
		// auto-derived actions, auto-derived ReadAfterWrite +
		// CountEqualsReference laws. No consumer code beyond the factory.
		storetest.AssertStoreModel(t, func() basic.Store {
			return basic.NewInMemoryStore()
		})
	})

	t.Run("tier 2 custom law", func(t *testing.T) {
		t.Parallel()
		// Tier 2: add a domain-specific law. Count must never go negative.
		storetest.AssertStoreModel(t, func() basic.Store {
			return basic.NewInMemoryStore()
		},
			storetest.StoreModelLaw(countNonNegative{}),
		)
	})

	t.Run("skip auto law", func(t *testing.T) {
		t.Parallel()
		// Opt out of an auto-derived law by ID.
		storetest.AssertStoreModel(t, func() basic.Store {
			return basic.NewInMemoryStore()
		},
			storetest.StoreModelSkipLaw("AUTO-COUNT-EQUALS-REFERENCE"),
		)
	})
}

// countNonNegative is a Tier 2 domain-specific law: Count >= 0 always.
type countNonNegative struct{}

func (countNonNegative) ID() string    { return "CUSTOM-COUNT-NON-NEGATIVE" }
func (countNonNegative) REQID() string { return "" }

func (countNonNegative) Check(rt *rapid.T, sut, ref basic.Store) error {
	n, err := sut.Count(rt.Context())
	if err != nil {
		return err
	}
	if n < 0 {
		return fmt.Errorf("count is negative: %d", n)
	}
	return nil
}

// Compile-time check that countNonNegative satisfies Law[basic.Store].
var _ law.Law[basic.Store] = countNonNegative{}
