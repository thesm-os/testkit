// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"embed"
	"strings"
	"text/template"

	"go.thesmos.sh/testkit/generator/suite/projection"
)

// genTemplatesFS is the rewrite's template tree, embedded apart from
// the incumbent's: two parsers with two function maps must never see
// each other's files, or a function one map lacks fails the other's
// parse.
//
//go:embed templates/gen
var genTemplatesFS embed.FS

// bodyView is one check body's rendering context: the concrete body
// variant plus the facts the emitted text needs that no projection
// carries — the subject receiver's ident, the check-name constant
// the failure attributes to, and the discard shape of the smoked
// call's returns ("_ =", "_, _ ="), which only the method's
// signature knows.
type bodyView struct {
	Recv    string
	Check   string
	Discard string
	Body    projection.Body
}

// bodyTemplates parses the body tree with the rendering functions.
// Parsed per call: generation-time code on a build-time budget, and a
// cache is the emitter's concern the day a profile asks for one.
func bodyTemplates() (*template.Template, error) {
	return template.New("gen").Funcs(template.FuncMap{
		"callExpr":      callExpr,
		"borrowedIdent": func() string { return string(projection.ExprBorrowed) },
		"producedIdent": func() string { return string(projection.ExprProduced) },
	}).ParseFS(genTemplatesFS, "templates/gen/body/*.tmpl")
}

// renderBody executes one body variant's template — the dispatch is
// the variant's own kind, which IS the template's name, so an
// unregistered variant fails by name rather than rendering nothing.
func renderBody(v bodyView) (string, error) {
	t, err := bodyTemplates()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := t.ExecuteTemplate(&b, string(v.Body.BodyKind()), v); err != nil {
		return "", err
	}
	return b.String(), nil
}

// callExpr spells one invocation on the subject receiver, the args
// already rendered by the derivation.
func callExpr(recv string, c projection.CallPlan) string {
	args := make([]string, len(c.Args))
	for i, a := range c.Args {
		args[i] = string(a)
	}
	return recv + "." + c.Method + "(" + strings.Join(args, ", ") + ")"
}
