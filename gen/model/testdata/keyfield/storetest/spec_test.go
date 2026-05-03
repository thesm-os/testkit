// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storetest_test

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/gen/model/testdata/keyfield"
	"go.thesmos.sh/testkit/gen/model/testdata/keyfield/storetest"
	"go.thesmos.sh/testkit/model/law"
)

func TestInMemoryStoreModel(t *testing.T) {
	t.Parallel()

	t.Run("tier 0 directive overrides heuristic", func(t *testing.T) {
		t.Parallel()
		// Tier 0: //testkit:keyfield Key overrides the ID heuristic.
		// refmap.MapStore uses record.Key for extraction. Record has
		// no "ID" field — without the directive, Tier 0 would fail.
		storetest.AssertStoreModel(t, func() keyfield.Store {
			return keyfield.NewInMemoryStore()
		})
	})

	t.Run("tier 2 custom law with directive keyfield", func(t *testing.T) {
		t.Parallel()
		// Custom law verifying that Put followed by Get preserves
		// the Value field (not just the Key used for lookup).
		storetest.AssertStoreModel(t, func() keyfield.Store {
			return keyfield.NewInMemoryStore()
		},
			storetest.StoreModelLaw(valuePreserved{}),
		)
	})
}

// valuePreserved checks that the Value field round-trips correctly.
type valuePreserved struct{}

func (valuePreserved) ID() string    { return "CUSTOM-VALUE-PRESERVED" }
func (valuePreserved) REQID() string { return "" }

func (valuePreserved) Check(rt *rapid.T, sut, ref keyfield.Store) error {
	key := rapid.SampledFrom([]string{"a", "b", "c", "d", "e"}).Draw(rt, "key")
	sutV, sutErr := sut.Get(rt.Context(), key)
	refV, refErr := ref.Get(rt.Context(), key)
	if sutErr != refErr {
		return fmt.Errorf("Get(%q): SUT err=%v, ref err=%v", key, sutErr, refErr)
	}
	if sutErr == nil && sutV.Value != refV.Value {
		return fmt.Errorf("Get(%q).Value: SUT=%q, ref=%q", key, sutV.Value, refV.Value)
	}
	return nil
}

var _ law.Law[keyfield.Store] = valuePreserved{}
