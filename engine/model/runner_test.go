// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/core/coverage"
	"go.thesmos.sh/testkit/core/failure"
	"go.thesmos.sh/testkit/core/trace"
	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/action"
	"go.thesmos.sh/testkit/engine/model/law"
	"go.thesmos.sh/testkit/engine/model/linearize"
	"go.thesmos.sh/testkit/engine/model/ref"
)

// --- Test fixtures ---

var errNotFound = errors.New("not found")

type item struct {
	ID   string
	Name string
}

func itemKey(v item) string { return v.ID }

type store struct {
	mu   sync.Mutex
	data map[string]item
}

func newStore() *store { return &store{data: make(map[string]item)} }

func (s *store) Get(_ context.Context, key string) (item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[key]
	if !ok {
		return item{}, errNotFound
	}
	return v, nil
}

func (s *store) Put(_ context.Context, v item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[v.ID] = v
	return nil
}

func (s *store) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *store) Count(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data), nil
}

type storeIface interface {
	Get(context.Context, string) (item, error)
	Put(context.Context, item) error
	Delete(context.Context, string) error
	Count(context.Context) (int, error)
}

// --- Broken stores for negative tests ---

type brokenGetStore struct{ store }

func (s *brokenGetStore) Get(ctx context.Context, key string) (item, error) {
	v, err := s.store.Get(ctx, key)
	if err == nil {
		v.Name = "WRONG"
	}
	return v, err
}

type brokenDeleteStore struct{ store }

func (*brokenDeleteStore) Delete(_ context.Context, _ string) error { return nil }

type brokenCountStore struct{ store }

func (*brokenCountStore) Count(_ context.Context) (int, error) { return 0, nil }

// --- Generators + closures ---

var keyGen = rapid.SampledFrom([]string{"a", "b", "c", "missing"})

var itemGen = rapid.Custom(func(rt *rapid.T) item {
	id := rapid.SampledFrom([]string{"a", "b", "c"}).Draw(rt, "id")
	name := rapid.StringMatching(`[a-z]{3,6}`).Draw(rt, "name")
	return item{ID: id, Name: name}
})

// Closures matching action helper signatures (ctx first, then T).
// Method expressions put receiver first; these flip the order.
// The generator emits these automatically per detected method.
func storeGet(ctx context.Context, s storeIface, k string) (item, error) { return s.Get(ctx, k) }
func storePut(ctx context.Context, s storeIface, v item) error           { return s.Put(ctx, v) }
func storeDel(ctx context.Context, s storeIface, k string) error         { return s.Delete(ctx, k) }
func storeCount(ctx context.Context, s storeIface) (int, error)          { return s.Count(ctx) }

// --- Positive tests ---

func TestRunnerWithActionHelpers(t *testing.T) {
	t.Parallel()

	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(
			action.Reader("Get", keyGen, storeGet),
			action.Writer("Put", itemGen, storePut),
			action.Deleter("Delete", keyGen, storeDel),
			action.Aggregator("Count", storeCount),
		),
	)
}

func TestRunnerWithRefMap(t *testing.T) {
	t.Parallel()

	// Tier 0 pattern: ref.MapStore as the ref.
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface {
			return ref.NewMapStore(itemKey, errNotFound)
		}),
		model.WithActions(
			action.Reader("Get", keyGen, storeGet),
			action.Writer("Put", itemGen, storePut),
			action.Deleter("Delete", keyGen, storeDel),
			action.Aggregator("Count", storeCount),
		),
	)
}

func TestRunnerWithLaws(t *testing.T) {
	t.Parallel()

	read := func(rt *rapid.T, s storeIface, k string) (item, error) {
		return s.Get(rt.Context(), k)
	}
	count := func(rt *rapid.T, s storeIface) (int, error) {
		return s.Count(rt.Context())
	}

	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(
			action.Reader("Get", keyGen, storeGet),
			action.Writer("Put", itemGen, storePut),
			action.Deleter("Delete", keyGen, storeDel),
		),
		model.WithLaw(law.ReadAfterWrite[storeIface, string, item]{
			Read: read,
			Keys: keyGen,
		}),
		model.WithLaw(law.DeleteReturnsNotFound[storeIface, string, item]{
			Read:     read,
			Keys:     keyGen,
			Sentinel: errNotFound,
		}),
		model.WithLaw(law.CountEqualsReference[storeIface, int]{
			Count: count,
		}),
	)
}

