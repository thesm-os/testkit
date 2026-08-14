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
