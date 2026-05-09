// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package lifecycleafterclose registers the
// //testkit:lifecycle-after-close consumer. The directive sits on
// the close method and names a paired reader; the contract subtest
// closes, then reads, and asserts the reader returns the closed
// sentinel (or [context.Canceled], depending on the impl's
// contract).
//
// Directive form (on the close method):
//
//	//testkit:lifecycle-after-close Reader=Get
//	Close(ctx context.Context) error
//
// Validation: the named reader must exist on the same interface.
package lifecycleafterclose

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
	spec.RegisterConsumer(directive.LifecycleAfterClose, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:lifecycle-after-close directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.LifecycleAfterClose)
}

// Has reports whether the method carries //testkit:lifecycle-after-close.
func Has(m *spec.Method) bool {
	return spec.Has(m.Attachments, directive.LifecycleAfterClose)
}

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("lifecycle-after-close: %w", err)
	}
	want := dir.Args[0]
	for _, m := range data.Interface.Methods {
		if m.Name == want {
			spec.Set(&method.Attachments, directive.LifecycleAfterClose, Payload{Reader: want})
			return nil
		}
	}
	return fmt.Errorf("lifecycle-after-close: reader method %q not found on interface %s",
		want, data.Interface.Name)
}
