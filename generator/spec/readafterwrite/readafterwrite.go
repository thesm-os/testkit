// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package readafterwrite registers the //testkit:read-after-write
// consumer. The directive sits on the writer method and names the
// paired reader; the contract subtest writes, then reads, and
// asserts the read returns the written value.
//
// Directive form (on the writer):
//
//	//testkit:read-after-write Reader=Get
//	Put(ctx context.Context, item Record) error
//
// Validation: the named reader method must exist on the same
// interface.
package readafterwrite

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved reader-method name.
type Payload struct {
	// Reader is the name of the paired reader on the same interface.
	Reader string
}

func init() {
	spec.RegisterConsumer(directive.ReadAfterWrite, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:read-after-write directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.ReadAfterWrite)
}

// Has reports whether the method carries //testkit:read-after-write.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.ReadAfterWrite) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("read-after-write: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.ReadAfterWrite, Payload{Reader: want})
			return nil
		}
	}
	return fmt.Errorf("read-after-write: reader method %q not found on interface %s",
		want, data.Interface.Name)
}
