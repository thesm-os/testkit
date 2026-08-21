// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package subject_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/subject"
)

// The word a parameter is drawn under decides which fixture field every
// tier reads it from, so the two tiers agree only if this does.
func TestDrawWord(t *testing.T) {
	t.Parallel()

	t.Run("a named type supplies the word", func(t *testing.T) {
		t.Parallel()
		p := golang.Param{Name: "k", Source: storefixture.Named("Key")}
		testkit.Equal(t, subject.DrawWord(p), "Key",
			"the declared type is what two methods taking one value have in common")
		testkit.Equal(t, subject.DrawField(p), "Key", "and the field is its exported form")
	})

	t.Run("a predeclared type falls back to the parameter's name", func(t *testing.T) {
		t.Parallel()
		// Every `string` would collide with every other, so the type says
		// nothing and the identifier is the only word available.
		p := golang.Param{Name: "key", Source: storefixture.Named("string")}
		testkit.Equal(t, subject.DrawWord(p), "key", "a predeclared type names nothing")
		testkit.Equal(t, subject.DrawField(p), "Key", "the field is still exported")
	})

	t.Run("an anonymous type falls back too", func(t *testing.T) {
		t.Parallel()
		p := golang.Param{Name: "payload"}
		testkit.Equal(t, subject.DrawWord(p), "payload", "a composite has no name to spell")
	})
}

// A name two methods share at one type is one field; at two types it is
// two, and both get qualified so neither spelling wins by walk order.
func TestGroupParams(t *testing.T) {
	t.Parallel()

	t.Run("one name at one type is one field", func(t *testing.T) {
		t.Parallel()
		// A `key Key` on the reader and one on the deleter are the same value
		// to a conformance run; separate fields would let a consumer override
		// one and silently not the other.
		groups := subject.GroupParams([]subject.Method{
			method("Get", named("k", "Key")),
			method("Delete", named("k", "Key")),
		})

		testkit.Len(t, groups, 1, "the second method contributes no field")
		testkit.Equal(t, groups[0].Name, "Key", "and the field keeps the plain name")
		testkit.Equal(t, groups[0].Method, "Get", "attributed to the method that introduced it")
	})

	t.Run("a declared type separates two parameters sharing a name", func(t *testing.T) {
		t.Parallel()
		// The type is the word, so `s Session` and `s string` were never one
		// field to begin with — no qualification is needed to tell them apart.
		groups := subject.GroupParams([]subject.Method{
			method("Put", named("s", "Session")),
			method("Get", named("s", "string")),
		})

		testkit.Equal(t, names(groups), []string{"Session", "S"},
			"each draws under its own word, and neither is qualified")
	})

	t.Run("one name at two predeclared types is two fields, both qualified", func(t *testing.T) {
		t.Parallel()
		// Where neither type supplies a word, the parameter's name is all
		// there is and the two genuinely contest it. A fixture keyed on that
		// alone holds one of them and hands it to the method that takes the
		// other, which does not compile.
		groups := subject.GroupParams([]subject.Method{
			method("Get", named("k", "string")),
			method("Count", named("k", "int")),
		})

		testkit.Equal(t, names(groups), []string{"GetK", "CountK"},
			"qualified by the method that introduced each type, which a reader can find")
	})

	t.Run("an uncontested name beside a contested one keeps its spelling", func(t *testing.T) {
		t.Parallel()
		// Qualifying everything would rename fields that had no conflict, and
		// a regeneration that renames a field a consumer named is a break.
		groups := subject.GroupParams([]subject.Method{
			method("Put", named("k", "Key"), named("v", "Value")),
			method("Count", named("k", "Prefix")),
		})

		byMethod := map[string]string{}
		for _, g := range groups {
			byMethod[g.Method+"/"+subject.DrawField(g.Param)] = g.Name
		}
		testkit.Equal(t, byMethod["Put/Value"], "Value", "Value is claimed by one type only")
	})

	t.Run("the context is not a field", func(t *testing.T) {
		t.Parallel()
		// The fixture supplies what a check chooses, and nothing chooses the
		// context: the checks that vary it build their own.
		groups := subject.GroupParams([]subject.Method{
			method("Get", ctxParam(), named("k", "Key")),
		})

		testkit.Len(t, groups, 1, "only the drawn parameter opens a field")
		testkit.Equal(t, groups[0].Name, "Key", "and it is the one after the context")
	})
}

