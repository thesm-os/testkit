// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

import "slices"

// Outcome is what a completed run observed about its own laws.
//
// # Why the runner returns anything
//
// A run used to communicate exactly one bit, and only in one direction: it
// failed the TB, or it said nothing. Everything else it learned — which laws
// engaged, which were declined on every single draw, whether the reference it
// compared against was another implementation or a second copy of the subject
// — was computed and then discarded at the boundary.
//
// That is the silent-green class one layer up. A suite reporting "44 legs, 44
// passed" cannot distinguish a run that checked forty-four claims from one
// whose preconditions never engaged, and the datum that tells them apart was
// already sitting in the registry.
//
// # What it does not describe
//
// Only a run that reached the end returns one. A law violation ends the run
// through the TB, so an Outcome always describes a green run — which is the
// case that needed a voice. A red run already has one.
type Outcome struct {
	// Laws is the per-law census, keyed by law ID. Nil when the run
	// registered no laws.
	Laws map[string]LawCensus
}

// LawCensus counts what became of one law across every check in a run.
//
// # Reading these numbers honestly
//
// Only [LawCensus.Fired] and the Ran/Vacuous *comparison* carry a sound
// signal. The absolute counts do not: rapid runs a property around a hundred
// times and re-runs it while shrinking a failure, and the registry accumulates
// across all of it. So "vacuous 340 times" is not a measurement of anything a
// consumer can compare between runs, and nothing should ratchet on it.
//
// What is sound is the qualitative reading, which is the one that matters: a
// law with Ran > 0 and Vacuous == Ran was declined on every draw this run
// supplied, and asserted nothing at all.
type LawCensus struct {
	// Ran is how many times the law was checked.
	Ran int

	// Fired is how many times the law found a violation. On a returned
	// Outcome this is always zero — a violation ends the run — so it is
	// here for the in-process census and for tests that drive a law
	// against a subject built to break it.
	Fired int

	// Vacuous is how many times the subject declined the precondition, so
	// the claim was never engaged.
	Vacuous int
}

// Engaged reports whether the law reached a verdict at least once.
//
// False means every check was declined: the binding read as coverage and
// checked nothing.
func (c LawCensus) Engaged() bool { return c.Ran > 0 && c.Vacuous < c.Ran }

// Engaged reports whether any law reached a verdict.
//
// False on a run that registered laws means the whole leg proved nothing, and
// a caller that reports per-leg outcomes should say so rather than counting it
// among the passes. A run with no laws at all is not vacuous — it had no
// claims to engage — so this reports false and the caller decides what that
// means for a leg whose oracle was the differential.
func (o Outcome) Engaged() bool {
	for _, c := range o.Laws {
		if c.Engaged() {
			return true
		}
	}
	return false
}

// Unengaged names the laws that ran and were declined every time, sorted for
// a stable message. Empty when every law engaged at least once.
func (o Outcome) Unengaged() []string {
	var out []string
	for id, c := range o.Laws {
		if c.Ran > 0 && !c.Engaged() {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

// outcomeOf reads the census off a registry. Tolerates a nil registry: a run
// configured with no laws is the common case for a pure differential leg.
func outcomeOf[T any](r *Registry[T]) Outcome {
	if r == nil {
		return Outcome{}
	}
	return Outcome{Laws: r.Census()}
}

// Census returns the per-law counts this registry accumulated.
//
// Replaces the older Coverage, which returned ran and fired as two maps and
// omitted the vacuous count — the one a caller needs to tell a law that held
// from a law that was never asked.
func (r *Registry[T]) Census() map[string]LawCensus {
	out := make(map[string]LawCensus, len(r.ran))
	for id, ran := range r.ran {
		out[id] = LawCensus{Ran: ran, Fired: r.fired[id], Vacuous: r.vacuous[id]}
	}
	// A law can be declined before it is ever counted as run by a path that
	// only tracks vacuity; fold those in rather than losing them.
	for id, v := range r.vacuous {
		if _, seen := out[id]; !seen {
			out[id] = LawCensus{Ran: v, Vacuous: v}
		}
	}
	return out
}
