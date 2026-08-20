// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/aggregator"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/lookup"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/pointerreader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/reader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/readernoerror"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/readerwithbool"

	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/suite/projection"
	"go.thesmos.sh/testkit/generator/tiers"
)

// stampRule derives one stamp's checks. The tables below key rules by
// the upstream registry's own name constants — the derive layer never
// respells the vocabulary, and never enumerates it either: an unknown
// mixin refuses loudly, a law-backed one is recognized through tiers
// (the one home of law-backed-ness), and the conformance census holds
// table ∪ tiers ∪ refusals equal to the registry.
//
// A rule answers with refusals as well as plans, because a stamp can
// be recognized and still be unstateable on the interface carrying it:
// the reader shape is a signature, and a signature does not say
// whether anything can supply the input a miss withholds.
type stampRule func(f Iface, m Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal)

// mixinRules is the mixin-axis derivation table: one row per attached
// classification this deriver speaks. Adding a row is the whole cost
// of covering a new deterministic mixin.
//
// The corpus's declared-not-idempotent form (`idempotent=false`, which
// words the accumulates claim) has no row yet: eidos's mixin directive
// requires a positional name and denies negation, so the declaration
// cannot stamp anything today. The grammar ruling is owed upstream;
// [AccumulatesClaim] keeps the wording ready.
func mixinRules() map[string]stampRule {
	return map[string]stampRule{
		MixinIdempotent: idempotentRule,
	}
}

// detectorRules is the detector-axis derivation table, keyed by the
// shape the annotator stamped. Only the reader family so far: the
// writer detectors are the seed derivation's input, the teardown
// shapes are the signature deriver's, and the remaining shapes join
// with the contracts deriver — the detector census arms when that
// deriver lands.
func detectorRules() map[string]stampRule {
	return map[string]stampRule{
		reader.Name:         missRule,
		readernoerror.Name:  missRule,
		readerwithbool.Name: missRule,
		lookup.Name:         missRule,
		pointerreader.Name:  missRule,
		aggregator.Name:     countRule,
	}
}

// Stamps derives the deterministic stamp families — the claims a
// probe or two settles without the property engine. Law-backed stamps
// (ttl, bounded, lifecycleafterclose, …) are recognized through the
// tiers catalogue and left to the laws deriver; a mixin neither
// tabled nor law-backed refuses with the census framing, because an
// uncovered classification must be a named gap, never a silent one.
type Stamps struct{}

// Name implements [Deriver].
func (Stamps) Name() DeriverName { return DeriverStamps }

// Derive implements [Deriver].
func (Stamps) Derive(f Iface) ([]projection.CheckPlan, []Refusal) {
	var plans []projection.CheckPlan
	var refusals []Refusal
	mixins := mixinRules()
	detectors := detectorRules()

	for _, m := range f.Methods {
		detected := m.Shape()
		if len(m.Mixins) == 0 && detected == "" {
			continue
		}
		if r, refused := argsRefusal(DeriverStamps, f, m, "'s stamp checks"); refused {
			refusals = append(refusals, r)
			continue
		}
		call := callOf(m)
		for _, name := range m.Mixins {
			switch rule, tabled := mixins[name]; {
			case tabled:
				ruled, refused := rule(f, m, call)
				plans = append(plans, ruled...)
				refusals = append(refusals, refused...)
			case len(tiers.LawsFor(name)) > 0:
				// The model tier's: the laws deriver binds it through
				// the tiers catalogue.
			case consumedStamps()[name]:
				// An input to another rule rather than a claim of its
				// own. It owes no check because the check it feeds is
				// already derived — and refusing it would report a gap
				// against a stamp that closed one.
			default:
				refusals = append(refusals, Refusal{
					Deriver: DeriverStamps,
					What:    name + " on " + m.Name,
					Why:     "no suite-side derivation rule and no law binds it",
					Remedy:  "add a rule row, a tiers binding, or record the gap in the census",
				})
			}
		}
		if rule, tabled := detectors[detected]; tabled {
			ruled, refused := rule(f, m, call)
			plans = append(plans, ruled...)
			refusals = append(refusals, refused...)
		}
	}
	return plans, refusals
}

// idempotentRule probes the repeat: two clean calls, the second
// changing nothing.
func idempotentRule(f Iface, m Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	return []projection.CheckPlan{{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegIdempotent},
		Class:       vocab.ClassIdempotent,
		Claim:       IdempotentClaim(m),
		Body:        projection.RepeatProbe{Call: call},
		Falsifiable: vocab.Proven(),
		Defect:      projection.SecondCallErrs{Option: projection.OptionName(f.Name, m.Name)},
	}}, nil
}

// The past-tense supply verbs a writer-fed miss claim speaks: the
// simple past beside a sentinel, the participle after the zero form's
// "has". The seed-seam interface speaks "seeded" in both shapes. A
// domain wording the defaults cannot reach — the corpus's "counted" —
// waits on a directive home, recorded in the design doc's frontier.
const (
	missVerbWrote   = "wrote"
	missVerbWritten = "written"
)

