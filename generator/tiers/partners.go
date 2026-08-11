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
	if drivenPartners[key] {
		return true, ""
	}
	if reason, classified := excludedPartners[key]; classified {
		return false, reason
	}
	return false, "an unclassified sibling reference, which defaults to excluded"
}

// PartnerClassified reports whether the pair carries a verdict at all — the
// census's question, exported so the gate can hold the tables total.
func PartnerClassified(mixin, param string) bool {
	key := mixin + "." + param
	_, excluded := excludedPartners[key]
	return drivenPartners[key] || excluded
}

// drivenPartners marks the sibling references that name ordinary methods.
//
//nolint:gochecknoglobals // a lookup table, read-only after init.
var drivenPartners = map[string]bool{
	mixinDeleteRemoves + ".read":    true,
	mixinPartition + ".read":        true,
	mixinReadAfterWrite + ".write":  true,
	mixinSideEffect + ".observe":    true,
	mixinStreamReflects + ".mutate": true,
	mixinStreamReflects + ".delete": true,
	mixinTTL + ".put":               true,
	mixinTTL + ".read":              true,
	mixinWindowed + ".incr":         true,
	mixinWindowed + ".count":        true,
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
	mixinScheduled + ".schedule":   "schedules against a clock the sequences never advance",
	mixinScheduled + ".fired":      "counts firings of a clock the sequences never advance",
}
