// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package action

// ReaderWithBoolOutput packs the (V, bool) return from a
// ReaderWithBool-shaped method for trace recording.
type ReaderWithBoolOutput struct {
	V  any
	OK bool
}

// LookupOutput packs the (R1, R2, bool) return from a
// Lookup-shaped method for trace recording.
type LookupOutput struct {
	R1 any
	R2 any
	OK bool
}
