// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package mixed exercises the spec generator with a mix of skipped methods,
// methods without directives, and active methods. Verifies that integration-only
// methods produce no subtests and that undirected methods are silent.
package mixed

import (
	"context"
	"errors"
)

//go:generate testkit suite -o processortest/processor_spec.gen.go Processor

// ErrInvalidInput is returned for bad input.
var ErrInvalidInput = errors.New("invalid input")

// Processor has a mix of directive and non-directive methods.
type Processor interface {
	//testkit:integration-only
	// Run is a long-running method that can't be spec-tested.
	Run(ctx context.Context) error

	//testkit:errors ErrInvalidInput
	//testkit:nilsafe
	// Process handles a single item.
	Process(ctx context.Context, data []byte) error

	// Describe has no directives — should produce no subtests.
	Describe() string

	//testkit:deprecated ProcessV2
	// LegacyProcess is deprecated.
	LegacyProcess(ctx context.Context, data []byte) error
}
