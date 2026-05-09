// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package atomic registers the //testkit:atomic consumer. The
// directive declares the method's mutation is atomic — on failure,
// observable state is unchanged (no partial writes).
//
// The contract subtest [suite.AssertAtomicNoTrace] forces the failure
// path with zero-valued args and asserts state equality before vs.
// after via [reflect.DeepEqual] (or a consumer-supplied
// [WithStateEqual]). The failure path is observable only when the
// method declares a sentinel error via //testkit:errors — without
// one, "the call failed" cannot be triggered deterministically and
// the contract is unverifiable. The consumer rejects this composition
// at consume time so the suite never tries to emit an unverifiable
// contract.
package atomic

import (
	"errors"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
	specerrors "go.thesmos.sh/testkit/generator/spec/errors"
)

// Presence is the empty payload type attached when atomic fires.
type Presence struct{}

func init() {
	spec.RegisterConsumer(directive.Atomic, consume)
}

// Has reports whether the method carries //testkit:atomic.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Atomic) }

// consume validates that the method declares at least one
// //testkit:errors sentinel — the failure path is unobservable
// without one, and the atomic contract is unverifiable.
func consume(method *spec.Method, _ directive.Directive, _ *spec.Data, _ *generator.Package) error {
	p, ok := specerrors.Get(method)
	if !ok || len(p.Sentinels) == 0 {
		return errors.New(
			"atomic: method must declare //testkit:errors with at least one sentinel " +
				"so the failure path is deterministically observable",
		)
	}
	spec.Set(&method.Attachments, directive.Atomic, Presence{})
	return nil
}
