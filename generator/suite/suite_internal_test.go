// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"strings"
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
)

// modelFixture builds one interface carrying the supplied directives and
// answers its node, for the predicate the header trusts.
func modelFixture(t *testing.T, generic bool, directives ...*sdk.Directive) *sdk.Interface {
	t.Helper()
	s := storefixture.New().
		Package("p", "example.com/p").
		Interface("Store", func(i *storefixture.InterfaceBuilder) {
			i.Pos(sdk.At("p/iface.go", 1, 1))
			for _, d := range directives {
				i.Directive(d)
			}
			if generic {
				i.TypeParam("V", nil)
			}
			i.Method("Get", func(m *storefixture.MethodBuilder) {
				m.Param("ctx", storefixture.PkgNamed("context", "Context"))
				m.Return(storefixture.Named("error"))
			})
		}).
		Build()
	items := s.Nodes().Interfaces().Items()
	testkit.Equal(t, len(items), 1, "one interface")
	return items[0]
}

// TestModelWillRun holds the header's "checked somewhere else" predicate to
// the model generator's own answer: armed, and — where generic — witnessed.
// A drifted copy is a header pointing at output that does not exist.
func TestModelWillRun(t *testing.T) {
	t.Parallel()

	testkit.False(t, modelWillRun(modelFixture(t, false)),
		"an unarmed interface runs no model tier")
	testkit.True(t, modelWillRun(modelFixture(t, false, storefixture.Directive(ModelDirective))),
		"an armed concrete interface runs it")
	testkit.False(t, modelWillRun(modelFixture(t, true, storefixture.Directive(ModelDirective))),
		"an armed generic interface without witnesses is refused by the model generator")
	testkit.True(t, modelWillRun(modelFixture(t, true,
		storefixture.Directive(ModelDirective, storefixture.KV(ModelWitnessKey, "int")))),
		"and the witness list is what makes it run")
}

// TestStampedSentinel pins the sentinel lift: a qualified stamp arrives as a
// renderable reference, and every form that cannot render keeps the
// presence-only check rather than a dangling name.
func TestStampedSentinel(t *testing.T) {
	t.Parallel()

	key := shape.MixinParamKey(MixinOrderAfter, "unready")
	method := func(stamp string) Method {
		src := &node.Method{Name: "Commit"}
		if stamp != "" {
			key.Set(src.EnsureMeta(), stamp, "test")
		}
		return Method{Sig: &golang.Sig{Name: "Commit", Source: src}}
	}

	testkit.True(t, stampedSentinel(method(""), key) == nil,
		"an unstamped declaration keeps the presence-only check")
	got := stampedSentinel(method("example.com/order.ErrNotReady"), key)
	testkit.True(t, got != nil, "a qualified stamp lifts")
	testkit.True(t, stampedSentinel(method("ErrNotReady"), key) == nil,
		"a bare spelling resolves to no package and stays presence-only")
}

// TestWhyUnseeded holds the three exits apart, because the fix differs for
// each and a single "no seed was derived" sends the reader to look for all
// three.
func TestWhyUnseeded(t *testing.T) {
	t.Parallel()

	testkit.Assert(t, whyUnseeded(nil, nil)).Contains("classified as a write",
		"an interface with no writer wants the consumer's own seed")

	testkit.Assert(t, whyUnseeded([]string{"Touch"}, nil)).Contains("reports no error",
		"a write that cannot fail out loud cannot seed, and the fix is its signature")

	testkit.Assert(t, whyUnseeded(nil, []string{"Store"})).Contains("through the fixture",
		"an unwritable argument wants a fixture, not a signature change")

	// Closest-to-usable first. A method that reached its arguments needs one
	// literal; a mute one needs a new return. Naming the mute writer ahead of
	// it would point at the larger change.
	both := whyUnseeded([]string{"Touch"}, []string{"Store"})
	testkit.Assert(t, both).Contains("Store", "the argument case is named")
	testkit.False(t, strings.Contains(both, "Touch"),
		"and the mute one is not, this run")
}

