// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package suite generates a contract-conformance test runner for a
// Go interface. Each Generate call emits one file: an
// `Assert<Iface>Contract(t, factory, opts...)` driver that runs the
// per-method, per-shape, per-directive contract subtests against any
// implementation supplied via the factory, plus an
// `Assert<Iface>ContractAcrossImpls(t, factories...)` driver that
// runs the same contract once per named implementation.
//
// Suite reuses the [spec.Data] analysis result — same shape catalog,
// same directive payloads, same Method.Attachments map — that stub
// consumes. Where stub generates a recording test double, suite
// generates verification: every method's contract is checked against
// the impl the consumer hands in.
//
// Multi-impl conformance is one call:
//
//	suite.AssertStoreContractAcrossImpls(t,
//	    suite.NamedFactory[basic.Store]{Name: "InMemory", Factory: newInMem},
//	    suite.NamedFactory[basic.Store]{Name: "Postgres",  Factory: newPostgres},
//	    suite.NamedFactory[basic.Store]{Name: "Stub+Delegate", Factory: stubDelegateFactory},
//	)
//
// The driver runs every contract once per impl with `t.Run(name, ...)`.
// Cross-method invariants (read-after-write, delete-removes,
// stream-reflects-mutations) are wired via `//testkit:cross`
// directives and dispatch to the runtime [suite/cross.go] helpers.
package suite
