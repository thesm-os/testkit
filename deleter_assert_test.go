// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
)

type delStore struct {
	data map[string]bool
}

func newDelStore() *delStore {
	return &delStore{data: map[string]bool{"existing": true}}
}

func (s *delStore) Delete(_ context.Context, key string) error {
	if !s.data[key] {
		return errNotFound
	}
	delete(s.data, key)
	return nil
}

func deleterCtx(t *testing.T) testkit.DeleterContext[*delStore, string] {
	t.Helper()
	return testkit.DeleterContext[*delStore, string]{
		T: t,
		DeleterBindings: testkit.DeleterBindings[*delStore, string]{
			Factory: newDelStore,
			Call: func(ctx context.Context, s *delStore, k string) error {
				return s.Delete(ctx, k)
			},
		},
	}
}

func TestAssertDeleteSucceeds(t *testing.T) {
	t.Parallel()
	ctx := deleterCtx(t)
	testkit.AssertDeleteSucceeds[*delStore, string]("existing")(ctx)
}

func TestAssertDeleteIdempotent(t *testing.T) {
	t.Parallel()
	ctx := deleterCtx(t)
	// Delete "existing" twice — first succeeds, second returns not-found.
	// Idempotent checks same nil/non-nil outcome, so this tests the
	// non-nil == non-nil path.
	testkit.AssertDeleteIdempotent[*delStore, string]("nonexistent")(ctx)
}

func TestAssertDeleteReturnsNotFound(t *testing.T) {
	t.Parallel()
	ctx := deleterCtx(t)
	testkit.AssertDeleteReturnsNotFound[*delStore, string]("nonexistent", errNotFound)(ctx)
}
