// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package law_test

import (
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit/core/trace"
	"go.thesmos.sh/testkit/engine/model/law"
)

// cev builds a ClientEvent for the causal-order checker.
func cev(client int, write bool, key string, version int64) law.ClientEvent[string] {
	return law.ClientEvent[string]{
		Client: client,
		Op:     law.ClientOp[string]{Write: write, Key: key, Version: version},
	}
}

// hbPairs builds a HappensBefore predicate from explicit
// (before, after) write pairs keyed by key@version.
func hbPairs(pairs ...[2]law.ClientOp[string]) func(a, b law.ClientOp[string]) bool {
	return func(a, b law.ClientOp[string]) bool {
		for _, p := range pairs {
			if p[0].Key == a.Key && p[0].Version == a.Version &&
				p[1].Key == b.Key && p[1].Version == b.Version {

				return true
			}
		}
		return false
	}
}

func TestCheckCausalOrder(t *testing.T) {
	t.Parallel()

	wx1 := law.ClientOp[string]{Write: true, Key: "x", Version: 1}
	wy1 := law.ClientOp[string]{Write: true, Key: "y", Version: 1}
	hb := hbPairs([2]law.ClientOp[string]{wx1, wy1}) // x@1 happens-before y@1

	t.Run("reads respecting the causal cut pass", func(t *testing.T) {
		t.Parallel()
		events := []law.ClientEvent[string]{
			cev(1, true, "x", 1),  // client 1 writes x@1
			cev(1, true, "y", 1),  // then y@1 (x@1 → y@1)
			cev(2, false, "y", 1), // client 2 observes y@1...
			cev(2, false, "x", 1), // ...and then x@1 — cut satisfied
		}
		if err := law.CheckCausalOrder(events, hb); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("read older than the causal cut fires", func(t *testing.T) {
		t.Parallel()
		events := []law.ClientEvent[string]{
			cev(1, true, "x", 1),
			cev(1, true, "y", 1),
			cev(2, false, "y", 1), // observing y@1 requires x@>=1
			cev(2, false, "x", 0), // stale x — causality violated
		}
		if err := law.CheckCausalOrder(events, hb); err == nil {
			t.Fatal("expected causal violation")
		}
	})

	t.Run("non-causally-related stale read on another client passes", func(t *testing.T) {
		t.Parallel()
		events := []law.ClientEvent[string]{
			cev(1, true, "x", 1),
			cev(1, true, "y", 1),
			cev(3, false, "x", 0), // client 3 observed nothing yet — allowed
		}
		if err := law.CheckCausalOrder(events, hb); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("session monotonicity: re-reading older version of same key fires", func(t *testing.T) {
		t.Parallel()
		events := []law.ClientEvent[string]{
			cev(1, true, "x", 2),
			cev(2, false, "x", 2),
			cev(2, false, "x", 1), // went backwards within the session
		}
		if err := law.CheckCausalOrder(events, hb); err == nil {
			t.Fatal("expected session-monotonicity violation")
		}
	})

	t.Run("writer observes its own writes' causal cut", func(t *testing.T) {
		t.Parallel()
		events := []law.ClientEvent[string]{
			cev(1, true, "x", 1),
			cev(1, true, "y", 1),
			cev(1, false, "x", 0), // writer reads its own key stale
		}
		if err := law.CheckCausalOrder(events, hb); err == nil {
			t.Fatal("expected own-write causal violation")
		}
	})

	// A version no write in the trace produced still enters the client's cut,
	// or the session would be free to read backwards from it afterwards.
	t.Run("a read of an unwritten version still raises the cut", func(t *testing.T) {
		t.Parallel()
		if err := law.CheckCausalOrder([]law.ClientEvent[string]{
			cev(1, false, "z", 3), // no write of z@3 appears anywhere
		}, hb); err != nil {
			t.Fatalf("reading a version nobody wrote is not itself a violation: %v", err)
		}

		if err := law.CheckCausalOrder([]law.ClientEvent[string]{
			cev(1, false, "z", 3),
			cev(1, false, "z", 1), // went backwards from the unwritten version
		}, hb); err == nil {
			t.Fatal("the unwritten version must have entered the cut")
		}
	})
}

func TestCausalOrdering(t *testing.T) {
	t.Parallel()

	wx1 := law.ClientOp[string]{Write: true, Key: "x", Version: 1}
	wy1 := law.ClientOp[string]{Write: true, Key: "y", Version: 1}
	hb := hbPairs([2]law.ClientOp[string]{wx1, wy1})

	t.Run("law classifies the trace and delegates", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(wr(1, "x", 1))
		tr.Record(wr(1, "y", 1))
		tr.Record(rd(2, "y", 1))
		tr.Record(rd(2, "x", 0)) // violation
		l := &law.CausalOrdering[struct{}, string]{
			Classify:      clientClassify,
			HappensBefore: hb,
		}
		l.BindTrace(tr)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err == nil {
				rt.Fatal("expected causal violation from trace")
			}
		})
	})

	t.Run("clean trace passes", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(wr(1, "x", 1))
		tr.Record(wr(1, "y", 1))
		tr.Record(rd(2, "y", 1))
		tr.Record(rd(2, "x", 1))
		l := &law.CausalOrdering[struct{}, string]{
			Classify:      clientClassify,
			HappensBefore: hb,
		}
		l.BindTrace(tr)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err != nil {
				rt.Fatal(err)
			}
		})
	})

	// A trace carries every recorded event, not just the ones this law can
	// interpret; unclassifiable events must be skipped, not misread as reads.
	t.Run("events Classify rejects are skipped", func(t *testing.T) {
		t.Parallel()
		tr := trace.New()
		tr.Record(wr(1, "x", 1))
		tr.Record(trace.Event{ClientID: 1, Method: "Ping"}) // no key, no version
		tr.Record(wr(1, "y", 1))
		tr.Record(rd(2, "y", 1))
		tr.Record(rd(2, "x", 0)) // violation, reachable only if Ping was skipped
		l := &law.CausalOrdering[struct{}, string]{
			Classify:      clientClassify,
			HappensBefore: hb,
		}
		l.BindTrace(tr)
		rapid.Check(t, func(rt *rapid.T) {
			if err := l.Check(rt, struct{}{}, struct{}{}); err == nil {
				rt.Fatal("an uninterpretable event must not mask the violation after it")
			}
		})
	})
}
