// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package fault

import (
	"fmt"
	"strconv"
	"strings"

	"go.thesmos.sh/eidos/sdk"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"

	"go.thesmos.sh/testkit/generator/stub"
)

// Name is the plugin's identity within the pipeline.
const Name = "fault"

// Version composes into the pipeline's plugin fingerprint, which frontends
// fold into their cache keys, so a change here invalidates a warm cache
// populated when this annotator stamped differently. An annotator declaring
// no version contributes an empty string and can never invalidate anything —
// and a stale stamp is worse than a stale file, because every generator
// reading it inherits the staleness.
//
// Bump it whenever what gets stamped changes: a new key, a different parse,
// a changed collision rule.
const Version = "1.0.0"

// DirectiveName is the directive this annotator owns, written under testkit's
// namespace as `//testkit:fault`.
const DirectiveName sdk.DirectiveName = "fault"

// Keys the directive accepts alongside its positional sentinels.
const (
	// RetryKey pins the attempt a scheduled fault stops firing on, so
	// `retry=3` fails twice and succeeds on the third call.
	RetryKey = "retry"

	// PartitionKey names the recorded-call field per-key fault targeting
	// matches against.
	PartitionKey = "partition"
)

// SentinelPrefix is stripped from an error variable's name to form its helper
// identifier. `ErrNotFound` yields `FaultNotFound`, which reads as the action
// performed rather than as the variable it happens to use.
const SentinelPrefix = "Err"

// Meta keys this annotator stamps. Generators read through the accessors
// below rather than touching these directly.
var (
	// MetaSentinels holds the error-variable names attached to a callable,
	// in the order they were written.
	MetaSentinels = sdk.EnsureKey("testkit.fault.sentinels", sdk.StringListParser)

	// MetaRetry holds the attempt a scheduled fault stops firing on.
	MetaRetry = sdk.EnsureKey("testkit.fault.retry", sdk.IntParser)

	// MetaPartition holds the recorded-call field per-key targeting matches
	// against.
	MetaPartition = sdk.EnsureKey("testkit.fault.partition", sdk.StringParser)
)

// Plugin owns the fault directive and renders the surface it configures.
//
// Both roles live on one plugin because a directive may be registered only
// once per run: a second instance declaring the same schema is rejected, so
// the plugin that reads the stamps has to be the plugin that made them.
//
// The embedded base answers every declaration method — name, version,
// priority, provides, requires, directives — and the per-language output,
// template and funcmap dispatch. Written out per plugin those drift: testkit
// carried sixteen hand-written copies of the language dispatch, and two of them
// tested the backend's language marker against a local constant rather than the
// backend's own — plugins that emitted nothing, with no diagnostic, because the
// string did not match.
type Plugin struct{ *sdkgolang.Base }

// New returns a plugin instance.
//
// # Failure mode
//
// Build panics on a declaration the pipeline cannot serve — a missing output
// suffix, a duplicate output tag, a template tree that is not there. Every one
// is a mistake in this function rather than in a consumer's source, so it fires
// on the first construction in any test rather than on a run that renders a
// short file and explains why in no output at all.
func New() *Plugin {
	return &Plugin{Base: sdkgolang.NewGenerator(Name, goTemplates, GoOutputs()...).
		Version(Version).
		// One bucket behind the generator this one contributes into, so the
		// double is queued by the time [Plugin.Generate] looks for it.
		//
		// The annotator and generator ladders share their numbering, so this
		// one value serves both phases: 300 is the cross-cutting generator
		// bucket and the annotator refinement bucket, which is where a
		// directive that reads nothing but its own source lines belongs
		// anyway — after shape detection, before validation.
		Priority(sdk.GeneratorCrossCutting).
		Provides(Capability).
		// The dependency on the generator this one contributes into.
		//
		// Documentary rather than load-bearing, and worth saying so: an
		// unsatisfied requirement is ignored silently, the priority buckets are
		// what order the two Generate passes, and the rendered position comes
		// from [SlotName] rather than from plugin order. What it earns is a
		// reader finding the dependency where they look for it.
		Requires(stub.Capability).
		Directives(directives()...).
		// No Funcs call: the templates reach testkit's runtime through the
		// backend's own `external` builtin rather than a helper this plugin
		// registers, and fault_go_test.go asserts that entry stays absent.
		Build()}
}

