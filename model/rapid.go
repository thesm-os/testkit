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

// IntRange creates a generator for int values in [min, max].
func IntRange(min, max int) *Generator[int] {
	return rapid.IntRange(min, max)
}

// Int8 creates a generator for all int8 values.
func Int8() *Generator[int8] { return rapid.Int8() }

// Int16Range creates a generator for int16 values in [min, max].
func Int16Range(min, max int16) *Generator[int16] {
	return rapid.Int16Range(min, max)
}

// Int32Range creates a generator for int32 values in [min, max].
func Int32Range(min, max int32) *Generator[int32] {
	return rapid.Int32Range(min, max)
}

// Int64Range creates a generator for int64 values in [min, max].
func Int64Range(min, max int64) *Generator[int64] {
	return rapid.Int64Range(min, max)
}

// UintRange creates a generator for uint values in [min, max].
func UintRange(min, max uint) *Generator[uint] {
	return rapid.UintRange(min, max)
}

// Uint8 creates a generator for all uint8 values.
func Uint8() *Generator[uint8] { return rapid.Uint8() }

// Uint16Range creates a generator for uint16 values in [min, max].
func Uint16Range(min, max uint16) *Generator[uint16] {
	return rapid.Uint16Range(min, max)
}

// Uint32Range creates a generator for uint32 values in [min, max].
func Uint32Range(min, max uint32) *Generator[uint32] {
	return rapid.Uint32Range(min, max)
}

// Uint64Range creates a generator for uint64 values in [min, max].
func Uint64Range(min, max uint64) *Generator[uint64] {
	return rapid.Uint64Range(min, max)
}

// Float32Range creates a generator for float32 values in [min, max].
func Float32Range(min, max float32) *Generator[float32] {
	return rapid.Float32Range(min, max)
}

// Float64Range creates a generator for float64 values in [min, max].
func Float64Range(min, max float64) *Generator[float64] {
	return rapid.Float64Range(min, max)
}

// StringMatching creates a generator for strings matching a regex.
func StringMatching(expr string) *Generator[string] {
	return rapid.StringMatching(expr)
}
