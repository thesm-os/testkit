// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enums

// Region is a multi-file enum — half the constants live here, half
// in multifile_more.go. Used by enum-generator tests to exercise
// ScanConstsOfType's source-position sort across files (filename
// lexical order disambiguates ties).
type Region int

const (
	RegionUS Region = iota // US
	RegionEU               // EU
)

// _internal is unexported and must NOT be returned by
// ScanConstsOfType.
const _internal Region = 99 //nolint:unused // fixture: scanner must skip
