// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"strings"
	"text/template"

	"go.thesmos.sh/testkit/generator/suite/projection"
)

// renderFuncs is the rewrite's template function map, contributed to
// the backend's merged map through the plugin builder's Funcs seam —
// which is what lets the body templates live in templates/golang
// beside the plugin's structural ones: the backend parses every .tmpl
// in the plugin's FS with one merged map, so the functions must be in
// it. Registered bare; the sdk prefixes them under the plugin's name,
// and the templates call the prefixed form (suite_callExpr).
func renderFuncs() template.FuncMap {
	return template.FuncMap{
		"callExpr":      callExpr,
		"borrowedIdent": func() string { return string(projection.ExprBorrowed) },
		"producedIdent": func() string { return string(projection.ExprProduced) },
	}
}

// bodyView is one check body's rendering context: the concrete body
// variant plus the facts the emitted text needs that no projection
// carries — the subject receiver's ident, the check-name constant
// the failure attributes to, and the discard shape of the smoked
// call's returns ("_ =", "_, _ ="), which only the method's
// signature knows. The emitter builds one per check as it walks the
// inventory beside the methods.
type bodyView struct {
	Recv    string
	Check   string
	Discard string
	Body    projection.Body
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
