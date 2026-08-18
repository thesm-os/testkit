// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package suite holds the types every generated conformance package uses
// and every consumer composes with. Generated code builds these values;
// nothing in this package is generated.
//
// A check is data: an identifier, the claim it pins, what it needs from a
// subject, and a function. Generated checks and hand-written checks are
// the same type, so everything that works on one works on the other.
//
// See RFC-0004 for the design and ADR-0019 through ADR-0027 for the
// decisions behind it.
package suite

import (
	"errors"
	"maps"
	"sort"
	"testing"

	"go.thesmos.sh/testkit/clock"
)

// ID names one check. It is stable across regenerations and follows the
// grammar in id.go: a scope segment (an exported method name, or a
// reserved lowercase family word) followed by at least one more segment.
type ID string

// Class groups checks by the kind of claim they discharge. The prefix is
// the axis the claim came from, so classes from different vocabularies
// never collide in a lock file.
type Class string

// The class families this report buckets. The set is closed — validate
// refuses a class outside it — because a family typo would mint a phantom
// bucket in the lock and the histogram. Three of the five ARE the ID
// families: the same word, one home, so a rename cannot half-happen.
const (
	ClassFamilySignature = "signature"
	ClassFamilyMixin     = "mixin"
	ClassFamilyModel     = FamilyModel
	ClassFamilySim       = FamilySim
	ClassFamilyHand      = FamilyHand
)

// The segment words the contract fixes. A check's ID segment and its
// class leaf are the same word — "the differential leg" is one concept —
// so both compose from these rather than spelling it twice.
const (
	SegSmoke      = "smoke"
	SegCancel     = "cancel"
	SegDeadline   = "deadline"
	SegNilContext = "nilcontext"
	SegZeroValue  = "zero-on-error"

	SegReader     = "reader"
	SegIdempotent = "idempotent"

	// The deterministic reader family's check segments: the miss and
	// the seeded hit/count, plus the negated-idempotent accumulation.
	// In the vocabulary because the emitter and the runtime must spell
	// one grammar — a slug with two homes drifts.
	SegMiss        = "miss"
	SegHit         = "hit"
	SegCount       = "count"
	SegAccumulates = "accumulates"

	SegDifferential = "differential"
	SegLaws         = "laws"
	SegConcurrent   = "concurrent"
	// SegLinearizable is the concurrent leg's own row: histories under
	// concurrent load linearize. Not a law identifier — the leg runs
	// the linearize engine, not a law binding — so its slug lives here
	// beside the family segments rather than in core/lawid.
	SegLinearizable = "AUTO-LINEARIZABLE"
	SegClocked      = "clocked"
	SegPoison       = "poison"
	SegLifecycle    = "lifecycle"
	SegAppender     = "appender"

	SegRecovery = "recovery"
	SegCrash    = "crash"
	SegFault    = "fault"

	SegHandWritten = "hand-written"
)

// The standard classes, composed from a family and a segment so a class
// and the ID segment naming the same leg cannot drift apart. A class
// outside this set is legal — the leaf stays open, buckets are additive —
// but it must still name a known family.
const (
	ClassSmoke      Class = ClassFamilySignature + "/" + SegSmoke
	ClassCancel     Class = ClassFamilySignature + "/" + SegCancel
	ClassDeadline   Class = ClassFamilySignature + "/" + SegDeadline
	ClassNilContext Class = ClassFamilySignature + "/" + SegNilContext
	ClassZeroValue  Class = ClassFamilySignature + "/" + SegZeroValue

	ClassReader     Class = ClassFamilyMixin + "/" + SegReader
	ClassIdempotent Class = ClassFamilyMixin + "/" + SegIdempotent

	ClassDifferential Class = ClassFamilyModel + "/" + SegDifferential
	ClassLaws         Class = ClassFamilyModel + "/" + SegLaws
	ClassConcurrent   Class = ClassFamilyModel + "/" + SegConcurrent
	ClassClocked      Class = ClassFamilyModel + "/" + SegClocked
	ClassPoison       Class = ClassFamilyModel + "/" + SegPoison
	ClassLifecycle    Class = ClassFamilyModel + "/" + SegLifecycle
	ClassAppender     Class = ClassFamilyModel + "/" + SegAppender

	ClassSimRecovery Class = ClassFamilySim + "/" + SegRecovery
	ClassSimCrash    Class = ClassFamilySim + "/" + SegCrash
	ClassSimFault    Class = ClassFamilySim + "/" + SegFault

	ClassHandWritten Class = ClassFamilyHand + "/" + SegHandWritten
)