// param builds one call argument at a named type, with the recorded-call field
// eidos derives from the identifier.
func param(name, typ string) golang.Param {
	return golang.Param{
		Name:   name,
		Field:  strings.ToUpper(name[:1]) + name[1:],
		Type:   sdk.Builtin(typ),
		Source: storefixture.Named(typ),
	}
}

// method wraps a parameter list in the shape [partnerArgs] reads.
func method(name string, params ...golang.Param) Method {
	return Method{Sig: &golang.Sig{Name: name, Params: params}}
}

// partnerArgs decides whether a relational check can be written at all, so its
// three answers are the three shapes of coverage this generator can have: the
// check, the absence with a reason, and a wrong check it must never write.
//
// Driven directly rather than through a selector because the selectors add
// their own preconditions — arity, return shape, contract role — and a table
// that had to satisfy all of them would be testing those instead.
func TestPartnerArgs(t *testing.T) {
	t.Parallel()

	m := method("Move",
		param("key", "string"), param("weight", "int"))

	t.Run("spells an identically named parameter", func(t *testing.T) {
		t.Parallel()
		args, why := partnerArgs(Fixture{}, m, method("Seen", param("key", "string")))
		testkit.Equal(t, why, "", "the ordinary case declines nothing")
		testkit.Equal(t, strings.Join(args, ", "), "key", "and hands the partner the method's own identifier")
	})

	t.Run("spells the sole parameter of the type under any name", func(t *testing.T) {
		t.Parallel()
		// The widening. `k` is not `key`, and one string is one candidate, so
		// the correspondence is derived rather than guessed — which is what
		// identical spelling was standing in for.
		args, why := partnerArgs(Fixture{}, m, method("Seen", param("k", "string")))
		testkit.Equal(t, why, "", "one candidate of the type needs no rename to be unambiguous")
		testkit.Equal(t, strings.Join(args, ", "), "key", "and it resolves to the parameter that has it")
	})

	t.Run("declines where two parameters could be it", func(t *testing.T) {
		t.Parallel()
		two := method("Move", param("from", "string"), param("to", "string"))
		_, why := partnerArgs(Fixture{}, two, method("At", param("where", "string")))
		testkit.Assert(t, why).Contains("more than one of the annotated method's parameters could be it",
			"the ambiguity is named rather than resolved by visit order")
		testkit.Assert(t, why).Contains("At takes where", "and the parameter that caused it is named")
	})

	t.Run("declines where nothing of the type is in scope", func(t *testing.T) {
		t.Parallel()
		_, why := partnerArgs(Fixture{}, method("Emit", param("id", "string")),
			method("Count", param("bucket", "int")))
		testkit.Assert(t, why).Contains("which the annotated method has nothing to fill",
			"an absent type is a different problem from an ambiguous one")
	})

	t.Run("does not let one candidate serve two slots", func(t *testing.T) {
		t.Parallel()
		// Both slots want the sole string, and consuming it for whichever came
		// first would leave the second unfillable — an answer that depends on
		// iteration order. Declining both is what makes "unambiguous" mean it.
		_, why := partnerArgs(Fixture{}, m, method("Between", param("a", "string"), param("b", "string")))
		testkit.True(t, why != "", "a contested candidate settles nothing for either slot")
	})

	t.Run("takes an exact match over a type match", func(t *testing.T) {
		t.Parallel()
		// `weight` is spelled on both sides and `n` is the only other int, so
		// a derivation running the passes in the wrong order would cross them.
		wide := method("Adjust", param("weight", "int"), param("n", "int"))
		args, why := partnerArgs(Fixture{}, wide, method("Seen", param("n", "int"), param("weight", "int")))
		testkit.Equal(t, why, "", "both slots resolve")
		testkit.Equal(t, strings.Join(args, ", "), "n, weight", "each to the parameter that spells it")
	})
}
