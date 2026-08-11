// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/writer"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/suite"
)

// A sample exists to be distinguishable, and how much of a struct it sets is
// the difference between a suite that catches a dropped field and one that
// does not.
func TestStructSample(t *testing.T) {
	t.Parallel()

	t.Run("sets every exported field", func(t *testing.T) {
		t.Parallel()
		// eidos sets the first settable field, for readability, which is right
		// where the value is handed to a setter and read straight back. An
		// implementation that silently drops a field passes every check built
		// from a sample that never set it.
		f := fieldOf(t, contractIn(t, mixed(t)).Fixture, "V")
		testkit.Contains(t, f.Sample.Text, "Key:", "the sample sets Key")
		testkit.Contains(t, f.Sample.Text, "Body:", "and Body, which eidos would have left")
	})

	t.Run("makes the second value differ in every field", func(t *testing.T) {
		t.Parallel()
		// Two values differing in one field are indistinguishable to a subject
		// keyed on another.
		f := fieldOf(t, contractIn(t, mixed(t)).Fixture, "V")
		testkit.Contains(t, f.Other.Text, "other-key", "the alternate differs in Key")
		testkit.Contains(t, f.Other.Text, "other-body", "and in Body")
	})

	t.Run("skips a field no literal can be written for", func(t *testing.T) {
		t.Parallel()
		// The fields around it still discriminate, and refusing here would drop
		// every check the parameter feeds.
		f := fieldOf(t, contractIn(t, partlyDerivable(t)).Fixture, "P")
		testkit.True(t, f.OK(), "a struct with one underivable field still yields a sample")
		testkit.Contains(t, f.Sample.Text, "Name:", "the settable field is set")
		testkit.False(t, containsText(f.Sample.Text, "Hook"), "the func field is not")
	})
}

// The one place a team writes down "here is a valid instance of this type",
// which a generator cannot derive for a type with real validation.
func TestCompanion(t *testing.T) {
	t.Parallel()

	t.Run("wins over the derived sample", func(t *testing.T) {
		t.Parallel()
		f := fieldOf(t, contractIn(t, withCompanion(t, "PayloadDefaults", 0, "Payload")).Fixture, "V")
		testkit.True(t, f.Companion != nil, "the companion is used where the source declares one")
	})

	t.Run("is refused when it takes arguments", func(t *testing.T) {
		t.Parallel()
		// A PayloadDefaults taking arguments is a different function that
		// happens to collide, and calling it would emit a fixture that does not
		// compile.
		f := fieldOf(t, contractIn(t, withCompanion(t, "PayloadDefaults", 1, "Payload")).Fixture, "V")
		testkit.True(t, f.Companion == nil, "the signature is checked, not only the name")
	})

	t.Run("is refused when it returns something else", func(t *testing.T) {
		t.Parallel()
		f := fieldOf(t, contractIn(t, withCompanion(t, "PayloadDefaults", 0, "Other")).Fixture, "V")
		testkit.True(t, f.Companion == nil, "a companion must return the type it defaults")
	})

	t.Run("is not found under another name", func(t *testing.T) {
		t.Parallel()
		f := fieldOf(t, contractIn(t, withCompanion(t, "NewPayload", 0, "Payload")).Fixture, "V")
		testkit.True(t, f.Companion == nil, "the convention is the name, not the shape alone")
	})

	t.Run("leaves the alternate derived", func(t *testing.T) {
		t.Parallel()
		// One function cannot answer both questions. A companion says "a value
		// this type accepts"; the alternate says "a value that should not be
		// found", which is what a miss check needs.
		f := fieldOf(t, contractIn(t, withCompanion(t, "PayloadDefaults", 0, "Payload")).Fixture, "V")
		testkit.True(t, f.Other.OK(), "the alternate is still derived")
	})
}

