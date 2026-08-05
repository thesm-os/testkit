// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"errors"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model"
)

func TestFailureKindString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind model.FailureKind
		want string
	}{
		{model.FailureUnclassified, "unclassified"},
		{model.FailureStructural, "structural"},
		{model.FailureSemantic, "semantic"},
		{model.FailureInvariant, "invariant"},
		{model.FailureLiveness, "liveness"},
		{model.FailureKind(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			testkit.Equal(t, tt.kind.String(), tt.want, "String()")
		})
	}
}

func TestFailureError(t *testing.T) {
	t.Parallel()

	t.Run("action failure without LawID", func(t *testing.T) {
		t.Parallel()
		f := &model.Failure{
			Kind:    model.FailureSemantic,
			StepRan: model.StepID{Index: 7},
			Err:     errors.New("mismatch"),
		}
		got := f.Error()
		testkit.True(t, strings.Contains(got, "[semantic]"), "contains kind")
		testkit.True(t, strings.Contains(got, "step 7"), "contains step")
		testkit.True(t, strings.Contains(got, "mismatch"), "contains error message")
	})

	t.Run("law failure with LawID", func(t *testing.T) {
		t.Parallel()
		f := &model.Failure{
			Kind:    model.FailureInvariant,
			LawID:   "AUTO-READ-AFTER-WRITE",
			StepRan: model.StepID{Index: 3},
			Err:     errors.New("diverged"),
		}
		got := f.Error()
		testkit.True(t, strings.Contains(got, "AUTO-READ-AFTER-WRITE"), "contains law ID")
		testkit.True(t, strings.Contains(got, "[invariant]"), "contains kind")
	})

	t.Run("failure with REQID prefixes the kind", func(t *testing.T) {
		t.Parallel()
		f := &model.Failure{
			Kind:    model.FailureStructural,
			REQID:   "REQ-PKG-FOO-001",
			LawID:   "CUSTOM-LAW",
			StepRan: model.StepID{Index: 0},
			Err:     errors.New("broke"),
		}
		got := f.Error()
		testkit.True(t, strings.Contains(got, "REQ-PKG-FOO-001"), "contains REQID")
		testkit.True(t, strings.Contains(got, "structural"), "contains kind")
	})

	t.Run("nil Err renders as <nil>", func(t *testing.T) {
		t.Parallel()
		f := &model.Failure{
			Kind:    model.FailureUnclassified,
			StepRan: model.StepID{Index: 0},
			Err:     nil,
		}
		got := f.Error()
		testkit.True(t, strings.Contains(got, "<nil>"), "nil error renders as <nil>")
	})
}

func TestFailureUnwrap(t *testing.T) {
	t.Parallel()

	t.Run("unwraps underlying error", func(t *testing.T) {
		t.Parallel()
		underlying := errors.New("root cause")
		f := &model.Failure{Err: underlying}
		testkit.ErrorIs(t, f, underlying, "Unwrap returns underlying")
	})

	t.Run("nil Err unwraps to nil", func(t *testing.T) {
		t.Parallel()
		f := &model.Failure{Err: nil}
		testkit.True(t, f.Unwrap() == nil, "Unwrap of nil Err is nil")
	})
}
