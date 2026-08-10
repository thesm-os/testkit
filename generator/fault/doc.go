// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package fault owns testkit's `//testkit:fault` directive: which errors a
// method can be made to fail with, and how. It both stamps the directive and
// renders the surface that configuration implies.
//
// # Why its own plugin rather than part of the stub generator
//
// The configuration is not the stub's alone. The double gains a
// `Fault<Name>()` helper, the suite generator writes "returns ErrNotFound for
// a miss" as a subtest, and the model tier needs the partition field to state
// an isolation law. A directive can only be declared once — eidos's registry
// rejects a duplicate name — so declaring it inside any one generator would
// make the others depend on that generator being registered.
//
// Owning it here also means the directive is parsed once and stamped, rather
// than re-read by each generator. Three copies of "strip the `Err` prefix,
// watch for helper-name collisions" would be three chances to disagree, and
// the disagreement would surface as generated code that does not compile.
//
// # How the rendered surface reaches the double
//
// Not by the stub generator rendering it. This plugin declares the same output
// suffixes the stub generator declares, which is what makes Layout resolve
// both contributions to one file — a contributor's Target comes from its own
// suffix table, so identical suffixes are the whole mechanism. The double
// lands in that file's `top` slot and the fault surface in its `bottom` slot,
// which is what puts the helpers after the types they hang off; see
// [SlotName] for why the slot rather than the plugin order decides that.
//
// The consequence worth knowing: the fault helpers are a block at the end of
// the file rather than lines interleaved into each method's configuration.
// Slot ordering is per-plugin, not per-item, so there is no interleaving to
// be had — and the block is attributed, which the interleaved version was not.
//
// # Relationship to shape mixins
//
// eidos's `//testkit:mixin errors` says a method reports misses through a
// sentinel — a law about what the implementation does, which the suite and
// model tiers assert. This directive says which sentinel a test double should
// offer a one-shot helper for, which is test-double ergonomics. Neither
// implies the other, and the two are read independently.
package fault
