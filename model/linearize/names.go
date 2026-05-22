// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package linearize

// Canonical operation names used by the shipped models. The
// generator emits aliases when SUT method names differ; the linearize
// models always switch on these stable strings.
const (
	OpGet    = "Get"
	OpPut    = "Put"
	OpDelete = "Delete"
	OpRead   = "Read"
	OpInc    = "Inc"
	OpDec    = "Dec"

	OpCAS = "CAS"

	OpAppend = "Append"
	OpAt     = "At"
	OpLen    = "Len"

	OpAdd      = "Add"
	OpRemove   = "Remove"
	OpContains = "Contains"

	OpAcquire = "Acquire"
	OpRelease = "Release"

	OpNext  = "Next"
	OpClose = "Close"
)
