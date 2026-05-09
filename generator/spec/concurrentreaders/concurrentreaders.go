// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package concurrentreaders registers the //testkit:concurrent-readers
// marker. The contract is RWMutex semantics: parallel reads tolerated,
// writes serialized. Templates emit a parallel-readers subtest forking
// many goroutines that all read concurrently, asserting they don't
// deadlock and the parallelism is real (not just safe).
package concurrentreaders

import (
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/marker"
)

func init() { marker.Register(directive.ConcurrentReaders) }

// Has reports whether the method carries //testkit:concurrent-readers.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.ConcurrentReaders) }
