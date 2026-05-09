// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package retrysucceeds registers the //testkit:retry-succeeds-on-attempt
// consumer. The stub uses the resulting payload to fail the first
// N-1 calls and succeed on the N-th — exercising consumers' retry
// logic against the contract suite.
//
// Directive form:
//
//	//testkit:retry-succeeds-on-attempt 3
package retrysucceeds

import (
	"fmt"
	"strconv"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the parsed attempt count.
type Payload struct {
	// N is the attempt on which the stub returns a successful
	// result; calls 1..N-1 inject the configured fault. Always
	// positive — the consumer rejects 0 and negative values.
	N int
}

func init() {
	spec.RegisterConsumer(directive.RetrySucceedsOnAttempt, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:retry-succeeds-on-attempt directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.RetrySucceedsOnAttempt)
}

// Has reports whether the method has a retry-succeeds-on-attempt
// directive.
func Has(m *spec.Method) bool {
	return spec.Has(m.Attachments, directive.RetrySucceedsOnAttempt)
}

func consume(method *spec.Method, dir directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("retry-succeeds-on-attempt: %w", err)
	}
	n, err := strconv.Atoi(dir.Args[0])
	if err != nil || n < 1 {
		return fmt.Errorf("retry-succeeds-on-attempt: %q is not a positive integer", dir.Args[0])
	}
	spec.Set(&method.Attachments, directive.RetrySucceedsOnAttempt, Payload{N: n})
	return nil
}
