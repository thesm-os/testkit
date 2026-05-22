// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/failure"
	"go.thesmos.sh/testkit/model"
	"go.thesmos.sh/testkit/model/action"
	"go.thesmos.sh/testkit/model/law"
	"go.thesmos.sh/testkit/model/refmap"
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

	// Tier 0 pattern: refmap.MapStore as the reference.
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface {
			return refmap.NewMapStore(itemKey, errNotFound)
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
