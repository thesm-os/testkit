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
		registrytest.OnLookup(
			testkit.AssertReturnsForKey[readers.Registry, string, readers.Handler](
				"handler-1", readers.Handler{Name: "handler-1", Version: 1},
			),
			testkit.AssertReturnsSentinel[readers.Registry, string, readers.Handler](
				"nonexistent", readers.ErrNotRegistered,
			),
			testkit.AssertConsistentReads[readers.Registry, string, readers.Handler]("handler-1", 5),
			testkit.AssertReaderConcurrentSafe[readers.Registry, string, readers.Handler]("handler-1", 4, 50),
		),

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
