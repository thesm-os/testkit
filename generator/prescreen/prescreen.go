// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

// Package prescreen rejects a testkit directive nobody registered.
//
// eidos parses every `//testkit:` line whether or not a schema exists for the
// name, and leaves the unmatched ones inert. That is deliberate — its
// [directive.Validate] docblock says so and says what to do about it:
//
//	Unregistered directive names are not errors — they parse and remain
//	inert. Frontends or plugins that want strict-mode validation can
//	pre-screen with Registry.Lookup and emit their own "unknown directive"
//	diagnostics with Registry.Suggest for "did you mean?" hints.
//
// So the gap is testkit's by design rather than a defect upstream, and this
// package is the pre-screen that closes it. Without one, `//testkit:sutie`
// generates nothing and reports nothing: the interface simply has no suite,
// which is exactly what a correct interface with no directive looks like.
package prescreen

import (
	"go.thesmos.sh/eidos/core/directive"
	"go.thesmos.sh/eidos/node"
	"go.thesmos.sh/eidos/pipeline"
	"go.thesmos.sh/eidos/sdk"
	sdkgolang "go.thesmos.sh/eidos/sdk/golang"
)

// Name is the plugin's identity within the pipeline.
const Name = "prescreen"

// Version composes into the pipeline's plugin fingerprint.
//
// Carried even though this plugin stamps nothing, because what it contributes
// to a run is a verdict on the source and the verdict can change: a name that
// this build rejects is one an older build accepted silently. A fingerprint
// that did not move would let a warm cache serve the older answer.
const Version = "1.0.0"

// Plugin is the directive-name pre-screen. The zero value is unusable; go
// through [New].
type Plugin struct {
	*sdkgolang.Base
	registry *directive.Registry
}

// New returns a pre-screen over the schemas the run registers.
//
// The schemas are passed in rather than read from a plugin registry here, for
// two reasons. This package sits under `generator/` beside the plugins it
// screens, so reaching the universe from inside it would be an import cycle.
// And the registry a pre-screen must agree with is the one the *pipeline*
// built — anything assembled independently answers about a build nobody runs,
// which is the failure a gate can least afford in the plugin whose whole job
// is to say "that name is not registered".
//
// A duplicate schema is dropped rather than reported. The pipeline's own Build
// rejects a duplicate directive name by construction, so a run that reached
// this constructor has none — and a second complaint here would name this
// plugin for something another plugin did.
func New(schemas []sdk.DirectiveSchema) *Plugin {
	r := directive.NewRegistry()
	for _, s := range schemas {
		_ = r.Register(s)
	}
	// The framework's own two. They are registered by the pipeline rather than
	// by any plugin, and `coreDirectives` is unexported — so a pre-screen
	// composed only from plugin schemas rejects every `//testkit:out` in the
	// tree, which is every routed fixture in the corpus.
	for _, n := range []directive.Name{pipeline.OutDirective, pipeline.ValueDirective} {
		_ = r.Register(directive.NewSchema(n).Build())
	}
	return &Plugin{
		Base:     sdkgolang.NewPlugin(Name).Version(Version).Build(),
		registry: r,
	}
}

// Annotate reports every directive whose name no schema claims.
//
// An error rather than a warning. An unregistered name produces exactly the
// output a missing directive produces — nothing — so the author's evidence
// that they asked for something is the line they wrote, and the run's evidence
// is silence. Every other outcome in this repository that looks like coverage
// and is not gets a hard failure, and a misspelt directive is the cheapest
// member of that family to catch.
//
// The whole graph is walked rather than the node kinds testkit's own
// directives attach to. A typo lands wherever the author's finger slipped, and
// screening only the kinds that matter would let `//testkit:sutie` on a struct
// pass while the same typo on an interface failed — which reads as the
// directive being conditionally supported.
func (p *Plugin) Annotate(ctx *sdk.AnnotatorContext) error {
	seen := map[node.Node]bool{}
	report := func(n node.Node) {
		if n == nil || seen[n] {
			return
		}
		seen[n] = true
		p.screen(ctx, n)
	}

	for _, f := range ctx.Reader.Files().Slice() {
		report(f)
	}
	// Self-referencing so the walk descends: [node.Walk] takes the visitor a
	// Visit returns as the one that drives that node's children, and a visitor
	// returning anything else stops one level down.
	var descend node.VisitorFunc
	descend = func(n node.Node) node.Visitor {
		report(n)
		return descend
	}
	for _, pkg := range ctx.Reader.Packages().Slice() {
		// Files hang off the package in the store rather than in the walk, so
		// they are read separately above. Everything else — types, their
		// members, their parameters — arrives through here, which is what
		// keeps a node kind added upstream screened without this plugin
		// learning its name.
		node.Walk(pkg, descend)
	}
	return nil
}

// screen reports one node's unregistered directives.
func (p *Plugin) screen(ctx *sdk.AnnotatorContext, n node.Node) {
	for _, d := range n.Directives() {
		if _, registered := p.registry.Lookup(d.Name); registered {
			continue
		}
		if did, plausible := p.registry.Suggest(d.Name); plausible {
			ctx.Diag.Errorf(d.Pos, "%s: no directive named %q — did you mean %q?", Name, d.Name, did)
			continue
		}
		ctx.Diag.Errorf(d.Pos,
			"%s: no directive named %q, and nothing registered is close enough to be the one you meant",
			Name, d.Name)
	}
}
