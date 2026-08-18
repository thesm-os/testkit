// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

import (
	golang "go.thesmos.sh/eidos/lang/golang"
)

// Expr is a rendered Go expression destined for a template hole. The
// type exists so a template argument cannot be confused with prose,
// and so the well-known spellings below have one home.
type Expr string

// ExprCtx is the context argument every ctx-taking call receives; the
// emitted body names its context ctx, and this constant is the only
// place that decision is spelled.
const ExprCtx Expr = "ctx"

// Option is a generated stub option's name ("WithLogAppend"),
// constructed only through [OptionName] so the naming policy has one
// home.
type Option string

// OptionName spells the per-method construction option the stub
// plugin emits: With<Iface><Method>.
func OptionName(iface, method string) Option {
	return Option("With" + iface + method)
}

// FixtureCall spells the fixture accessor call for a drawn field:
// token + exported field name + call parens ("logEntry()" for
// ("log", "entry")). The casing is the platform's own Go convention —
// initialisms included, so ("kv", "id") is "kvID()" — because a
// second casing implementation would drift from what every other
// plugin emits.
func FixtureCall(token, field string) Expr {
	if field == "" {
		return Expr(token + "()")
	}
	return Expr(token + golang.ExportedName(field) + "()")
}
