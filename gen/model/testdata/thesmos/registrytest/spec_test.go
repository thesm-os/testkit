// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package registrytest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/model/testdata/thesmos"
	"go.thesmos.sh/testkit/gen/model/testdata/thesmos/registrytest"
)

func TestKindRegistryModel(t *testing.T) {
	t.Parallel()

	t.Run("map SUT vs slice ref", func(t *testing.T) {
		t.Parallel()
		// Different impls, pre-populated so Lookup has data to read.
		registrytest.AssertKindRegistryModel(t,
			func() thesmos.KindRegistry {
				r := thesmos.NewInMemoryRegistry()
				seedRegistry(r)
				return r
			},
			registrytest.KindRegistryModelReference(func() thesmos.KindRegistry {
				r := thesmos.NewSliceRegistry()
				seedRegistry(r)
				return r
			}),
		)
	})

	t.Run("catches broken Lookup", func(t *testing.T) {
		t.Parallel()
		// Negative: BrokenLookupRegistry corrupts spec.Name on Lookup.
		// Pre-populated so Lookup finds entries and triggers the corruption.
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			registrytest.AssertKindRegistryModel(ft,
				func() thesmos.KindRegistry {
					r := thesmos.NewBrokenLookupRegistry()
					seedRegistry(r)
					return r
				},
				registrytest.KindRegistryModelReference(func() thesmos.KindRegistry {
					r := thesmos.NewSliceRegistry()
					seedRegistry(r)
					return r
				}),
			)
		}()
		<-done
		if !ft.Failed() {
			t.Fatal("framework should have caught corrupted Lookup spec")
		}
	})
}

// sharedFold is a package-level function so both SUT and ref get the same
// function value. cmp.Diff compares function pointers, so closures allocated
// separately would always differ.
var sharedFold thesmos.FoldFunc = func([]byte) error { return nil }

func seedRegistry(r thesmos.KindRegistry) {
	for i := range 5 {
		_ = r.Register(thesmos.KindSpec{Name: "kind", Version: i + 1}, sharedFold)
	}
}
