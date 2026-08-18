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

// ExprBorrowed is the local a borrow-first smoke binds the producing
// sibling's answer to; the returning call's args reference it where
// the parameter takes the produced type.
const ExprBorrowed Expr = "borrowed"

// The emitted-surface suffixes, composed only through the policy
// functions below so each generated identifier's spelling has one
// home.
const (
	harnessSuffix = "Harness"
	veneerSuffix  = "Suite"
	configSuffix  = "Config"
)

// HarnessName is the generated harness type's identifier — the config
// literal a consumer writes per implementation.
func HarnessName(iface string) string { return iface + harnessSuffix }

// VeneerName is the generated veneer's identifier — the exported
// entry value a consumer's test file reads checks and runs through.
func VeneerName(iface string) string { return iface + veneerSuffix }

// ConfigName is the generated run-config type's identifier.
func ConfigName(iface string) string { return iface + configSuffix }

// poolSuffix trails every drawn-pool config field; the config doc
// promises "fields ending in Pool are drawn from", and this is where
// that promise is spelled.
const poolSuffix = "Pool"

// PoolFieldName is a role's config field, derived from the stamped
// field's own exported name — kv's Key and Value fields open KeyPool
// and ValuePool.
func PoolFieldName(field string) string { return golang.ExportedName(field) + poolSuffix }

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
