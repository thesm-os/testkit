// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"io/fs"
	"path"
	"regexp"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
)

// TestWitnessedPartner pins the nilable half of the guard substitution: an
// absent partner stays absent, a present one lands at the witnesses, and a
// non-generic interface passes its methods through untouched.
func TestWitnessedPartner(t *testing.T) {
	t.Parallel()

	testkit.True(t, witnessedPartner(nil, map[string]sdk.Ref{"V": sdk.Builtin("int")}) == nil,
		"no partner, nothing to rewrite")

	m := Method{Sig: &golang.Sig{
		Name:    "Put",
		Params:  []golang.Param{{Name: "v", Type: sdk.Builtin("V")}},
		Returns: []golang.Return{{Type: sdk.Builtin("error")}},
	}}
	testkit.True(t, witnessedPartner(&m, nil) == &m,
		"a nil binding map is the non-generic passthrough")

	w := witnessedPartner(&m, map[string]sdk.Ref{"V": sdk.Builtin("int")})
	got, isBuiltin := w.Sig.Params[0].Type.(*sdk.BuiltinRef)
	testkit.True(t, isBuiltin && got.Name == "int", "the parameter lands at the witness")
	orig := m.Sig.Params[0].Type.(*sdk.BuiltinRef)
	testkit.Equal(t, orig.Name, "V", "and the shared projection is untouched")
}

// probeMethod and probePartner are the identifiers every phrase is expanded at
// for the collision sweep.
//
// Deliberately unlike each other and unlike any real method name, so a phrase
// that collides does so on its own words rather than on an unlucky pair of
// identifiers — the sweep is about the sentences, and picking `Get` and `GetAll`
// would test Go's naming instead.
const (
	probeMethod  = "Alpha"
	probePartner = "Omega"
)

// interpolated matches the two identifier interpolations a check template uses
// for the method under check and its partner, in any of the whitespace and
// trim-marker forms Go templates admit.
var interpolated = regexp.MustCompile(`\{\{-?\s*\$([mp])\.Name\s*-?\}\}`)

// phraseForm rewrites a template's source into the vocabulary a phrase is
// written in: every identifier interpolation collapses to the token that spells
// it, so the two are comparable as plain strings whatever spacing the template
// used.
//
// This direction rather than expanding the phrase into template text, because
// the template is the thing that varies. `{{ $m.Name }}` and `{{- $m.Name -}}`
// render identically and a phrase can only be written one way, so normalising
// the phrase would make a correct template fail on its trim markers.
func phraseForm(src string) string {
	return interpolated.ReplaceAllStringFunc(src, func(m string) string {
		if strings.Contains(m, "$m.") {
			return becauseMethod
		}
		return becausePartner
	})
}

// checkTemplates reads every check template the plugin ships, keyed by the kind
// it renders.
//
// Walked rather than composed from a directory constant, because the templates
// are filed by where the check comes from — signature, mixin, contract,
// detector — and a test that hard-coded those four would go quietly blind to a
// fifth.
func checkTemplates(t *testing.T) map[sdk.Kind]string {
	t.Helper()

	tree, ok := New().Templates(golang.Language)
	testkit.True(t, ok, "the plugin reports a Go template tree")

	found := map[sdk.Kind]string{}
	err := fs.WalkDir(tree, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := strings.TrimSuffix(path.Base(p), ".tmpl")
		if strings.HasPrefix(name, "suite.check.") {
			found[sdk.Kind(name)] = p
		}
		return nil
	})
	testkit.NoError(t, err, "the template tree walks")

	out := make(map[sdk.Kind]string, len(found))
	for kind, p := range found {
		src, readErr := fs.ReadFile(tree, p)
		testkit.NoError(t, readErr, p+" reads")
		out[kind] = string(src)
	}
	return out
}

