// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package stub is a placeholder for the stub generator port from the
// legacy gen/stub package. The CLI dispatches to [Generator] so
// tooling stays wired through the [generator.Generator] interface;
// invocations return a port-pending error until the analyze + render
// implementation lands.
package stub

import (
	"errors"

	"go.thesmos.sh/testkit/generator"
)

// Generator is the placeholder implementation of [generator.Generator]
// for the stub subcommand. It carries no state and returns an error
// from Generate explaining that the port is in progress.
type Generator struct{}

// Name returns the subcommand name.
func (*Generator) Name() string { return "stub" }

// Generate is not yet implemented. The legacy implementation lived in
// gen/stub; the rebuild moves it under this package once the
// underlying primitives in [generator] (BuildOutputCtx, ScanVars,
// HasMethod, etc.) are exercised by a second consumer.
func (*Generator) Generate(
	_ *generator.Package, _ []string, _ generator.Config, _ generator.Options,
) (*generator.Result, error) {
	return nil, errors.New("stub: generator port pending")
}
