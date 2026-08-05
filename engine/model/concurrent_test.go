// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/linearize"
)

func TestConcurrentLinearizable(t *testing.T) {
	t.Parallel()
	// Correct mutex-guarded store must be linearizable under concurrent access.
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithConcurrent(model.ConcurrentConfig[storeIface]{
			Workers:      4,
			OpsPerWorker: 30,
			Model:        linearize.KV[string, item](errNotFound),
			Actions: []model.ConcurrentAction[storeIface]{
				linearize.ConcurrentReader("Get", keyGen, storeGet),
				linearize.ConcurrentWriter("Put", itemGen, storePut, itemKey),
				linearize.ConcurrentDeleter("Delete", keyGen, storeDel),
			},
		}),
	)
}

// A linearizability search that runs out of time proves nothing either way,
// so it degrades to a logged warning rather than a failure.
//
// The subject is the correct store: a wrong one could finish the search and
// report Illegal before the budget expires, which is a different verdict.
// Slowing the *model* rather than shrinking the budget is what makes the
// timeout certain — a nanosecond budget alone races the checker.
func TestConcurrentCheckTimeoutIsAWarning(t *testing.T) {
	t.Parallel()

	slow := linearize.KV[string, item](errNotFound)
	inner := slow.Step
	slow.Step = func(state, input, output any) (bool, any) {
		time.Sleep(200 * time.Microsecond)
		return inner(state, input, output)
	}

	ft := testkit.NewFailableTB().WithGoexit()
	done := make(chan struct{})
	go func() {
		defer close(done)
		model.Assert(
			ft,
			func() storeIface { return newStore() },
			model.WithConcurrent(model.ConcurrentConfig[storeIface]{
				Workers:      2,
				OpsPerWorker: 3,
				Timeout:      time.Nanosecond,
				Model:        slow,
				Actions: []model.ConcurrentAction[storeIface]{
					linearize.ConcurrentReader("Get", keyGen, storeGet),
					linearize.ConcurrentWriter("Put", itemGen, storePut, itemKey),
				},
			}),
		)
	}()
	<-done

	if ft.Failed() {
		t.Fatalf("an inconclusive check must not fail the test: %s", ft.Msg())
	}
}

func TestConcurrentNotLinearizable(t *testing.T) {
	t.Parallel()
	// nonLinearizableStore is thread-safe (mutex-guarded) but not
	// linearizable (Get reads stale snapshot). Safe to run under -race.
	ft := testkit.NewFailableTB().WithGoexit()
	done := make(chan struct{})
	go func() {
		defer close(done)
		model.Assert(
			ft,
			func() storeIface { return newNonLinearizableStore() },
			model.WithConcurrent(model.ConcurrentConfig[storeIface]{
				Workers:      4,
				OpsPerWorker: 30,
				Model:        linearize.KV[string, item](errNotFound),
				Actions: []model.ConcurrentAction[storeIface]{
					linearize.ConcurrentReader("Get", keyGen, storeGet),
					linearize.ConcurrentWriter("Put", itemGen, storePut, itemKey),
					linearize.ConcurrentDeleter("Delete", keyGen, storeDel),
				},
			}),
		)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("concurrent negative test timed out")
	}
	if !ft.Failed() {
		t.Fatal("racy store should fail linearizability check")
	}
}

// nonLinearizableStore uses a mutex but has a bug: Get returns a
// stale snapshot taken at construction time, not the current value.
// This is thread-safe but not linearizable — a Put followed by Get
// returns the old value.
type nonLinearizableStore struct {
	mu       sync.Mutex
	data     map[string]item
	snapshot map[string]item // stale snapshot
}

func newNonLinearizableStore() *nonLinearizableStore {
	return &nonLinearizableStore{
		data:     make(map[string]item),
		snapshot: make(map[string]item),
	}
}

func (s *nonLinearizableStore) Get(_ context.Context, key string) (item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// BUG: reads from stale snapshot, not current data
	v, ok := s.snapshot[key]
	if !ok {
		return item{}, errNotFound
	}
	return v, nil
}

func (s *nonLinearizableStore) Put(_ context.Context, v item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[v.ID] = v
	// BUG: snapshot is NEVER updated — reads always return stale data
	return nil
}

func (s *nonLinearizableStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *nonLinearizableStore) Count(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data), nil
}