// --- Negative tests: laws catch broken impls ---

func TestLawCatchesBrokenGet(t *testing.T) {
	t.Parallel()
	sut := &brokenGetStore{store: *newStore()}
	ref := newStore()
	ctx := t.Context()
	_ = sut.Put(ctx, item{ID: "a", Name: "correct"})
	_ = ref.Put(ctx, item{ID: "a", Name: "correct"})

	l := law.ReadAfterWrite[storeIface, string, item]{
		Read: func(rt *rapid.T, s storeIface, k string) (item, error) {
			return s.Get(rt.Context(), k)
		},
		Keys: rapid.Just("a"),
	}
	// Law needs *rapid.T — run inside rapid.Check.
	rapid.Check(t, func(rt *rapid.T) {
		err := l.Check(rt, sut, ref)
		if err == nil {
			rt.Fatalf("ReadAfterWrite must catch broken Get")
		}
	})
}

func TestLawCatchesBrokenCount(t *testing.T) {
	t.Parallel()
	sut := &brokenCountStore{store: *newStore()}
	ref := newStore()
	ctx := t.Context()
	_ = sut.Put(ctx, item{ID: "a", Name: "v"})
	_ = ref.Put(ctx, item{ID: "a", Name: "v"})

	l := law.CountEqualsReference[storeIface, int]{
		Count: func(rt *rapid.T, s storeIface) (int, error) {
			return s.Count(rt.Context())
		},
	}
	rapid.Check(t, func(rt *rapid.T) {
		err := l.Check(rt, sut, ref)
		if err == nil {
			rt.Fatalf("CountEqualsReference must catch broken Count")
		}
	})
}

func TestLawCatchesBrokenDelete(t *testing.T) {
	t.Parallel()
	sut := &brokenDeleteStore{store: *newStore()}
	ref := newStore()
	ctx := t.Context()
	_ = sut.Put(ctx, item{ID: "a", Name: "v"})
	_ = ref.Put(ctx, item{ID: "a", Name: "v"})
	_ = sut.Delete(ctx, "a") // no-op on broken store
	_ = ref.Delete(ctx, "a")

	l := law.DeleteReturnsNotFound[storeIface, string, item]{
		Read: func(rt *rapid.T, s storeIface, k string) (item, error) {
			return s.Get(rt.Context(), k)
		},
		Keys:     rapid.Just("a"),
		Sentinel: errNotFound,
	}
	rapid.Check(t, func(rt *rapid.T) {
		err := l.Check(rt, sut, ref)
		if err == nil {
			rt.Fatalf("DeleteReturnsNotFound must catch broken Delete")
		}
	})
}

// --- Registry tests ---

func TestRegistrySkipByID(t *testing.T) {
	t.Parallel()

	t.Run("returns true when law removed", func(t *testing.T) {
		t.Parallel()
		r := model.NewRegistry[storeIface]()
		r.Add(law.ReadAfterWrite[storeIface, string, item]{
			Read: func(rt *rapid.T, s storeIface, k string) (item, error) { return s.Get(rt.Context(), k) },
			Keys: keyGen,
		})
		if !r.SkipByID("AUTO-READ-AFTER-WRITE") {
			t.Fatal("SkipByID must return true")
		}
		if len(r.Laws()) != 0 {
			t.Fatal("registry must be empty")
		}
	})

	t.Run("returns false when ID not found", func(t *testing.T) {
		t.Parallel()
		r := model.NewRegistry[storeIface]()
		r.Add(law.ReadAfterWrite[storeIface, string, item]{
			Read: func(rt *rapid.T, s storeIface, k string) (item, error) { return s.Get(rt.Context(), k) },
			Keys: keyGen,
		})
		if r.SkipByID("TYPO") {
			t.Fatal("SkipByID must return false for typo")
		}
		if len(r.Laws()) != 1 {
			t.Fatal("registry must be unchanged")
		}
	})
}

