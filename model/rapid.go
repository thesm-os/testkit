// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

// Re-exports of pgregory.net/rapid types and functions. Generated
// templates import model instead of rapid directly, keeping rapid
// out of consumer go.mod files. Consumers extending generated model
// specs also use these re-exports so rapid never leaks into their
// dependency graph.

import (
	"testing"
	"unicode"

	"pgregory.net/rapid"
)

// ─── Types ──────────────────────────────────────────────────────

// T is a rapid property test state.
type T = rapid.T

// TB is the test interface accepted by [Check].
type TB = rapid.TB

// Generator is a typed value generator.
type Generator[V any] = rapid.Generator[V]

// MakeConfig configures [MakeCustom] generators.
type MakeConfig = rapid.MakeConfig

// StateMachine is the interface for stateful property tests.
type StateMachine = rapid.StateMachine

// ─── Check / Fuzz ───────────────────────────────────────────────

// Check runs a rapid property test.
func Check(t TB, prop func(*T)) {
	rapid.Check(t, prop)
}

// MakeCheck converts a rapid property into a standalone test function.
func MakeCheck(prop func(*T)) func(*testing.T) {
	return rapid.MakeCheck(prop)
}

// MakeFuzz converts a rapid property into a go test -fuzz target.
func MakeFuzz(prop func(*T)) func(*testing.T, []byte) {
	return rapid.MakeFuzz(prop)
}

// SyncTest runs prop inside T.Helper concurrency control.
func SyncTest(t *T, prop func(*T)) {
	rapid.SyncTest(t, prop)
}

// StateMachineActions extracts named actions from a [StateMachine].
func StateMachineActions(sm StateMachine) map[string]func(*T) {
	return rapid.StateMachineActions(sm)
}

// ID is the identity function — useful as a key function.
func ID[V any](v V) V {
	return rapid.ID(v)
}

// ─── Bool ───────────────────────────────────────────────────────

// Bool creates a generator for bool values.
func Bool() *Generator[bool] { return rapid.Bool() }

// ─── Byte ───────────────────────────────────────────────────────

// Byte creates a generator for all byte values.
func Byte() *Generator[byte] { return rapid.Byte() }

// ByteMin creates a generator for byte values >= lo.
func ByteMin(lo byte) *Generator[byte] { return rapid.ByteMin(lo) }

// ByteMax creates a generator for byte values <= hi.
func ByteMax(hi byte) *Generator[byte] { return rapid.ByteMax(hi) }

// ByteRange creates a generator for byte values in [lo, hi].
func ByteRange(lo, hi byte) *Generator[byte] { return rapid.ByteRange(lo, hi) }

// ─── Rune ───────────────────────────────────────────────────────

// Rune creates a generator for rune values.
func Rune() *Generator[rune] { return rapid.Rune() }

// RuneFrom creates a generator for runes from the given set.
func RuneFrom(runes []rune, tables ...*unicode.RangeTable) *Generator[rune] {
	return rapid.RuneFrom(runes, tables...)
}

// ─── Int ────────────────────────────────────────────────────────

// Int creates a generator for all int values.
func Int() *Generator[int] { return rapid.Int() }

// IntMin creates a generator for int values >= lo.
func IntMin(lo int) *Generator[int] { return rapid.IntMin(lo) }

// IntMax creates a generator for int values <= hi.
func IntMax(hi int) *Generator[int] { return rapid.IntMax(hi) }

// IntRange creates a generator for int values in [lo, hi].
func IntRange(lo, hi int) *Generator[int] { return rapid.IntRange(lo, hi) }

// ─── Int8 ───────────────────────────────────────────────────────

// Int8 creates a generator for all int8 values.
func Int8() *Generator[int8] { return rapid.Int8() }

// Int8Min creates a generator for int8 values >= lo.
func Int8Min(lo int8) *Generator[int8] { return rapid.Int8Min(lo) }

// Int8Max creates a generator for int8 values <= hi.
func Int8Max(hi int8) *Generator[int8] { return rapid.Int8Max(hi) }

// Int8Range creates a generator for int8 values in [lo, hi].
func Int8Range(lo, hi int8) *Generator[int8] { return rapid.Int8Range(lo, hi) }

