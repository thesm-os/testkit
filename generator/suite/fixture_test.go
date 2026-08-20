// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/compositewriter"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/multiargwriter"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/mutator"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/writer"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/causal"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/gentest"
	"go.thesmos.sh/testkit/generator/suite"
)

// An integer parameter on a Go interface is usually a position or a size, and
// eidos's 42 is an index panic against anything a fixture seeds.
func TestIntegerSample(t *testing.T) {
	t.Parallel()

	t.Run("draws a value any collection admits", func(t *testing.T) {
		t.Parallel()
		f := fieldOf(t, contractIn(t, indexed(t)).Fixture, "I")
		testkit.Equal(t, f.Sample.Text, "1", "small enough to index what the harness seeded")
		testkit.Equal(t, f.Other.Text, "2", "and the alternate with it")
	})

	t.Run("keeps the pair discriminating", func(t *testing.T) {
		t.Parallel()
		// Zero would read as "the subject dropped the field", which is the
		// vacuity a sample exists to rule out.
		f := fieldOf(t, contractIn(t, indexed(t)).Fixture, "I")
		testkit.False(t, f.Sample.Text == "0" || f.Other.Text == "0",
			"neither half is the zero value")
		testkit.False(t, f.Sample.Text == f.Other.Text, "and the two differ")
	})

	t.Run("narrows an integer field of a struct too", func(t *testing.T) {
		t.Parallel()
		// One policy across both derivations: a field and the parameter
		// carrying it must not disagree about what an int is.
		f := fieldOf(t, contractIn(t, indexed(t)).Fixture, "Page")
		testkit.Equal(t, partNamed(t, f, "Offset").Sample.Text, "1",
			"the composed part draws the same pair")
	})
}

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
		f := fieldOf(t, contractIn(t, mixed(t)).Fixture, "Payload")
		testkit.Equal(t, partNames(f), []string{"Key", "Body"},
			"every exported field is set, not only the first eidos would take")
	})

	t.Run("makes the second value differ in every field", func(t *testing.T) {
		t.Parallel()
		// Two values differing in one field are indistinguishable to a subject
		// keyed on another.
		f := fieldOf(t, contractIn(t, mixed(t)).Fixture, "Payload")
		for _, p := range f.Parts {
			testkit.False(t, p.Sample.Text == p.Other.Text,
				"the two values differ in "+p.Name)
		}
	})

	t.Run("skips a field no literal can be written for", func(t *testing.T) {
		t.Parallel()
		// The fields around it still discriminate, and refusing here would drop
		// every check the parameter feeds.
		f := fieldOf(t, contractIn(t, partlyDerivable(t)).Fixture, "Params")
		testkit.True(t, f.OK(), "a struct with one underivable field still yields a value")
		testkit.Equal(t, partNames(f), []string{"Name", "Hook"},
			"the settable fields are set and the unresolvable one is left at its zero")
	})

	t.Run("writes a func field rather than skipping it", func(t *testing.T) {
		t.Parallel()
		// A func has no LITERAL and does have a value: the no-op closure, which
		// is the one thing a caller can pass that asserts nothing. A struct
		// carrying one is composable, so the checks its parameter feeds survive.
		f := fieldOf(t, contractIn(t, partlyDerivable(t)).Fixture, "Params")
		for _, p := range f.Parts {
			if p.Name == "Hook" {
				testkit.True(t, p.Sample.OK(), "the func field carries a value")
				return
			}
		}
		t.Fatal("the func field is absent from the composed parts")
	})

	t.Run("keeps the reference a nested struct field needs", func(t *testing.T) {
		t.Parallel()
		// Go forbids type elision in a struct field's value, so `{Inner: {F:
		// "x"}}` is not a composite literal — it is a compile error. Only the
		// backend knows how to spell `Inner` for the file being written and to
		// register the import it needs, which is why a part carries a
		// [golang.Sample] rather than the text of one.
		f := fieldOf(t, contractIn(t, nestedStruct(t)).Fixture, "Outer")
		inner := partNamed(t, f, "Inner")
		testkit.True(
			t,
			inner.Sample.Ref != nil,
			"the nested type is carried, not flattened to text",
		)
		testkit.True(
			t,
			inner.Sample.Composite,
			"and recorded as a composite rather than a conversion",
		)
	})
}

