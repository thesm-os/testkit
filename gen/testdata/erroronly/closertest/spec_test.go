// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package closertest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/gen/testdata/erroronly"
	"go.thesmos.sh/testkit/gen/testdata/erroronly/closertest"
	"go.thesmos.sh/testkit/suite"
)

func inmemoryFactory() erroronly.Closer {
	return erroronly.NewInMemoryCloser()
}

func stubFactory() erroronly.Closer {
	return closertest.NewCloserStub(nil,
		closertest.CloserStubDelegateTo(inmemoryFactory()))
}

func stubBenchFactory() erroronly.Closer {
	return closertest.NewCloserStub(nil,
		closertest.CloserStubBenchMode(),
		closertest.CloserStubDelegateTo(inmemoryFactory()))
}

// --- InMemory ---

func TestCloserContract_InMemory(t *testing.T) {
	t.Parallel()
	closertest.AssertCloserContract(t, inmemoryFactory,
		closertest.CloserPrePopulate(func(ctx context.Context, c erroronly.Closer) {
			_ = c.Open(ctx)
		}),
		closertest.CloserOnOpen(
			suite.AssertLifecycleSucceeds[erroronly.Closer](),
			suite.AssertLifecycleIdempotent[erroronly.Closer](),
		),
		closertest.CloserOnClose(
			suite.AssertLifecycleSucceeds[erroronly.Closer](),
		),
	)
}

func BenchmarkCloserContract_InMemory(b *testing.B) {
	closertest.BenchmarkCloserContract(b, inmemoryFactory)
}

// --- Stub+DelegateTo ---

func TestCloserContract_StubDelegateTo(t *testing.T) {
	t.Parallel()
	closertest.AssertCloserContract(t, stubFactory)
}

func BenchmarkCloserContract_StubDelegateTo(b *testing.B) {
	closertest.BenchmarkCloserContract(b, stubFactory)
}

// --- Stub+BenchMode ---

func BenchmarkCloserContract_StubBenchMode(b *testing.B) {
	closertest.BenchmarkCloserContract(b, stubBenchFactory)
}
