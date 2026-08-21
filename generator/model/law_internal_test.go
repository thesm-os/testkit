// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"path"
	"testing"

	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
)

// A binding's kind is the template it renders through, and the backend
// dispatches on it by name.
//
// A kind naming no template is not a render error. The backend finds
// nothing to run for the value and contributes nothing, so the registry
// comes out without that law in it — a suite that got smaller, reported
// as a suite that passed.
func TestLawBindingKindNamesItsTemplate(t *testing.T) {
	t.Parallel()

	var b LawBinding
	testkit.Equal(t, string(b.Kind()), "model.law", "the one template every binding renders through")
	testkit.True(t, lawTemplateExists(t, "model.law"),
		"and it is in the embedded tree the backend dispatches through")
}

// A field carries its own kind, because the field's template is chosen
// by the closure's SHAPE rather than by the field's name.
//
// The catalogue spells Read on a keyed store and Read on a version cell
// with one word, and the two closures could not be less alike — so the
// accessor hands back what the shape resolution decided rather than
// composing a name from the field.
func TestLawFieldKindIsWhatTheShapeDecided(t *testing.T) {
	t.Parallel()

	f := LawField{KindName: sdk.Kind(LawFieldKindPrefix + string(shapeKeyedRead))}
	testkit.Equal(t, string(f.Kind()), "model.lawfield.Read",
		"the kind the shape resolution stamped, passed through unaltered")

	var unset LawField
	testkit.Equal(t, string(unset.Kind()), "",
		"a field nothing resolved names no template, rather than defaulting into one")
}

// The prefix composes the kind, so the two halves have one home.
func TestLawFieldKindPrefixComposes(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, LawFieldKindPrefix+string(shapeHandleCall), "model.lawfield.HandleCall",
		"the template name is the prefix and the shape, spelled nowhere else")
	testkit.HasSuffix(t, LawFieldKindPrefix, ".",
		"the separator belongs to the prefix, so no caller composes one")
}

// The field templates take the runner's *T, so they need its import path
// and read it off the value rather than being told.
func TestLawFieldSurfacesTheRunnerPackage(t *testing.T) {
	t.Parallel()

	var f LawField
	testkit.Equal(t, f.ModelPkg(), ModelPkg,
		"one home for the path, which the closures' parameter type names")
	testkit.Contains(t, ModelPkg, "engine/model",
		"and it is the runner's own package")
}

// lawTemplateExists reports whether a template of that name is in the
// embedded tree.
func lawTemplateExists(t *testing.T, name string) bool {
	t.Helper()
	_, err := goTemplatesFS.ReadFile(path.Join("templates", "golang", "law", name+".tmpl"))
	return err == nil
}
