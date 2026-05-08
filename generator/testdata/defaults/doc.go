// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package defaults holds fixtures exercising default-value
// mechanisms. Both mechanisms live here side by side because they
// solve the same problem (seeding a builder with non-zero values)
// via different APIs:
//
//   - convention.go    — Request: paired with sibling
//                        defaultstest.RequestDefaults() function
//                        (the chicken-and-egg pattern: the defaults
//                        function lives in the test package the
//                        generator emits into).
//   - field.go         — Config: per-field //testkit:default
//                        directives.
//
// The defaultstest/ subdirectory holds the sibling test package
// where RequestDefaults lives and the generated builder lands.
package defaults
