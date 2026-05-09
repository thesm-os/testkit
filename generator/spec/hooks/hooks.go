// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package hooks registers the //testkit:hooks consumer. The
// directive declares the method fires named callbacks during
// execution; the contract subtest constructs a HookRecorder,
// passes it through context, calls the method, and asserts each
// declared hook fired at least once.
//
// Directive form:
//
//	//testkit:hooks BeforeWrite AfterWrite OnError
//
// One or more hook names. Names are recorded verbatim — the
// contract verifies them against [suite.HookRecorder]'s recorded
// calls at runtime.
package hooks

import (
	"errors"
	"slices"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
)

// Payload carries the declared hook names.
type Payload struct {
	// Names is the verbatim list of hook names — one entry per
	// directive arg, in declaration order.
	Names []string
}

func init() {
	spec.RegisterConsumer(directive.Hooks, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:hooks directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Hooks)
}

// Has reports whether the method carries //testkit:hooks.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Hooks) }

func consume(method *spec.Method, dir directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if len(dir.Args) == 0 {
		return errors.New("hooks: requires at least one hook name")
	}
	names := slices.Clone(dir.Args)
	spec.Set(&method.Attachments, directive.Hooks, Payload{Names: names})
	return nil
}
