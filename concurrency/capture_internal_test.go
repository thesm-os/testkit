// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package concurrency

import (
	"bytes"
	"testing"
)

// The grow-to-fit loop is where a stack dump silently loses goroutines, so it
// is tested against a dump function whose size is chosen rather than observed.
func TestCaptureStacksGrowth(t *testing.T) {
	t.Parallel()

	// filling returns a dump func that reports `size` bytes, clipped to
	// whatever buffer it is handed — the same contract as runtime.Stack.
	filling := func(size int) (func([]byte) int, *int) {
		calls := 0
		return func(b []byte) int {
			calls++
			return min(size, len(b))
		}, &calls
	}

	t.Run("a dump that fits is returned as-is", func(t *testing.T) {
		t.Parallel()
		dump, calls := filling(1024)
		got := captureStacks(dump)
		if len(got) != 1024 {
			t.Fatalf("expected the reported length, got %d", len(got))
		}
		if *calls != 1 {
			t.Fatalf("a fitting dump needs one call, got %d", *calls)
		}
	})

	t.Run("the buffer doubles until the dump fits", func(t *testing.T) {
		t.Parallel()
		// Just past 2MB: one growth to 2MB still clips, the next to 4MB fits.
		dump, calls := filling(2<<20 + 1)
		got := captureStacks(dump)
		if len(got) != 2<<20+1 {
			t.Fatalf("expected the full dump after growth, got %d", len(got))
		}
		if *calls != 3 {
			t.Fatalf("1MB → 2MB → 4MB is three calls, got %d", *calls)
		}
		if bytes.Contains(got, []byte("TRUNCATED")) {
			t.Fatal("a dump that fits below the cap must not be marked truncated")
		}
	})

	// At the cap the loss is real, and the marker is the only way a caller
	// can tell a complete dump from a clipped one.
	t.Run("a dump larger than the cap is marked truncated", func(t *testing.T) {
		t.Parallel()
		dump, _ := filling(maxStackBuf * 2)
		got := captureStacks(dump)
		if !bytes.HasSuffix(got, []byte("\n... TRUNCATED\n")) {
			t.Fatal("a clipped dump must announce itself")
		}
		if len(got) <= maxStackBuf {
			t.Fatalf("the capped buffer must still carry its %d bytes, got %d", maxStackBuf, len(got))
		}
	})
}
