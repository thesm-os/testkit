// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package nocontexttest_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/nocontext"
	"go.thesmos.sh/testkit/conformance/corpus/iface/lang/nocontext/nocontexttest"
)

// An interface taking no context earns a smoke call per method and nothing
// else: cancellation, deadline and nil-context are claims about a parameter it
// does not have, and emitting them would not compile.
//
// So almost everything worth checking here is the author's, which is what the
// row table is for. The zero divisor comes from the row rather than the
// fixture: a generator derives plausible integers and every plausible divisor
// divides, so the error path is one derivation cannot reach.
func TestCalculatorContract(t *testing.T) {
	t.Parallel()

	nocontexttest.RunCalculator(t,
		nocontexttest.CalculatorHarness[*nocontexttest.InMemory]{Name: "in-memory", New: nocontexttest.NewInMemory},
		nocontexttest.CalculatorChecks{
			{
				Method: "Add",
				Name:   "is-commutative",
				Claim:  "Add is commutative",
				Run: func(tb testing.TB, s nocontext.Calculator, fx nocontexttest.CalculatorFixture) {
					tb.Helper()
					testkit.Equal(tb, s.Add(fx.A(), fx.B()), s.Add(fx.B(), fx.A()),
						"addition does not depend on the order of its operands")
				},
			},
			{
				Method: "Divide",
				Name:   "reports-a-zero-divisor",
				Claim:  "Divide reports a zero divisor",
				Run: func(tb testing.TB, s nocontext.Calculator, fx nocontexttest.CalculatorFixture) {
					tb.Helper()
					_, err := s.Divide(fx.A(), 0)
					testkit.ErrorIs(tb, err, nocontexttest.ErrDivideByZero,
						"a zero divisor is an error rather than a panic")
				},
			},
		},
	)
}

// Suppression, against the same subject: what is under test is the harness
// declining what it was told to, not the implementation.
func TestCalculatorContractSuppression(t *testing.T) {
	t.Parallel()

	nocontexttest.RunCalculator(t,
		nocontexttest.CalculatorHarness[*nocontexttest.InMemory]{Name: "in-memory", New: nocontexttest.NewInMemory},
		nocontexttest.CalculatorSuite.Without(nocontexttest.CalculatorSuite.Checks.Add.Smoke()),
	)
}
