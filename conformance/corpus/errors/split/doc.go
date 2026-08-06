// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package split declares its error contract across several files.
//
// The directive is package-scoped, so it lives wherever the package comment
// is — here, a doc.go carrying no declarations at all. A generator that read
// the annotated file for its sentinels would find none and emit nothing, and
// the failure would look exactly like a package nobody had annotated.
//
// The sentinels are then spread over two more files, and a custom error type
// sits in a third. Splitting errors by the operation that raises them is an
// ordinary way to organise a package of any size, so the arrangement has to
// work rather than merely not crash.
//
// The output is named after the file holding the first sentinel in source
// order — read.go, not doc.go. That is worth a fixture of its own: the
// filename is derived from a declaration rather than from the annotated file,
// which is the opposite of what a reader would guess.
//
//testkit:sentinel
package split
