# Brand

```go
import "go.thesmos.sh/testkit/core/brand"
```

Four identifiers derive from one token, and all four have to agree: the directive namespace a consumer writes in their source, the header markers rendered into generated files, the project-local state directory, and the config filename.

Any two disagreeing is a silent failure — directives that parse but are never read, a config file nothing discovers, artifacts written where nothing looks for them.

## Constants

| Constant | Value | Used as |
|---|---|---|
| `Name` | `testkit` | The project identity. Everything else derives from it. |
| `DirectivePrefix` | `testkit` | The namespace consumers write directives under: `//testkit:mixin idempotent` |
| `ConfigFile` | `.testkit.yaml` | Discovered by walking up from the working directory |
| `StateDir` | `.testkit` | The manifest and cached pipeline state |

`StateDir` and `ConfigFile` are derived from `Name` rather than written out, so they cannot drift apart under a rename. A rename is one edit.

## Why constants rather than configuration

eidos takes the brand once at program start, never from a flag or a config key. Changing it mid-project would orphan every artifact already on disk — a state directory nothing reads, generated headers no longer recognised as testkit's, directives in a namespace the parser no longer accepts.

## The failure this prevents

`DirectivePrefix` is what testkit passes eidos's parser, and the annotator reads nothing written under any other namespace. A corpus and a CLI that disagree here produce a corpus that **stamps nothing** — every directive parses, nothing errors, and no generator sees a single annotation. That failure mode is what [ADR-0016](../../adr/0016-directives-are-positive-only.md) records.

One config filename, no alternates: a second accepted spelling is permanent ambiguity in every lookup, error message and document ([ADR-0009](../../adr/0009-one-config-filename.md)). The `.yaml` extension is the one the YAML specification recommends.

## Status

Infrastructure rather than a test primitive. Imported by `cmd/testkit/cmds`, `conformance/gate` and the `engine/model` tree. A test author does not call it; a tool built on testkit does.

It has no `doc.go` — the package comment lives on the constants file.

## See also

- [Configuration](../configuration.md) — what `.testkit.yaml` holds.
- [Generators](../generators/README.md) — the directive namespace in use.
