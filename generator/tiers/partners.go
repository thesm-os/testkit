// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package tiers

// PartnerDriven reports whether a method referenced as the named mixin's
// sibling parameter may still be driven as an ordinary action, and where it
// may not, why.
//
// A sibling reference means one of two things, and shape cannot tell them
// apart. Most name an ordinary method the law needs a handle on — `ttl.put`
// names a put that is a writer, `deleteremoves.read` a read that is a reader —
// and excluding those empties the sequences of the very methods the fixture
// exists to drive. A minority name a method whose role overrides its shape:
// `validates.fn` is a validator that merely looks like a writer, and driving
// it as one corrupts the reference with stores the subject never made;
// `tamperevident.tamper` and `poisonable.induce` corrupt or poison the
// subject one side at a time. Those stay out of the sequences and inert in
// the adapter, and the reason is what the generated header prints.
//
// Data because the difference is semantic, here because this is the module
// the census closes: every sibling parameter eidos registers must carry a
// verdict in exactly one of the two tables, so a mixin landing upstream is
// classified in the same build that makes it stampable. An unclassified pair
// is treated as excluded — undriven and inert together keeps the pair
// synchronized — and the census is what keeps that arm theoretical.
func PartnerDriven(mixin, param string) (bool, string) {
	key := mixin + "." + param
	reason, excluded := excludedPartners[key]
	return partnerVerdict(drivenPartners[key], excluded, reason)
}

// partnerVerdict resolves the two table lookups into the one answer, including
// the answer for a pair both tables claim.
//
// Split out so all four inputs are reachable from a test. The conflict arm is
// unreachable through [PartnerDriven] by construction — the census fails a
// build where any pair is in both tables — and an arm nothing can reach is an
// arm nothing checks, which is the shape of defect this whole item is about.
//
// Neither table wins a conflict. Whichever did would be a verdict decided by
// the order of two `if`s rather than by anyone, and this pair spent a release
// in exactly that state: the driven rows won, their exclusion reasons went
// unread, and by the time anyone read them they described a clock the clocked
// mode had made moot. So the answer is excluded, which is the safe side, and
// the reason says the tables disagree — a sentence a reader can act on, where
// silently taking one side is not.
func partnerVerdict(driven, excluded bool, reason string) (bool, string) {
	switch {
	case driven && excluded:
		return false, "listed as both driven and excluded, which is a defect in the partner tables"
	case excluded:
		return false, reason
	case driven:
		return true, ""
	default:
		return false, "an unclassified sibling reference, which defaults to excluded"
	}
}

// PartnerClassified reports whether the pair carries a verdict in exactly one
// table — the census's question, exported so the gate can hold the tables both
// total and disjoint.
//
// Exactly one rather than at least one. Totality alone cannot see a pair listed
// twice, and a census that cannot see its own conflict is how two dead rows
// survived: they satisfied the question being asked, and the question was the
// wrong one.
func PartnerClassified(mixin, param string) bool {
	key := mixin + "." + param
	_, excluded := excludedPartners[key]
	return drivenPartners[key] != excluded
}

// drivenPartners marks the sibling references that name ordinary methods.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var drivenPartners = map[string]bool{
	mixinDeleteRemoves + ".read": true,
	// The sizing method is an ordinary read that happens to bound an index.
	// Excluding it would drop a real observation from the sequences on the
	// strength of a second method naming it — and a subject whose size no
	// sequence ever reads is one whose size nothing checks.
	mixinIndexed + ".by":            true,
	mixinPartition + ".read":        true,
	mixinReadAfterWrite + ".write":  true,
	mixinSideEffect + ".observe":    true,
	mixinStreamReflects + ".mutate": true,
	mixinStreamReflects + ".delete": true,
	mixinTTL + ".put":               true,
	mixinTTL + ".read":              true,
	mixinWindowed + ".incr":         true,
	mixinWindowed + ".count":        true,
	// Both were listed as excluded too, on the grounds that the sequences
	// never advance the clock. The sequences still do not — the
	// scheduled-fires law advances it inside its own Check — but excluding
	// the pair leaves this fixture with no action at all, and the model
	// generator refuses an interface whose sequences would drive nothing. The
	// exclusion would have deleted the tier that states the claim.
	mixinScheduled + ".schedule": true,
	mixinScheduled + ".fired":    true,
}

// excludedPartners marks the references whose role overrides their shape,
// each with the reason the generated header prints.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var excludedPartners = map[string]string{
	mixinValidates + ".fn":         "a validator, whose call proves nothing its smoke check does not",
	mixinOrderAfter + ".fn":        "an ordering constraint, not an operation to drive at random",
	mixinWrappedVia + ".fn":        "the wrapper reference, not an operation of the subject's own",
	mixinSample + ".builder":       "an input builder, not an operation of the subject's own",
	mixinHooks + ".register":       "a callback registrar, which a random sequence has nothing to hand",
	mixinPoisonable + ".induce":    "the poison inducer, which would kill one side of the pair",
	mixinTamperEvident + ".tamper": "the corruptor, which would tamper one side of the pair",
	mixinTamperEvident + ".verify": "the integrity check, whose claims the tamper law owns",
	mixinLeakFree + ".open":        "half of a cycle the sequences cannot keep balanced",
	mixinLeakFree + ".close":       "half of a cycle the sequences cannot keep balanced",
	mixinLifecycleAfter + ".close": "a close, which would end one side of the pair mid-sequence",
	mixinEventually + ".settle":    "a convergence forcer the derived reference cannot mirror",
	mixinEventually + ".sync":      "a convergence forcer the derived reference cannot mirror",
}