// The one place a team writes down "here is a valid instance of this type",
// which a generator cannot derive for a type with real validation.
func TestCompanion(t *testing.T) {
	t.Parallel()

	t.Run("wins over the derived sample", func(t *testing.T) {
		t.Parallel()
		f := fieldOf(
			t,
			contractIn(t, withCompanion(t, "PayloadDefaults", 0, "Payload")).Fixture,
			"Payload",
		)
		testkit.True(t, f.Companion != nil, "the companion is used where the source declares one")
	})

	t.Run("is refused when it takes arguments", func(t *testing.T) {
		t.Parallel()
		// A PayloadDefaults taking arguments is a different function that
		// happens to collide, and calling it would emit a fixture that does not
		// compile.
		f := fieldOf(
			t,
			contractIn(t, withCompanion(t, "PayloadDefaults", 1, "Payload")).Fixture,
			"Payload",
		)
		testkit.True(t, f.Companion == nil, "the signature is checked, not only the name")
	})

	t.Run("is refused when it returns something else", func(t *testing.T) {
		t.Parallel()
		f := fieldOf(t, contractIn(t, withCompanion(t, "PayloadDefaults", 0, "Other")).Fixture, "Payload")
		testkit.True(t, f.Companion == nil, "a companion must return the type it defaults")
	})

	t.Run("is not found under another name", func(t *testing.T) {
		t.Parallel()
		f := fieldOf(t, contractIn(t, withCompanion(t, "NewPayload", 0, "Payload")).Fixture, "Payload")
		testkit.True(t, f.Companion == nil, "the convention is the name, not the shape alone")
	})

	t.Run("leaves the alternate derived", func(t *testing.T) {
		t.Parallel()
		// One function cannot answer both questions. A companion says "a value
		// this type accepts"; the alternate says "a value that should not be
		// found", which is what a miss check needs.
		f := fieldOf(
			t,
			contractIn(t, withCompanion(t, "PayloadDefaults", 0, "Payload")).Fixture,
			"Payload",
		)
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
		testkit.Equal(t, contractIn(t, seeded(t)).Seed.Args, []string{"Payload"},
			"the seed writes the fixture's own sample")
	})

	t.Run("writes through a writer of any arity", func(t *testing.T) {
		t.Parallel()
		// The three writer detectors differ only in how many non-context
		// arguments they take, and the seed passes whatever the method declares
		// — so arity is not something it has to know. Keying on `writer` alone
		// left `Put(ctx, key, v)` unable to seed itself, which is the ordinary
		// keyed store.
		for _, name := range []string{compositewriter.Name, multiargwriter.Name} {
			got := contractIn(t, keyedWriter(t, name)).Seed
			testkit.True(t, got != nil, name+" seeds the subject")
			testkit.Equal(t, got.Args, []string{"Key", "Payload"},
				"and is handed every argument it declares")
		}
	})

	t.Run("declines a writer that cannot report its own failure", func(t *testing.T) {
		t.Parallel()
		// A mutator writes and returns nothing, so a seed through one fails
		// silently — and every check after it then runs against an empty
		// subject, passing and asserting nothing.
		testkit.True(t, contractIn(t, voidWriter(t)).Seed == nil,
			"a seed with no error return is no seed")
	})

	t.Run("is absent where no method is classified writer", func(t *testing.T) {
		t.Parallel()
		// A read-only interface over external state, which is the case a
		// consumer's own seed exists for.
		testkit.True(t, contractIn(t, mixed(t)).Seed == nil,
			"an unstamped interface derives no seed")
	})
}