// missWording derives what a miss claim speaks: the declared sentinel
// where one is stamped — the ttl declaration's notfound param is the
// one stamped home today — and the supply verb from how the subject
// is populated.
//
// Keyed on Corpus rather than on seeded(): the two agree on every
// interface that has a key and a payload to zip, and diverge on one
// that writes nothing and has no pools either — a codec — where
// seeded() is vacuously true and no run seeds anything. "Nothing has
// seeded" is not what a transform's miss would mean.
func missWording(f Iface, m Method) (sentinel, verb string) {
	sentinel, _ = MissSentinel(m)
	switch {
	case f.Corpus:
		verb = supplySeeded
	case sentinel == "":
		verb = missVerbWritten
	default:
		verb = missVerbWrote
	}
	return sentinel, verb
}

// missRule derives the miss, and the seeded hit beside it. The miss
// is reached by choosing an input that is not there, so a method
// taking nothing after its context offers nowhere to put one and the
// rule licenses nothing.
//
// The shape alone does not license it either. A codec's Encode is
// reader-shaped down to the return pair — one input, a value and an
// error — and nothing on the interface writes, so there is no input
// it has not been given: every draw is as valid as the canonical one
// and a check asserting the zero for the alternate asserts a
// falsehood. Either the declaration names what a miss reports, or the
// run has to be able to make one; without both the rule refuses, so
// the gap is named in the header rather than emitted as a claim.
func missRule(f Iface, m Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	if !m.HasInput() {
		return nil, nil
	}
	sentinel, verb := missWording(f, m)
	if sentinel == "" && !f.supplies() {
		return nil, []Refusal{
			{
				Deriver: DeriverStamps,
				What:    m.Name + "'s miss check",
				Why:     "nothing on this interface writes and no corpus seeds it, so no input is one nothing supplied",
				Remedy:  "declare what a miss reports with //testkit:mixin notfound sentinel=Err…, or write the claim as a row",
			},
		}
	}
	plans := []projection.CheckPlan{{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegMiss},
		Class:       vocab.ClassReader,
		Claim:       MissClaim(m, sentinel, verb),
		Body:        projection.MissProbe{Call: missCall(f, m), Sentinel: projection.Expr(sentinel)},
		Falsifiable: vocab.Proven(),
		Defect:      projection.InventsHit{Option: projection.OptionName(f.Name, m.Name)},
	}}
	if f.Corpus {
		plans = append(plans, projection.CheckPlan{
			ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegHit},
			Class:       vocab.ClassReader,
			Claim:       HitClaim(m),
			Body:        projection.HitProbe{Call: hitCall(m)},
			Falsifiable: vocab.Proven(),
			Defect:      projection.SwapsValues{Option: projection.OptionName(f.Name, m.Name)},
		})
	}
	return plans, nil
}

// countRule derives the seeded-aggregator equality. An aggregator on
// an interface that writes has no fixed number to equal — its count
// claims are the law catalogue's territory — and one on an interface
// nothing seeds has no number at all, so the rule licenses nothing in
// either case. Silent rather than refused: the count is the hit's
// companion and the miss beside it already names the gap.
func countRule(f Iface, m Method, call projection.CallPlan) ([]projection.CheckPlan, []Refusal) {
	if !f.Corpus {
		return nil, nil
	}
	return []projection.CheckPlan{{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegCount},
		Class:       vocab.ClassReader,
		Claim:       CountClaim(m),
		Body:        projection.CountProbe{Call: call},
		Falsifiable: vocab.Proven(),
		Defect:      projection.FreezeReturn{Option: projection.OptionName(f.Name, m.Name)},
	}}, nil
}

// consumedStamps are the classifications another derivation READS
// rather than deriving a check from.
//
// The category the census had no name for. A stamp usually states a
// claim and owes a check; these state an identity some other rule needs
// — and a stamp that closed a gap being reported AS a gap is the
// refusal list saying the opposite of what happened.
func consumedStamps() map[string]bool {
	return map[string]bool{
		// Names WHAT a miss reports. The claim that a miss IS reported
		// belongs to the reader shape, whose rule reads this to choose
		// the sentinel arm of the body over the zero arm.
		MixinNotFound: true,
	}
}

// hitCall is [callOf] with the drawn key replaced by the loop variable
// the hit body ranges the corpus with.
//
// Every seeded key, not the fixture's one. The fixture holds a member of
// the key pool and the corpus holds all of them, so a body drawing the
// fixture asks about the same entry once per iteration — which passes
// for a subject that kept the first thing it was given and dropped the
// rest, the exact failure a hit check is for.
func hitCall(m Method) projection.CallPlan {
	call := callOf(m)
	for i, arg := range call.Args {
		if arg == projection.ExprCtx {
			continue
		}
		call.Args[i] = projection.ExprSeededKey
		break
	}
	return call
}
