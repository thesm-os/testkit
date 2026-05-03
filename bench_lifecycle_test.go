// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
)

func benchLifecycleCtx(b *testing.B) testkit.BenchLifecycleContext[*lifecycle] {
	b.Helper()
	return testkit.BenchLifecycleContext[*lifecycle]{
		B: b,
		LifecycleBindings: testkit.LifecycleBindings[*lifecycle]{
			Factory: newLifecycle,
			Call: func(ctx context.Context, l *lifecycle) error {
				return l.Open(ctx)
			},
		},
	}
}

func BenchmarkLifecycleAllocsWithin(b *testing.B) {
	ctx := benchLifecycleCtx(b)
	testkit.BenchLifecycleAllocsWithin[*lifecycle](0)(ctx)
}
