// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"errors"
	"fmt"
	"strings"

	"go.thesmos.sh/testkit/gen"
)

// enrichers maps directive names to functions that enrich MethodData.
// Only implemented enrichers are listed — no stubs or placeholders.
var enrichers = map[string]func(*MethodData, gen.Directive, *gen.Package) error{
	"errors":           enrichErrors,
	"integration-only": enrichIntegrationOnly,
	"deprecated":       enrichDeprecated,
}

// Enrich runs directive enrichers on all methods in the data model.
// Unknown directives are silently skipped — global validation catches typos.
func Enrich(data *Data, pkg *gen.Package) error {
	for i := range data.Interfaces {
		iface := &data.Interfaces[i]
		for _, m := range iface.Methods {
			for _, d := range m.Directives {
				fn, ok := enrichers[d.Name]
				if !ok {
					continue
				}
				err := fn(m, d, pkg)
				if err != nil {
					return gen.WrapErr(m.Pos, err, "directive %s on %s.%s", d.Name, iface.Name, m.Name)
				}
			}
		}
	}
	return nil
}

// EnrichIntegrationOnlyForTest exports enrichIntegrationOnly for testing.
func EnrichIntegrationOnlyForTest(m *MethodData, d gen.Directive, pkg *gen.Package) error {
	return enrichIntegrationOnly(m, d, pkg)
}

// EnrichDeprecatedForTest exports enrichDeprecated for testing.
func EnrichDeprecatedForTest(m *MethodData, d gen.Directive, pkg *gen.Package) error {
	return enrichDeprecated(m, d, pkg)
}

// enrichErrors validates sentinel names against the source package
// and populates the Sentinels field with qualified references.
func enrichErrors(m *MethodData, d gen.Directive, pkg *gen.Package) error {
	if len(d.Args) == 0 {
		return errors.New("errors directive requires at least one argument")
	}
	for _, arg := range d.Args {
		_, err := pkg.Var(arg)
		if err != nil {
			return fmt.Errorf("sentinel %q not found in package", arg)
		}
		qualifier := m.tracker.AddPath(m.iface.sourcePkgPath)
		m.Sentinels = append(m.Sentinels, SentinelInfo{
			VarName:   arg,
			ShortName: strings.TrimPrefix(arg, "Err"),
			Qualified: qualifiedName(qualifier, arg),
		})
	}
	return nil
}

// enrichIntegrationOnly marks the method to be skipped in stub emission.
func enrichIntegrationOnly(m *MethodData, _ gen.Directive, _ *gen.Package) error {
	m.Skip = true
	return nil
}

func qualifiedName(qualifier, name string) string {
	if qualifier != "" {
		return qualifier + "." + name
	}
	return name
}

// enrichDeprecated records the replacement method name.
func enrichDeprecated(m *MethodData, d gen.Directive, _ *gen.Package) error {
	if len(d.Args) != 1 {
		return errors.New("deprecated directive requires exactly one argument (replacement method name)")
	}
	m.Deprecated = d.Args[0]
	return nil
}
