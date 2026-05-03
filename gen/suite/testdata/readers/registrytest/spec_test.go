// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package registrytest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/suite/testdata/readers"
	"go.thesmos.sh/testkit/gen/suite/testdata/readers/registrytest"
)

func TestInMemoryRegistryContract(t *testing.T) {
	t.Parallel()
	factory := func() readers.Registry {
		r := readers.NewInMemoryRegistry()
		r.Register(readers.Handler{Name: "handler-1", Version: 1})
		r.Register(readers.Handler{Name: "handler-2", Version: 2})
		return r
	}

	registrytest.AssertRegistryContract(t, factory,
		registrytest.PrePopulate(func(_ testing.TB, _ readers.Registry) {
			// Factory already populates — PrePopulate is a no-op here.
		}),
		registrytest.Known("handler-1"),
		registrytest.Unknown("nonexistent"),
		registrytest.Expect("handler-1", readers.Handler{Name: "handler-1", Version: 1}),

		// Reader plug-in primitives on Lookup.
		registrytest.OnLookup(
			testkit.AssertReturnsForKey[readers.Registry, string, readers.Handler](),
			testkit.AssertReturnsSentinel[readers.Registry, string, readers.Handler](readers.ErrNotRegistered),
			testkit.AssertConsistentReads[readers.Registry, string, readers.Handler](5),
			testkit.AssertReaderConcurrentSafe[readers.Registry, string, readers.Handler](4, 50),
		),

		// Free-form custom subtest.
		registrytest.AssertCustom("List returns all registered handlers", func(t *testing.T, r readers.Registry) {
			count := 0
			for _, err := range r.List(t.Context()) {
				testkit.NoError(t, err, "List must not error")
				count++
			}
			testkit.Equal(t, count, 2, "must list all registered handlers")
		}),
	)
}
