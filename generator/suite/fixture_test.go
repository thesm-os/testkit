// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/batchreader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/compositewriter"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/lookup"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/multiargwriter"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/mutator"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/pointerreader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/predicate"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/readernoerror"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/readerwithbool"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/writer"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/causal"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
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
		f := fieldOf(t, contractIn(t, indexed(t)).Fixture, "P")
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
		f := fieldOf(t, contractIn(t, mixed(t)).Fixture, "V")
		testkit.Equal(t, partNames(f), []string{"Key", "Body"},
			"every exported field is set, not only the first eidos would take")
	})

	t.Run("makes the second value differ in every field", func(t *testing.T) {
		t.Parallel()
		// Two values differing in one field are indistinguishable to a subject
		// keyed on another.
		f := fieldOf(t, contractIn(t, mixed(t)).Fixture, "V")
		for _, p := range f.Parts {
			testkit.False(t, p.Sample.Text == p.Other.Text,
				"the two values differ in "+p.Name)
		}
	})

	t.Run("skips a field no literal can be written for", func(t *testing.T) {
		t.Parallel()
		// The fields around it still discriminate, and refusing here would drop
		// every check the parameter feeds.
		f := fieldOf(t, contractIn(t, partlyDerivable(t)).Fixture, "P")
		testkit.True(t, f.OK(), "a struct with one underivable field still yields a value")
		testkit.Equal(t, partNames(f), []string{"Name"},
			"the settable field is set and the func field is left at its zero")
	})

	t.Run("keeps the reference a nested struct field needs", func(t *testing.T) {
		t.Parallel()
		// Go forbids type elision in a struct field's value, so `{Inner: {F:
		// "x"}}` is not a composite literal — it is a compile error. Only the
		// backend knows how to spell `Inner` for the file being written and to
		// register the import it needs, which is why a part carries a
		// [golang.Sample] rather than the text of one.
		f := fieldOf(t, contractIn(t, nestedStruct(t)).Fixture, "O")
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
			"V",
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
			"V",
		)
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
		f := fieldOf(
			t,
			contractIn(t, withCompanion(t, "PayloadDefaults", 0, "Payload")).Fixture,
			"V",
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
		testkit.Equal(t, contractIn(t, seeded(t)).Seed.Args, []string{"V"},
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
			testkit.Equal(t, got.Args, []string{"Key", "V"},
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
		got := plugintest.Generate(t, suite.New(), collidingFixture(t)).Diagnostics()
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
		// A func is the type nothing can write down: there is no literal for
		// it, and no wider run produces one.
		testkit.False(
			t,
			hasCheckIn(t, funcParamFixture(t), "Get", "an error carries the zero value"),
			"the miss check needs a value derivation could reach",
		)
	})

	t.Run("keeps it for a composite eidos can write down", func(t *testing.T) {
		t.Parallel()
		// A slice used to be underivable and now is not, so the drop is a
		// property of the element rather than of composites. Asserting the
		// positive is what keeps the guard from quietly widening back.
		testkit.True(t, hasCheckIn(t, sliceFixture(t), "Get", "an error carries the zero value"),
			"a []byte is a value the fixture can supply")
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
func TestMissChecks(t *testing.T) {
	t.Parallel()

	t.Run("holds every slot but the flag to its zero", func(t *testing.T) {
		t.Parallel()
		// The flag is the signal, not a result: asserting `false` is `false`
		// is a check that cannot fail.
		m := methodNamed(t, contractIn(t, missFixture(t, readerwithbool.Name)), "Load")
		testkit.Len(t, m.MissReturns(), 1, "the value slot is held, the flag is not")
		testkit.True(t, m.FlagReturn() != nil, "the trailing bool is the signal")
	})

	t.Run("takes every slot where nothing flags the miss", func(t *testing.T) {
		t.Parallel()
		// A pointer reader has neither an error nor a flag, so the zero — nil —
		// is the only signal, and every returned slot carries it.
		m := methodNamed(t, contractIn(t, missNoFlagFixture(t)), "Load")
		testkit.True(t, m.FlagReturn() == nil, "no trailing bool to exclude")
		testkit.Len(t, m.MissReturns(), 2, "so every value slot is held to its zero")
	})

	t.Run("emits the check for a shape that owns it", func(t *testing.T) {
		t.Parallel()
		for _, name := range []string{
			readernoerror.Name, readerwithbool.Name, lookup.Name, pointerreader.Name,
		} {
			testkit.True(t, hasCheckIn(t, missFixture(t, name), "Load", "reports a miss"),
				name+" reports absence in a value, so it owes the check")
		}
	})

	t.Run("emits nothing for a shape that does not", func(t *testing.T) {
		t.Parallel()
		// The identical signature under another classification. A
		// `Validate(v) (Report, bool)` returns a verdict rather than an answer
		// about presence, and holding its report to the zero asserts nothing.
		testkit.False(t, hasCheckIn(t, missFixture(t, predicate.Name), "Load", "reports a miss"),
			"the stamp decides, not the shape of the return list")
	})

	t.Run("needs an input to miss on", func(t *testing.T) {
		t.Parallel()
		// The same rule zero-on-error follows: the miss is reached by choosing
		// an input that is not there.
		testkit.False(t, hasCheckIn(t, missNoInputFixture(t), "Load", "reports a miss"),
			"a method taking nothing after its context has no miss to reach")
	})

	t.Run("emits nothing when the flag is the only result", func(t *testing.T) {
		t.Parallel()
		// `Load(ctx, key) bool` reports absence and returns nothing else, so
		// the check would assert that false is false.
		//
		// Reachable despite no detector matching this signature: a source
		// directive overrides a classification at an authority above anything
		// an annotator wrote, and the generator cannot tell the difference.
		s := missShapeFixture(t, storefixture.Named("bool"))
		testkit.False(t, hasCheckIn(t, s, "Load", "reports a miss"),
			"a lone flag is the signal, and there is nothing beside it to hold")
	})

	t.Run("emits nothing when there is no result at all", func(t *testing.T) {
		t.Parallel()
		// `Load(ctx, key) error` under a lookup stamp — an override again. No
		// value slot means no flag to find and nothing to compare.
		s := missShapeFixture(t, storefixture.Named("error"))
		testkit.False(t, hasCheckIn(t, s, "Load", "reports a miss"),
			"an error-only return carries no value a miss could zero")
	})
}

// A classification the annotator attached, read off the projection rather than
// off the source node — which is not in scope by the time a template renders.
func TestMixinChecks(t *testing.T) {
	t.Parallel()

	t.Run("emits nilsafe where the mixin is attached", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasCheckIn(t, mixinFixture(t, "nilsafe", ""), "Load", "nilsafe"),
			"a method carrying the mixin owes the check")
	})

	t.Run("emits nothing where it is not", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, hasCheckIn(t, mixinFixture(t, "", ""), "Load", "nilsafe"),
			"an unclassified method owes nothing")
	})

	t.Run("gates timeout on the duration rather than the mixin", func(t *testing.T) {
		t.Parallel()
		// "Within a budget" is not a statement until one is named, so a bare
		// `//testkit:mixin timeout` has nothing to assert.
		testkit.True(t, hasCheckIn(t, mixinFixture(t, "timeout", "5s"), "Load", "timeout"),
			"a declared duration is a budget to measure against")
		testkit.False(t, hasCheckIn(t, mixinFixture(t, "timeout", ""), "Load", "timeout"),
			"a bare timeout mixin names no budget")
	})

	t.Run("cuts a sibling param back to its local name", func(t *testing.T) {
		t.Parallel()
		// The resolver rewrites a sibling into a qualified name so it is
		// unambiguous across packages; a generated call site holds the subject
		// and cannot spell that form.
		m := methodNamed(
			t,
			contractIn(t, mixinFixture(t, "orderafter", "example.com/miss.Store.Prepare")),
			"Load",
		)
		testkit.Equal(t, m.MixinPartner("orderafter", "fn"), "Prepare",
			"the trailing identifier is what a call site can use")
	})
}

