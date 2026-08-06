// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package fault

import (
	"fmt"

	"go.thesmos.sh/eidos/emit"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"

	"go.thesmos.sh/testkit/generator/stub"
)

// Capability is the label the plugin advertises, so a later generator can
// declare a dependency on the fault surface existing.
const Capability = "fault"

// SlotName is the [emit.File] slot the contributions land in.
//
// `bottom` rather than the `top` the stub generator uses, because the fault
// surface has to render after the types it hangs off, and the slot is the only
// thing that guarantees it. Ordering within one slot is by resolved plugin
// order, and this plugin's first appearance there is as an annotator — ahead
// of every generator, including its host. Two contributions to `top` would
// therefore put the helpers above the double they extend.
//
// Go does not care about declaration order, so the previous arrangement
// compiled; it just read backwards.
const SlotName = "bottom"

// KindHelpers and KindTests are the plugin-defined emit kinds. The backend
// resolves a template by the kind's string value, so each constant doubles as
// the name the matching template defines.
const (
	KindHelpers sdk.Kind = "fault.helpers"
	KindTests   sdk.Kind = "fault.test"
)

// Sentinel is one error variable a method's double can be made to fail with.
type Sentinel struct {
	// Name is the source-side variable identifier, e.g. `ErrNotFound`.
	Name string

	// Helper is the generated method's identifier, e.g. `FaultNotFound`.
	Helper string

	// Ref qualifies the variable from wherever the double is routed to. A
	// double redirected into its own package cannot reach the sentinel
	// unqualified, and the source package is known at Generate time.
	Ref *emit.Expr
}

// Method is one method's fault surface, projected onto the double's method.
//
// The embedded [stub.Method] is taken from the queued double rather than
// re-derived from the source: the identifiers these helpers hang off —
// `<Iface><Method>Stub`, `<Iface><Method>Call` — are the stub generator's
// convention, and a second derivation here would be free to disagree with it.
type Method struct {
	stub.Method

	// Sentinels are the error variables this method can be made to fail with,
	// each gaining a one-shot helper.
	Sentinels []Sentinel

	// Retry is the attempt a scheduled fault stops firing on, zero when the
	// method configures none.
	Retry int

	// Partition is the recorded-call field per-key targeting matches against,
	// empty when the method configures none.
	Partition string
}

// Helpers is the emit value rendered into the double's file.
type Helpers struct {
	sdk.BaseEmit

	// IfaceName is the source interface's identifier, used in the generated
	// doc comments.
	IfaceName string

	Methods []Method
}

// Kind returns [KindHelpers].
func (*Helpers) Kind() sdk.Kind { return KindHelpers }

// Tests is the emit value rendered into the double's companion file.
//
// The companion lands in an external test package, so it reaches neither the
// double nor its constructor unqualified. Both references follow the double's
// routing, which is not decided until Layout — hence
// [emit.OutputPackageSetter].
type Tests struct {
	sdk.BaseEmit

	// TypeName is the double's identifier, which names the generated checks.
	TypeName string

	// StubRef qualifies the generated double. Set during Generate against the
	// source package as a provisional value, then corrected once routing
	// resolves.
	StubRef *emit.Expr

	// CtorRef qualifies the double's constructor, which lives beside it and
	// therefore follows the same routing.
	CtorRef *emit.Expr

	Methods []Method
}

// Kind returns [KindTests].
func (*Tests) Kind() sdk.Kind { return KindTests }

// SetOutputPackages repoints the references at wherever Layout routed the
// double's primary output.
//
// An empty path means the Target resolved without a derivable import path,
// which centralised layout does. The provisional source-package reference is
// left in place rather than replaced with a bare name: a wrong package is a
// compile error naming the symbol, while a bare name silently binds to
// whatever else is in scope.
func (t *Tests) SetOutputPackages(byTag map[string]string) {
	if path := byTag[""]; path != "" {
		t.StubRef = sdk.NewExternal(path, t.TypeName)
		t.CtorRef = sdk.NewExternal(path, "New"+t.TypeName)
	}
}

