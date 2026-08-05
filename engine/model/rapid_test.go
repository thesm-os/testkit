// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"strings"
	"testing"
	"unicode"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/engine/model"
)

// The re-exports in rapid.go exist so generated code and consumer extensions
// never import pgregory.net/rapid directly, keeping it out of consumer go.mod
// files. They are eighty-nine near-identical one-line delegations, which makes
// mis-delegation — Min forwarding to Max, Int32 forwarding to Int64 — the
// realistic defect. Asserting a returned generator is non-nil would not catch
// any of that, so the bounded generators are driven through a property and
// checked against their own bounds.

func TestRapidBoolAndBytes(t *testing.T) {
	t.Parallel()

	t.Run("Bool draws both values", func(t *testing.T) {
		t.Parallel()
		seen := map[bool]bool{}
		model.Check(t, func(rt *model.T) {
			seen[model.Bool().Draw(rt, "b")] = true
		})
		testkit.Equal(t, len(seen), 2, "Bool must be able to draw true and false")
	})

	t.Run("ByteMin respects its lower bound", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.ByteMin(200).Draw(rt, "v")
			testkit.True(t, v >= 200, "ByteMin must not draw below lo")
		})
	})

	t.Run("ByteMax respects its upper bound", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.ByteMax(10).Draw(rt, "v")
			testkit.True(t, v <= 10, "ByteMax must not draw above hi")
		})
	})

	t.Run("ByteRange stays inside both bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.ByteRange(40, 60).Draw(rt, "v")
			testkit.True(t, v >= 40 && v <= 60, "ByteRange must stay in [lo, hi]")
		})
	})

	t.Run("Byte is drawable", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) { _ = model.Byte().Draw(rt, "v") })
	})
}

func TestRapidRunes(t *testing.T) {
	t.Parallel()

	t.Run("Rune is drawable", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) { _ = model.Rune().Draw(rt, "v") })
	})

	t.Run("RuneFrom draws only from the supplied set", func(t *testing.T) {
		t.Parallel()
		allowed := []rune{'a', 'b', 'c'}
		model.Check(t, func(rt *model.T) {
			v := model.RuneFrom(allowed).Draw(rt, "v")
			testkit.True(t, v == 'a' || v == 'b' || v == 'c',
				"RuneFrom must draw only from the supplied runes")
		})
	})

	t.Run("RuneFrom honours unicode tables", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.RuneFrom(nil, unicode.Latin).Draw(rt, "v")
			testkit.True(t, unicode.Is(unicode.Latin, v),
				"RuneFrom must draw from the supplied table")
		})
	})
}

// Every signed and unsigned width shares one delegation shape, so they are
// checked uniformly: Min never draws below lo, Max never above hi, Range stays
// inside both. A width forwarding to the wrong rapid function fails here.
func TestRapidSignedIntegers(t *testing.T) {
	t.Parallel()

	t.Run("Int bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			testkit.True(t, model.IntMin(100).Draw(rt, "min") >= 100, "IntMin")
			testkit.True(t, model.IntMax(-100).Draw(rt, "max") <= -100, "IntMax")
			r := model.IntRange(-5, 5).Draw(rt, "rng")
			testkit.True(t, r >= -5 && r <= 5, "IntRange")
			_ = model.Int().Draw(rt, "any")
		})
	})

	t.Run("Int8 bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			testkit.True(t, model.Int8Min(100).Draw(rt, "min") >= 100, "Int8Min")
			testkit.True(t, model.Int8Max(-100).Draw(rt, "max") <= -100, "Int8Max")
			r := model.Int8Range(-5, 5).Draw(rt, "rng")
			testkit.True(t, r >= -5 && r <= 5, "Int8Range")
			_ = model.Int8().Draw(rt, "any")
		})
	})

	t.Run("Int16 bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			testkit.True(t, model.Int16Min(1000).Draw(rt, "min") >= 1000, "Int16Min")
			testkit.True(t, model.Int16Max(-1000).Draw(rt, "max") <= -1000, "Int16Max")
			r := model.Int16Range(-5, 5).Draw(rt, "rng")
			testkit.True(t, r >= -5 && r <= 5, "Int16Range")
			_ = model.Int16().Draw(rt, "any")
		})
	})

	t.Run("Int32 bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			testkit.True(t, model.Int32Min(1000).Draw(rt, "min") >= 1000, "Int32Min")
			testkit.True(t, model.Int32Max(-1000).Draw(rt, "max") <= -1000, "Int32Max")
			r := model.Int32Range(-5, 5).Draw(rt, "rng")
			testkit.True(t, r >= -5 && r <= 5, "Int32Range")
			_ = model.Int32().Draw(rt, "any")
		})
	})

	t.Run("Int64 bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			testkit.True(t, model.Int64Min(1000).Draw(rt, "min") >= 1000, "Int64Min")
			testkit.True(t, model.Int64Max(-1000).Draw(rt, "max") <= -1000, "Int64Max")
			r := model.Int64Range(-5, 5).Draw(rt, "rng")
			testkit.True(t, r >= -5 && r <= 5, "Int64Range")
			_ = model.Int64().Draw(rt, "any")
		})
	})
}