// A relational classification names a second callable, and the check calls it.
//
// Until eidos declared the sibling param there was no second callable to reach
// — the stamp held a bare name with no package and no owner, so a generator
// could confirm a relationship existed and do nothing about it — the mixin
// schema declares no parameter naming the partner.
func TestRelationalMixin(t *testing.T) {
	t.Parallel()

	t.Run("calls the partner the directive names", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasCheckIn(t, relationalFixture(t, "Observed"), "Touch", "sideeffect"),
			"a named partner is one the check can observe through")
	})

	t.Run("emits nothing where the mixin names none", func(t *testing.T) {
		t.Parallel()
		// The param is optional by design: a bare mixin is still a
		// classification, and a consumer may want only to record that an effect
		// exists. So its absence is a check not generated, not a fault.
		testkit.False(t, hasCheckIn(t, relationalFixture(t, ""), "Touch", "sideeffect"),
			"an unnamed partner is nothing to call")
	})

	t.Run("emits nothing where the partner is not in the method set", func(t *testing.T) {
		t.Parallel()
		// The resolver refuses a name it cannot see, so this is unreachable
		// through a real run — but a check composing a call to a method the
		// subject does not declare would not compile, and a render error is a
		// file that came out short.
		testkit.False(t, hasCheckIn(t, relationalFixture(t, "Absent"), "Touch", "sideeffect"),
			"a partner outside the interface is one the subject cannot be asked for")
	})
}