// The concurrent runner's configuration tunables all default when left unset,
// so a caller who supplies only a Model and one Action still gets a real run
// rather than zero workers doing nothing.
func TestConcurrentConfigDefaults(t *testing.T) {
	t.Parallel()

	var ops atomic.Int64
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithConcurrent(model.ConcurrentConfig[storeIface]{
			// Workers, OpsPerWorker and Timeout deliberately unset.
			Model: linearize.KV[string, item](errNotFound),
			Actions: []model.ConcurrentAction[storeIface]{
				{
					Name: "Get",
					Gen:  func(rt *rapid.T) any { ops.Add(1); return keyGen.Draw(rt, "k") },
					Apply: func(ctx context.Context, s storeIface, in any) any {
						v, err := storeGet(ctx, s, in.(string))
						return linearize.ReaderResult[item]{Value: v, Err: err}
					},
					PartitionKey: func(in any) string { return in.(string) },
				},
			},
		}),
	)
	if ops.Load() == 0 {
		t.Fatal("the defaulted worker and op counts must produce actual operations")
	}
}

// StressActions run alongside the recorded workers and are deliberately not
// part of the linearizability history — their job is to create contention for
// the race detector.
func TestConcurrentStressActionsRun(t *testing.T) {
	t.Parallel()

	var stressed atomic.Int64
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithConcurrent(model.ConcurrentConfig[storeIface]{
			Workers:      2,
			OpsPerWorker: 5,
			Model:        linearize.KV[string, item](errNotFound),
			Actions: []model.ConcurrentAction[storeIface]{
				linearize.ConcurrentReader("Get", keyGen, storeGet),
			},
			StressActions: []model.Action[storeIface]{
				{
					Name: "stress",
					Run: func(rt *rapid.T, sut, _ storeIface) model.ActionResult {
						stressed.Add(1)
						_, _ = sut.Get(rt.Context(), "k")
						return model.ActionResult{}
					},
				},
			},
		}),
	)
	if stressed.Load() == 0 {
		t.Fatal("stress actions must run alongside the recorded workers")
	}
}

// A config carrying only stress actions records no history, so there is
// nothing for porcupine to check — the runner must return cleanly rather than
// hand an empty history to the checker.
func TestConcurrentStressOnlyRecordsNoHistory(t *testing.T) {
	t.Parallel()

	var stressed atomic.Int64
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithConcurrent(model.ConcurrentConfig[storeIface]{
			Workers:      2,
			OpsPerWorker: 3,
			Model:        linearize.KV[string, item](errNotFound),
			StressActions: []model.Action[storeIface]{
				{
					Name: "stress",
					Run: func(rt *rapid.T, sut, _ storeIface) model.ActionResult {
						stressed.Add(1)
						_, _ = sut.Get(rt.Context(), "k")
						return model.ActionResult{}
					},
				},
			},
		}),
	)
	if stressed.Load() == 0 {
		t.Fatal("stress-only configs must still run their actions")
	}
}

// Cleanup runs for the SUT and, when a reference factory is configured, for
// the reference too — a leaked reference is as bad as a leaked subject.
func TestConcurrentCleanupCoversSUTAndRef(t *testing.T) {
	t.Parallel()

	var cleanups atomic.Int64
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithCleanup(func(storeIface) { cleanups.Add(1) }),
		model.WithConcurrent(model.ConcurrentConfig[storeIface]{
			Workers:      2,
			OpsPerWorker: 3,
			Model:        linearize.KV[string, item](errNotFound),
			Actions: []model.ConcurrentAction[storeIface]{
				linearize.ConcurrentReader("Get", keyGen, storeGet),
			},
		}),
	)
	if cleanups.Load() < 2 {
		t.Fatalf("both SUT and reference must be cleaned up, got %d", cleanups.Load())
	}
}

// A config with neither Actions nor StressActions cannot test anything, and
// saying so is more useful than running an empty loop and passing.
func TestConcurrentRequiresAtLeastOneAction(t *testing.T) {
	t.Parallel()

	ft := testkit.NewFailableTB().WithGoexit()
	done := make(chan struct{})
	go func() {
		defer close(done)
		model.Assert(
			ft,
			func() storeIface { return newStore() },
			model.WithConcurrent(model.ConcurrentConfig[storeIface]{
				Model: linearize.KV[string, item](errNotFound),
			}),
		)
	}()
	<-done
	if !ft.Failed() {
		t.Fatal("a concurrent config with no actions must fail loudly")
	}
	if !strings.Contains(ft.Msg(), "Action") {
		t.Fatalf("the diagnostic must say what is missing, got: %s", ft.Msg())
	}
}
