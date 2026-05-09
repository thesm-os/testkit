// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package concurrent registers the //testkit:concurrent marker.
// Elevates the shape's existing ConcurrentSafe baseline to a strict
// requirement with higher load — the directive presence implies the
// impl claims safety under heavy parallelism, and the contract
// verifies with more pressure than the baseline.
package concurrent

import (
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	"go.thesmos.sh/testkit/generator/spec/internal/marker"
)

func init() { marker.Register(directive.Concurrent) }

// Has reports whether the method carries //testkit:concurrent.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Concurrent) }
