// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/suite"
)

// TestTemplateSurfaces pins the accessors the templates read: each answers a
// constant the render pass composes imports from, and a wrong one is a
// generated file that fails to compile in whichever package arms it.
func TestTemplateSurfaces(t *testing.T) {
	t.Parallel()

	b := &Bindings{}
	testkit.Equal(t, b.ModelPkg(), ModelPkg, "the runner's import path")
	testkit.Equal(t, b.LinearizePkg(), LinearizePkg, "the Porcupine wiring's import path")
	testkit.Equal(t, (&Action{}).ModelPkg(), ModelPkg, "and the actions' own view of it")
}

// TestNeedsFixture pins the fixture obligation: the property constructs the
// derived inputs exactly when something reads them — a pool, a law anchored
// on a fixture key, a per-position pair — because an unused local is a
// compile error in a generated file.
func TestNeedsFixture(t *testing.T) {
	t.Parallel()

	testkit.False(t, (&Bindings{}).NeedsFixture(), "nothing read, nothing built")
	testkit.True(t, (&Bindings{LawsUseFixture: true}).NeedsFixture(),
		"a fixture-anchored law obliges it")
	testkit.True(t, (&Bindings{Actions: []*Action{{Pool: poolKeys}}}).NeedsFixture(),
		"a drawing pool obliges it")
	testkit.True(t, (&Bindings{Actions: []*Action{{Args: []ActionArg{{Field: "A"}}}}}).NeedsFixture(),
		"a per-position pair obliges it")
}

// TestStampedSentinel pins the declared-sentinel resolution: a contract error
// row naming a stamped parameter hands the oracle the declaration's own error
// identity, and every other shape of the stamp falls back to minting.
func TestStampedSentinel(t *testing.T) {
	t.Parallel()

	carrier := &suite.Method{Sig: &golang.Sig{Source: &node.Method{}}}
	carrier.Contracts = []string{"lease"}

	t.Run("an unnamed parameter mints", func(t *testing.T) {
		t.Parallel()
		_, stamped := stampedSentinel(nil, carrier, "lease", "")
		testkit.False(t, stamped, "no parameter, no stamp to read")
	})

	t.Run("an unstamped parameter mints", func(t *testing.T) {
		t.Parallel()
		_, stamped := stampedSentinel(nil, carrier, "lease", "held")
		testkit.False(t, stamped, "the declaration said nothing")
	})

	t.Run("an unqualified stamp mints", func(t *testing.T) {
		t.Parallel()
		bare := &suite.Method{Sig: &golang.Sig{Source: &node.Method{}}}
		bare.Contracts = []string{"lease"}
		shape.ContractParamKey("lease", "held").Set(bare.Source.EnsureMeta(), "ErrHeld", "test")
		_, stamped := stampedSentinel(nil, bare, "lease", "held")
		testkit.False(t, stamped, "a bare name carries no package to import")
	})

	t.Run("a qualified stamp is the oracle's sentinel", func(t *testing.T) {
		t.Parallel()
		host := &suite.Method{Sig: &golang.Sig{Source: &node.Method{}}}
		host.Contracts = []string{"lease"}
		shape.ContractParamKey("lease", "held").
			Set(host.Source.EnsureMeta(), "example.com/lease.ErrHeld", "test")
		sym, stamped := stampedSentinel(nil, host, "lease", "held")
		testkit.True(t, stamped && sym != nil, "one identity for the oracle and the law")
	})
}