// Generate contributes the fault surface into whatever doubles this run is
// already producing.
//
// # Why it reads the emit queue rather than the source directive
//
// Layout materialises an origin-anchored slot contribution through FileFor,
// which is lookup-or-create. A contributor that emits where the host did not
// therefore creates the file on its own — a fragment of methods hanging off
// types nothing declared. Requires does not prevent that: an unsatisfied
// requirement is ignored silently.
//
// Reading the queued [stub.Stub] is exact where a directive check would only
// be close. It also inherits the double's own method set: a method the stub
// generator dropped — integration-only, say — is absent from the queue, so no
// helper can be emitted against a method the double does not carry.
//
// Interfaces are still walked through ctx.Reader rather than off the queue,
// because the reads a plugin makes through the Reader are its cache key. A
// generator that read only the emit graph would cache against nothing.
func (*Plugin) Generate(ctx *sdk.GeneratorContext) error {
	doubles := doublesByOrigin(ctx)
	if len(doubles) == 0 {
		return nil
	}

	c := sdk.NewProvenance(Name, sdk.EmitTarget{})
	for _, iface := range ctx.Reader.Interfaces().Slice() {
		double, hosted := doubles[node.Node(iface)]
		if !hosted {
			continue
		}
		methods := methodsOf(ctx, iface, double)
		if len(methods) == 0 {
			continue
		}

		base := sdk.BaseEmit{
			OriginNode: iface,
			SetByName:  c.SetBy(),
			SourcePos:  iface.Pos(),
		}
		testBase := base
		testBase.OutputTagName = GoTestOutputTag

		for _, value := range []sdk.EmitNode{
			&Helpers{
				BaseEmit:  base,
				IfaceName: iface.Name,
				Methods:   methods,
			},
			&Tests{
				BaseEmit: testBase,
				TypeName: double.TypeName,
				StubRef:  sdk.NewExternal(iface.Package, double.TypeName),
				CtorRef:  sdk.NewExternal(iface.Package, "New"+double.TypeName),
				Methods:  methods,
			},
		} {
			prov := c.Provenance(string(value.Kind()) + "." + iface.Name)
			if err := ctx.Store.Emit().AppendOriginSlot(iface, SlotName, value, prov); err != nil {
				return fmt.Errorf("%s: append %s slot for %q: %w", Name, value.Kind(), iface.Name, err)
			}
		}
	}
	return nil
}

// doublesByOrigin indexes the doubles this run has queued, by the interface
// each stands in for.
//
// The type assertion is the whole guard: it matches only a value the stub
// generator built, so a same-named kind from somewhere else cannot be mistaken
// for a host.
func doublesByOrigin(ctx *sdk.GeneratorContext) map[node.Node]*stub.Stub {
	pending := ctx.Store.Emit().PendingOriginSlots()
	out := make(map[node.Node]*stub.Stub, len(pending))
	for i := range pending {
		if double, ok := pending[i].Item.(*stub.Stub); ok {
			out[pending[i].Origin] = double
		}
	}
	return out
}

// methodsOf projects every method the double carries that the source
// configured a fault for.
//
// Driven off the double's method set rather than the interface's declarations,
// because after flattening the two differ: a method an embedded interface
// contributed is carried by the double and absent from the declarations, and
// walking the declarations would silently skip its fault configuration.
//
// A method configuring nothing is dropped rather than emitted empty: the
// helpers are the whole contribution, and a method with none would otherwise
// contribute blank lines to somebody else's file.
func methodsOf(ctx *sdk.GeneratorContext, iface *node.Interface, double *stub.Stub) []Method {
	// Indexed rather than ranged by value: stub.Method is wide enough that
	// copying one per iteration is measurable, and nothing here needs a copy.
	out := make([]Method, 0, len(double.Methods))
	for i := range double.Methods {
		host := double.Methods[i]
		m := host.Source
		projected := Method{
			Method:    host,
			Sentinels: sentinelsOf(iface, m),
			Retry:     Retry(m.Meta()),
			Partition: partitionOf(ctx, iface, m, host),
		}
		if len(projected.Sentinels) == 0 && projected.Retry == 0 && projected.Partition == "" {
			continue
		}
		out = append(out, projected)
	}
	return out
}

// sentinelsOf lifts the stamped sentinel names into rendered form. The names
// and the collision rule belong to [Annotate]; what is added here is the
// qualified reference the template renders.
func sentinelsOf(iface *node.Interface, m *node.Method) []Sentinel {
	names := Sentinels(m.Meta())
	out := make([]Sentinel, 0, len(names))
	for _, name := range names {
		out = append(out, Sentinel{
			Name:   name,
			Helper: Helper(name),
			Ref:    sdk.NewExternal(iface.Package, name),
		})
	}
	return out
}

// partitionOf returns the recorded-call field per-key targeting matches
// against, reporting a field the method has no parameter for.
//
// The generated helper types its key by whichever parameter the field names,
// so a field naming nothing would render a helper with no parameter type at
// all — code that does not compile, blamed on the generator rather than on the
// directive that asked for it. Reported and dropped instead.
func partitionOf(ctx *sdk.GeneratorContext, iface *node.Interface, m *node.Method, host stub.Method) string {
	field := Partition(m.Meta())
	if field == "" {
		return ""
	}
	for _, p := range host.Params {
		if p.Field == field {
			return field
		}
	}
	ctx.Diag.Errorf(m.Pos(),
		"%s: %s=%q on %s.%s names no parameter of the method; the recorded call has no such field",
		Name, PartitionKey, field, iface.Name, m.Name)
	return ""
}
