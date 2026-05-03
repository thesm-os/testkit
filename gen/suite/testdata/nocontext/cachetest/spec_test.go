// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cachetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/suite/testdata/nocontext"
	"go.thesmos.sh/testkit/gen/suite/testdata/nocontext/cachetest"
)

func TestInMemoryCacheContract(t *testing.T) {
	t.Parallel()
	factory := func() nocontext.Cache { return nocontext.NewInMemoryCache() }

	cachetest.AssertCacheContract(t, factory)
}
