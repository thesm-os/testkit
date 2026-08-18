// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/suite"
)

// TestSignaturePrimitives pins each primitive's two sides: green on the
// conforming behaviour, red — naming the method — on the defect it exists
// to catch.
func TestSignaturePrimitives(t *testing.T) {
	t.Parallel()

	red := func(t *testing.T, name string, drive func(tb testing.TB)) {
		t.Helper()
		f := testkit.NewFailableTB().WithGoexit()
		done := make(chan struct{})
		go func() { defer close(done); drive(f) }()
		<-done
		if !f.Failed() || !strings.Contains(f.Msg(), "Method") {
			t.Errorf("%s must fail naming the method, got failed=%v msg=%q", name, f.Failed(), f.Msg())
		}
	}

	t.Run("Survives", func(t *testing.T) {
		t.Parallel()
		suite.Survives(t, "Method", func(context.Context) {})
		red(t, "a panicking method", func(tb testing.TB) {
			suite.Survives(tb, "Method", func(context.Context) { panic("planted") })
		})
	})

	t.Run("ReportsCancelled", func(t *testing.T) {
		t.Parallel()
		suite.ReportsCancelled(t, "Method", func(ctx context.Context) error { return ctx.Err() })
		red(t, "a context-deaf method", func(tb testing.TB) {
			suite.ReportsCancelled(tb, "Method", func(context.Context) error { return nil })
		})
	})

	t.Run("ReportsDeadlineExceeded", func(t *testing.T) {
		t.Parallel()
		suite.ReportsDeadlineExceeded(t, "Method", func(ctx context.Context) error { return ctx.Err() })
		red(t, "a deadline-deaf method", func(tb testing.TB) {
			suite.ReportsDeadlineExceeded(tb, "Method", func(context.Context) error { return nil })
		})
	})

	t.Run("ToleratesNilContext", func(t *testing.T) {
		t.Parallel()
		suite.ToleratesNilContext(t, "Method", func(context.Context) error {
			return errors.New("refused, politely")
		})
		red(t, "a nil-dereferencing method", func(tb testing.TB) {
			suite.ToleratesNilContext(tb, "Method", func(ctx context.Context) error {
				return ctx.Err() // the defect: no nil guard
			})
		})
	})
}
