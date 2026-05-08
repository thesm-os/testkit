// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cachetest_test

import (
	"testing"

	"go.thesmos.sh/testkit/gen/stub/testdata/generics"
	"go.thesmos.sh/testkit/gen/stub/testdata/generics/cachetest"
)

type item struct {
	ID   string
	Name string
}

func TestCacheStub(t *testing.T) {
	t.Parallel()

	t.Run("construct and call", func(t *testing.T) {
		t.Parallel()
		s := cachetest.NewCacheStub[string, item](t)
		s.OnGet.Returns(item{ID: "a", Name: "test"}, nil)
		got, err := s.Get(t.Context(), "a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "a" {
			t.Fatalf("got %v, want ID=a", got)
		}
	})

	t.Run("DelegateTo", func(t *testing.T) {
		t.Parallel()
		inner := generics.NewInMemoryCache[string, item](func(v item) string { return v.ID })
		_ = inner.Put(t.Context(), item{ID: "b", Name: "delegate"})
		s := cachetest.NewCacheStub[string, item](t,
			cachetest.CacheStubDelegateTo[string, item](inner),
		)
		got, err := s.Get(t.Context(), "b")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "delegate" {
			t.Fatalf("got %v, want Name=delegate", got)
		}
	})
}
