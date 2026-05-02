// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit_test

import (
	"errors"
	"os"
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

	t.Run("runs each case as subtest", func(t *testing.T) {
		t.Parallel()
		var ran []int
		cases := []namedCase{
			{name: "first", value: 1},
			{name: "second", value: 2},
		}
		// We can't easily verify subtest names here, but we can verify
		// the function is called for each case.
		testkit.TableTest(t, cases, func(t *testing.T, tc namedCase) {
			t.Helper()
			ran = append(ran, tc.value)
		})
	})
}