// directives declares the `//testkit:fault` schema.
//
// Sentinels are variadic positionals so a method reporting several errors
// stays readable on one line, and the directive repeats so it can also be
// split across lines — one concern each.
//
// Negation is denied: a fault helper exists exactly where one is declared, so
// deleting the line is the suppression.
func directives() []sdk.DirectiveSchema {
	return []sdk.DirectiveSchema{
		sdk.NewDirective(DirectiveName).
			Describe(
				"Configures how the annotated method can be made to fail on a "+
					"generated test double. Positional arguments name error "+
					"variables in the source package, each gaining a Fault<Name> "+
					"helper; retry pins the attempt a scheduled fault stops firing "+
					"on, and partition names the recorded-call field per-key "+
					"targeting matches against. Repeatable — sentinels union across "+
					"lines, keys take the last value written.",
			).
			Positional("sentinel").
			AllowExtraPositional().
			AllowedKeys(RetryKey, PartitionKey).
			On(sdk.NodeKindMethod).
			DenyNegation().
			Build(),
	}
}

// Annotate folds every fault directive on every method into stamped metadata.
//
// Malformed input is reported and dropped rather than guessed at. A retry
// count that does not parse would otherwise silently become zero, which reads
// as "no retry configured" — the one answer indistinguishable from the
// directive being absent.
func (*Plugin) Annotate(ctx *sdk.AnnotatorContext) error {
	for _, iface := range ctx.Reader.Interfaces().Slice() {
		for _, m := range iface.Methods {
			annotate(ctx, iface, m)
		}
	}
	return nil
}

// annotate stamps one method's fault configuration.
func annotate(ctx *sdk.AnnotatorContext, iface *sdk.Interface, m *sdk.Method) {
	var (
		sentinels []string
		seen      = make(map[string]string)
	)

	for _, dir := range m.Directives() {
		if dir.Name != DirectiveName {
			continue
		}
		for _, name := range dir.Args {
			// An unexported sentinel is unreachable from the generated file.
			// The double is routed out of the source package by `out=`, so the
			// reference is qualified — `storepkg.errNotFound` — and that is a
			// compile error in a consumer's repository, blamed on the
			// generator. Refused here rather than emitted, because the
			// same-package spelling that would work is the arrangement this
			// plugin's own fixture calls unrepresentative.
			//
			// It also defeats the guard below: [Helper] strips an `Err` prefix
			// the lowercase spelling does not have, so `errNotFound` and
			// `ErrNotFound` generate different helpers and do not read as a
			// collision.
			if !sdk.IsExportedName(name) {
				ctx.Diag.Errorf(dir.Pos,
					"%s: sentinel %q on %s.%s is not exported, so a double routed out "+
						"of %s cannot name it; export it or drop it",
					Name, name, iface.Name, m.Name, iface.Package)
				continue
			}
			helper := Helper(name)
			if prior, clash := seen[helper]; clash {
				ctx.Diag.Errorf(dir.Pos,
					"%s: sentinels %q and %q on %s.%s both generate %s; rename one or drop it",
					Name, prior, name, iface.Name, m.Name, helper)
				continue
			}
			seen[helper] = name
			sentinels = append(sentinels, name)
		}
		if raw, ok := dir.KV[RetryKey]; ok {
			stampRetry(ctx, iface, m, dir.Pos, raw)
		}
		if field, ok := dir.KV[PartitionKey]; ok {
			MetaPartition.Set(m.EnsureMeta(), field, Name)
		}
	}

	if len(sentinels) > 0 {
		MetaSentinels.Set(m.EnsureMeta(), sentinels, Name)
	}
}

// stampRetry records a retry count, reporting anything that is not a positive
// attempt number.
func stampRetry(ctx *sdk.AnnotatorContext, iface *sdk.Interface, m *sdk.Method, pos sdk.Pos, raw string) {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		ctx.Diag.Errorf(pos,
			"%s: %s=%q on %s.%s is not a positive attempt count",
			Name, RetryKey, raw, iface.Name, m.Name)
		return
	}
	MetaRetry.Set(m.EnsureMeta(), n, Name)
}

