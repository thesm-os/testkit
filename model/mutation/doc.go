// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package mutation provides runtime mutation operators that inject
// synthetic bugs into a SUT, used by the model generator's mutation
// harness to verify the auto-derived law suite catches each
// distortion.
//
// Each operator is a small typed struct with a stable [Operator.Name]
// and a per-shape helper method (ShouldDrop, ShouldDup, Retarget,
// etc.). The generator emits one wrapped-call site per (operator,
// method) pair that the operator is compatible with; the wrapper
// consults the operator's helper before delegating to the underlying
// impl. An operator that the test suite fails to kill is reported
// as "Unkilled" — a hole in the law catalog.
//
// Operators are not directly composable with one another at this
// layer; the generator's harness applies one operator at a time per
// run.
package mutation