// ─── Int16 ──────────────────────────────────────────────────────

// Int16 creates a generator for all int16 values.
func Int16() *Generator[int16] { return rapid.Int16() }

// Int16Min creates a generator for int16 values >= lo.
func Int16Min(lo int16) *Generator[int16] { return rapid.Int16Min(lo) }

// Int16Max creates a generator for int16 values <= hi.
func Int16Max(hi int16) *Generator[int16] { return rapid.Int16Max(hi) }

// Int16Range creates a generator for int16 values in [lo, hi].
func Int16Range(lo, hi int16) *Generator[int16] { return rapid.Int16Range(lo, hi) }

// ─── Int32 ──────────────────────────────────────────────────────

// Int32 creates a generator for all int32 values.
func Int32() *Generator[int32] { return rapid.Int32() }

// Int32Min creates a generator for int32 values >= lo.
func Int32Min(lo int32) *Generator[int32] { return rapid.Int32Min(lo) }

// Int32Max creates a generator for int32 values <= hi.
func Int32Max(hi int32) *Generator[int32] { return rapid.Int32Max(hi) }

// Int32Range creates a generator for int32 values in [lo, hi].
func Int32Range(lo, hi int32) *Generator[int32] { return rapid.Int32Range(lo, hi) }

// ─── Int64 ──────────────────────────────────────────────────────

// Int64 creates a generator for all int64 values.
func Int64() *Generator[int64] { return rapid.Int64() }

// Int64Min creates a generator for int64 values >= lo.
func Int64Min(lo int64) *Generator[int64] { return rapid.Int64Min(lo) }

// Int64Max creates a generator for int64 values <= hi.
func Int64Max(hi int64) *Generator[int64] { return rapid.Int64Max(hi) }

// Int64Range creates a generator for int64 values in [lo, hi].
func Int64Range(lo, hi int64) *Generator[int64] { return rapid.Int64Range(lo, hi) }

// ─── Uint ───────────────────────────────────────────────────────

// Uint creates a generator for all uint values.
func Uint() *Generator[uint] { return rapid.Uint() }

// UintMin creates a generator for uint values >= lo.
func UintMin(lo uint) *Generator[uint] { return rapid.UintMin(lo) }

// UintMax creates a generator for uint values <= hi.
func UintMax(hi uint) *Generator[uint] { return rapid.UintMax(hi) }

// UintRange creates a generator for uint values in [lo, hi].
func UintRange(lo, hi uint) *Generator[uint] { return rapid.UintRange(lo, hi) }

// ─── Uint8 ──────────────────────────────────────────────────────

// Uint8 creates a generator for all uint8 values.
func Uint8() *Generator[uint8] { return rapid.Uint8() }

// Uint8Min creates a generator for uint8 values >= lo.
func Uint8Min(lo uint8) *Generator[uint8] { return rapid.Uint8Min(lo) }

// Uint8Max creates a generator for uint8 values <= hi.
func Uint8Max(hi uint8) *Generator[uint8] { return rapid.Uint8Max(hi) }

// Uint8Range creates a generator for uint8 values in [lo, hi].
func Uint8Range(lo, hi uint8) *Generator[uint8] { return rapid.Uint8Range(lo, hi) }

// ─── Uint16 ─────────────────────────────────────────────────────

// Uint16 creates a generator for all uint16 values.
func Uint16() *Generator[uint16] { return rapid.Uint16() }

// Uint16Min creates a generator for uint16 values >= lo.
func Uint16Min(lo uint16) *Generator[uint16] { return rapid.Uint16Min(lo) }

// Uint16Max creates a generator for uint16 values <= hi.
func Uint16Max(hi uint16) *Generator[uint16] { return rapid.Uint16Max(hi) }

// Uint16Range creates a generator for uint16 values in [lo, hi].
func Uint16Range(lo, hi uint16) *Generator[uint16] { return rapid.Uint16Range(lo, hi) }

// ─── Uint32 ─────────────────────────────────────────────────────

// Uint32 creates a generator for all uint32 values.
func Uint32() *Generator[uint32] { return rapid.Uint32() }

