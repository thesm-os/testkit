// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/plugintest"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/model"
)

// TestConformance holds the plugin to the pipeline's own contract.
func TestConformance(t *testing.T) {
	t.Parallel()
	plugintest.RunSuite(t, model.New())
}

// field is one struct field of a kvStore fixture.
type field struct{ name, typ string }

// TestBindings walks the derivation for the corpus's own fixture shape: a
// stamped writer carrying the validates mixin, its partner, and a reader.
func TestBindings(t *testing.T) {
	t.Parallel()

	b := bindingsOf(t, mixed(t))

	t.Run("drives the reader and the writer", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, len(b.Actions), 2, "two methods map to actions")
		testkit.Equal(t, b.Actions[0].Method, "Store", "in declaration order")
		testkit.Equal(t, b.Actions[0].KindName, sdk.Kind("model.action.writer"),
			"the writer renders through its shape's template")
		testkit.Equal(t, b.Actions[1].Method, "Read", "the reader follows")
	})

	t.Run("excludes the partner with its reason", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, len(b.Skipped), 1, "one method is not driven")
		testkit.Equal(t, b.Skipped[0].Method, "Validate",
			"the method the mixin references")
		testkit.Assert(t, b.Skipped[0].Reason).Contains("validates.fn",
			"naming the stamp that claims it")
	})

	t.Run("derives the map reference", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, b.Reference.Supplied(), "nothing was named, so it is derived")
		testkit.Equal(t, b.Reference.KeyField, "Key",
			"keyed on the one string field matching the reader's key")
		testkit.Equal(t, len(b.Adapter), 3, "every method has an adapter body")
		testkit.Equal(t, b.Adapter[0].Op, "Put", "the writer delegates")
		testkit.Equal(t, b.Adapter[2].Op, "Get", "the reader delegates")
		testkit.True(t, b.Adapter[1].Op == "" && b.Adapter[1].Reason != "",
			"the partner is inert, with why")
	})

	t.Run("draws from the fixture pools", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, b.Keys.Field, "Key", "keys are the reader's fixture pair")
		testkit.Equal(t, b.Keys.OtherField, "KeyOther", "with the companion beside it")
		testkit.Equal(t, b.Values.Field, "Payload", "values are the writer's")
	})

	t.Run("the validates claim narrows the values", func(t *testing.T) {
		t.Parallel()
		testkit.False(t, b.Values.Wide, "a validating subject may refuse a raw draw")
		testkit.Assert(t, b.Values.WhyNarrow).Contains("validates claim on Store",
			"the header names the claim and its carrier")
		testkit.Equal(t, b.Values.Pin, "",
			"and even a recombined fixture body is a value nothing proved accepted")
	})
}

// TestRenderSurface hits what only the templates otherwise read, so a rename
// that breaks the template's field lookup fails here with a name rather than
// in the backend with a short file.
func TestRenderSurface(t *testing.T) {
	t.Parallel()

	b := bindingsOf(t, mixed(t))
	testkit.Equal(t, b.Kind(), model.KindBindings, "the bindings render as themselves")
	testkit.Equal(t, b.Actions[0].Kind(), b.Actions[0].KindName,
		"an action renders through its shape's template")
	testkit.Equal(t, b.ModelPkg(), model.ModelPkg, "the runner's import path")
	testkit.Equal(t, b.RefPkg(), model.RefPkg, "the oracle's import path")
	testkit.Equal(t, b.TierName(), model.TierName, "the path the run reports under")
	testkit.True(t, b.UsesKeys() && b.UsesValues(),
		"a reader and a writer draw from both pools")

	t.Run("the miss prefix follows the routed package", func(t *testing.T) {
		t.Parallel()
		testkit.Equal(t, b.MissPrefix(), "mixed",
			"provisional before Layout: the interface's own spelling")
		b := bindingsOf(t, mixed(t))
		b.SetOutputPackages(map[string]string{"": "example.com/validates/validatestest"})
		testkit.Equal(t, b.MissPrefix(), "validatestest",
			"and the routed package once Layout resolved it")
		b.SetOutputPackages(map[string]string{})
		testkit.Equal(t, b.MissPrefix(), "validatestest",
			"a partial map on a later call clears nothing")
	})
}
