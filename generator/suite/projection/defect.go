// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package projection

// DefectKind names a defect variant; the value is the variant's
// template name, composed from the one dispatch prefix.
type DefectKind string

// DefectKindPrefix namespaces defect templates in the dispatch table.
const DefectKindPrefix = "defect."

// The defect kinds, one per variant below.
const (
	KindStubPanic       DefectKind = DefectKindPrefix + "stub-panic"
	KindCtxSwap         DefectKind = DefectKindPrefix + "ctx-swap"
	KindAcceptsNil      DefectKind = DefectKindPrefix + "accepts-nil"
	KindDiscardWrite    DefectKind = DefectKindPrefix + "discard-write"
	KindFreezeReturn    DefectKind = DefectKindPrefix + "freeze-return"
	KindFreshMedium     DefectKind = DefectKindPrefix + "fresh-medium"
	KindSentinelOnce    DefectKind = DefectKindPrefix + "sentinel-once"
	KindPartialOutlive  DefectKind = DefectKindPrefix + "partial-outlive"
	KindExceedBound     DefectKind = DefectKindPrefix + "exceed-bound"
	KindEchoBesideError DefectKind = DefectKindPrefix + "echo-beside-error"
	KindSecondCallErrs  DefectKind = DefectKindPrefix + "second-call-errs"
	KindInventsHit      DefectKind = DefectKindPrefix + "invents-hit"
	KindSwapsValues     DefectKind = DefectKindPrefix + "swaps-values"
)

// The defect variants: one per proofs rule in derivation-rules.md.
// Each is a small plan over the generated double (or a hand shape
// where the double cannot express the defect), and every Proven check
// carries exactly one.

// StubPanic is the smoke family's defect: the named method panics.
type StubPanic struct{ Option Option }

// CtxSwap ignores the caller's context — the cancel/deadline
// families' defect.
type CtxSwap struct{ Option Option }

// AcceptsNil forgives a nil context and answers — the nilcontext
// claim's "returns an error" arm.
type AcceptsNil struct{ Option Option }

// DiscardWrite acknowledges a write and drops it.
type DiscardWrite struct{ Option Option }

// FreezeReturn pins a monotonic return to a constant.
type FreezeReturn struct{ Option Option }

// FreshMedium recovers onto a new empty medium.
type FreshMedium struct{}

// SentinelOnce reports the sentinel once, then heals.
type SentinelOnce struct{ Sentinel Expr }

// PartialOutlive keeps exactly one stamped method alive after Close —
// the defect that forces multi-probe lifecycle laws to stay plural.
type PartialOutlive struct{ Option Option }

// ExceedBound reports an accounting number past the declared limit.
type ExceedBound struct{ Option Option }

// EchoBesideError returns a live value beside a non-nil error — the
// zero-on-error family's defect.
type EchoBesideError struct{ Option Option }

// SecondCallErrs succeeds once and errors on the repeat — the
// idempotent claim's defect.
type SecondCallErrs struct{ Option Option }

// InventsHit answers for an input nothing supplied — the reader miss
// claim's defect.
type InventsHit struct{ Option Option }

// SwapsValues answers a hit with another entry's value — the seeded
// hit claim's defect.
type SwapsValues struct{ Option Option }

// DefectKind names the template that plants a panicking stub.
func (StubPanic) DefectKind() DefectKind { return KindStubPanic }

// DefectKind names the template that plants a swapped context.
func (CtxSwap) DefectKind() DefectKind { return KindCtxSwap }

// DefectKind names the template that plants a subject that accepts nil.
func (AcceptsNil) DefectKind() DefectKind { return KindAcceptsNil }

// DefectKind names the template that plants a discarded write.
func (DiscardWrite) DefectKind() DefectKind { return KindDiscardWrite }

// DefectKind names the template that plants a frozen return.
func (FreezeReturn) DefectKind() DefectKind { return KindFreezeReturn }

// DefectKind names the template that plants a fresh medium.
func (FreshMedium) DefectKind() DefectKind { return KindFreshMedium }

// DefectKind names the template that plants a sentinel reported once.
func (SentinelOnce) DefectKind() DefectKind { return KindSentinelOnce }

// DefectKind names the template that plants a partial that outlives its close.
func (PartialOutlive) DefectKind() DefectKind { return KindPartialOutlive }

// DefectKind names the template that plants an exceeded bound.
func (ExceedBound) DefectKind() DefectKind { return KindExceedBound }

// DefectKind names the template that plants an echo beside the error.
func (EchoBesideError) DefectKind() DefectKind { return KindEchoBesideError }

// DefectKind names the template that plants a second call that errors.
func (SecondCallErrs) DefectKind() DefectKind { return KindSecondCallErrs }

// DefectKind names the template that plants an invented hit.
func (InventsHit) DefectKind() DefectKind { return KindInventsHit }

// DefectKind names the template that plants swapped values.
func (SwapsValues) DefectKind() DefectKind { return KindSwapsValues }

// DefectKinds enumerates every registered defect variant, for the
// template census.
func DefectKinds() []DefectKind {
	return []DefectKind{
		StubPanic{}.DefectKind(),
		CtxSwap{}.DefectKind(),
		AcceptsNil{}.DefectKind(),
		DiscardWrite{}.DefectKind(),
		FreezeReturn{}.DefectKind(),
		FreshMedium{}.DefectKind(),
		SentinelOnce{}.DefectKind(),
		PartialOutlive{}.DefectKind(),
		ExceedBound{}.DefectKind(),
		EchoBesideError{}.DefectKind(),
		SecondCallErrs{}.DefectKind(),
		InventsHit{}.DefectKind(),
		SwapsValues{}.DefectKind(),
	}
}
