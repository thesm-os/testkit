// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"errors"
	"fmt"
	"strings"

	"go.thesmos.sh/testkit/gen"
)

// enrichers maps directive names to functions that enrich SpecMethodData.
// Only directives consumed by the spec generator are listed.
var enrichers = map[string]func(*SpecMethodData, gen.Directive, *gen.Package) error{
	"errors":           enrichErrors,
	"nilsafe":          enrichNilSafe,
	"ctx":              enrichCtx,
	"timeout":          enrichTimeout,
	"pure":             enrichPure,
	"validates":        enrichValidates,
	"bounded":          enrichBounded,
	"deprecated":       enrichDeprecated,
	"integration-only": enrichIntegrationOnly,
}

// Enrich runs directive enrichers on all methods in the data model.
// Unknown directives are silently skipped — global validation catches typos.
func Enrich(data *SpecData, pkg *gen.Package) error {
	for _, m := range data.Methods {
		for _, d := range m.Directives {
			fn, ok := enrichers[d.Name]
			if !ok {
				continue
			}
			err := fn(m, d, pkg)
			if err != nil {
				return gen.WrapErr(m.Pos, err, "directive %s on %s.%s",
					d.Name, data.InterfaceName, m.Name)
			}
		}
	}
	return nil
}

// enrichErrors validates sentinel names and populates the Sentinels field.
func enrichErrors(m *SpecMethodData, d gen.Directive, pkg *gen.Package) error {
	if len(d.Args) == 0 {
		return errors.New("errors directive requires at least one argument")
	}
	for _, arg := range d.Args {
		v, importPath, err := pkg.ResolveVar(arg)
		if err != nil {
			return fmt.Errorf("sentinel %q not found in package", arg)
		}
		short := strings.TrimPrefix(v.Name, "Err")
		var pkgPath string
		if importPath != "" {
			pkgPath = importPath
		} else {
			pkgPath = pkg.Pkg.Path()
		}
		qualifier := m.tracker.AddPath(pkgPath)
		m.Sentinels = append(m.Sentinels, SentinelInfo{
			VarName:   v.Name,
			ShortName: short,
			Qualified: qualifiedName(qualifier, v.Name),
		})
	}
	return nil
}

//nolint:unparam // enricher interface requires error return
func enrichNilSafe(m *SpecMethodData, _ gen.Directive, _ *gen.Package) error {
	m.NilSafe = true
	return nil
}

func enrichCtx(m *SpecMethodData, _ gen.Directive, _ *gen.Package) error {
	if !m.ReturnsError() {
		return errors.New("ctx directive requires a method that returns error")
	}
	m.Ctx = true
	return nil
}

func enrichTimeout(m *SpecMethodData, d gen.Directive, _ *gen.Package) error {
	if len(d.Args) != 1 {
		return errors.New("timeout directive requires exactly one argument (duration, e.g. 5s)")
	}
	if !m.ReturnsError() {
		return errors.New("timeout directive requires a method that returns error")
	}
	m.Timeout = d.Args[0]
	return nil
}

//nolint:unparam // enricher interface requires error return
func enrichPure(m *SpecMethodData, _ gen.Directive, _ *gen.Package) error {
	m.Pure = true
	return nil
}

func enrichValidates(m *SpecMethodData, d gen.Directive, _ *gen.Package) error {
	if len(d.Args) == 0 {
		return errors.New("validates directive requires at least one field name")
	}
	m.Validates = append(m.Validates, d.Args...)
	return nil
}

func enrichBounded(m *SpecMethodData, d gen.Directive, _ *gen.Package) error {
	if len(d.Args) != 2 {
		return errors.New("bounded directive requires exactly two arguments (min max)")
	}
	m.BoundedMin = d.Args[0]
	m.BoundedMax = d.Args[1]
	return nil
}

func enrichDeprecated(m *SpecMethodData, d gen.Directive, _ *gen.Package) error {
	if len(d.Args) != 1 {
		return errors.New("deprecated directive requires exactly one argument (replacement method name)")
	}
	m.Deprecated = d.Args[0]
	return nil
}

//nolint:unparam // enricher interface requires error return
func enrichIntegrationOnly(m *SpecMethodData, _ gen.Directive, _ *gen.Package) error {
	m.Skip = true
	return nil
}

func qualifiedName(qualifier, name string) string {
	if qualifier != "" {
		return qualifier + "." + name
	}
	return name
}
