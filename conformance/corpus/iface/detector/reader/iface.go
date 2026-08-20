// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package reader is the detector-axis fixture for the reader shape:
// a single-key fetch that can fail — the commonest shape in any
// store-like interface, and what every miss-sentinel law attaches to.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package reader

import (
	"context"
	"errors"
)

// ErrNotFound is the miss sentinel this fixture's reads report.
var ErrNotFound = errors.New("reader: not found")

// Value is the payload the fixture reads.
type Value struct{ Key, Body string }

// Reader is the fixture interface.
//
//testkit:out readertest/ pkg=readertest
//testkit:stub
//testkit:suite
//testkit:model
type Reader interface {
	//testkit:mixin notfound sentinel=ErrNotFound
	Get(ctx context.Context, key string) (Value, error)
}
