// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package basic

// Minimal is a tiny interface with only void-no-ctx methods. Used
// by stub tests to exercise the nil-return paths of Data helpers
// (FirstContextMethod, FirstErrorMethod, FirstNonSkipMethodWithSampleableResults).
type Minimal interface {
	Reset()
	Clear()
}