// A fixture field is a parameter name *at a type*. Two methods naming one
// parameter alike is ordinary Go, and the interface that does it is legitimate
// source rather than something to complain about.
func TestFixtureKeying(t *testing.T) {
	t.Parallel()

	t.Run("takes one field for a name two methods share at one type", func(t *testing.T) {
		t.Parallel()
		// A `key string` on the reader and one on the deleter are the same
		// value as far as a conformance run is concerned; separate fields would
		// let a consumer override one and silently not the other.
		testkit.Len(t, contractIn(t, sharedKey(t)).Fixture.Fields, 1,
			"one field serves both methods")
	})

	t.Run("holds both where one name carries two types", func(t *testing.T) {
		t.Parallel()
		// Keyed on the name alone, the fixture held one of them and handed it
		// to the method taking the other, which does not compile. The earlier
		// answer was a diagnostic telling the author to rename a parameter,
		// which is bad advice about correct source.
		testkit.Len(t, contractIn(t, collidingFixture(t)).Fixture.Fields, 2,
			"two types under one name are two fields")
	})

	t.Run("reports nothing about it", func(t *testing.T) {
		t.Parallel()
		got := gentest.About(gentest.Diagnostics(t, suite.New(), collidingFixture(t)),
			"fixture")
		testkit.Len(t, got, 0, "there is nothing wrong with the source")
	})

	t.Run("qualifies every contested spelling, not just the later one", func(t *testing.T) {
		t.Parallel()
		// Both, so neither is privileged by the order the walk happened to
		// take — and by the method rather than the type, because a composite
		// has no name to spell and `KeySlice` would be one this package
		// invented.
		f := contractIn(t, collidingFixture(t)).Fixture
		for _, want := range []string{"GetKey", "PutKey"} {
			_, ok := f.Field(want)
			testkit.True(t, ok, "the fixture declares "+want)
		}
	})
}

// A variadic method takes many and its checks pass one, which is a narrowing
// the generated file has to say out loud.
//
// Nothing about the derivation is wrong — a fixture holds one value per
// parameter and `...T`'s element type is T — but a reader meeting
// `AssertFinderFindSmoke(tb, subject, keys)` has no way to see that `Find` was
// declared variadic, and would take the one-element call for the whole claim.
func TestVariadicIsAnnounced(t *testing.T) {
	t.Parallel()

	t.Run("marks the field derived from a variadic parameter", func(t *testing.T) {
		t.Parallel()
		f := fieldOf(t, contractIn(t, variadicFixture(t)).Fixture, "Keys")
		testkit.True(t, f.Variadic, "the field holds one element of a list")
	})

	t.Run("leaves an ordinary parameter unmarked", func(t *testing.T) {
		t.Parallel()
		// Or the note would appear on every field and mean nothing.
		f := fieldOf(t, contractIn(t, variadicFixture(t)).Fixture, "Limit")
		testkit.False(t, f.Variadic, "a fixed parameter is not narrowed")
	})

	t.Run("names the parameter on the method", func(t *testing.T) {
		t.Parallel()
		// Go permits one variadic parameter, in final position, so one answer
		// covers the signature.
		m := methodNamed(t, contractIn(t, variadicFixture(t)), "Find")
		got := m.VariadicParam()
		testkit.True(t, got != nil, "Find declares one")
		testkit.Equal(t, got.Name, "keys", "and it is the one the source named")
	})

	t.Run("answers nil for a method with none", func(t *testing.T) {
		t.Parallel()
		m := methodNamed(t, contractIn(t, mixed(t)), "Read")
		testkit.True(t, m.VariadicParam() == nil, "Read takes a fixed list")
	})
}

