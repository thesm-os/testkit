// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enum

// Data is the top-level template input for one enum generation.
type Data struct {
	PackageName string

	// ImportPath is the source package's import path. Empty when the
	// output file lives inside the source package.
	ImportPath string

	// Qualifier is the dotted prefix used to reference source-package
	// symbols ("basic." or ""). Pre-formatted with the trailing dot.
	Qualifier string

	// GoldenFile is the wire-compat JSON filename emitted next to
	// the test file, e.g. "status.gen_wire.json". Holds one entry
	// per analyzed type as `{<TypeName>: {<ConstName>: int}}` —
	// per-type wire-compat subtests assert their own slice via
	// [golden.AssertGoldenJSONField] without the other types'
	// contents in the diff.
	GoldenFile string

	Enums []TypeData

	// Directives are the //testkit: package-level directives consumed
	// by this generation, pre-rendered as source lines so the header
	// partial can echo them into the output. Empty when none apply.
	Directives []string
}

// HasContent reports whether the package has any enums worth
// generating tests for.
func (d *Data) HasContent() bool {
	return len(d.Enums) > 0
}

// HasStringer reports whether any enum has a String() method.
// Drives the conditional `fmt` import in the header partial.
func (d *Data) HasStringer() bool {
	for _, e := range d.Enums {
		if e.HasString {
			return true
		}
	}
	return false
}

// HasText reports whether any enum implements
// encoding.TextMarshaler / encoding.TextUnmarshaler. Used by the
// header partial to gate the "marshal text" doc bullet.
func (d *Data) HasText() bool {
	for _, e := range d.Enums {
		if e.HasMarshalText {
			return true
		}
	}
	return false
}

// HasJSON reports whether any enum implements json.Marshaler /
// json.Unmarshaler. Drives the conditional `encoding/json` import.
func (d *Data) HasJSON() bool {
	for _, e := range d.Enums {
		if e.HasMarshalJSON {
			return true
		}
	}
	return false
}

// HasBinary reports whether any enum implements
// encoding.BinaryMarshaler / encoding.BinaryUnmarshaler. Drives the
// conditional `bytes` import in the header partial (the round-trip
// subtest uses bytes.Equal).
func (d *Data) HasBinary() bool {
	for _, e := range d.Enums {
		if e.HasMarshalBinary {
			return true
		}
	}
	return false
}

// TypeData holds one enum type's full per-type metadata.
type TypeData struct {
	TypeName  string
	Qualifier string // copied from Data.Qualifier so partials don't need parent context
	Values    []Value

	// MaxValue is the highest int value across all constants of this
	// type. Used by the boundary subtest as `MaxValue + 1` to drive
	// the stringer's fallback path.
	MaxValue int64

	HasString        bool   // type defines String() string
	HasParse         bool   // package defines Parse<Type>(string) (<Type>, error)
	ParseFunc        string // "ParseStatus"
	HasMarshalText   bool   // type implements encoding.Text{Marshaler,Unmarshaler}
	HasMarshalJSON   bool   // type implements json.{Marshaler,Unmarshaler}
	HasMarshalBinary bool   // type implements encoding.Binary{Marshaler,Unmarshaler}

	// ZeroValueName is the constant name whose IntValue == 0. Empty
	// when no constant has a zero value (rare for iota enums but
	// possible with explicit values).
	ZeroValueName string
}

// Value holds one constant of an enum type.
type Value struct {
	// Name is the Go constant identifier, e.g. "StatusPending".
	Name string

	// ExpectedStr is the canonical stringer output for this constant
	// — either the inline comment (preferred) or a prefix-stripped
	// fallback derived from Name. Only meaningful when the type has
	// a String() method.
	ExpectedStr string

	// IntValue is the numeric value. Used by the wire-compat golden
	// (mapping ConstName -> IntValue) and by the distinct-values
	// assertion implicitly via Go's type-distinct semantics.
	IntValue int64
}
