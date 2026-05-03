// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

import (
	"errors"
	"fmt"
	"go/types"
	"strconv"
	"strings"

	"go.thesmos.sh/testkit/gen"
)

// enrichers maps directive names to functions that enrich MethodData.
// Only implemented enrichers are listed — no stubs or placeholders.
var enrichers = map[string]func(*MethodData, gen.Directive, *gen.Package) error{
	"errors":                    enrichErrors,
	"integration-only":          enrichIntegrationOnly,
	"deprecated":                enrichDeprecated,
	"retry-succeeds-on-attempt": enrichRetrySucceeds,
	"partition":                 enrichPartition,
	"order-after":               enrichOrderAfter,
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

// enrichErrors validates sentinel names against the source package (or
// imported packages for qualified names like "otherpkg.ErrXxx") and
// populates the Sentinels field with qualified references.
func enrichErrors(m *MethodData, d gen.Directive, pkg *gen.Package) error {
	if len(d.Args) == 0 {
		return errors.New("errors directive requires at least one argument")
	}
	for _, arg := range d.Args {
		v, importPath, err := pkg.ResolveVar(arg)
		if err != nil {
			return fmt.Errorf("sentinel %q not found in package", arg)
		}

		// Derive the short name for the FaultXxx() helper.
		short := strings.TrimPrefix(v.Name, "Err")

		// Detect ShortName collisions.
		for _, existing := range m.Sentinels {
			if existing.ShortName == short {
				return fmt.Errorf(
					"sentinel name collision: %s and %s both produce Fault%s",
					existing.VarName, arg, short,
				)
			}
		}

		// Determine the qualified reference for the generated code.
		var pkgPath string
		if importPath != "" {
			pkgPath = importPath
		} else {
			pkgPath = m.iface.sourcePkgPath
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

// enrichRetrySucceeds parses the attempt count and records it.
func enrichRetrySucceeds(m *MethodData, d gen.Directive, _ *gen.Package) error {
	if len(d.Args) != 1 {
		return errors.New("retry-succeeds-on-attempt directive requires exactly one argument (attempt number)")
	}
	n, err := strconv.Atoi(d.Args[0])
	if err != nil || n < 1 {
		return fmt.Errorf("retry-succeeds-on-attempt: %q is not a valid positive integer", d.Args[0])
	}
	m.RetryN = n
	return nil
}

// EnrichRetrySucceedsForTest exports enrichRetrySucceeds for testing.
func EnrichRetrySucceedsForTest(m *MethodData, d gen.Directive, pkg *gen.Package) error {
	return enrichRetrySucceeds(m, d, pkg)
}

// enrichPartition resolves the partition field path and records it.
// Searches direct params first, then struct-typed param fields.
func enrichPartition(m *MethodData, d gen.Directive, _ *gen.Package) error {
	if len(d.Args) != 1 {
		return errors.New("partition directive requires exactly one argument (field name)")
	}
	fieldName := d.Args[0]

	// Search params for the field. First check direct param names.
	for _, p := range m.Params {
		if p.FieldName == fieldName {
			m.Partition = &PartitionInfo{
				FieldPath: fieldName,
				FieldName: fieldName,
				FieldType: p.TypeStr,
			}
			return nil
		}
	}

	// Search struct-typed params for nested fields.
	params := m.Signature.Params()
	n := params.Len()
	if m.Signature.Variadic() {
		n--
	}
	for i := range n {
		if gen.IsContextType(params.At(i).Type()) {
			continue
		}
		st, ok := params.At(i).Type().Underlying().(*types.Struct)
		if !ok {
			continue
		}
		paramField := m.Params[i]
		for f := range st.Fields() {
			if f.Name() == fieldName && f.Exported() {
				typeStr := types.TypeString(f.Type(), m.tracker.Qualifier())
				m.Partition = &PartitionInfo{
					FieldPath: paramField.FieldName + "." + fieldName,
					FieldName: fieldName,
					FieldType: typeStr,
				}
				return nil
			}
		}
	}

	return fmt.Errorf("partition field %q not found in parameters of %s", fieldName, m.Name)
}

// EnrichPartitionForTest exports enrichPartition for testing.
func EnrichPartitionForTest(m *MethodData, d gen.Directive, pkg *gen.Package) error {
	return enrichPartition(m, d, pkg)
}

// enrichOrderAfter records the prerequisite method name.
func enrichOrderAfter(m *MethodData, d gen.Directive, _ *gen.Package) error {
	if len(d.Args) != 1 {
		return errors.New("order-after directive requires exactly one argument (method name)")
	}
	m.OrderAfter = d.Args[0]
	return nil
}

// EnrichOrderAfterForTest exports enrichOrderAfter for testing.
func EnrichOrderAfterForTest(m *MethodData, d gen.Directive, pkg *gen.Package) error {
	return enrichOrderAfter(m, d, pkg)
}
