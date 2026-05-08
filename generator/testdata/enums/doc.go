// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package enums holds enum-shape variations beyond the canonical
// iota cases in basic. Each file demonstrates one departure from
// the canonical shape:
//
//   - explicit.go     — explicit non-iota values (negative,
//                       non-contiguous)
//   - multifile.go +  — declarations spread across two files,
//     multifile_more.go exercising cross-file source-position sort
//
// Tests address types by name (`enum Color`, `enum Region`) so the
// fixtures can co-exist in one package.
package enums

//go:generate testkit enum -o explicit.gen_test.go Color
//go:generate testkit enum -o multifile.gen_test.go Region
