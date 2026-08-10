# Shape classification

A generator that reads only a signature can write a double. A generator that knows what the signature *means* can write the checks — that a read after a write returns what was written, that a second identical call changes nothing, that a cancelled context is honoured. Classification is what turns the first into the second.

testkit does not implement classification. It configures eidos's shape annotator and reads the stamps ([ADR-0004](../../adr/0004-consume-only-the-annotator-plugin.md)). This page is the vocabulary that annotator provides, as of `go.thesmos.sh/eidos/plugins v1.7.3`.

## Three orthogonal axes

An earlier design used one priority-ordered registry with first-match-wins, so a permissive shape could shadow a specific one and the loser was invisible. That is gone. Classification now runs on three independent axes, and a callable can carry stamps from all three at once.

| Axis | Scope | Stamped by | Directive |
|---|---|---|---|
| **Detector** | One callable | Signature analysis | `//testkit:shape <name>` overrides |
| **Contract** | Several callables bound into one protocol | The author | `//testkit:contract <name> role=<role> …` |
| **Mixin** | One callable, layered on its detected shape | The author | `//testkit:mixin <name> [k=v …]` |

The mental model is layered. The detector picks the base laws — a writer owes write-then-observe. Each mixin adds one more invariant on top. A contract binds a method to its partners, so `Commit` is known to be the commit half of the `Begin` above it.

### What a detector stamps

Three meta keys, and they are the same across every source language:

| Key | Carries |
|---|---|
| `shape` | The canonical name, e.g. `reader` |
| `shape.key_type` | Qualified type of the key or input parameter, where the shape has one |
| `shape.value_type` | Qualified type of the value or output, where the shape has one |

A Go `func(ctx, K) (V, error)` and a hypothetical Rust `fn(&self, K) -> Result<V, E>` both surface as `shape = "reader"`. A generator branches on the stamp without knowing which frontend produced it.

Detectors still carry a `Priority`, but it now orders dispatch within the detector axis only — it cannot shadow a contract or a mixin. Higher runs first; equal priorities tie-break on registration order.

## Detectors

Twenty, and every one has a fixture in `conformance/corpus/iface/detector/`. A leading `context.Context` is optional unless the table says otherwise — `ctx?` means both `func(ctx, K)` and `func(K)` detect.

| Shape | Priority | Signature | Notes |
|---|---|---|---|
| `streamreader` | 1000 | `func(ctx?, K?) iter.Seq[V]` | Also `iter.Seq2`. The variant is stamped so a consumer can tell the two apart. |
| `batchreader` | 950 | `func(ctx?, ...K) ([]V, error)` | The only non-context parameter is the variadic tail. |
| `lookup` | 850 | `func(ctx?, K) (V, M, bool)` | No error return. The metadata type is stamped alongside the triple. |
| `readerwithbool` | 840 | `func(ctx?, K) (V, bool)` | Map-style presence. No error return. |
| `poisonaccessor` | 830 | `func() error` | Takes nothing. Forbids a context, which is what separates it from `lifecycle`. |
| `predicate` | 820 | `func() bool` | Takes nothing. |
| `voidlifecycle` | 810 | `func()` | No parameters, no returns. |
| `pure` | 800 | `func(…) T` | No context, no error, exactly one return. Parameters are unconstrained — the shape is about return discipline, not input shape. |
| `multiargwriter` | 750 | `func(ctx?, P1, P2, P3, …) error` | Three or more non-context parameters. The full argument-type list is stamped. |
| `compositewriter` | 700 | `func(ctx?, K, V) error` | Exactly two non-context parameters. |
| `multireader` | 650 | `func(ctx?, K) (V1, V2, …, error)` | Two or more non-error returns. The full list is stamped. |
| `multiaggregator` | 600 | `func(ctx?) (V1, V2, …, error)` | No non-context parameters, two or more non-error returns. |
| `writer` | 500 | `func(ctx?, V) error` | `error` is the *only* return, not "at most one" — see below. |
| `streamconsumer` | 470 | `func(ctx, iface) (V, error)` | The context is **required**. |
| `pointerreader` | 450 | `func(ctx?, K) *V` | No error return. |
| `reader` | 420 | `func(ctx?, K) (V, error)` | The parameter must be usable as a key. |
| `readernoerror` | 400 | `func(ctx?, K) V` | Infallible fetch. |
| `aggregator` | 350 | `func(ctx?) (T, error?)` | No non-context parameters. |
| `mutator` | 300 | `func(ctx?, V)` | Zero returns. A `*V` parameter is unwrapped so the stamped value type names the element. |
| `lifecycle` | 200 | `func(ctx) error` | The context is **required**, which is what separates it from `poisonaccessor`. |

