// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package bench_test

import (
	"context"
	"testing"
	"time"

	"go.thesmos.sh/testkit/bench"
	"go.thesmos.sh/testkit/bindings"
)

func benchStreamConsumerCtx(b *testing.B) bench.StreamConsumerContext[*chanConsumer, chanStream, int] {
	b.Helper()
	return bench.StreamConsumerContext[*chanConsumer, chanStream, int]{
		B: b,
		StreamConsumerBindings: bindings.StreamConsumerBindings[*chanConsumer, chanStream, int]{
			Factory: newChanConsumer,
			Call: func(ctx context.Context, c *chanConsumer, src chanStream) (int, error) {
				return c.Sum(ctx, src)
			},
		},
	}
}

func newChanStream() chanStream { return chanStream{1, 2, 3} }

func BenchmarkStreamConsumerHotPath(b *testing.B) {
	ctx := benchStreamConsumerCtx(b)
	bench.StreamConsumerHotPath[*chanConsumer, chanStream, int](newChanStream)(ctx)
}

func BenchmarkStreamConsumerAllocsWithin(b *testing.B) {
	ctx := benchStreamConsumerCtx(b)
	// Budget covers the per-iteration stream allocation produced by the factory.
	bench.StreamConsumerAllocsWithin[*chanConsumer, chanStream, int](newChanStream, 1)(ctx)
}

func BenchmarkStreamConsumerLatencyWithin(b *testing.B) {
	ctx := benchStreamConsumerCtx(b)
	bench.StreamConsumerLatencyWithin[*chanConsumer, chanStream, int](newChanStream, 100*time.Millisecond)(ctx)
}

func BenchmarkStreamConsumerConcurrentThroughput(b *testing.B) {
	ctx := benchStreamConsumerCtx(b)
	bench.StreamConsumerConcurrentThroughput[*chanConsumer, chanStream, int](newChanStream, 4)(ctx)
}
