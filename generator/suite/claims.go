// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"strings"

	"go.thesmos.sh/eidos/lang/golang"
	"go.thesmos.sh/eidos/node"

	"go.thesmos.sh/testkit/generator/suite/projection"
)

// The claim wording policy for the derived families, spelled from the
// corpus manifests. One function per family: a claim is derived text,
// and its one home is here — the corpus's historical wavering between
// "a derived entry" and "derived inputs" for one-parameter methods
// reconciles to the noun form, which says more. The noun is the
// parameter's own identifier, read from the shared signature
// projection.

// The supply adjectives a claim can speak: "derived" for inputs the
// fixture derives from pools, "seeded" on the seed-seam interface
// whose corpus the harness receives.
const (
	supplyDerived = "derived"
	supplySeeded  = "seeded"
)

// SmokeClaim words the smoke family by drawn arity: no draws →
// "survives a call"; one → "survives a call with a derived <noun>";
// several → "survives a call with derived inputs" — "seeded"
// replacing "derived" where the interface's corpus is seeded.
func SmokeClaim(m Method, seeded bool) string {
	supply := supplyDerived
	if seeded {
		supply = supplySeeded
	}
	draws := m.CallArgs()
	switch len(draws) {
	case 0:
		return m.Name + " survives a call"
	case 1:
		return m.Name + " survives a call with a " + supply + " " + drawNoun(draws[0])
	default:
		return m.Name + " survives a call with " + supply + " inputs"
	}
}

// OpenerSmokeClaim words the producing method's smoke: the call
// survives and the handle it opens closes. The produced noun is the
// contract's own name — the vocabulary's one home for what the
// handle is called.
func OpenerSmokeClaim(m Method, produced string) string {
	return m.Name + " survives a call and the " + produced + " it opens closes"
}

// BorrowSmokeClaim words the returning method's smoke: the resource
// it returns was borrowed from the producing sibling. "resource" is
// the corpus's word for the borrowed thing; a second borrowing domain
// argues for deriving it before a rule invents one.
func BorrowSmokeClaim(m Method) string {
	return m.Name + " survives returning a borrowed resource"
}

// CancelClaim words the cancel family.
func CancelClaim(m Method) string {
	return m.Name + " reports a cancelled context as cancelled"
}

// DeadlineClaim words the deadline family.
func DeadlineClaim(m Method) string {
	return m.Name + " reports an expired deadline as exceeded"
}

// NilCtxClaim words the nilcontext family.
func NilCtxClaim(m Method) string {
	return m.Name + " returns an error rather than panicking on a nil context"
}

// ZeroOnErrorClaim words the zero family by the first value result's
// shape: "a nil channel" for channels, "the zero <Name>" for named
// types, "zero" for builtins — and for the synthesized signatures unit
// tests build without a source type, since a shape nobody declared has
// no name to speak. Empty for a method with no value result, which the
// deriver gates on before wording anything.
func ZeroOnErrorClaim(m Method) string {
	values := m.ValueReturns()
	if len(values) == 0 {
		return ""
	}
	if len(values) > 1 {
		// The whole answer, because that is what the body judges. A
		// claim naming one slot of two is the understatement a subject
		// leaking its metadata slips through.
		return m.Name + " returns " + zeroNouns(values) + " alongside any error"
	}
	src := values[0].Source
	switch {
	case src != nil && golang.IsChannel(src):
		// The prose names the channel where the comparison only knows
		// it is nil-compared: a claim is read by someone deciding
		// whether it holds, and "nothing" would not tell them what.
		return m.Name + " returns a nil channel alongside any error"
	case ZeroShapeOf(m) == ZeroNil:
		return m.Name + " returns nothing alongside any error"
	case src != nil && src.Name != "" && !golang.IsPredeclared(src.Name):
		// Predeclared rather than IsBuiltin: the frontend records an
		// in-package named type with no package, exactly like "int",
		// and "the zero Value" needs the two told apart.
		return m.Name + " returns the zero " + src.Name + " alongside any error"
	default:
		return m.Name + " returns zero alongside any error"
	}
}

// ZeroShape is how a body compares a result against its zero.
type ZeroShape int

// Two shapes, and the split is comparability rather than spelling: a
// slice, map or func may only be compared against nil — `got != zero`
// does not compile for one — while every other type has a zero that can
// be declared and compared, predeclared types included.
const (
	// ZeroDeclared declares a zero of the result's own type and
	// compares against it: `var zero kv.Value`, then `got != zero`.
	// Right for named types, strings, bools and numbers alike.
	ZeroDeclared ZeroShape = iota

	// ZeroNil compares against nil, for the kinds that admit nothing
	// else.
	ZeroNil
)

// ZeroShapeOf classifies a method's first result.
//
// One home because the claim and the body it words are the same
// judgment about the same type: a claim promising "the zero Value"
// beside a body comparing against nil is the drift a single inventory
// exists to prevent.
func ZeroShapeOf(m Method) ZeroShape {
	values := m.ValueReturns()
	if len(values) == 0 {
		return ZeroDeclared
	}
	src := values[0].Source
	if src == nil {
		return ZeroDeclared
	}
	switch src.TypeKind {
	case node.TypeRefSlice, node.TypeRefMap, node.TypeRefFunc, node.TypeRefPointer:
		// A pointer is comparable, so a declared zero would work — but
		// it has no name to declare one of, and nil is what the
		// language calls its zero anyway.
		return ZeroNil
	default:
		if golang.IsChannel(src) {
			// A channel arrives as a named ref with the frontend's own
			// stamp on it, never as a kind of its own.
			return ZeroNil
		}
		return ZeroDeclared
	}
}

