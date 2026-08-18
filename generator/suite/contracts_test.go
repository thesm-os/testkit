// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite_test

import (
	"slices"
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/ifabsent"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/ifmatch"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/outbox"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/pool"

	"go.thesmos.sh/testkit"
	"go.thesmos.sh/testkit/generator/suite"
)

// The roles and keys this generator selects on are spelled as constants, and a
// spelling is only as good as what checks it.
//
// eidos publishes each contract's vocabulary as a slice rather than as named
// constants, so nothing but this holds the two together. Without it a role
// renamed upstream selects nothing, every contract check disappears, and the
// corpus still compiles — which is the failure mode the whole tier exists to
// prevent.
func TestContractVocabularyIsUpstream(t *testing.T) {
	t.Parallel()

	t.Run("names roles the contracts declare", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, slices.Contains(ifabsent.Roles, suite.ContractIfAbsentRole),
			"if-absent declares the role the check selects on")
		testkit.True(t, slices.Contains(ifmatch.Roles, suite.ContractIfMatchRole),
			"if-match declares the role the check selects on")
		testkit.True(t, slices.Contains(outbox.Roles, suite.ContractOutboxRole),
			"outbox declares the role the check selects on")
		testkit.True(t, slices.Contains(outbox.Roles, suite.ContractOutboxPartner),
			"and the partner role the check reaches through")
		testkit.True(t, slices.Contains(pool.Roles, suite.ContractPoolGet),
			"pool declares the producing role the borrow reaches through")
		testkit.True(t, slices.Contains(pool.Roles, suite.ContractPoolPut),
			"and the returning role the borrow smoke selects on")
	})

	t.Run("names the predicate role the contract declares", func(t *testing.T) {
		t.Parallel()
		testkit.True(t, slices.Contains(ifmatch.Roles, suite.ContractIfMatchMatch),
			"if-match declares the role naming its predicate")
	})
}
