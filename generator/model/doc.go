// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package model generates property-based state-machine tests for an interface
// carrying `//testkit:model`: random sequences of its methods run against the
// subject and a known-good in-memory reference side by side, compared after
// every call.
//
// The plugin reads three things and derives nothing twice. The projection
// [generator/suite] queues carries every generated identifier, the method
// signatures and the derived fixture the pools sample. The annotator's stamps
// carry each method's shape, which selects its action constructor through
// [generator/tiers] — the table the conformance gate holds to both registries.
// And the directive itself carries the one escape hatch: `ref=` names a
// constructor for an interface whose semantics no shipped oracle models.
//
// The output is one file beside the harness, reached through one option:
// `<Iface>Model()` registers the run as a contract extension reporting under
// "model", and `<Iface>Without("model")` declines it. Deleting the directive
// deletes the file — and with it the only import of the `engine` module, which
// is why the tier is directive-gated rather than triggered by classifications
// alone.
package model
