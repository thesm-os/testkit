// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anishathalye/porcupine"
	"pgregory.net/rapid"
)

// runConcurrent executes concurrent linearizability testing.
//
// Three phases per rapid iteration:
//  1. Pre-draw: all inputs drawn in a single goroutine (rapid safe).
//  2. Execute: workers call Apply concurrently; stress actions run in parallel.
//  3. Check: Porcupine validates the recorded history.
func runConcurrent[T any](t rapid.TB, cfg Config[T]) {
	t.Helper()

	cc := cfg.Concurrent
	if cc.Workers <= 0 {
		cc.Workers = 4
	}
	if cc.OpsPerWorker <= 0 {
		cc.OpsPerWorker = 50
	}
	if cc.Timeout == 0 {
		cc.Timeout = 10 * time.Second
	}
	if len(cc.Actions) == 0 && len(cc.StressActions) == 0 {
		t.Fatal("model.runConcurrent: at least one Action or StressAction required")
	}

	rapid.Check(t, func(rt *rapid.T) {
		sut := cfg.SUTFactory()

		// Ref needed for stress actions (they call both SUT and ref).
		var ref T
		if cfg.RefFactory != nil {
			ref = cfg.RefFactory()
		}
		if cfg.Cleanup != nil {
			defer cfg.Cleanup(sut)
			if cfg.RefFactory != nil {
				defer cfg.Cleanup(ref)
			}
		}

		// Phase 1: Pre-draw all inputs (single goroutine — rapid safe).
		type scheduledOp struct {
			action  ConcurrentAction[T]
			input   any
			partKey string
		}

		var schedule [][]scheduledOp
		if len(cc.Actions) > 0 {
			schedule = make([][]scheduledOp, cc.Workers)
			for w := range cc.Workers {
				schedule[w] = make([]scheduledOp, cc.OpsPerWorker)
				for i := range cc.OpsPerWorker {
					idx := rapid.IntRange(0, len(cc.Actions)-1).Draw(rt, "action_idx")
					act := cc.Actions[idx]
					input := act.Gen(rt)
					partKey := ""
					if act.PartitionKey != nil {
						partKey = act.PartitionKey(input)
					}
					schedule[w][i] = scheduledOp{
						action: act, input: input, partKey: partKey,
					}
				}
			}
		}

		// Phase 2: Execute concurrently.
		var (
			mu      sync.Mutex
			history []porcupine.Operation
			clock   atomic.Int64
			wg      sync.WaitGroup
		)

		// Linearizability workers — recorded to Porcupine history.
		for clientID, ops := range schedule {
			wg.Go(func() {
				for _, op := range ops {
					callTS := clock.Add(1)
					result := op.action.Apply(rt.Context(), sut, op.input)
					returnTS := clock.Add(1)
					pOp := porcupine.Operation{
						ClientId: clientID,
						Input:    OpInput{Name: op.action.Name, PartitionKey: op.partKey, Args: op.input},
						Output:   OpOutput{Result: result},
						Call:     callTS,
						Return:   returnTS,
						Metadata: fmt.Sprintf("%s(%v)", op.action.Name, op.input),
					}
					mu.Lock()
					history = append(history, pOp)
					mu.Unlock()
				}
			})
		}

		// Stress workers — run in parallel, NOT recorded.
		// Purpose: race detection under -race. StressActions should use
		// action.Stress which only calls the SUT without comparison.
		for _, sa := range cc.StressActions {
			wg.Go(func() {
				for range cc.OpsPerWorker {
					sa.Run(rt, sut, ref)
				}
			})
		}

		wg.Wait()

		// Phase 3: Check linearizability (skip if no linearizable ops).
		if len(history) == 0 {
			return
		}
		result, info := porcupine.CheckOperationsVerbose(
			cc.Model, history, cc.Timeout)
		switch result {
		case porcupine.Ok:
			// Linearizable — pass.
		case porcupine.Illegal:
			vizPath := writeVisualization(cc.Model, info)
			rt.Fatalf("history is not linearizable (%d ops, %d workers)\n  visualization: %s",
				len(history), cc.Workers, vizPath)
		case porcupine.Unknown:
			rt.Logf("linearizability check timed out (%v) — treating as warning",
				cc.Timeout)
		}
	})
}

// writeVisualization writes a Porcupine visualization HTML file and
// returns the path. Unique per invocation to avoid collisions.
func writeVisualization(model porcupine.Model, info porcupine.LinearizationInfo) string {
	path := filepath.Join(os.TempDir(),
		fmt.Sprintf("linearizability-%d.html", time.Now().UnixNano()))
	err := porcupine.VisualizePath(model, info, path)
	if err != nil {
		return "(failed: " + err.Error() + ")"
	}
	return path
}
