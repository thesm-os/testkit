// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/model/law"
)

// Shorthand history builders.
func top(key string, version int64) law.TxnOp[string] {
	return law.TxnOp[string]{Key: key, Version: version}
}

// serialHistory is a clean serial execution: T1 installs x@1, T2
// reads it and installs x@2. No anomaly of any class.
func serialHistory() []law.Txn[string] {
	return []law.Txn[string]{
		{ID: 1, Reads: []law.TxnOp[string]{top("x", 0)}, Writes: []law.TxnOp[string]{top("x", 1)}},
		{ID: 2, Reads: []law.TxnOp[string]{top("x", 1)}, Writes: []law.TxnOp[string]{top("x", 2)}},
	}
}

// writeSkewHistory is the canonical SI anomaly: both txns read the
// initial state of both keys and each writes one — an rw cycle with
// no ww conflict and no wr edge.
func writeSkewHistory() []law.Txn[string] {
	return []law.Txn[string]{
		{ID: 1, Reads: []law.TxnOp[string]{top("x", 0), top("y", 0)}, Writes: []law.TxnOp[string]{top("x", 1)}},
		{ID: 2, Reads: []law.TxnOp[string]{top("x", 0), top("y", 0)}, Writes: []law.TxnOp[string]{top("y", 1)}},
	}
}

func TestSnapshotIsolationG0(t *testing.T) {
	t.Parallel()

	t.Run("serial history passes", func(t *testing.T) {
		t.Parallel()
		if err := law.CheckSIG0(serialHistory()); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("interleaved conflicting writes fire a ww cycle", func(t *testing.T) {
		t.Parallel()
		// T1's x precedes T2's x, but T2's y precedes T1's y.
		txns := []law.Txn[string]{
			{ID: 1, Writes: []law.TxnOp[string]{top("x", 1), top("y", 2)}},
			{ID: 2, Writes: []law.TxnOp[string]{top("x", 2), top("y", 1)}},
		}
		if err := law.CheckSIG0(txns); err == nil {
			t.Fatal("expected G0 write-cycle detection")
		}
	})

	t.Run("write skew does not fire G0", func(t *testing.T) {
		t.Parallel()
		if err := law.CheckSIG0(writeSkewHistory()); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("law delegates to the checker", func(t *testing.T) {
		t.Parallel()
		l := law.SnapshotIsolationG0[struct{}, string]{
			History: func(*rapid.T, struct{}) []law.Txn[string] { return serialHistory() },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

func TestSnapshotIsolationG1(t *testing.T) {
	t.Parallel()

	t.Run("serial history passes", func(t *testing.T) {
		t.Parallel()
		if err := law.CheckSIG1(serialHistory()); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("reading an aborted write fires G1a", func(t *testing.T) {
		t.Parallel()
		txns := []law.Txn[string]{
			{ID: 1, Writes: []law.TxnOp[string]{top("x", 1)}},
			{ID: 2, Aborted: true, Writes: []law.TxnOp[string]{top("x", 2)}},
			{ID: 3, Reads: []law.TxnOp[string]{top("x", 2)}},
		}
		if err := law.CheckSIG1(txns); err == nil {
			t.Fatal("expected G1a aborted-read detection")
		}
	})

	t.Run("reading an intermediate write fires G1b", func(t *testing.T) {
		t.Parallel()
		txns := []law.Txn[string]{
			{ID: 1, Writes: []law.TxnOp[string]{top("x", 1), top("x", 2)}}, // final is x@2
			{ID: 2, Reads: []law.TxnOp[string]{top("x", 1)}},               // reads the intermediate
		}
		if err := law.CheckSIG1(txns); err == nil {
			t.Fatal("expected G1b intermediate-read detection")
		}
	})

	t.Run("circular information flow fires G1c", func(t *testing.T) {
		t.Parallel()
		txns := []law.Txn[string]{
			{ID: 1, Reads: []law.TxnOp[string]{top("y", 1)}, Writes: []law.TxnOp[string]{top("x", 1)}},
			{ID: 2, Reads: []law.TxnOp[string]{top("x", 1)}, Writes: []law.TxnOp[string]{top("y", 1)}},
		}
		if err := law.CheckSIG1(txns); err == nil {
			t.Fatal("expected G1c wr-cycle detection")
		}
	})

	t.Run("write skew does not fire G1", func(t *testing.T) {
		t.Parallel()
		if err := law.CheckSIG1(writeSkewHistory()); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("law delegates to the checker", func(t *testing.T) {
		t.Parallel()
		l := law.SnapshotIsolationG1[struct{}, string]{
			History: func(*rapid.T, struct{}) []law.Txn[string] { return serialHistory() },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err != nil {
				rt.Fatal(err)
			}
		})
	})
}

func TestSnapshotIsolationG2(t *testing.T) {
	t.Parallel()

	t.Run("serial history passes", func(t *testing.T) {
		t.Parallel()
		if err := law.CheckSIG2(serialHistory()); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("write skew fires G2", func(t *testing.T) {
		t.Parallel()
		if err := law.CheckSIG2(writeSkewHistory()); err == nil {
			t.Fatal("expected G2 anti-dependency-cycle detection")
		}
	})

	t.Run("law delegates to the checker", func(t *testing.T) {
		t.Parallel()
		l := law.SnapshotIsolationG2[struct{}, string]{
			History: func(*rapid.T, struct{}) []law.Txn[string] { return writeSkewHistory() },
		}
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err == nil {
				rt.Fatal("expected G2 via the law")
			}
		})
	})
}
