// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package deprecated registers the //testkit:deprecated consumer.
// The directive takes a single replacement-method string; the
// payload carries it for templates to render a deprecation
// annotation pointing the caller at the replacement.
//
// Directive form:
//
//	//testkit:deprecated NewMethod
package deprecated

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the replacement method name from the directive.
type Payload struct {
	// Replacement is the name the caller should migrate to. The
	// stub renders this in the deprecation comment; it is not
	// validated against the source package — the migration target
	// can live elsewhere.
	Replacement string
}

func init() {
	spec.RegisterConsumer(directive.Deprecated, consume)
}

func consume(method *spec.Method, dir directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("deprecated: %w", err)
	}
	spec.Set(&method.Attachments, directive.Deprecated, Payload{Replacement: dir.Args[0]})
	return nil
}
