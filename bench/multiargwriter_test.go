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

func benchMultiArgWriterCtx(b *testing.B) bench.MultiArgWriterContext[*subscriber, string, string, int] {
	b.Helper()
	return bench.MultiArgWriterContext[*subscriber, string, string, int]{
		B: b,
		MultiArgWriterBindings: bindings.MultiArgWriterBindings[*subscriber, string, string, int]{
			Factory: newSubscriber,
			Call: func(ctx context.Context, s *subscriber, topic, group string, prio int) error {
				return s.Subscribe(ctx, topic, group, prio)
			},
		},
	}
}

func BenchmarkMultiArgWriterHotPath(b *testing.B) {
	ctx := benchMultiArgWriterCtx(b)
	bench.MultiArgWriterHotPath[*subscriber, string, string, int]("topic", "group", 1)(ctx)
}

func BenchmarkMultiArgWriterAllocsWithin(b *testing.B) {
	ctx := benchMultiArgWriterCtx(b)
	bench.MultiArgWriterAllocsWithin[*subscriber, string, string, int]("topic", "group", 1, 1)(ctx)
}

func BenchmarkMultiArgWriterLatencyWithin(b *testing.B) {
	ctx := benchMultiArgWriterCtx(b)
	bench.MultiArgWriterLatencyWithin[*subscriber, string, string, int]("topic", "group", 1, 100*time.Millisecond)(ctx)
}

func BenchmarkMultiArgWriterConcurrentThroughput(b *testing.B) {
	ctx := benchMultiArgWriterCtx(b)
	bench.MultiArgWriterConcurrentThroughput[*subscriber, string, string, int]("topic", "group", 1, 4)(ctx)
}