// No `because` phrase is a substring of another kind's.
//
// A guard drives one check and holds the rejection to a phrase. Where one
// phrase contains another, the shorter one's guard passes on the longer one's
// rejection — it proved that *something* failed, which every guard in this file
// proves by construction, and nothing about the check it was written for.
//
// This is not hypothetical drift. The table shipped with `"must accept"` under
// three kinds' messages, `"panicked on"` under every panic the suite reports,
// and one phrase spelled identically for three separate zero-value claims.
func TestBecausePhrasesDoNotCollide(t *testing.T) {
	t.Parallel()

	kinds := CheckKinds()
	expanded := make(map[sdk.Kind]string, len(kinds))
	for _, k := range kinds {
		v, known := violators[k]
		testkit.True(t, known, string(k)+" is in the violators table")
		testkit.True(t, v.because != "", string(k)+" holds its rejection to something")
		expanded[k] = expandBecause(v.because, probeMethod, probePartner)
	}

	for _, a := range kinds {
		for _, b := range kinds {
			if a == b {
				continue
			}
			testkit.False(t, strings.Contains(expanded[b], expanded[a]),
				string(a)+" is told apart from "+string(b)+": "+
					expanded[b]+" must not contain "+expanded[a])
		}
	}
}

// Every phrase appears exactly once in the template it claims to quote.
//
// Twice over. Once, that the phrase is the check's own words rather than a
// paraphrase — a guard asserting a sentence the check never reports fails the
// moment it runs, but a guard asserting a sentence some *other* check reports
// passes forever. And once, that the phrase names one assertion line: a
// template stating the same words twice gives its guard two ways to be
// satisfied, and only one of them is the claim.
//
// This is what holds the [violators] docblock to its word. Rewording a check's
// message without moving the phrase now fails here rather than three months
// later, silently, in a guard that stopped proving anything.
func TestBecauseIsQuotedFromItsTemplate(t *testing.T) {
	t.Parallel()

	sources := checkTemplates(t)
	for _, k := range CheckKinds() {
		t.Run(string(k), func(t *testing.T) {
			t.Parallel()

			src, shipped := sources[k]
			testkit.True(t, shipped, string(k)+" ships a check template")

			phrase := violators[k].because
			testkit.Equal(t, strings.Count(phraseForm(src), phrase), 1,
				"the guard's phrase appears once in the check that reports it: "+phrase)
		})
	}
}

// A phrase names a partner only where its check has one to name.
//
// [expandBecause] substitutes an empty string for an absent partner, so a
// phrase naming one under a check that spans nothing would reach the generated
// guard with the token still in it. That fails loudly rather than passing
// wrongly, which is the right direction — but it fails in a consumer's
// generated output, and the fact that decides it is right here in the plugin.
//
// One direction only. A check with a partner whose distinguishing sentence does
// not need to name it is fine and several exist; the unsafe pairing is the
// other one.
func TestBecauseNamesAPartnerOnlyWhereOneExists(t *testing.T) {
	t.Parallel()

	sources := checkTemplates(t)
	for _, k := range CheckKinds() {
		if !strings.Contains(violators[k].because, becausePartner) {
			continue
		}
		testkit.True(t, strings.Contains(sources[k], "$p := .Partner"),
			string(k)+" resolves the partner its phrase names")
	}
}

// The two token forms round-trip, so the sweep above compares like with like.
//
// [phraseForm] is what lets a template written `{{- $m.Name -}}` still match a
// phrase written `$M`. Pinned separately because a regression there would make
// [TestBecauseIsQuotedFromItsTemplate] fail on a correct template, and a guard
// test that cries wolf gets deleted.
func TestPhraseAndTemplateFormsAgree(t *testing.T) {
	t.Parallel()

	testkit.Equal(t, phraseForm("{{ $m.Name }} must accept what {{ $p.Name }} produced"),
		"$M must accept what $P produced", "the spaced form collapses")
	testkit.Equal(t, phraseForm("{{- $m.Name -}} and {{$p.Name}}"), "$M and $P",
		"and so do the trimmed and unspaced ones")
	testkit.Equal(t, phraseForm("{{ .IfaceName }} and {{ $prereq }}"),
		"{{ .IfaceName }} and {{ $prereq }}",
		"and every other interpolation is left where it stands")
	testkit.Equal(t, expandBecause("$M reads $P", "Get", "Put"), "Get reads Put",
		"expansion puts the identifiers in")
}
