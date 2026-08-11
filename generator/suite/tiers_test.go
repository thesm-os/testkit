// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"slices"
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/suite"
)

// ADR-0018 assigns each classification to a tier by a rule, and a rule needs a
// mapping to be mechanical rather than an opinion.
//
// Held to the live registries rather than to a checked-in count: a
// classification added or renamed upstream fails here, which is how the header
// learns the vocabulary grew. Without it a new name would render as covered by
// nobody, which reads as a decision and is an omission.
func TestOwnershipIsComplete(t *testing.T) {
	t.Parallel()

	t.Run("assigns every registered classification", func(t *testing.T) {
		t.Parallel()
		for _, d := range detectors.All() {
			_, known := suite.OwnershipOf(d.Name)
			testkit.True(t, known, "detector "+d.Name+" has a tier")
		}
		for _, m := range mixins.All() {
			_, known := suite.OwnershipOf(m.Name)
			testkit.True(t, known, "mixin "+m.Name+" has a tier")
		}
		for _, c := range contracts.All() {
			_, known := suite.OwnershipOf(c.Name)
			testkit.True(t, known, "contract "+c.Name+" has a tier")
		}
	})

	t.Run("names a law for every model-tier classification and none otherwise", func(t *testing.T) {
		t.Parallel()
		// The rule is "where a law exists the classification is the model
		// tier's", so an entry claiming that tier without naming one is an
		// assignment nothing supports — and one naming a law while claiming
		// this tier says the property is implemented twice.
		for _, name := range registered(t) {
			o, _ := suite.OwnershipOf(name)
			switch o.Tier {
			case suite.TierModel:
				testkit.Assert(t, o.Law).HasPrefix("AUTO-", name+" names the law that carries it")
			case suite.TierSuite, suite.TierNone:
				testkit.Equal(t, o.Law, "", name+" is not the model tier's, so it names no law")
			}
		}
	})

	t.Run("knows nothing the registries do not", func(t *testing.T) {
		t.Parallel()
		// The other direction. An entry for a classification eidos dropped is
		// a header line nothing can produce, and a name nobody notices is
		// stale until somebody reads the table against the registry by hand.
		for _, name := range registered(t) {
			_, known := suite.OwnershipOf(name)
			testkit.True(t, known, name+" is registered and assigned")
		}
		testkit.Equal(t, suite.OwnershipNames(), registered(t),
			"the table and the registries name the same set")
	})
}

// registered returns every classification eidos ships, deduplicated and sorted.
//
// Deduplicated because `pure` is both a detector and a mixin: it is one
// property either way, so it takes one entry.
func registered(t *testing.T) []string {
	t.Helper()
	seen := map[string]struct{}{}
	for _, d := range detectors.All() {
		seen[d.Name] = struct{}{}
	}
	for _, m := range mixins.All() {
		seen[m.Name] = struct{}{}
	}
	for _, c := range contracts.All() {
		seen[c.Name] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

// The header lists what the file does not check, which is what a consumer can
// act on — so the list has to be right about which classifications those are.
func TestCoverage(t *testing.T) {
	t.Parallel()

	t.Run("counts a shape check under the shape that earned it", func(t *testing.T) {
		t.Parallel()
		// A detector's check reports under what it asserts rather than under
		// the classification, so a harness that checks a shape would otherwise
		// list it as unchecked and send a consumer to write one that exists.
		c := contractIn(t, missShapeFixture(t, storefixture.Named("string")))
		testkit.False(t, unchecked(c, "readerwithbool"),
			"the miss check is the readerwithbool check")
	})

	t.Run("lists a classification nothing here asserts", func(t *testing.T) {
		t.Parallel()
		c := contractIn(t, mixinFixture(t, "deleteremoves", ""))
		testkit.True(t, unchecked(c, "deleteremoves"),
			"a classification with no check is what the header is for")
	})

	t.Run("names the methods carrying it", func(t *testing.T) {
		t.Parallel()
		c := contractIn(t, mixinFixture(t, "deleteremoves", ""))
		for _, cov := range c.Unchecked() {
			testkit.Assert(t, cov.MethodList()).Contains("Load",
				"so a consumer knows which extension point to write it with")
		}
	})

	t.Run("recognises every detector check by the shape that earned it", func(t *testing.T) {
		t.Parallel()
		// A detector's check reports under what it asserts rather than under
		// the classification, so the two are tied together in one map — and a
		// kind added without an entry produces a header telling a consumer to
		// write a check that already exists. Which is what shipped for
		// `batchreader` until this asserted it.
		for _, kind := range suite.CheckKinds() {
			if !suite.DetectorCheck(kind) {
				continue
			}
			testkit.True(t, len(suite.ShapesCheckedBy(kind)) > 0,
				string(kind)+" is claimed by the shape that earns it")
		}
	})

	t.Run("keeps a classification the table does not know", func(t *testing.T) {
		t.Parallel()
		// Unreachable from a real run, which TestOwnershipIsComplete is what
		// makes true. Kept rather than dropped: a name nothing recognises is
		// exactly what a reader needs to see.
		c := contractIn(t, mixinFixture(t, "invented-upstream", ""))
		testkit.True(t, unchecked(c, "invented-upstream"),
			"an unknown classification is listed rather than swallowed")
	})

	t.Run("splits what a consumer should write from what they should not", func(t *testing.T) {
		t.Parallel()
		// Conflating the two produced a header that told consumers to
		// hand-write `deleteremoves`, which needs a reference implementation
		// nobody running a suite has. The advice differs, so the lists do.
		c := contractIn(t, mixinFixture(t, "deleteremoves", ""))
		testkit.Len(t, c.Unwritten(), 0, "nothing here is a consumer's to write")
		testkit.True(t, len(c.Elsewhere()) > 0, "and the model-tier one is named as covered elsewhere")

		own := contractIn(t, mixinFixture(t, "scope", ""))
		testkit.True(t, len(own.Unwritten()) > 0,
			"a classification this tier owns and has not written is a consumer's to write")
	})

	t.Run("answers for a method with no source behind it", func(t *testing.T) {
		t.Parallel()
		// A projection built from the emit side carries no declaration, and
		// asking it for a stamp is a nil dereference rather than an empty
		// answer.
		testkit.Equal(t, suite.Method{Sig: &golang.Sig{}}.Shape(), "",
			"a projection with no source is stamped with nothing")
	})
}

// unchecked reports whether the harness lists that classification as one it
// does not assert.
func unchecked(c *suite.Contract, name string) bool {
	for _, cov := range c.Unchecked() {
		if cov.Name == name {
			return true
		}
	}
	return false
}
