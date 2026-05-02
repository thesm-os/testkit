// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package concrete

import "context"

// Service has methods on a concrete type.
type Service struct{}

//testkit:timeout 5s
// Run executes the service.
func (*Service) Run(ctx context.Context) error { return nil }

// Stop halts the service.
func (*Service) Stop() {}
