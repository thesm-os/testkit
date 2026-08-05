// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"strings"
	"testing"
	"time"
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

func TestSplitGoroutineStacks(t *testing.T) {
	t.Parallel()

	t.Run("keys every section by goroutine ID", func(t *testing.T) {
		t.Parallel()
		stacks := splitGoroutineStacks([]byte(sampleStack))
		if len(stacks) != 4 {
			t.Fatalf("expected 4 sections, got %d", len(stacks))
		}
		for _, expected := range []uint64{1, 18, 42, 99} {
			if _, ok := stacks[expected]; !ok {
				t.Fatalf("missing goroutine ID %d", expected)
			}
		}
	})

	// Anything before the first "goroutine " header is preamble, not a
	// section: a dump that has been trimmed at the front must not be read as
	// an unnamed goroutine.
	t.Run("leading text that is not a header is discarded", func(t *testing.T) {
		t.Parallel()
		stacks := splitGoroutineStacks([]byte("truncated preamble\ngoroutine 7 [running]:\nmain.f()\n"))
		if len(stacks) != 1 {
			t.Fatalf("expected only the real section, got %d", len(stacks))
		}
		if _, ok := stacks[7]; !ok {
			t.Fatal("the section after the preamble must survive")
		}
	})
}

func TestFilterFrameworkGoroutines(t *testing.T) {
	t.Parallel()

	t.Run("removes framework goroutines", func(t *testing.T) {
		t.Parallel()
		ids := map[uint64]struct{}{18: {}, 42: {}, 99: {}}
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
		ids := map[uint64]struct{}{18: {}, 99: {}}
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
		ids := map[uint64]struct{}{42: {}}
		out := extractStacksForIDs([]byte(sampleStack), ids)
		if !strings.Contains(out, "leakyFunc") {
			t.Fatalf("expected leakyFunc in output: %s", out)
		}
		if strings.Contains(out, "pgregory.net/rapid") {
			t.Fatalf("should not contain rapid frames: %s", out)
		}
	})
}

// leakReport captures what CheckGoroutineLeaks says without failing the test.
type leakReport struct {
	name   string
	errorf []string
}

func (*leakReport) Helper()                     {}
func (r *leakReport) Name() string              { return r.name }
func (r *leakReport) Errorf(f string, a ...any) { r.errorf = append(r.errorf, fmt.Sprintf(f, a...)) }
func (*leakReport) Logf(string, ...any)         {}

// A goroutine that is genuinely still running but whose frames are all
// framework code is noise, not a leak. This test lives in the internal
// package precisely so its own goroutine looks like framework code — the
// package prefix is on the filter list.
//
//nolint:paralleltest // goroutine leak detection requires exclusive goroutine control
func TestCheckGoroutineLeaksFiltersFrameworkOnlyLeaks(t *testing.T) {
	block := make(chan struct{})
	t.Cleanup(func() { close(block) })

	r := &leakReport{name: "framework-only"}
	CheckGoroutineLeaks(r, "", func() {
		go func() { <-block }() // still parked when the check samples
		time.Sleep(20 * time.Millisecond)
	})

	if len(r.errorf) != 0 {
		t.Fatalf("a framework-only goroutine must not be reported: %v", r.errorf)
	}
}