// A hooks check constructs the callback it registers, so the partner has to be
// a registration and not merely a method the directive was pointed at.
func TestHooksCheck(t *testing.T) {
	t.Parallel()

	t.Run("emits where the partner takes one callback", func(t *testing.T) {
		t.Parallel()
		// Params and returns both, so the literal the check builds has a
		// signature to spell rather than an empty one.
		cb := storefixture.Func(
			[]*sdk.TypeRef{storefixture.Named("string")},
			[]*sdk.TypeRef{storefixture.Named("error")},
		)
		testkit.True(t, hasCheckIn(t, hooksFixture(t, cb), "Fire", "hooks"),
			"a func-typed parameter is a callback the check can build")
	})

	t.Run("emits nothing where the partner takes no func", func(t *testing.T) {
		t.Parallel()
		// `OnEvent(name string)` is something else the directive was aimed at,
		// and a func literal passed to it would not compile.
		testkit.False(
			t,
			hasCheckIn(t, hooksFixture(t, storefixture.Named("string")), "Fire", "hooks"),
			"a registration takes a callback, not a name",
		)
	})

	t.Run("emits nothing where the partner takes several parameters", func(t *testing.T) {
		t.Parallel()
		// Which one is the callback is a guess, and the mixin says nothing.
		testkit.False(t, hasCheckIn(t, hooksTwoParamFixture(t), "Fire", "hooks"),
			"two parameters and no rule for which registers")
	})
}

// A sample check passes the builder's output straight to the method, so the two
// have to fit — checked rather than assumed, since a mismatch is a generated
// call the toolchain refuses and a render error is a file that came out short.
func TestSampleCheck(t *testing.T) {
	t.Parallel()

	t.Run("emits where the builder produces what the method takes", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasCheckIn(t, sampleFixture(t, "string", 1), "Process", "sample"),
			"one parameter fed by one produced value of the same type")
	})

	t.Run("emits nothing where the types disagree", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, hasCheckIn(t, sampleFixture(t, "int", 1), "Process", "sample"),
			"a builder producing an int cannot feed a string parameter")
	})

	t.Run("emits nothing where the builder produces several values", func(t *testing.T) {
		t.Parallel()
		// Which one feeds the parameter is a guess, and the mixin names the
		// builder without saying — the ambiguity partition needed an axis for.
		testkit.False(t, hasCheckIn(t, sampleFixture(t, "string", 2), "Process", "sample"),
			"two produced values and one parameter is a pairing nothing states")
	})
}

