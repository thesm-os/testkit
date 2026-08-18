// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"slices"
	"strings"

	vocab "go.thesmos.sh/testkit/engine/suite"
	"go.thesmos.sh/testkit/generator/suite/projection"
	"go.thesmos.sh/testkit/generator/tiers"
)

// Differential plans the model tier's reference-comparison row: one
// per interface whose oracle derives, worded by the reference's kind.
// The reference RESOLUTION — adapters, inert arms, type arguments —
// stays the model plugin's; this deriver answers existence and claim,
// which is what the lock and index need. The body's assert seam fills
// when the emission templates land.
type Differential struct{}

var _ Deriver = Differential{}

// Name implements [Deriver].
func (Differential) Name() DeriverName { return DeriverDifferential }

// Derive implements [Deriver]. The arms, in precedence order:
//
//   - a mixin that defeats every oracle refuses — the twin floor's
//     wording has no corpus pin yet, and a claim without a spelling
//     must not render;
//   - the cursor contract drains: the writer and the opener name the
//     sequence, the comparison is ordered entries;
//   - a contract with a store row agrees — on every outcome where
//     the oracle speaks error semantics, plainly otherwise;
//   - a writer beside an oracle-modelled read shape agrees plainly;
//   - a seeded read-only surface agrees with a reference seeded
//     identically.
//
// An interface no arm reaches owes no differential and gets no row —
// that absence is the coverage header's to report, not a refusal.
func (Differential) Derive(f Iface) ([]projection.CheckPlan, []Refusal) {
	for _, m := range f.Methods {
		for _, mixin := range m.Mixins {
			if reason, defeated := tiers.DefeatsOracles(mixin); defeated {
				return nil, []Refusal{{
					Deriver: DeriverDifferential,
					What:    "the differential leg for " + f.Name,
					Why:     reason,
					Remedy:  "override with ref=, or the twin-floor wording earns a corpus pin first",
				}}
			}
		}
	}
	claim, refusals, derived := differentialClaim(f)
	if !derived {
		return nil, refusals
	}
	plan := projection.CheckPlan{
		ID:          projection.IDPlan{Family: vocab.FamilyModel, Qualifier: f.Token, Seg: vocab.SegDifferential},
		Class:       vocab.ClassDifferential,
		Claim:       claim,
		Body:        projection.DifferentialLeg{},
		Falsifiable: vocab.Argued(argueProofsPending),
	}
	if defect, proven := observationDefect(f); proven {
		plan.Falsifiable = vocab.Proven()
		plan.Defect = defect
	}
	return []projection.CheckPlan{plan}, refusals
}

// differentialClaim words the row by the derived reference's kind,
// false where no oracle derives.
func differentialClaim(f Iface) (string, []Refusal, bool) {
	if opener := roleMethod(f.Methods, ContractCursor, ContractCursorOpen); opener != nil {
		if writer := firstWriter(f.Methods); writer != nil {
			sequence := strings.ToLower(writer.Name) + "-" + strings.ToLower(opener.Name)
			return DifferentialDrainClaim(sequence), nil, true
		}
	}
	if roles, outcomes, refusal, held := storeContract(f); held {
		if refusal != nil {
			return "", []Refusal{*refusal}, false
		}
		sequence := seqOperation
		if outcomes && len(roles) >= 2 {
			// The outcome-compared oracles speak their protocol's
			// role pair; the plain ones speak "operation" — the
			// corpus's own split (lease vs chain).
			sequence = roles[0] + "-" + roles[1]
		}
		return DifferentialAgreeClaim(sequence, f.Token, false, outcomes), nil, true
	}
	readable := oracleReadable(f)
	switch {
	case readable && f.seeded():
		return DifferentialAgreeClaim(seqRead, f.Token, true, false), nil, true
	case readable && firstWriter(f.Methods) != nil:
		return DifferentialAgreeClaim(seqOperation, f.Token, false, false), nil, true
	}
	return "", nil, false
}

// storeContract finds the one contract the interface carries whose
// oracle ships — tiers' rows, or the interim table below. Two such
// contracts is a choice this deriver must not invent, so it answers
// with a refusal instead of picking.
func storeContract(f Iface) (roles []string, outcomes bool, refusal *Refusal, held bool) {
	var found []string
	for _, m := range f.Methods {
		for _, contract := range m.Contracts {
			if slices.Contains(found, contract) {
				continue
			}
			spec, shipped := tiers.ContractStore(contract)
			interim, interimShipped := interimStores()[contract]
			if !shipped && !interimShipped {
				continue
			}
			found = append(found, contract)
			outcomes = interim || len(spec.Errs) > 0
		}
	}
	switch len(found) {
	case 0:
		return nil, false, nil, false
	case 1:
		return tiers.ContractRoles(found[0]), outcomes, nil, true
	default:
		return nil, false, &Refusal{
			Deriver: DeriverDifferential,
			What:    "the differential leg for " + f.Name,
			Why:     "two contract oracles derive (" + strings.Join(found, ", ") + ") and choosing one would invent semantics",
			Remedy:  "override with ref= naming the oracle the interface means",
		}, true
	}
}

// interimStores lists contract oracles the engine ships but the tiers
// catalogue does not yet table, with the outcome semantics the future
// spec row will carry. The rows land in tiers with the model plugin's
// migration, because a catalogue row changes the incumbent's emission
// today — the same holding pattern as the poison extra-rule.
func interimStores() map[string]bool {
	return map[string]bool{
		ContractPool: true,
	}
}

// firstWriter is the first method the shape annotator classified as a
// write, nil where nothing writes.
func firstWriter(methods []Method) *Method {
	for i := range methods {
		if writesSomething(methods[i]) {
			return &methods[i]
		}
	}
	return nil
}

// oracleReadable reports whether any non-writing method's shape is
// one the shipped oracles model — the read half a comparison needs.
func oracleReadable(f Iface) bool {
	return firstOracleReader(f) != nil
}

// firstOracleReader is the first non-writing method whose shape the
// shipped oracles model. Nil where none reads.
func firstOracleReader(f Iface) *Method {
	for i := range f.Methods {
		m := &f.Methods[i]
		s := m.Shape()
		if s == "" || writesSomething(*m) {
			continue
		}
		if _, ok := tiers.MapStoreOp(s); ok {
			return m
		}
		if _, ok := tiers.KeyedStoreOp(s); ok {
			return m
		}
		if _, ok := tiers.CollectionOp(s); ok {
			return m
		}
	}
	return nil
}