func TestRapidUnsignedIntegers(t *testing.T) {
	t.Parallel()

	t.Run("Uint bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			testkit.True(t, model.UintMin(1000).Draw(rt, "min") >= 1000, "UintMin")
			testkit.True(t, model.UintMax(10).Draw(rt, "max") <= 10, "UintMax")
			r := model.UintRange(3, 9).Draw(rt, "rng")
			testkit.True(t, r >= 3 && r <= 9, "UintRange")
			_ = model.Uint().Draw(rt, "any")
		})
	})

	t.Run("Uint8 bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			testkit.True(t, model.Uint8Min(200).Draw(rt, "min") >= 200, "Uint8Min")
			testkit.True(t, model.Uint8Max(10).Draw(rt, "max") <= 10, "Uint8Max")
			r := model.Uint8Range(3, 9).Draw(rt, "rng")
			testkit.True(t, r >= 3 && r <= 9, "Uint8Range")
			_ = model.Uint8().Draw(rt, "any")
		})
	})

	t.Run("Uint16 bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			testkit.True(t, model.Uint16Min(1000).Draw(rt, "min") >= 1000, "Uint16Min")
			testkit.True(t, model.Uint16Max(10).Draw(rt, "max") <= 10, "Uint16Max")
			r := model.Uint16Range(3, 9).Draw(rt, "rng")
			testkit.True(t, r >= 3 && r <= 9, "Uint16Range")
			_ = model.Uint16().Draw(rt, "any")
		})
	})

	t.Run("Uint32 bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			testkit.True(t, model.Uint32Min(1000).Draw(rt, "min") >= 1000, "Uint32Min")
			testkit.True(t, model.Uint32Max(10).Draw(rt, "max") <= 10, "Uint32Max")
			r := model.Uint32Range(3, 9).Draw(rt, "rng")
			testkit.True(t, r >= 3 && r <= 9, "Uint32Range")
			_ = model.Uint32().Draw(rt, "any")
		})
	})

	t.Run("Uint64 bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			testkit.True(t, model.Uint64Min(1000).Draw(rt, "min") >= 1000, "Uint64Min")
			testkit.True(t, model.Uint64Max(10).Draw(rt, "max") <= 10, "Uint64Max")
			r := model.Uint64Range(3, 9).Draw(rt, "rng")
			testkit.True(t, r >= 3 && r <= 9, "Uint64Range")
			_ = model.Uint64().Draw(rt, "any")
		})
	})

	t.Run("Uintptr bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			testkit.True(t, model.UintptrMin(1000).Draw(rt, "min") >= 1000, "UintptrMin")
			testkit.True(t, model.UintptrMax(10).Draw(rt, "max") <= 10, "UintptrMax")
			r := model.UintptrRange(3, 9).Draw(rt, "rng")
			testkit.True(t, r >= 3 && r <= 9, "UintptrRange")
			_ = model.Uintptr().Draw(rt, "any")
		})
	})
}

func TestRapidFloats(t *testing.T) {
	t.Parallel()

	t.Run("Float32 bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			testkit.True(t, model.Float32Min(100).Draw(rt, "min") >= 100, "Float32Min")
			testkit.True(t, model.Float32Max(-100).Draw(rt, "max") <= -100, "Float32Max")
			r := model.Float32Range(-1, 1).Draw(rt, "rng")
			testkit.True(t, r >= -1 && r <= 1, "Float32Range")
			_ = model.Float32().Draw(rt, "any")
		})
	})

	t.Run("Float64 bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			testkit.True(t, model.Float64Min(100).Draw(rt, "min") >= 100, "Float64Min")
			testkit.True(t, model.Float64Max(-100).Draw(rt, "max") <= -100, "Float64Max")
			r := model.Float64Range(-1, 1).Draw(rt, "rng")
			testkit.True(t, r >= -1 && r <= 1, "Float64Range")
			_ = model.Float64().Draw(rt, "any")
		})
	})
}

