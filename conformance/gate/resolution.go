// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins"
)

// Unresolved is one sibling reference a directive named that still holds the
// identifier the author wrote rather than a qualified name.
//
// A directive that names a partner — a contract's `release=Release`, a mixin's
// `fn=Validate` — records that name verbatim when the shape classifier stamps
// it. eidos's contract resolver, which runs one priority bucket later, is what
// rewrites it into `<pkg-path>.<Type>.<Method>` and back-stamps the membership
// onto the callable it names. A raw name surviving to the end of a run means
// the resolver either did not run or could not find the sibling.
type Unresolved struct {
	// Callable is the method or function carrying the directive.
	Callable string

	// Axis is [AxisContract] or [AxisMixin].
	Axis string

	// Name is the contract or mixin the reference belongs to.
	Name string

	// Param is the partner's role for a contract, or the parameter key for a
	// mixin.
	Param string

	// Value is what the stamp holds.
	Value string
}

// String renders one finding as a line a reader can act on.
func (u Unresolved) String() string {
	return fmt.Sprintf("%s: %s %s: %s=%q is not a qualified name",
		u.Callable, u.Axis, u.Name, u.Param, u.Value)
}

// Resolution reports every sibling reference in the corpus that the shape
// resolver did not rewrite. An empty result means the resolver ran and found
// every partner a directive named.
//
// This is the half [Coverage] cannot see. Coverage counts which classifications
// were stamped, and the callable that declares a contract stamps it whether or
// not the resolver ever runs — so a pipeline registering the classifier without
// its resolver produces a complete coverage report, an unqualified partner
// reference no generator can turn into a call, and no signal at all. That
// configuration shipped once.
//
// # Hazards
//
// Runs the corpus through a second pipeline, so it costs another full parse.
// Both the classifier and the resolver come from [generator.Annotators], which
// is the point: this measures the set a real run registers rather than one
// assembled here.
func Resolution(ctx context.Context, root string, patterns ...string) ([]Unresolved, error) {
	pipe, err := run(ctx, root, patterns...)
	if err != nil {
		return nil, err
	}

	siblingParams := mixinSiblingParams()
	roles := contractRoles()
	out := make([]Unresolved, 0)

	for _, m := range pipe.Store().Nodes().Methods().Items() {
		out = append(out, unresolvedOn(m.Name, m.Meta(), roles, siblingParams)...)
	}
	// Free functions carry the same directives. A contract whose roles are
	// filled by package-level functions rather than by methods on one type
	// resolves through a different scope, and skipping them here would leave
	// that path unmeasured.
	for _, f := range pipe.Store().Nodes().Functions().Items() {
		out = append(out, unresolvedOn(f.Name, f.Meta(), roles, siblingParams)...)
	}

	slices.SortFunc(out, compareUnresolved)
	return out, nil
}

// compareUnresolved orders findings by callable and then by parameter.
//
// Named rather than inline because the corpus is the case where there are no
// findings to order, so an inline comparator would be code the gate can never
// reach and never check. Ordering matters for the same reason any generated
// output's does: a gate that prints its findings in map order prints a
// different diff every run.
func compareUnresolved(a, b Unresolved) int {
	return cmp.Or(
		cmp.Compare(a.Callable, b.Callable),
		cmp.Compare(a.Param, b.Param),
	)
}

// unresolvedOn collects the raw references on one callable's metadata bag.
func unresolvedOn(
	callable string,
	bag *meta.Bag,
	roles map[string][]string,
	siblingParams map[string][]string,
) []Unresolved {
	if bag == nil {
		return nil
	}
	var out []Unresolved

	for _, name := range shape.Contracts(bag) {
		role, _ := shape.ContractRoleKey(name).Get(bag)
		for _, partnerRole := range roles[name] {
			// A callable fills one role and points at the others. The stamp for
			// its own role is the role itself, not a partner reference.
			if partnerRole == role {
				continue
			}
			value, ok := shape.ContractPartnerKey(name, partnerRole).Get(bag)
			if !ok || value == "" || qualified(value) {
				continue
			}
			out = append(out, Unresolved{
				Callable: callable, Axis: AxisContract,
				Name: name, Param: partnerRole, Value: value,
			})
		}
	}

	for _, name := range shape.Mixins(bag) {
		for _, param := range siblingParams[name] {
			value, ok := shape.MixinParamKey(name, param).Get(bag)
			if !ok || value == "" || qualified(value) {
				continue
			}
			out = append(out, Unresolved{
				Callable: callable, Axis: AxisMixin,
				Name: name, Param: param, Value: value,
			})
		}
	}
	return out
}

// qualified reports whether a stamp holds a qualified name. The test is the
// same conservative one the resolver uses to decide a value needs no rewriting:
// a Go source identifier never contains a dot, so its presence is unambiguous.
func qualified(value string) bool {
	return strings.Contains(value, ".")
}

// mixinSiblingParams maps each registered mixin to the parameter keys whose
// values name a sibling callable — the callable-kinded entries of the param
// schema, now that each key declares its kind beside it.
//
// Read from the registry rather than listed here, for the same reason
// [Registered] is: a mixin that gains a sibling parameter upstream starts being
// measured on the next build instead of when someone remembers.
func mixinSiblingParams() map[string][]string {
	out := make(map[string][]string)
	for _, m := range mixins.All() {
		var keys []string
		for _, p := range m.Params {
			if p.Kind == shape.KindCallable {
				keys = append(keys, p.Key)
			}
		}
		if len(keys) > 0 {
			out[m.Name] = keys
		}
	}
	return out
}

// contractRoles maps each registered contract to its role vocabulary, which is
// the set of partner keys a member may carry.
func contractRoles() map[string][]string {
	out := make(map[string][]string, len(contracts.All()))
	for _, c := range contracts.All() {
		out[c.Name] = slices.Clone(c.Roles)
	}
	return out
}
