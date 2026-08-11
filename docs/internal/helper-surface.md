# Helper surface

Which testkit helpers the generated output actually reaches, which it hand-rolls
instead, and which nothing reaches at all. This is the audit behind the claim
[ADR-0017](../adr/0017-every-classification-owes-an-assertion.md) rests on — that
a generated check is worth having — read from the other end: not what the
generators assert, but what they assert it *with*.

**Result: 15 of the runtime's package-level helpers reach generated code.** Five
places hand-roll a helper that already ships. The rest of the surface divides
into helpers a planned tier will bind and helpers nothing will.

Working note. Delete when the five items in [What it hand-rolls
instead](#what-it-hand-rolls-instead) have landed.

## What generated output reaches

Call sites across the 86 harnesses, 86 doubles, 6 builders, 6 enums and 5
sentinel packages in `conformance/corpus`. Counted, not listed — the command is
in [Reproducing this](#reproducing-this).

| Helper | Call sites | Emitted by |
|---|---|---|
| `Equal` | 1011 | suite, stub, builder, enum, sentinel |
| `Rejects` | 589 | suite falsification |
| `Assert` | 589 | suite falsification |
| `ErrorIs` | 408 | suite, stub, fault, builder, enum, sentinel |
| `True` | 268 | suite, stub, builder, enum, sentinel |
| `Len` | 159 | suite, stub, enum |
| `TestError` | 140 | stub, fault |
| `False` | 39 | enum, sentinel |
| `NoError` | 34 | suite, fault, enum |
| `NotEqual` | 17 | suite, enum, sentinel |
| `HasPrefix` | 10 | sentinel |
| `Error` | 4 | suite |
| `Contains` | 4 | sentinel |
| `RequireEnv` | 1 | suite |
| `NewFailableTB` | 1 | stub |

Sub-packages, same corpus:

| Package | Reached | Not reached |
|---|---|---|
| `stub/` | `MethodStub` `Recorder` `Answer`/`Arms` `Subject`/`Behaviour` `Double`/`DoubleBehaviour` `Instance` `Configurable` `OrderTracker` | `Gate` `WaitForN` `WaitFor` `Filter`/`First`/`Any`/`All` `BenchMode` |
| `clock/` | `Clock` `RealClock` `TestClock` | — |
| `fault/` | `And` `NewCountedFault` `NewPredicateFault` `NewRetryFault` | `Or` `NewProbabilityFault` `NewWindowedFault` |
| `rand/` | `Source` | `FixedRandSource` `DefaultRandSource` |
| `concurrency/` `polling/` `golden/` | — | everything |

`Assert` reaching 589 sites overstates the fluent family: every one is
`Assert(t, got).Contains(…)` in a falsification guard. The other seventeen
matchers have no generated caller.

## What it hand-rolls instead

Five places where a shipped helper is re-implemented. One row per item; each
carries the change and the argument against making it.

- **`enum` asserts the weakest form at the decoder boundary.**
  `enum.test.tmpl:205` writes `True(t, err != nil)` where `ErrorIs` against the
  sentinel is available.
- **`sentinel` hand-rolls `ErrorIsNot` three times.**
  `sentinel.tests.tmpl:96,111,139`.
- **`suite` re-declares `core/factory.Named` per interface.**
  `suite.options.tmpl:90`, 86 times in the corpus.
- **`<Iface>WithClock` is inert in 84 of 86 harnesses.**
  `suite.options.tmpl:101` declares the field unconditionally; only `timeout`
  and `outbox` checks read it.
- **`suite` emits no falsification file for a generic interface.**
  86 harnesses, 84 companions. `stub` solved the same problem with `witness=`.

## What no consumer reaches

Every package-level helper with no caller outside its own declaration and its
own test. Each is classified **keep**, **delete**, or **blocked on tier N** —
never a bare "unused", because the tier that would bind it may not be built yet.

Covers: the behavioural family (`AssertPure` `AssertBounded` `AssertNilSafe`
`AssertNilCtx` `AssertCtxCancellation` `AssertCtxDeadline` `AssertTimeout`), the
benchmark contract (`StartContract` `AllocsMax` `LatencyMax` `End`), the fluent
matchers, and the utility set (`Sequence` `ErrorAs` `ErrorIsNot` `NotContains`
`ContainsInOrder` `SeededRand` `TableTest` `DiffMap` `SortedKeys` `FreePort`
`MustMarshal` `MustUnmarshal` `MustDecodeHex` `Quiet` `FailingReader`
`FailingWriter`), plus `concurrency/`, `polling/` and `golden/` whole.

## What is documented as absent

Subtracted, not restated. Three findings that look like gaps are the recorded
plan of record, and this note cites them rather than re-deriving them:

- The `model` tier — [RFC-0003](../rfc/0003-the-projection-consumers.md),
  [generators README](../reference/generators/README.md).
- The `bench` tier — same two.
- `core/{trace,failure,coverage,visualize,equivalence}` — runtime for the
  tier-5 generators the same README lists as planned.

## The open decision

`concurrent` and `concurrentreaders` are the suite tier's under
[ADR-0018](../adr/0018-one-tier-owns-each-classification.md) and generate
nothing. `concurrency.ConcurrentStress` and `stub.Gate` are the primitives a
check would use. What blocks it is which half of the claim the gate can see.

## Reproducing this

Counts in prose go stale; the commands do not.

```bash
# Helpers reached by generated output, by call site.
find conformance/corpus \( -name '*.gen.go' -o -name '*.gen_test.go' \
  -o -name '*.enum_test.go' \) -print0 \
  | xargs -0 grep -hoE 'testkit\.[A-Z][A-Za-z]*\(' \
  | sed 's/testkit\.//; s/(//' | sort | uniq -c | sort -rn

# Sub-package symbols reached by generated output.
find conformance/corpus \( -name '*.gen.go' -o -name '*.gen_test.go' \
  -o -name '*.enum_test.go' \) -print0 \
  | xargs -0 grep -hoE '\b(clock|stub|fault|rand|polling|golden|concurrency)\.[A-Z][A-Za-z]*' \
  | sort | uniq -c | sort -rn

# One helper's consumers, minus its own declaration and its own test.
grep -rl 'testkit\.AssertPure\b' --include='*.go' --include='*.tmpl' . \
  | grep -vE '^\./[a-z_]*(_test)?\.go$'

# Harnesses against falsification companions — the generic gap.
find conformance/corpus -name 'iface_suite.gen.go' | wc -l
find conformance/corpus -name 'iface_suite.gen_test.go' | wc -l
```
