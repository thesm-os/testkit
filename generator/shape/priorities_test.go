// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/shape"
)

func TestPriorities(t *testing.T) {
	t.Parallel()

	t.Run("priorities are unique across all detectors", func(t *testing.T) {
		t.Parallel()
		dets := shape.DefaultRegistry().Detectors()
		seen := make(map[int]string, len(dets))
		for _, d := range dets {
			if other, dup := seen[d.Priority()]; dup {
				t.Errorf("priority %d shared by %s and %s", d.Priority(), other, d.Name())
			}
			seen[d.Priority()] = d.Name()
		}
	})

	t.Run("registry exposes detectors in descending priority", func(t *testing.T) {
		t.Parallel()
		dets := shape.DefaultRegistry().Detectors()
		for i := 1; i < len(dets); i++ {
			testkit.True(t, dets[i-1].Priority() >= dets[i].Priority(),
				"descending order: "+dets[i-1].Name()+" before "+dets[i].Name())
		}
	})

	t.Run("documented constants expose every detector's priority", func(t *testing.T) {
		t.Parallel()
		// One test that maps every shipped constant to a known detector
		// — guards against silent additions where someone forgets the
		// constant. Using inline string lookup keeps the dependency
		// shallow.
		expected := map[string]int{
			"StreamReader":    shape.PriorityStreamReader,
			"BatchReader":     shape.PriorityBatchReader,
			"StreamConsumer":  shape.PriorityStreamConsumer,
			"Lookup":          shape.PriorityLookup,
			"ReaderWithBool":  shape.PriorityReaderWithBool,
			"PoisonAccessor":  shape.PriorityPoisonAccessor,
			"Predicate":       shape.PriorityPredicate,
			"VoidLifecycle":   shape.PriorityVoidLifecycle,
			"Pure":            shape.PriorityPure,
			"MultiArgWriter":  shape.PriorityMultiArgWriter,
			"CompositeWriter": shape.PriorityCompositeWriter,
			"MultiReader":     shape.PriorityMultiReader,
			"MultiAggregator": shape.PriorityMultiAggregator,
			"Deleter":         shape.PriorityDeleter,
			"Writer":          shape.PriorityWriter,
			"PointerReader":   shape.PriorityPointerReader,
			"Reader":          shape.PriorityReader,
			"ReaderNoError":   shape.PriorityReaderNoError,
			"Aggregator":      shape.PriorityAggregator,
			"Mutator":         shape.PriorityMutator,
			"Lifecycle":       shape.PriorityLifecycle,
		}
		dets := shape.DefaultRegistry().Detectors()
		got := make(map[string]int, len(dets))
		for _, d := range dets {
			got[d.Name()] = d.Priority()
		}
		for name, want := range expected {
			testkit.Equal(t, got[name], want, "priority constant for "+name)
		}
	})
}