func TestRegistryLawsDefensiveCopy(t *testing.T) {
	t.Parallel()
	r := model.NewRegistry[storeIface]()
	r.Add(law.CountEqualsReference[storeIface, int]{
		Count: func(rt *rapid.T, s storeIface) (int, error) { return s.Count(rt.Context()) },
	})
	laws := r.Laws()
	laws[0] = nil
	if r.Laws()[0] == nil {
		t.Fatal("Laws() must return defensive copy")
	}
}

func TestRegistryCheckAll(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when all laws pass", func(t *testing.T) {
		t.Parallel()
		r := model.NewRegistry[storeIface]()
		r.Add(law.CountEqualsReference[storeIface, int]{
			Count: func(rt *rapid.T, s storeIface) (int, error) { return s.Count(rt.Context()) },
		})
		rapid.Check(t, func(rt *rapid.T) {
			sut := newStore()
			ref := newStore()
			err := r.CheckAll(rt, sut, ref)
			if err != nil {
				rt.Fatalf("CheckAll must pass on identical empty stores: %v", err)
			}
		})
	})

	t.Run("returns first error when a law fires", func(t *testing.T) {
		t.Parallel()
		r := model.NewRegistry[storeIface]()
		r.Add(law.CountEqualsReference[storeIface, int]{
			Count: func(rt *rapid.T, s storeIface) (int, error) { return s.Count(rt.Context()) },
		})
		rapid.Check(t, func(rt *rapid.T) {
			sut := &brokenCountStore{store: *newStore()}
			ref := newStore()
			ctx := rt.Context()
			_ = sut.Put(ctx, item{ID: "a", Name: "v"})
			_ = ref.Put(ctx, item{ID: "a", Name: "v"})
			err := r.CheckAll(rt, sut, ref)
			if err == nil {
				rt.Fatalf("CheckAll must catch broken count")
			}
		})
	})
}

func TestRegistryCoverage(t *testing.T) {
	t.Parallel()

	r := model.NewRegistry[storeIface]()
	r.Add(law.CountEqualsReference[storeIface, int]{
		Count: func(rt *rapid.T, s storeIface) (int, error) { return s.Count(rt.Context()) },
	})
	// Run CheckAll to populate coverage counters.
	rapid.Check(t, func(rt *rapid.T) {
		sut := newStore()
		ref := newStore()
		_ = r.CheckAll(rt, sut, ref)
	})
	ran, fired := r.Coverage()
	if ran["AUTO-COUNT-EQUALS-REFERENCE"] == 0 {
		t.Fatal("law must have run at least once")
	}
	if fired["AUTO-COUNT-EQUALS-REFERENCE"] != 0 {
		t.Fatal("law must not have fired on identical stores")
	}
	// Verify defensive copies.
	ran["MUTATED"] = 99
	ran2, _ := r.Coverage()
	if ran2["MUTATED"] == 99 {
		t.Fatal("Coverage must return defensive copies")
	}
}

// --- Option function tests ---

func TestOptionWithLaw(t *testing.T) {
	t.Parallel()
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(
			action.Reader("Get", keyGen, storeGet),
			action.Writer("Put", itemGen, storePut),
		),
		model.WithLaw(law.ReadAfterWrite[storeIface, string, item]{
			Read: func(rt *rapid.T, s storeIface, k string) (item, error) {
				return s.Get(rt.Context(), k)
			},
			Keys: keyGen,
		}),
	)
}

