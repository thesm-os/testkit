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
type stampRule func(f Iface, m Method, call projection.CallPlan) []projection.CheckPlan

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
		call := callOf(f, m)
		for _, name := range m.Mixins {
			switch rule, tabled := mixins[name]; {
			case tabled:
				plans = append(plans, rule(f, m, call)...)
			case len(tiers.LawsFor(name)) > 0:
				// The model tier's: the laws deriver binds it through
				// the tiers catalogue.
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
			plans = append(plans, rule(f, m, call)...)
		}
	}
	return plans, refusals
}

// idempotentRule probes the repeat: two clean calls, the second
// changing nothing.
func idempotentRule(f Iface, m Method, call projection.CallPlan) []projection.CheckPlan {
	return []projection.CheckPlan{{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegIdempotent},
		Class:       vocab.ClassIdempotent,
		Claim:       IdempotentClaim(m),
		Body:        projection.MixinProbe{Mixin: MixinIdempotent, Calls: []projection.CallPlan{call, call}},
		Falsifiable: vocab.Proven(),
		Defect:      projection.SecondCallErrs{Option: projection.OptionName(f.Name, m.Name)},
	}}
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
func missWording(f Iface, m Method) (sentinel, verb string) {
	sentinel, _ = m.MixinParam(MixinTTL, MixinTTLNotFound)
	switch {
	case f.seeded():
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
func missRule(f Iface, m Method, call projection.CallPlan) []projection.CheckPlan {
	if !m.HasInput() {
		return nil
	}
	sentinel, verb := missWording(f, m)
	plans := []projection.CheckPlan{{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegMiss},
		Class:       vocab.ClassReader,
		Claim:       MissClaim(m, sentinel, verb),
		Body:        projection.MixinProbe{Mixin: m.Shape(), Calls: []projection.CallPlan{call}},
		Falsifiable: vocab.Proven(),
		Defect:      projection.InventsHit{Option: projection.OptionName(f.Name, m.Name)},
	}}
	if f.seeded() {
		plans = append(plans, projection.CheckPlan{
			ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegHit},
			Class:       vocab.ClassReader,
			Claim:       HitClaim(m),
			Body:        projection.MixinProbe{Mixin: m.Shape(), Calls: []projection.CallPlan{call}},
			Falsifiable: vocab.Proven(),
			Defect:      projection.SwapsValues{Option: projection.OptionName(f.Name, m.Name)},
		})
	}
	return plans
}

// countRule derives the seeded-aggregator equality. An aggregator on
// an interface that writes has no fixed number to equal — its count
// claims are the law catalogue's territory — so the rule licenses
// nothing there.
func countRule(f Iface, m Method, call projection.CallPlan) []projection.CheckPlan {
	if !f.seeded() {
		return nil
	}
	return []projection.CheckPlan{{
		ID:          projection.IDPlan{Method: m.Name, Seg: vocab.SegCount},
		Class:       vocab.ClassReader,
		Claim:       CountClaim(m),
		Body:        projection.MixinProbe{Mixin: m.Shape(), Calls: []projection.CallPlan{call}},
		Falsifiable: vocab.Proven(),
		Defect:      projection.FreezeReturn{Option: projection.OptionName(f.Name, m.Name)},
	}}
}
