// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
)

func TestTestError(t *testing.T) {
	t.Parallel()

	t.Run("same name satisfies errors.Is", func(t *testing.T) {
		t.Parallel()
		a := testkit.TestError("boom")
		b := testkit.TestError("boom")
		if !errors.Is(a, b) {
			t.Fatal("same name must satisfy errors.Is")
		}
	})

	t.Run("different names do not match", func(t *testing.T) {
		t.Parallel()
		a := testkit.TestError("boom")
		b := testkit.TestError("crash")
		if errors.Is(a, b) {
			t.Fatal("different names must not match")
		}
	})

	t.Run("error message includes name", func(t *testing.T) {
		t.Parallel()
		e := testkit.TestError("boom")
		if e.Error() != "testkit: boom" {
			t.Fatalf("unexpected message: %q", e.Error())
		}
	})

	t.Run("does not match non-TestError", func(t *testing.T) {
		t.Parallel()
		a := testkit.TestError("boom")
		b := errors.New("boom")
		if errors.Is(a, b) {
			t.Fatal("TestError must not match plain error")
		}
	})
}

func TestRequireEnv(t *testing.T) {
	t.Parallel()

	t.Run("returns value when set", func(t *testing.T) {
		t.Parallel()
		// PATH is always set
		v := testkit.RequireEnv(t, "PATH")
		if v == "" {
			t.Fatal("PATH should not be empty")
		}
	})

	t.Run("skips when not set", func(t *testing.T) {
		t.Parallel()
		testkit.RequireEnv(t, "TESTKIT_DEFINITELY_NOT_SET_EVER")
		t.Fatal("should have been skipped")
	})
}

func TestSeededRand(t *testing.T) {
	t.Parallel()

	t.Run("same name produces same sequence", func(t *testing.T) {
		t.Parallel()
		f1 := testkit.NewFailableTB().WithName("TestFoo")
		f2 := testkit.NewFailableTB().WithName("TestFoo")
		r1 := testkit.SeededRand(f1)
		r2 := testkit.SeededRand(f2)
		for range 10 {
			if r1.Int64() != r2.Int64() {
				t.Fatal("same name must produce same sequence")
			}
		}
	})

	t.Run("different names produce different sequences", func(t *testing.T) {
		t.Parallel()
		f1 := testkit.NewFailableTB().WithName("TestFoo")
		f2 := testkit.NewFailableTB().WithName("TestBar")
		r1 := testkit.SeededRand(f1)
		r2 := testkit.SeededRand(f2)
		same := true
		for range 10 {
			if r1.Int64() != r2.Int64() {
				same = false
				break
			}
		}
		if same {
			t.Fatal("different names should produce different sequences")
		}
	})
}

func TestTempFile(t *testing.T) {
	t.Parallel()

	t.Run("creates file with content", func(t *testing.T) {
		t.Parallel()
		path := testkit.TempFile(t, "test.txt", []byte("hello"))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read temp file: %v", err)
		}
		if string(data) != "hello" {
			t.Fatalf("unexpected content: %q", data)
		}
	})
}

// tempDirTB gives a FailableTB the one method TempFile needs beyond the
// embedded nil interface, so a write failure can be observed instead of
// aborting the test that provokes it.
type tempDirTB struct {
	*testkit.FailableTB
	dir string
}

func (d tempDirTB) TempDir() string { return d.dir }

func TestTempFileWriteFailure(t *testing.T) {
	t.Parallel()

	// A name carrying a directory component that does not exist is the
	// cheapest honest write failure — no permission games, no root needed.
	tb := tempDirTB{FailableTB: testkit.NewFailableTB(), dir: t.TempDir()}
	testkit.TempFile(tb, "missing-dir/config.json", []byte("{}"))

	if !tb.Failed() {
		t.Fatal("a write into a nonexistent directory must fail the test")
	}
	if got := tb.Msg(); !strings.Contains(got, "TempFile: write missing-dir/config.json") {
		t.Fatalf("the diagnostic must name the file, got: %q", got)
	}
}

