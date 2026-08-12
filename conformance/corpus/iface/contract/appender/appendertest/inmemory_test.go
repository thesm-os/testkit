// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package appendertest_test

import (
	"testing"

	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/appender"
	"go.thesmos.sh/testkit/conformance/corpus/iface/contract/appender/appendertest"
)

// appender is the model tier's under ADR-0018: `AUTO-APPEND-ONLY-GROWS` states
// it, and the suite tier implements no property a law already carries.
//
// So what the harness generates here is the signature-derived family, and it is
// not nothing — a log that panicked on a derived key, or applied a write for a
// cancelled caller, fails before any law runs.
func TestContractContract(t *testing.T) {
	t.Parallel()

	appendertest.AssertContractContract(t,
		appendertest.ContractModel(),
		appendertest.ContractSubject("in-memory", func() appender.Contract {
			return appendertest.NewInMemory()
		}),
		// Dropped rather than satisfied: the log appends every value, so
		// there is no input Run refuses — the zero-on-error check has no
		// miss to find.
		appendertest.ContractWithout("Run/an error carries the zero value"),
	)
}

// Declining the double is separate from dropping a check.
func TestContractContractWithoutTheDouble(t *testing.T) {
	t.Parallel()

	appendertest.AssertContractContract(t,
		appendertest.ContractSubject("in-memory", func() appender.Contract {
			return appendertest.NewInMemory()
		}),
		// Dropped rather than satisfied: the log appends every value, so
		// there is no input Run refuses — the zero-on-error check has no
		// miss to find.
		appendertest.ContractWithout("Run/an error carries the zero value"),
		appendertest.ContractWithout("Run/smoke"),
		appendertest.ContractWithoutDouble(),
	)
}
