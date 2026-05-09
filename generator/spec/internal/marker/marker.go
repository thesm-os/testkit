// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package marker provides a one-line consumer registration for
// no-arg directives — every Pattern A directive (per spec/doc.go's
// taxonomy: idempotent, pure, atomic, nilsafe, ctx, retryable,
// fuzz, deleter, mutator, ...).
//
// A marker consumer's only job is to record presence: when the
// directive appears on a method, the consumer attaches an empty
// [Presence] value to [spec.Method.Attachments]. Templates and
// emitters check presence via [spec.Has] (or [spec.Get] when they
// want the typed value).
//
// Usage from a marker package:
//
//	// generator/spec/atomic/atomic.go
//	package atomic
//
//	import (
//	    "go.thesmos.sh/testkit/generator/directive"
//	    "go.thesmos.sh/testkit/generator/spec/internal/marker"
//	)
//
//	func init() { marker.Register(directive.Atomic) }
//
// That's the entire file. The marker helper handles registration,
// dispatch, and attachment.
package marker

import (
	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
)

// Presence is the empty payload type marker consumers attach when
// their directive fires. Read sites that just need to know
// "directive present?" use [spec.Has]; sites that want a typed
// value to hang documentation off use [spec.Get] with this type.
type Presence struct{}

// Register wires a presence-only consumer for directiveName. After
// init the directive becomes a no-op marker: when it appears on a
// method, [spec.Method.Attachments] gains a [Presence] entry under
// directiveName; nothing else fires.
//
// Directive args (if any) are ignored — markers by definition take
// no semantic arguments. If a directive in the future develops
// arg-bearing semantics, replace [Register] with a custom consumer
// in that directive's own package.
func Register(directiveName string) {
	spec.RegisterConsumer(directiveName, attach)
}

// attach is the shared callback every marker dispatches through.
func attach(method *spec.Method, dir directive.Directive, _ *spec.Data, _ *generator.Package) error {
	spec.Set(&method.Attachments, dir.Name, Presence{})
	return nil
}