// zeroNouns words what a multi-slot answer owes a zero of.
//
// The slots are named where naming them helps — "the zero Value and
// Meta" tells a reader which two results the check compares. Where a
// slot has no type name, or two share one, the names stop
// distinguishing anything and the claim says "every result" instead:
// "the zero int and int" is worse than saying nothing about which.
func zeroNouns(values []golang.Return) string {
	names := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, ret := range values {
		src := ret.Source
		if src == nil || src.Name == "" || seen[src.Name] {
			return "the zero for every result"
		}
		seen[src.Name] = true
		names = append(names, src.Name)
	}
	last := len(names) - 1
	return "the zero " + strings.Join(names[:last], ", ") + " and " + names[last]
}

// IdempotentClaim words the second-call claim.
func IdempotentClaim(m Method) string {
	return "a second " + m.Name + " after a clean one changes nothing"
}

// AccumulatesClaim words the declared-not-idempotent claim; noun is
// the declaration's repeated-action word, "call" when it names none.
//
// The wording is corpus-pinned and ready; no rule derives it yet,
// because the declaration that would license it (`idempotent=false`)
// cannot stamp under eidos's mixin grammar — the ruling is owed
// upstream.
func AccumulatesClaim(m Method, noun string) string {
	if noun == "" {
		noun = "call"
	}
	claim := "two " + noun + "s"
	if draws := m.CallArgs(); len(draws) > 0 {
		claim += " of one " + drawNoun(draws[0])
	}
	return claim + " total two, because " + m.Name + " is declared not idempotent"
}

// MissClaim words the reader miss by its answer shape and supply verb:
// the sentinel form where the declaration names one, the zero form
// otherwise. A qualified sentinel ("kv.ErrNotFound") speaks its bare
// name — claims read as prose, and the qualifier is the generated
// code's concern.
func MissClaim(m Method, sentinel, verb string) string {
	noun := missNoun(m)
	if sentinel == "" {
		return m.Name + " reports zero for a " + noun + " nothing has " + verb
	}
	if i := strings.LastIndex(sentinel, "."); i >= 0 {
		sentinel = sentinel[i+1:]
	}
	return m.Name + " reports " + sentinel + " for a " + noun + " nothing " + verb
}

// HitClaim words the seeded hit.
func HitClaim(m Method) string {
	return m.Name + " returns the seeded value for every seeded " + missNoun(m)
}

// CountClaim words the seeded aggregator.
func CountClaim(m Method) string {
	return m.Name + " equals the number of seeded entries"
}

// The leg-level wording policy. Per-law claims live beside their
// identifiers in [lawid]; the two sentences here describe LEGS — the
// linearize run and the observational bundle — which have segments
// rather than law identifiers, so their wording is the suite's.

// LinearizableClaim words the concurrent leg's row.
func LinearizableClaim() string {
	return "concurrent operation histories are linearizable"
}

// BundleClaim words the observational bundle: the chain-shaped
// protocol speaks "chain law", everything else the plain form. The
// corpus's one domain-specific bundle wording (pool's accounting
// sentence) is a spelling this derivation cannot reach — recorded in
// the design doc's frontier.
func BundleClaim(chain bool) string {
	if chain {
		return "every bound chain law holds over random operation sequences"
	}
	return "every bound law holds over random operation sequences"
}

// The sequence nouns the differential wordings speak where no
// protocol names its own pair.
const (
	seqOperation = "operation"
	seqRead      = "read"
)

// DifferentialAgreeClaim words the reference-comparison row: the
// sequence noun, the subject's token, and the reference's arms — a
// reference seeded identically for read-only surfaces, agreement on
// every outcome where the oracle speaks error semantics.
func DifferentialAgreeClaim(sequence, token string, seeded, outcomes bool) string {
	reference := "the reference"
	if seeded {
		reference = "a reference seeded identically"
	}
	claim := "every " + sequence + " sequence leaves the " + token + " agreeing with " + reference
	if outcomes {
		claim += " on every outcome"
	}
	return claim
}

// DifferentialDrainClaim words the produced-handle comparison: the
// drain answers the same entries, in order.
func DifferentialDrainClaim(sequence string) string {
	return "every " + sequence + " sequence drains the same entries as the reference, in order"
}

// missNoun is the word the reader claims call their probed input.
func missNoun(m Method) string {
	if draws := m.CallArgs(); len(draws) > 0 {
		return drawNoun(draws[0])
	}
	return "input"
}

// drawNoun is the word a claim calls one drawn parameter, lower-cased —
// `Lookup(ctx, id Key)` draws "a seeded key", per the corpus.
//
// The word itself is [projection.DrawWord], which the fixture field is
// also spelled from: a claim naming one thing and a field holding
// another is the drift a single inventory exists to prevent. The
// composite-request form ("derived inputs" for a one-struct draw)
// needs the fixture's composed-field fact and arrives with the emitter
// wiring.
func drawNoun(p golang.Param) string {
	return strings.ToLower(projection.DrawWord(p))
}