func TestFreePort(t *testing.T) {
	t.Parallel()

	t.Run("returns valid port", func(t *testing.T) {
		t.Parallel()
		port := testkit.FreePort(t)
		if port < 1 || port > 65535 {
			t.Fatalf("port %d out of range", port)
		}
	})

	t.Run("returns different ports on successive calls", func(t *testing.T) {
		t.Parallel()
		p1 := testkit.FreePort(t)
		p2 := testkit.FreePort(t)
		if p1 == p2 {
			t.Fatal("successive calls should return different ports")
		}
	})
}

func TestSortedKeys(t *testing.T) {
	t.Parallel()

	t.Run("returns keys in sorted order", func(t *testing.T) {
		t.Parallel()
		m := map[string]int{"c": 3, "a": 1, "b": 2}
		keys := testkit.SortedKeys(m)
		testkit.Equal(t, keys, []string{"a", "b", "c"}, "keys must be sorted")
	})

	t.Run("empty map returns empty slice", func(t *testing.T) {
		t.Parallel()
		keys := testkit.SortedKeys(map[string]int{})
		testkit.Len(t, keys, 0, "empty map must return empty slice")
	})
}

func TestDiffMap(t *testing.T) {
	t.Parallel()

	t.Run("detects added keys", func(t *testing.T) {
		t.Parallel()
		before := map[string]int{"a": 1}
		after := map[string]int{"a": 1, "b": 2}
		diff := testkit.DiffMap(before, after)
		testkit.Equal(t, diff.Added, map[string]int{"b": 2}, "must detect added key")
		testkit.Len(t, diff.Removed, 0, "nothing removed")
		testkit.Len(t, diff.Changed, 0, "nothing changed")
	})

	t.Run("detects removed keys", func(t *testing.T) {
		t.Parallel()
		before := map[string]int{"a": 1, "b": 2}
		after := map[string]int{"a": 1}
		diff := testkit.DiffMap(before, after)
		testkit.Equal(t, diff.Removed, map[string]int{"b": 2}, "must detect removed key")
		testkit.Len(t, diff.Added, 0, "nothing added")
	})

	t.Run("detects changed values", func(t *testing.T) {
		t.Parallel()
		before := map[string]int{"a": 1}
		after := map[string]int{"a": 99}
		diff := testkit.DiffMap(before, after)
		testkit.Equal(t, diff.Changed, map[string][2]int{"a": {1, 99}}, "must detect changed value")
	})

	t.Run("identical maps produce empty diff", func(t *testing.T) {
		t.Parallel()
		m := map[string]int{"a": 1}
		diff := testkit.DiffMap(m, m)
		testkit.Len(t, diff.Added, 0, "nothing added")
		testkit.Len(t, diff.Removed, 0, "nothing removed")
		testkit.Len(t, diff.Changed, 0, "nothing changed")
	})
}

type namedCase struct {
	name  string
	value int
}

func (c namedCase) Name() string { return c.name }

func TestTableTest(t *testing.T) {
	t.Parallel()

	t.Run("runs each case as subtest with correct values", func(t *testing.T) {
		t.Parallel()
		cases := []namedCase{
			{name: "case-a", value: 1},
			{name: "case-b", value: 2},
		}
		testkit.TableTest(t, cases, func(t *testing.T, tc namedCase) {
			t.Helper()
			testkit.True(t, tc.value > 0, "value must be positive")
		})
	})
}

