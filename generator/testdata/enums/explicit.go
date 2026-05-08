// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enums

// Color uses explicit (non-iota) values, including a negative one
// and a non-contiguous range. Used by enum-generator tests to
// exercise ScanConstsOfType against declared values: MaxValue must
// be computed from those values (not assume iota), and
// ZeroValueName must remain empty when no declared constant equals
// zero.
type Color int

const (
	ColorUnknown   Color = -1
	ColorRed       Color = 10
	ColorBlue      Color = 20
	ColorChartreus Color = 999
)
