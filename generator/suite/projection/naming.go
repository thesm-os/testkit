// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

import (
	"go.thesmos.sh/eidos/core/naming"
	"go.thesmos.sh/eidos/lang/golang"

	"go.thesmos.sh/testkit/engine/suite"
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

// AssertName is the generated assertion's identifier —
// `storeAssertGetHonoursDeadline`.
//
// The word for the segment is the assertion's, not the index's, and the
// two genuinely differ: the index reads as a noun a consumer names
// (`ix.Put.Deadline()`) while the assertion reads as the sentence it
// checks (`…HonoursDeadline`). Transcribed from the packs, which spell
// every one of them.
func AssertName(token, method, seg string) string {
	return token + assertInfix + golang.ExportedName(method) + assertWord(seg)
}

// assertWord is the segment as an assertion reads it, falling back to
// the index's word where the packs never spelled one — a segment with
// no assertion in any pack has no validated sentence, and the index's
// noun is the honest stand-in until one exists.
func assertWord(seg string) string {
	if w, ok := assertWords()[seg]; ok {
		return w
	}
	name, _ := segAccessor(seg)
	return name
}

// assertWords are the segments whose assertion reads differently from
// their index entry.
func assertWords() map[string]string {
	return map[string]string{
		suite.SegDeadline:   "HonoursDeadline",
		suite.SegNilContext: "ToleratesNilContext",
		suite.SegZeroValue:  "ZeroOnError",
	}
}

// assertInfix separates the subject from what is asserted about it.
const assertInfix = "Assert"

// DrawWord is the word a drawn parameter is known by: the named type's
// own word where the source declares one, and the parameter's
// identifier otherwise.
//
// The type rather than the parameter, because a fixture holds one value
// per thing drawn and the thing is the type: `Put(ctx, v Value)` and
// `Get(ctx, key Key)` draw a Value and a Key, whichever letters the
// author happened to name the parameters. A predeclared type says
// nothing — every `string` would collide with every other — so there
// the parameter's own identifier is the only word available.
//
// One home because the claim text and the fixture field are the same
// word cased differently: "a seeded key" and Key. They were derived
// separately, and only the claim side had the rule.
func DrawWord(p golang.Param) string {
	if p.Source != nil && p.Source.Name != "" && !golang.IsPredeclared(p.Source.Name) {
		return p.Source.Name
	}
	return p.Name
}

// DrawField is [DrawWord] as the fixture's exported field.
func DrawField(p golang.Param) string { return golang.ExportedName(DrawWord(p)) }

// ExprFixture is the local a body reads its draws through. The
// generated assert function takes the fixture by this name wherever any
// of its calls draws, and never where none does.
const ExprFixture Expr = "fx"

// FixtureCall spells one drawn field as the body reads it — `fx.Value`
// for ("fx", "value").
//
// Through the fixture rather than a package-level accessor, because the
// value has to be the RUN's: a consumer replacing the fixture through
// WithFixture must reach every check that draws, and a package function
// returning a literal reaches none of them. The packs spell this
// `fx.Key()` with parens because their accessors compute from the
// config's pools; ours are fields until those pools are emitted, and
// the parens arrive with them.
func FixtureCall(recv Expr, field string) Expr {
	if field == "" {
		return recv
	}
	return recv + "." + Expr(golang.ExportedName(field))
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

// IDQualifier is the interface's word inside a family-scoped check ID —
// "log", "paginated-reader".
//
// A slug rather than [Token], because the two qualify different things
// and the ID grammar admits only a-z, 0-9 and '-'. Token names Go
// declarations and reads as lower camel; an ID is what a lock file row
// and a Without() call are written against, and "paginatedReader"
// there is refused by the grammar rather than merely ugly. Every
// validated pack has a single-word interface, which is why one word
// served both jobs until the corpus asked.
func IDQualifier(iface string) string { return naming.Kebab(iface) }

// MethodConst is the generated constant holding a method's name —
// `logAppend = "Append"`.
//
// One home per name: the index accessors, the check bodies and the
// failure messages all spell the method, and a literal repeated across
// three emitted sections is three chances to rename two of them.
func MethodConst(token, method string) string {
	return token + golang.ExportedName(method)
}

// The run surface's identifiers, composed here rather than in the
// template so the half-dozen names that have to agree with each other
// agree by construction: a veneer naming an index its own file does not
// declare is a compile error a consumer meets, not one a run does.
// [HarnessName] and [VeneerName] are the two a consumer writes; these
// are the machinery those hang off.

// DefaultConfigName is the constructor for what this run derived.
func DefaultConfigName(token string) string { return token + defaultConfigSuffix }

// defaultConfigSuffix names the derived config apart from the type a
// consumer declares one of.
const defaultConfigSuffix = "DefaultConfig"

// ChecksName is the builder holding every check this run derived.
func ChecksName(token string) string { return token + checksBuilderSuffix }

// checksBuilderSuffix follows the packs' majority spelling, which names
// the tier the checks come from rather than the family — one builder
// per tier, paired with the model tier's own.
const checksBuilderSuffix = "SignatureChecks"

// RunName is the entry point a consumer calls to run the suite.
func RunName(iface string) string { return runPrefix + iface }

// ProveName is the entry point that runs a check set against a
// deliberately broken subject.
func ProveName(iface string) string { return provePrefix + iface }

// RunOptName is the interface every run option satisfies.
func RunOptName(iface string) string { return iface + runOptSuffix }

// RunConfigName is what the run options accumulate into.
func RunConfigName(token string) string { return token + runConfigSuffix }

// DropOptName is the option that declines checks by identity.
func DropOptName(token string) string { return token + dropOptSuffix }

// WithoutName is the constructor a consumer calls to decline them.
func WithoutName(token string) string { return token + withoutSuffix }

// VeneerTypeName is the veneer's type; [VeneerName] is the value a
// consumer reads it through.
func VeneerTypeName(token string) string { return token + veneerTypeSuffix }

// IndexPathName maps every emitted ID to the path that drops it.
func IndexPathName(token string) string { return token + indexPathSuffix }

// DropHintName is the reporter that turns a dropped ID into that path.
func DropHintName(token string) string { return token + dropHintSuffix }

// The run surface's fixed words.
const (
	runPrefix        = "Run"
	provePrefix      = "Prove"
	runOptSuffix     = "RunOpt"
	runConfigSuffix  = "RunConfig"
	dropOptSuffix    = "DropOpt"
	withoutSuffix    = "Without"
	veneerTypeSuffix = "Veneer"
	indexPathSuffix  = "IndexPath"
	dropHintSuffix   = "DropHint"
)

// QualifierConst is the generated constant holding the interface's word
// inside a family-scoped ID — `logQualifier = "log"`.
//
// A constant for the same reason the method names are: the qualifier is
// spelled once per family accessor, and a literal repeated per accessor
// is a rename that compiles after changing some of them. The hand
// written packs spell it inline; this is the rule they were written
// before.
func QualifierConst(token string) string { return token + qualifierSuffix }

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
	qualifierSuffix = "Qualifier"
	indexSuffix     = "CheckIndex"
	indexTypeSuffix = "T"
	checksSuffix    = "Checks"
)
