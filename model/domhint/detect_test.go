// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package domhint_test

import (
	"io"
	"reflect"
	"testing"

	"pgregory.net/rapid"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/model/domhint"
)

type publicStruct struct {
	A string
	B int
}

type privateStruct struct {
	a string //nolint:unused // fixture for unexported-field detection
}

type unexportedAlias = privateStruct

func TestRequiresHint(t *testing.T) {
	t.Parallel()

	t.Run("nil type returns false", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, domhint.RequiresHint(nil), "nil")
	})

	t.Run("scalar kinds are reflection-generatable", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, domhint.RequiresHint(reflect.TypeFor[string]()), "string")
		testkit.False(t, domhint.RequiresHint(reflect.TypeFor[int]()), "int")
		testkit.False(t, domhint.RequiresHint(reflect.TypeFor[float64]()), "float64")
		testkit.False(t, domhint.RequiresHint(reflect.TypeFor[bool]()), "bool")
	})

	t.Run("interface kind requires a hint", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, domhint.RequiresHint(reflect.TypeFor[io.Reader]()), "io.Reader")
	})

	t.Run("func, chan, unsafe.Pointer kinds require a hint", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, domhint.RequiresHint(reflect.TypeFor[func()]()), "func")
		testkit.True(t, domhint.RequiresHint(reflect.TypeFor[chan int]()), "chan")
	})

	t.Run("structs with only exported fields are reflective", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, domhint.RequiresHint(reflect.TypeFor[publicStruct]()), "all-public struct")
	})

	t.Run("structs with unexported fields require a hint", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, domhint.RequiresHint(reflect.TypeFor[privateStruct]()), "private-field struct")
		testkit.True(t, domhint.RequiresHint(reflect.TypeFor[unexportedAlias]()), "alias preserves rule")
	})

	t.Run("composite types delegate to element type", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, domhint.RequiresHint(reflect.TypeFor[[]string]()), "slice of string")
		testkit.False(t, domhint.RequiresHint(reflect.TypeFor[*string]()), "pointer to string")
		testkit.True(t, domhint.RequiresHint(reflect.TypeFor[[]privateStruct]()), "slice of opaque")
		testkit.True(t, domhint.RequiresHint(reflect.TypeFor[*privateStruct]()), "pointer to opaque")
		testkit.True(t, domhint.RequiresHint(reflect.TypeFor[map[string]privateStruct]()), "map value opaque")
		testkit.True(t, domhint.RequiresHint(reflect.TypeFor[map[privateStruct]string]()), "map key opaque")
	})
}

func TestResolve(t *testing.T) {
	t.Parallel()

	t.Run("registered hint returns hint plus name", func(t *testing.T) {
		t.Parallel()
		r := domhint.NewRegistry()
		domhint.Register(r, "ids.RunID", rapid.Just(runID("x")))
		h, name, needs := domhint.Resolve(r, reflect.TypeFor[runID]())
		testkit.True(t, h != nil, "hint returned")
		testkit.Equal(t, name, "ids.RunID", "display name")
		testkit.True(t, needs, "registration counts as needs")
	})

	t.Run("opaque type without hint flags needs-hint", func(t *testing.T) {
		t.Parallel()
		r := domhint.NewRegistry()
		h, _, needs := domhint.Resolve(r, reflect.TypeFor[io.Reader]())
		testkit.True(t, h == nil, "no hint")
		testkit.True(t, needs, "needs hint reported")
	})

	t.Run("reflective type without hint returns no-action", func(t *testing.T) {
		t.Parallel()
		r := domhint.NewRegistry()
		h, _, needs := domhint.Resolve(r, reflect.TypeFor[string]())
		testkit.True(t, h == nil, "no hint")
		testkit.False(t, needs, "rapid.Make handles strings")
	})

	t.Run("nil type yields no-action", func(t *testing.T) {
		t.Parallel()
		_, _, needs := domhint.Resolve(nil, nil)
		testkit.False(t, needs, "nil safe")
	})
}
