// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package testkit

import (
	"pgregory.net/rapid"
)

// DrawString generates a random string with the given prefix using [rapid].
// The prefix is prepended to a random alphanumeric suffix, producing
// deterministic-per-seed values suitable for property-based tests.
//
//	name := testkit.DrawString(t, "user")  // e.g. "user-a7f3b2"
func DrawString(t *rapid.T, prefix string) string {
	suffix := rapid.StringMatching(`[a-z0-9]{6}`).Draw(t, prefix+"-suffix")
	return prefix + "-" + suffix
}

// DrawBytes generates a random byte slice of up to maxLen bytes.
//
//	data := testkit.DrawBytes(t, 256)
func DrawBytes(t *rapid.T, maxLen int) []byte {
	n := rapid.IntRange(0, maxLen).Draw(t, "len")
	b := make([]byte, n)
	for i := range n {
		b[i] = byte(rapid.IntRange(0, 255).Draw(t, "byte"))
	}
	return b
}

// DrawEnum generates a random enum value in [0, max]. Use this for enum types
// represented as integer constants.
//
//	status := testkit.DrawEnum[Status](t, StatusMax)
func DrawEnum[T ~uint8 | ~uint16 | ~uint32 | ~int8 | ~int16 | ~int32 | ~int](t *rapid.T, upper T) T {
	return T(rapid.IntRange(int(0), int(upper)).Draw(t, "enum"))
}

// DrawUint64 generates a random uint64 value.
func DrawUint64(t *rapid.T) uint64 {
	return rapid.Uint64().Draw(t, "uint64")
}

// DrawInt generates a random int in [lo, hi].
func DrawInt(t *rapid.T, lo, hi int) int {
	return rapid.IntRange(lo, hi).Draw(t, "int")
}