// A shape that reports absence in a value rather than an error owes a check the
// signature cannot derive.
//
// An error return says on its own that a call can fail. A trailing bool, or a
// bare value, says nothing without knowing the method answers a question about
// presence — which is what the shape stamp supplies (docs/adr/0018).
// A func admits no literal under any run. A type in a package the patterns did
// not reach admits one perfectly well — the run simply did not look. Reporting
// the second as settled sends an author to change source that is already
// correct, so the diagnostic reads the refusal rather than assuming.
func TestUndeliverableReason(t *testing.T) {
	t.Parallel()

	t.Run("calls a func a type with no literal", func(t *testing.T) {
		t.Parallel()
		f := fieldOf(t, contractIn(t, funcParamFixture(t)).Fixture, "Fn")
		testkit.Equal(t, f.Reason(), "which no literal can be written for",
			"nothing about a wider run changes this answer")
	})

	t.Run("calls an unloaded type a gap in this run", func(t *testing.T) {
		t.Parallel()
		f := fieldOf(t, contractIn(t, unloadedParamFixture(t)).Fixture, "Thing")
		testkit.Equal(
			t,
			f.Reason(),
			"which this run did not resolve, so no value was derived for it",
			"a run reaching the declaring package would derive one",
		)
	})

	// The diagnostic arm moved with the check assembly: an
	// undeliverable draw now reaches the author as a deriver refusal,
	// asserted where the derivers are tested.
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

// partNames returns the fields a composed value sets, in order.
func partNames(f suite.FixtureField) []string {
	out := make([]string, 0, len(f.Parts))
	for _, p := range f.Parts {
		out = append(out, p.Name)
	}
	return out
}

// methodNamed returns the contract's method of that name, failing when absent.
func methodNamed(t *testing.T, c *suite.Contract, name string) suite.Method {
	t.Helper()
	for _, m := range c.Methods {
		if m.Name == name {
			return m
		}
	}
	t.Fatalf("the contract carries no method %q", name)
	return suite.Method{}
}

// partNamed returns the composed part of that name, failing when absent.
func partNamed(t *testing.T, f suite.FixtureField, name string) suite.FixturePart {
	t.Helper()
	for _, p := range f.Parts {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("the value composes no part %q; it sets %v", name, partNames(f))
	return suite.FixturePart{}
}

// nestedStruct declares a struct one of whose fields is itself a struct, which
// is the shape a value carried as text cannot spell.
func nestedStruct(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("nest", "example.com/nest").
		Struct("Leaf", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.AtFile("nest/iface.go"))
			b.Field("F", storefixture.Named("string"), nil)
		}).
		Struct("Outer", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.AtFile("nest/iface.go"))
			b.Field("Name", storefixture.Named("string"), nil)
			b.Field("Inner", storefixture.PkgNamed("example.com/nest", "Leaf"), nil)
		}).
		Interface("Nested", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("nest/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("o", storefixture.PkgNamed("example.com/nest", "Outer"))
				gentest.Err(m)
			})
		}).
		Build()
}

// causalEntry declares a log whose element names its causes, under the mixin
// that makes naming them a precondition — the shape whose derived seed the
// subject correctly refuses.
func causalEntry(t *testing.T, carriesMixin bool) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("log", "example.com/log").
		Struct("Entry", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.AtFile("log/iface.go"))
			b.Field("ID", storefixture.Named("string"), nil)
			b.Field("DependsOn", storefixture.Slice(storefixture.Named("string")), nil)
			b.Field("Body", storefixture.Named("string"), nil)
		}).
		Interface("Log", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("log/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Append", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("e", storefixture.PkgNamed("example.com/log", "Entry"))
				gentest.Err(m)
			})
		}).
		Build()
	if carriesMixin {
		forMethod(s, "Append", func(bag *sdk.Bag) {
			shape.MetaMixins.Set(bag, []string{causal.Name}, "test")
		})
	}
	return s
}

