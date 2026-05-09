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

type compositeStore struct {
	mu   sync.Mutex
	data map[string]string
}

var errCompInvalid = errors.New("composite: invalid value")

func newCompositeStore() *compositeStore { return &compositeStore{data: make(map[string]string)} }

func (s *compositeStore) Set(_ context.Context, k1, value string) error {
	if value == "INVALID" {
		return errCompInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[k1] = value
	return nil
}

func compositeWriterCtx(t *testing.T) suite.CompositeWriterContext[*compositeStore, string, string] {
	t.Helper()
	return suite.CompositeWriterContext[*compositeStore, string, string]{
		T: t,
		CompositeWriterBindings: bindings.CompositeWriterBindings[*compositeStore, string, string]{
			Factory: newCompositeStore,
			Call: func(ctx context.Context, s *compositeStore, k1, v string) error {
				return s.Set(ctx, k1, v)
			},
		},
	}
}

func TestCompositeWriter(t *testing.T) {
	t.Parallel()

	t.Run("WriteSucceeds for valid (k, v) pair", func(t *testing.T) {
		t.Parallel()
		suite.AssertCompositeWriteSucceeds[*compositeStore, string, string](
			"k1", "value")(compositeWriterCtx(t))
	})

	t.Run("WriteRejectInvalid surfaces the sentinel", func(t *testing.T) {
		t.Parallel()
		suite.AssertCompositeWriteRejectInvalid[*compositeStore, string, string](
			"k1", "INVALID", errCompInvalid)(compositeWriterCtx(t))
	})

	t.Run("Idempotent succeeds on repeat write of same pair", func(t *testing.T) {
		t.Parallel()
		suite.AssertCompositeWriterIdempotent[*compositeStore, string, string](
			"k1", "value")(compositeWriterCtx(t))
	})

	t.Run("RespectsContext surfaces ctx.Canceled on cancelled call", func(t *testing.T) {
		t.Parallel()
		ctx := suite.CompositeWriterContext[*compositeStore, string, string]{
			T: t,
			CompositeWriterBindings: bindings.CompositeWriterBindings[*compositeStore, string, string]{
				Factory: newCompositeStore,
				Call: func(c context.Context, _ *compositeStore, _, _ string) error {
					return c.Err()
				},
			},
		}
		suite.AssertCompositeWriterRespectsContext[*compositeStore, string, string](
			"k1", "v")(ctx)
	})

	t.Run("ConcurrentSafe runs without races", func(t *testing.T) {
		t.Parallel()
		suite.AssertCompositeWriterConcurrentSafe[*compositeStore, string, string](
			"k1", "value", 4, 50)(compositeWriterCtx(t))
	})
}
