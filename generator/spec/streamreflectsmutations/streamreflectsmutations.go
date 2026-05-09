// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package streamreflectsmutations registers the
// //testkit:stream-reflects-mutations consumer. The directive sits
// on the writer method and names a paired stream-reader; the
// contract subtest writes a value, then iterates the stream and
// asserts the value appears.
//
// Directive form (on the writer):
//
//	//testkit:stream-reflects-mutations Stream=Scan
//	Put(ctx context.Context, item Record) error
//
// Validation: the named stream method must exist on the same
// interface.
package streamreflectsmutations

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved stream-method name.
type Payload struct {
	// Stream is the name of the paired stream-reader on the same
	// interface.
	Stream string
}

func init() {
	spec.RegisterConsumer(directive.StreamReflectsMutations, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:stream-reflects-mutations
// directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.StreamReflectsMutations)
}

// Has reports whether the method carries //testkit:stream-reflects-mutations.
func Has(m *spec.Method) bool {
	return spec.Has(m.Attachments, directive.StreamReflectsMutations)
}

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("stream-reflects-mutations: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.StreamReflectsMutations, Payload{Stream: want})
			return nil
		}
	}
	return fmt.Errorf("stream-reflects-mutations: stream method %q not found on interface %s",
		want, data.Interface.Name)
}
