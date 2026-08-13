// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package answeringwriter is the detector-axis fixture for the answeringwriter
// shape: one value in, the same type out beside the error — a write that
// answers the stored state it produced.
//
// The boundary is type identity on a named type. A basic-typed coincidence —
// a string-keyed read answering a string — stays with the reader, because
// the answered state this shape exists for is a struct carrying identity and
// stamps, and the corpus proved the wider rule reclassified transformers and
// caches across twenty-one fixtures.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package answeringwriter

import (
	"context"
)

// Value is the stored form the write answers — the stamped state, which is
// what separates this shape from a transformer.
type Value struct{ Key, Body string }

// AnsweringWriter is the fixture interface.
//
//testkit:out answeringwritertest/ pkg=answeringwritertest
//testkit:stub
//testkit:suite
//testkit:model
type AnsweringWriter interface {
	// Put stores the value and answers the stored state — the same named
	// type in and out, which is the identity the detector draws on.
	Put(ctx context.Context, v Value) (Value, error)
}
