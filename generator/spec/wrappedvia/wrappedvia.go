// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package wrappedvia registers the //testkit:wrapped-via consumer.
// The directive declares the wrapping discipline a method honors —
// every error the method returns wraps (via fmt.Errorf %w / errors.Is
// chain) the named sentinel. The stub renders fault helpers that
// match this contract: FaultErrInner returns "outer: %w" wrapping
// ErrInner, so consumers' errors.Is checks round-trip.
//
// Directive forms:
//
//	//testkit:wrapped-via ErrInternal
//	//testkit:wrapped-via go.thesmos.sh/myproj/errs.ErrInternal
//
// The arg must resolve to a variable assignable to the builtin
// error interface — same validator as //testkit:errors, but always
// a single value (the wrap target is one error).
package wrappedvia

import (
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Payload carries the resolved wrap-target sentinel.
type Payload struct {
	// VarName is the bare variable name as declared.
	VarName string

	// Qualified is the rendered call expression, alias-qualified
	// for cross-package or bare for local.
	Qualified string
}

func init() {
	spec.RegisterConsumer(directive.WrappedVia, consume)
}

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, pkg *generator.Package) error {
	if err := resolver.RequireArgs(dir, 1); err != nil {
		return fmt.Errorf("wrapped-via: %w", err)
	}
	arg := dir.Args[0]
	r, err := resolver.Resolve(arg, data, pkg)
	if err != nil {
		return fmt.Errorf("wrapped-via %q: %w", arg, err)
	}
	if err := resolver.VarOfType(r.Obj, resolver.ErrorType()); err != nil {
		return fmt.Errorf("wrapped-via %q: %w", arg, err)
	}
	spec.Set(&method.Attachments, directive.WrappedVia, Payload{
		VarName:   r.Name,
		Qualified: r.Render(),
	})
	return nil
}