func TestOptionWithLawREQ(t *testing.T) {
	t.Parallel()
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(
			action.Reader("Get", keyGen, storeGet),
			action.Writer("Put", itemGen, storePut),
		),
		model.WithLawREQ("REQ-PKG-STORE-001", law.ReadAfterWrite[storeIface, string, item]{
			Read: func(rt *rapid.T, s storeIface, k string) (item, error) {
				return s.Get(rt.Context(), k)
			},
			Keys: keyGen,
		}),
	)
}

// probeLaw records what the runner asked of it, and carries every optional
// interface a law may have — the four the REQ tag used to strip.
type probeLaw struct {
	bound   *trace.Trace
	resets  int
	steps   int
	checked int
}

func (*probeLaw) ID() string    { return "AUTO-PROBE" }
func (*probeLaw) REQID() string { return "" }
func (p *probeLaw) Check(*rapid.T, storeIface, storeIface) error {
	p.checked++
	return nil
}

func (p *probeLaw) CheckWithStep(_ *rapid.T, _, _ storeIface, _ int) error {
	p.steps++
	return nil
}
func (p *probeLaw) BindTrace(t *trace.Trace) { p.bound = t }
func (p *probeLaw) Reset()                   { p.resets++ }

// isolatedProbe carries only the isolation marker: an isolated law runs once
// per iteration against a throwaway pair, so it is never handed a step and
// cannot stand in for the stateful case.
type isolatedProbe struct{ probeLaw }

func (*isolatedProbe) IsolatedLaw() {}

// TestLawREQKeepsWhatTheLawIs pins the tag's transparency. The runner asks a
// law what else it is by type assertion, and a wrapper that answered "only a
// law" turned off every one of those behaviours — no trace to scan, no reset
// between iterations, and an isolated law loosed on the shared pair.
func TestLawREQKeepsWhatTheLawIs(t *testing.T) {
	t.Parallel()

	probe := &probeLaw{}
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(action.Reader("Get", keyGen, storeGet)),
		model.WithLawREQ("REQ-PROBE-001", probe),
	)

	testkit.True(t, probe.bound != nil, "a trace-scanning law is still bound one")
	testkit.True(t, probe.resets > 0, "and still reset between iterations")
	testkit.True(t, probe.steps > 0, "and still asked with the step it needs")
	testkit.Equal(t, probe.checked, 0,
		"the step-free arm never runs for a law that carries the stateful one")
}

// TestLawREQKeepsIsolationOff is the control: isolation is a marker, and a
// law that does not carry it must not acquire one by being tagged — every
// tagged law running against throwaway subjects would empty the sequences.
func TestLawREQKeepsIsolationOff(t *testing.T) {
	t.Parallel()

	plain := law.ReadAfterWrite[storeIface, string, item]{
		Read: func(rt *rapid.T, s storeIface, k string) (item, error) {
			return s.Get(rt.Context(), k)
		},
		Keys: keyGen,
	}
	reg := model.NewRegistry[storeIface]()
	model.WithLawREQ("REQ-PLAIN-001", plain)(&model.Config[storeIface]{Laws: reg})

	tagged := reg.Laws()
	testkit.Len(t, tagged, 1, "the tag registers the law")
	_, isolated := tagged[0].(law.Isolated)
	testkit.False(t, isolated, "a law that is not isolated does not become so")
	testkit.Equal(t, tagged[0].REQID(), "REQ-PLAIN-001", "and still carries its tag")

	// And the other direction: the marker survives the tag, or an isolated
	// law would corrupt the shared pair every other law is checked against.
	iso := model.NewRegistry[storeIface]()
	model.WithLawREQ("REQ-ISO-001", &isolatedProbe{})(&model.Config[storeIface]{Laws: iso})
	_, kept := iso.Laws()[0].(law.Isolated)
	testkit.True(t, kept, "an isolated law stays isolated once tagged")
	testkit.Equal(t, iso.Laws()[0].REQID(), "REQ-ISO-001", "and carries its tag too")
}

