// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/bindings"
	"go.thesmos.sh/testkit/suite"
)

type ptrStore struct{ data map[string]string }

func newPtrStore(data map[string]string) *ptrStore { return &ptrStore{data: data} }

func (s *ptrStore) Find(_ context.Context, k string) *string {
	v, ok := s.data[k]
	if !ok {
		return nil
	}
	return &v
}

func pointerReaderCtx(t *testing.T, data map[string]string) suite.PointerReaderContext[*ptrStore, string, string] {
	t.Helper()
	return suite.PointerReaderContext[*ptrStore, string, string]{
		T: t,
		PointerReaderBindings: bindings.PointerReaderBindings[*ptrStore, string, string]{
			Factory: func() *ptrStore { return newPtrStore(data) },
			Call: func(ctx context.Context, s *ptrStore, k string) *string {
				return s.Find(ctx, k)
			},
		},
	}
}

func TestPointerReader(t *testing.T) {
	t.Parallel()
	data := map[string]string{"k1": "alpha"}

	t.Run("ReturnsForKey surfaces the dereferenced value", func(t *testing.T) {
		t.Parallel()
		want := "alpha"
		suite.AssertPointerReaderReturnsForKey[*ptrStore, string, string](
			"k1", &want)(pointerReaderCtx(t, data))
	})

	t.Run("NilOnUnknown returns nil for missing keys", func(t *testing.T) {
		t.Parallel()
		suite.AssertPointerReaderNilOnUnknown[*ptrStore, string, string](
			"missing")(pointerReaderCtx(t, data))
	})

	t.Run("Consistent yields equal values across N calls", func(t *testing.T) {
		t.Parallel()
		suite.AssertPointerReaderConsistent[*ptrStore, string, string](
			"k1", 4)(pointerReaderCtx(t, data))
	})

	t.Run("RespectsContext surfaces nil under cancelled ctx", func(t *testing.T) {
		t.Parallel()
		ctx := suite.PointerReaderContext[*ptrStore, string, string]{
			T: t,
			PointerReaderBindings: bindings.PointerReaderBindings[*ptrStore, string, string]{
				Factory: func() *ptrStore { return newPtrStore(data) },
				Call: func(c context.Context, s *ptrStore, k string) *string {
					if c.Err() != nil {
						return nil
					}
					return s.Find(c, k)
				},
			},
		}
		suite.AssertPointerReaderRespectsContext[*ptrStore, string, string]("k1")(ctx)
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertPointerReaderConcurrentSafe[*ptrStore, string, string](
			"k1", 4, 50)(pointerReaderCtx(t, data))
	})
}

func TestAssertPointerReaderBaseline(t *testing.T) {
	t.Parallel()
	data := map[string]string{"k1": "alpha"}
	want := "alpha"
	ctx := suite.PointerReaderContext[*ptrStore, string, string]{
		T: t,
		PointerReaderBindings: bindings.PointerReaderBindings[*ptrStore, string, string]{
			Factory: func() *ptrStore { return newPtrStore(data) },
			Call: func(c context.Context, s *ptrStore, k string) *string {
				if c.Err() != nil {
					return nil
				}
				return s.Find(c, k)
			},
		},
	}
	suite.AssertPointerReaderBaseline[*ptrStore, string, string](
		"k1", &want, "missing")(ctx)
}
