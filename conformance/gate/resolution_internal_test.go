// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package gate

import (
	"testing"

	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/plugins/annotator/shape"
	"go.thesmos.sh/eidos/plugins/annotator/shape/contracts/lease"
	"go.thesmos.sh/eidos/plugins/annotator/shape/mixins/validates"
)

// The corpus resolves every reference, which is the point of the gate and also
// why these paths cannot be reached through it: a finding only exists where the
// resolver failed, and the corpus is the case where it did not. So the bags
// here are built by hand, stamped the way the classifier stamps them before the
// resolver runs.
func TestUnresolvedOn(t *testing.T) {
	t.Parallel()

	t.Run("reports a contract partner the resolver left raw", func(t *testing.T) {
		t.Parallel()
		bag := stamped(t, func(b *node.Method) {
			shape.MetaContracts.Set(b.EnsureMeta(), []string{lease.Name}, "test")
			shape.ContractRoleKey(lease.Name).Set(b.EnsureMeta(), "acquire", "test")
			shape.ContractPartnerKey(lease.Name, "release").Set(b.EnsureMeta(), "Release", "test")
		})
		got := unresolvedOn("Acquire", bag, contractRoles(), mixinSiblingParams())
		if len(got) != 1 {
			t.Fatalf("unresolvedOn = %d findings, want 1: %v", len(got), got)
		}
		if got[0].Value != "Release" || got[0].Param != "release" {
			t.Fatalf("finding = %+v, want release=Release", got[0])
		}
		if got[0].Axis != AxisContract {
			t.Fatalf("axis = %q, want %q", got[0].Axis, AxisContract)
		}
	})

	t.Run("accepts a contract partner the resolver rewrote", func(t *testing.T) {
		t.Parallel()
		// The qualified form is what a generator can split into a call. Reading
		// it as unresolved would fail every corpus fixture that works.
		bag := stamped(t, func(b *node.Method) {
			shape.MetaContracts.Set(b.EnsureMeta(), []string{lease.Name}, "test")
			shape.ContractRoleKey(lease.Name).Set(b.EnsureMeta(), "acquire", "test")
			shape.ContractPartnerKey(lease.Name, "release").
				Set(b.EnsureMeta(), "example.com/x.Contract.Release", "test")
		})
		if got := unresolvedOn("Acquire", bag, contractRoles(), mixinSiblingParams()); len(got) != 0 {
			t.Fatalf("unresolvedOn = %v, want none", got)
		}
	})

	t.Run("skips the role the callable fills itself", func(t *testing.T) {
		t.Parallel()
		// A member's own role is stamped as a role, not as a partner reference.
		// Treating the vocabulary as partner keys without that exclusion would
		// report a finding for every contract member in the corpus.
		bag := stamped(t, func(b *node.Method) {
			shape.MetaContracts.Set(b.EnsureMeta(), []string{lease.Name}, "test")
			shape.ContractRoleKey(lease.Name).Set(b.EnsureMeta(), "acquire", "test")
			shape.ContractPartnerKey(lease.Name, "acquire").Set(b.EnsureMeta(), "Acquire", "test")
		})
		if got := unresolvedOn("Acquire", bag, contractRoles(), mixinSiblingParams()); len(got) != 0 {
			t.Fatalf("unresolvedOn = %v, want none", got)
		}
	})

	t.Run("reports a mixin sibling param the resolver left raw", func(t *testing.T) {
		t.Parallel()
		bag := stamped(t, func(b *node.Method) {
			shape.MetaMixins.Set(b.EnsureMeta(), []string{validates.Name}, "test")
			shape.MixinParamKey(validates.Name, "fn").Set(b.EnsureMeta(), "Validate", "test")
		})
		got := unresolvedOn("Store", bag, contractRoles(), mixinSiblingParams())
		if len(got) != 1 {
			t.Fatalf("unresolvedOn = %d findings, want 1: %v", len(got), got)
		}
		if got[0].Axis != AxisMixin || got[0].Param != "fn" {
			t.Fatalf("finding = %+v, want mixin fn", got[0])
		}
	})

	t.Run("ignores a contract carrying no partner reference", func(t *testing.T) {
		t.Parallel()
		// Most contracts declare one role, so a member points at nothing. An
		// absent stamp is the normal case, not a resolution failure.
		bag := stamped(t, func(b *node.Method) {
			shape.MetaContracts.Set(b.EnsureMeta(), []string{lease.Name}, "test")
			shape.ContractRoleKey(lease.Name).Set(b.EnsureMeta(), "acquire", "test")
		})
		if got := unresolvedOn("Acquire", bag, contractRoles(), mixinSiblingParams()); len(got) != 0 {
			t.Fatalf("unresolvedOn = %v, want none", got)
		}
	})

	t.Run("ignores a mixin carrying no sibling param", func(t *testing.T) {
		t.Parallel()
		bag := stamped(t, func(b *node.Method) {
			shape.MetaMixins.Set(b.EnsureMeta(), []string{"idempotent"}, "test")
		})
		if got := unresolvedOn("Put", bag, contractRoles(), mixinSiblingParams()); len(got) != 0 {
			t.Fatalf("unresolvedOn = %v, want none", got)
		}
	})

	t.Run("tolerates a callable carrying no metadata", func(t *testing.T) {
		t.Parallel()
		// Reader access returns nil rather than allocating, so most callables in
		// a run arrive here with nothing at all.
		if got := unresolvedOn("Plain", nil, contractRoles(), mixinSiblingParams()); len(got) != 0 {
			t.Fatalf("unresolvedOn = %v, want none", got)
		}
	})
}