// A wear that answers "nothing" for a stream must answer an empty sequence
// rather than the zero value, because ranging over a nil iterator panics —
// which takes the run down instead of letting the law it was worn for speak.
func TestEmptySeq(t *testing.T) {
	t.Parallel()

	t.Run("one-value sequences yield nothing", func(t *testing.T) {
		t.Parallel()
		full := slices.Values([]string{"a", "b"})
		count := 0
		for range testkit.EmptySeq(full) {
			count++
		}
		testkit.Equal(t, count, 0, "the empty sequence yields no element")
	})

	t.Run("two-value sequences yield nothing", func(t *testing.T) {
		t.Parallel()
		full := slices.All([]string{"a", "b"})
		count := 0
		for range testkit.EmptySeq2(full) {
			count++
		}
		testkit.Equal(t, count, 0, "the empty sequence yields no pair")
	})

	t.Run("the sequence handed in is not drained", func(t *testing.T) {
		t.Parallel()
		// Read for its type and nothing else: it is the real call the wear
		// stands in for, and draining it would run the very work the defect
		// exists to suppress.
		drained := 0
		full := func(yield func(int) bool) {
			drained++
			yield(1)
		}
		for range testkit.EmptySeq(full) {
			t.Fatal("nothing is yielded")
		}
		testkit.Equal(t, drained, 0, "the original is never started")
	})
}

// The stream defects a drain claim can be false against: an order reversed
// and an element dropped, and an element repeated. Both are collected shapes
// — order and length are properties of the whole sequence — and both must
// stop when the consumer does, or a bounded drain never returns.
func TestSeqDefects(t *testing.T) {
	t.Parallel()

	t.Run("faded reverses and drops the last", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, slices.Collect(testkit.FadedSeq(slices.Values([]int{1, 2, 3}))),
			[]int{3, 2}, "reversed, one short")
		testkit.Equal(t, slices.Collect(testkit.FadedSeq(slices.Values([]int{}))),
			[]int(nil), "an empty drain has nothing to drop")
	})

	t.Run("faded pairs keep their keys", func(t *testing.T) {
		t.Parallel()
		var keys []int
		for k := range testkit.FadedSeq2(slices.All([]string{"a", "b", "c"})) {
			keys = append(keys, k)
		}
		testkit.Equal(t, keys, []int{2, 1}, "the pair travels whole")
	})

	t.Run("doubled repeats every element", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, slices.Collect(testkit.DoubledSeq(slices.Values([]int{1, 2}))),
			[]int{1, 1, 2, 2}, "each element twice, in place")
		var got []string
		for _, v := range testkit.DoubledSeq2(slices.All([]string{"a"})) {
			got = append(got, v)
		}
		testkit.Equal(t, got, []string{"a", "a"}, "and the pair form with it")
	})

	t.Run("both stop when the consumer stops", func(t *testing.T) {
		t.Parallel()
		// The property that keeps a bounded drain bounded. A defect that
		// ignored the consumer's answer would run until the process died,
		// which is not a failing test — it is no test at all.
		for v := range testkit.DoubledSeq(slices.Values([]int{1, 2, 3})) {
			testkit.Equal(t, v, 1, "the first element, then the break")
			break
		}
		for range testkit.FadedSeq(slices.Values([]int{1, 2, 3})) {
			break
		}
		for range testkit.DoubledSeq2(slices.All([]int{1, 2, 3})) {
			break
		}
		for range testkit.FadedSeq2(slices.All([]int{1, 2, 3})) {
			break
		}
	})

	t.Run("doubled stops on the repeat too", func(t *testing.T) {
		t.Parallel()
		// The other half of the brake: a consumer that takes the element and
		// stops on its repeat. Both yields have to be answered, or a drain
		// that wanted one more element than the source holds runs on.
		seen := 0
		for range testkit.DoubledSeq(slices.Values([]int{1, 2})) {
			seen++
			if seen == 2 {
				break
			}
		}
		testkit.Equal(t, seen, 2, "the element and its repeat, then the stop")

		seen = 0
		for range testkit.DoubledSeq2(slices.All([]int{1, 2})) {
			seen++
			if seen == 2 {
				break
			}
		}
		testkit.Equal(t, seen, 2, "and the same for the pair form")
	})
}