// An admission precondition is a claim about the subject's history, and a
// derived value cannot satisfy one: the causes it names have never landed.
func TestAdmissionConstrainedFields(t *testing.T) {
	t.Parallel()

	t.Run("leaves the dependency collection at its zero", func(t *testing.T) {
		t.Parallel()
		// The whole seed is refused otherwise — the harness's first act fails,
		// and every check after it reports against a subject that was never
		// populated. The subject is right and the fixture is wrong.
		f := fieldOf(t, contractIn(t, causalEntry(t, true)).Fixture, "Entry")
		testkit.Equal(t, partNames(f), []string{"ID", "Body"},
			"the scalar fields still discriminate; the cause list is dropped")
	})

	t.Run("keeps it where nothing claims a precondition", func(t *testing.T) {
		t.Parallel()
		// The same struct without the claim is an ordinary value, and dropping
		// a field there would lose discrimination for nothing.
		f := fieldOf(t, contractIn(t, causalEntry(t, false)).Fixture, "Entry")
		testkit.Equal(t, partNames(f), []string{"ID", "DependsOn", "Body"},
			"every exported field is set when the value is self-contained")
	})
}

// indexed declares the shape the integer policy exists for: a comparator
// taking two bare-int positions, beside a defined type over an integer and a
// struct carrying one.
func indexed(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("idx", "example.com/idx").
		Struct("Page", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.AtFile("idx/iface.go"))
			b.Field("Name", storefixture.Named("string"), nil)
			b.Field("Offset", storefixture.Named("int"), nil)
		}).
		Interface("Sorter", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("idx/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Less", func(m *storefixture.MethodBuilder) {
				m.Param("i", storefixture.Named("int"))
				m.Param("j", storefixture.Named("int"))
				m.Return(storefixture.Named("bool"))
			})
			i.Method("Seek", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("p", storefixture.PkgNamed("example.com/idx", "Page"))
				gentest.Err(m)
			})
		}).
		Build()
}

// partlyDerivable declares a struct with one settable field and one that no
// literal can be written for.
func partlyDerivable(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("hooks", "example.com/hooks").
		Struct("Params", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.AtFile("hooks/iface.go"))
			b.Field("Name", storefixture.Named("string"), nil)
			b.Field("Hook", storefixture.Func(nil, nil), nil)
			// The underivable one. A func and a channel both sample now,
			// so the field that cannot be written has to be a type the
			// run never read and nothing curates.
			b.Field("Handle", storefixture.PkgNamed("example.com/other", "Handle"), nil)
		}).
		Interface("Runner", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("hooks/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Run", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("p", storefixture.PkgNamed("example.com/hooks", "Params"))
				gentest.Err(m)
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
			s.Pos(gentest.AtFile("cfg/iface.go"))
			s.Field("Key", storefixture.Named("string"), nil)
		}).
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("cfg/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Store", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("v", storefixture.PkgNamed("example.com/cfg", "Payload"))
				gentest.Err(m)
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

// keyedWriter is a store whose write takes a key beside its value, stamped with
// the given shape.
//
// `Put(ctx, key, v) error` is what compositewriter matches; the third argument
// pushes it to multiargwriter. The stamp is set by hand rather than detected,
// for the reason [seeded] gives: plugintest drives one plugin, so the shape
// annotator does not run.
func keyedWriter(t *testing.T, shapeName string) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("kv", "example.com/kv").
		Struct("Payload", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.At())
			b.Field("Body", storefixture.Named("string"), nil)
		}).
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.At())
			i.Directive(storefixture.Directive("suite"))
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Named("string"))
				m.Param("v", storefixture.PkgNamed("example.com/kv", "Payload"))
				gentest.Err(m)
			})
		}).
		Build()
	stamp(s, "Put", shapeName)
	return s
}

