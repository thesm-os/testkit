// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package namedreturns is the language-axis fixture for named return values, which the recorded-call field names are derived
// from when they are present.
//
// This axis varies the Go type system rather than the classification, so
// these break generators independently of any directive.
package namedreturns

import (
	"context"
)

// Service names its returns. A generated call type takes its field names from
// them, falling back to positional names only when they are absent — so this
// fixture and the unnamed one below must produce different field names.
type Service interface {
	Named(ctx context.Context, id string) (item string, err error)
	Unnamed(ctx context.Context, id string) (string, error)
	PartiallyNamed(ctx context.Context, id string) (item string, _ error)
}
