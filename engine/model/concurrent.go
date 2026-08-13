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

	"go.thesmos.sh/testkit/core/trace"
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
			cc.Model, history, cc.Timeout,
		)
		switch result {
		case porcupine.Ok:
			// Linearizable — pass.
		case porcupine.Illegal:
			artifactDir := ResolveArtifactDir(cfg.ArtifactDir)
			vizPath := writeVisualization(rt, cc.Model, info, artifactDir)
			// Convert Porcupine history to trace events for the formatter.
			traceEvents := make([]trace.Event, len(history))
			for i, op := range history {
				input := op.Input.(OpInput)
				traceEvents[i] = trace.Event{
					StartNs:  op.Call,
					EndNs:    op.Return,
					Method:   input.Name,
					ClientID: op.ClientId,
					Inputs:   []any{input.Args},
					Output:   op.Output.(OpOutput).Result,
				}
			}
			f := &Failure{
				// SUT vs linearizable-reference mismatch is semantic.
				Kind:         FailureSemantic,
				StepRan:      StepID{WorkerID: -1, Index: 0},
				StepReported: StepID{WorkerID: -1, Index: 0},
				Err: fmt.Errorf("history is not linearizable (%d ops, %d workers)",
					len(history), cc.Workers),
				Trace: traceEvents,
			}
			if vizPath != "" {
				f.ArtifactPaths = append(f.ArtifactPaths, "viz: "+vizPath)
			}
			if jsonPath := emitClassifiedJSON(rt, cfg.ArtifactDir, f); jsonPath != "" {
				f.ArtifactPaths = append(f.ArtifactPaths, "json: "+jsonPath)
			}
			for _, p := range emitPerClientJSON(rt, cfg.ArtifactDir, f) {
				f.ArtifactPaths = append(f.ArtifactPaths, "json: "+p)
			}
			rt.Fatalf("%s", formatFailure(f))
		case porcupine.Unknown:
			// Undecided is not evidence: a run whose verdict never arrived
			// used to log a warning and pass, which is a linearizability
			// claim nothing checked. The timeout is the config's own knob —
			// a history this check cannot decide in its budget needs a
			// bigger budget or a smaller history, and either is the
			// caller's choice to make, not this run's to make silently.
			rt.Fatalf("linearizability undecided after %v — raise "+
				"ConcurrentConfig.Timeout or reduce the concurrent load; an "+
				"undecided run asserts nothing", cc.Timeout)
		}
	})
}

// writeVisualization writes a Porcupine visualization HTML file to
// the configured artifact directory. Returns the path or empty
// string on write failure (logged via rt.Logf). The path is reported
// via rt.Logf so the surrounding [Fatalf] message folds the path
// into the failure context.
//
// Naming follows the spec convention linearizability-<seed>.html
// where seed is the sanitized rapid test name. The parallel public
// helper [linearize.VisualizeOnFailure] writes the same shape for
// consumers driving Porcupine outside the runner; both writers
// share the naming convention so artifacts collected from either
// path are interchangeable for CI tooling.
func writeVisualization(rt rapid.TB, m porcupine.Model, info porcupine.LinearizationInfo, artifactDir string) string {
	if artifactDir == "" {
		return ""
	}
	if err := os.MkdirAll(artifactDir, 0o750); err != nil { //nolint:gosec // test artifacts
		rt.Logf("failed to create artifact dir: %v", err)
		return ""
	}
	seed := sanitizeForFilename(rt.Name())
	filename := "linearizability-" + seed + ".html"
	path := filepath.Join(artifactDir, filename)
	if err := porcupine.VisualizePath(m, info, path); err != nil {
		rt.Logf("failed to write artifact: %v", err)
		return ""
	}
	rt.Logf("linearizability viz: %s", path)
	return path
}
