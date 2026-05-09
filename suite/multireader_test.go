// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type (
	entityWithMeta struct{ name, etag string }
	entityStore    struct{ data map[string]entityWithMeta }
)

var errEntityMissing = errors.New("entity: missing")

func newEntityStore(data map[string]entityWithMeta) *entityStore { return &entityStore{data: data} }

func (s *entityStore) Fetch(_ context.Context, k string) (string, string, error) {
	e, ok := s.data[k]
	if !ok {
		return "", "", errEntityMissing
	}
	return e.name, e.etag, nil
}

func multiReaderCtx(
	t *testing.T, data map[string]entityWithMeta,
) suite.MultiReaderContext[*entityStore, string, string, string] {
	t.Helper()
	return suite.MultiReaderContext[*entityStore, string, string, string]{
		T: t,
		MultiReaderBindings: bindings.MultiReaderBindings[*entityStore, string, string, string]{
			Factory: func() *entityStore { return newEntityStore(data) },
			Call: func(ctx context.Context, s *entityStore, k string) (string, string, error) {
				return s.Fetch(ctx, k)
			},
		},
	}
}

func TestMultiReader(t *testing.T) {
	t.Parallel()
	data := map[string]entityWithMeta{"k1": {name: "alpha", etag: "v1"}}

	t.Run("ReturnsForKey surfaces both values", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiReaderReturnsForKey[*entityStore, string, string, string](
			"k1", "alpha", "v1")(multiReaderCtx(t, data))
	})

	t.Run("ReturnsSentinel surfaces the configured error for unknown key", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiReaderReturnsSentinel[*entityStore, string, string, string](
			"missing", errEntityMissing)(multiReaderCtx(t, data))
	})

	t.Run("Consistent yields equal pairs across N calls", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiReaderConsistent[*entityStore, string, string, string](
			"k1", 4)(multiReaderCtx(t, data))
	})

	t.Run("RespectsContext surfaces ctx.Canceled", func(t *testing.T) {
		t.Parallel()
		ctx := suite.MultiReaderContext[*entityStore, string, string, string]{
			T: t,
			MultiReaderBindings: bindings.MultiReaderBindings[*entityStore, string, string, string]{
				Factory: func() *entityStore { return newEntityStore(data) },
				Call: func(c context.Context, _ *entityStore, _ string) (string, string, error) {
					return "", "", c.Err()
				},
			},
		}
		suite.AssertMultiReaderRespectsContext[*entityStore, string, string, string]("k1")(ctx)
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertMultiReaderConcurrentSafe[*entityStore, string, string, string](
			"k1", 4, 50)(multiReaderCtx(t, data))
	})
}

func TestAssertMultiReaderBaseline(t *testing.T) {
	t.Parallel()
	data := map[string]entityWithMeta{"k1": {name: "alpha", etag: "v1"}}
	ctx := suite.MultiReaderContext[*entityStore, string, string, string]{
		T: t,
		MultiReaderBindings: bindings.MultiReaderBindings[*entityStore, string, string, string]{
			Factory: func() *entityStore { return newEntityStore(data) },
			Call: func(c context.Context, s *entityStore, k string) (string, string, error) {
				if err := c.Err(); err != nil {
					return "", "", err
				}
				return s.Fetch(c, k)
			},
		},
	}
	suite.AssertMultiReaderBaseline[*entityStore, string, string, string](
		"k1", "alpha", "v1")(ctx)
}
