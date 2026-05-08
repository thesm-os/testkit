// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive

// Category groups directives by what they *do* in the pipeline. The
// category drives doc-gen layout, validation messaging ("you put a
// Mixin where a ContractTier was expected"), and IDE tooling (autocomplete
// per category).
//
// Categories are orthogonal to [Phase] (which describes rollout
// timing): a Phase 1 Mixin and a Phase 5 Mixin are both Mixin
// directives.
type Category int

// Recognized categories. CategoryUnspecified is the zero value and
// indicates a descriptor whose category was not set; the validator
// flags it as a programmer error at registration time.
const (
	CategoryUnspecified Category = iota

	// SignatureHint nudges shape detection (e.g. //testkit:deleter
	// elevates a Writer signature to Deleter; //testkit:keyfield
	// names the key field for reference synthesis). Detected by the
	// shape package, not by an enricher or emitter.
	SignatureHint

	// ContractTier introduces a directive-triggered base shape
	// (Persister, CompareAndSwap, Appender, ...). Signature-shared
	// with a base shape but carries distinct invariants and emits
	// distinct primitives.
	ContractTier

	// Mixin is an orthogonal invariant that composes with any base
	// shape (atomic, idempotent, conservative, bounded, roundtrip,
	// ...). One mixin per directive; emission is independent of
	// other mixins on the same method.
	Mixin

	// Enrichment populates a generator's per-method data model with
	// information drawn from the directive (errors → fault helpers,
	// sample → bench input replacement, deprecated → //Deprecated:
	// banner). Consumed by [Enricher] functions registered in
	// [ConsumerRegistry].
	Enrichment

	// Documentation does not affect generated code; it surfaces in
	// spec headers, godoc, and audit reports. //testkit:req for
	// requirement traceability is the canonical example.
	Documentation
)

// String returns the canonical category name. Used in error messages
// and doc-gen.
func (c Category) String() string {
	switch c {
	case SignatureHint:
		return "SignatureHint"
	case ContractTier:
		return "ContractTier"
	case Mixin:
		return "Mixin"
	case Enrichment:
		return "Enrichment"
	case Documentation:
		return "Documentation"
	default:
		return "Unspecified"
	}
}
