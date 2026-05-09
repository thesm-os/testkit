// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package nilsafe registers the //testkit:nilsafe marker. The contract
// is "calling with zero/nil inputs must not panic" — orthogonal to
// validates (which asserts a specific validation error). Templates
// emit a no-panic subtest calling with [m.ZeroArgs] inside a recover
// block.
package nilsafe

import (
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/marker"
)

func init() { marker.Register(directive.NilSafe) }

// Has reports whether the method carries //testkit:nilsafe.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.NilSafe) }
