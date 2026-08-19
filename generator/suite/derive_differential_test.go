// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Internal for the reason derive_stamps_test.go is: the fixtures
// populate the unexported stamp projections through the real keys on
// real bags, which only this package's own constructors reach.
package suite

import (
	"testing"

	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/reader"
	"go.thesmos.sh/eidos/plugins/annotator/shape/detectors/writer"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/eventually"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit"
	vocab "go.thesmos.sh/testkit/engine/suite"
)

// contractMember stamps one method into a contract, role optional —
// the membership is what the differential's store arm reads.
func contractMember(name, contract, role string) Method {
	bag := sdk.NewBag()
	shape.MetaContracts.Set(bag, []string{contract}, "test")
	if role != "" {
		shape.ContractRoleKey(contract).Set(bag, role, "test")
	}
	m := stampMethod(name, "")
	m.Contracts = shape.Contracts(bag)
	roles, partners := contractDataOf(bag)
	m.contractRoles = roles
	m.contractPartners = partners
	return m
}

// differentialCase is one interface shape and the claim its derived
// reference speaks; a nil want asserts no row derives.
type differentialCase struct {
	name  string
	iface Iface
	want  string
}

func (c differentialCase) Name() string { return c.name }

func TestDifferentialWordsTheReference(t *testing.T) {
	t.Parallel()

	testkit.TableTest(t, []differentialCase{
		{
			// Reconciled wording: the corpus wavered between "the
			// subject" (store) and the token (journal, catalog); the
			// token says more and every other row already speaks it.
			"a writer beside a modelled read agrees plainly",
			Iface{Name: "Store", Token: "store", Qualifier: "store", Methods: []Method{
				stampMethod("Put", writer.Name),
				stampMethod("Get", reader.Name),
			}},
			"every operation sequence leaves the store agreeing with the reference",
		},
		{
			"a seeded read-only surface agrees with a seeded reference",
			Iface{Name: "Catalog", Token: "catalog", Qualifier: "catalog", Methods: []Method{
				stampMethod("Lookup", reader.Name),
			}},
			"every read sequence leaves the catalog agreeing with a reference seeded identically",
		},
		{
			"an outcome-speaking contract oracle names its role pair",
			Iface{Name: "Lease", Token: "lease", Qualifier: "lease", Methods: []Method{
				contractMember("Acquire", ContractLease, ""),
				contractMember("Release", ContractLease, ""),
			}},
			"every acquire-release sequence leaves the lease agreeing with the reference on every outcome",
		},
		{
			"a plain contract oracle speaks operation",
			Iface{Name: "Journal", Token: "journal", Qualifier: "journal", Methods: []Method{
				contractMember("Append", ContractChain, ""),
			}},
			"every operation sequence leaves the journal agreeing with the reference",
		},
		{
			"a produced cursor drains, writer-opener named",
			Iface{Name: "Log", Token: "log", Qualifier: "log", Methods: []Method{
				stampMethod("Append", writer.Name),
				contractMember("Scan", ContractCursor, ContractCursorOpen),
			}},
			"every append-scan sequence drains the same entries as the reference, in order",
		},
		{
			"no oracle-shaped surface, no row",
			Iface{Name: "Notifier", Token: "notifier", Qualifier: "notifier", Methods: []Method{
				stampMethod("Ping", ""),
			}},
			"",
		},
	}, func(t *testing.T, tc differentialCase) {
		plans, refusals := Differential{}.Derive(tc.iface)
		testkit.Len(t, refusals, 0, "a derivable oracle refuses nothing")
		if tc.want == "" {
			testkit.Len(t, plans, 0, "absence is the coverage header's, not a row's")
			return
		}
		testkit.Len(t, plans, 1, "one reference, one row")
		id, err := plans[0].ID.Render()
		testkit.NoError(t, err, "the derived ID is well formed")
		testkit.Equal(t, id, vocab.FamilyID(vocab.FamilyModel, tc.iface.Token, vocab.SegDifferential),
			"the family-scoped differential ID")
		testkit.Equal(t, plans[0].Class, vocab.ClassDifferential, "under the differential class")
		testkit.Equal(t, plans[0].Claim, tc.want, "the claim speaks the derived reference's kind")
	})
}

func TestDifferentialRefusesTheUnmodellable(t *testing.T) {
	t.Parallel()

	t.Run("an oracle-defeating mixin refuses with its reason", func(t *testing.T) {
		t.Parallel()
		iface := Iface{Name: "Feed", Token: "feed", Qualifier: "feed", Methods: []Method{
			stampMethod("Put", writer.Name, eventually.Name),
			stampMethod("Get", reader.Name),
		}}
		plans, refusals := Differential{}.Derive(iface)
		testkit.Len(t, plans, 0, "a claim without a pinned wording must not render")
		testkit.Len(t, refusals, 1, "the twin floor is a named gap, not a silent absence")
		testkit.Contains(t, refusals[0].Remedy, "ref=", "the remedy names the override seam")
	})

	t.Run("two contract oracles refuse rather than choose", func(t *testing.T) {
		t.Parallel()
		iface := Iface{Name: "Mixed", Token: "mixed", Qualifier: "mixed", Methods: []Method{
			contractMember("Acquire", ContractLease, ""),
			contractMember("Get", ContractPool, ""),
		}}
		plans, refusals := Differential{}.Derive(iface)
		testkit.Len(t, plans, 0, "choosing an oracle would invent semantics")
		testkit.Len(t, refusals, 1, "the ambiguity is named")
		testkit.Contains(t, refusals[0].Why, ContractLease, "the refusal lists the contestants")
		testkit.Contains(t, refusals[0].Why, ContractPool, "both of them")
	})
}