func TestRapidStrings(t *testing.T) {
	t.Parallel()

	t.Run("String is drawable", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) { _ = model.String().Draw(rt, "v") })
	})

	t.Run("StringN honours its length bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.StringN(2, 4, 8).Draw(rt, "v")
			n := len([]rune(v))
			testkit.True(t, n >= 2 && n <= 4, "StringN must respect min/max runes")
			testkit.True(t, len(v) <= 8, "StringN must respect maxLen")
		})
	})

	t.Run("StringOf draws from the element generator", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.StringOf(model.RuneFrom([]rune{'x'})).Draw(rt, "v")
			testkit.True(t, strings.Trim(v, "x") == "",
				"StringOf must use the supplied rune generator")
		})
	})

	t.Run("StringOfN combines element and length bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.StringOfN(model.RuneFrom([]rune{'y'}), 3, 3, 3).Draw(rt, "v")
			testkit.Equal(t, v, "yyy", "StringOfN must honour both element and length")
		})
	})

	t.Run("StringMatching satisfies the expression", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.StringMatching(`^a+$`).Draw(rt, "v")
			testkit.True(t, len(v) > 0 && strings.Trim(v, "a") == "",
				"StringMatching must satisfy the pattern")
		})
	})

	t.Run("SliceOfBytesMatching satisfies the expression", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.SliceOfBytesMatching(`^b+$`).Draw(rt, "v")
			testkit.True(t, len(v) > 0 && strings.Trim(string(v), "b") == "",
				"SliceOfBytesMatching must satisfy the pattern")
		})
	})
}

func TestRapidCombinators(t *testing.T) {
	t.Parallel()

	t.Run("SliceOf draws elements from its generator", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.SliceOf(model.Just(7)).Draw(rt, "v")
			for _, e := range v {
				testkit.Equal(t, e, 7, "SliceOf must use the element generator")
			}
		})
	})

	t.Run("SliceOfN honours its length bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.SliceOfN(model.Just(1), 2, 4).Draw(rt, "v")
			testkit.True(t, len(v) >= 2 && len(v) <= 4, "SliceOfN must respect [lo, hi]")
		})
	})

	t.Run("Just always yields the same value", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			testkit.Equal(t, model.Just("fixed").Draw(rt, "v"), "fixed", "Just")
		})
	})

	t.Run("OneOf draws from the supplied alternatives", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.OneOf(model.Just(1), model.Just(2)).Draw(rt, "v")
			testkit.True(t, v == 1 || v == 2, "OneOf must pick one of its generators")
		})
	})

	t.Run("Deferred resolves lazily", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			g := model.Deferred(func() *model.Generator[int] { return model.Just(9) })
			testkit.Equal(t, g.Draw(rt, "v"), 9, "Deferred must resolve to its generator")
		})
	})

	t.Run("Ptr without nil always dereferences", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			p := model.Ptr(model.Just(4), false).Draw(rt, "v")
			if p == nil {
				t.Fatal("Ptr with allowNil=false must not draw nil")
			}
			testkit.Equal(t, *p, 4, "Ptr must point at the element value")
		})
	})

	t.Run("Ptr with nil can draw nil", func(t *testing.T) {
		t.Parallel()
		sawNil := false
		model.Check(t, func(rt *model.T) {
			if model.Ptr(model.Just(4), true).Draw(rt, "v") == nil {
				sawNil = true
			}
		})
		testkit.True(t, sawNil, "Ptr with allowNil=true must be able to draw nil")
	})

	t.Run("Map applies the function", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.Map(model.Just(3), func(i int) string {
				return strings.Repeat("z", i)
			}).Draw(rt, "v")
			testkit.Equal(t, v, "zzz", "Map must apply fn to the drawn value")
		})
	})

	t.Run("Custom runs the supplied function", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			// rapid requires a Custom generator to consume from the
			// bitstream, so the function draws before deriving its result.
			v := model.Custom(func(ct *model.T) int {
				return model.IntRange(1, 3).Draw(ct, "inner") * 10
			}).Draw(rt, "v")
			testkit.True(t, v == 10 || v == 20 || v == 30,
				"Custom must yield the function's value")
		})
	})

	t.Run("Make derives a generator from the type", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) { _ = model.Make[int]().Draw(rt, "v") })
	})

	t.Run("MakeCustom accepts a config", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			_ = model.MakeCustom[int](model.MakeConfig{}).Draw(rt, "v")
		})
	})

	t.Run("ID returns its argument unchanged", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, model.ID(42), 42, "ID must be the identity function")
		testkit.Equal(t, model.ID("s"), "s", "ID must be generic over its argument")
	})
}

func TestRapidPropertyEntryPoints(t *testing.T) {
	t.Parallel()

	t.Run("MakeCheck produces a runnable test function", func(t *testing.T) {
		t.Parallel()
		ran := false
		fn := model.MakeCheck(func(*model.T) { ran = true })
		fn(t)
		testkit.True(t, ran, "MakeCheck must run the property")
	})

	t.Run("MakeFuzz produces a runnable fuzz target", func(t *testing.T) {
		t.Parallel()
		ran := false
		fn := model.MakeFuzz(func(*model.T) { ran = true })
		fn(t, []byte{0x01, 0x02, 0x03, 0x04})
		testkit.True(t, ran, "MakeFuzz must run the property")
	})

	t.Run("SyncTest runs the nested property", func(t *testing.T) {
		t.Parallel()
		ran := false
		model.Check(t, func(rt *model.T) {
			model.SyncTest(rt, func(*model.T) { ran = true })
		})
		testkit.True(t, ran, "SyncTest must run the nested property")
	})
}

