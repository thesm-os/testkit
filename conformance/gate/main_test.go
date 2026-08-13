// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate_test

import (
	"os"
	"runtime"
	"testing"
)

// TestMain pins this binary to one processor. go/types resolves an Alias
// lazily through an unsynchronized memoization, and the frontend's parallel
// package loading trips the race detector on it roughly every third full
// corpus load (the root cause is golang/go, out of both
// repositories' reach, and the gotypesalias knob is no longer read). On one
// processor the loader's goroutines run far enough apart that the detector's
// shadow history no longer pairs the accesses, which is containment rather
// than a fix — remove this when the upstream resolution lands.
func TestMain(m *testing.M) {
	runtime.GOMAXPROCS(1)
	os.Exit(m.Run())
}
