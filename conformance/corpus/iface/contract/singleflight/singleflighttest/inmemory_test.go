// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package singleflighttest_test

import (
	"runtime"
	"testing"
	"time"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/clock"
	"go.thesmos.sh/testkit/concurrency"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/singleflight"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/singleflight/singleflighttest"
)

// singleflight is the model tier's under ADR-0018:
// `AUTO-SINGLEFLIGHT-COALESCES` states it, and stating it needs concurrent
// callers — which the harness, running a fixed sequence against one subject,
// has no way to produce.
//
// A generated check would call Run once, coalesce nothing, and report success
// against an implementation that never looks at what is in flight.
func TestContractContract(t *testing.T) {
	t.Parallel()

	// One clock for the run and for every subject it builds, which is what
	// WithClock exists to arrange: a window measured on one clock and advanced
	// on another is not a window at all.
	clk := clock.NewTestClock(origin)

	singleflighttest.AssertContractContract(t,
		singleflighttest.ContractSubject("in-memory", func() singleflight.Contract {
			return singleflighttest.NewInMemory(clk)
		}),
		singleflighttest.ContractWithClock(clk),
		singleflighttest.ContractOnRun("coalesces callers that find one in flight", func(
			tb testing.TB, subject singleflight.Contract, _ string,
		) {
			tb.Helper()
			// Concurrency is what a *generated* check cannot arrange — the
			// harness drives a fixed sequence — and what a check written here
			// can. Determinism is what a hand-rolled version gets wrong: the
			// work has to still be running when the second caller arrives, and
			// a subject whose work completes instantly coalesces only when the
			// scheduler happens to oblige.
			//
			// So the leader parks on the clock, and this pumps it until every
			// caller has been answered. A caller arriving after a release
			// becomes the next leader and is released on the next turn.
			defer concurrency.GoroutineLeak(tb)()

			const callers = 4

			ctx, done := tb.Context(), make(chan struct{}, callers)
			for range callers {
				go func() {
					testkit.NoError(tb, subject.Run(ctx, singleflighttest.SlowKey),
						"a concurrent caller is answered")
					done <- struct{}{}
				}()
			}

			// The clock is pumped rather than advanced a fixed number of
			// times. A caller that arrives after a release becomes the next
			// leader and parks, and one that arrives before it coalesces and
			// never parks at all — so how many advances are needed is a fact
			// about the interleaving rather than about the callers.
			//
			// Advancing an idle clock is a no-op, which is what makes the pump
			// safe to run until every caller has been answered.
			stop := make(chan struct{})
			pumped := make(chan struct{})
			go func() {
				defer close(pumped)
				for {
					select {
					case <-stop:
						return
					default:
						clk.Advance(singleflighttest.WorkDuration)
						runtime.Gosched()
					}
				}
			}()

			for range callers {
				<-done
			}
			close(stop)
			<-pumped
		}),
	)
}

// origin is where the run's clock starts.
//
// A fixed instant rather than time.Now, so a failure reproduces exactly.
var origin = time.Date(2026, time.August, 11, 0, 0, 0, 0, time.UTC)

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	singleflighttest.AssertContractContract(t,
		singleflighttest.ContractSubject("in-memory", func() singleflight.Contract {
			return singleflighttest.NewInMemory(clock.NewTestClock(origin))
		}),
		singleflighttest.ContractWithout("Run/smoke"),
		singleflighttest.ContractWithoutDouble(),
	)
}
