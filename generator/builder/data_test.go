// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package builder_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/builder"
)

func TestData(t *testing.T) {
	t.Parallel()

	t.Run("HasContent is false for zero-value Data", func(t *testing.T) {
		t.Parallel()
		var d builder.Data
		testkit.False(t, d.HasContent(), "no structs → skippable")
	})

	t.Run("HasContent is true when any struct is present", func(t *testing.T) {
		t.Parallel()
		d := builder.Data{Structs: []builder.StructData{{Name: "X"}}}
		testkit.True(t, d.HasContent(), "one struct is enough")
	})

	t.Run("FirstComparableField returns first basic-comparable field", func(t *testing.T) {
		t.Parallel()
		s := builder.StructData{
			Fields: []builder.FieldData{
				{Name: "List", IsSlice: true},
				{Name: "ID", IsBasicComparable: true},
				{Name: "Count", IsBasicComparable: true},
			},
		}
		got := s.FirstComparableField()
		testkit.True(t, got != nil, "non-nil when basic-comparable field exists")
		testkit.Equal(t, got.Name, "ID", "first by declaration order")
	})

	t.Run("FirstComparableField returns nil when none qualify", func(t *testing.T) {
		t.Parallel()
		s := builder.StructData{
			Fields: []builder.FieldData{
				{Name: "List", IsSlice: true},
				{Name: "Map", IsMap: true},
				{Name: "Stringer"}, // interface field — no IsBasicComparable
			},
		}
		testkit.True(t, s.FirstComparableField() == nil,
			"no basic-comparable field → nil (so {{with}} skips Mutate/Clone)")
	})

	t.Run("FirstComparableField skips slice/map/bytes/struct/pointer", func(t *testing.T) {
		t.Parallel()
		// All of these are NOT basic-comparable. Even though they
		// might have IsBasicComparable false, the predicate is
		// gated on the explicit flag — not on the absence of other
		// flags — so this test pins the contract.
		s := builder.StructData{
			Fields: []builder.FieldData{
				{Name: "Tags", IsSlice: true, IsBasicComparable: false},
				{Name: "Bytes", IsBytes: true, IsBasicComparable: false},
				{Name: "Map", IsMap: true, IsBasicComparable: false},
				{Name: "Nested", IsStruct: true, IsBasicComparable: false},
				{Name: "Owner", IsPointer: true, IsBasicComparable: false},
			},
		}
		testkit.True(t, s.FirstComparableField() == nil, "all non-comparable")
	})

	t.Run("EffectiveSample prefers TestSample when set", func(t *testing.T) {
		t.Parallel()
		f := builder.FieldData{SampleValue: `"basic"`, TestSample: `"generic"`}
		testkit.Equal(t, f.EffectiveSample(), `"generic"`,
			"generic instantiation overrides plain sample")
	})

	t.Run("EffectiveSample falls back to SampleValue when TestSample empty", func(t *testing.T) {
		t.Parallel()
		f := builder.FieldData{SampleValue: `"basic"`}
		testkit.Equal(t, f.EffectiveSample(), `"basic"`,
			"non-generic fields use SampleValue")
	})
}