Three of these encode a decision worth knowing about.

**`writer` requires `error` to be the only return.** An earlier form accepted "at most one non-error return", which made every signature `reader` recognises a strict subset of `writer` — and `writer` runs first. `reader` never won a dispatch, and the harm was not the dead rule but the wrong stamp that replaced it.

**`reader` refuses a parameter that cannot serve as a key.** The stamp's whole content is that the parameter *is* a key: same-key-same-value, read-after-write and deterministic re-read all derive from it. A parameter with no equality makes those checks vacuous rather than false, which is worse.

**`streamconsumer` requires the context deliberately.** `func(io.Reader) (V, error)` is too ambiguous to claim — constructors and helpers share it. The cost is a missed stamp on an unusual form, against mislabelling a common one.

## Contracts

A contract binds several callables into one protocol. The author declares it, because no signature analysis can tell that `Commit` belongs to the `Begin` three lines above. Twenty-four, each with a fixture in `conformance/corpus/iface/contract/`.

Every member carries `role=`, and the roles a contract accepts are a closed set — a directive naming an undeclared role is rejected. Partner roles reference sibling methods by name and are resolved to qualified names during annotation.

| Contract | Directive as written |
|---|---|
| `appender` | `//testkit:contract appender role=fn` |
| `batch-writer` | `//testkit:contract batch-writer role=writer mode=atomic` |
| `cache` | `//testkit:contract cache role=cache backing=Fetch` |
| `cas` | `//testkit:contract cas role=writer version=Version` |
| `circuit-breaker` | `//testkit:contract circuit-breaker role=fn` |
| `cursor` | `//testkit:contract cursor role=next close=Close` |
| `if-absent` | `//testkit:contract if-absent role=writer` |
| `if-match` | `//testkit:contract if-match role=writer pred=Match` |
| `leader-election` | `//testkit:contract leader-election role=campaign resign=Resign isleader=IsLeader` |
| `lease` | `//testkit:contract lease role=acquire release=Release` |
| `outbox` | `//testkit:contract outbox role=append subscribe=Subscribe` |
| `pagination` | `//testkit:contract pagination role=reader cursor=Cursor` |
| `persister` | `//testkit:contract persister role=writer reader=Get` |
| `pool` | `//testkit:contract pool role=get put=Put` |
| `publisher` | `//testkit:contract publisher role=publish subscribe=Subscribe` |
| `rate-limit` | `//testkit:contract rate-limit role=fn rate=100 burst=10` |
| `saga` | `//testkit:contract saga role=step compensate=Compensate` |
| `singleflight` | `//testkit:contract singleflight role=fn` |
| `transaction` | `//testkit:contract transaction role=fn` |
| `tx` | `//testkit:contract tx role=begin commit=Commit rollback=Rollback` |
| `updater` | `//testkit:contract updater role=writer reader=Get` |
| `upserter` | `//testkit:contract upserter role=writer reader=Get` |
| `watcher` | `//testkit:contract watcher role=watch trigger=Trigger` |
| `workflow` | `//testkit:contract workflow role=fn transitions=Draft>Live` |

