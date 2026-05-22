// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package mutation

import (
	"sort"
	"strings"
	"testing"

	"go.thesmos.sh/testkit"
)

// Result reports the outcome of one (operator, property-suite) run.
type Result struct {
	// Operator is the operator's Name(). One Result per operator
	// per Run invocation.
	Operator string

	// Killed reports whether the property suite caught the
	// operator's induced bug. true → at least one law failed when
	// the suite ran against the wrapped SUT.
	Killed bool

	// FailureMsg is the first failure message captured by the
	// suite, populated when Killed is true. Empty otherwise.
	FailureMsg string
}

// Report aggregates [Result] entries across a Run.
type Report struct {
	// Results lists one entry per operator in the order Run
	// processed them (which matches the operators slice).
	Results []Result
}

// KillRate returns the fraction of operators the suite caught,
// rounded as a float in [0.0, 1.0]. An empty Report returns 0.
func (r Report) KillRate() float64 {
	if len(r.Results) == 0 {
		return 0
	}
	killed := 0
	for _, res := range r.Results {
		if res.Killed {
			killed++
		}
	}
	return float64(killed) / float64(len(r.Results))
}

// Unkilled returns the names of operators the suite failed to
// catch, sorted alphabetically.
func (r Report) Unkilled() []string {
	var out []string
	for _, res := range r.Results {
		if !res.Killed {
			out = append(out, res.Operator)
		}
	}
	sort.Strings(out)
	return out
}

// Killed returns the names of operators the suite caught, sorted
// alphabetically.
func (r Report) Killed() []string {
	var out []string
	for _, res := range r.Results {
		if res.Killed {
			out = append(out, res.Operator)
		}
	}
	sort.Strings(out)
	return out
}

// Run executes runWith once per operator, recording whether the
// property suite catches the operator's induced bug. The runWith
// closure receives a [testing.TB] surrogate plus the active
// operator; it should construct the SUT, wrap each mutated method
// call using the operator's decision helpers, and invoke the
// auto-derived property suite (e.g., AssertStoreModel) against the
// wrapped SUT.
//
// The TB surrogate is a [testkit.FailableTB]: when the suite
// records a failure, the operator is considered killed. The
// surrogate's failure machinery does not propagate to the outer
// testing.T — Run handles all reporting.
//
// Run is sequential: operators are not composed across iterations.
// Equivalence-class analysis runs post-hoc via [EquivalenceClasses].
func Run(_ testing.TB, operators []Operator, runWith func(tb testing.TB, op Operator)) Report {
	report := Report{Results: make([]Result, 0, len(operators))}
	for _, op := range operators {
		surrogate := testkit.NewFailableTB().WithName("mutation:" + op.Name())
		runWith(surrogate, op)
		res := Result{
			Operator: op.Name(),
			Killed:   surrogate.Failed(),
		}
		if res.Killed {
			res.FailureMsg = strings.TrimSpace(surrogate.Msg())
		}
		report.Results = append(report.Results, res)
	}
	return report
}
