// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package structs

import "time"

//go:generate testkit builder -o structstest/fields.gen.go Item

// Item exercises every field shape the builder generator branches
// on, plus a handful of "scalar but unusual" types so the
// With<Field> fallback path is exercised against more than the
// textbook string/int/bool triple.
//
// Coverage:
//   - scalar across kinds: string, int, bool, float64, time.Time
//     (imported scalar — drives ImportTracker integration)
//   - slice non-byte: []string (string element), []int (int element)
//   - []byte (special case: WithString convenience setter)
//   - map basic and map with non-string key
//   - pointer scalar
//   - one unexported field that must NOT receive a setter
type Item struct {
	ID      string
	Count   int
	Active  bool
	Ratio   float64
	Created time.Time

	Tags  []string
	Codes []int
	Data  []byte

	Metadata map[string]string
	Counters map[int]bool

	Owner *string

	hidden int //nolint:unused // fixture: setter must not be emitted
}
