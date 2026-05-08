// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
)

// TestTypes is a smoke test that constructs each loader output type
// and asserts the basic invariants their consumers depend on. It
// guards against accidental field renames or removals.
func TestTypes(t *testing.T) {
	t.Parallel()

	t.Run("MethodInfo carries Name, Doc, Directives, Pos", func(t *testing.T) {
		t.Parallel()
		m := generator.MethodInfo{
			Name:       "Get",
			Doc:        "fetches by key",
			Directives: []directive.Directive{{Name: "errors"}},
		}
		testkit.Equal(t, m.Name, "Get", "Name")
		testkit.Equal(t, m.Doc, "fetches by key", "Doc")
		testkit.Len(t, m.Directives, 1, "Directives")
	})

	t.Run("InterfaceInfo carries TypeParams + Methods + Directives", func(t *testing.T) {
		t.Parallel()
		i := generator.InterfaceInfo{
			Name:       "Cache",
			Methods:    []generator.MethodInfo{{Name: "Get"}, {Name: "Put"}},
			TypeParams: []generator.TypeParamInfo{{Name: "K"}, {Name: "V"}},
		}
		testkit.Equal(t, i.Name, "Cache", "Name")
		testkit.Len(t, i.Methods, 2, "two methods")
		testkit.Len(t, i.TypeParams, 2, "two type params")
	})

	t.Run("StructInfo and FieldInfo cover builder shape", func(t *testing.T) {
		t.Parallel()
		s := generator.StructInfo{
			Name: "Counter",
			Fields: []generator.FieldInfo{
				{Name: "N", Exported: true},
				{Name: "tag", Exported: false, Tag: `json:"tag"`},
			},
		}
		testkit.Equal(t, s.Name, "Counter", "Name")
		testkit.True(t, s.Fields[0].Exported, "first field exported")
		testkit.False(t, s.Fields[1].Exported, "second field unexported")
		testkit.Equal(t, s.Fields[1].Tag, `json:"tag"`, "tag preserved")
	})

	t.Run("FieldData carries rendered FieldName/TypeStr/ZeroValue/IsError", func(t *testing.T) {
		t.Parallel()
		f := generator.FieldData{
			FieldName: "Result",
			TypeStr:   "string",
			ZeroValue: `""`,
		}
		testkit.Equal(t, f.FieldName, "Result", "FieldName")
		testkit.Equal(t, f.TypeStr, "string", "TypeStr")
		testkit.Equal(t, f.ZeroValue, `""`, "ZeroValue")
		testkit.False(t, f.IsError, "IsError defaults false")
	})

	t.Run("IterSeqInfo discriminates Seq vs Seq2", func(t *testing.T) {
		t.Parallel()
		seq := generator.IterSeqInfo{IsSeq: true, ValType: "int"}
		seq2 := generator.IterSeqInfo{IsSeq2: true, Seq2Error: true, ValType: "int", ErrType: "error"}
		testkit.True(t, seq.IsSeq && !seq.IsSeq2, "Seq discriminator")
		testkit.True(t, seq2.IsSeq2 && seq2.Seq2Error, "Seq2 with error")
		testkit.Equal(t, seq.ValType, "int", "Seq carries ValType")
		testkit.Equal(t, seq2.ValType, "int", "Seq2 carries ValType")
		testkit.Equal(t, seq2.ErrType, "error", "Seq2 carries ErrType")
	})
}