// Caps says what a check needs from a subject beyond being constructed.
// The zero value needs nothing.
//
// A subject that cannot meet a requirement makes the check fail, with the
// missing piece named in the message. It never makes the check skip: a
// skipped check for a missing capability is how a claim goes unchecked
// while the run stays green.
type Caps map[Capability]any

// Capability names one thing a check can need from a subject.
//
// A keyed set rather than a struct of fields, and the engine is what settles
// it. Two capabilities are about the subject — build me on a clock, put
// yourself in this state — and the rest are values only the consumer has: the
// expected multiset a permutation claim compares against, the subset an
// over-match claim requires, the history a snapshot-isolation claim reads,
// the accounting a pool claim balances. The conformance gate's unarmed-door
// register holds six kinds today, keyed `<law-id>.<field>`.
//
// Two fields could carry two of those. A law needing an expected multiset
// would then declare nothing — which is the silent-green class this whole
// contract exists to remove, arriving inside the mechanism built to remove
// it. So the set is open, and adding a capability is a registry row under the
// same census-or-red discipline the role keywords take.
type Capability string

const (
	// CapClock is answered by [Subject.OnClock]: the subject must be
	// constructible on the run's test clock, so a check can move time. Its
	// value in a Caps is ignored; presence is the whole declaration.
	CapClock Capability = "clock"

	// CapInduce is answered by [Subject.Induces]: the subject must be able to
	// enter the state a sentinel names, on demand. Its value is the sentinel.
	CapInduce Capability = "induce"

	// CapRecover is answered by [Subject.Recover]. Reserved for the
	// simulation RFC; no generated check declares it.
	CapRecover Capability = "recover"
)

// Needs builds a Caps from one capability and its value, for the common case
// of a check needing exactly one thing.
func Needs(c Capability, v any) Caps { return Caps{c: v} }

// NeedsClock declares that a check moves time. A subject without OnClock
// fails it by name.
func NeedsClock() Caps { return Needs(CapClock, nil) }

// NeedsInduce declares that a check must put the subject into the state
// the sentinel names. A subject with no trigger for it fails by name.
func NeedsInduce(sentinel error) Caps { return Needs(CapInduce, sentinel) }

// NeedsRecover declares that a check rebuilds over durable state.
func NeedsRecover() Caps { return Needs(CapRecover, nil) }

// Check is one assertion about a subject.
type Check[S any] struct {
	// ID is the check's stable identity: the handle a drop, a rerun, the
	// manifest and the report all share. See ValidateID for the grammar.
	ID     ID
	Method string // empty for checks that are not about one method
	// Class buckets the check for the report's census; <family>/<slug>,
	// validated at run time. Identity lives in ID — Class is presentation.
	Class Class
	Claim string // one sentence; shown in the report and written to checks.lock
	Needs Caps

	// Binds names the assertion bodies this check delegates to — law IDs
	// with their probe sets — so the manifest carries them. The claim and
	// the ID survive a body change unchanged; this column is what makes
	// such a change diff. Empty for a check whose body is its own
	// assertion.
	Binds []string

	// Falsifiable records whether anything has shown this check able to
	// fail, and where it cannot, the argument for why.
	//
	// The datum the first draft of this contract had no field for, and the
	// one a consumer most needs. A run that says "44 legs, 44 passed" does
	// not distinguish a suite that works from one that asserts nothing, and
	// both tiers already compute the difference: the suite tier drives every
	// generated check against a stand-in built to break it, and the model
	// tier requires the kill to come from a defect of the law's own declared
	// class.
	//
	// Neither answer reached the consumer. Now the report carries it, and a
	// conformance statement reads "64 checks, 61 proven able to fail, 3
	// argued" rather than "64 passed".
	Falsifiable Falsifiability

	// Exactly one of Run and RunWith must be set.
	//
	// Run gets one fresh subject. This is the shape of nearly every
	// check.
	//
	// RunWith gets the Subject itself, so it can build, close, and
	// rebuild instances through the subject's own constructors. Clocked
	// laws, recovery checks, and differential comparisons need this.
	Run func(tb testing.TB, s S)
	// RunWith receives the subject itself, for a claim one instance
	// cannot state: build two, move the clock, induce a failure. Only a
	// RunWith body can note vacuity or a tier — those are properties of
	// a driven sequence.
	RunWith func(tb testing.TB, sub Subject[S])
}