// Uint32Min creates a generator for uint32 values >= lo.
func Uint32Min(lo uint32) *Generator[uint32] { return rapid.Uint32Min(lo) }

// Uint32Max creates a generator for uint32 values <= hi.
func Uint32Max(hi uint32) *Generator[uint32] { return rapid.Uint32Max(hi) }

// Uint32Range creates a generator for uint32 values in [lo, hi].
func Uint32Range(lo, hi uint32) *Generator[uint32] { return rapid.Uint32Range(lo, hi) }

// ─── Uint64 ─────────────────────────────────────────────────────

// Uint64 creates a generator for all uint64 values.
func Uint64() *Generator[uint64] { return rapid.Uint64() }

// Uint64Min creates a generator for uint64 values >= lo.
func Uint64Min(lo uint64) *Generator[uint64] { return rapid.Uint64Min(lo) }

// Uint64Max creates a generator for uint64 values <= hi.
func Uint64Max(hi uint64) *Generator[uint64] { return rapid.Uint64Max(hi) }

// Uint64Range creates a generator for uint64 values in [lo, hi].
func Uint64Range(lo, hi uint64) *Generator[uint64] { return rapid.Uint64Range(lo, hi) }

// ─── Uintptr ────────────────────────────────────────────────────

// Uintptr creates a generator for all uintptr values.
func Uintptr() *Generator[uintptr] { return rapid.Uintptr() }

// UintptrMin creates a generator for uintptr values >= lo.
func UintptrMin(lo uintptr) *Generator[uintptr] { return rapid.UintptrMin(lo) }

// UintptrMax creates a generator for uintptr values <= hi.
func UintptrMax(hi uintptr) *Generator[uintptr] { return rapid.UintptrMax(hi) }

// UintptrRange creates a generator for uintptr values in [lo, hi].
func UintptrRange(lo, hi uintptr) *Generator[uintptr] {
	return rapid.UintptrRange(lo, hi)
}

// ─── Float32 ────────────────────────────────────────────────────

// Float32 creates a generator for all float32 values.
func Float32() *Generator[float32] { return rapid.Float32() }

// Float32Min creates a generator for float32 values >= lo.
func Float32Min(lo float32) *Generator[float32] { return rapid.Float32Min(lo) }

// Float32Max creates a generator for float32 values <= hi.
func Float32Max(hi float32) *Generator[float32] { return rapid.Float32Max(hi) }

// Float32Range creates a generator for float32 values in [lo, hi].
func Float32Range(lo, hi float32) *Generator[float32] {
	return rapid.Float32Range(lo, hi)
}

// ─── Float64 ────────────────────────────────────────────────────

// Float64 creates a generator for all float64 values.
func Float64() *Generator[float64] { return rapid.Float64() }

// Float64Min creates a generator for float64 values >= lo.
func Float64Min(lo float64) *Generator[float64] { return rapid.Float64Min(lo) }

// Float64Max creates a generator for float64 values <= hi.
func Float64Max(hi float64) *Generator[float64] { return rapid.Float64Max(hi) }

// Float64Range creates a generator for float64 values in [lo, hi].
func Float64Range(lo, hi float64) *Generator[float64] {
	return rapid.Float64Range(lo, hi)
}

// ─── String ─────────────────────────────────────────────────────

// String creates a generator for arbitrary strings.
func String() *Generator[string] { return rapid.String() }

// StringN creates a generator for strings with length constraints.
func StringN(minRunes, maxRunes, maxLen int) *Generator[string] {
	return rapid.StringN(minRunes, maxRunes, maxLen)
}

// StringOf creates a generator for strings composed of runes from elem.
func StringOf(elem *Generator[rune]) *Generator[string] {
	return rapid.StringOf(elem)
}

// StringOfN creates a generator for strings with rune and length constraints.
func StringOfN(elem *Generator[rune], minRunes, maxRunes, maxLen int) *Generator[string] {
	return rapid.StringOfN(elem, minRunes, maxRunes, maxLen)
}

// StringMatching creates a generator for strings matching a regex.
func StringMatching(expr string) *Generator[string] {
	return rapid.StringMatching(expr)
}

// ─── Slice ──────────────────────────────────────────────────────

