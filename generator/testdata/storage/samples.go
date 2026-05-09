// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package storage

// SampleKey is a `func() string` referenced by cross-package sample
// directive tests in generator/spec/sample and the resolver. Living
// here (rather than in basic) lets those tests verify that a
// qualified directive arg resolves through a sibling package via
// the loader + tracker.
func SampleKey() string { return "storage-key" }
