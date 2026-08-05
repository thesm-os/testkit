// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"sync"
	"testing"

	"go.thesmos.sh/testkit/engine/model/history"
)

func TestNew(t *testing.T) {
	t.Parallel()
	h := history.New[string, int]()
	if h.TotalLen() != 0 {
		t.Fatal("new history should be empty")
	}
	if len(h.Partitions()) != 0 {
		t.Fatal("new history should have zero partitions")
	}
}

func TestRecordAndSnapshot(t *testing.T) {
	t.Parallel()
	h := history.New[string, int]()
	h.Record("a", 1)
	h.Record("a", 2)
	h.Record("a", 3)

	snap := h.Snapshot("a")
	if len(snap) != 3 || snap[0] != 1 || snap[1] != 2 || snap[2] != 3 {
		t.Fatalf("expected [1,2,3], got %v", snap)
	}
}

func TestMultiPartition(t *testing.T) {
	t.Parallel()
	h := history.New[string, int]()
	h.Record("b", 10)
	h.Record("a", 20)
	h.Record("c", 30)

	parts := h.Partitions()
	if len(parts) != 3 {
		t.Fatalf("expected 3 partitions, got %d", len(parts))
	}
	// Sorted by fmt.Sprint.
	if parts[0] != "a" || parts[1] != "b" || parts[2] != "c" {
		t.Fatalf("expected [a,b,c], got %v", parts)
	}
}

func TestSnapshotByPartition(t *testing.T) {
	t.Parallel()
	h := history.New[string, int]()
	h.Record("x", 1)
	h.Record("y", 2)

	m := h.SnapshotByPartition()
	// Mutate the copy — original should be unchanged.
	m["x"][0] = 999
	if h.Snapshot("x")[0] != 1 {
		t.Fatal("SnapshotByPartition must return defensive copy")
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
	h := history.New[string, int]()
	h.Record("a", 1)
	h.Record("b", 2)
	h.Reset()

	if h.TotalLen() != 0 {
		t.Fatal("reset should clear all entries")
	}
	if len(h.Partitions()) != 0 {
		t.Fatal("reset should clear all partitions")
	}
}

func TestLen(t *testing.T) {
	t.Parallel()
	h := history.New[string, int]()
	h.Record("a", 1)
	h.Record("a", 2)
	h.Record("b", 3)

	if h.Len("a") != 2 {
		t.Fatalf("expected Len(a)=2, got %d", h.Len("a"))
	}
	if h.Len("b") != 1 {
		t.Fatalf("expected Len(b)=1, got %d", h.Len("b"))
	}
	if h.TotalLen() != 3 {
		t.Fatalf("expected TotalLen=3, got %d", h.TotalLen())
	}
}

func TestConcurrentRecord(t *testing.T) {
	t.Parallel()
	h := history.New[string, int]()
	const workers = 8
	const perWorker = 100

	var wg sync.WaitGroup
	for w := range workers {
		wg.Go(func() {
			part := string(rune('a' + w))
			for i := range perWorker {
				h.Record(part, i)
			}
		})
	}
	wg.Wait()

	if h.TotalLen() != workers*perWorker {
		t.Fatalf("expected %d, got %d", workers*perWorker, h.TotalLen())
	}
	for w := range workers {
		part := string(rune('a' + w))
		if h.Len(part) != perWorker {
			t.Fatalf("partition %s: expected %d, got %d", part, perWorker, h.Len(part))
		}
	}
}

func TestGlobalTimeline(t *testing.T) {
	t.Parallel()
	h := history.New[string, int]()
	h.Record("a", 1)
	h.Record("b", 2)
	h.Record("a", 3)

	tl := h.GlobalTimeline()
	if len(tl) != 3 {
		t.Fatalf("expected 3 entries in global timeline, got %d", len(tl))
	}
	// Entries appear in insertion order with monotonic indices.
	if tl[0].Index != 0 || tl[0].PartKey != "a" || tl[0].Entry != 1 {
		t.Fatalf("entry 0: expected {0, a, 1}, got %+v", tl[0])
	}
	if tl[1].Index != 1 || tl[1].PartKey != "b" || tl[1].Entry != 2 {
		t.Fatalf("entry 1: expected {1, b, 2}, got %+v", tl[1])
	}
	if tl[2].Index != 2 || tl[2].PartKey != "a" || tl[2].Entry != 3 {
		t.Fatalf("entry 2: expected {2, a, 3}, got %+v", tl[2])
	}
	// Defensive copy.
	tl[0].Entry = 999
	if h.GlobalTimeline()[0].Entry != 1 {
		t.Fatal("GlobalTimeline must return defensive copy")
	}
}

func TestDegenerateStructKey(t *testing.T) {
	t.Parallel()
	h := history.New[struct{}, int]()
	h.Record(struct{}{}, 1)
	h.Record(struct{}{}, 2)
	h.Record(struct{}{}, 3)

	if h.TotalLen() != 3 {
		t.Fatalf("expected 3, got %d", h.TotalLen())
	}
	parts := h.Partitions()
	if len(parts) != 1 {
		t.Fatalf("struct{} key should collapse to 1 partition, got %d", len(parts))
	}
	if h.Len(struct{}{}) != 3 {
		t.Fatalf("expected 3 in single partition, got %d", h.Len(struct{}{}))
	}
}
