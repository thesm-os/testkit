// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

//go:build broken_fixtures

package storetest_test

import (
	"context"
	"errors"
	"testing"

	"go.thesmos.sh/testkit/generator/testdata/basic"
)

// errBrokenWrongSentinel is a stand-in error the broken impl returns
// in place of the contract-declared [basic.ErrConflict]. The
// AssertWriteRejectInvalid subtest asserts errors.Is(got, ErrConflict);
// this fixture's error doesn't satisfy that.
var errBrokenWrongSentinel = errors.New("broken: wrong sentinel")

// brokenWrongSentinelStore is a deliberately-broken [basic.Store]:
// Put on an empty-ID Item returns errBrokenWrongSentinel instead of
// the contract-declared ErrConflict. The Writer baseline's reject-
// invalid extra (gated on //testkit:errors) asserts the returned
// error matches one of the declared sentinels — this impl violates
// that.
type brokenWrongSentinelStore struct {
	inner basic.Store
}

func (s brokenWrongSentinelStore) Get(ctx context.Context, key string) (basic.Item, error) {
	return s.inner.Get(ctx, key)
}

func (s brokenWrongSentinelStore) Put(ctx context.Context, item basic.Item) error {
	if item.ID == "" {
		return errBrokenWrongSentinel
	}
	return s.inner.Put(ctx, item)
}

// TestBrokenStoreWrongSentinel is expected to FAIL — the Writer
// baseline's reject-invalid extra reports the returned error is not
// one of the declared //testkit:errors sentinels.
func TestBrokenStoreWrongSentinel(t *testing.T) {
	t.Parallel()
	AssertStoreContract(t, func() basic.Store {
		inmem := basic.NewInMemoryStore()
		inmem.Seed("test-key", basic.Item{ID: "test-id"})
		return brokenWrongSentinelStore{inner: inmem}
	})
}
