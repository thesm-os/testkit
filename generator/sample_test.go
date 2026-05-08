// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package generator_test

import (
	"go/types"
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator"
)

func TestSampleBasicValue(t *testing.T) {
	t.Parallel()

	t.Run("string field renders as test-<fieldname>", func(t *testing.T) {
		t.Parallel()
		s := types.Typ[types.String]
		got := generator.SampleBasicValue(s, "Name")
		testkit.Equal(t, got, `"test-name"`, "lowercased seeded string")
	})

	t.Run("integer renders as 42", func(t *testing.T) {
		t.Parallel()
		got := generator.SampleBasicValue(types.Typ[types.Int], "")
		testkit.Equal(t, got, "42", "non-zero int sample")
	})

	t.Run("bool renders as true", func(t *testing.T) {
		t.Parallel()
		got := generator.SampleBasicValue(types.Typ[types.Bool], "")
		testkit.Equal(t, got, "true", "non-zero bool sample")
	})

	t.Run("float renders as 3.14", func(t *testing.T) {
		t.Parallel()
		got := generator.SampleBasicValue(types.Typ[types.Float64], "")
		testkit.Equal(t, got, "3.14", "non-zero float sample")
	})
}

func TestSampleValueOf(t *testing.T) {
	t.Parallel()

	const src = `package p
type Item struct {
	ID   string
	Name string
}
type I interface {
	S(s string)
	I(i int)
	Sl(s []int)
	M(m map[string]int)
	P(p *Item)
	St(it Item)
	C(c chan int)
}
`
	iface, tracker := loadIface(t, src, "I")

	// fieldName seeds string-sample contents — the param's source name
	// is used here so the expected value is predictable.
	cases := map[string]string{
		"S":  `"test-s"`,
		"I":  "42",
		"Sl": "[]int{42}",
		"M":  `map[string]int{"test-m": 42}`,
		"P":  `&Item{ID: "test-id"}`,
		"St": `Item{ID: "test-id"}`,
		"C":  "nil", // chan → nil
	}
	for name, want := range cases {
		sig := methodSig(t, iface, name)
		paramName := sig.Params().At(0).Name()
		got := generator.SampleValueOf(sig.Params().At(0).Type(), paramName, tracker)
		testkit.Equal(t, got, want, "SampleValueOf "+name)
	}
}

func TestSampleValueOf_StructLiteral(t *testing.T) {
	t.Parallel()

	t.Run("struct populates first exported basic field", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type Item struct {
	ID   string
	Inner *Item
}
type I interface { F() Item }
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.SampleValueOf(methodSig(t, iface, "F").Results().At(0).Type(), "", tracker)
		testkit.Equal(t, got, `Item{ID: "test-id"}`, "first basic field seeded")
	})

	t.Run("struct without basic fields falls back to zero", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type Inner struct{}
type Outer struct {
	X Inner
}
type I interface { F() Outer }
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.SampleValueOf(methodSig(t, iface, "F").Results().At(0).Type(), "", tracker)
		testkit.Equal(t, got, "Outer{}", "no basic field → zero literal")
	})
}
