// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

// passiveLog is the snapshot-isolation verdict, shared by the three
// anomaly doors because it is one judgment about one subject.
const passiveLog = "the subject is a passive log; a history drawn from pools fabricates its anomalies, which arming proved on its first run"

// UnarmedDoors registers every generated door, clock, and optional role no
// corpus consumer arms, each with the verdict that keeps the absence
// honest. The census derives the door set from the same in-memory emission
// the assertion gate reads, then looks for the arming in the fixture's own
// hand-written tests — the generated option's spelling, which is the one
// convention the corpus consumers already follow. Both directions are
// enforced by name: a door that is neither armed nor registered is a
// visible skip pretending to be a check, and a row for a door some
// consumer now arms is a stale excuse that must be deleted.
//
// Keys are `<corpus-dir>/<law-id>.<item>`, where the item is the door's
// config field, the role the header calls unarmed, or the literal "clock".
//
//nolint:gochecknoglobals // a debt register, read-only, test-facing.
var UnarmedDoors = map[string]string{
	// The two mode fixtures that keep the redeliver role undeclared prove
	// the role's optional omission — the path every optional role rides —
	// while publisher-redeliver and the exactly-once sibling arm it.
	"iface/contract/publisher-atleastonce/AUTO-PUBLISHER-AT-LEAST-ONCE.Redeliver": "the unarmed sibling proves the role's optional omission; publisher-redeliver arms the duplicate",
	"iface/contract/publisher-atmostonce/AUTO-PUBLISHER-AT-MOST-ONCE.Redeliver":   "a redelivery under at-most-once is the violation itself; the bound is proven on the single publish",

	// The pool and snapshot-isolation verdicts were already argued in their
	// consumers' comments; the rows lift them where the census can hold
	// them, which is the difference between a judgment and a habit.
	"iface/contract/pool/AUTO-POOL-LEAK-FREE.balanced":                 "the claim holds at quiescence and the shared walk checks between steps, where a taken value is legitimately still out",
	"iface/mixin/snapshotisolation/AUTO-SNAPSHOT-ISOLATION-G0.history": passiveLog,
	"iface/mixin/snapshotisolation/AUTO-SNAPSHOT-ISOLATION-G1.history": passiveLog,
	"iface/mixin/snapshotisolation/AUTO-SNAPSHOT-ISOLATION-G2.history": passiveLog,

	// The stream doors take the domain's own knowledge, and these subjects
	// expose nothing beside the drain — a door armed from the drain would
	// compare the stream to itself.
	"iface/mixin/overmatch/AUTO-STREAM-OVER-MATCH.required":    "the required subset is the consumer's domain knowledge; this subject exposes nothing beside the drain itself",
	"iface/mixin/permutation/AUTO-STREAM-PERMUTATION.expected": "the expected multiset is the consumer's domain knowledge; this subject exposes nothing beside the drain itself",
}
