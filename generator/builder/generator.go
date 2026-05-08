// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package builder is a placeholder for the builder generator port
// from the legacy gen/builder package. The CLI dispatches to
// [Generator] so tooling stays wired through the
// [generator.Generator] interface; invocations return a port-pending
// error until the analyze + render implementation lands.
package builder

import (
	"errors"

	"go.thesmos.sh/testkit/generator"
)

// Generator is the placeholder implementation of [generator.Generator]
// for the builder subcommand.
type Generator struct{}

// Name returns the subcommand name.
func (*Generator) Name() string { return "builder" }

// Generate is not yet implemented. The legacy implementation lived
// in gen/builder; the rebuild moves it under this package.
func (*Generator) Generate(
	_ *generator.Package, _ []string, _ generator.Config, _ generator.Options,
) (*generator.Result, error) {
	return nil, errors.New("builder: generator port pending")
}
