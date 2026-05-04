// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package repotest_test

import (
	"context"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen/model/testdata/generic"
	"go.thesmos.sh/testkit/gen/model/testdata/generic/repotest"
)

func TestInMemoryRepositoryModel(t *testing.T) {
	t.Parallel()

	t.Run("monomorphic via type alias", func(t *testing.T) {
		t.Parallel()
		// Repository[string, Item] instantiated via type alias.
		// Codegen-time keyfield heuristic finds Item.ID.
		// Auto refmap synthesis works because V is concrete.
		repotest.AssertItemRepositoryModel(t, func() generic.ItemRepository {
			return generic.NewInMemoryRepository()
		})
	})

	t.Run("parameterized emission instantiated at call site", func(t *testing.T) {
		t.Parallel()
		// AssertRepositoryModel[string, generic.Item] — parameterized
		// function instantiated with concrete types. Consumer supplies
		// keyGen, keyFunc, and sentinel for Tier 0 reference synthesis.
		repotest.AssertRepositoryModel[string, generic.Item](t,
			func() generic.Repository[string, generic.Item] {
				return generic.NewInMemoryRepository()
			},
			repotest.RepositoryModelKeyGen[string, generic.Item](
				rapid.SampledFrom([]string{"a", "b", "c", "d", "e"}),
			),
			repotest.RepositoryModelKeyFunc[string, generic.Item](
				func(v generic.Item) string { return v.ID },
			),
			repotest.RepositoryModelSentinel[string, generic.Item](
				generic.ErrNotFound,
			),
		)
	})

	t.Run("catches broken parameterized impl", func(t *testing.T) {
		t.Parallel()
		// Negative test: brokenRepository silently drops Put values.
		// ReadAfterWrite must catch it via the parameterized path.
		keyGen := rapid.SampledFrom([]string{"a", "b", "c", "d", "e"})
		ft := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() {
			defer close(done)
			repotest.AssertRepositoryModel[string, generic.Item](ft,
				func() generic.Repository[string, generic.Item] {
					return &brokenRepository{data: map[string]generic.Item{}}
				},
				repotest.RepositoryModelKeyGen[string, generic.Item](keyGen),
				repotest.RepositoryModelKeyFunc[string, generic.Item](
					func(v generic.Item) string { return v.ID },
				),
				repotest.RepositoryModelSentinel[string, generic.Item](
					generic.ErrNotFound,
				),
			)
		}()
		<-done
		if !ft.Failed() {
			t.Fatal("ReadAfterWrite should have caught broken Put")
		}
	})
}

// brokenRepository silently drops Put values — Get always returns ErrNotFound.
type brokenRepository struct {
	data map[string]generic.Item
}

func (r *brokenRepository) Get(_ context.Context, k string) (generic.Item, error) {
	v, ok := r.data[k]
	if !ok {
		return generic.Item{}, generic.ErrNotFound
	}
	return v, nil
}

func (r *brokenRepository) Put(_ context.Context, _ generic.Item) error {
	// BUG: silently drops the value
	return nil
}

func (r *brokenRepository) Delete(_ context.Context, k string) error {
	delete(r.data, k)
	return nil
}

func (r *brokenRepository) Count(_ context.Context) (int, error) {
	return len(r.data), nil
}

// Compile-time check.
var _ generic.Repository[string, generic.Item] = (*brokenRepository)(nil)
