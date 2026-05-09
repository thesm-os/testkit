// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package all_test

import (
	"testing"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	_ "go.thesmos.sh/testkit/generator/spec/all" // registers every shipped consumer
)

func TestAllRegistersConsumers(t *testing.T) {
	t.Parallel()

	// The umbrella import must wire every shipped consumer. Adding
	// a new consumer subpackage and forgetting to add it to spec/all
	// would silently leave it un-registered for downstream
	// generators — this test catches that.
	wired := []string{
		directive.Sample,
		directive.Atomic,
		directive.Idempotent,
		directive.IntegrationOnly,
		directive.Deprecated,
		directive.RetrySucceedsOnAttempt,
		directive.OrderAfter,
		directive.Partition,
		directive.Errors,
		directive.WrappedVia,
		directive.Allocs,
		directive.Latency,
	}
	for _, name := range wired {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			testkit.True(t, len(spec.Consumers(name)) > 0,
				"consumer registered for "+name)
		})
	}
}