// A reader over an empty subject asserts nothing, so something has to write
// first — and for an interface carrying a writer, that something is itself.
func TestSeed(t *testing.T) {
	t.Parallel()

	t.Run("writes through the interface's own writer", func(t *testing.T) {
		t.Parallel()
		got := contractIn(t, seeded(t)).Seed
		testkit.True(t, got != nil, "an interface with a writer seeds itself")
		testkit.Equal(t, got.Method.Name, "Store", "through the method the annotator classified")
	})

	t.Run("hands it the derived value", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, contractIn(t, seeded(t)).Seed.Args, []string{"V"},
			"the seed writes the fixture's own sample")
	})

	t.Run("is absent where no method is classified writer", func(t *testing.T) {
		t.Parallel()
		// A read-only interface over external state, which is the case a
		// consumer's own seed exists for.
		testkit.True(t, contractIn(t, mixed(t)).Seed == nil,
			"an unstamped interface derives no seed")
	})
}

// Two parameters sharing a name and differing in type cannot both live in one
// fixture field, and the guard has to see composite types to say so.
func TestFixtureNameCollision(t *testing.T) {
	t.Parallel()

	t.Run("reports two composite types under one name", func(t *testing.T) {
		t.Parallel()
		// QName is empty for every composite ref, so a guard comparing QNames
		// found []byte and []string equal — blind to exactly the shapes it
		// exists for.
		got := plugintest.Generate(t, suite.New(), collidingFixture(t)).Diagnostics()
		testkit.Len(t, got, 1, "a name carrying two types is reported once")
		testkit.Contains(t, got[0].Message, "rename one parameter", "the message says what to do")
	})

	t.Run("takes one field for a name two methods share", func(t *testing.T) {
		t.Parallel()
		// A `key string` on the reader and one on the deleter are the same
		// value as far as a conformance run is concerned; separate fields would
		// let a consumer override one and silently not the other.
		testkit.Len(t, contractIn(t, sharedKey(t)).Fixture.Fields, 1,
			"one field serves both methods")
	})
}

// A parameter whose type admits no literal must not cost a method the checks
// that never look at it.
func TestUnderivableParameter(t *testing.T) {
	t.Parallel()

	t.Run("keeps the checks that do not read the value", func(t *testing.T) {
		t.Parallel()
		want := []string{
			"smoke",
			"reports a cancelled context",
			"reports an expired deadline",
			"tolerates a nil context",
		}
		for _, w := range want {
			testkit.True(t, hasCheckIn(t, callbackFixture(t), "Watch", w),
				"Watch must keep "+w+" despite its callback parameter")
		}
	})

	t.Run("drops only the check whose meaning is the value", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, hasCheckIn(t, sliceFixture(t), "Get", "an error carries the zero value"),
			"the miss check needs a value derivation could reach")
	})

	t.Run("still emits a fixture field for it", func(t *testing.T) {
		t.Parallel()
		// Declared and left at its zero, so a consumer can supply one and write
		// the check the generator declined to.
		f, ok := contractIn(t, callbackFixture(t)).Fixture.Field("Fn")
		testkit.True(t, ok, "the field is declared even where no value could be derived")
		testkit.False(t, f.OK(), "and reports that nothing was derived for it")
	})

	t.Run("names the companion field for every derived one", func(t *testing.T) {
		t.Parallel()
		f := fieldOf(t, contractIn(t, mixed(t)).Fixture, "Key")
		testkit.Equal(t, f.OtherName(), "KeyOther", "the alternate is named for its sample")
	})
}

// fieldOf returns the fixture's field of that name, failing when absent.
func fieldOf(t *testing.T, f suite.Fixture, name string) suite.FixtureField {
	t.Helper()
	got, ok := f.Field(name)
	if !ok {
		t.Fatalf("the fixture declares no field %q", name)
	}
	return got
}

// containsText reports whether the sample text mentions a field.
func containsText(text, want string) bool {
	for i := 0; i+len(want) <= len(text); i++ {
		if text[i:i+len(want)] == want {
			return true
		}
	}
	return false
}

