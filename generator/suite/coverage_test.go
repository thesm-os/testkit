// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"testing"

	"go.thesmos.sh/eidos/eidostest/storefixture"
	"go.thesmos.sh/eidos/lang/golang"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/suite"
)

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
		// A classification no rule reaches and no check asserts reads the
		// same way as one eidos never registered, and that is the point: the
		// header names it either way rather than swallowing a name it does
		// not recognise. Which of the two it is belongs to the conformance
		// gate, where the registries are the reference.
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
