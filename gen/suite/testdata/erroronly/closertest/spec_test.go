// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package closertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/suite/testdata/erroronly"
	"go.thesmos.sh/testkit/gen/suite/testdata/erroronly/closertest"
)

func TestInMemoryCloserContract(t *testing.T) {
	t.Parallel()
	factory := func() erroronly.Closer { return erroronly.NewInMemoryCloser() }

	closertest.AssertCloserContract(t, factory)
}
