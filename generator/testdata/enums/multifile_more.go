// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package enums

// Continued constants of type Region, declared in a second file to
// exercise ScanConstsOfType's cross-file sort. Source-position
// ordering must place RegionAP after RegionEU (declaration order
// preserved across files via filename then offset).
const (
	RegionAP Region = 2 // AP
	RegionLA Region = 3 // LA
)
