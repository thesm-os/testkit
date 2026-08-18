// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package legs holds the engine-facing leg idioms every generated
// conformance package shares: the oracle-or-derived reference pick, the
// law leg, the differential leg, the provenance-gated adversarial
// blend, and the vacuity note.
//
// It exists because of a doctrine and a duplication meeting head-on.
// The doctrine: package suite imports testing and clock and NOTHING
// else, so a consumer's non-test code composes with the vocabulary
// without pulling the model tier's dependency graph. The duplication:
// with suite unable to see the engine, every generated package
// re-emitted the same six idioms as private functions, and five copies
// of a semantics-bearing idiom drift where one cannot. This package is
// the sanctioned bridge: generated files import suite AND legs, legs
// imports the engine, and suite still does not.
package legs

import (
	"testing"

	"go.thesmos.sh/testkit/engine/model"
	"go.thesmos.sh/testkit/engine/model/law"
	"go.thesmos.sh/testkit/engine/suite"
)

// CompatV1 is the compatibility witness for this package's leg
// contract, referenced once per generated model file:
//
//	var _ = legs.CompatV1
//
// A library the generated files ride can drift from the files a plugin
// version emitted — the same skew CompatV2 guards on suite. A breaking
// change here renames the witness, and every package generated against
// v1 stops compiling with the skew named.
func CompatV1() {}

// Reference picks the reference a model leg compares against and names
// the tier for the report: the run's declared oracle when one exists,
// the given derived reference otherwise. The twin is not reachable from
// here at all, which is the point — a second copy of the subject agrees
// with it about every bug they share, so the derived reference replaced
// the twin fallback for generated legs.
func Reference[S any](tb testing.TB, sub suite.Subject[S], derived func() S) (func() S, suite.Tier) {
	tb.Helper()
	if build, differential := sub.Reference(); differential {
		return func() S { return build(tb) }, suite.TierDifferential
	}
	return derived, suite.TierDerived
}

// NoteVacuity is the one home of the vacuous-outcome idiom: a law
// behind a precondition the run never met passed nothing, and the leg
// says so instead of joining the passes. The engine counts engagement
// in its census; this hands the count to the leg note the report reads.
func NoteVacuity[S any](tb testing.TB, sub suite.Subject[S], out model.Outcome) {
	tb.Helper()
	if out.Engaged() {
		// Engaged means AT LEAST one law reached a verdict — a bundle
		// where one law fires and four never engage is engaged, and
		// counting it a plain pass masks the four. The leg stays green
		// (something was asserted); the report carries which laws were
		// not.
		if names := out.Unengaged(); len(names) > 0 {
			sub.NoteUnengaged(names)
			tb.Logf("laws that never engaged on any draw: %v", names)
		}
		return
	}
	sub.Note(suite.ReasonVacuous)
	// Unengaged is NOT guaranteed non-empty here: a law whose Check was
	// never called has Ran == 0 and appears in neither census list —
	// the engine's Outcome doc owns that contract. Naming an empty list
	// would say "[]" and teach nothing.
	if names := out.Unengaged(); len(names) > 0 {
		tb.Logf("no law engaged on any draw: %v", names)
	} else {
		tb.Logf("no law reached a single check: the drawn sequences never invoked one")
	}
}

// Law is the one body every law leg shares: the given laws as the run's
// only oracle over the leg's action stream, vacuity reported through
// the leg note. sut is explicit rather than derived from the subject
// because a clocked leg builds its instances on a per-iteration clock;
// extra carries leg-specific options — a history reset, say — without a
// second leg shape.
//
// LawsOnly is the A1 split: with the differential armed, it would catch
// every defect first and make "can this law fail" unanswerable.
func Law[S any](
	tb testing.TB, sub suite.Subject[S], sut, ref func() S,
	actions []model.Action[S], laws []law.Law[S], extra ...model.Option[S],
) {
	tb.Helper()
	opts := make([]model.Option[S], 0, 3+len(laws)+len(extra))
	opts = append(opts,
		model.WithReference(ref),
		model.WithActions(actions...),
		model.WithLawsOnly[S](true),
	)
	for _, l := range laws {
		opts = append(opts, model.WithLaw(l))
	}
	opts = append(opts, extra...)
	NoteVacuity(tb, sub, model.Assert(tb, sut, opts...))
}

// Differential is the differential leg: random action sequences against
// the subject and the reference [Reference] picks, the tier noted for
// the report, extra options appended for legs that carry more — a
// history reset, say.
func Differential[S any](
	tb testing.TB, sub suite.Subject[S], derived func() S,
	actions []model.Action[S], extra ...model.Option[S],
) {
	tb.Helper()
	buildRef, tier := Reference(tb, sub, derived)
	sub.NoteTier(tier)
	opts := make([]model.Option[S], 0, 2+len(extra))
	opts = append(opts,
		model.WithReference(buildRef),
		model.WithActions(actions...),
	)
	opts = append(opts, extra...)
	model.Assert(tb, func() S { return sub.New(tb) }, opts...)
}

// Blend is the provenance-gated adversarial widening: a DERIVED pool
// blends with the hostile half of the string space, and a pool the
// consumer RESTRICTED reaches every tier verbatim — a restricted pool
// is a statement about what the implementation accepts, and blending
// hostility past it would red correct code against inputs its owner
// ruled out.
func Blend[V any](derived bool, pool *model.Generator[V], hostile func(string) V) *model.Generator[V] {
	if !derived {
		return pool
	}
	return model.OneOf(pool, model.Map(model.AdversarialStrings(), hostile))
}
