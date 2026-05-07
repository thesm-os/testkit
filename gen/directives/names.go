// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package directives defines the canonical directive name constants
// shared across the gen package and its sub-packages. It is a leaf
// package with no dependencies on gen or gen/directive, which breaks
// the import cycle that would otherwise arise from gen needing
// directive names and gen/directive needing gen types.
package directives

// Directive name constants. These match the //testkit:<name> syntax
// parsed from Go source comments.
const (
	Pure              = "pure"
	Cacheable         = "cacheable"
	Idempotent        = "idempotent"
	Monotonic         = "monotonic"
	SideEffect        = "sideeffect"
	Concurrent        = "concurrent"
	ConcurrentReaders = "concurrent-readers"
	Retryable         = "retryable"
	RetrySucceedsOn   = "retry-succeeds-on-attempt"
	Deleter           = "deleter"
	Mutator           = "mutator"
	KeyField          = "keyfield"
	Appends           = "appends"
	Verifies          = "verifies"
	Replays           = "replays"
	PartitionBy       = "partition-by"
	EntryIDField      = "entry-id"
	DependsOnField    = "depends-on"
	HashFunc          = "hash"
	TimeAware         = "time-aware"
	Nondeterministic  = "nondeterministic"
)