// Isolation is a claim about two writes not reaching each other, and every way
// of getting it slightly wrong produces a check that cannot fail.
//
// Two drafts of this shipped-in-progress before the axis existed: one varied
// every parameter, so the writes never collided on a key; one held the payload,
// so an implementation ignoring partitions clobbered the first write with an
// identical value. Both passed against a store with a single flat namespace,
// which is the one subject the check exists to reject.
func TestPartitionCheck(t *testing.T) {
	t.Parallel()

	t.Run("varies the axis and the payload, holds the key", func(t *testing.T) {
		t.Parallel()
		ck := checkNamed(t, contractIn(t, partitionFixture(t, "part")), "Put", "partition")
		testkit.Equal(t, ck.SecondCall, []string{"partOther", "key", "valueOther"},
			"only the axis and the payload differ between the two writes")
		testkit.Equal(t, ck.CompareAgainst, "value",
			"and the read is held up to the first write's payload")
	})

	t.Run("emits nothing where no axis is named", func(t *testing.T) {
		t.Parallel()
		// Without it the check has to guess which parameter isolates, and every
		// guess produces one that passes for a subject ignoring partitions.
		testkit.False(t, hasCheckIn(t, partitionFixture(t, ""), "Put", "partition"),
			"an unnamed axis is one no check should invent")
	})

	t.Run("emits nothing where the axis names no parameter", func(t *testing.T) {
		t.Parallel()
		// eidos validates this and reports it, so a run never reaches here —
		// but a check varying a parameter the method does not take would not
		// compile, and a render error is a file that came out short.
		testkit.False(t, hasCheckIn(t, partitionFixture(t, "absent"), "Put", "partition"),
			"an axis outside the parameter list is nothing to vary")
	})

	t.Run("emits nothing where the reader needs what the writer does not take", func(t *testing.T) {
		t.Parallel()
		// A generated check receives the writer's parameters and nothing else,
		// so a reader wanting more cannot be called from inside it.
		testkit.False(t, hasCheckIn(t, partitionWiderReaderFixture(t), "Put", "partition"),
			"a reader taking a parameter the writer does not is one the check cannot call")
	})

	t.Run("emits nothing where the axis has no second value", func(t *testing.T) {
		t.Parallel()
		// Two partitions need two partition values, and a func-typed axis
		// yields none — so there is no second write to make.
		testkit.False(t, hasCheckIn(t, partitionUnderivableAxisFixture(t), "Put", "partition"),
			"an axis nothing can be written for is one nothing can be varied along")
	})

	t.Run("emits nothing where every parameter identifies the slot", func(t *testing.T) {
		t.Parallel()
		// Writer and reader taking the same list leaves no payload, so the two
		// writes differ in where they land and in nothing else — there is
		// nothing for the read to be wrong about.
		testkit.False(t, hasCheckIn(t, payloadlessPartitionFixture(t), "Put", "partition"),
			"a write carrying no value has no isolation to demonstrate")
	})
}

