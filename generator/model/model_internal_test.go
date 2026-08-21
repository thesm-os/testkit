// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"go.thesmos.sh/testkit"
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

// TestTemplateImportAccessors pins the paths the templates qualify
// through: each is a constant the emit layer registers as an import.
func TestTemplateImportAccessors(t *testing.T) {
	t.Parallel()
	b := &Bindings{}
	testkit.True(t, b.TracePath() != "" && b.LawPath() != "",
		"the trace and law packages are spelled for the classifier and the doors")
}
