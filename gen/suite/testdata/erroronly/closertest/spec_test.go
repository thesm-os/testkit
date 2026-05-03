// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package closertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/gen/suite/testdata/erroronly"
	"go.thesmos.sh/testkit/gen/suite/testdata/erroronly/closertest"
	"go.thesmos.sh/testkit/suite"
)

func TestInMemoryCloserContract(t *testing.T) {
	t.Parallel()
	factory := func() erroronly.Closer { return erroronly.NewInMemoryCloser() }

	closertest.AssertCloserContract(t, factory,
		closertest.CloserPrePopulate(func(ctx context.Context, c erroronly.Closer) {
			_ = c.Open(ctx)
		}),
		closertest.CloserOnOpen(
			suite.AssertLifecycleSucceeds[erroronly.Closer](),
		),
		closertest.CloserOnClose(
			suite.AssertLifecycleSucceeds[erroronly.Closer](),
		),
	)
}
