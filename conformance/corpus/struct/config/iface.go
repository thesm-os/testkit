// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package config is the field-directive half of the builder's defaults
// handling.
//
// The generator seeds New<Type>() from one of three sources: a Defaults()
// companion in the test package, per-field //testkit:default directives, or
// the zero value. Each produces a different constructor, so they are three
// fixtures rather than one — [go.thesmos.sh/testkit/conformance/corpus/struct/domain]
// carries the companion, this carries the directives, and
// [go.thesmos.sh/testkit/conformance/corpus/struct/plain] carries neither.
//
// There is deliberately no Defaults function here. Adding one would make the
// fixture test precedence between the two sources rather than the directive
// path itself, and precedence is a question for a composite.
package config

// Config carries a default on every literal kind the directive accepts, since
// each is parsed differently: quoting for strings, base detection for ints,
// keyword for bools, and the untyped nil that has no literal form of its own.
//
//testkit:out configtest/ pkg=configtest
//testkit:builder
type Config struct {
	Host string //testkit:default "localhost"

	Port int //testkit:default 8080

	Verbose bool //testkit:default true

	Ratio float64 //testkit:default 0.75

	// Retries has a zero default stated explicitly. A generator that treats
	// "zero" as "no directive" emits the same constructor either way, and this
	// is the field that catches it.
	Retries int //testkit:default 0

	// Fallback is a pointer defaulted to nil. Its literal is the same as the
	// zero value, so it tests that the directive is read rather than inferred.
	Fallback *string //testkit:default nil

	// Undirected carries no directive and must stay at its zero value even
	// though its neighbours do not.
	Undirected string
}
