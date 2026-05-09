// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//go:build broken_fixtures

// Build-tagged out by default — run via:
//
//	go test -tags=broken_fixtures -run TestBrokenStoreReturnsWrongValue .
//
// The companion [TestBrokenFixturesAreCaught] subprocess-runs each
// broken case under this tag and asserts the contract caught the
// violation with the documented failure substring. Without the tag
// these files are invisible so a default `go test ./...` doesn't
// see deliberately-failing tests.

package storetest_test

import (
	"context"
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/basic"
)

// brokenReturnsWrongValueStore is a deliberately-broken [basic.Store]:
// Get returns the wrong Item.ID for the contract's sample key. The
// suite's Reader baseline asserts equality on the returned value, so
// this triggers `must return expected value: ... -test-id +WRONG`.
// Put forwards to the in-mem so the Writer baseline still passes —
// the broken case is scoped to one specific assertion.
type brokenReturnsWrongValueStore struct {
	inner basic.Store
}

func (s brokenReturnsWrongValueStore) Get(_ context.Context, _ string) (basic.Item, error) {
	return basic.Item{ID: "WRONG"}, nil
}

func (s brokenReturnsWrongValueStore) Put(ctx context.Context, item basic.Item) error {
	return s.inner.Put(ctx, item)
}

// TestBrokenStoreReturnsWrongValue is expected to FAIL — the Reader
// baseline's "returns for key" subtest will report a value mismatch.
// [TestBrokenFixturesAreCaught] subprocesses this test and asserts
// the failure surfaces with the expected substring.
func TestBrokenStoreReturnsWrongValue(t *testing.T) {
	t.Parallel()
	AssertStoreContract(t, func() basic.Store {
		inmem := basic.NewInMemoryStore()
		inmem.Seed("test-key", basic.Item{ID: "test-id"})
		return brokenReturnsWrongValueStore{inner: inmem}
	})
}