// Two reasons a value is missing, and only one of them is the author's to fix.
//
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
		f := fieldOf(t, contractIn(t, unloadedParamFixture(t)).Fixture, "T")
		testkit.Equal(
			t,
			f.Reason(),
			"which this run did not resolve, so no value was derived for it",
			"a run reaching the declaring package would derive one",
		)
	})

	t.Run("says so in the diagnostic", func(t *testing.T) {
		t.Parallel()
		// The reason is only worth deriving if it reaches the author.
		got := plugintest.Generate(t, suite.New(), unloadedParamFixture(t)).Diagnostics()
		testkit.Len(t, got, 1, "the dropped check is reported once")
		testkit.Assert(t, got[0].Message).Contains("which this run did not resolve",
			"the diagnostic carries the refusal, not a fixed phrase")
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
			b.Pos(sdk.At("nest/iface.go", 1, 1))
			b.Field("F", storefixture.Named("string"), nil)
		}).
		Struct("Outer", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("nest/iface.go", 1, 1))
			b.Field("Name", storefixture.Named("string"), nil)
			b.Field("Inner", storefixture.PkgNamed("example.com/nest", "Leaf"), nil)
		}).
		Interface("Nested", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("nest/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("o", storefixture.PkgNamed("example.com/nest", "Outer"))
				m.Return(storefixture.Named("error"))
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
			b.Pos(sdk.At("log/iface.go", 1, 1))
			b.Field("ID", storefixture.Named("string"), nil)
			b.Field("DependsOn", storefixture.Slice(storefixture.Named("string")), nil)
			b.Field("Body", storefixture.Named("string"), nil)
		}).
		Interface("Log", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("log/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Append", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("e", storefixture.PkgNamed("example.com/log", "Entry"))
				m.Return(storefixture.Named("error"))
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
		f := fieldOf(t, contractIn(t, causalEntry(t, true)).Fixture, "E")
		testkit.Equal(t, partNames(f), []string{"ID", "Body"},
			"the scalar fields still discriminate; the cause list is dropped")
	})

	t.Run("keeps it where nothing claims a precondition", func(t *testing.T) {
		t.Parallel()
		// The same struct without the claim is an ordinary value, and dropping
		// a field there would lose discrimination for nothing.
		f := fieldOf(t, contractIn(t, causalEntry(t, false)).Fixture, "E")
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
			b.Pos(sdk.At("idx/iface.go", 1, 1))
			b.Field("Name", storefixture.Named("string"), nil)
			b.Field("Offset", storefixture.Named("int"), nil)
		}).
		Interface("Sorter", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("idx/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Less", func(m *storefixture.MethodBuilder) {
				m.Param("i", storefixture.Named("int"))
				m.Param("j", storefixture.Named("int"))
				m.Return(storefixture.Named("bool"))
			})
			i.Method("Seek", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("p", storefixture.PkgNamed("example.com/idx", "Page"))
				m.Return(storefixture.Named("error"))
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

// missFixture is a two-slot comma-ok read stamped with the given shape, so the
// gate can be exercised independently of the signature.
//
// The same declaration under every stamp: what changes between the cases is the
// classification, which is the whole point of gating on it.
func missFixture(t *testing.T, shapeName string) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("miss", "example.com/miss").
		Struct("Value", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("miss/iface.go", 1, 1))
			b.Field("Body", storefixture.Named("string"), nil)
		}).
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("miss/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Load", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.PkgNamed("example.com/miss", "Value"))
				m.Return(storefixture.Named("bool"))
			})
		}).
		Build()
	stamp(s, "Load", shapeName)
	return s
}

// hooksFixture is a firing method beside a registration taking one parameter of
// the given type.
func hooksFixture(t *testing.T, param *sdk.TypeRef) *sdk.Store {
	t.Helper()
	return hooksStore(t, func(m *storefixture.MethodBuilder) { m.Param("fn", param) })
}

// hooksTwoParamFixture gives the registration two parameters, so which one
// registers is unstated.
func hooksTwoParamFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return hooksStore(t, func(m *storefixture.MethodBuilder) {
		m.Param("fn", storefixture.Func(nil, nil))
		m.Param("name", storefixture.Named("string"))
	})
}

// hooksStore builds the fixture with the given registration signature.
func hooksStore(t *testing.T, register func(*storefixture.MethodBuilder)) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("hk", "example.com/hk").
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("hk/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Fire", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("event", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
			i.Method("OnEvent", register)
		}).
		Build()
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Fire" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaMixins.Set(bag, []string{suite.MixinHooks}, "test")
			shape.MixinParamKey(suite.MixinHooks, suite.MixinHooksParam).
				Set(bag, "OnEvent", "test")
		}
	}
	return s
}

// sampleFixture is a single-parameter method beside a builder producing
// produces values of the given type.
func sampleFixture(t *testing.T, built string, produces int) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("sp", "example.com/sp").
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("sp/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Process", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("input", storefixture.Named("string"))
				m.Return(storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
			i.Method("NewInput", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				for range produces {
					m.Return(storefixture.Named(built))
				}
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Process" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaMixins.Set(bag, []string{suite.MixinSample}, "test")
			shape.MixinParamKey(suite.MixinSample, suite.MixinSampleParam).
				Set(bag, "NewInput", "test")
		}
	}
	return s
}

// partitionFixture is a partitioned write beside its reader, with the isolation
// axis named by the given parameter.
func partitionFixture(t *testing.T, axis string) *sdk.Store {
	t.Helper()
	return partitionStore(t, axis, true)
}

// payloadlessPartitionFixture is a write whose every parameter the reader also
// takes, so nothing distinguishes what was written from where.
func payloadlessPartitionFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return partitionStore(t, "part", false)
}

