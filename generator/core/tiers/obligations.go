// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers

// Obligation is one claim a classification carries, named by what the
// claim NEEDS rather than by which classification carries it.
//
// The unit of tier ownership, per docs/adr/0028. A classification is a
// bundle of claims and the claims differ in what they need: `ttl` asks
// both "a lapsed read answers the declared sentinel", which one call
// settles, and "an entry stops being readable once the clock passes its
// lifetime", which needs a clock the run advances. Owning the whole
// classification means picking one and dropping the other.
//
// Named per kind rather than per classification because the kinds
// repeat: `sideeffect`, `orderafter`, `validates`, `readafterwrite` and
// `deleteremoves` all carry the named-pair obligation under different
// member names, and one parameterised body serves them all.
type Obligation string

// The suite tier's whole vocabulary. Anything a claim needs that is not
// on this list belongs to another tier.
//
// The test is uniform: what does this claim need that a CALLER does not
// have? Nothing, and it is the suite tier's — a fixed call sequence on
// one subject, from derived inputs.
const (
	// ObSurvives is that the call does not panic.
	ObSurvives Obligation = "survives"

	// ObContext is that a cancelled, expired or nil context is reported
	// rather than ignored.
	ObContext Obligation = "honours the context"

	// ObZeroes is that every value slot is the zero when the error is
	// non-nil.
	ObZeroes Obligation = "zeroes beside an error"

	// ObDeclaredAnswer is that where a directive names a sentinel, a
	// bound or a value, the subject gives it. The directive's parameter
	// is what a caller does have, which is why this is the suite's.
	ObDeclaredAnswer Obligation = "the declared answer"

	// ObNamedPair is that where a directive names a partner member, a
	// fixed sequence over the two holds the stated relation.
	ObNamedPair Obligation = "the named pair"

	// ObSelfAgreement is that two calls with nothing observed between
	// them answer alike.
	ObSelfAgreement Obligation = "self-agreement"
)

// The model tier's. Each names a thing the run has to build that a
// caller writing a test by hand does not.
const (
	// ObUniversal is that the claim holds for ANY sequence, not the one
	// we wrote — which needs generated sequences and shrinking.
	ObUniversal Obligation = "universal"

	// ObDifferential is that subject and reference agree after any
	// sequence, which needs a reference implementation.
	ObDifferential Obligation = "differential"

	// ObTemporal is a claim that turns on time passing, which needs a
	// clock the run advances rather than one it waits on.
	ObTemporal Obligation = "temporal"

	// ObConcurrent is that interleavings linearize and readers do not
	// corrupt each other, which needs concurrent drivers.
	ObConcurrent Obligation = "concurrent"

	// ObInducedFailure is a claim that needs a call which fails on
	// demand.
	ObInducedFailure Obligation = "induced failure"

	// ObMultiSubject is a claim about a group rather than an instance,
	// which needs two or more subjects in one relation.
	ObMultiSubject Obligation = "multi-subject"
)

// ObSurvival is the sim tier's one kind: the claim holds across a
// crash, a restart or a partial write, which needs a killable process
// or a breakable medium.
const ObSurvival Obligation = "survival"

// Tier names which half of testkit's evidence owns an obligation.
type Tier string

// The three tiers, spelled as the register spells them.
const (
	TierSuite Tier = "suite"
	TierModel Tier = "model"
	TierSim   Tier = "sim"
)

// Tier answers who owns this obligation, which is fixed by what the
// obligation needs and is therefore a property of the kind rather than
// a decision a rule makes.
//
// An unknown obligation reports the model tier and false. Reporting the
// suite tier for something nobody classified would let a claim be
// emitted where it cannot be stated, which is the vacuity ADR-0018 and
// ADR-0028 both exist to prevent — so the fallback is the tier that
// needs machinery, and the caller is told it was a fallback.
func (o Obligation) Tier() (Tier, bool) {
	switch o {
	case ObSurvives, ObContext, ObZeroes, ObDeclaredAnswer, ObNamedPair, ObSelfAgreement:
		return TierSuite, true
	case ObUniversal, ObDifferential, ObTemporal, ObConcurrent, ObInducedFailure, ObMultiSubject:
		return TierModel, true
	case ObSurvival:
		return TierSim, true
	default:
		return TierModel, false
	}
}

// Obligations returns every obligation kind, in register order.
func Obligations() []Obligation {
	return []Obligation{
		ObSurvives, ObContext, ObZeroes, ObDeclaredAnswer, ObNamedPair, ObSelfAgreement,
		ObUniversal, ObDifferential, ObTemporal, ObConcurrent, ObInducedFailure, ObMultiSubject,
		ObSurvival,
	}
}

// ObligationsFor returns the obligations another tier holds for a
// classification, which is what a generated header has to name beside
// the checks it did emit.
//
// Every law in the catalogue is the model tier's by construction: the
// suite tier implements no property [engine/model/law] already carries,
// so a classification with a law has a model-tier obligation whether or
// not the suite tier also covers part of it. That is the whole point of
// ADR-0028 — `idempotent` earns a suite row for the repeat AND binds
// two laws for the sequences, and a header naming only the first reads
// as a file that covered the classification.
//
// Reported as ObUniversal rather than per law. Which of the six model
// kinds a given law needs is a judgement the catalogue does not yet
// record, and every one of them is at least universal — the claim holds
// for any sequence, not the one we wrote. A law column naming the
// narrower kind belongs beside [Binding], and until it exists this says
// the true weaker thing rather than guessing the stronger one.
func ObligationsFor(classification string) []Obligation {
	if len(LawsFor(classification)) == 0 {
		return nil
	}
	return []Obligation{ObUniversal}
}