// voidWriter is a mutator: it writes and returns nothing, so it cannot report a
// failed seed.
func voidWriter(t *testing.T) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("mut", "example.com/mut").
		Interface("Mutator", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("mut/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Set", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("v", storefixture.Named("string"))
			})
		}).
		Build()
	stamp(s, "Set", mutator.Name)
	return s
}

// stamp sets the shape meta the annotator would have written for one method.
func stamp(s *sdk.Store, method, shapeName string) {
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name == method {
				shape.MetaShape.Set(m.EnsureMeta(), shapeName, "test")
			}
		}
	}
}

// sharedKey declares one parameter name across two methods at one type.
func sharedKey(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("col", "example.com/col").
		Interface("Col", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("col/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Named("string"))
				gentest.Err(m)
			})
			i.Method("Delete", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Named("string"))
				gentest.Err(m)
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
			b.Pos(gentest.AtFile("hooks/iface.go"))
			// Both a func and a channel sample now, so neither makes a
			// struct opaque. What still does is a type the run never read
			// and no curated table answers for.
			b.Field("Handle", storefixture.PkgNamed("example.com/other", "Handle"), nil)
			b.Field("Token", storefixture.PkgNamed("example.com/other", "Token"), nil)
		}).
		Interface("Runner", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("hooks/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Run", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("p", storefixture.PkgNamed("example.com/hooks", "Params"))
				gentest.Err(m)
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

// FieldFor is how a check names the field its argument lands in, and it has to
// be total: a caller composing an argument list should not have to prove the
// fixture holds every parameter first.
func TestFieldFor(t *testing.T) {
	t.Parallel()

	t.Run("names the field a parameter was grouped into", func(t *testing.T) {
		t.Parallel()
		f := contractIn(t, collidingFixture(t)).Fixture
		p := paramOf(t, contractIn(t, collidingFixture(t)), "Get", "key")
		testkit.Equal(t, f.FieldFor(p), "GetKey",
			"a contested name resolves to the qualified field")
	})

	t.Run("falls back to the parameter's own name", func(t *testing.T) {
		t.Parallel()
		// Unreached from the generator, which only ever asks about parameters
		// of the methods the fixture was built from. What the fallback buys is
		// that a wrong answer is a name a reader can find rather than the empty
		// string, which would compose `cfg.Fixture.` into generated source.
		f := contractIn(t, mixed(t)).Fixture
		testkit.Equal(t, f.FieldFor(golang.Param{Field: "Absent"}), "Absent",
			"a parameter the fixture never saw answers as itself")
	})
}

// paramOf returns the named parameter of the named method.
func paramOf(t *testing.T, c *suite.Contract, method, param string) golang.Param {
	t.Helper()
	for _, m := range c.Methods {
		if m.Name != method {
			continue
		}
		for _, p := range m.Params {
			if p.Name == param {
				return p
			}
		}
	}
	t.Fatalf("no parameter %q on %s", param, method)
	return golang.Param{}
}

// contractIn runs the plugin over the store and returns the queued
// projection carrier — the harness the fixture derivation feeds.
func contractIn(t *testing.T, s *sdk.Store) *suite.Contract {
	t.Helper()
	gentest.Diagnostics(t, suite.New(), s)
	for _, p := range s.Emit().PendingOriginSlots() {
		if c, ok := p.Item.(*suite.Contract); ok {
			return c
		}
	}
	t.Fatal("the run queued no contract")
	return nil
}

// mixed is the corpus fixture in store form: a writer carrying a mixin, the
// validator it names, and a reader.
//
// The same three methods conformance/corpus/iface/mixin/validates declares, so
// what this asserts about the projection is what the corpus compiles.
func mixed(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("validates", "example.com/validates").
		Struct("Payload", func(b *storefixture.StructBuilder) {
			b.Pos(gentest.AtFile("validates/iface.go"))
			b.Field("Key", storefixture.Named("string"), nil)
			b.Field("Body", storefixture.Named("string"), nil)
		}).
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("validates/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Store", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("v", storefixture.PkgNamed("example.com/validates", "Payload"))
				gentest.Err(m)
			})
			i.Method("Validate", func(m *storefixture.MethodBuilder) {
				m.Param("v", storefixture.PkgNamed("example.com/validates", "Payload"))
				gentest.Err(m)
			})
			i.Method("Read", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.PkgNamed("example.com/validates", "Payload"))
				gentest.Err(m)
			})
		}).
		Build()
}