// Helper returns the generated helper identifier for a sentinel variable.
//
// Exported because the collision rule and the emitted method name have to
// agree: the annotator reports a clash using this, and a generator names the
// method with it.
func Helper(sentinel string) string {
	return "Fault" + strings.TrimPrefix(sentinel, SentinelPrefix)
}

// Sentinels returns the error variables attached to a callable, in the order
// written. Empty when the callable declares none.
func Sentinels(bag *sdk.Bag) []string {
	if bag == nil {
		return nil
	}
	out, _ := MetaSentinels.Get(bag)
	return out
}

// Retry returns the attempt a scheduled fault stops firing on, or zero when
// the callable configures none.
func Retry(bag *sdk.Bag) int {
	if bag == nil {
		return 0
	}
	out, _ := MetaRetry.Get(bag)
	return out
}

// Partition returns the recorded-call field per-key targeting matches
// against, or empty when the callable configures none.
func Partition(bag *sdk.Bag) string {
	if bag == nil {
		return ""
	}
	out, _ := MetaPartition.Get(bag)
	return out
}

// Capability is the label the plugin advertises, so a later generator can
// declare a dependency on the fault surface existing.
const Capability = "fault"

// SlotName is the [sdk.EmitFile] slot the contributions land in.
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
	Ref *sdk.Expr
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
	RuntimePaths

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
// [sdk.EmitOutputPackageSetter].
type Tests struct {
	sdk.BaseEmit
	RuntimePaths

	// TypeName is the double's identifier, which names the generated checks.
	TypeName string

	// StubRef qualifies the generated double. Set during Generate against the
	// source package as a provisional value, then corrected once routing
	// resolves.
	StubRef *sdk.Expr

	// CtorRef qualifies the double's constructor, which lives beside it and
	// therefore follows the same routing.
	CtorRef *sdk.Expr

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
	path, ok := sdk.PrimaryPackage(byTag)
	if !ok {
		return
	}
	t.StubRef = sdk.NewExternal(path, t.TypeName)
	t.CtorRef = sdk.NewExternal(path, "New"+t.TypeName)
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

	c := sdk.NewProvenance(Name)
	for _, iface := range ctx.Reader.Interfaces().Slice() {
		double, hosted := doubles[sdk.Node(iface)]
		if !hosted {
			continue
		}
		methods := methodsOf(ctx, iface, double)
		if len(methods) == 0 {
			continue
		}

		base := sdk.EmitBase(c, iface)

		// Queued in one call rather than two. The pair differs only in its emit
		// kind and output tag, and a second append is where the two would drift.
		// The error the queue returns already names this plugin and the kind
		// that failed: swallowing it would read downstream as a method nobody
		// configured rather than as a fault, and the helpers are this plugin's
		// whole output.
		if err := sdk.QueueEmit(ctx.Store.Emit(), c, SlotName, iface,
			&Helpers{
				BaseEmit:     base,
				RuntimePaths: GoRuntime(),
				IfaceName:    iface.Name,
				Methods:      methods,
			},
			&Tests{
				BaseEmit:     sdk.EmitBaseTagged(base, GoTestOutputTag),
				RuntimePaths: GoRuntime(),
				TypeName:     double.TypeName,
				StubRef:      sdk.NewExternal(iface.Package, double.TypeName),
				CtorRef:      sdk.NewExternal(iface.Package, "New"+double.TypeName),
				Methods:      methods,
			},
		); err != nil {
			// Wrapped even though the queue names the plugin and the slot: what
			// it cannot name is which declaration the run was on when it failed,
			// and that is the only part a reader needs to find the source line.
			return fmt.Errorf("%s: queue interface %q: %w", Name, iface.Name, err)
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
func doublesByOrigin(ctx *sdk.GeneratorContext) map[sdk.Node]*stub.Stub {
	return sdk.PendingByOrigin[*stub.Stub](ctx.Store.Emit())
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
func methodsOf(ctx *sdk.GeneratorContext, iface *sdk.Interface, double *stub.Stub) []Method {
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
func sentinelsOf(iface *sdk.Interface, m *sdk.Method) []Sentinel {
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
func partitionOf(ctx *sdk.GeneratorContext, iface *sdk.Interface, m *sdk.Method, host stub.Method) string {
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
