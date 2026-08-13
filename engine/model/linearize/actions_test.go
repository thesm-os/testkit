// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize_test

import (
	"context"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/engine/model/linearize"
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
		// The partition key routes the op into a per-key sub-history;
		// without it every op would land in one partition and the
		// check would serialize.
		if got := a.PartitionKey(input); got != "k" {
			rt.Fatalf("the drawn key must be the partition key, got %q", got)
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
		// The partition key routes the op into a per-key sub-history;
		// without it every op would land in one partition and the
		// check would serialize.
		if got := a.PartitionKey(input); got != "k" {
			rt.Fatalf("the drawn key must be the partition key, got %q", got)
		}
	})
}

// TestConcurrentAnsweringWriter pins the answering wrapper: the stored
// state the subject answered rides to the trace whole, and the call's own
// error travels beside it rather than dying with the call.
func TestConcurrentAnsweringWriter(t *testing.T) {
	t.Parallel()

	type stamped struct {
		Key string
		Rev int64
	}
	store := map[string]stamped{}
	a := linearize.ConcurrentAnsweringWriter("Store",
		rapid.Just(stamped{Key: "a"}),
		func(_ context.Context, _ struct{}, v stamped) (stamped, error) {
			v.Rev = int64(len(store) + 1)
			store[v.Key] = v
			return v, nil
		},
		func(v stamped) string { return v.Key })

	rapid.Check(t, func(rt *rapid.T) {
		in := a.Gen(rt)
		out := a.Apply(t.Context(), struct{}{}, in)
		res, ok := out.(linearize.AnsweringResult[stamped])
		if !ok {
			t.Fatalf("the wrapper answers its typed result, got %T", out)
		}
		val, err := res.TraceOutput()
		if err != nil {
			t.Fatalf("a clean write carries no error: %v", err)
		}
		if val.(stamped).Rev == 0 {
			t.Fatal("the store-assigned stamp must survive to the trace")
		}
		if a.PartitionKey(in) != "a" {
			t.Fatal("partitioned by the value's own key")
		}
	})
}
