// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package compositewriter is the detector-axis fixture for the compositewriter shape:
// a value in, the stored value out. The caller cannot construct the return
// because the implementation assigns part of it.
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
type CompositeWriter interface {
	Store(ctx context.Context, v Value) (Value, error)
}
