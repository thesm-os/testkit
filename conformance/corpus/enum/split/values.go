// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package split

// The declared values, in a file of their own. The directive lives on the type
// in iface.go, so nothing here says "enum" — which is the point: opting in and
// declaring the members are separate statements in separate files.
const (
	Green Signal = iota + 1
	Amber
	Red
)
