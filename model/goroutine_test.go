// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"strings"
	"testing"
)

const sampleStack = `goroutine 1 [running]:
main.main()
	/tmp/main.go:5 +0x25

goroutine 18 [chan receive]:
pgregory.net/rapid.(*worker).run(...)
	/go/pkg/mod/pgregory.net/rapid@v1.3.0/engine.go:100

goroutine 42 [sleep]:
time.Sleep(0x3b9aca00)
	/usr/local/go/src/runtime/time.go:195 +0x113
myapp.leakyFunc()
	/tmp/myapp/leak.go:10 +0x25

goroutine 99 [chan receive]:
pgregory.net/rapid.(*checkRunner).shrink(...)
	/go/pkg/mod/pgregory.net/rapid@v1.3.0/engine.go:200
`

func TestParseGoroutineIDs(t *testing.T) {
	t.Parallel()

	t.Run("extracts all IDs", func(t *testing.T) {
		t.Parallel()
		ids := parseGoroutineIDs([]byte(sampleStack))
		if len(ids) != 4 {
			t.Fatalf("expected 4 goroutine IDs, got %d", len(ids))
		}
		for _, expected := range []int{1, 18, 42, 99} {
			if _, ok := ids[expected]; !ok {
				t.Fatalf("missing goroutine ID %d", expected)
			}
		}
	})

	t.Run("empty input returns empty set", func(t *testing.T) {
		t.Parallel()
		ids := parseGoroutineIDs([]byte(""))
		if len(ids) != 0 {
			t.Fatalf("expected 0 IDs, got %d", len(ids))
		}
	})
}

func TestDiffGoroutineIDs(t *testing.T) {
	t.Parallel()

	t.Run("detects new goroutines", func(t *testing.T) {
		t.Parallel()
		start := map[int]struct{}{1: {}, 18: {}}
		end := map[int]struct{}{1: {}, 18: {}, 42: {}, 99: {}}
		diff := diffGoroutineIDs(start, end)
		if len(diff) != 2 {
			t.Fatalf("expected 2 new goroutines, got %d", len(diff))
		}
		if _, ok := diff[42]; !ok {
			t.Fatal("missing goroutine 42")
		}
		if _, ok := diff[99]; !ok {
			t.Fatal("missing goroutine 99")
		}
	})

	t.Run("empty diff when no change", func(t *testing.T) {
		t.Parallel()
		start := map[int]struct{}{1: {}, 18: {}}
		end := map[int]struct{}{1: {}, 18: {}}
		diff := diffGoroutineIDs(start, end)
		if len(diff) != 0 {
			t.Fatalf("expected 0 diff, got %d", len(diff))
		}
	})
}

func TestFilterFrameworkGoroutines(t *testing.T) {
	t.Parallel()

	t.Run("removes framework goroutines", func(t *testing.T) {
		t.Parallel()
		ids := map[int]struct{}{18: {}, 42: {}, 99: {}}
		remaining := filterFrameworkGoroutines([]byte(sampleStack), ids)
		// 18 and 99 have rapid frames; 42 has user code (leakyFunc).
		if len(remaining) != 1 {
			t.Fatalf("expected 1 remaining, got %d", len(remaining))
		}
		if _, ok := remaining[42]; !ok {
			t.Fatal("goroutine 42 should remain (user code)")
		}
	})

	t.Run("empty when all are framework", func(t *testing.T) {
		t.Parallel()
		ids := map[int]struct{}{18: {}, 99: {}}
		remaining := filterFrameworkGoroutines([]byte(sampleStack), ids)
		if len(remaining) != 0 {
			t.Fatalf("expected 0 remaining, got %d", len(remaining))
		}
	})
}

func TestExtractStacksForIDs(t *testing.T) {
	t.Parallel()

	t.Run("extracts specific goroutine stacks", func(t *testing.T) {
		t.Parallel()
		ids := map[int]struct{}{42: {}}
		out := extractStacksForIDs([]byte(sampleStack), ids)
		if !strings.Contains(out, "leakyFunc") {
			t.Fatalf("expected leakyFunc in output: %s", out)
		}
		if strings.Contains(out, "pgregory.net/rapid") {
			t.Fatalf("should not contain rapid frames: %s", out)
		}
	})
}

func TestCaptureAllStacks(t *testing.T) {
	t.Parallel()

	t.Run("captures current process stacks", func(t *testing.T) {
		t.Parallel()
		buf := captureAllStacks()
		if len(buf) == 0 {
			t.Fatal("expected non-empty stack output")
		}
		if !strings.Contains(string(buf), "goroutine") {
			t.Fatal("expected goroutine headers in output")
		}
	})
}