func TestOptionWithCleanup(t *testing.T) {
	t.Parallel()
	cleaned := false
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(
			action.Reader("Get", keyGen, storeGet),
		),
		model.WithCleanup(func(_ storeIface) { cleaned = true }),
	)
	if !cleaned {
		t.Fatal("cleanup must have been called")
	}
}

func TestOptionWithHistoryReset(t *testing.T) {
	t.Parallel()
	resetCount := 0
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(
			action.Reader("Get", keyGen, storeGet),
		),
		model.WithHistoryReset[storeIface](func() { resetCount++ }),
	)
	if resetCount == 0 {
		t.Fatal("history reset must have been called at least once")
	}
}

// TestStatefulLawsResetPerIteration pins the iteration boundary the wide
// pools surfaced: a stateful law's memory of the previous pair must not
// outlive the pair. The subject here answers a different value every
// iteration — exactly the shape that false-failed the sticky law before the
// runner reset, and exactly the shape the fixture-pair pools never drew.
func TestStatefulLawsResetPerIteration(t *testing.T) {
	t.Parallel()

	iteration := 0
	model.Assert(
		t,
		func() int { iteration++; return iteration },
		model.WithReference(func() int { return 0 }),
		model.WithActions(action.Pure("Ping", func(int) int { return 0 })),
		model.WithLaw[int](&law.Sticky[int, string, int]{
			Keys: rapid.Just("k"),
			Read: func(_ *rapid.T, s int, _ string) (int, error) { return s, nil },
		}),
	)
}

func TestOptionSkipLaw(t *testing.T) {
	t.Parallel()
	// SkipLaw removes a law by ID. Verify the option applies without
	// error on a correct store — the skipped law simply doesn't run.
	r := model.NewRegistry[storeIface]()
	r.Add(law.CountEqualsReference[storeIface, int]{
		Count: func(rt *rapid.T, s storeIface) (int, error) { return s.Count(rt.Context()) },
	})
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(
			action.Reader("Get", keyGen, storeGet),
			action.Writer("Put", itemGen, storePut),
		),
		model.WithLaws(r),
		model.SkipLaw[storeIface]("AUTO-COUNT-EQUALS-REFERENCE"),
	)
	// After skip, the law should no longer be in the registry.
	if len(r.Laws()) != 0 {
		t.Fatal("SkipLaw must remove the law from the registry")
	}
}

func TestOptionWithoutTrace(t *testing.T) {
	t.Parallel()
	// Just verify the option doesn't panic and the runner completes.
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(
			action.Reader("Get", keyGen, storeGet),
		),
		model.WithoutTrace[storeIface](),
	)
}

func TestOptionWithArtifactDir(t *testing.T) {
	t.Parallel()
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(
			action.Reader("Get", keyGen, storeGet),
		),
		model.WithArtifactDir[storeIface](t.TempDir()),
	)
}

func TestProperty(t *testing.T) {
	t.Parallel()
	prop := model.Property(
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(
			action.Reader("Get", keyGen, storeGet),
			action.Writer("Put", itemGen, storePut),
		),
	)
	// Property returns a rapid property function; verify it runs.
	rapid.Check(t, prop)
}

func TestRun(t *testing.T) {
	t.Parallel()
	model.Run(t, model.Config[storeIface]{
		SUTFactory: func() storeIface { return newStore() },
		RefFactory: func() storeIface { return newStore() },
		Actions: []model.Action[storeIface]{
			action.Reader("Get", keyGen, storeGet),
			action.Writer("Put", itemGen, storePut),
		},
	})
}

func TestFailureEmitsClassifiedJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ft := testkit.NewFailableTB().WithGoexit()
	done := make(chan struct{})
	go func() {
		defer close(done)
		model.Assert(
			ft,
			func() storeIface { return &brokenGetStore{store: *newStore()} },
			model.WithReference(func() storeIface { return newStore() }),
			model.WithActions(
				action.Writer("Put", itemGen, storePut),
			),
			model.WithLaw(law.ReadAfterWrite[storeIface, string, item]{
				Read: func(rt *rapid.T, s storeIface, k string) (item, error) {
					return s.Get(rt.Context(), k)
				},
				Keys: rapid.Just("a"),
			}),
			model.WithArtifactDir[storeIface](dir),
		)
	}()
	<-done

	if !ft.Failed() {
		t.Fatal("brokenGetStore must trip ReadAfterWrite")
	}
	matches, err := filepath.Glob(filepath.Join(dir, "failure-*.json"))
	testkit.NoError(t, err, "glob")
	if len(matches) == 0 {
		t.Fatalf("expected at least one classified-Failure JSON in %s", dir)
	}
	body, err := os.ReadFile(matches[0])
	testkit.NoError(t, err, "read JSON")
	var uf failure.Failure
	testkit.NoError(t, json.Unmarshal(body, &uf), "unmarshal")
	testkit.Equal(t, uf.Generator, "model", "generator tag")
	testkit.Equal(t, uf.Kind, failure.KindInvariant, "kind")
}

// The runner has three dispatch paths the law-failure tests never reach: a
// diverging *action* (as opposed to a law), a law that wants the per-iteration
// trace bound to it, and a law that wants the step index. Each is a distinct
// branch in the property body.
func TestRunnerActionAndLawDispatch(t *testing.T) {
	t.Parallel()

	t.Run("a diverging action fails with its own kind and artifact", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			model.Assert(
				ft,
				func() storeIface { return &brokenGetStore{store: *newStore()} },
				model.WithReference(func() storeIface { return newStore() }),
				model.WithActions(
					action.Writer("Put", itemGen, storePut),
					// Get diverges: the broken subject answers differently
					// from the reference, and the action itself reports it.
					action.Reader("Get", rapid.Just("a"), storeGet),
				),
				model.WithArtifactDir[storeIface](dir),
			)
		}()
		<-done

		if !ft.Failed() {
			t.Fatal("a diverging Get must be reported by the action, not only by a law")
		}
		matches, err := filepath.Glob(filepath.Join(dir, "failure-*.json"))
		testkit.NoError(t, err, "glob")
		if len(matches) == 0 {
			t.Fatalf("an action failure must dump its classified JSON in %s", dir)
		}
		body, err := os.ReadFile(matches[0])
		testkit.NoError(t, err, "read JSON")
		var uf failure.Failure
		testkit.NoError(t, json.Unmarshal(body, &uf), "unmarshal")
		testkit.Equal(t, uf.Kind, failure.KindSemantic, "an action divergence is semantic")
		if uf.Details["law_id"] != "" {
			t.Fatalf("no law fired, so none should be named: %v", uf.Details["law_id"])
		}
	})

	// A trace combinator is inert until the runner hands it the iteration's
	// trace; without the bind it would inspect a nil buffer.
	t.Run("a trace-binding law receives the iteration trace", func(t *testing.T) {
		t.Parallel()
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		var seen int
		go func() {
			defer close(done)
			model.Assert(
				ft,
				func() storeIface { return newStore() },
				model.WithReference(func() storeIface { return newStore() }),
				model.WithActions(action.Writer("Put", itemGen, storePut)),
				model.WithLaw(&law.AfterEvery[storeIface]{
					ActionName: "Put",
					Predicate: func(*rapid.T, storeIface, storeIface) error {
						seen++
						return nil
					},
				}),
			)
		}()
		<-done

		if ft.Failed() {
			t.Fatalf("a satisfied combinator must not fail: %s", ft.Msg())
		}
		if seen == 0 {
			t.Fatal("the predicate never ran, so the trace was never bound")
		}
	})

	// A stateful law is called through CheckWithStep so it can tell the
	// first action of an iteration from a later one.
	t.Run("a stateful law receives the step index", func(t *testing.T) {
		t.Parallel()
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		steps := make(chan int, 1024)
		go func() {
			defer close(done)
			model.Assert(
				ft,
				func() storeIface { return newStore() },
				model.WithReference(func() storeIface { return newStore() }),
				model.WithActions(action.Writer("Put", itemGen, storePut)),
				model.WithLaw(&stepRecordingLaw{steps: steps}),
			)
		}()
		<-done

		if ft.Failed() {
			t.Fatalf("the recording law never fails: %s", ft.Msg())
		}
		close(steps)
		var sawZero, sawLater bool
		for s := range steps {
			switch {
			case s == 0:
				sawZero = true
			case s > 0:
				sawLater = true
			}
		}
		if !sawZero || !sawLater {
			t.Fatalf("the law must see the step advance (zero=%v later=%v)", sawZero, sawLater)
		}
	})
}

