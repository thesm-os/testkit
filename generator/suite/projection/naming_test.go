// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/suite/projection"
)

func TestOptionNamePolicy(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, projection.OptionName("Log", "Append"), projection.Option("WithLogAppend"),
		"the option policy is With<Iface><Method>")
}

// fixtureCase is one fixture-accessor spelling.
type fixtureCase struct {
	name  string
	token string
	field string
	want  projection.Expr
}

func (c fixtureCase) Name() string { return c.name }

func TestEmittedSurfaceNames(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, projection.HarnessName("Store"), "StoreHarness",
		"the per-implementation config literal's type")
	testkit.Equal(t, projection.VeneerName("Store"), "StoreSuite",
		"the exported entry value a consumer reads through")
	testkit.Equal(t, projection.ConfigName("Store"), "StoreConfig",
		"the run-config type")
}

func TestFixtureCallPolicy(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []fixtureCase{
		{"token plus exported field plus parens", "log", "entry", "logEntry()"},
		{"initialisms case the platform's way", "kv", "id", "kvID()"},
		{"an empty field degrades to the bare token call", "kv", "", "kv()"},
	}, func(t *testing.T, tc fixtureCase) {
		testkit.Equal(t, projection.FixtureCall(tc.token, tc.field), tc.want,
			"the fixture accessor spelling has this one home")
	})
}

// The token is what every emitted identifier is qualified by, so it
// has to read as a name rather than as a run-together slug.
//
// Lower camel is the whole rule, and the multiword case is why: a
// plain lower-casing produces kvstorecheckindex, which compiles and
// which nobody wants to meet in a stack trace.
func TestTokenIsLowerCamel(t *testing.T) {
	t.Parallel()

	type tokenCase struct {
		name  string
		iface string
		want  string
	}

	testkit.TableTest(t, []tokenCase{
		{"a one-word interface lowers", "Log", "log"},
		{"a multiword interface stays readable", "KVStore", "kvStore"},
		{"an initialism keeps the platform's casing", "HTTPClient", "httpClient"},
	}, func(t *testing.T, tc tokenCase) {
		testkit.Equal(t, projection.Token(tc.iface), tc.want, tc.name)
	})
}

// The emitted identifiers compose from the token and nothing else.
//
// Pinned because these names are the generated file's whole surface: a
// consumer writes logCheckIndex.Append.Smoke(), and every one of the
// four spellings below has to agree with the others or the file does
// not compile.
func TestEmittedIdentifiersCompose(t *testing.T) {
	t.Parallel()

	tok := projection.Token("Log")

	t.Run("a method's name constant carries the token", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, projection.MethodConst(tok, "Append"), "logAppend",
			"the one home for the method's name")
	})

	t.Run("the index value reads, and its type takes the awkward name", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, projection.IndexVar(tok), "logCheckIndex", "what a consumer writes")
		testkit.Equal(t, projection.IndexType(tok), "logCheckIndexT", "what nobody writes")
	})

	t.Run("both scopes' groups spell one rule", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, projection.GroupType(tok, "Append"), "logAppendChecks",
			"a method group")
		testkit.Equal(t, projection.GroupType(tok, "Model"), "logModelChecks",
			"and a family group, by the same rule rather than the packs' second one")
	})
}
