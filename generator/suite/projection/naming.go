// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

import (
	"go.thesmos.sh/eidos/core/naming"
	"go.thesmos.sh/eidos/lang/golang"
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

// ExprProduced is the local an opener smoke binds its answered handle
// to before closing it — the body's one name for what the opener
// owns.
const ExprProduced Expr = "produced"

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

// Token is the interface's qualifier in every generated identifier —
// "log" for Log, "kvStore" for KVStore.
//
// Lower camel rather than a plain lower-casing, because the token
// prefixes identifiers a human reads (logAppend, kvStoreCheckIndex)
// and "kvstorecheckindex" is not a name anybody wants in a stack
// trace. The casing engine is the platform's own, initialisms
// included, so the token agrees with what every other plugin emits.
func Token(iface string) string { return naming.Camel(iface) }

// MethodConst is the generated constant holding a method's name —
// `logAppend = "Append"`.
//
// One home per name: the index accessors, the check bodies and the
// failure messages all spell the method, and a literal repeated across
// three emitted sections is three chances to rename two of them.
func MethodConst(token, method string) string {
	return token + golang.ExportedName(method)
}

// IndexVar is the generated index value a consumer reaches through —
// `logCheckIndex`.
func IndexVar(token string) string { return token + indexSuffix }

// IndexType is the index value's type — `logCheckIndexT`.
//
// The suffix exists because the value carries the readable name: a
// consumer writes `logCheckIndex.Append.Smoke()` and never writes the
// type, so the type takes the awkward one.
func IndexType(token string) string { return IndexVar(token) + indexTypeSuffix }

// GroupType is one index member's type — `logAppendChecks` for a
// method group, `logModelChecks` for a family's.
//
// Uniform across both scopes, which is a decision the hand-written
// packs did not have to make: they spell the family group's type
// `<token>ModelIndex` and the method groups' `<token><Method>Checks`.
// A generator emits one rule, and the group is a group of checks
// whichever scope named it.
func GroupType(token, field string) string {
	return token + golang.ExportedName(field) + checksSuffix
}

// The emitted index's fixed words.
const (
	indexSuffix     = "CheckIndex"
	indexTypeSuffix = "T"
	checksSuffix    = "Checks"
)
