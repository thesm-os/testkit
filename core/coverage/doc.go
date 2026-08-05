// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package coverage aggregates per-component and subsystem-level
// coverage signals from any generator's runtime. Per
// INTEGRATION.md§Coverage aggregation, this is the unified surface
// every layer reports into: model contributes per-interface
// state-space + per-law fire-rate + REQ-to-law mappings; sim
// contributes subsystem-level invariant fire-rate + cross-component
// causality reach; chaos contributes fault-coverage rollups; diff-
// rollout contributes divergence-rate.
//
// The package owns the data shape and the aggregation/reporting
// machinery. The runtime trackers that produce the data
// (state-space hash sets, fire-rate counters, branch-coverage
// hooks) live alongside their producing generator and ship later;
// this package describes the contract they fulfill.
//
// JSON serialization uses stdlib json tags throughout so the
// `<artifactDir>/coverage-<run>.json` artifact format is
// self-describing.
package coverage
