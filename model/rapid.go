// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

// Re-exports of pgregory.net/rapid types used by generated code.
// Generated templates import model instead of rapid directly,
// keeping rapid out of consumer go.mod files.

import (
	"testing"

	"pgregory.net/rapid"
)

// T is a rapid property test state. Alias so generated code
// references model.T instead of rapid.T.
type T = rapid.T

// TB is the test interface accepted by Check. Alias so generated
// code references model.TB instead of rapid.TB.
type TB = rapid.TB

// Generator is a typed value generator. Alias so generated code
// references model.Generator instead of rapid.Generator.
type Generator[V any] = rapid.Generator[V]

// MakeConfig configures MakeCustom generators. Alias so generated
// code references model.MakeConfig instead of rapid.MakeConfig.
type MakeConfig = rapid.MakeConfig

// Check runs a rapid property test.
func Check(t TB, prop func(*T)) {
	rapid.Check(t, prop)
}

// MakeFuzz converts a rapid property into a go test -fuzz target.
func MakeFuzz(prop func(*T)) func(*testing.T, []byte) {
	return rapid.MakeFuzz(prop)
}

// SampledFrom creates a generator that draws uniformly from the
// given values.
func SampledFrom[V any](vs []V) *Generator[V] {
	return rapid.SampledFrom(vs)
}

// Make creates a generator that produces values of type V via
// reflection.
func Make[V any]() *Generator[V] {
	return rapid.Make[V]()
}

// MakeCustom creates a generator with field-level overrides.
func MakeCustom[V any](cfg MakeConfig) *Generator[V] {
	return rapid.MakeCustom[V](cfg)
}

// IntRange creates a generator for int values in [lo, hi].
func IntRange(lo, hi int) *Generator[int] {
	return rapid.IntRange(lo, hi)
}

// Int8 creates a generator for all int8 values.
func Int8() *Generator[int8] { return rapid.Int8() }

// Int16Range creates a generator for int16 values in [lo, hi].
func Int16Range(lo, hi int16) *Generator[int16] {
	return rapid.Int16Range(lo, hi)
}

// Int32Range creates a generator for int32 values in [lo, hi].
func Int32Range(lo, hi int32) *Generator[int32] {
	return rapid.Int32Range(lo, hi)
}

// Int64Range creates a generator for int64 values in [lo, hi].
func Int64Range(lo, hi int64) *Generator[int64] {
	return rapid.Int64Range(lo, hi)
}

// UintRange creates a generator for uint values in [lo, hi].
func UintRange(lo, hi uint) *Generator[uint] {
	return rapid.UintRange(lo, hi)
}

// Uint8 creates a generator for all uint8 values.
func Uint8() *Generator[uint8] { return rapid.Uint8() }

// Uint16Range creates a generator for uint16 values in [lo, hi].
func Uint16Range(lo, hi uint16) *Generator[uint16] {
	return rapid.Uint16Range(lo, hi)
}

// Uint32Range creates a generator for uint32 values in [lo, hi].
func Uint32Range(lo, hi uint32) *Generator[uint32] {
	return rapid.Uint32Range(lo, hi)
}

// Uint64Range creates a generator for uint64 values in [lo, hi].
func Uint64Range(lo, hi uint64) *Generator[uint64] {
	return rapid.Uint64Range(lo, hi)
}

// Float32Range creates a generator for float32 values in [lo, hi].
func Float32Range(lo, hi float32) *Generator[float32] {
	return rapid.Float32Range(lo, hi)
}

// Float64Range creates a generator for float64 values in [lo, hi].
func Float64Range(lo, hi float64) *Generator[float64] {
	return rapid.Float64Range(lo, hi)
}

// StringMatching creates a generator for strings matching a regex.
func StringMatching(expr string) *Generator[string] {
	return rapid.StringMatching(expr)
}