// Subject is one implementation under test.
type Subject[S any] struct {
	// Name labels the subtest tree and every report row this subject
	// produces.
	Name string

	// New builds a fresh instance. tb carries cleanup, so a subject
	// backed by a container or a connection pool can register teardown.
	New func(tb testing.TB) S

	// OnClock builds an instance reading the run's test clock. When it is
	// nil, checks that need a clock fail and name this field.
	OnClock func(tb testing.TB, clk *clock.TestClock) S

	// Induces maps a sentinel to the trigger that puts the subject into
	// that state. A check needing a sentinel with no entry here fails and
	// names the sentinel.
	Induces map[error]func(tb testing.TB, s S)

	// Recover rebuilds over the durable state prior left behind — the
	// crash-recovery seam. It takes the prior instance because the medium
	// is the instance's, not the subject's: every parallel check owns its
	// own world, and a recovery constructor with no handle to the crashed
	// instance could only reopen a shared one.
	Recover func(tb testing.TB, prior S) S

	// Oracle marks this subject as the reference for the run. Every other
	// subject's model checks compare against it instead of against a
	// second copy of themselves. At most one subject per run may set it.
	Oracle bool

	// Serial keeps this subject out of the parallel group. Its checks run
	// one at a time, and never concurrently with any other subject's:
	// serial subjects execute inline while parallel subtests queue behind
	// the parent, so a serial subject runs BEFORE the parallel group,
	// alone. Real backends that cannot take N simultaneous constructions
	// need it. Serial and Oracle together are refused: the oracle's
	// constructor is called concurrently by every parallel subject's
	// model legs.
	Serial bool

	// reference is the oracle subject's constructor. Run sets it before
	// executing; consumers never do.
	reference func(tb testing.TB) S

	// Setup runs once per subject, before any of its legs; Teardown runs
	// once after the last leg ends, through the subtest's cleanup. They
	// exist for fixtures whose cost dwarfs a leg — a container, a
	// migrated database — where per-leg construction through New would
	// be the run's whole wall-clock. New still runs per leg: Setup owns
	// the medium, New the instance over it.
	Setup    func(tb testing.TB)
	Teardown func(tb testing.TB)

	// Provides answers capabilities beyond the named fields — the open
	// half of the registry the Capability doc promises. The runner's gate
	// consults it for any key without a dedicated field; the value is
	// whatever the capability's checks read (a history, an expected
	// multiset, a pool accounting). The named fields stay named because
	// their types are the documentation; this map is for the doors only a
	// given interface's checks know.
	Provides map[Capability]any

	// Excused marks checks this subject structurally cannot take — a
	// memory-only store has no medium to recover, however its harness is
	// armed. Not a silent skip: the leg is recorded as "did not run:
	// excused", a reason deliberately distinct from a reviewer's
	// run-level drop, so a rule keyed on either never conflates them. A
	// capability the subject COULD provide but does not stays a failure —
	// red-with-the-fix is for wiring, Excused is for impossibility.
	Excused map[ID]bool

	// note is where a check writes back what only it can know. Run installs
	// a fresh one per leg; consumers never set it.
	//
	// A pointer because Subject is handed to a check by value. [Subject.New]
	// and the rest are read-only, so a copy is harmless for them; these two
	// facts travel the other way, and the indirection is what lets them.
	note *legNote
}

// Tier names how a model check got its reference. A typed vocabulary
// because generated code writes it and the report switches on it: a typo
// in a naked string compiled, and the report answered "no reference".
type Tier string

const (
	// TierDifferential is a declared oracle — another real implementation.
	TierDifferential Tier = "differential"
	// TierDerived is a reference built from the interface's shape.
	TierDerived Tier = "derived"
	// TierTwin is a second copy of the subject. Generated model checks
	// never report it — a deterministic bug is shared by both copies, so
	// the derived reference replaced the twin fallback — and it stays in
	// the vocabulary for a consumer check that genuinely compares two
	// instances, isolation being the claim that needs exactly that.
	//
	// TierTwin is a second copy of the subject, which cannot catch a
	// deterministic bug because both copies have it.
	TierTwin Tier = "twin"
)

