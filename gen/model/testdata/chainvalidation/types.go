// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package chainvalidation contains interfaces that violate chain
// codegen constraints. Used by generator_test.go to exercise
// validateChainShape error paths.
package chainvalidation

import (
	"context"
	"iter"
)

// --- Rule 1: Mutator + appends without poison or verify ---

// MutatorChainNoPoisonEntry is the entry type for MutatorChainNoPoison.
type MutatorChainNoPoisonEntry struct {
	ID   string
	Data string
}

// MutatorChainNoPoison has a Mutator-shaped Append (no return value,
// //testkit:mutator) with //testkit:appends but no //testkit:verifies
// or Err() method.
type MutatorChainNoPoison interface {
	//testkit:mutator
	//testkit:appends
	Append(ctx context.Context, entry MutatorChainNoPoisonEntry)

	//testkit:replays
	Replay(ctx context.Context) iter.Seq2[MutatorChainNoPoisonEntry, error]
}

// --- Rule 2: depends-on without entry-id ---

// DepsNoIDEntry is the entry type for DepsWithoutEntryID.
type DepsNoIDEntry struct {
	ID        string
	DependsOn []string
}

// DepsWithoutEntryID has //testkit:depends-on but no //testkit:entry-id.
type DepsWithoutEntryID interface {
	//testkit:appends
	Append(ctx context.Context, entry DepsNoIDEntry) error

	//testkit:replays
	//testkit:depends-on DependsOn
	Replay(ctx context.Context) iter.Seq2[DepsNoIDEntry, error]
}

// --- Rule 3: partition-by without replays ---

// PartNoReplayEntry is the entry type for PartitionWithoutReplay.
type PartNoReplayEntry struct {
	Tenant string
	Data   string
}

// PartitionWithoutReplay has //testkit:partition-by but no //testkit:replays.
type PartitionWithoutReplay interface {
	//testkit:appends
	//testkit:partition-by Tenant
	Append(ctx context.Context, entry PartNoReplayEntry) error

	//testkit:verifies
	Verify(ctx context.Context) error
}

// --- Rule 4: causal ordering without replays ---

// CausalNoReplayEntry is the entry type for CausalWithoutReplay.
type CausalNoReplayEntry struct {
	ID        string
	DependsOn []string
}

// CausalWithoutReplay has //testkit:entry-id and //testkit:depends-on
// on a non-replay method (should fail: needs //testkit:replays).
type CausalWithoutReplay interface {
	//testkit:appends
	//testkit:entry-id ID
	//testkit:depends-on DependsOn
	Append(ctx context.Context, entry CausalNoReplayEntry) error

	//testkit:verifies
	Verify(ctx context.Context) error
}
