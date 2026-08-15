// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"slices"

	"go.thesmos.sh/testkit/core/lawid"
	"go.thesmos.sh/testkit/generator/model"
)

// UnprovableLaws registers every law whose defect class no wear in the
// wardrobe produces, with what building the wear would take.
//
// The register the defect-class axis exists to produce, and the reason that
// axis is worth its cost. The saturation prover used to answer one question —
// did some defect make this law fail — which every law passed and which
// `AUTO-CAS-ATOMIC-ONE-WINNER` passed on a Put that did nothing. Splitting it
// into "did a defect of the law's own class make it fail" turns a green row
// into two distinguishable answers: the law is proved, or the wardrobe cannot
// produce the defect the law is named for.
//
// The second is a gap in the wardrobe, not in the law, and that is a different
// piece of work with a different owner. These rows name it.
//
// Keys are the law identifier. Both directions are enforced: a law neither
// provable nor registered fails the census, and a row for a law some wear now
// proves is a stale excuse the census deletes by failing.
//
//nolint:gochecknoglobals // a debt register, read-only, test-facing.
var UnprovableLaws = map[string]string{
	// atomicity — a partial effect where the claim is all or nothing. Every
	// wear in the wardrobe is total: it changes what an operation answers or
	// whether it happens, never whether it happened *halfway*. The wear this
	// needs applies a multi-step effect and abandons it between steps, which
	// no signature-derived dressing can compose because the steps are the
	// subject's own.
	lawid.AtomicWrite: "no wear leaves a write half-applied; the wardrobe's defects are " +
		"total, and a partial effect has to be staged from inside the operation",
	lawid.SagaFullCompensation: "the same: a compensation that runs for some steps and not " +
		"others needs a wear that knows where the steps are",
	lawid.TransactionRollback: "a rollback that keeps some of what it undid is a partial " +
		"effect, and the wardrobe has no way to make one",
	lawid.TwoPhaseRollbackAfterCommit: "the same, one phase later: a rollback issued after " +
		"commit has to leave part of the committed effect standing to be seen at all",

	// resource — something taken and not given back. The wardrobe wears a
	// method's answers; a leak is about what the subject retains after the
	// call, which no dressing over the interface can reach. The wear this
	// needs holds a reference the release path would have dropped, and only
	// the subject knows what that is.
	lawid.LeakFree: "a leak is state the subject keeps after the call, and every wear here " +
		"acts on what the call answers rather than on what it retains",
	lawid.PoolBalanced: "the same: an unbalanced pool is a value taken and not returned, " +
		"which is a fact about the pool between calls rather than about any one call",
	lawid.PoolLeakFree: "the same, over the pool's own accounting: a value the pool " +
		"stopped tracking is invisible to every wear that acts on an answer",
	lawid.LeaseReleasedOnCancel: "a lease held past its cancel is retention rather than a " +
		"wrong answer, and retention is what the wardrobe has no vocabulary for",

	// permissive — an operation that succeeds where the claim requires a
	// refusal. Every wear here produces a *wrong answer*; this class needs a
	// *missing refusal*, and these four were being proved by `stick` or
	// `spill` — wears that make an operation refuse or leak, which is either
	// the behaviour the law wants or a different claim entirely.
	//
	// The wear this needs strips a guard: it lets a closed subject keep
	// serving, a held lease grant again, a committed transaction accept a
	// second terminal op. That is the inverse of every dressing in the
	// wardrobe, which add wrongness rather than removing a check.
	lawid.CursorNextAfterClose: "a Next after Close that answers instead of refusing is a " +
		"missing refusal, and every wear here produces a wrong answer instead",
	lawid.LifecycleAfterClose: "the same over a lifecycle: `stick` makes the operation refuse " +
		"after its first call, which is what this law wants rather than what breaks it",
	lawid.TwoPhaseMutex: "a committed transaction that accepts a second terminal op is a guard " +
		"that did not fire, and no dressing over the interface can remove one",
	lawid.LeaseDoubleAcquireBlocks: "a second acquire that grants is the same missing refusal, " +
		"and duplicating a stream element is not the same defect as duplicating a lease holder",
}

// Unprovable returns every bound law no wear can prove and no row argues.
func Unprovable() []string { return unprovableAgainst(UnprovableLaws) }

// ArguedButProvable returns the register rows a wear has since covered.
func ArguedButProvable() []string { return Stale(UnprovableLaws) }

// Reports answers whether one law would be reported against a given register.
//
// Exported so the selecting arm can be driven from a test. Against the real
// register the answer is no for every law — which is the state this gate
// holds, and the reason nothing exercises the yes.
func Reports(law string, register map[string]string) bool {
	return slices.Contains(unprovableAgainst(register), law)
}

// Stale returns the rows of a register that name a law some wear proves.
func Stale(register map[string]string) []string {
	var out []string
	for id := range register {
		if provable(id) {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

// unprovableAgainst is [Unprovable] over an arbitrary register.
func unprovableAgainst(register map[string]string) []string {
	var out []string
	for _, id := range lawid.All() {
		if provable(id) {
			continue
		}
		if _, argued := register[id]; argued {
			continue
		}
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

// provable reports whether any wear in the wardrobe produces a defect of a
// class this law names.
func provable(law string) bool {
	for _, kind := range model.Wardrobe() {
		if model.Proves(kind, law) {
			return true
		}
	}
	return false
}
