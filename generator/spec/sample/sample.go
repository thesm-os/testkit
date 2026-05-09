// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package sample is the consumer for the //testkit:sample directive.
//
// Importing the package (typically as a blank import from a generator
// or from spec/all) registers the consumer with [spec.RegisterConsumer]
// so methods carrying `//testkit:sample <Func>...` get a [Payload]
// attached to [spec.Method.Attachments] under [directive.Sample].
//
// Wire from a generator:
//
//	import _ "go.thesmos.sh/testkit/generator/spec/sample"
//
// Read from generator code:
//
//	if p, ok := spec.Get[sample.Payload](m.Attachments, directive.Sample); ok {
//	    // p.Calls is one rendered call expression per non-ctx param.
//	}
//
// Directive forms (resolution shared with all Pattern C consumers
// via [resolver.Resolve]):
//
//	//testkit:sample LocalFunc                                 // same package
//	//testkit:sample go.thesmos.sh/myproj/fixtures.NewKey      // remote package
//
// The qualified form is detected by a "/" in the LHS of the last
// dot. Remote functions are loaded via [spec.Data.Loader] and the
// resolved import is registered on [spec.Data.Tracker] so the call
// expression uses the tracker's alias for the package.
//
// What sample is for:
//
//   - Smoke tests that need non-zero inputs to avoid panic on
//     zero-value parameters (smoke→fail discipline).
//   - Bench hot-paths that must measure the success path, not the
//     zero-value error path.
package sample

import (
	"fmt"
	"go/types"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved sample-function call expressions for
// one method. One entry per non-ctx parameter, in declaration order.
//
// Each entry is the rendered call prefix (no trailing parens):
// templates wrap as `Calls[i] + "()"` to invoke. For local funcs
// the entry is the bare name ("SampleKey"); for funcs in another
// package the entry is the tracker-aliased reference
// ("fixtures.NewKey").
type Payload struct {
	// Calls lists the rendered call expressions, one per non-ctx
	// parameter, in declaration order.
	Calls []string
}

func init() {
	spec.RegisterConsumer(directive.Sample, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:sample directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Sample)
}

// Has reports whether the method has a sample directive.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Sample) }

// consume validates the directive's args against the method's
// non-ctx params and attaches a [Payload] to the method.
func consume(method *spec.Method, dir directive.Directive, data *spec.Data, pkg *generator.Package) error {
	if err := resolver.RequireArgs(dir, method.NonCtxParamCount()); err != nil {
		return fmt.Errorf("sample: %w", err)
	}
	calls := make([]string, len(dir.Args))
	for i, arg := range dir.Args {
		r, err := resolver.Resolve(arg, data, pkg)
		if err != nil {
			return fmt.Errorf("sample %q: %w", arg, err)
		}
		want := resolver.FuncSig{Results: []types.Type{method.NonCtxParamAt(i)}}
		if err := want.Check(r.Obj); err != nil {
			return fmt.Errorf("sample %q: %w", arg, err)
		}
		calls[i] = r.Render()
	}
	spec.Set(&method.Attachments, directive.Sample, Payload{Calls: calls})
	return nil
}
