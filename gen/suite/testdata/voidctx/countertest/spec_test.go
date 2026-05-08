// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package countertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/suite/testdata/voidctx"
	"go.thesmos.sh/testkit/gen/suite/testdata/voidctx/countertest"
)

func TestCounterContract(t *testing.T) {
	t.Parallel()
	factory := func() voidctx.Counter { return voidctx.NewInMemoryCounter("test") }
	countertest.AssertCounterContract(t, factory)
}

func BenchmarkCounterContract(b *testing.B) {
	factory := func() voidctx.Counter { return voidctx.NewInMemoryCounter("test") }
	countertest.BenchmarkCounterContract(b, factory)
}
