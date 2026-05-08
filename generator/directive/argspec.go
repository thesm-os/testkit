// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package directive

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// ArgKind classifies a directive argument's type. The parser reports
// kind-mismatch errors (e.g. //testkit:bounded foo when min..max is
// expected) at directive-validation time.
type ArgKind int

// Argument kinds.
const (
	// ArgString accepts any non-empty token.
	ArgString ArgKind = iota

	// ArgIdent accepts a Go identifier — letters, digits, underscore,
	// must not start with a digit. Used for names of methods, fields,
	// errors, and other Go-symbol references.
	ArgIdent

	// ArgInt accepts a base-10 signed integer.
	ArgInt

	// ArgDuration accepts a Go duration string ("1s", "100ms", "5m").
	ArgDuration

	// ArgRange accepts "min..max" where min and max parse as numbers
	// (signed floats — parsed via strconv.ParseFloat).
	ArgRange

	// ArgEnum accepts only values listed in [ArgSpec.Enum].
	ArgEnum

	// ArgKey accepts an identifier whose existence is verified later
	// against the package being analyzed (method name, field name,
	// error-var name). Parsing-time check: must be a valid Go ident.
	// Cross-package validation happens at the consumer/emitter level.
	ArgKey
)

// String returns the human-readable kind name for diagnostics.
func (k ArgKind) String() string {
	switch k {
	case ArgString:
		return "string"
	case ArgIdent:
		return "ident"
	case ArgInt:
		return "int"
	case ArgDuration:
		return "duration"
	case ArgRange:
		return "range"
	case ArgEnum:
		return "enum"
	case ArgKey:
		return "key"
	default:
		return "unknown"
	}
}

// ArgSpec declares one argument slot in a directive's signature.
// Descriptors carry a slice of ArgSpec; the validator walks them in
// order against the actual args present on a directive occurrence.
type ArgSpec struct {
	// Name is the parameter name for diagnostics ("ErrName",
	// "min..max", "Field"). Not user-facing in the directive line —
	// purely for help text and error messages.
	Name string

	// Kind classifies the accepted value. See [ArgKind].
	Kind ArgKind

	// Required reports whether the argument must be present. False
	// allows omission; the consumer/emitter sees an absent arg as
	// the zero value.
	Required bool

	// Multi reports whether the slot accepts multiple values. When
	// true, all remaining args at this position are consumed
	// (variadic-style). Only the last [ArgSpec] in a slice may have
	// Multi=true.
	Multi bool

	// Enum lists accepted values when Kind=ArgEnum. Ignored otherwise.
	Enum []string
}

// ArgOption mutates an [ArgSpec] during construction. Used as the
// trailing varargs to [Arg].
type ArgOption func(*ArgSpec)

// Required marks the argument as mandatory.
func Required(s *ArgSpec) { s.Required = true }

// Multi marks the argument slot as variadic (consumes all remaining
// positional args). Only valid on the last slot of a descriptor.
func Multi(s *ArgSpec) { s.Multi = true }

// OneOf restricts the argument to a fixed set of values, switching
// the kind to [ArgEnum]. Replaces any previous Kind on the spec.
func OneOf(values ...string) ArgOption {
	return func(s *ArgSpec) {
		s.Kind = ArgEnum
		s.Enum = append([]string(nil), values...)
	}
}

// validateArg checks one argument string against one [ArgSpec].
// Returns a descriptive error or nil. Pure — no allocation beyond
// error messages.
func validateArg(spec ArgSpec, value string) error {
	switch spec.Kind {
	case ArgString:
		if value == "" {
			return fmt.Errorf("%s: empty string", spec.Name)
		}
	case ArgIdent, ArgKey:
		if !isGoIdent(value) {
			return fmt.Errorf("%s: %q is not a valid Go identifier", spec.Name, value)
		}
	case ArgInt:
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return fmt.Errorf("%s: %q is not an integer", spec.Name, value)
		}
	case ArgDuration:
		if _, err := time.ParseDuration(value); err != nil {
			return fmt.Errorf("%s: %q is not a duration", spec.Name, value)
		}
	case ArgRange:
		lo, hi, ok := strings.Cut(value, "..")
		if !ok {
			return fmt.Errorf("%s: %q is not a min..max range", spec.Name, value)
		}
		if _, err := strconv.ParseFloat(lo, 64); err != nil {
			return fmt.Errorf("%s: range lower bound %q is not numeric", spec.Name, lo)
		}
		if _, err := strconv.ParseFloat(hi, 64); err != nil {
			return fmt.Errorf("%s: range upper bound %q is not numeric", spec.Name, hi)
		}
	case ArgEnum:
		if slices.Contains(spec.Enum, value) {
			return nil
		}
		return fmt.Errorf("%s: %q is not one of %v", spec.Name, value, spec.Enum)
	}
	return nil
}

// isGoIdent reports whether s is a valid Go identifier (letters,
// digits, underscore; must not start with a digit; non-empty).
// Stripped-down version of go/token.IsIdentifier to avoid the import.
func isGoIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_':
			// allowed in any position
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			// allowed in any position
		case r >= '0' && r <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}
