// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package deleteremoves registers the //testkit:delete-removes
// consumer. The directive sits on the deleter method and names the
// paired reader; the contract subtest deletes, then reads, and
// asserts the reader returns the not-found sentinel.
//
// Directive form (on the deleter):
//
//	//testkit:delete-removes Reader=Get
//	Delete(ctx context.Context, key string) error
//
// Validation: the named reader must exist on the same interface.
package deleteremoves

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
	spec.RegisterConsumer(directive.DeleteRemoves, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:delete-removes directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.DeleteRemoves)
}

// Has reports whether the method carries //testkit:delete-removes.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.DeleteRemoves) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("delete-removes: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.DeleteRemoves, Payload{Reader: want})
			return nil
		}
	}
	return fmt.Errorf("delete-removes: reader method %q not found on interface %s",
		want, data.Interface.Name)
}
