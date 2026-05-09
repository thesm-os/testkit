// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package concurrency

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// ConcurrentStress spawns goroutines goroutines, each calling work(g, i) for
// iterations iterations. It waits for all goroutines to finish before
// returning. The caller owns per-goroutine state — use g (goroutine index)
// and i (iteration index) to partition.
//
//	testkit.ConcurrentStress(t, 8, 100, func(g, i int) {
//	    store.Put(ctx, fmt.Sprintf("key-%d-%d", g, i), "v")
//	})
func ConcurrentStress(tb testing.TB, goroutines, iterations int, work func(g, i int)) {
	tb.Helper()
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Go(func() {
			for i := range iterations {
				work(g, i)
			}
		})
	}
	wg.Wait()
}

// GoroutineLeak returns a cleanup function that fails the test if any
// goroutines started after the call to GoroutineLeak are still running at
// cleanup time. Detection is ID-based (not count-based), so it is safe to
// use in parallel tests where unrelated goroutines may exist.
//
// The cleanup function polls for up to 100ms with 5ms intervals to allow
// goroutines to shut down naturally before failing.
//
//	cleanup := testkit.GoroutineLeak(t)
//	defer cleanup()
//	go worker(ctx) // must finish before test ends
func GoroutineLeak(tb testing.TB) func() {
	tb.Helper()
	before := CaptureGoroutineIDs()
	return func() {
		tb.Helper()
		deadline := time.Now().Add(500 * time.Millisecond)
		for {
			after := CaptureGoroutineIDs()
			leaked := DiffGoroutineIDs(before, after)
			if len(leaked) == 0 {
				return
			}
			if time.Now().After(deadline) {
				tb.Fatalf("GoroutineLeak: %d goroutine(s) still running: %v", len(leaked), leaked)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
}

// CaptureGoroutineIDs returns the set of currently-live goroutine
// IDs parsed from runtime.Stack output. The buffer grows from 1MB
// to an 8MB cap so test processes with many goroutines don't
// silently truncate near the end of the dump.
//
// Pairs with [DiffGoroutineIDs] for before/after leak detection,
// and with [CaptureGoroutineStacks] when callers need per-goroutine
// stack text (e.g. framework-frame filtering, artifact emission).
func CaptureGoroutineIDs() map[uint64]struct{} {
	return parseGoroutineIDs(CaptureGoroutineStacks())
}

// CaptureGoroutineStacks returns runtime.Stack output for every
// currently-live goroutine. Uses grow-to-fit buffering (1MB → 8MB
// cap) so callers that need to walk per-goroutine call frames
// don't lose stacks past the 1MB mark.
//
// At the cap the output is suffixed with "... TRUNCATED" so callers
// can detect (and warn about) truncated captures.
func CaptureGoroutineStacks() []byte {
	buf := make([]byte, 1<<20) // 1MB
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return buf[:n]
		}
		if len(buf) >= 8<<20 { // 8MB cap
			return append(buf[:n], "\n... TRUNCATED\n"...)
		}
		buf = make([]byte, len(buf)*2)
	}
}

// DiffGoroutineIDs returns IDs present in after but not in before.
// Order is unspecified — callers that need stable order should sort
// the result.
func DiffGoroutineIDs(before, after map[uint64]struct{}) []uint64 {
	var leaked []uint64
	for id := range after {
		if _, ok := before[id]; !ok {
			leaked = append(leaked, id)
		}
	}
	return leaked
}

// parseGoroutineIDs extracts goroutine IDs from runtime.Stack output.
func parseGoroutineIDs(stack []byte) map[uint64]struct{} {
	ids := make(map[uint64]struct{})
	for line := range strings.SplitSeq(string(stack), "\n") {
		if !strings.HasPrefix(line, "goroutine ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		id, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		ids[id] = struct{}{}
	}
	return ids
}

// Timeout returns a [context.Context] derived from tb.Context() with the
// given deadline. If the deadline fires before the test completes, the test
// fails with a clear message. The cancel function is registered via
// tb.Cleanup — callers do not need to defer it.
//
//	ctx := testkit.Timeout(t, 5*time.Second)
//	result, err := slowOperation(ctx)
func Timeout(tb testing.TB, d time.Duration) context.Context {
	tb.Helper()
	ctx, cancel := context.WithTimeout(tb.Context(), d)
	tb.Cleanup(cancel)
	return ctx
}
