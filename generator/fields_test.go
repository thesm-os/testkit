// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

func TestFields(t *testing.T) {
	t.Parallel()

	t.Run("BuildResultFields names single non-error as Result", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type I interface { F() (string, error) }
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.BuildResultFields(methodSig(t, iface, "F").Results(), tracker)
		testkit.Len(t, got, 2, "two result fields")
		testkit.Equal(t, got[0].FieldName, "Result", "single non-error → Result")
		testkit.Equal(t, got[0].TypeStr, "string", "type rendered")
		testkit.Equal(t, got[1].FieldName, "Err", "error → Err")
		testkit.True(t, got[1].IsError, "IsError flag set on error result")
	})

	t.Run("BuildResultFields numbers multiple non-error returns", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type I interface { F() (int, int, string, error) }
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.BuildResultFields(methodSig(t, iface, "F").Results(), tracker)
		testkit.Len(t, got, 4, "four result fields")
		testkit.Equal(t, got[0].FieldName, "Result0", "first numbered")
		testkit.Equal(t, got[1].FieldName, "Result1", "second numbered")
		testkit.Equal(t, got[2].FieldName, "Result2", "third numbered")
		testkit.Equal(t, got[3].FieldName, "Err", "error always Err")
		testkit.True(t, got[3].IsError, "IsError on error result")
	})

	t.Run("BuildResultFields preserves named return identifiers", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type I interface { F() (old string, new string, err error) }
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.BuildResultFields(methodSig(t, iface, "F").Results(), tracker)
		testkit.Equal(t, got[0].FieldName, "Old", "title-cased name")
		testkit.Equal(t, got[1].FieldName, "New", "title-cased name")
		testkit.Equal(t, got[2].FieldName, "Err", "title-cased name")
		testkit.True(t, got[2].IsError, "IsError on error")
	})

	t.Run("BuildResultFields promotes initialisms in named returns", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type I interface { F() (id string, count int) }
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.BuildResultFields(methodSig(t, iface, "F").Results(), tracker)
		testkit.Equal(t, got[0].FieldName, "ID", "id → ID")
		testkit.Equal(t, got[1].FieldName, "Count", "count → Count")
	})

	t.Run("BuildResultFields returns nil for empty tuple", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type I interface { F() }
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.BuildResultFields(methodSig(t, iface, "F").Results(), tracker)
		testkit.True(t, got == nil, "empty result tuple → nil")
	})

	t.Run("BuildParamFields skips ctx by default", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface { F(ctx context.Context, key string) error }
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.BuildParamFields(methodSig(t, iface, "F").Params(), tracker, false, false)
		testkit.Len(t, got, 1, "ctx skipped")
		testkit.Equal(t, got[0].FieldName, "Key", "remaining param title-cased")
	})

	t.Run("BuildParamFields keeps ctx when keepContext=true", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface { F(ctx context.Context, key string) error }
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.BuildParamFields(methodSig(t, iface, "F").Params(), tracker, false, true)
		testkit.Len(t, got, 2, "ctx kept")
		testkit.Equal(t, got[0].FieldName, "Ctx", "ctx → Ctx")
	})

	t.Run("BuildParamFields synthesizes pN for unnamed params", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type I interface { F(string, int) }
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.BuildParamFields(methodSig(t, iface, "F").Params(), tracker, false, true)
		testkit.Len(t, got, 2, "two synthesized params")
		testkit.Equal(t, got[0].FieldName, "P0", "first synthesized")
		testkit.Equal(t, got[1].FieldName, "P1", "second synthesized")
	})
}
