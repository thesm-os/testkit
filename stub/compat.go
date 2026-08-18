// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package stub

// CompatV1 is the compatibility witness for the double runtime,
// referenced once per generated stub file:
//
//	var _ = stub.CompatV1
//
// The stub, clock and rand packages version together with the root
// module, so one witness covers the surface a generated double rides —
// the protobuf gencode/runtime lesson. A breaking change to the double
// runtime renames the witness, and every file generated against v1
// stops compiling with the skew named instead of dispatching wrong.
func CompatV1() {}
