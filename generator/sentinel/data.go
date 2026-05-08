// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package sentinel

// Data is the top-level template input for one sentinel generation.
type Data struct {
	PackageName string

	// ImportPath is the source package's import path. Empty when the
	// output file lives inside the source package and no qualifier is
	// needed.
	ImportPath string

	// Qualifier is the dotted prefix used to reference source-package
	// symbols ("basic." or ""). Pre-formatted with the trailing dot so
	// templates can write `{{.Qualifier}}{{.Name}}`.
	Qualifier string

	TestName string // e.g. "BasicSentinelErrors"
	Prefix   string // "<pkg>: " — expected prefix for every Err* message

	Sentinels  []ErrorVar
	ErrorTypes []ErrorType

	// CrossPackages lists peer packages whose sentinels must be
	// non-overlapping with this package's. Populated from
	// //testkit:sentinel-no-overlap-with (G24).
	CrossPackages []CrossPackage

	// Directives are the //testkit: package directives consumed by
	// this generation, pre-rendered as source lines so the header
	// partial can echo them into the output for the reader.
	Directives []string
}

// CrossPackage describes one peer package whose sentinel set must
// not overlap with the local set.
type CrossPackage struct {
	ImportPath string
	Alias      string
	Sentinels  []ErrorVar
}

// HasContent reports whether the package has any sentinels or error
// types worth generating tests for. Cross-package overlap data
// alone does not count: a package that only declares the directive
// but has no local sentinels still produces no useful test file.
func (d *Data) HasContent() bool {
	return len(d.Sentinels) > 0 || len(d.ErrorTypes) > 0
}

// ErrorVar holds one exported Err* variable.
type ErrorVar struct {
	Name string
}

// ErrorType holds one exported type implementing the error interface
// plus the metadata sentinel needs to render type-specific subtests.
type ErrorType struct {
	Name      string
	Qualifier string
	Fields    []FieldData

	HasIs     bool // type has Is(error) bool
	HasUnwrap bool // type has Unwrap() error

	// UnwrapField names the error-typed field whose value Unwrap is
	// expected to return. Empty when HasUnwrap is false or the type
	// has no error-typed field.
	UnwrapField string

	// OtherTypes are the names of every other custom error type in
	// the same package — used for cross-error-type non-overlap
	// subtests that catch degenerate Is impls.
	OtherTypes []string

	// FormatCheckOrder is the ordered list of substring values
	// expected in the rendered Error() output. Built from each
	// Field's FormatCheckValue, skipping empties.
	FormatCheckOrder []string
}

// FieldData holds one exported field of an error type, pre-rendered
// for direct use in templates.
type FieldData struct {
	Name    string
	TypeStr string

	// SampleValue is the literal expression that populates this
	// field in test fixtures. String fields → `"test-<lower>"`;
	// error fields → `errors.New("test-<lower>")`; everything else
	// → [generator.SampleValueOf].
	SampleValue string

	// FormatCheckValue is the unquoted substring expected in the
	// rendered Error() output. Empty for fields whose type doesn't
	// contribute identifiable text (numeric, struct, etc).
	FormatCheckValue string

	// IsError flags error-typed fields so templates branch to
	// ErrorIs instead of Equal for round-trip assertions.
	IsError bool
}
