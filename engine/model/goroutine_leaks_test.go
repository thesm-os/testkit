// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// These live in the external test package on purpose. The leak filter drops
// stacks whose frames are all framework code, and "go.thesmos.sh/testkit/
// engine/model." is on that list — so a goroutine started from an internal
// test is indistinguishable from framework noise and is filtered out before
// the reporting path is ever reached.
package model_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.thesmos.sh/testkit/engine/model"
)

// fakeTB captures what CheckGoroutineLeaks reports without failing the real
// test. Leak detection has to be driven against a subject that genuinely
// leaks, so the reporting side needs a stand-in.
type fakeTB struct {
	name   string
	errorf []string
	logf   []string
}

func (*fakeTB) Helper()                     {}
func (f *fakeTB) Name() string              { return f.name }
func (f *fakeTB) Errorf(s string, a ...any) { f.errorf = append(f.errorf, sprintf(s, a...)) }
func (f *fakeTB) Logf(s string, a ...any)   { f.logf = append(f.logf, sprintf(s, a...)) }

func sprintf(format string, a ...any) string { return fmt.Sprintf(format, a...) }

//nolint:paralleltest // goroutine leak detection requires exclusive goroutine control
func TestCheckGoroutineLeaks(t *testing.T) {
	// Deliberately not parallel, and neither are the subtests: the subject
	// samples process-global goroutine counts, so any test running
	// concurrently would show up as a leak here.

	t.Run("a clean function reports nothing", func(t *testing.T) { //nolint:paralleltest
		tb := &fakeTB{name: "clean"}
		model.CheckGoroutineLeaks(tb, "", func() {})
		if len(tb.errorf) != 0 {
			t.Fatalf("a function that leaks nothing must not report: %v", tb.errorf)
		}
	})

	// Goroutines that belong to the framework — rapid's workers, the testing
	// package's own — are filtered out, so a subject that only stirs those
	// must not be reported as leaking.
	t.Run("framework goroutines are not reported", func(t *testing.T) { //nolint:paralleltest
		tb := &fakeTB{name: "framework"}
		model.CheckGoroutineLeaks(tb, "", func() {
			done := make(chan struct{})
			go func() { close(done) }()
			<-done // fully joined before returning
		})
		if len(tb.errorf) != 0 {
			t.Fatalf("a joined goroutine is not a leak: %v", tb.errorf)
		}
	})

	t.Run("a genuine leak is reported without an artifact dir", func(t *testing.T) { //nolint:paralleltest
		tb := &fakeTB{name: "leaky"}
		block := make(chan struct{})
		t.Cleanup(func() { close(block) })

		model.CheckGoroutineLeaks(tb, "", func() {
			go func() { <-block }() // still parked when the check samples
			time.Sleep(20 * time.Millisecond)
		})
		if len(tb.errorf) == 0 {
			t.Fatal("a parked goroutine must be reported as leaked")
		}
		if !strings.Contains(tb.errorf[0], "AUTO-NO-GOROUTINE-LEAKS") {
			t.Fatalf("the report must name the law, got: %s", tb.errorf[0])
		}
		if len(tb.logf) != 0 {
			t.Fatalf("no artifact dir means no artifact log line, got: %v", tb.logf)
		}
	})

	t.Run("a genuine leak writes an artifact and links it", func(t *testing.T) { //nolint:paralleltest
		dir := t.TempDir()
		tb := &fakeTB{name: "leaky/with artifact"}
		block := make(chan struct{})
		t.Cleanup(func() { close(block) })

		model.CheckGoroutineLeaks(tb, dir, func() {
			go func() { <-block }()
			time.Sleep(20 * time.Millisecond)
		})
		if len(tb.errorf) == 0 {
			t.Fatal("a parked goroutine must be reported as leaked")
		}
		if !strings.Contains(tb.errorf[0], "goroutine stacks:") {
			t.Fatalf("the report must link the artifact, got: %s", tb.errorf[0])
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) == 0 {
			t.Fatalf("an artifact file must exist in %s (err=%v)", dir, err)
		}
		if !strings.HasSuffix(entries[0].Name(), "-goroutines.txt") {
			t.Fatalf("unexpected artifact name %q", entries[0].Name())
		}
		// The test name carries a slash; the filename must not.
		if strings.Contains(entries[0].Name(), "/") {
			t.Fatalf("the artifact name must be filesystem-safe, got %q", entries[0].Name())
		}
	})

	// An unwritable artifact dir must not swallow the leak report — the
	// diagnostic still has to reach the developer.
	t.Run("an unusable artifact dir still reports the leak", func(t *testing.T) { //nolint:paralleltest
		blocker := filepath.Join(t.TempDir(), "not-a-dir")
		if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
			t.Fatalf("setup: %v", err)
		}
		tb := &fakeTB{name: "leaky-nodir"}
		block := make(chan struct{})
		t.Cleanup(func() { close(block) })

		model.CheckGoroutineLeaks(tb, filepath.Join(blocker, "sub"), func() {
			go func() { <-block }()
			time.Sleep(20 * time.Millisecond)
		})
		if len(tb.errorf) == 0 {
			t.Fatal("a failed artifact write must not suppress the report")
		}
		if strings.Contains(tb.errorf[0], "goroutine stacks:") {
			t.Fatalf("no artifact was written, so none should be linked: %s", tb.errorf[0])
		}
	})
}