// collidingFixture names one parameter identically across two composite types.
func collidingFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("col", "example.com/col").
		Interface("Col", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("col/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Slice(storefixture.Named("byte")))
				gentest.Err(m)
			})
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("key", storefixture.Slice(storefixture.Named("string")))
				gentest.Err(m)
			})
		}).
		Build()
}

// An interface is opted in by the directive, and a package generally holds more
// than the ones a harness was asked for.
func TestUndirectedInterface(t *testing.T) {
	t.Parallel()

	t.Run("generates nothing for it", func(t *testing.T) {
		t.Parallel()
		s := undirected(t)
		gentest.Diagnostics(t, suite.New(), s)
		testkit.Len(t, s.Emit().PendingOriginSlots(), 0,
			"a harness is generated where one is declared")
	})
}

// A directive on an interface declaring nothing asks for a harness that would
// assert nothing at all.
func TestEmptyInterface(t *testing.T) {
	t.Parallel()

	t.Run("reports it", func(t *testing.T) {
		t.Parallel()
		got := gentest.Diagnostics(t, suite.New(), emptyIface(t))
		testkit.Len(t, got, 1, "an interface with no method is reported once")
		testkit.Contains(t, got[0].Message, "declares no method", "and named for what is wrong")
	})
}

// Swallowing a failed append reads downstream as an interface nobody annotated
// rather than as a fault, and the harness is this generator's whole output.

// funcParamFixture takes a func beside a value return: a type no literal can be
// written for, on a method that would otherwise owe a miss check.
func funcParamFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("col", "example.com/col").
		Interface("Col", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("col/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("fn", storefixture.Func(nil, nil))
				m.Return(storefixture.Named("string"))
				gentest.Err(m)
			})
		}).
		Build()
}

// variadicFixture declares `...T` beside a fixed parameter, so the narrowing is
// distinguishable from a note the generator puts on everything.
func variadicFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("col", "example.com/col").
		Interface("Finder", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("col/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Find", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("limit", storefixture.Named("int"))
				m.Variadic("keys", storefixture.Named("string"))
				m.Return(storefixture.Slice(storefixture.Named("string")))
				gentest.Err(m)
			})
		}).
		Build()
}

// unloadedParamFixture takes a named type from a package the store never
// declares, which is what a narrow `run` pattern produces in real use.
func unloadedParamFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("col", "example.com/col").
		Interface("Col", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("col/iface.go"))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				gentest.Ctx(m)
				m.Param("t", storefixture.PkgNamed("example.com/elsewhere", "Thing"))
				m.Return(storefixture.Named("string"))
				gentest.Err(m)
			})
		}).
		Build()
}

// forMethod applies fn to the meta bag of every method of that name.
func forMethod(s *sdk.Store, method string, fn func(*sdk.Bag)) {
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name == method {
				fn(m.EnsureMeta())
			}
		}
	}
}

// undirected declares an interface carrying no directive.
func undirected(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Interface("Internal", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("cfg/iface.go"))
			i.Method("Ping", func(m *storefixture.MethodBuilder) {
				gentest.Err(m)
			})
		}).
		Build()
}

// emptyIface carries the directive and declares nothing for it to cover.
func emptyIface(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("cfg", "example.com/cfg").
		Interface("Empty", func(i *storefixture.InterfaceBuilder) {
			i.Pos(gentest.AtFile("cfg/iface.go"))
			i.Directive(storefixture.Directive("suite"))
		}).
		Build()
}
