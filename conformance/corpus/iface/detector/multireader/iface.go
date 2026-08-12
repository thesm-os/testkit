// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package multireader is the detector-axis fixture for the multireader shape:
// one key in, two values and an error out — the failable counterpart of a
// lookup.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package multireader

import (
	"context"
	"errors"
)

// ErrNotFound is the miss sentinel this fixture's reads report.
var ErrNotFound = errors.New("multireader: not found")

// Value is the primary return.
type Value struct{ Key, Body string }

// Meta is the secondary return.
type Meta struct{ Revision int }

// MultiReader is the fixture interface.
//
//testkit:out multireadertest/ pkg=multireadertest
//testkit:stub
//testkit:suite
//testkit:model
type MultiReader interface {
	GetWithMeta(ctx context.Context, key string) (Value, Meta, error)
}
