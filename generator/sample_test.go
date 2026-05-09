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

	t.Run("non-string/bool/int/float kinds render as 0", func(t *testing.T) {
		t.Parallel()
		// Complex falls through to the default `0` literal — fine for
		// the rare struct field of complex type.
		got := generator.SampleBasicValue(types.Typ[types.Complex64], "")
		testkit.Equal(t, got, "0", "default fallback for unhandled kinds")
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

func TestSampleForConcreteType(t *testing.T) {
	t.Parallel()

	t.Run("known basic name dispatches to the candidate's Sample", func(t *testing.T) {
		t.Parallel()
		// "string" is in DefaultConcreteTypes; its Sample seeds via fieldName.
		testkit.Equal(t, generator.SampleForConcreteType("string", "Name"),
			`"test-name"`, "string sample uses lowerASCII(fieldName)")
		testkit.Equal(t, generator.SampleForConcreteType("int", ""), "42",
			"int sample is the canonical 42")
	})

	t.Run("slice prefix recurses on the element name", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, generator.SampleForConcreteType("[]int", ""),
			"[]int{42}", "slice wraps element sample")
	})

	t.Run("unknown name falls back to typed-zero literal", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, generator.SampleForConcreteType("foo.Bar", ""),
			"foo.Bar{}", "unknown type renders zero composite")
	})
}

func TestZeroParamExprs(t *testing.T) {
	t.Parallel()

	t.Run("ctx maps to t.Context() and other params zero out", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type Item struct{ ID string }
type I interface {
	F(ctx context.Context, key string, item Item) error
}
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.ZeroParamExprs(methodSig(t, iface, "F"), tracker)
		testkit.Equal(t, len(got), 3, "three params kept")
		testkit.Equal(t, got[0], "t.Context()", "ctx replaced")
		testkit.Equal(t, got[1], `""`, "string zero")
		testkit.Equal(t, got[2], "Item{}", "struct zero")
	})

	t.Run("variadic last param drops out", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
	F(ctx context.Context, ids ...string) error
}
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.ZeroParamExprs(methodSig(t, iface, "F"), tracker)
		testkit.Equal(t, len(got), 1, "variadic last excluded")
		testkit.Equal(t, got[0], "t.Context()", "only ctx remains")
	})
}

func TestSampleParamExprs(t *testing.T) {
	t.Parallel()

	t.Run("ctx maps to t.Context() and others get sample literals", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
	F(ctx context.Context, key string, n int) error
}
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.SampleParamExprs(methodSig(t, iface, "F"), tracker)
		testkit.Equal(t, got[0], "t.Context()", "ctx replaced")
		testkit.Equal(t, got[1], `"test-key"`, "string sample seeded by name")
		testkit.Equal(t, got[2], "42", "int sample")
	})

	t.Run("anonymous params synthesize names via ParamName", func(t *testing.T) {
		t.Parallel()
		// All params unnamed; the first non-ctx param falls back to "p1".
		const src = `package p
import "context"
type I interface {
	F(context.Context, string) error
}
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.SampleParamExprs(methodSig(t, iface, "F"), tracker)
		testkit.Equal(t, got[0], "t.Context()", "ctx slot")
		testkit.Equal(t, got[1], `"test-p1"`, "synthesized name p1 seeds the sample")
	})

	t.Run("variadic last param drops out", func(t *testing.T) {
		t.Parallel()
		const src = `package p
import "context"
type I interface {
	F(ctx context.Context, ids ...string) error
}
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.SampleParamExprs(methodSig(t, iface, "F"), tracker)
		testkit.Equal(t, len(got), 1, "variadic excluded")
	})
}

func TestSampleValueOf_NamedAndPointerEdges(t *testing.T) {
	t.Parallel()

	t.Run("named basic wraps the underlying sample in a conversion", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type Status int
type I interface { F(s Status) }
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.SampleValueOf(methodSig(t, iface, "F").Params().At(0).Type(), "s", tracker)
		testkit.Equal(t, got, "Status(42)", "named basic → Type(<sample>)")
	})

	t.Run("named slice recurses through the underlying", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type Names []string
type I interface { F(n Names) }
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.SampleValueOf(methodSig(t, iface, "F").Params().At(0).Type(), "n", tracker)
		testkit.Equal(t, got, `[]string{"test-n"}`, "named slice unwraps to underlying")
	})

	t.Run("pointer to non-struct renders as nil", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type I interface { F(p *string) }
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.SampleValueOf(methodSig(t, iface, "F").Params().At(0).Type(), "p", tracker)
		testkit.Equal(t, got, "nil", "pointer to basic type → nil")
	})

	t.Run("array renders the typed-zero with a single seed", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type I interface { F(a [3]int) }
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.SampleValueOf(methodSig(t, iface, "F").Params().At(0).Type(), "a", tracker)
		testkit.Equal(t, got, "[3]int{1}", "array seeds first element")
	})
}

func TestSampleValueOf_TypeParam(t *testing.T) {
	t.Parallel()
	// Type-parameter types resolve to a concrete only at the
	// instantiation site; SampleValueOf renders the universal
	// `*new(T)` zero idiom that consumers' SubstituteTypeParams pass
	// can rewrite to the concrete instantiation.
	const src = `package p
type Gen[T any] interface { F(t T) }
`
	iface, tracker := loadIface(t, src, "Gen")
	got := generator.SampleValueOf(methodSig(t, iface, "F").Params().At(0).Type(), "t", tracker)
	testkit.Equal(t, got, "*new(T)", "type-param renders as *new(T)")
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

	t.Run("struct skips unexported and non-basic fields, lands on first exported basic", func(t *testing.T) {
		t.Parallel()
		const src = `package p
type Inner struct{}
type Item struct {
	internal string
	Nested   Inner
	ID       string
}
type I interface { F() Item }
`
		iface, tracker := loadIface(t, src, "I")
		got := generator.SampleValueOf(methodSig(t, iface, "F").Results().At(0).Type(), "", tracker)
		testkit.Equal(t, got, `Item{ID: "test-id"}`,
			"unexported + non-basic skipped, lands on ID")
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
