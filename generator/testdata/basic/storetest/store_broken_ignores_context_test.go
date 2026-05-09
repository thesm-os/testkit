// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//go:build broken_fixtures

package storetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/basic"
)

// brokenIgnoresContextStore is a deliberately-broken [basic.Store]:
// Get and Put both ignore ctx.Err(). The suite's Reader/Writer
// baseline "respects context" subtest invokes with an already-
// cancelled ctx and expects context.Canceled — this impl returns
// success instead.
type brokenIgnoresContextStore struct {
	inner basic.Store
}

func (s brokenIgnoresContextStore) Get(_ context.Context, key string) (basic.Item, error) {
	return s.inner.Get(context.Background(), key)
}

func (s brokenIgnoresContextStore) Put(_ context.Context, item basic.Item) error {
	return s.inner.Put(context.Background(), item)
}

// TestBrokenStoreIgnoresContext is expected to FAIL — the Reader/
// Writer baseline's "respects context" subtest will report no error
// from a cancelled-context call.
func TestBrokenStoreIgnoresContext(t *testing.T) {
	t.Parallel()
	AssertStoreContract(t, func() basic.Store {
		inmem := basic.NewInMemoryStore()
		inmem.Seed("test-key", basic.Item{ID: "test-id"})
		return brokenIgnoresContextStore{inner: inmem}
	})
}
