// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package readernoerror is the detector-axis fixture for the readernoerror shape:
// an infallible fetch. Having no error return is the whole distinction
// from a reader: a miss has to be carried by the value, so it is the zero value.
//
// No directive appears here: the classification comes from the signature
// alone, so a detector that misfires shows up as wrong generated output
// rather than as a directive that was misread.
package readernoerror

import (
	"context"
)

// Value is the payload the fixture reads.
type Value struct{ Key, Body string }

// ReaderNoError is the fixture interface.
//
//testkit:out readernoerrortest/ pkg=readernoerrortest
//testkit:stub
type ReaderNoError interface {
	Lookup(ctx context.Context, key string) Value
}
