// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package allshapestest_test

import (
	"context"
	"io"
	"iter"
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/interfaces"
	"go.thesmos.sh/testkit/generator/testdata/interfaces/allshapestest"
)

// BenchmarkAllShapesStub runs the contract driver against the
// generated stub in BenchMode for every shape the registry detects.
// With recording disabled and no-op Func overrides on every method,
// the dispatch chain must be alloc-free across all 21 shape
// primitives — proves the stub generator's "transparent recording"
// claim broadly, not just on simple Reader/Writer methods.
func BenchmarkAllShapesStub(b *testing.B) {
	allshapestest.BenchmarkAllShapesContract(b, func() interfaces.AllShapes {
		return allshapestest.NewAllShapesStub(b,
			allshapestest.AllShapesStubBenchMode(),
			allshapestest.WithAllShapesAll(func(_ context.Context) iter.Seq[interfaces.Record] {
				return func(yield func(interfaces.Record) bool) {}
			}),
			allshapestest.WithAllShapesCount(func(_ context.Context) (int, error) { return 0, nil }),
			allshapestest.WithAllShapesDescription(func() string { return "" }),
			allshapestest.WithAllShapesErr(func() error { return nil }),
			allshapestest.WithAllShapesFetch(func(_ context.Context, _ string) (interfaces.Record, string, error) {
				return interfaces.Record{}, "", nil
			}),
			allshapestest.WithAllShapesFind(func(_ context.Context, _ string) *interfaces.Record { return nil }),
			allshapestest.WithAllShapesGet(func(_ context.Context, _ string) (interfaces.Record, error) {
				return interfaces.Record{}, nil
			}),
			allshapestest.WithAllShapesInit(func(_ context.Context) error { return nil }),
			allshapestest.WithAllShapesInspect(func(_ context.Context, _ string) (interfaces.Record, string, bool) {
				return interfaces.Record{}, "", false
			}),
			allshapestest.WithAllShapesIsHealthy(func() bool { return true }),
			allshapestest.WithAllShapesLoad(func(_ context.Context, _ string) (interfaces.Record, bool) {
				return interfaces.Record{}, false
			}),
			allshapestest.WithAllShapesLookup(func(_ context.Context, _ string) interfaces.Record {
				return interfaces.Record{}
			}),
			allshapestest.WithAllShapesMany(func(_ context.Context, _ ...string) ([]interfaces.Record, error) {
				return nil, nil
			}),
			allshapestest.WithAllShapesPut(func(_ context.Context, _ interfaces.Record) error { return nil }),
			allshapestest.WithAllShapesReadFrom(func(_ context.Context, _ io.Reader) (int, error) { return 0, nil }),
			allshapestest.WithAllShapesRemove(func(_ context.Context, _ string) error { return nil }),
			allshapestest.WithAllShapesReset(func() {}),
			allshapestest.WithAllShapesScan(func(_ context.Context) iter.Seq2[interfaces.Record, error] {
				return func(yield func(interfaces.Record, error) bool) {}
			}),
			allshapestest.WithAllShapesSchedule(func(_ context.Context, _ string, _ interfaces.Record, _ int) error {
				return nil
			}),
			allshapestest.WithAllShapesSet(func(_ context.Context, _ string, _ interfaces.Record) error { return nil }),
			allshapestest.WithAllShapesStatistics(func(_ context.Context) (int, int, int, error) {
				return 0, 0, 0, nil
			}),
			allshapestest.WithAllShapesStats(func(_ context.Context) (int, int, error) { return 0, 0, nil }),
			allshapestest.WithAllShapesTouch(func(_ context.Context, _ string) {}),
		)
	})
}
