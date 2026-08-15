// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import (
	"slices"

	"go.thesmos.sh/testkit/core/lawid"
)

// The four stamp-derived wears, which have no constant beside the structural
// ones because the generator composes them from a classification's own
// parameters rather than from a signature.
//
// Named here so the wardrobe census can be total over all twenty: a wear
// missing from the classification table is a defect nothing knows the meaning
// of, and the prover would silently never count its kills.
const (
	kindWane = "wane"
	kindWax  = "wax"
	kindFade = "fade"
	kindEcho = "echo"
)

// wardrobe is what each wear in the wardrobe actually does wrong.
//
// The other half of [lawid.DefectClass]. A law declares the defect its name
// promises; a wear declares the defect it produces; and the saturation prover
// counts a kill when the two intersect, rather than counting any kill at all.
//
// # The tagging is about the defect, not about the mechanism
//
// `inert` answers zeros and `passthrough` returns its input unchanged, and
// those are different code — but a claim they break, they break the same way:
// the operation had no effect. `flicker` and `flap` likewise differ in what
// they alternate and agree on what a claim sees, which is an answer that moved
// when the claim said it would not.
//
// Several carry two classes, and the second is not decoration. `latch` keeps
// the first write and drops the rest, which is both an ordering defect — the
// fold is no longer commutative — and a loss defect, because a write went
// missing. A law of either class is entitled to count its kill.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var wardrobe = map[string][]lawid.DefectClass{
	kindInert:       {lawid.ClassNoEffect, lawid.ClassLoss},
	kindPassthrough: {lawid.ClassNoEffect, lawid.ClassIntegrity},
	kindFlicker:     {lawid.ClassInstability, lawid.ClassLoss},
	kindFlap:        {lawid.ClassInstability},
	kindFade:        {lawid.ClassInstability, lawid.ClassStaleness},
	kindEcho:        {lawid.ClassStaleness},
	kindSputter:     {lawid.ClassSpuriousFailure},
	kindStick:       {lawid.ClassRepeatability, lawid.ClassSpuriousFailure},
	kindRegress:     {lawid.ClassRegression},
	kindWane:        {lawid.ClassRegression, lawid.ClassLoss},
	kindWax:         {lawid.ClassBound, lawid.ClassDuplication},
	kindOvershoot:   {lawid.ClassBound},
	kindFlood:       {lawid.ClassBound, lawid.ClassDuplication},
	kindDupSeq:      {lawid.ClassDuplication},
	kindDupDrain:    {lawid.ClassDuplication},
	kindGreedy:      {lawid.ClassDuplication},
	kindFadeSeq:     {lawid.ClassLoss},
	kindLatch:       {lawid.ClassOrdering, lawid.ClassLoss},
	kindJumble:      {lawid.ClassOrdering},
	kindSpill:       {lawid.ClassIsolation},
}

// ClassesOf returns what a wear does wrong, empty for a kind the wardrobe does
// not name.
func ClassesOf(kind string) []lawid.DefectClass {
	out := slices.Clone(wardrobe[kind])
	slices.Sort(out)
	return out
}

// Wardrobe returns every wear the classification table names, sorted.
func Wardrobe() []string {
	out := make([]string, 0, len(wardrobe))
	for kind := range wardrobe {
		out = append(out, kind)
	}
	slices.Sort(out)
	return out
}

// Proves reports whether a wear can produce the defect a law is named for.
//
// The kill criterion, one level up from the string match the prover does
// today. That match answers "did this law fail", which every wear on the law's
// own methods can make true — `AUTO-CAS-ATOMIC-ONE-WINNER` failed because a Put
// did nothing, and a Put that does nothing breaks every claim about what a Put
// leaves behind. This answers "did it fail for a reason its name mentions".
//
// A law nothing proves is not a broken law. It is a law whose defect the
// wardrobe cannot produce, which is a gap in the wardrobe — and the whole value
// of separating the two questions is that the second one names what to build.
func Proves(kind, law string) bool {
	for _, c := range ClassesOf(kind) {
		if slices.Contains(lawid.ClassOf(law), c) {
			return true
		}
	}
	return false
}

// unreached registers every law a wear of its own class exists for and never
// reaches, with the reason the dressing cannot get to it.
//
// The third verdict, and the one the defect-class join cannot give. That join
// answers whether the wardrobe *has* the defect a law is named for;
// `gate.UnprovableLaws` holds the laws it does not. These are laws where it
// does — `AUTO-CAUSAL-ORDERING` is an ordering law and `jumble` is an ordering
// wear — and the wear still never reaches the check, because the two are
// looking at different things.
//
// Read out of the law bodies rather than inferred: the four trace laws declare
// `Check(_ *rapid.T, _, _ T)`, ignoring the subject and the reference
// outright, and scan the run's recorded operations. A wear dresses what a
// method answers. Nothing about dressing an answer reorders a trace that was
// already recorded.
//
// A row here is a claim about mechanism, so it names one. "The wear does not
// fire" would be the finding restated.
//
//nolint:gochecknoglobals // a debt register, read-only, test-facing.
var unreached = map[string]string{
	lawid.CausalOrdering: "a trace law: Check ignores the subject and the reference and scans " +
		"the run's recorded operations, which no dressing of a method's answer reorders",
	lawid.MonotonicWrites: "the same trace, read for each client's own write versions rather " +
		"than for the happens-before edges",
	lawid.ReadYourWrites: "the same trace, read for whether a client saw a version older than " +
		"one it wrote itself",
	lawid.WritesFollowReads: "the same trace, read for whether a write landed on a version " +
		"older than the read that preceded it",
	lawid.ReplayCausalOrdering: "a replay law: the ordering it checks is the chain's on the way " +
		"back out, and an ordering wear on the append method changes what goes in",

	lawid.PaginatorNoDuplicates: "the duplication wears dress the page reader's own answer, and " +
		"the claim is about a key appearing in two *different* pages — one dressing cannot " +
		"span the walk",
	lawid.PublisherAtMostOnce: "the same across a delivery: dupdrain repeats within one drain, " +
		"and the bound is on redelivery across drains",

	lawid.TransactionNoMidTxVisibility: "spill crosses a partition boundary; this claim is about " +
		"a boundary in time — a write staged inside a transaction and read before commit",
	lawid.DefaultOnError: "no wear mints an error *beside* a plausible value: sputter's refusal " +
		"arrives with the zero, which is what the declared default usually is, so the " +
		"comparison holds either way",
}

// Unreached returns the reason a wear of this law's own class never reaches
// it, empty where one does.
func Unreached(law string) string { return unreached[law] }

// UnreachedLaws returns every registered law, sorted.
func UnreachedLaws() []string {
	out := make([]string, 0, len(unreached))
	for id := range unreached {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}
