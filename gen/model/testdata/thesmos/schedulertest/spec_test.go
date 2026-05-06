// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package schedulertest_test

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/model/testdata/thesmos"
	"go.thesmos.sh/testkit/gen/model/testdata/thesmos/schedulertest"
	"go.thesmos.sh/testkit/model"
	"go.thesmos.sh/testkit/model/action"
)

// readyRequestGen generates a random DAG with 3-8 vertices.
var readyRequestGen = rapid.Custom(func(rt *rapid.T) thesmos.ReadyRequest {
	n := rapid.IntRange(3, 8).Draw(rt, "vertex_count")
	verts := make(map[thesmos.VertexID]thesmos.VertexState, n)
	deps := make(map[thesmos.VertexID][]thesmos.VertexID, n)
	ids := make([]thesmos.VertexID, n)
	for i := range n {
		id := thesmos.VertexID(rapid.StringMatching(`v[0-9]`).Draw(rt, "vid"))
		ids[i] = id
		state := thesmos.VertexState(rapid.IntRange(0, 2).Draw(rt, "vstate"))
		verts[id] = state
		// Deps only point to earlier vertices (ensures DAG, no cycles).
		if i > 0 {
			depCount := rapid.IntRange(0, min(i, 2)).Draw(rt, "dep_count")
			for range depCount {
				depIdx := rapid.IntRange(0, i-1).Draw(rt, "dep_idx")
				deps[id] = append(deps[id], ids[depIdx])
			}
		}
	}
	return thesmos.ReadyRequest{Vertices: verts, Deps: deps}
})

// readyAction is a consumer-supplied action for the Pure-with-params
// Ready method. The framework can't auto-derive this because the
// method takes a complex request parameter.
var readyAction = action.Unknown[thesmos.Scheduler]("Ready",
	func(rt *rapid.T, sut, ref thesmos.Scheduler) model.ActionResult {
		req := readyRequestGen.Draw(rt, "req")
		sutResult := sut.Ready(req)
		refResult := ref.Ready(req)
		if len(sutResult.Ready) != len(refResult.Ready) {
			return model.ActionResult{Err: fmt.Errorf("Ready: SUT has %d ready, ref has %d",
				len(sutResult.Ready), len(refResult.Ready))}
		}
		for i, v := range sutResult.Ready {
			if v != refResult.Ready[i] {
				return model.ActionResult{Err: fmt.Errorf("Ready[%d]: SUT=%s, ref=%s", i, v, refResult.Ready[i])}
			}
		}
		return model.ActionResult{}
	},
)

func TestSchedulerModel(t *testing.T) {
	t.Parallel()

	t.Run("map SUT vs filter ref", func(t *testing.T) {
		t.Parallel()
		// Different algorithms: MapScheduler (iterate + check deps)
		// vs FilterScheduler (build complete set + filter).
		schedulertest.AssertSchedulerModel(t,
			func() thesmos.Scheduler { return thesmos.NewMapScheduler() },
			schedulertest.SchedulerModelReference(func() thesmos.Scheduler {
				return thesmos.NewFilterScheduler()
			}),
			schedulertest.SchedulerModelExtraActions(readyAction),
		)
	})

	t.Run("catches broken dependency check", func(t *testing.T) {
		t.Parallel()
		// Negative: BrokenScheduler only checks first dep.
		// With a vertex that has 2+ deps where only the first is complete,
		// the broken impl wrongly reports it as ready.
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			schedulertest.AssertSchedulerModel(ft,
				func() thesmos.Scheduler { return thesmos.NewBrokenScheduler() },
				schedulertest.SchedulerModelReference(func() thesmos.Scheduler {
					return thesmos.NewFilterScheduler()
				}),
				schedulertest.SchedulerModelExtraActions(readyAction),
			)
		}()
		<-done
		if !ft.Failed() {
			t.Fatal("framework should have caught broken dependency check")
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
