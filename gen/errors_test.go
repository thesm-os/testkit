// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gen_test

import (
	"errors"
	"go/token"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/gen"
)

func TestError(t *testing.T) {
	t.Parallel()

	t.Run("with valid position includes file and line", func(t *testing.T) {
		t.Parallel()
		pos := token.Position{Filename: "store.go", Line: 42, Column: 5}
		e := gen.Errorf(pos, "type %q not found", "Store")
		testkit.Assert(t, e.Error()).
			Contains("store.go", "must include filename").
			Contains("42", "must include line number").
			Contains(`type "Store" not found`, "must include message")
	})

	t.Run("without position omits file info", func(t *testing.T) {
		t.Parallel()
		e := gen.Errorf(token.Position{}, "something failed")
		testkit.Equal(t, e.Error(), "something failed", "must be plain message")
	})

	t.Run("with cause includes both message and cause", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("underlying")
		pos := token.Position{Filename: "store.go", Line: 10}
		e := gen.WrapErr(pos, cause, "loading failed")
		testkit.Assert(t, e.Error()).
			Contains("store.go", "must include filename").
			Contains("loading failed", "must include message").
			Contains("underlying", "must include cause")
	})

	t.Run("with cause but no position", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("underlying")
		e := gen.WrapErr(token.Position{}, cause, "loading failed")
		testkit.Equal(t, e.Error(), "loading failed: underlying", "must format message: cause")
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("root cause")
		e := gen.WrapErr(token.Position{}, cause, "wrapper")
		testkit.True(t, errors.Is(e, cause), "must unwrap to cause")
	})

	t.Run("Unwrap returns nil when no cause", func(t *testing.T) {
		t.Parallel()
		e := gen.Errorf(token.Position{}, "no cause")
		testkit.True(t, e.Unwrap() == nil, "must return nil without cause")
	})
}