// Membership is by name AND type, which is what stops one method's value
// being handed to another method that took a different one.
func TestFindGroup(t *testing.T) {
	t.Parallel()

	groups := subject.GroupParams([]subject.Method{method("Get", named("k", "string"))})

	testkit.True(t, subject.FindGroup(groups, named("k", "string")),
		"the same word at the same type is already held")
	testkit.False(t, subject.FindGroup(groups, named("k", "int")),
		"the same word at a different type is a different value")
	testkit.False(t, subject.FindGroup(groups, named("other", "string")),
		"a different word is a different field")
}

// A check naming its own argument has to name the field the value landed
// in, which is not the parameter's name wherever two types contest one.
func TestFixtureFieldFor(t *testing.T) {
	t.Parallel()

	methods := []subject.Method{
		method("Get", named("k", "string")),
		method("Count", named("k", "int")),
	}
	fx := subject.Fixture{Groups: subject.GroupParams(methods)}

	t.Run("a contested parameter reads the qualified field", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, fx.FieldFor(named("k", "string")), "GetK",
			"the field the value actually landed in")
		testkit.Equal(t, fx.FieldFor(named("k", "int")), "CountK",
			"and the other type reads the other field")
	})

	t.Run("a parameter no group holds falls back to its own field", func(t *testing.T) {
		t.Parallel()
		// A `""` would compose `cfg.Fixture.` into generated source rather
		// than failing where a reader could see it.
		p := golang.Param{Name: "n", Field: "N", Source: storefixture.Named("int")}
		testkit.Equal(t, fx.FieldFor(p), "N", "the fixture's name for an uncontested field")
	})
}

// Field is the lookup a template makes, and its miss has to be a miss.
func TestFixtureField(t *testing.T) {
	t.Parallel()

	fx := subject.Fixture{Fields: []subject.FixtureField{
		{Name: "Key", Sample: golang.Sample{Text: `"k"`}},
	}}

	got, ok := fx.Field("Key")
	testkit.True(t, ok, "a derived field is found")
	testkit.Equal(t, got.Sample.Text, `"k"`, "with its value")

	_, ok = fx.Field("Missing")
	testkit.False(t, ok, "and one nothing derived is reported absent rather than zero")
}

// Both halves of a field have to be derivable, because a check needs a
// value the subject holds AND one it does not.
//
//nolint:thelper // the case body is the test, not a helper; see core/lawid
func TestFixtureFieldOK(t *testing.T) {
	t.Parallel()

	companion := sdk.NewIdent("cfgDefaults")

	testkit.TableTest(t, []okCase{
		{"both values derived", subject.FixtureField{
			Sample: sample("a"), Other: sample("b"),
		}, true},
		{"a companion supplies the sample", subject.FixtureField{
			Companion: companion, Other: sample("b"),
		}, true},
		{
			// The companion answers "a value this type accepts". A miss check
			// needs "a value that should not be found", which is a different
			// claim — and accepting the companion as proof of both let the
			// alternate render as a silent zero that real data collides with.
			"a companion does not supply the alternate", subject.FixtureField{
				Companion: companion,
			}, false,
		},
		{"no sample and no companion", subject.FixtureField{
			Other: sample("b"),
		}, false},
		{"a sample with no alternate", subject.FixtureField{
			Sample: sample("a"),
		}, false},
		{
			// A struct's value is composed from its parts, so the whole never
			// had to be derivable on its own.
			"a composed field needs neither", subject.FixtureField{
				Parts: []subject.FixturePart{{Name: "Region", Sample: sample("a")}},
			}, true,
		},
	}, func(t *testing.T, tc okCase) {
		testkit.Equal(t, tc.field.OK(), tc.want,
			"a check emitted against an underivable value passes whatever the subject does")
	})
}

