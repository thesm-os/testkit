// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package atomic registers the //testkit:atomic marker. Importing
// the package (typically via spec/all) wires the consumer; templates
// then check `spec.Has(method.Attachments, directive.Atomic)` to
// emit the atomicity assertion suite.
package atomic

import (
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/marker"
)

func init() { marker.Register(directive.Atomic) }

// Has reports whether the method carries //testkit:atomic.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Atomic) }
