// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/law"
)

// Minimal fixtures for the internal (package model) coverage tests;
// the package model_test store fixtures are not visible here.
type covItem struct {
	ID   string
	Name string
}

type storeCovIface interface {
	Get(context.Context, string) (covItem, error)
	Count(context.Context) (int, error)
}

func TestStateSpaceTracker(t *testing.T) {
	t.Parallel()

	t.Run("distinct hashes grow the explored set", func(t *testing.T) {
		t.Parallel()
		tr := newStateSpaceTracker(50)
		m := tr.observe(1)
		if m.Explored != 1 || m.IterationsSinceLastNew != 0 || m.Saturated {
			t.Fatalf("first observe: %+v", m)
		}
		m = tr.observe(2)
		if m.Explored != 2 || m.IterationsSinceLastNew != 0 {
			t.Fatalf("second distinct observe: %+v", m)
		}
	})

	t.Run("repeat increments iterations-since-new and saturates at threshold", func(t *testing.T) {
		t.Parallel()
		tr := newStateSpaceTracker(3)
		tr.observe(1) // Explored=1, sinceNew=0
		m := tr.observe(1)
		if m.IterationsSinceLastNew != 1 || m.Saturated {
			t.Fatalf("first repeat: %+v", m)
		}
		tr.observe(1) // sinceNew=2
		m = tr.observe(1)
		if m.IterationsSinceLastNew != 3 || !m.Saturated {
			t.Fatalf("expected saturation at threshold 3: %+v", m)
		}
	})

	t.Run("a new state resets the saturation counter", func(t *testing.T) {
		t.Parallel()
		tr := newStateSpaceTracker(2)
		tr.observe(1)
		tr.observe(1)
		m := tr.observe(1)
		if !m.Saturated {
			t.Fatalf("should be saturated: %+v", m)
		}
		m = tr.observe(2) // new state resets
		if m.Explored != 2 || m.IterationsSinceLastNew != 0 || m.Saturated {
			t.Fatalf("new state should reset saturation: %+v", m)
		}
	})

	t.Run("zero threshold uses the default", func(t *testing.T) {
		t.Parallel()
		tr := newStateSpaceTracker(0)
		if tr.threshold != defaultSaturationThreshold {
			t.Fatalf("threshold = %d, want default %d", tr.threshold, defaultSaturationThreshold)
		}
	})
}

func TestBuildREQToLaw(t *testing.T) {
	t.Parallel()

	read := func(rt *rapid.T, s storeCovIface, k string) (covItem, error) { return s.Get(rt.Context(), k) }

	t.Run("groups law IDs by REQ tag, sorted", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry[storeCovIface]()
		// Two laws under REQ-1, one under REQ-2, one untagged (excluded).
		r.Add(&taggedLaw[storeCovIface]{
			Law:   law.ReadAfterWrite[storeCovIface, string, covItem]{Read: read, Keys: rapid.Just("a")},
			reqID: "REQ-2",
		})
		r.Add(&taggedLaw[storeCovIface]{
			Law:   law.DeleteReturnsNotFound[storeCovIface, string, covItem]{Read: read, Keys: rapid.Just("a")},
			reqID: "REQ-1",
		})
		r.Add(&taggedLaw[storeCovIface]{
			Law: law.CountEqualsReference[storeCovIface, int]{
				Count: func(rt *rapid.T, s storeCovIface) (int, error) { return s.Count(rt.Context()) },
			},
			reqID: "REQ-1",
		})
		r.Add(law.ReadAfterWrite[storeCovIface, string, covItem]{Read: read, Keys: rapid.Just("a")}) // untagged

		got := buildREQToLaw(r)
		if len(got["REQ-1"]) != 2 {
			t.Fatalf("REQ-1 laws = %v", got["REQ-1"])
		}
		if got["REQ-1"][0] != "AUTO-COUNT-EQUALS-REFERENCE" || got["REQ-1"][1] != "AUTO-DELETE-RETURNS-NOT-FOUND" {
			t.Fatalf("REQ-1 not sorted: %v", got["REQ-1"])
		}
		if len(got["REQ-2"]) != 1 || got["REQ-2"][0] != "AUTO-READ-AFTER-WRITE" {
			t.Fatalf("REQ-2 = %v", got["REQ-2"])
		}
		if _, ok := got[""]; ok {
			t.Fatal("untagged law leaked into the matrix")
		}
	})

	t.Run("nil registry yields nil", func(t *testing.T) {
		t.Parallel()
		if buildREQToLaw[storeCovIface](nil) != nil {
			t.Fatal("expected nil for nil registry")
		}
	})
}