// partitionWiderReaderFixture gives the reader a parameter the writer does not
// take, so a check cannot spell the call.
func partitionWiderReaderFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return partitionCustom(t, "part", func(m *storefixture.MethodBuilder) {
		m.Param("ctx", storefixture.PkgNamed("context", "Context"))
		m.Param("part", storefixture.Named("string"))
		m.Param("key", storefixture.Named("string"))
		m.Param("at", storefixture.Named("int"))
		m.Return(storefixture.Named("string"))
		m.Return(storefixture.Named("error"))
	}, true)
}

// partitionUnderivableAxisFixture makes the axis a type no literal can be
// written for, so it has no alternate to vary along.
func partitionUnderivableAxisFixture(t *testing.T) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("pt", "example.com/pt").
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("pt/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("part", storefixture.Func(nil, nil))
				m.Param("key", storefixture.Named("string"))
				m.Param("value", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
			i.Method("Read", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("part", storefixture.Func(nil, nil))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
	stampPartition(s, "part")
	return s
}

// partitionCustom builds the fixture with a caller-supplied reader signature.
func partitionCustom(
	t *testing.T, axis string, read func(*storefixture.MethodBuilder), payload bool,
) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("pt", "example.com/pt").
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("pt/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("part", storefixture.Named("string"))
				m.Param("key", storefixture.Named("string"))
				if payload {
					m.Param("value", storefixture.Named("string"))
				}
				m.Return(storefixture.Named("error"))
			})
			i.Method("Read", read)
		}).
		Build()
	stampPartition(s, axis)
	return s
}

// stampPartition attaches the mixin with its read partner and axis.
func stampPartition(s *sdk.Store, axis string) {
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Put" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaMixins.Set(bag, []string{suite.MixinPartition}, "test")
			shape.MixinParamKey(suite.MixinPartition, suite.MixinPartitionRead).
				Set(bag, "Read", "test")
			if axis != "" {
				shape.MixinParamKey(suite.MixinPartition, suite.MixinPartitionAxis).
					Set(bag, axis, "test")
			}
		}
	}
}

// partitionStore builds the fixture, optionally giving the write a payload the
// reader does not share.
func partitionStore(t *testing.T, axis string, payload bool) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("pt", "example.com/pt").
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("pt/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("part", storefixture.Named("string"))
				m.Param("key", storefixture.Named("string"))
				if payload {
					m.Param("value", storefixture.Named("string"))
				}
				m.Return(storefixture.Named("error"))
			})
			i.Method("Read", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("part", storefixture.Named("string"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Put" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaMixins.Set(bag, []string{suite.MixinPartition}, "test")
			shape.MixinParamKey(suite.MixinPartition, suite.MixinPartitionRead).
				Set(bag, "Read", "test")
			if axis != "" {
				shape.MixinParamKey(suite.MixinPartition, suite.MixinPartitionAxis).
					Set(bag, axis, "test")
			}
		}
	}
	return s
}

// checkNamed returns the method's check reporting under subtest.
func checkNamed(t *testing.T, c *suite.Contract, method, subtest string) *suite.Check {
	t.Helper()
	for _, m := range c.Methods {
		if m.Name != method {
			continue
		}
		for _, ck := range m.Checks {
			if ck.Subtest == subtest {
				return ck
			}
		}
	}
	t.Fatalf("%s carries no %q check", method, subtest)
	return nil
}

// relationalFixture is a method whose effect is out of band beside the method
// that observes it, with the partner named by the given identifier.
func relationalFixture(t *testing.T, partner string) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("fx", "example.com/fx").
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("fx/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Touch", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
			i.Method("Observed", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("int"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Touch" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaMixins.Set(bag, []string{suite.MixinSideEffect}, "test")
			if partner != "" {
				shape.MixinParamKey(suite.MixinSideEffect, suite.MixinSideEffectParam).
					Set(bag, partner, "test")
			}
		}
	}
	return s
}

// mixinFixture attaches the named mixin to a method, with an optional parameter
// value, so selection can be exercised without a whole corpus package.
//
// One declaration under every classification: what changes between the cases is
// the stamp, which is what the gate reads.
func mixinFixture(t *testing.T, mixinName, param string) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("miss", "example.com/miss").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("miss/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			// Armed the way every corpus fixture is: the coverage header's
			// "checked somewhere else" turns on the model tier actually
			// running, and an unarmed interface would put every law-bearing
			// classification in the consumer's own list.
			i.Directive(storefixture.Directive("model"))
			i.Method("Load", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
	if mixinName == "" {
		return s
	}
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Load" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaMixins.Set(bag, []string{mixinName}, "test")
			if param != "" {
				shape.MixinParamKey(mixinName, mixinParamOf(mixinName)).Set(bag, param, "test")
			}
		}
	}
	return s
}

