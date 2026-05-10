// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package saga registers the //testkit:saga consumer. The directive
// promotes the entry method to the Saga composite-tier shape:
// either every step completes in order, or the partial run executes
// full compensation in reverse.
//
// Directive form:
//
//	//testkit:saga Step1 Step2 Step3
//
// Validation: every named step method must exist on the same
// interface. Argument count: at least one.
package saga

import (
	"errors"
	"fmt"

	"go.thesmos.sh/testkit/generator"
	"go.thesmos.sh/testkit/generator/directive"
	"go.thesmos.sh/testkit/generator/spec"
)

// Payload carries the resolved list of step-method names.
type Payload struct {
	// Steps is the ordered list of step methods that compose the
	// saga. Each is validated to exist on the same interface.
	Steps []string
}

func init() {
	spec.RegisterConsumer(directive.Saga, consume)
}

// Get retrieves the resolved [Payload]. Returns (zero, false) when
// the method carries no //testkit:saga directive.
func Get(m *spec.Method) (Payload, bool) {
	return spec.Get[Payload](m.Attachments, directive.Saga)
}

// Has reports whether the method carries //testkit:saga.
func Has(m *spec.Method) bool { return spec.Has(m.Attachments, directive.Saga) }

func consume(method *spec.Method, dir directive.Directive, data *spec.Data, _ *generator.Package) error {
	if len(dir.Args) == 0 {
		return errors.New("saga: needs at least one step")
	}
	steps := make([]string, len(dir.Args))
	for i, want := range dir.Args {
		if !methodExists(data, want) {
			return fmt.Errorf("saga: step method %q not found on interface %s",
				want, data.Interface.Name)
		}
		steps[i] = want
	}
	spec.Set(&method.Attachments, directive.Saga, Payload{Steps: steps})
	return nil
}

func methodExists(data *spec.Data, name string) bool {
	for _, m := range data.Interface.Methods {
		if m.Name == name {
			return true
		}
	}
	return false
}