// SliceOf creates a generator for slices of elem.
func SliceOf[E any](elem *Generator[E]) *Generator[[]E] {
	return rapid.SliceOf(elem)
}

// SliceOfN creates a generator for slices with length in [lo, hi].
func SliceOfN[E any](elem *Generator[E], lo, hi int) *Generator[[]E] {
	return rapid.SliceOfN(elem, lo, hi)
}

// SliceOfDistinct creates a generator for slices with distinct keys.
func SliceOfDistinct[E any, K comparable](
	elem *Generator[E], keyFn func(E) K,
) *Generator[[]E] {
	return rapid.SliceOfDistinct(elem, keyFn)
}

// SliceOfNDistinct creates a generator for distinct-key slices with
// length in [lo, hi].
func SliceOfNDistinct[E any, K comparable](
	elem *Generator[E], lo, hi int, keyFn func(E) K,
) *Generator[[]E] {
	return rapid.SliceOfNDistinct(elem, lo, hi, keyFn)
}

// SliceOfBytesMatching creates a generator for byte slices matching a regex.
func SliceOfBytesMatching(expr string) *Generator[[]byte] {
	return rapid.SliceOfBytesMatching(expr)
}

// Permutation creates a generator that shuffles the given slice.
func Permutation[S ~[]E, E any](slice S) *Generator[S] {
	return rapid.Permutation(slice)
}

// ─── Map ────────────────────────────────────────────────────────

// MapOf creates a generator for maps with the given key and value generators.
func MapOf[K comparable, V any](
	key *Generator[K], val *Generator[V],
) *Generator[map[K]V] {
	return rapid.MapOf(key, val)
}

// MapOfN creates a generator for maps with length in [lo, hi].
func MapOfN[K comparable, V any](
	key *Generator[K], val *Generator[V], lo, hi int,
) *Generator[map[K]V] {
	return rapid.MapOfN(key, val, lo, hi)
}

// MapOfValues creates a generator for maps keyed by a function of the value.
func MapOfValues[K comparable, V any](
	val *Generator[V], keyFn func(V) K,
) *Generator[map[K]V] {
	return rapid.MapOfValues(val, keyFn)
}

// MapOfNValues creates a generator for keyed-by-value maps with length
// in [lo, hi].
func MapOfNValues[K comparable, V any](
	val *Generator[V], lo, hi int, keyFn func(V) K,
) *Generator[map[K]V] {
	return rapid.MapOfNValues(val, lo, hi, keyFn)
}

// ─── Combinators ────────────────────────────────────────────────

// Custom creates a generator from a user-defined draw function.
func Custom[V any](fn func(*T) V) *Generator[V] {
	return rapid.Custom(fn)
}

// Just creates a generator that always returns val.
func Just[V any](val V) *Generator[V] {
	return rapid.Just(val)
}

// OneOf creates a generator that draws from one of the given generators.
func OneOf[V any](gens ...*Generator[V]) *Generator[V] {
	return rapid.OneOf(gens...)
}

// Deferred creates a generator that lazily evaluates fn. Useful for
// recursive generators.
func Deferred[V any](fn func() *Generator[V]) *Generator[V] {
	return rapid.Deferred(fn)
}

// Ptr creates a generator for pointer values; nil is possible when
// allowNil is true.
func Ptr[E any](elem *Generator[E], allowNil bool) *Generator[*E] {
	return rapid.Ptr(elem, allowNil)
}

// Map transforms generator output through fn.
func Map[U, V any](g *Generator[U], fn func(U) V) *Generator[V] {
	return rapid.Map(g, fn)
}

// SampledFrom creates a generator that draws uniformly from the
// given values.
func SampledFrom[S ~[]E, E any](slice S) *Generator[E] {
	return rapid.SampledFrom(slice)
}

// ─── User-defined types ─────────────────────────────────────────

// Make creates a generator that produces values of type V via
// reflection.
func Make[V any]() *Generator[V] {
	return rapid.Make[V]()
}

// MakeCustom creates a generator with field-level overrides.
func MakeCustom[V any](cfg MakeConfig) *Generator[V] {
	return rapid.MakeCustom[V](cfg)
}
