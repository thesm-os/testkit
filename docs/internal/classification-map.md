# Classification map

How testkit's classification vocabulary maps onto the one eidos's `shape`
annotator provides. This is the audit behind
[ADR-0004](../adr/0004-consume-only-the-annotator-plugin.md), which committed
to consuming eidos's annotator without having checked the vocabulary one
classification at a time.

**Result: no gaps.** Every classification testkit had has an upstream home. No
eidos feature request is required to port the generators.

## Two models

testkit classified a method once, into one of 21 shapes, using a priority
registry where the first detector to match wins
([reference/generators/shapes.md](../reference/generators/shapes.md)). A method was a `Paginator` *or* a `Reader`, never
both, and the ordering between them was load-bearing.

eidos classifies on three orthogonal axes, and a method carries a value on each:

| Axis | Question it answers | Count |
|---|---|---|
| Detector | What is the signature? | 20 |
| Contract | What multi-callable protocol does it participate in, and in which role? | 24 |
| Mixin | What does it guarantee? | 28 |

A paginated reader is a `reader` detector *and* a `pagination` contract with
`role=reader`. Nothing is shadowed by priority, because nothing competes.

That difference is the substance of the port. Most of the work is not finding
missing classifications — there are none — it is that seven things testkit
detected as *signatures* are upstream *contracts* or *mixins*, and generators
that branch on a shape name have to branch on contract membership instead.

## Detectors

All 20 eidos detectors are used. Every testkit detector resolves.

| testkit `generator/shape` | eidos axis | eidos name |
|---|---|---|
| `aggregator` | detector | `aggregator` |
| `batch_reader` | detector | `batchreader` |
| `composite_writer` | detector | `compositewriter` |
| `lifecycle` | detector | `lifecycle` |
| `lookup` | detector | `lookup` |
| `multi_aggregator` | detector | `multiaggregator` |
| `multi_arg_writer` | detector | `multiargwriter` |
| `multi_reader` | detector | `multireader` |
| `mutator` | detector | `mutator` |
| `pointer_reader` | detector | `pointerreader` |
| `poison_accessor` | detector | `poisonaccessor` |
| `predicate` | detector | `predicate` |
| `pure` | detector | `pure` |
| `reader` | detector | `reader` |
| `reader_noerror` | detector | `readernoerror` |
| `reader_with_bool` | detector | `readerwithbool` |
| `stream_consumer` | detector | `streamconsumer` |
| `stream_reader` | detector | `streamreader` |
| `void_lifecycle` | detector | `voidlifecycle` |
| `writer` | detector | `writer` |
| `appender` | contract | `appender` |
| `cas` | contract | `cas` |
| `cursor` | contract | `cursor` |
| `persister` | contract | `persister` |
| `pool` | contract | `pool` |
| `publisher` | contract | `publisher` |
| `saga` | contract | `saga` |
| `updater` | contract | `updater` |
| `upserter` | contract | `upserter` |
| `watcher` | contract | `watcher` |

### The seven that change axis

These were detectors in testkit and are contracts — or a detector plus a mixin —
upstream. Each is a real port task, not a lookup.

| testkit detector | Becomes | Why |
|---|---|---|
| `acquire_lease` | `lease` contract, roles `acquire` / `release` | testkit carried `acquire` and `lease` as two spec packages; upstream they are two roles of one contract, so the pairing is validated rather than assumed |
| `two_phase` | `tx` contract, roles `begin` / `commit` / `rollback` | The Begin host declares both partners and the validator reports a missing one |
| `transaction_func` | `transaction` contract, role `fn` | Single-role marker. Distinct from `tx`: this one says "runs inside a transactional scope", not "is the begin of a two-phase triple" |
| `paginator` | `pagination` contract, role `reader`, `cursor=` param | The cursor field name becomes an opaque directive parameter instead of being inferred from the signature |
| `subscriber` | `publisher` contract, role `subscribe` | Subscriber was never an independent shape; it is the partner role of publish |
| `get_or_compute` | `singleflight` contract, role `fn` | Same semantics: concurrent calls with one key share one in-flight computation |
| `deleter` | `writer` detector + `deleteremoves` mixin | testkit made this a detector so the suite could emit delete-removes invariants. Upstream the signature is a writer and the guarantee is a mixin, which is the cleaner split |

## Contracts

11 of eidos's 24 contracts have a same-named testkit precedent, and 3 more are
the renames above. The remaining **nine are new capability**, gained by adopting
the upstream vocabulary rather than by writing anything:

`batchwriter` · `circuitbreaker` · `ifabsent` · `ifmatch` · `leaderelection` ·
`outbox` · `ratelimit` · `workflow` · `writethroughcache`

Each is a protocol a generated suite can assert against once a plugin consumes
it. None is required for parity; all are available.

## Mixins

Exact 1:1. All 28 of eidos's mixins had a same-named testkit spec package, and
testkit had no mixin eidos lacks:

`atomic` · `bounded` · `cacheable` · `concurrent` · `concurrentreaders` ·
`crdtmerge` · `deleteremoves` · `deprecated` · `errors` · `eventually` ·
`hooks` · `idempotent` · `integrationonly` · `lifecycleafterclose` ·
`monotonic` · `nilsafe` · `orderafter` · `partition` · `pure` ·
`readafterwrite` · `retrysucceeds` · `sample` · `scope` · `sideeffect` ·
`streamreflectsmutations` · `timeout` · `validates` · `wrappedvia`

This axis is a pure adoption: nothing to port, nothing to request.

## The three that are not classifications

The design assumed three testkit-only classifications would need a home of their
own. The audit finds they are not classifications at all.

`allocs`, `latency`, and `percentiles` are **benchmark budgets** — a ceiling on
allocations per operation, a mean latency bound, a per-percentile bound. They say
nothing about a method's shape, protocol, or guarantees; they are values the
`bench` generator turns into assertions.

They belong in testkit's directive schema alongside the other generator inputs,
not in a classification package and not upstream. eidos has no reason to carry
benchmark budgets, and asking it to would push testing concerns into a
general-purpose codegen substrate.

## Reproducing this

The upstream vocabulary moves. Counts in prose go stale; the commands do not.

```bash
# eidos, per axis. Check for a newer version first:
#   go list -m -f '{{.Version}}' go.thesmos.sh/eidos/plugins@latest
for a in detectors contracts mixins; do
  printf '%s: ' "$a"
  ls ../eidos/plugins/annotator/shape/$a | grep -v '\.go$' | grep -v internal | tr '\n' ' '
  echo
done

# testkit's classification set, from the last commit that carried it
git ls-tree -d --name-only 9e3622f:generator/spec
git ls-tree --name-only 9e3622f:generator/shape
```
