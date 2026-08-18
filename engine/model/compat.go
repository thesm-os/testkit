// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package model

// CompatV1 is the compatibility witness for the engine module's
// assertion surface, referenced once per generated model file:
//
//	var _ = model.CompatV1
//
// The law, action, linearize, ref and timeaware packages version
// together with this module, so one witness covers the set: generated
// files that bind assertion bodies here can skew against the module a
// build resolves — the protobuf gencode/runtime lesson. A breaking
// change to the assertion surface renames the witness, and every file
// generated against v1 stops compiling with the skew named instead of
// silently asserting something else.
func CompatV1() {}