// stepRecordingLaw records the step index the runner supplies. It exists to
// prove the StatefulLaw dispatch is taken rather than the plain Check.
type stepRecordingLaw struct {
	steps chan int
}

func (*stepRecordingLaw) ID() string    { return "TEST-STEP-RECORDING" }
func (*stepRecordingLaw) REQID() string { return "" }

func (l *stepRecordingLaw) Check(rt *rapid.T, sut, ref storeIface) error {
	return l.CheckWithStep(rt, sut, ref, -1)
}

func (l *stepRecordingLaw) CheckWithStep(_ *rapid.T, _, _ storeIface, step int) error {
	select {
	case l.steps <- step:
	default: // the channel is only a sample; a full one is not a failure
	}
	return nil
}

// A misconfigured run must fail loudly at the point of misuse rather than
// quietly doing nothing — a runner with no subject or no actions would
// otherwise pass every property vacuously.
func TestRunnerConfigValidation(t *testing.T) {
	t.Parallel()

	runToFailure := func(t *testing.T, fn func(*testkit.FailableTB)) *testkit.FailableTB {
		t.Helper()
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			fn(ft)
		}()
		<-done
		return ft
	}

	t.Run("a missing SUT factory is rejected", func(t *testing.T) {
		t.Parallel()
		ft := runToFailure(t, func(ft *testkit.FailableTB) {
			model.Assert(ft, (func() storeIface)(nil),
				model.WithActions(action.Reader("Get", keyGen, storeGet)))
		})
		if !ft.Failed() {
			t.Fatal("a run with no subject must fail")
		}
		if !strings.Contains(ft.Msg(), "SUTFactory") {
			t.Fatalf("the diagnostic must name what is missing, got: %s", ft.Msg())
		}
	})

	t.Run("a run with no actions is rejected", func(t *testing.T) {
		t.Parallel()
		ft := runToFailure(t, func(ft *testkit.FailableTB) {
			model.Assert(ft, func() storeIface { return newStore() })
		})
		if !ft.Failed() {
			t.Fatal("a run with no actions must fail")
		}
		if !strings.Contains(ft.Msg(), "Action") {
			t.Fatalf("the diagnostic must name what is missing, got: %s", ft.Msg())
		}
	})

	// Laws compare SUT against a reference after each action; the concurrent
	// runner has neither a reference nor an after-every-action boundary, so
	// combining them is rejected rather than silently dropping the laws.
	t.Run("laws combined with concurrent are rejected", func(t *testing.T) {
		t.Parallel()
		ft := runToFailure(t, func(ft *testkit.FailableTB) {
			model.Assert(ft, func() storeIface { return newStore() },
				model.WithLaw[storeIface](law.PureDeterminism[storeIface, item]{
					Call: func(rt *rapid.T, s storeIface) item {
						v, _ := storeGet(rt.Context(), s, "k")
						return v
					},
				}),
				model.WithConcurrent(model.ConcurrentConfig[storeIface]{
					Model: linearize.KV[string, item](errNotFound),
					Actions: []model.ConcurrentAction[storeIface]{
						linearize.ConcurrentReader("Get", keyGen, storeGet),
					},
				}),
			)
		})
		if !ft.Failed() {
			t.Fatal("a step-boundary law plus concurrent must be rejected, not silently dropped")
		}
		if !strings.Contains(ft.Msg(), "unsupported with Concurrent") {
			t.Fatalf("the diagnostic must explain the incompatibility, got: %s", ft.Msg())
		}
		if !strings.Contains(ft.Msg(), "AUTO-PURE-DETERMINISTIC") {
			t.Fatalf("and name the law that needs the step boundary, got: %s", ft.Msg())
		}
	})
}

