// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package multiargwriter is the detector-axis fixture for the multiargwriter shape:
// several parameters rather than one composite value. The arity is the
// distinguishing feature.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package multiargwriter

import (
	"context"
)

// Value is the stored form.
type Value struct{ Key, Body string }

// MultiArgWriter is the fixture interface.
//
//testkit:out multiargwritertest/ pkg=multiargwritertest
//testkit:stub
//testkit:suite
//testkit:model
type MultiArgWriter interface {
	Set(ctx context.Context, key, body string) error
}
