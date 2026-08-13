// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package compositewriter is the detector-axis fixture for the compositewriter shape:
// a key beside a value, an error and nothing else — exactly two non-context
// parameters, the boundary the detector documents and draws.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package compositewriter

import (
	"context"
)

// Value is the stored form. Rev is assigned by the implementation, which is
// why the write has to return it.
type Value struct {
	Key  string
	Body string
	Rev  string
}

// CompositeWriter is the fixture interface.
//
//testkit:out compositewritertest/ pkg=compositewritertest
//testkit:stub
//testkit:suite
//testkit:model
type CompositeWriter interface {
	// Set is the documented shape whole: a key beside a value, an error and
	// nothing else — exactly two non-context parameters, which is the
	// boundary the detector draws.
	Set(ctx context.Context, key string, v Value) error
}
