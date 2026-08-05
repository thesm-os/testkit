// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize_test

import (
	"testing"

	"go.thesmos.sh/testkit/engine/model/law"
	"go.thesmos.sh/testkit/engine/model/linearize"
)

func TestEventual(t *testing.T) {
	t.Parallel()

	merge := func(a, b int) int { return max(a, b) }

	t.Run("post states equal to the join pass", func(t *testing.T) {
		t.Parallel()
		e := linearize.Eventual[int]{Merge: merge}
		if err := e.Check([]int{1, 3, 2}, []int{3, 3, 3}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("replica below the join fires", func(t *testing.T) {
		t.Parallel()
		e := linearize.Eventual[int]{Merge: merge}
		if err := e.Check([]int{1, 3, 2}, []int{3, 2, 3}); err == nil {
			t.Fatal("expected convergence failure")
		}
	})
}

func TestCausal(t *testing.T) {
	t.Parallel()

	wx1 := law.ClientOp[string]{Write: true, Key: "x", Version: 1}
	wy1 := law.ClientOp[string]{Write: true, Key: "y", Version: 1}
	c := linearize.Causal[string]{
		HappensBefore: func(a, b law.ClientOp[string]) bool {
			return a == wx1 && b == wy1
		},
	}
	events := []law.ClientEvent[string]{
		{Client: 1, Op: wx1},
		{Client: 1, Op: wy1},
		{Client: 2, Op: law.ClientOp[string]{Key: "y", Version: 1}},
		{Client: 2, Op: law.ClientOp[string]{Key: "x", Version: 0}}, // violates the cut
	}

	t.Run("delegates to the causal checker", func(t *testing.T) {
		t.Parallel()
		if err := c.Check(events); err == nil {
			t.Fatal("expected causal violation")
		}
	})
}

func TestSnapshotIsolation(t *testing.T) {
	t.Parallel()

	writeSkew := []law.Txn[string]{
		{
			ID:     1,
			Reads:  []law.TxnOp[string]{{Key: "x"}, {Key: "y"}},
			Writes: []law.TxnOp[string]{{Key: "x", Version: 1}},
		},
		{
			ID:     2,
			Reads:  []law.TxnOp[string]{{Key: "x"}, {Key: "y"}},
			Writes: []law.TxnOp[string]{{Key: "y", Version: 1}},
		},
	}
	si := linearize.SnapshotIsolation[string]{}

	t.Run("G0 and G1 pass on write skew, G2 fires", func(t *testing.T) {
		t.Parallel()
		if err := si.G0(writeSkew); err != nil {
			t.Fatal(err)
		}
		if err := si.G1(writeSkew); err != nil {
			t.Fatal(err)
		}
		if err := si.G2(writeSkew); err == nil {
			t.Fatal("expected G2 on write skew")
		}
	})
}
