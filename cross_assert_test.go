// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit"
)

type kvStore struct {
	data map[string]string
}

func newKVStore() *kvStore { return &kvStore{data: make(map[string]string)} }

func (s *kvStore) Put(_ context.Context, key, value string) error {
	s.data[key] = value
	return nil
}

func (s *kvStore) Get(_ context.Context, key string) (string, error) {
	v, ok := s.data[key]
	if !ok {
		return "", errNotFound
	}
	return v, nil
}

func (s *kvStore) Delete(_ context.Context, key string) error {
	delete(s.data, key)
	return nil
}

func (s *kvStore) All() []string {
	out := make([]string, 0, len(s.data))
	for _, v := range s.data {
		out = append(out, v)
	}
	return out
}

// Wrap kvStore methods to match the cross-method primitive signatures.
// Cross-method uses entry{Key, Value} for V and string for K.

func TestAssertReadAfterWrite(t *testing.T) {
	t.Parallel()
	ctx := testkit.CrossContext[*kvStore]{
		T:       t,
		Factory: newKVStore,
	}
	testkit.AssertReadAfterWrite[*kvStore, string, entry](
		entry{Key: "a", Value: "alpha"},
		func(s *kvStore, ctx context.Context, e entry) error { return s.Put(ctx, e.Key, e.Value) },
		func(s *kvStore, ctx context.Context, k string) (entry, error) {
			v, err := s.Get(ctx, k)
			return entry{Key: k, Value: v}, err
		},
		func(e entry) string { return e.Key },
	)(ctx)
}

func TestAssertDeleteRemovesValue(t *testing.T) {
	t.Parallel()
	ctx := testkit.CrossContext[*kvStore]{
		T:       t,
		Factory: newKVStore,
	}
	testkit.AssertDeleteRemovesValue[*kvStore, string, entry](
		entry{Key: "a", Value: "alpha"},
		func(s *kvStore, ctx context.Context, e entry) error { return s.Put(ctx, e.Key, e.Value) },
		func(s *kvStore, ctx context.Context, k string) error { return s.Delete(ctx, k) },
		func(s *kvStore, ctx context.Context, k string) (entry, error) {
			v, err := s.Get(ctx, k)
			return entry{Key: k, Value: v}, err
		},
		func(e entry) string { return e.Key },
		errNotFound,
	)(ctx)
}

func TestAssertStreamReflectsMutations(t *testing.T) {
	t.Parallel()
	ctx := testkit.CrossContext[*kvStore]{
		T:       t,
		Factory: newKVStore,
	}
	testkit.AssertStreamReflectsMutations[*kvStore, string, entry](
		[]entry{{Key: "a", Value: "alpha"}, {Key: "b", Value: "beta"}},
		func(s *kvStore, ctx context.Context, e entry) error { return s.Put(ctx, e.Key, e.Value) },
		func(s *kvStore) []entry {
			out := make([]entry, 0, len(s.data))
			for k, v := range s.data {
				out = append(out, entry{Key: k, Value: v})
			}
			return out
		},
		func(e entry) string { return e.Key },
	)(ctx)
}
