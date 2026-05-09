// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package errors registers the //testkit:errors consumer. The
// directive lists sentinel error variables the method may return;
// the stub uses the resolved payload to emit one FaultErrXxx()
// helper per sentinel.
//
// Directive forms:
//
//	//testkit:errors ErrNotFound ErrConflict
//	//testkit:errors go.thesmos.sh/myproj/errs.ErrTimeout
//
// Each arg is either a local sentinel name or a fully-qualified
// reference resolved via [resolver.Resolve]. The validator
// requires each resolved object to be a variable whose type is
// assignable to the builtin error interface.
//
// ShortName collisions (two sentinels both producing FaultBadRequest)
// surface as a hard error — fault-helper names must be unique per
// method.
package errors

import (
	stderrors "errors"
	"fmt"
	"strings"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/resolver"
)

// Sentinel is one resolved sentinel-error reference.
type Sentinel struct {
	// VarName is the bare variable name as declared ("ErrNotFound").
	VarName string

	// ShortName is the fault-helper suffix derived by stripping the
	// "Err" prefix ("NotFound"). Used to compose FaultNotFound() in
	// the generated stub.
	ShortName string

	// Qualified is the rendered call expression for the var,
	// alias-qualified when cross-package ("errs.ErrTimeout") or bare
	// when local ("ErrNotFound").
	Qualified string
}

// Payload carries the resolved sentinels in directive order.
type Payload struct {
	// Sentinels lists the resolved references. Order matches the
	// directive's argument order so generators emit FaultXxx
	// helpers in the order the user declared them.
	Sentinels []Sentinel
}

func init() {
	spec.RegisterConsumer(directive.Errors, consume)
}

// Get retrieves the resolved [Payload] for a method. Returns
// (zero, false) when the method carries no //testkit:errors
// directive. Wraps [spec.Get] so consumers don't repeat the
// directive name.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Errors)
}

// Has reports whether the method carries //testkit:errors.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Errors) }

// FaultReturn renders the trailing-error swap return list a
// fault-helper emits when invoked: non-error results stay zero,
// the error slot carries the sentinel's qualified expression.
//
// Wraps [spec.Method.FaultReturn] — kept here so consumers'
// templates pull this from the errors package alongside the rest
// of the directive's render surface.
func FaultReturn(m *spec.Method, t *generator.ImportTracker, sentinel Sentinel) string {
	return m.FaultReturn(t, sentinel.Qualified)
}

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, pkg *generator.Package) error {
	if len(dir.Args) == 0 {
		return stderrors.New("errors: requires at least one sentinel name")
	}
	errType := resolver.ErrorType()
	out := make([]Sentinel, 0, len(dir.Args))
	for _, arg := range dir.Args {
		r, err := resolver.Resolve(arg, data, pkg)
		if err != nil {
			return fmt.Errorf("errors %q: %w", arg, err)
		}
		if err := resolver.VarOfType(r.Obj, errType); err != nil {
			return fmt.Errorf("errors %q: %w", arg, err)
		}
		short := strings.TrimPrefix(r.Name, "Err")
		// Reject same-method ShortName collisions (two distinct
		// sentinels both producing FaultBadRequest, ...).
		for _, existing := range out {
			if existing.ShortName == short {
				return fmt.Errorf(
					"errors: %s and %s both produce Fault%s",
					existing.VarName, r.Name, short)
			}
		}
		out = append(out, Sentinel{
			VarName:   r.Name,
			ShortName: short,
			Qualified: r.Render(),
		})
	}
	spec.Set(&method.Attachments, directive.Errors, Payload{Sentinels: out})
	return nil
}
