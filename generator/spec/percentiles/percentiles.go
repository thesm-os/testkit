// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package percentiles registers the //testkit:percentiles consumer.
// The bench generator reads the resolved payload to emit a
// [bench.LatencyPercentilesWithin] gate per method — per-percentile
// latency ceilings (p50/p95/p99/...) that fail the benchmark when
// the measured distribution exceeds any of the consumer-supplied
// budgets. Pairs with [//testkit:latency] (mean-only) for shapes
// where tail latency matters more than the average.
//
// Directive form:
//
//	//testkit:percentiles p99=100us
//	//testkit:percentiles p50=10us p95=50us p99=100us
//
// Each arg is `p<N>=<duration>` where N is an integer percentile
// (1..99) and duration parses via [time.ParseDuration]. The
// generator emits the gate so consumers see p50/p95/p99 reported as
// custom metrics regardless of which percentiles they budget.
package percentiles

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
)

// Budget pairs a percentile (1..99) with a per-call duration ceiling.
type Budget struct {
	// Percentile is the percentile rank as an integer (e.g. 99 for p99).
	Percentile int

	// Max is the inclusive ceiling on per-call latency at this
	// percentile. Always positive — the consumer rejects zero and
	// negative durations.
	Max time.Duration

	// Raw is the original directive arg ("p99=100us"). Templates
	// emit it inline for the generated map literal so durations
	// round-trip stably without locale-sensitive formatting.
	Raw string
}

// Payload carries the resolved per-percentile budgets, ordered by
// ascending percentile so generated output is deterministic across
// regenerations.
type Payload struct {
	Budgets []Budget
}

func init() {
	spec.RegisterConsumer(directive.Percentiles, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:percentiles directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Percentiles)
}

// Has reports whether the method carries //testkit:percentiles.
func Has(m *spec.Method) bool {
	return spec.Has(m.Attachments, directive.Percentiles)
}

func consume(method *spec.Method, dir directive.Directive, _ *spec.Data, _ *generator.Package) error {
	if len(dir.Args) == 0 {
		return errors.New("percentiles: at least one budget required (e.g. p99=100us)")
	}
	seen := map[int]bool{}
	budgets := make([]Budget, 0, len(dir.Args))
	for _, arg := range dir.Args {
		b, err := parseBudget(arg)
		if err != nil {
			return fmt.Errorf("percentiles: %w", err)
		}
		if seen[b.Percentile] {
			return fmt.Errorf("percentiles: percentile p%d declared twice", b.Percentile)
		}
		seen[b.Percentile] = true
		budgets = append(budgets, b)
	}
	// Sort ascending by percentile for deterministic output.
	for i := 1; i < len(budgets); i++ {
		for j := i; j > 0 && budgets[j].Percentile < budgets[j-1].Percentile; j-- {
			budgets[j], budgets[j-1] = budgets[j-1], budgets[j]
		}
	}
	spec.Set(&method.Attachments, directive.Percentiles, Payload{Budgets: budgets})
	return nil
}

func parseBudget(arg string) (Budget, error) {
	key, val, ok := strings.Cut(arg, "=")
	if !ok {
		return Budget{}, fmt.Errorf("%q: expected p<N>=<duration> form", arg)
	}
	if !strings.HasPrefix(key, "p") {
		return Budget{}, fmt.Errorf("%q: percentile key must start with 'p' (e.g. p99)", arg)
	}
	n, err := strconv.Atoi(key[1:])
	if err != nil || n < 1 || n > 99 {
		return Budget{}, fmt.Errorf("%q: percentile must be an integer in [1, 99]", arg)
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return Budget{}, fmt.Errorf("%q: %w", arg, err)
	}
	if d <= 0 {
		return Budget{}, fmt.Errorf("%q: duration must be positive", arg)
	}
	return Budget{Percentile: n, Max: d, Raw: arg}, nil
}