func TestRapidDistinctAndMapGenerators(t *testing.T) {
	t.Parallel()

	t.Run("SliceOfDistinct yields distinct keys", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.SliceOfDistinct(
				model.IntRange(0, 20), model.ID[int],
			).Draw(rt, "v")
			seen := map[int]bool{}
			for _, e := range v {
				testkit.False(t, seen[e], "SliceOfDistinct must not repeat a key")
				seen[e] = true
			}
		})
	})

	t.Run("SliceOfNDistinct honours its length bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.SliceOfNDistinct(
				model.IntRange(0, 20), 2, 4, model.ID[int],
			).Draw(rt, "v")
			testkit.True(t, len(v) >= 2 && len(v) <= 4,
				"SliceOfNDistinct must respect [lo, hi]")
		})
	})

	t.Run("Permutation preserves the multiset", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.Permutation([]int{1, 2, 3, 4}).Draw(rt, "v")
			testkit.Equal(t, len(v), 4, "Permutation must preserve length")
			sum := 0
			for _, e := range v {
				sum += e
			}
			testkit.Equal(t, sum, 10, "Permutation must preserve the elements")
		})
	})

	t.Run("SampledFrom draws only supplied values", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			v := model.SampledFrom([]string{"a", "b"}).Draw(rt, "v")
			testkit.True(t, v == "a" || v == "b",
				"SampledFrom must draw from the supplied slice")
		})
	})

	t.Run("MapOf uses both generators", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			m := model.MapOf(model.IntRange(0, 5), model.Just("v")).Draw(rt, "m")
			for k, val := range m {
				testkit.True(t, k >= 0 && k <= 5, "MapOf must use the key generator")
				testkit.Equal(t, val, "v", "MapOf must use the value generator")
			}
		})
	})

	t.Run("MapOfN honours its length bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			m := model.MapOfN(model.IntRange(0, 50), model.Just(1), 2, 4).Draw(rt, "m")
			testkit.True(t, len(m) >= 2 && len(m) <= 4,
				"MapOfN must respect [lo, hi]")
		})
	})

	t.Run("MapOfValues keys by the supplied function", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			m := model.MapOfValues(
				model.IntRange(0, 20), func(v int) string { return string(rune('a' + v%26)) },
			).Draw(rt, "m")
			for k, v := range m {
				testkit.Equal(t, k, string(rune('a'+v%26)),
					"MapOfValues must key by the supplied function")
			}
		})
	})

	t.Run("MapOfNValues honours its length bounds", func(t *testing.T) {
		t.Parallel()
		model.Check(t, func(rt *model.T) {
			m := model.MapOfNValues(
				model.IntRange(0, 50), 2, 4, model.ID[int],
			).Draw(rt, "m")
			testkit.True(t, len(m) >= 2 && len(m) <= 4,
				"MapOfNValues must respect [lo, hi]")
		})
	})
}

// counterMachine is a minimal rapid.StateMachine: a Check invariant plus two
// exported action methods, which is what StateMachineActions reflects over.
type counterMachine struct{ n int }

func (m *counterMachine) Inc(*model.T) { m.n++ }
func (m *counterMachine) Dec(*model.T) { m.n-- }
func (*counterMachine) Check(*model.T) {}

func TestRapidStateMachineActions(t *testing.T) {
	t.Parallel()

	t.Run("reflects every exported action but not Check", func(t *testing.T) {
		t.Parallel()
		actions := model.StateMachineActions(&counterMachine{})

		for _, name := range []string{"Inc", "Dec"} {
			if _, ok := actions[name]; !ok {
				t.Fatalf("%s must be reflected as an action, got %v", name, actions)
			}
		}
		if _, ok := actions["Check"]; ok {
			t.Fatal("Check must not be reflected under its own name")
		}
		// rapid files the invariant under the empty key, which is what
		// T.Repeat looks for — so the map holds two actions plus the check.
		if _, ok := actions[""]; !ok {
			t.Fatalf("the Check invariant must be filed under \"\", got %v", actions)
		}
		testkit.Equal(t, len(actions), 3, "two actions plus the check hook")
	})

	t.Run("the returned actions drive the machine", func(t *testing.T) {
		t.Parallel()
		m := &counterMachine{}
		actions := model.StateMachineActions(m)
		model.Check(t, func(rt *model.T) {
			actions["Inc"](rt)
		})
		testkit.True(t, m.n > 0, "invoking the reflected action must mutate state")
	})
}