// mixinParamOf names the parameter each gated mixin reads, so the fixture does
// not restate the pairing the generator already declares.
func mixinParamOf(mixinName string) string {
	if mixinName == suite.MixinTimeout {
		return suite.MixinTimeoutParam
	}
	return suite.MixinOrderAfterParam
}

// missShapeFixture stamps a lookup classification onto a method returning only
// the given slot, which is what a source directive overriding a detector
// produces: an authority above the annotator, and invisible to this generator.
func missShapeFixture(t *testing.T, ret *sdk.TypeRef) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("miss", "example.com/miss").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("miss/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Load", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(ret)
			})
		}).
		Build()
	stamp(s, "Load", readerwithbool.Name)
	return s
}

// missNoFlagFixture returns two values and no bool, so nothing flags the miss
// and every slot carries it.
func missNoFlagFixture(t *testing.T) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("miss", "example.com/miss").
		Struct("Value", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("miss/iface.go", 1, 1))
			b.Field("Body", storefixture.Named("string"), nil)
		}).
		Struct("Meta", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("miss/iface.go", 1, 1))
			b.Field("Revision", storefixture.Named("int"), nil)
		}).
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("miss/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Load", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Return(storefixture.PkgNamed("example.com/miss", "Value"))
				m.Return(storefixture.PkgNamed("example.com/miss", "Meta"))
			})
		}).
		Build()
	stamp(s, "Load", pointerreader.Name)
	return s
}

