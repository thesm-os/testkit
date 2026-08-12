// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package batchreader is the detector-axis fixture for the batchreader shape:
// many keys in one call. The variadic key list is the marker, and the
// returned slice need not match the request length — absent keys are simply absent.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package batchreader

import (
	"context"
)

// Value is the payload the fixture reads.
type Value struct{ Key, Body string }

// BatchReader is the fixture interface.
//
//testkit:out batchreadertest/ pkg=batchreadertest
//testkit:stub
//testkit:suite
//testkit:model
type BatchReader interface {
	GetAll(ctx context.Context, keys ...string) ([]Value, error)
}
