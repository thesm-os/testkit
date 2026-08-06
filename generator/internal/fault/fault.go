// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package fault owns testkit's `//testkit:fault` directive: which errors a
// method can be made to fail with, and how. It both stamps the directive and
// renders the surface that configuration implies.
//
// # Why its own plugin rather than part of the stub generator
//
// The configuration is not the stub's alone. The double gains a
// `Fault<Name>()` helper, the suite generator writes "returns ErrNotFound for
// a miss" as a subtest, and the model tier needs the partition field to state
// an isolation law. A directive can only be declared once — eidos's registry
// rejects a duplicate name — so declaring it inside any one generator would
// make the others depend on that generator being registered.
//
// Owning it here also means the directive is parsed once and stamped, rather
// than re-read by each generator. Three copies of "strip the `Err` prefix,
// watch for helper-name collisions" would be three chances to disagree, and
// the disagreement would surface as generated code that does not compile.
//
// # How the rendered surface reaches the double
//
// Not by the stub generator rendering it. This plugin declares the same output
// suffixes the stub generator declares, which is what makes Layout resolve
// both contributions to one file — a contributor's Target comes from its own
// suffix table, so identical suffixes are the whole mechanism. The double
// lands in that file's `top` slot and the fault surface in its `bottom` slot,
// which is what puts the helpers after the types they hang off; see
// [SlotName] for why the slot rather than the plugin order decides that.
//
// The consequence worth knowing: the fault helpers are a block at the end of
// the file rather than lines interleaved into each method's configuration.
// Slot ordering is per-plugin, not per-item, so there is no interleaving to
// be had — and the block is attributed, which the interleaved version was not.
//
// # Relationship to shape mixins
//
// eidos's `//testkit:mixin errors` says a method reports misses through a
// sentinel — a law about what the implementation does, which the suite and
// model tiers assert. This directive says which sentinel a test double should
// offer a one-shot helper for, which is test-double ergonomics. Neither
// implies the other, and the two are read independently.
package fault

import (
	"strconv"
	"strings"

	"go.thesmos.sh/eidos/core/meta"
	"go.thesmos.sh/eidos/core/position"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/sdk"

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
	MetaSentinels = meta.EnsureKey("testkit.fault.sentinels", meta.StringListParser)

	// MetaRetry holds the attempt a scheduled fault stops firing on.
	MetaRetry = meta.EnsureKey("testkit.fault.retry", meta.IntParser)

	// MetaPartition holds the recorded-call field per-key targeting matches
	// against.
	MetaPartition = meta.EnsureKey("testkit.fault.partition", meta.StringParser)
)

// Plugin owns the fault directive and renders the surface it configures.
//
// Both roles live on one plugin because a directive may be registered only
// once per run: a second instance declaring the same schema is rejected, so
// the plugin that reads the stamps has to be the plugin that made them.
type Plugin struct{}

// New returns a plugin instance.
func New() *Plugin { return &Plugin{} }

// Name returns [Name].
func (*Plugin) Name() string { return Name }

// Version returns [Version].
func (*Plugin) Version() string { return Version }

// Priority places the plugin one bucket behind the generator it contributes
// into, so the double is queued by the time [Plugin.Generate] looks for it.
//
// The annotator and generator ladders share their numbering, so this one value
// serves both phases: 300 is the cross-cutting generator bucket and the
// annotator refinement bucket, which is where a directive that reads nothing
// but its own source lines belongs anyway — after shape detection, before
// validation.
func (*Plugin) Priority() sdk.Priority { return sdk.GeneratorCrossCutting }

// Provides advertises [Capability].
func (*Plugin) Provides() []string { return []string{Capability} }

// Requires declares the dependency on the generator this one contributes into.
//
// Documentary rather than load-bearing, and worth saying so: an unsatisfied
// requirement is ignored silently, the priority buckets are what order the two
// Generate passes, and the rendered position comes from [SlotName] rather than
// from plugin order. What it earns is a reader finding the dependency where
// they look for it.
func (*Plugin) Requires() []string { return []string{stub.Capability} }

// Directives declares the `//testkit:fault` schema.
//
// Sentinels are variadic positionals so a method reporting several errors
// stays readable on one line, and the directive repeats so it can also be
// split across lines — one concern each.
//
// Negation is denied: a fault helper exists exactly where one is declared, so
// deleting the line is the suppression.
func (*Plugin) Directives() []sdk.DirectiveSchema {
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
			On(node.KindMethod).
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
func annotate(ctx *sdk.AnnotatorContext, iface *node.Interface, m *node.Method) {
	var (
		sentinels []string
		seen      = make(map[string]string)
	)

	for _, dir := range m.Directives() {
		if dir.Name != DirectiveName {
			continue
		}
		for _, name := range dir.Args {
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
func stampRetry(ctx *sdk.AnnotatorContext, iface *node.Interface, m *node.Method, pos position.Pos, raw string) {
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
func Sentinels(bag *meta.Bag) []string {
	if bag == nil {
		return nil
	}
	out, _ := MetaSentinels.Get(bag)
	return out
}

// Retry returns the attempt a scheduled fault stops firing on, or zero when
// the callable configures none.
func Retry(bag *meta.Bag) int {
	if bag == nil {
		return 0
	}
	out, _ := MetaRetry.Get(bag)
	return out
}

// Partition returns the recorded-call field per-key targeting matches
// against, or empty when the callable configures none.
func Partition(bag *meta.Bag) string {
	if bag == nil {
		return ""
	}
	out, _ := MetaPartition.Get(bag)
	return out
}