// missNoInputFixture is the same shape with nothing after the context, so the
// miss has nowhere to come from.
func missNoInputFixture(t *testing.T) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("miss", "example.com/miss").
		Struct("Value", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At("miss/iface.go", 1, 1))
			b.Field("Body", storefixture.Named("string"), nil)
		}).
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("miss/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Load", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Return(storefixture.PkgNamed("example.com/miss", "Value"))
				m.Return(storefixture.Named("bool"))
			})
		}).
		Build()
	stamp(s, "Load", readerwithbool.Name)
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
			b.Pos(sdk.At("kv/iface.go", 1, 1))
			b.Field("Body", storefixture.Named("string"), nil)
		}).
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("kv/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Put", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("key", storefixture.Named("string"))
				m.Param("v", storefixture.PkgNamed("example.com/kv", "Payload"))
				m.Return(storefixture.Named("error"))
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
			i.Pos(sdk.At("mut/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Set", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
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
		testkit.False(t, f.Composed(), "and composes no value")
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

// A batch read answers once per key, which is the one detector claim about
// arity rather than absence — and the one the miss family cannot state.
func TestBatchSizeCheck(t *testing.T) {
	t.Parallel()

	t.Run("emits for a variadic read with a derivable pair", func(t *testing.T) {
		t.Parallel()
		testkit.True(
			t,
			hasCheckIn(
				t,
				batchFixture(t, storefixture.Named("string")),
				"GetAll",
				"answers once per key",
			),
			"two keys is what distinguishes per-key from at-all",
		)
	})

	t.Run("varies the key and holds nothing else", func(t *testing.T) {
		t.Parallel()
		ck := checkNamed(
			t,
			contractIn(t, batchFixture(t, storefixture.Named("string"))),
			"GetAll",
			"answers once per key",
		)
		testkit.Equal(t, ck.SecondCall, []string{"keys", "keysOther"},
			"the call is handed both derived values")
	})

	t.Run("emits nothing for a shape that is not stamped batchreader", func(t *testing.T) {
		t.Parallel()
		// A structural stamp is what says the slice is per-key. Without it a
		// variadic method returning a slice is any of a dozen things.
		testkit.False(t, hasCheckIn(t, unstampedBatchFixture(t), "GetAll", "answers once per key"),
			"an unstamped variadic read owes no count")
	})

	t.Run("emits nothing where the read takes more than the keys", func(t *testing.T) {
		t.Parallel()
		// A second parameter is a batch read of something else — a limit, a
		// scope — and the count claim is about the keys alone.
		testkit.False(t, hasCheckIn(t, widerBatchFixture(t), "GetAll", "answers once per key"),
			"a read taking more than a batch of keys is a different shape")
	})

	t.Run("emits nothing where the key admits no second value", func(t *testing.T) {
		t.Parallel()
		// Two keys is the whole content, and a func-typed key yields none — so
		// there is no second element to request.
		testkit.False(
			t,
			hasCheckIn(
				t,
				batchFixture(t, storefixture.Func(nil, nil)),
				"GetAll",
				"answers once per key",
			),
			"a key nothing can be written for is one nothing can be varied along",
		)
	})
}

// batchFixture is a variadic read stamped batchreader, keyed on the given type.
func batchFixture(t *testing.T, key *sdk.TypeRef) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("batch", "example.com/batch").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("batch/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("GetAll", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Variadic("keys", key)
				m.Return(storefixture.Slice(storefixture.Named("string")))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
	stamp(s, "GetAll", batchreader.Name)
	return s
}

// widerBatchFixture takes a second parameter beside the batch.
func widerBatchFixture(t *testing.T) *sdk.Store {
	t.Helper()
	s := storefixture.New().
		Package("batch", "example.com/batch").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("batch/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("GetAll", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("limit", storefixture.Named("int"))
				m.Variadic("keys", storefixture.Named("string"))
				m.Return(storefixture.Slice(storefixture.Named("string")))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
	stamp(s, "GetAll", batchreader.Name)
	return s
}

// unstampedBatchFixture is the same shape with no classification on it.
func unstampedBatchFixture(t *testing.T) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("batch", "example.com/batch").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("batch/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("GetAll", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Variadic("keys", storefixture.Named("string"))
				m.Return(storefixture.Slice(storefixture.Named("string")))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
}

// Two mixins that name a partner and compare against it: one asks whether the
// pair agree, the other whether the failure carries what the pair reports.
func TestPartnerComparisonChecks(t *testing.T) {
	t.Parallel()

	t.Run("emits where the validator answers about the same value", func(t *testing.T) {
		t.Parallel()
		testkit.True(
			t,
			hasCheckIn(
				t,
				pairFixture(t, "validates", "fn", "Validate", true),
				"Store",
				"validates",
			),
			"a validator over the writer's own parameter is one the check can call",
		)
	})

	t.Run("emits nothing where the validator answers about something else", func(t *testing.T) {
		t.Parallel()
		// A check receives the writer's arguments and nothing else.
		testkit.False(
			t,
			hasCheckIn(
				t,
				pairFixture(t, "validates", "fn", "Validate", false),
				"Store",
				"validates",
			),
			"a validator over a different parameter list is one the check cannot call",
		)
	})

	t.Run("emits where the cause reports an error and nothing else", func(t *testing.T) {
		t.Parallel()
		testkit.True(
			t,
			hasCheckIn(t, pairFixture(t, "wrappedvia", "fn", "Cause", true), "Store", "wrappedvia"),
			"a cause that reports only an error is one the failure can be held up to",
		)
	})
}

// pairFixture is a writer beside a partner the named mixin points at, over the
// writer's own parameter or over another type.
func pairFixture(t *testing.T, mixinName, param, partner string, sameType bool) *sdk.Store {
	t.Helper()
	over := storefixture.Named("string")
	if !sameType {
		over = storefixture.Named("int")
	}
	s := storefixture.New().
		Package("pair", "example.com/pair").
		Interface("Mixed", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("pair/iface.go", 1, 1))
			i.Directive(storefixture.Directive("suite"))
			i.Method("Store", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Param("v", storefixture.Named("string"))
				m.Return(storefixture.Named("error"))
			})
			i.Method(partner, func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				if partner != "Cause" {
					m.Param("v", over)
				}
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name != "Store" {
				continue
			}
			bag := m.EnsureMeta()
			shape.MetaMixins.Set(bag, []string{mixinName}, "test")
			shape.MixinParamKey(mixinName, param).Set(bag, partner, "test")
		}
	}
	return s
}
