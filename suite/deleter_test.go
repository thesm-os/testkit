// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type delStore struct {
	mu   sync.Mutex
	data map[string]bool
}

func newDelStore() *delStore {
	return &delStore{data: map[string]bool{"existing": true}}
}

func (s *delStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.data[key] {
		return errNotFound
	}
	delete(s.data, key)
	return nil
}

func deleterCtx(t *testing.T) suite.DeleterContext[*delStore, string] {
	t.Helper()
	return suite.DeleterContext[*delStore, string]{
		T: t,
		DeleterBindings: bindings.DeleterBindings[*delStore, string]{
			Factory: newDelStore,
			Call: func(ctx context.Context, s *delStore, k string) error {
				return s.Delete(ctx, k)
			},
		},
	}
}

func TestDeleter(t *testing.T) {
	t.Parallel()

	t.Run("DeleteSucceeds for an existing key", func(t *testing.T) {
		t.Parallel()
		suite.AssertDeleteSucceeds[*delStore, string]("existing")(deleterCtx(t))
	})

	t.Run("DeleteIdempotent yields the same nil/non-nil outcome twice", func(t *testing.T) {
		t.Parallel()
		// "nonexistent" returns errNotFound twice — both calls observe
		// the same non-nil error (idempotent under the same input).
		suite.AssertDeleteIdempotent[*delStore, string]("nonexistent")(deleterCtx(t))
	})

	t.Run("DeleteReturnsNotFound surfaces the configured sentinel for unknown keys", func(t *testing.T) {
		t.Parallel()
		suite.AssertDeleteReturnsNotFound[*delStore, string](
			"nonexistent", errNotFound,
		)(deleterCtx(t))
	})

	t.Run("RespectsContext surfaces ctx.Canceled on cancelled call", func(t *testing.T) {
		t.Parallel()
		ctx := suite.DeleterContext[*delStore, string]{
			T: t,
			DeleterBindings: bindings.DeleterBindings[*delStore, string]{
				Factory: newDelStore,
				Call: func(c context.Context, _ *delStore, _ string) error {
					return c.Err()
				},
			},
		}
		suite.AssertDeleterRespectsContext[*delStore, string]("existing")(ctx)
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertDeleterConcurrentSafe[*delStore, string](
			"nonexistent", 4, 50,
		)(deleterCtx(t))
	})
}

func TestAssertDeleterBaseline(t *testing.T) {
	t.Parallel()
	// Baseline needs a Call that (a) respects context and (b) is idempotent
	// (no error on missing key). Wrap Delete to swallow errNotFound.
	ctx := suite.DeleterContext[*delStore, string]{
		T: t,
		DeleterBindings: bindings.DeleterBindings[*delStore, string]{
			Factory: newDelStore,
			Call: func(c context.Context, s *delStore, k string) error {
				if err := c.Err(); err != nil {
					return err
				}
				err := s.Delete(c, k)
				if errors.Is(err, errNotFound) {
					return nil
				}
				return err
			},
		},
	}
	suite.AssertDeleterBaseline[*delStore, string]("existing")(ctx)
}
