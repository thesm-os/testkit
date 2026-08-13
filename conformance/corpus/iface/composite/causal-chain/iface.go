// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package causalchain is the composite fixture pairing the chain contract
// with the causal mixin: an append-only log whose admission policy is
// causality — an entry lands only after everything it depends on.
//
// The pairing exists because the replay-causality rule needs both stamps on
// one method: the chain says the log replays, the causal claim says the
// replay respects the dependency order, and neither alone states the rule.
package causalchain

import (
	"context"
)

// Entry is the log's element: an identifier, the entries it depends on, and
// the payload they order.
type Entry struct {
	ID        string
	DependsOn []string
	Body      string
}

// Log is the fixture interface.
//
//testkit:out causalchaintest/ pkg=causalchaintest
//testkit:stub
//testkit:suite
//testkit:model
type Log interface {
	// Append admits an entry only after everything it depends on — the
	// admission policy the causal claim states over the chain's log.
	//testkit:contract chain role=append replay=Replay
	//testkit:mixin causal
	Append(ctx context.Context, e Entry) error

	// Replay is the chain contract's replay role.
	Replay(ctx context.Context) ([]Entry, error)
}
