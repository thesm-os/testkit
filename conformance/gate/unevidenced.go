// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

// UnevidencedClassifications registers every classification eidos ships that
// neither tier asserts, each with the verdict that keeps the absence honest.
//
// The third register of its kind, and the pattern is deliberate: a gap the
// build can read beats a gap confessed in a fixture comment, because a comment
// goes stale silently and a row reddens when it stops being true. Both
// directions are enforced by name — a classification neither asserted nor
// registered fails the census, and a row for one some tier now asserts is a
// stale excuse the census deletes by failing.
//
// # What a row is allowed to say
//
// That the claim needs something a generated run does not have: a second
// client, an injected failure, a clock the sequences advance, a declaration
// naming which parameter carries the claim. Those are facts about the claim.
//
// What a row may not say is that the classification is unimportant. Every one
// of these is stamped by a fixture that exists, and three registers now hold
// the same line: the absence is argued or the absence is a defect, and there is
// no third state where nobody looked.
//
// Keys are the classification name as eidos registers it.
//
//nolint:gochecknoglobals // a debt register, read-only, test-facing.
var UnevidencedClassifications = map[string]string{
	// Contracts. Each names a protocol whose claim needs a condition a
	// single-subject run against a static clock cannot produce.
	"circuit-breaker": "the claim is a transition between closed, open and half-open, reached by a failure count and a cooldown; the suite tier injects no failures and the derived reference models none, so both would assert the closed state and call it a breaker",
	"leader-election": "the claim is that exactly one of several clients leads; a suite run holds one subject and the derived reference has no notion of a peer, so exclusivity has nothing to be exclusive against",
	"rate-limit":      "the claim is calls per unit time, and both tiers would measure it against a clock nothing advances — a limiter so measured either admits everything or refuses everything, whichever side of the burst the run starts on",

	// Detectors. A shape earns the checks its signature admits; these three
	// have signatures that admit nothing beyond the smoke family, which is a
	// signature check rather than the shape's.
	"mutator":        "state change with nothing returned: no error to assert on and no value to compare, so what the shape earns is the smoke family, which the signature earns anyway — the effect is observable only through another method, which is the `sideeffect` mixin's claim and is checked there",
	"streamconsumer": "a stream taken in rather than handed out; a check would supply one and then ask a second method what became of it, which is `sideeffect` again — the shape itself claims only that a parameter is a stream",
	"voidlifecycle":  "a teardown that cannot fail, so a single call admits no assertion at all; that calling it twice is safe is the `idempotent` mixin's claim and is checked under that name",

	// Measured, not argued. This one is a defect rather than a debt and the row
	// says so: `closer` is stamped corpus-wide, but its only carrier is the
	// nullary Close that `lang/embeddedforeign` takes from io.Closer, and the
	// stamp does not survive flattening into the local method set — the suite
	// projection reads an empty shape for it, so no header names it either.
	// Compare stays green on the foreign node. A locally declared nullary
	// closer would close this, and would delete this row.
	"closer": "DEFECT, not a debt: the sole carrier is a Close flattened in from io.Closer, and the shape stamp does not survive the flattening — the suite projection reads no shape, so nothing generates and no header says so",

	// Mixins. Three shapes of absence: a marker with no claim of its own, a
	// claim another classification already checks under its own name, and a
	// claim needing a condition the harness cannot produce.
	"deprecated":      "a fact about a method's lifecycle rather than about its behaviour; there is nothing to assert and the generated documentation states it, which is where a caller needs it",
	"integrationonly": "a run modifier, not a claim: it gates every other check behind an environment, so a check about it would be a check about the harness rather than the subject",
	"timeaware":       "a marker that a callable depends on a clock, deliberately without saying which quantity — the quantities are `ttl`, `timeout` and `windowed`, and those carry the checkable claims",

	"concurrentreaders": "the claim needs readers overlapping a writer; the suite tier's concurrency check is the signature-derived smoke under load, which any method earns, and the model tier's linearizability leg is selected from shape rather than from this stamp",
	"retrysucceeds":     "the claim is that a transient failure succeeds on a later attempt, which needs a failure the harness can induce and then withdraw; neither tier injects one, so a subject that never fails satisfies it vacuously",
	"scope":             "confinement to a named scope is the claim `partition` states, and `partition`'s check needs `axis=` to say which parameter carries it — `scope` declares no such parameter, so a derived check would vary the wrong argument and pass against a subject ignoring scopes entirely",
	"errors":            "the stamp declares that misses report through a named sentinel; what a run can compare is the sentinel's identity, which the miss family already asserts under the detector shape that earns it — the mixin adds a name to a claim rather than a claim",
}
