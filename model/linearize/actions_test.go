// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize_test

import (
	"context"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/linearize"
)

type kvStore struct {
	data map[string]string
}

func newKV() *kvStore { return &kvStore{data: make(map[string]string)} }

func TestConcurrentReader(t *testing.T) {
	t.Parallel()
	a := linearize.ConcurrentReader(
		"Get",
		rapid.Just("k"),
		func(_ context.Context, s *kvStore, k string) (string, error) {
			v, ok := s.data[k]
			if !ok {
				return "", errNotFound
			}
			return v, nil
		},
	)
	if a.Name != "Get" {
		t.Fatalf("expected name Get, got %s", a.Name)
	}
	rapid.Check(t, func(rt *rapid.T) {
		s := newKV()
		input := a.Gen(rt)
		result := a.Apply(t.Context(), s, input)
		r := result.(linearize.ReaderResult[string])
		if r.Err == nil {
			t.Fatal("expected not-found error on empty store")
		}
		pk := a.PartitionKey(input)
		if pk != "k" {
			t.Fatalf("expected partition key 'k', got %q", pk)
		}
	})
}

func TestConcurrentReaderWithBool(t *testing.T) {
	t.Parallel()
	a := linearize.ConcurrentReaderWithBool(
		"Load",
		rapid.Just("k"),
		func(_ context.Context, s *kvStore, k string) (string, bool) {
			v, ok := s.data[k]
			return v, ok
		},
	)
	if a.Name != "Load" {
		t.Fatalf("expected name Load, got %s", a.Name)
	}
	rapid.Check(t, func(rt *rapid.T) {
		s := newKV()
		input := a.Gen(rt)
		result := a.Apply(t.Context(), s, input)
		r := result.(linearize.ReaderBoolResult[string])
		if r.OK {
			t.Fatal("expected ok=false on empty store")
		}
	})
}

func TestConcurrentWriter(t *testing.T) {
	t.Parallel()
	a := linearize.ConcurrentWriter(
		"Put",
		rapid.Just("val"),
		func(_ context.Context, s *kvStore, v string) error {
			s.data[v] = v
			return nil
		},
		func(v string) string { return v },
	)
	if a.Name != "Put" {
		t.Fatalf("expected name Put, got %s", a.Name)
	}
	rapid.Check(t, func(rt *rapid.T) {
		s := newKV()
		input := a.Gen(rt)
		result := a.Apply(t.Context(), s, input)
		r := result.(linearize.WriterResult)
		if r.Err != nil {
			t.Fatalf("unexpected error: %v", r.Err)
		}
		pk := a.PartitionKey(input)
		if pk != "val" {
			t.Fatalf("expected partition key 'val', got %q", pk)
		}
	})
}

func TestConcurrentDeleter(t *testing.T) {
	t.Parallel()
	a := linearize.ConcurrentDeleter(
		"Delete",
		rapid.Just("k"),
		func(_ context.Context, s *kvStore, k string) error {
			delete(s.data, k)
			return nil
		},
	)
	if a.Name != "Delete" {
		t.Fatalf("expected name Delete, got %s", a.Name)
	}
	rapid.Check(t, func(rt *rapid.T) {
		s := newKV()
		input := a.Gen(rt)
		result := a.Apply(t.Context(), s, input)
		r := result.(linearize.WriterResult)
		if r.Err != nil {
			t.Fatalf("unexpected error: %v", r.Err)
		}
	})
}

func TestConcurrentLookup(t *testing.T) {
	t.Parallel()
	a := linearize.ConcurrentLookup(
		"Inspect",
		rapid.Just("k"),
		func(s *kvStore, k string) (string, int, bool) {
			v, ok := s.data[k]
			return v, len(s.data), ok
		},
	)
	if a.Name != "Inspect" {
		t.Fatalf("expected name Inspect, got %s", a.Name)
	}
	rapid.Check(t, func(rt *rapid.T) {
		s := newKV()
		input := a.Gen(rt)
		result := a.Apply(t.Context(), s, input)
		r := result.(linearize.LookupResult[string, int])
		if r.OK {
			t.Fatal("expected ok=false on empty store")
		}
	})
}
