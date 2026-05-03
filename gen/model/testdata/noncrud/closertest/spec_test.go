// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package closertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/gen/model/testdata/noncrud"
	"go.thesmos.sh/testkit/gen/model/testdata/noncrud/closertest"
	"go.thesmos.sh/testkit/model/action"
)

func TestInMemoryCloserModel(t *testing.T) {
	t.Parallel()

	factory := func() noncrud.Closer { return noncrud.NewInMemoryCloser() }

	t.Run("lifecycle only with consumer reference", func(t *testing.T) {
		t.Parallel()
		// Non-CRUD: refmap synthesis unavailable. Consumer must supply
		// a reference. Both SUT and ref are InMemoryCloser instances.
		// The only auto-derived actions are Lifecycle (Close, Ping).
		// No auto-laws since there's no Reader+Writer combination.
		closertest.AssertCloserModel(t, factory,
			closertest.CloserModelReference(factory),
		)
	})

	t.Run("replace all actions", func(t *testing.T) {
		t.Parallel()
		// CloserModelActions replaces the entire auto-derived action set.
		// Useful when the consumer wants full control over what gets tested.
		closertest.AssertCloserModel(t, factory,
			closertest.CloserModelReference(factory),
			closertest.CloserModelActions(
				// Only test Ping, skip Close entirely.
				action.Lifecycle("Ping",
					func(ctx context.Context, impl noncrud.Closer) error {
						return impl.Ping(ctx)
					},
				),
			),
		)
	})
}
