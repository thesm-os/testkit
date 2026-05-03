// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package closertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/suite/testdata/erroronly"
	"go.thesmos.sh/testkit/gen/suite/testdata/erroronly/closertest"
)

func TestInMemoryCloserContract(t *testing.T) {
	t.Parallel()
	factory := func() erroronly.Closer { return erroronly.NewInMemoryCloser() }

	closertest.AssertCloserContract(t, factory,
		closertest.CloserPrePopulate(func(ctx context.Context, c erroronly.Closer) {
			_ = c.Open(ctx)
		}),
		closertest.CloserOnOpen(
			testkit.AssertLifecycleSucceeds[erroronly.Closer](),
		),
		closertest.CloserOnClose(
			testkit.AssertLifecycleSucceeds[erroronly.Closer](),
		),
	)
}
