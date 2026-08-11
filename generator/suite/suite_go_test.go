// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"io/fs"
	"testing"

	"go.thesmos.sh/eidos/lang/golang"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/suite"
)

// Layout composes every filename from a declared suffix, and reads the language
// off the backend — so a plugin keyed on anything but eidos's own constant
// answers nothing for every real run, and validates an empty set on the way
// past without reporting it.
func TestGoOutputs(t *testing.T) {
	t.Parallel()

	t.Run("declares the harness and its companion", func(t *testing.T) {
		t.Parallel()
		testkit.Len(t, suite.GoOutputs(), 2, "the harness and the file that proves it")
	})

	t.Run("puts the primary at index zero", func(t *testing.T) {
		t.Parallel()
		// The empty tag is the primary, and the pipeline validates the position
		// at Build — pinned here so the reason survives a reordering.
		testkit.Equal(t, suite.GoOutputs()[0].Tag, "", "the empty tag leads")
	})

	t.Run("gives the companion the test suffix Layout keys on", func(t *testing.T) {
		t.Parallel()
		// A suffix ending `_test.go` earns the external-test-package shift,
		// which is what makes the companion reach the harness across a package
		// boundary the way a consumer does.
		testkit.Assert(t, suite.GoTestSuffix).HasSuffix("_test.go",
			"the shift is keyed on the suffix, not on the tag")
	})

	t.Run("answers nothing for another language", func(t *testing.T) {
		t.Parallel()
		// Returning nil is what makes Layout report a missing provider rather
		// than compose Go-shaped filenames for a backend that is not Go.
		testkit.Assert(t, suite.New().Outputs("rust")).IsEmpty("only Go is served")
	})

	t.Run("hands back a slice the caller may keep", func(t *testing.T) {
		t.Parallel()
		// The pipeline holds what this returns for the whole run; a caller
		// mutating it would rewrite the plugin's routing.
		got := suite.New().Outputs(golang.Language)
		got[0].Suffix = "mutated"
		testkit.Equal(t, suite.New().Outputs(golang.Language)[0].Suffix, suite.GoPrimarySuffix,
			"the accessor clones")
	})
}

// A template the backend cannot find renders nothing and fails nowhere, so the
// file simply comes out short.
func TestGoTemplates(t *testing.T) {
	t.Parallel()

	t.Run("ships the harness template", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, hasTemplate(t, "suite.contract.tmpl"), "the harness template must ship")
	})

	t.Run("ships a template per check kind", func(t *testing.T) {
		t.Parallel()
		// One file per check is what keeps seventy-two of them readable, and
		// the nesting is only reachable because the loader walks depth-first —
		// a `*.tmpl` embed pattern would reach the top level alone.
		for _, name := range []string{
			"signature/suite.check.smoke.tmpl",
			"signature/suite.check.cancel.tmpl",
			"signature/suite.check.deadline.tmpl",
			"signature/suite.check.nilcontext.tmpl",
			"signature/suite.check.zeroonerror.tmpl",
			"detector/suite.check.misszero.tmpl",
			"detector/suite.check.missflag.tmpl",
			"mixin/suite.check.nilsafe.tmpl",
			"mixin/suite.check.timeout.tmpl",
			"mixin/suite.check.orderafter.tmpl",
			"mixin/suite.check.sideeffect.tmpl",
		} {
			testkit.True(t, hasTemplate(t, name), name+" must ship")
		}
	})

	t.Run("reaches a nested directory at all", func(t *testing.T) {
		t.Parallel()
		// The claim the embed pattern rests on. If `//go:embed templates/golang`
		// were narrowed back to a glob, every check template would vanish and
		// the harness would render with no checks in it.
		testkit.True(t, len(templateNames(t)) > 5,
			"the tree carries more than its top level")
	})

	t.Run("answers nothing for another language", func(t *testing.T) {
		t.Parallel()
		_, ok := suite.New().Templates("rust")
		testkit.False(t, ok, "only Go is served")
	})
}

// hasTemplate reports whether the named template ships in the plugin's Go tree.
//
// Looked up by the path relative to the tree root, which is how the backend
// registers them: sdk/golang subs the FS to templates/golang, so a name is the
// path below that.
func hasTemplate(t *testing.T, name string) bool {
	t.Helper()
	tree, ok := suite.New().Templates(golang.Language)
	if !ok {
		t.Fatal("the plugin reports no Go template tree")
	}
	_, err := fs.Stat(tree, name)
	return err == nil
}

// templateNames returns every template the tree carries, at any depth.
func templateNames(t *testing.T) []string {
	t.Helper()
	tree, ok := suite.New().Templates(golang.Language)
	if !ok {
		t.Fatal("the plugin reports no Go template tree")
	}
	var out []string
	if err := fs.WalkDir(tree, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			out = append(out, p)
		}
		return nil
	}); err != nil {
		t.Fatalf("walking the template tree: %v", err)
	}
	return out
}