// partlyDerivable declares a struct with one settable field and one that no
// literal can be written for.
func partlyDerivable(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("hooks", "example.com/hooks").
		Struct("Params", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("hooks/iface.go", 1, 1))
			b.Field("Name", storefixture.Named("string"), nil)
			b.Field("Hook", storefixture.Func(nil, nil), nil)
		}).
		Interface("Runner", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("hooks/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Run", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("p", storefixture.PkgNamed("example.com/hooks", "Params"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
}

// withCompanion declares a candidate companion with the given name, parameter
// count and return type, so the signature check can be exercised.
func withCompanion(t *testing.T, name string, params int, returns string) *sdk.Store {
	t.Helper()
	b := storefixture.New().
		Package("cfg", "example.com/cfg").
		Struct("Payload", func(s *storefixture.StructBuilder) {
			s.Pos(sdk.At("cfg/iface.go", 1, 1))
			s.Field("Key", storefixture.Named("string"), nil)
		}).
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("cfg/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Store", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("v", storefixture.PkgNamed("example.com/cfg", "Payload"))
				m.Return(storefixture.Named("error"))
			})
		})
	b.Function(name, func(f *storefixture.FunctionBuilder) {
		for i := range params {
			f.Param("arg"+string(rune('0'+i)), storefixture.Named("string"))
		}
		f.Return(storefixture.PkgNamed("example.com/cfg", returns))
	})
	return b.Build()
}

// seeded stamps the writer classification the seed is derived from.
//
// Stamped by hand because plugintest drives one plugin: the shape annotator
// does not run, so a fixture that needs its output has to carry it. What is
// under test is what this generator does with the stamp, not that eidos
// produces one.
func seeded(t *testing.T) *sdk.Store {
	t.Helper()
	s := mixed(t)
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name == "Store" {
				shape.MetaShape.Set(m.EnsureMeta(), writer.Name, "test")
			}
		}
	}
	return s
}

// sharedKey declares one parameter name across two methods at one type.
func sharedKey(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("col", "example.com/col").
		Interface("Col", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("col/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
			i.Method("Delete", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
}

// The fixture is looked up by name from templates and from the seed, and a
// missing name must answer rather than panic.
func TestFixtureLookup(t *testing.T) {
	t.Parallel()

	t.Run("reports a name it does not hold", func(t *testing.T) {
		t.Parallel()
		_, ok := contractIn(t, mixed(t)).Fixture.Field("Nonexistent")
		testkit.False(t, ok, "a missing field is reported, not invented")
	})
}

// A struct none of whose fields admit a literal yields no sample at all, rather
// than an empty composite that reads as a value.
func TestWhollyUnderivableStruct(t *testing.T) {
	t.Parallel()

	t.Run("yields no sample", func(t *testing.T) {
		t.Parallel()
		// `Params{}` and `Params{Name: "x"}` are different claims, and only the
		// second is a sample.
		f := fieldOf(t, contractIn(t, opaqueStruct(t)).Fixture, "P")
		testkit.False(t, f.OK(), "a struct with no settable field derives nothing")
	})

	t.Run("keeps the checks that never read it", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasCheckIn(t, opaqueStruct(t), "Run", "reports a cancelled context"),
			"cancellation never looked at the value")
	})
}

// A writer whose own inputs cannot be derived cannot seed anything.
func TestSeedNeedsADerivableValue(t *testing.T) {
	t.Parallel()

	t.Run("is absent where the writer takes what nothing can supply", func(t *testing.T) {
		t.Parallel()
		// Seeding with a zero would populate the subject with a value no check
		// can then look for, which is worse than not seeding: every read after
		// it misses, and the miss checks pass for the wrong reason.
		testkit.True(t, contractIn(t, opaqueWriter(t)).Seed == nil,
			"a seed needs a value it can write")
	})
}

// opaqueStruct declares a struct no field of which admits a literal.
func opaqueStruct(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("hooks", "example.com/hooks").
		Struct("Params", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("hooks/iface.go", 1, 1))
			b.Field("Hook", storefixture.Func(nil, nil), nil)
			b.Field("Done", storefixture.Chan(storefixture.Named("struct{}")), nil)
		}).
		Interface("Runner", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("hooks/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Run", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("p", storefixture.PkgNamed("example.com/hooks", "Params"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
}

// opaqueWriter stamps the writer classification on a method whose value cannot
// be derived.
func opaqueWriter(t *testing.T) *sdk.Store {
	t.Helper()
	s := opaqueStruct(t)
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			shape.MetaShape.Set(m.EnsureMeta(), writer.Name, "test")
		}
	}
	return s
}
