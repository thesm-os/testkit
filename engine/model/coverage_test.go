// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"hash/fnv"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/coverage"
	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/action"
	"go.thesmos.sh/testkit/engine/model/law"
)

// hashStore hashes a store's contents order-independently for
// state-space coverage.
func hashStore(s storeIface) uint64 {
	st, ok := s.(*store)
	if !ok {
		return 0
	}
	var sum uint64
	for k, v := range st.data {
		h := fnv.New64a()
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte(v.Name))
		sum += h.Sum64() // xor/add keeps it order-independent
	}
	return sum
}

func TestCoverageStateSpace(t *testing.T) {
	t.Parallel()

	var cov coverage.ComponentCoverage
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(
			action.Reader("Get", keyGen, storeGet),
			action.Writer("Put", itemGen, storePut),
			action.Deleter("Delete", keyGen, storeDel),
		),
		model.WithStateHash[storeIface](hashStore),
		model.WithCoverageSink[storeIface](&cov),
	)

	if cov.StateSpace.Explored < 1 {
		t.Fatalf("expected at least one explored state, got %+v", cov.StateSpace)
	}
}

func TestCoverageREQMatrix(t *testing.T) {
	t.Parallel()

	read := func(rt *rapid.T, s storeIface, k string) (item, error) { return s.Get(rt.Context(), k) }

	var cov coverage.ComponentCoverage
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(action.Reader("Get", keyGen, storeGet)),
		model.WithLawREQ("REQ-STORE-001", law.ReadAfterWrite[storeIface, string, item]{
			Read: read,
			Keys: keyGen,
		}),
		model.WithCoverageSink[storeIface](&cov),
	)

	laws := cov.REQToLaw["REQ-STORE-001"]
	if len(laws) != 1 || laws[0] != "AUTO-READ-AFTER-WRITE" {
		t.Fatalf("REQ-STORE-001 → %v, want [AUTO-READ-AFTER-WRITE]", laws)
	}
}

func TestCoverageSinkOptionalNoop(t *testing.T) {
	t.Parallel()
	// No sink: run must behave exactly as before (no panic, no effect).
	model.Assert(
		t,
		func() storeIface { return newStore() },
		model.WithReference(func() storeIface { return newStore() }),
		model.WithActions(action.Writer("Put", itemGen, storePut)),
		model.WithStateHash[storeIface](hashStore), // hash set but no sink → ignored
	)
}