// legNote carries a check's own account of the leg it just ran.
//
// Two facts the runner cannot derive. Whether preconditions ever engaged is
// known only inside the check — the engine counts it and nothing above could
// see it. And which reference tier ran is the check's choice: it picked a
// derived oracle or fell back to a twin, and from outside those are the same
// call.
type legNote struct {
	reason    string   // why no verdict was reached; empty when one was
	tier      string   // how the reference was built; empty for a non-model check
	unengaged []string // laws that never engaged, on a leg that still passed
}

// Note records why this leg reached no verdict — that its preconditions never
// engaged, say, so the green result asserts nothing.
//
// Only a RunWith check can call it. A Run check is handed the instance rather
// than the subject, which is the right split: vacuity is a property of a
// driven sequence, not of a single call.
func (s Subject[S]) Note(reason string) {
	if s.note != nil {
		s.note.reason = reason
	}
}

// NoteUnengaged records the laws a PASSING leg never engaged: the leg
// asserted something, but not everything it bound, and a report that
// counted it among the plain passes would mask the partial vacuity —
// one engaged law out of five reads as "the bundle held" otherwise.
func (s Subject[S]) NoteUnengaged(names []string) {
	if s.note != nil && len(names) > 0 {
		s.note.unengaged = append([]string(nil), names...)
	}
}

// NoteTier records how this check got its reference: "differential" for a
// second implementation, "derived" for a reference the engine builds from the
// interface's shape, "twin" for a second copy of the subject.
//
// The runner can tell differential from twin on its own, and cannot see
// derived at all — a derived reference is built inside the check, and from
// outside it is just another constructor. So the check says.
func (s Subject[S]) NoteTier(tier Tier) {
	if s.note != nil {
		s.note.tier = string(tier)
	}
}

// Reference returns the oracle's constructor when the run declared one.
// A check that can use a real reference calls this; when ok is false it
// must fall back to a second instance of the subject, which only catches
// nondeterminism.
func (s Subject[S]) Reference() (build func(tb testing.TB) S, ok bool) {
	return s.reference, s.reference != nil
}

// Inducer resolves the trigger for a sentinel, matching with errors.Is
// rather than map identity.
//
// Identity is how the map is keyed, and it is the wrong comparison for
// errors: a consumer who registered a wrapped sentinel — or whose driver
// wraps the interface's — followed the advice in the failure message and
// would still be told they had not. Every check in the generated suite
// compares errors with errors.Is; the capability gate has to speak the
// same language.
//
// A registered trigger that is nil counts as absent: a nil func passing
// the gate would turn the named capability failure into a panic at the
// call site.
func (s Subject[S]) Inducer(sentinel error) (trigger func(tb testing.TB, s S), ok bool) {
	if t, exact := s.Induces[sentinel]; exact && t != nil {
		return t, true
	}
	// One direction only: a registered trigger covers the asked sentinel
	// when the trigger's key IS that sentinel under wrapping — asking the
	// reverse would let a broad registration answer for a narrower ask.
	// The walk is sorted so two matching registrations resolve the same
	// way on every run instead of by map order.
	keys := make([]error, 0, len(s.Induces))
	for key := range s.Induces {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Error() < keys[j].Error() })
	for _, key := range keys {
		if t := s.Induces[key]; t != nil && errors.Is(key, sentinel) {
			return t, true
		}
	}
	return nil, false
}

// CompatV2 is the compatibility witness for this library's v2 check
// contract. Every generated package references it once:
//
//	var _ = suite.CompatV2
//
// Generated code and the library it runs on can drift — the generated
// files pin a plugin version in a comment, and a comment enforces nothing
// (protobuf learned this as gencode/runtime skew). A breaking change to
// this library renames the witness to CompatV3, and every package
// generated against v2 stops compiling with the skew named, instead of
// misbehaving at run time.
func CompatV2() {}

