// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package integrationonly registers the //testkit:integration-only
// marker. Methods carrying this directive are skipped by the stub
// generator entirely — useful for integration-only methods that the
// stub layer can't usefully record (network handshakes, blocking
// long-poll calls, ...).
package integrationonly

import (
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/marker"
)

func init() { marker.Register(directive.IntegrationOnly) }

// Has reports whether the method carries //testkit:integration-only.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.IntegrationOnly) }