A key whose value names a partner method is resolved; a key whose value is a literal is not. `cas version=Version` names a field, `rate-limit rate=100` is a number, and `workflow transitions=Draft>Live` is a state expression — none of those is looked up as a sibling.

## Mixins

A mixin asserts one orthogonal invariant about a single callable. Unlike a contract it binds nothing to anything. Twenty-eight, each with a fixture in `conformance/corpus/iface/mixin/`.

| Mixin | Directive as written |
|---|---|
| `atomic` | `//testkit:mixin atomic` |
| `bounded` | `//testkit:mixin bounded limit=100` |
| `cacheable` | `//testkit:mixin cacheable` |
| `concurrent` | `//testkit:mixin concurrent` |
| `concurrentreaders` | `//testkit:mixin concurrentreaders` |
| `crdtmerge` | `//testkit:mixin crdtmerge` |
| `deleteremoves` | `//testkit:mixin deleteremoves` |
| `deprecated` | `//testkit:mixin deprecated` |
| `errors` | `//testkit:mixin errors` |
| `eventually` | `//testkit:mixin eventually` |
| `hooks` | `//testkit:mixin hooks` |
| `idempotent` | `//testkit:mixin idempotent` |
| `integrationonly` | `//testkit:mixin integrationonly` |
| `lifecycleafterclose` | `//testkit:mixin lifecycleafterclose` |
| `monotonic` | `//testkit:mixin monotonic` |
| `nilsafe` | `//testkit:mixin nilsafe` |
| `orderafter` | `//testkit:mixin orderafter fn=Prepare` |
| `partition` | `//testkit:mixin partition` |
| `pure` | `//testkit:mixin pure` |
| `readafterwrite` | `//testkit:mixin readafterwrite write=Put` |
| `retrysucceeds` | `//testkit:mixin retrysucceeds` |
| `sample` | `//testkit:mixin sample` |
| `scope` | `//testkit:mixin scope name=tenant` |
| `sideeffect` | `//testkit:mixin sideeffect` |
| `streamreflectsmutations` | `//testkit:mixin streamreflectsmutations` |
| `timeout` | `//testkit:mixin timeout duration=5s` |
| `validates` | `//testkit:mixin validates fn=Validate` |
| `wrappedvia` | `//testkit:mixin wrappedvia fn=Cause` |

Several mixins fit on one line, and the axis is genuinely orthogonal:

```go
//testkit:mixin idempotent concurrent sideeffect
Put(ctx context.Context, r Record) error
```

`orderafter fn=`, `readafterwrite write=`, `validates fn=` and `wrappedvia fn=` name sibling methods, which the resolver rewrites to qualified names and a validator checks resolve to something real. `bounded limit=`, `scope name=` and `timeout duration=` are opaque literals.

## Overriding detection

```go
//testkit:shape reader
Fetch(ctx context.Context, id string) (Record, error)
```

A `//testkit:shape` directive wins over every detector. The plugin checks the directive list first per callable; on a hit it stamps the named shape and skips the cascade entirely.

The corpus contains no shape overrides. That is deliberate — a fixture whose classification came from a directive proves the directive was read, not that the detector works, so every detector fixture is written to be recognised from its signature alone.

## What consumes the stamps

Only `stub` today, and it reads a narrow slice: the mixin axis for `deprecated` and `orderafter`, and the detector axis for the iterator variants. The `suite`, `bench` and `model` generators are designed against the full vocabulary and are not implemented — see their pages.

testkit registers the *entire* eidos registry rather than a curated subset ([ADR-0004](../../adr/0004-consume-only-the-annotator-plugin.md)), so a classification added upstream is available the moment the dependency moves, and the conformance gate starts asking for a fixture in the same build.

## See also

- [Classification map](../../internal/classification-map.md) — the mapping from these names to the laws each implies.
- [Stub](stub.md) — the one shipped generator that reads these stamps.
- [ADR-0004](../../adr/0004-consume-only-the-annotator-plugin.md) — why testkit consumes eidos's annotator rather than implementing its own.