// The refusal a consumer reads has to separate a fact about their type
// from a fact about this run's own inputs.
func TestFixtureFieldReason(t *testing.T) {
	t.Parallel()

	t.Run("a type admitting no literal is settled", func(t *testing.T) {
		t.Parallel()
		f := subject.FixtureField{
			Sample: golang.Sample{Refusal: golang.RefusedNoLiteral},
		}
		testkit.Equal(t, f.Reason(), "which no literal can be written for",
			"a func or a channel is the same answer under every run")
	})

	t.Run("a package the run did not reach is not", func(t *testing.T) {
		t.Parallel()
		// Reporting this as settled sends an author to change source that is
		// already correct; the same source under a wider pattern resolves.
		f := subject.FixtureField{
			Sample: golang.Sample{Refusal: golang.RefusedUnresolved},
		}
		testkit.Equal(t, f.Reason(), "which this run did not resolve, so no value was derived for it",
			"the fix is the run's patterns, not the declaration")
	})
}

// Choose is what lets one sub-template spell both values, so the two
// spellings cannot drift apart.
func TestFixtureFieldChoose(t *testing.T) {
	t.Parallel()

	f := subject.FixtureField{
		Name:   "Key",
		Sample: sample("a"),
		Other:  sample("b"),
		Parts: []subject.FixturePart{
			{Name: "Region", Sample: sample("eu"), Other: sample("us"), Pool: "RegionPool"},
		},
	}

	t.Run("the canonical value draws index zero", func(t *testing.T) {
		t.Parallel()
		v := f.Choose(false)
		testkit.Equal(t, v.Value.Text, `"a"`, "the sample")
		testkit.Equal(t, v.Alternate, 0, "which is the pool's first member")
		testkit.Equal(t, v.Parts[0].Value.Text, `"eu"`, "and its parts follow it")
	})

	t.Run("the alternate draws index one", func(t *testing.T) {
		t.Parallel()
		// Carried as the index rather than a bool because that is what the
		// emitted subscript spells.
		v := f.Choose(true)
		testkit.Equal(t, v.Value.Text, `"b"`, "the second value")
		testkit.Equal(t, v.Alternate, 1, "which is the member that funds a miss")
		testkit.Equal(t, v.Parts[0].Value.Text, `"us"`, "and its parts follow it too")
		testkit.Equal(t, v.Parts[0].Pool, "RegionPool", "the part keeps the pool it draws from")
	})

	t.Run("the companion field is named by suffix", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, f.OtherName(), "KeyOther", "one rule, so a check can compose it")
		testkit.True(t, f.Composed(), "a field with parts is composed")
	})
}

// The signature questions every check body is gated on.
func TestMethodSignatureQueries(t *testing.T) {
	t.Parallel()

	get := method("Get", ctxParam(), named("k", "Key"))
	get.Returns = []golang.Return{{Source: storefixture.Named("Value")}, {Error: true}}

	bare := method("Close")
	bare.Returns = []golang.Return{{Error: true}}

	t.Run("a context is recognised by its qualified spelling", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, get.TakesContext(), "the first parameter is context.Context")
		testkit.False(t, bare.TakesContext(), "and a method taking none says so")
	})

	t.Run("the call passes what follows the context", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, get.CallArgs(), 1, "the context is not an argument a check chooses")
		testkit.True(t, get.HasInput(), "and the method has a lever the harness can pull")
		testkit.Len(t, bare.CallArgs(), 0, "a method taking nothing passes nothing")
		testkit.False(t, bare.HasInput(),
			"so a check meaning \"this input misses\" cannot reach the failure it is about")
	})

	t.Run("the error is not a value return", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, get.ValueReturns(), 1, "a zero-value check compares the rest")
		testkit.Len(t, bare.ValueReturns(), 0, "and an error-only method has none")
	})

	t.Run("the variadic parameter is found where one exists", func(t *testing.T) {
		t.Parallel()
		// Go allows at most one and only in final position, so one answer
		// covers the signature.
		put := method("Put", ctxParam(), named("k", "Key"))
		put.Params = append(put.Params, golang.Param{
			Name: "opts", Variadic: true, Source: storefixture.Named("Option"),
		})
		testkit.True(t, put.VariadicParam() != nil, "the method declares one")
		testkit.Equal(t, put.VariadicParam().Name, "opts", "and it is the last")
		testkit.True(t, get.VariadicParam() == nil, "a method declaring none reports none")
	})
}