// Both registries are read rather than listed, so a classification that gains a
// sibling parameter upstream is measured on the next build.
func TestRegistryProjections(t *testing.T) {
	t.Parallel()

	t.Run("collects only mixins declaring sibling params", func(t *testing.T) {
		t.Parallel()
		got := mixinSiblingParams()
		if _, ok := got[validates.Name]; !ok {
			t.Fatalf("mixinSiblingParams is missing %q: %v", validates.Name, got)
		}
		if _, ok := got["idempotent"]; ok {
			t.Fatal("mixinSiblingParams carries a mixin with no sibling params")
		}
	})

	t.Run("collects every contract's role vocabulary", func(t *testing.T) {
		t.Parallel()
		got := contractRoles()
		if want := []string{"acquire", "release"}; len(got[lease.Name]) != len(want) {
			t.Fatalf("contractRoles[%q] = %v, want %v", lease.Name, got[lease.Name], want)
		}
	})
}

func TestQualified(t *testing.T) {
	t.Parallel()

	t.Run("accepts a name carrying a package path", func(t *testing.T) {
		t.Parallel()
		if !qualified("example.com/x.Contract.Release") {
			t.Fatal("a qualified name must be recognised")
		}
	})

	t.Run("declines a bare Go identifier", func(t *testing.T) {
		t.Parallel()
		// A Go identifier cannot contain a dot, which is what makes the test
		// unambiguous rather than a heuristic.
		if qualified("Release") {
			t.Fatal("a bare identifier must not be read as qualified")
		}
	})
}

// A gate that prints its findings in traversal order prints a different list
// each run, so the ordering is part of the output rather than a detail. It is
// also unreachable through the corpus, which has nothing to order.
func TestCompareUnresolved(t *testing.T) {
	t.Parallel()

	a := Unresolved{Callable: "Acquire", Param: "release"}
	b := Unresolved{Callable: "Publish", Param: "subscribe"}

	t.Run("orders by callable first", func(t *testing.T) {
		t.Parallel()
		if compareUnresolved(a, b) >= 0 {
			t.Fatal("Acquire must sort before Publish")
		}
		if compareUnresolved(b, a) <= 0 {
			t.Fatal("the comparison must be antisymmetric")
		}
	})

	t.Run("falls back to the parameter within one callable", func(t *testing.T) {
		t.Parallel()
		// Two roles on one member is the ordinary case for tx, which names both
		// commit and rollback from begin.
		first := Unresolved{Callable: "Begin", Param: "commit"}
		second := Unresolved{Callable: "Begin", Param: "rollback"}
		if compareUnresolved(first, second) >= 0 {
			t.Fatal("commit must sort before rollback")
		}
	})

	t.Run("reports two identical findings as equal", func(t *testing.T) {
		t.Parallel()
		if compareUnresolved(a, a) != 0 {
			t.Fatal("a finding must compare equal to itself")
		}
	})
}

func TestUnresolvedString(t *testing.T) {
	t.Parallel()

	t.Run("names the callable, the classification and the value", func(t *testing.T) {
		t.Parallel()
		// The message is the only thing a reader gets when the gate fails, so it
		// has to carry enough to find the directive that produced it.
		u := Unresolved{
			Callable: "Acquire", Axis: AxisContract,
			Name: lease.Name, Param: "release", Value: "Release",
		}
		want := `Acquire: contract lease: release="Release" is not a qualified name`
		if got := u.String(); got != want {
			t.Fatalf("String() = %q, want %q", got, want)
		}
	})
}

// stamped builds a method and applies the given stamps to its metadata bag.
func stamped(t *testing.T, apply func(*node.Method)) *meta.Bag {
	t.Helper()
	m := &node.Method{Name: "Subject"}
	apply(m)
	return m.Meta()
}
