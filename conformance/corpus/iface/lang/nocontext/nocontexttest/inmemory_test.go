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
// extension point is for.
func TestCalculatorContract(t *testing.T) {
	t.Parallel()

	nocontexttest.AssertCalculatorContract(t,
		nocontexttest.CalculatorSubject("in-memory", func() nocontext.Calculator {
			return nocontexttest.NewInMemory()
		}),
		nocontexttest.CalculatorWithFixture(divisibleFixture()),
		nocontexttest.CalculatorOnAdd("is commutative", func(
			tb testing.TB, subject nocontext.Calculator, a, b int,
		) {
			tb.Helper()
			testkit.Equal(tb, subject.Add(a, b), subject.Add(b, a),
				"addition does not depend on the order of its operands")
		}),
		nocontexttest.CalculatorOnDivide("reports a zero divisor", func(
			tb testing.TB, subject nocontext.Calculator, a, b int,
		) {
			tb.Helper()
			_, err := subject.Divide(a, 0)
			testkit.ErrorIs(tb, err, nocontexttest.ErrDivideByZero,
				"a zero divisor is an error rather than a panic")
		}),
	)
}

// Suppression, against the same subject: what is under test is the harness
// declining what it was told to, not the implementation.
func TestCalculatorContractSuppression(t *testing.T) {
	t.Parallel()

	nocontexttest.AssertCalculatorContract(t,
		nocontexttest.CalculatorSubject("in-memory", func() nocontext.Calculator {
			return nocontexttest.NewInMemory()
		}),
		nocontexttest.CalculatorWithFixture(divisibleFixture()),
		nocontexttest.CalculatorWithout("Add/smoke"),
		nocontexttest.CalculatorWithoutDouble(),
	)
}

// divisibleFixture supplies the divisor the derivation could not.
//
// A generator derives plausible integers, and every plausible divisor divides —
// so the zero-value check has no way to reach the error path, and says so by
// name rather than passing. Zero is the one divisor that misses.
func divisibleFixture() nocontexttest.CalculatorFixture {
	f := nocontexttest.DefaultCalculatorFixture()
	f.BOther = 0
	return f
}