// A method's stamps, read the one way, so the two tiers select from the
// same set.
func TestMethodStamps(t *testing.T) {
	t.Parallel()

	m := method("Put", ctxParam(), named("k", "Key"))
	m.Mixins = []string{"idempotent", "ttl"}
	m.Contracts = []string{"outbox"}
	m.MixinParams = map[string]string{"ttl.clock": "pkg.Clock"}
	m.ContractRoles = map[string]string{"outbox": "writer"}
	m.ContractPartners = map[string]string{"outbox.reader": "other.Drain"}
	m.ContractParams = map[string]string{"outbox.conflict": "ErrConflict"}

	t.Run("an attached mixin is found and an unattached one is not", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, m.HasMixin("ttl"), "the annotator attached it")
		testkit.False(t, m.HasMixin("cas"), "and did not attach this")
	})

	t.Run("a mixin's argument comes back verbatim", func(t *testing.T) {
		t.Parallel()
		// Right for identity and wrong for a call site, which is the contract
		// axis's problem rather than this one's.
		v, ok := m.MixinParam("ttl", "clock")
		testkit.True(t, ok, "the directive wrote one")
		testkit.Equal(t, v, "pkg.Clock", "qualified as the resolver left it")

		_, ok = m.MixinParam("ttl", "absent")
		testkit.False(t, ok, "and a param nobody wrote is reported absent")
	})

	t.Run("a contract needs both the name and the role", func(t *testing.T) {
		t.Parallel()
		// A protocol rather than a property: an outbox check written against
		// the subscriber would call the wrong method.
		testkit.True(t, m.HasContractRole("outbox", "writer"), "the method fills this role")
		testkit.False(t, m.HasContractRole("outbox", "reader"), "and not the other one")
		testkit.False(t, m.HasContractRole("saga", "writer"), "nor this role in another contract")
	})

	t.Run("a partner is cut back to the name a call can use", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, m.ContractPartner("outbox", "reader"), "Drain",
			"a generated call is on a subject the check already holds")
		testkit.Equal(t, m.ContractPartner("outbox", "absent"), "",
			"and a role the directive named none for is empty")
	})

	t.Run("a param is not, because nothing resolves it", func(t *testing.T) {
		t.Parallel()
		v, ok := m.ContractParam("outbox", "conflict")
		testkit.True(t, ok, "the directive wrote one")
		testkit.Equal(t, v, "ErrConflict", "a param names a value, not a callable")
	})

	t.Run("the classification set composes shape, mixins and contracts", func(t *testing.T) {
		t.Parallel()
		// The one home of the composition: the model generator selects from
		// exactly this, so the two tiers cannot disagree about what the run
		// classified.
		testkit.Equal(t, m.Classifications(),
			[]string{"idempotent", "ttl", "outbox"},
			"every stamp in tiers' one namespace")
	})

	t.Run("a method with no source node has no shape", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, m.Shape(), "", "and the accessor does not panic reaching for one")
	})
}

// okCase is one field and whether both its values could be produced.
type okCase struct {
	name  string
	field subject.FixtureField
	want  bool
}

func (c okCase) Name() string { return c.name }

// method builds a signature with the given parameters and no returns.
func method(name string, params ...golang.Param) subject.Method {
	return subject.Method{Sig: &golang.Sig{Name: name, Params: params}}
}

// named is a parameter of a declared type, which is the word it draws under.
func named(ident, typ string) golang.Param {
	return golang.Param{Name: ident, Field: golang.ExportedName(ident), Source: storefixture.Named(typ)}
}

// ctxParam is the context every second method takes. Named by its
// qualified spelling, which is what [golang.IsContext] answers from.
func ctxParam() golang.Param {
	return golang.Param{Name: "ctx", Source: storefixture.PkgNamed("context", "Context")}
}

// names is the field spelling of each group, in order.
func names(groups []subject.ParamGroup) []string {
	out := make([]string, 0, len(groups))
	for _, g := range groups {
		out = append(out, g.Name)
	}
	return out
}

// sample is a derived literal.
func sample(text string) golang.Sample { return golang.Sample{Text: `"` + text + `"`} }
