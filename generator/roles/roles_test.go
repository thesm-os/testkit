// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package roles_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/sdk"
	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/roles"
)

const (
	fixturePkg  = "example.com/kv"
	fixtureFile = "kv/kv.go"
)

func TestAnnotate(t *testing.T) {
	t.Parallel()

	t.Run("stamps the declared role verbatim", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, stamped(t, "key"), "key",
			"the role word travels verbatim; its readers own the vocabulary")
	})

	t.Run("leaves an unroled field alone", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, stamped(t, ""), "",
			"an unroled field draws no pool, and absence must stay readable as absence")
	})

	t.Run("takes the last of two declarations", func(t *testing.T) {
		t.Parallel()
		s := fixture(t, "key", "payload")
		annotate(t, s)
		testkit.Equal(t, read(t, s), "payload", "the last role written wins")
	})
}

// stamped runs the annotator over a field declaring the role and
// returns what landed on its bag.
func stamped(t *testing.T, role string) string {
	t.Helper()
	s := fixture(t, role)
	annotate(t, s)
	return read(t, s)
}

// fixture returns a store whose Request.Key declares each role in
// order; an empty value leaves the field unroled.
func fixture(t *testing.T, values ...string) *sdk.Store {
	t.Helper()
	return storefixture.New().
		Package("kv", fixturePkg).
		Struct("Request", func(b *storefixture.StructBuilder) {
			b.Pos(sdk.At(fixtureFile, 1, 1))
			b.Field("Key", storefixture.Named("string"), func(f *storefixture.FieldBuilder) {
				f.Pos(sdk.At(fixtureFile, 2, 1))
				for _, v := range values {
					if v == "" {
						continue
					}
					f.Directive(storefixture.Directive("role", storefixture.Arg(v)))
				}
			})
		}).
		Build()
}

func annotate(t *testing.T, s *sdk.Store) {
	t.Helper()
	plugintest.Annotate(t, roles.New(), s)
}

// read returns the stamp on Request.Key.
func read(t *testing.T, s *sdk.Store) string {
	t.Helper()
	for _, st := range s.Nodes().Structs().Items() {
		for _, f := range st.Fields {
			if f.Name == "Key" {
				return roles.Of(f.Meta())
			}
		}
	}
	t.Fatal("fixture has no Request.Key")
	return ""
}
