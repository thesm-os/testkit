// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gentest_test

import (
	"testing"

	"go.thesmos.sh/eidos/core/diag"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/internal/gentest"
)

// About is the one piece of this package with a decision in it, and the
// decision matters: a gap assertion that took the whole diagnostic list
// passes when SOME method was refused, which is not what it claims.
func TestAbout(t *testing.T) {
	t.Parallel()

	diags := []diag.Diag{
		{Message: "suite: Get draws no value"},
		{Message: "suite: Put draws no value"},
		{Message: "suite: Delete draws no value"},
	}

	t.Run("narrows to the diagnostics naming the subject", func(t *testing.T) {
		t.Parallel()
		got := gentest.About(diags, "Put")
		testkit.Len(t, got, 1, "one method's gap, not the run's")
		testkit.Contains(t, got[0].Message, "Put", "and it is that method's")
	})

	t.Run("a subject nothing named comes back empty", func(t *testing.T) {
		t.Parallel()
		// Empty rather than nil: a caller ranging over it reads the same
		// either way, and a caller asserting a length gets 0 rather than a
		// panic.
		testkit.Len(t, gentest.About(diags, "Scan"), 0,
			"a method the run did not refuse has no gap to find")
	})

	t.Run("matching is on the message, so a substring counts", func(t *testing.T) {
		t.Parallel()
		// Worth stating out loud: a subject that is a prefix of another
		// method's name matches both, and a test asserting a count would
		// catch that where one asserting non-empty would not.
		testkit.Len(t, gentest.About(diags, "draws no value"), 3,
			"any text the message holds narrows the list")
	})
}

// The fixture scaffolding, which every plugin's tests build through.
func TestFixtureScaffolding(t *testing.T) {
	t.Parallel()

	t.Run("every declaration shares one source file", func(t *testing.T) {
		t.Parallel()
		// Several generators compose the output filename from the source
		// basename, so a fixture with a different position per test makes two
		// goldens differ for a reason that is not the code under test.
		testkit.Equal(t, gentest.At().File, gentest.SourceFile,
			"the file every fixture that does not care declares in")
	})

	t.Run("a fixture that needs its own file gets one", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, gentest.AtFile("other/iface.go").File, "other/iface.go",
			"a cross-package case, or two interfaces a generator must not merge")
	})

	t.Run("the context and error spellings are the same everywhere", func(t *testing.T) {
		t.Parallel()
		// The point of carrying them here: a fixture spelling the context
		// differently would be testing a signature family the generator
		// classifies differently, without saying so.
		s := storefixture.New().
			Package("kv", "example.com/kv").
			Interface("Store", func(b *storefixture.InterfaceBuilder) {
				b.Pos(gentest.At())
				b.Method("Get", func(m *storefixture.MethodBuilder) {
					gentest.Ctx(m)
					gentest.Err(m)
				})
			}).
			Build()

		m := methodOf(t, s, "Get")
		testkit.Len(t, m.Params, 1, "the context is the only parameter")
		testkit.Equal(t, m.Params[0].Name, "ctx", "spelled the one way")
		testkit.Len(t, m.Returns, 1, "and the error is the only result")
	})
}

// methodOf returns the named method of the store's only interface.
func methodOf(t *testing.T, s *sdk.Store, name string) *sdk.Method {
	t.Helper()
	for _, iface := range s.Nodes().Interfaces().Items() {
		for _, m := range iface.Methods {
			if m.Name == name {
				return m
			}
		}
	}
	t.Fatalf("fixture declares no %s", name)
	return nil
}
