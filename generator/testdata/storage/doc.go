// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package storage is a cross-package fixture used by sentinel tests to
// verify the //testkit:sentinel-no-overlap-with directive (G24).
//
// The package declares its own Err* sentinels with the "storage:"
// prefix. When testdata/basic declares
// //testkit:sentinel-no-overlap-with go.thesmos.sh/testkit/generator/testdata/storage,
// the sentinel generator emits cross-package non-overlap subtests
// asserting that none of basic's sentinels match storage's via
// errors.Is.
package storage
