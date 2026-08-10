// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package enum generates an enumerated type's textual surface and the checks
// that hold it to what the declaration says.
//
// A Go enum is a convention, not a language feature: a defined type and a
// block of typed constants. Nothing stops a conversion admitting a value
// outside the set, nothing notices when a variant is added without the switch
// arm it needs, and nothing relates the type's textual form to the values it
// was declared with. Each of those is a one-line mistake that compiles.
//
// # What is generated
//
// String, a parse function and its sentinel, MarshalText, UnmarshalText,
// Values and IsValid — every one of them skipped when the type already
// declares it, and all of them suppressed by `methods=off`. An author who
// wrote their own String meant to keep it; a generator that refused to run
// until they deleted it would be demanding they give up the more specific
// statement.
//
// # The textual form
//
// For a numeric enum the identifier is the only textual form the declaration
// carries, so `StatusActive` on `type Status int` renders as `Active` — the
// type name is context wherever the value appears, and repeating it is noise
// in every log line.
//
// For a string enum the value *is* the textual form, and it is already written
// down. `US Region = "us-east"` renders as `us-east` and parses from it.
// Deriving `US` instead would discard the only thing the declaration said and
// break every value arriving from JSON, a database column or a query
// parameter — while still round-tripping against itself, so the failure would
// be invisible to a check that only tested the generated pair.
//
// A `//testkit:value` directive on a variant overrides both, for the case
// where the derived spelling clashes with a protocol's and the derivation
// cannot be taught about it.
//
// # Text rather than JSON
//
// MarshalText over MarshalJSON: encoding/json reaches for TextMarshaler on its
// own and so does YAML, and it is what makes the type legal as a map key,
// which a JSON marshaller alone does not.
//
// # What the checks cannot say about a float enum
//
// Every generated check needs a value outside the declared set to probe with,
// and for a float-valued enum there is none to be had: eidos reads a variant's
// declared value as an integer and knows the bounds of no float type, so the
// set reports no values at all and nothing can name a successor to them. The
// two subtests that need one — an undeclared value is invalid, and it does not
// render as a declared one — are dropped rather than written against a guess
// that the set might turn out to declare. Everything else a float enum earns
// is unaffected: the arity, the distinctness of values and of text, the zero,
// the round trip and the encoding boundary are all derivable without a
// boundary value.
//
// # Where the output lands
//
// Beside its source, and it cannot be routed elsewhere — the API declares
// methods on the enum's type, which Go permits only in that type's own
// package. An `out` directive sending it away produces a file naming an
// undefined type. The checks travel with it and take the external test package
// the `_test.go` ending gives them.
package enum