// Suite is a set of checks. Every method returns a copy; the zero value
// is empty and Run refuses it.
//
// DropHint, when set by the generated package, renders an ID the way a
// consumer should WRITE it — the typed index path
// (StoreSuite.Checks.Model.TTLExpiry()) rather than a string literal.
// Every capability-failure message routes through it: the string form
// compiles too (ID is a string type), but it forfeits the
// compile-break-on-regeneration protection the index exists for, so the
// fix the runner prints must never teach it.
type Suite[S any] struct {
	// DropHint renders an ID as the index expression a drop should be
	// written with. Nil falls back to Without("<id>").
	DropHint func(ID) string

	// Name heads the report; conventionally "<package>.<Interface>".
	Name string
	// Checks is the assertion inventory, in emission order. The manifest
	// and the typed index are both projections of this slice.
	Checks []Check[S]

	// dropped records what Without asked for. Run fails when an entry
	// names no check, so a typo in a hand-written drop is caught rather
	// than silently keeping the check.
	dropped map[ID]bool
}

// With adds checks. Run fails when an added ID is already present.
func (s Suite[S]) With(extra ...Check[S]) Suite[S] {
	out := s.clone()
	out.Checks = append(out.Checks, extra...)
	return out
}

// Without marks checks as dropped. They stay in the suite so the report
// can name them and so Run can tell a real drop from a typo.
func (s Suite[S]) Without(ids ...ID) Suite[S] {
	out := s.clone()
	if out.dropped == nil {
		out.dropped = make(map[ID]bool, len(ids))
	}
	for _, id := range ids {
		out.dropped[id] = true
	}
	return out
}

// Dropped reports whether an ID was dropped.
func (s Suite[S]) Dropped(id ID) bool { return s.dropped[id] }

// IDs lists every check's ID in emission order, dropped ones included: the
// set of names this suite answers to, which is what a parity gate compares
// an index or a manifest against.
func (s Suite[S]) IDs() []ID {
	ids := make([]ID, 0, len(s.Checks))
	for _, c := range s.Checks {
		ids = append(ids, c.ID)
	}
	return ids
}

func (s Suite[S]) clone() Suite[S] {
	out := Suite[S]{Name: s.Name, Checks: make([]Check[S], len(s.Checks))}
	copy(out.Checks, s.Checks)
	// The copy above shares each check's Needs map; a caller mutating a
	// clone's capability gate must not reach through to every other
	// clone, so the maps are copied too.
	for i, c := range out.Checks {
		if c.Needs == nil {
			continue
		}
		needs := make(Caps, len(c.Needs))
		maps.Copy(needs, c.Needs)
		out.Checks[i].Needs = needs
	}
	if s.dropped != nil {
		out.dropped = make(map[ID]bool, len(s.dropped))
		for id := range s.dropped {
			out.dropped[id] = true
		}
	}
	return out
}

// Falsifiability is what is known about a check's ability to fail.
//
// Three states, and the middle one is not a weaker pass. A check can be
// argued unprovable for reasons that are facts about the claim rather than
// about the check: no defect the harness can produce expresses what the law
// is named for, no defect reaches the law's own observation point, or the
// claim needs a value only the consumer has. Each of those is a gap somewhere
// specific, and telling them apart is what makes the gap actionable.
type Falsifiability struct {
	// State is proven, argued or unproven. The zero value is unproven,
	// which is the honest default for a hand-written check nobody has
	// driven against a broken subject.
	State FalsifiableState

	// Why carries the argument for an argued check, empty otherwise.
	Why string
}

// FalsifiableState is how a check's ability to fail has been established.
type FalsifiableState string

const (
	// FalsifiableUnproven is the zero value: nothing has shown this check
	// able to fail, and nothing has argued that it cannot be shown.
	FalsifiableUnproven FalsifiableState = "unproven"

	// FalsifiableProven means a defect was constructed that this check
	// caught — the falsification companion for a generated assertion, the
	// saturation prover for a bound law.
	FalsifiableProven FalsifiableState = "proven"

	// FalsifiableArgued means the check cannot be shown able to fail and the
	// reason is recorded rather than assumed.
	FalsifiableArgued FalsifiableState = "argued"
)

// Proven marks a check as shown able to fail.
func Proven() Falsifiability { return Falsifiability{State: FalsifiableProven} }

// Argued marks a check nothing can currently falsify, with the reason.
func Argued(why string) Falsifiability {
	return Falsifiability{State: FalsifiableArgued, Why: why}
}
