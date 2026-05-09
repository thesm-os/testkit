// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package idempotent registers the //testkit:idempotent marker.
// Importing the package wires the consumer; templates check
// `spec.Has(method.Attachments, directive.Idempotent)` to emit the
// idempotence assertion (call twice → same outcome).
package idempotent

import (
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec/internal/marker"
)

func init() { marker.Register(directive.Idempotent) }
