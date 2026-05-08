// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package namedreturns

import (
	"context"
	"time"
)

//go:generate testkit stub -o servicetest/service_stub.gen.go Service
//go:generate testkit suite -o servicetest/service_spec.gen.go Service
//go:generate testkit bench -o servicetest/service_bench.gen.go Service

// Service exercises named return values and multiple non-error returns.
type Service interface {
	// Swap returns old and new values. Multiple non-error named returns.
	Swap(ctx context.Context, key string, value string) (old string, new string, err error)
	// Timestamps returns two time values.
	Timestamps(ctx context.Context) (created time.Time, updated time.Time, err error)
}