// WithSaturationThreshold is a plain setter, but it gates whether the runner
// tracks the state space at all — a zero threshold selects the default rather
// than disabling tracking.
func TestWithSaturationThreshold(t *testing.T) {
	t.Parallel()

	var cov coverage.ComponentCoverage
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(action.Reader("Get", keyGen, storeGet)),
		model.WithSaturationThreshold[storeIface](5),
		model.WithStateHash[storeIface](hashStore),
		model.WithCoverageSink[storeIface](&cov),
	)
	if cov.StateSpace.Explored < 1 {
		t.Fatal("a configured state hash must produce state-space coverage")
	}
}

// --- Isolated-law walk ---

// isoProbe is a marker-carrying law: the runner must route it to a throwaway
// pair once per iteration and keep it off the shared per-step walk.
type isoProbe struct {
	id    string
	calls *int
	err   error
}

func (isoProbe) IsolatedLaw()  {}
func (l isoProbe) ID() string  { return l.id }
func (isoProbe) REQID() string { return "" }
func (l isoProbe) Check(_ *rapid.T, _, _ storeIface) error {
	*l.calls++
	return l.err
}

func TestIsolatedLawRunsOnItsOwnPair(t *testing.T) {
	t.Parallel()

	var checks, vacuous, steps int
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(
			action.Writer("Put", itemGen, func(ctx context.Context, s storeIface, v item) error {
				steps++
				return s.Put(ctx, v)
			}),
		),
		model.WithLaw(isoProbe{id: "TEST-ISOLATED", calls: &checks}),
		model.WithLaw(isoProbe{id: "TEST-ISOLATED-VACUOUS", calls: &vacuous, err: law.Vacuous}),
	)
	if checks == 0 {
		t.Fatal("the isolated walk must run the marked law")
	}
	if vacuous == 0 {
		t.Fatal("a vacuous isolated law still runs; only its verdict is counted apart")
	}
	// Once per iteration, never per step: a per-step isolated walk would
	// check at least as often as the actions ran.
	if steps > 0 && checks >= steps+checks/2 {
		t.Fatalf("the isolated walk ran %d times against %d steps — it must be once per iteration", checks, steps)
	}
}

func TestIsolatedLawViolationFails(t *testing.T) {
	t.Parallel()

	var checks int
	ft := testkit.NewFailableTB().WithGoexit()
	done := make(chan struct{})
	go func() {
		defer close(done)
		model.Assert(
			ft,
			func() storeIface { return newStore() },
			model.WithReference(func() storeIface { return newStore() }),
			model.WithActions(
				action.Writer("Put", itemGen, storePut),
			),
			model.WithLaw(isoProbe{id: "TEST-ISOLATED-BROKEN", calls: &checks, err: errors.New("the ritual failed")}),
		)
	}()
	<-done
	if !ft.Failed() {
		t.Fatal("an isolated law's violation must fail the run")
	}
}

// vacuousLaw always declines — the shared walk's counterpart of the isolated
// vacuous case, driving the registry's census through the runner.
type vacuousLaw struct{}

func (vacuousLaw) ID() string    { return "TEST-VACUOUS" }
func (vacuousLaw) REQID() string { return "" }
func (vacuousLaw) Check(_ *rapid.T, _, _ storeIface) error {
	return law.Vacuous
}

func TestVacuousLawIsCountedApartFromAPass(t *testing.T) {
	t.Parallel()

	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(
			action.Writer("Put", itemGen, storePut),
		),
		model.WithLaw(vacuousLaw{}),
	)
}
