// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestShape(t *testing.T) {
	t.Parallel()

	t.Run("String returns canonical name for every shape", func(t *testing.T) {
		t.Parallel()
		cases := map[shape.Shape]string{
			shape.Reader:          "Reader",
			shape.ReaderNoError:   "ReaderNoError",
			shape.ReaderWithBool:  "ReaderWithBool",
			shape.Lookup:          "Lookup",
			shape.PointerReader:   "PointerReader",
			shape.MultiReader:     "MultiReader",
			shape.BatchReader:     "BatchReader",
			shape.Writer:          "Writer",
			shape.CompositeWriter: "CompositeWriter",
			shape.Mutator:         "Mutator",
			shape.Deleter:         "Deleter",
			shape.MultiArgWriter:  "MultiArgWriter",
			shape.Aggregator:      "Aggregator",
			shape.MultiAggregator: "MultiAggregator",
			shape.StreamReader:    "StreamReader",
			shape.StreamConsumer:  "StreamConsumer",
			shape.Pure:            "Pure",
			shape.Predicate:       "Predicate",
			shape.PoisonAccessor:  "PoisonAccessor",
			shape.Lifecycle:       "Lifecycle",
			shape.VoidLifecycle:   "VoidLifecycle",
			shape.Unknown:         "Unknown",
		}
		for s, want := range cases {
			testkit.Equal(t, s.String(), want, "Shape String")
		}
	})

	t.Run("Info zero value defaults to Unknown", func(t *testing.T) {
		t.Parallel()
		var i shape.Info
		testkit.Equal(t, i.Shape, shape.Unknown, "default Shape is Unknown")
		testkit.Equal(t, i.KeyType, "", "no KeyType")
		testkit.Equal(t, i.ValType, "", "no ValType")
	})
}
