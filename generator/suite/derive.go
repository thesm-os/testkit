// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package suite

import (
	"slices"

	"go.thesmos.sh/testkit/generator/suite/projection"
)

// Iface is one interface as the derivers read it: the projections the
// plugin already computes — [Method], [Fixture] — plus the directive
// facts no projection carries. It exists so derivation is
// unit-testable without the eidos pipeline, and it invents nothing:
// every field is either an incumbent projection or a directive the
// plugin shell reads once.
type Iface struct {
	// Name is the interface's exported name ("Log"); Token its Go
	// identifier qualifier ("log", "paginatedReader"), which every
	// emitted declaration is named from.
	Name  string
	Token string

	// Package is the interface's own import path, which a body naming
	// something declared beside it — a miss sentinel — resolves
	// against.
	Package string

	// Qualifier is the interface's word inside a family-scoped ID
	// ("log", "paginated-reader"). Slug rather than identifier: the
	// grammar admits a-z, 0-9 and '-' only, so the two qualifiers
	// diverge the moment an interface name has two words.
	Qualifier string

	Methods []Method

	// Fixture is the derived input set; a draw it cannot deliver turns
	// a method's derived families into one refusal.
	Fixture Fixture
}

// seeded reports the seed-seam interface: nothing on it can write, so
// the harness receives its corpus from the pools and the derived
// claims speak "seeded" rather than "derived".
func (f Iface) seeded() bool {
	return !slices.ContainsFunc(f.Methods, writesSomething)
}

// DeriverName identifies a deriver in the registry and in refusal
// attributions.
type DeriverName string

// The deriver names, one per rule family.
const (
	// DeriverSignature derives the per-method families: smoke, cancel,
	// deadline, nilcontext, zero-on-error.
	DeriverSignature DeriverName = "signature"

	// DeriverStamps derives the deterministic stamp families: the
	// mixin axis (idempotent) and the detector axis (the reader
	// miss/hit/count set).
	DeriverStamps DeriverName = "stamps"

	// DeriverLaws plans the model tier's law rows — which laws tiers
	// selects, on which legs, under which claims.
	DeriverLaws DeriverName = "laws"

	// DeriverDifferential plans the model tier's reference-comparison
	// row, worded by the derived reference's kind.
	DeriverDifferential DeriverName = "differential"
)

// Deriver is one rule family: the interface's projections in, plans
// and refusals out. A deriver returns every check its rules license
// and a refusal for every check its rules reach but cannot complete —
// the two lists together are the family's whole answer, and silence
// is not in the vocabulary.
type Deriver interface {
	Name() DeriverName
	Derive(f Iface) ([]projection.CheckPlan, []Refusal)
}

// Refusal is a check the rules reached and could not derive: what it
// would have asserted, why it cannot, and the consumer action that
// closes the gap. Refusals render into the generated header — a claim
// the reader cannot see refused reads as a claim the run checks.
type Refusal struct {
	Deriver DeriverName
	What    string
	Why     string
	Remedy  string
}

// Registry returns the derivers in derivation order. Closed like the
// projection's variant sets: the conformance census holds this list
// and the rule tables equal, so a family added to derivation-rules.md
// without a deriver is a build failure.
func Registry() []Deriver {
	return []Deriver{
		Signature{},
		Stamps{},
		Laws{},
		Differential{},
	}
}

// argsRefusal folds a method whose draws the fixture cannot supply
// into the one refusal its whole derived family set shares, false
// when every draw has a supplier.
func argsRefusal(d DeriverName, f Iface, m Method, what string) (Refusal, bool) {
	arg, field, missing := undeliverableArgs(f.Fixture, m.ArgFields)
	if !missing {
		return Refusal{}, false
	}
	return Refusal{
		Deriver: d,
		What:    m.Name + what,
		Why:     "its " + arg + " argument needs a value " + field.Reason(),
		Remedy: "supply the value through " + f.Name + "WithFixture and write the check as " +
			f.Name + "On" + m.Name,
	}, true
}

// callOf renders the method's invocation: the context first when the
// method takes one, then every draw through the fixture policy.
func callOf(m Method) projection.CallPlan {
	var args []projection.Expr
	if m.TakesContext() {
		args = append(args, projection.ExprCtx)
	}
	for _, field := range m.ArgFields {
		args = append(args, projection.FixtureCall(projection.ExprFixture, field))
	}
	return projection.CallPlan{Method: m.Name, Args: args}
}

// InventoryOf runs every deriver over the interface and folds their
// answers into the one inventory every artifact projects from.
//
// The registry's order is the emission order, and it is preserved: the
// generated file's sections read in the order the rules are written
// down, so a reader following derivation-rules.md top to bottom is
// following the output too.
//
// Refusals accumulate beside the plans rather than short-circuiting.
// A deriver refusing one method's family says nothing about the next
// deriver's answer, and a run that stopped at the first refusal would
// report one gap where the header owes the reader all of them.
func InventoryOf(f Iface) (projection.Inventory, []Refusal) {
	inv := projection.Inventory{Iface: f.Name, Token: f.Token}
	var refusals []Refusal
	for _, d := range Registry() {
		plans, refused := d.Derive(f)
		inv.Checks = append(inv.Checks, plans...)
		refusals = append(refusals, refused...)
	}
	return inv, refusals
}
