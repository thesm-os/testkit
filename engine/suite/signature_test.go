// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"context"
	"errors"
	"reflect"
	"runtime"
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

	t.Run("ToleratesNilArgument", func(t *testing.T) {
		t.Parallel()
		suite.ToleratesNilArgument(t, "Method", func(context.Context) error {
			return errors.New("refused, politely")
		})
		red(t, "a method that accepts nil and carries on", func(tb testing.TB) {
			suite.ToleratesNilArgument(tb, "Method", func(context.Context) error { return nil })
		})
		red(t, "a method that dereferences the nil", func(tb testing.TB) {
			suite.ToleratesNilArgument(tb, "Method", func(context.Context) error {
				_ = *nilPayload // the defect: no nil guard
				return nil
			})
		})
	})
}

// nilPayload stands in for an argument that arrived nil.
//
// A package variable rather than a local because a local nil is one the
// compiler's own analysis can fold, and folding it away would leave the
// dereference arm asserting nothing. At run time a nil argument is
// exactly this: a pointer whose value nothing here can see.
var nilPayload *int

// guardCase ties one exported guard name to the function it claims to
// name.
type guardCase struct {
	name string
	fn   func(testing.TB, string, func(context.Context) error)
}

func (c guardCase) Name() string { return c.name }

// TestGuardNamesAreTheFunctions holds each Guard constant to the
// identifier of the function it names.
//
// The generator emits `suite.ReportsCancelled(...)` from the constant
// rather than from a literal, so the constant is the only thing standing
// between a rename here and a generated file that calls a function this
// package no longer exports — a break a consumer meets in their own
// build, long after the rename. Reading the name back off the function
// value is what makes the constant a claim rather than a comment.
func TestGuardNamesAreTheFunctions(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []guardCase{
		{suite.GuardCancelled, suite.ReportsCancelled},
		{suite.GuardDeadline, suite.ReportsDeadlineExceeded},
		{suite.GuardNilContext, suite.ToleratesNilContext},
		{suite.GuardNilArgument, suite.ToleratesNilArgument},
	}, func(t *testing.T, tc guardCase) {
		full := runtime.FuncForPC(reflect.ValueOf(tc.fn).Pointer()).Name()
		testkit.Equal(t, full[strings.LastIndex(full, ".")+1:], tc.name,
			"the constant is what the generator spells; a drifted one emits a call to nothing")
	})
}
