# Generated-suite audit

Whether the generated conformance suites assert what the classifications claim,
measured against the repo's own standards —
[ADR-0017](../adr/0017-every-classification-owes-an-assertion.md)'s "every
classification owes an assertion",
[ADR-0018](../adr/0018-one-tier-owns-each-classification.md)'s "one tier owns
each", and [RFC-0002](../rfc/0002-the-suite-generator.md)'s "can each check
fail?". Read from the corpus outward: not what the generators were designed to
emit, but what the working tree actually asserts, verified by breaking fixtures
and watching what reddens.

**Result: the union gate measures fixture existence per classification, never
check emission — and the model tier does not deliver the classifications
ADR-0018 assigned to it.** 78 of 103 model harnesses compare the subject
against a second instance of itself; 5 of ~54 mixins and 0 of 26 contracts have
their own law bound; deleting the `bounded` fixture's entire claim keeps the
corpus green. ADR-0018's Negative section predicted this exact failure ("a
classification parked in the model tier … would not be caught"); it has
happened at scale. The suite tier, by contrast, is genuinely well-gated — 826
generated falsification tests, each asserting the reason for rejection — with a
short list of specific defects.

Working note. Delete when the [improvement programme](#improvement-programme)
has landed — each finding names its fix.

Audited 2026-08-12, on the working tree at `model 0.8.0` / `suite 1.0.0`
(uncommitted template changes to `model.bindings.tmpl` and
`model.companion.tmpl` included).

## Method

Every load-bearing claim was verified by execution, not reading:

- **Break experiments.** Two fixtures were deliberately broken and the corpus
  run: `bounded` (twin reference) and `idempotent` (derived reference). Both
  edits reverted; commands in [Reproducing this](#reproducing-this).
- **Mechanical census.** Reference kind, driven sequences, bound laws, unbound
  laws, Porcupine leg and kill-matrix presence were extracted from the 103
  generated `iface_model.gen.go` files by script, not by hand — the numbers
  below re-derive from the greps in [Reproducing this](#reproducing-this).
- **Source audit.** `engine/model` (laws, runner, actions, linearize, bmc,
  mutation, coverage, shrinker, ref), `generator/model`, `generator/tiers`,
  `generator/suite`, and `conformance/gate` read end to end, with file:line
  evidence carried per claim.

## The two execution proofs

**`bounded` — twin reference, all green against a broken subject.** The mixin's
whole claim is `Limit = 100` (`boundedtest/inmemory.go`). Removing the
`min(len(s.items), Limit)` clamp — the only behaviour the mixin declares —
leaves every test in the corpus green. The suite header defers the claim to the
model tier; the model file answers `Not bound: AUTO-AGGREGATOR-BOUNDED — the
catalogue carries no instantiation spec for it`; the reference is the subject's
own factory, so both twins violate identically.

**`idempotent` — derived reference, red in milliseconds.** Making `Put`
accumulate (`s.items[key] += value`) fails `TestMixedContract` at the first
divergent read, with a minimal shrunk counterexample (`Put(k, "test-value")`,
`Put(k, "")`, `Read(k)`) and a classified failure artifact — despite
`AUTO-IDEMPOTENT-WRITE` being unbound there too.

The pair defines the real boundary: **a derived oracle gives strong implicit
coverage even without the named law; a twin gives a determinism smoke test
wearing a property-test shape.** Every finding below hangs off it.

## F1 — the tier-union hole (critical)

Census of the 103 model files in the working tree:

| Measure | Count |
|---|---|
| Reference is a **twin** (subject vs. second instance of itself) | **78 / 103** |
| Twins driving no state-mutating action (nothing can diverge) | ~40 |
| Files binding any law | 17 |
| Law types ever emitted | 6 (`WriteObservable` ×12, `ReadAfterWrite` ×2, `PointInTime`, `DeleteReturnsNotFound`, `DefaultOnError`, `StreamNoDuplicates` — 18 instances) |
| Mixins whose **own** law is bound in their fixture | **5** (`defaultonerror`, `deleteremoves`, `noduplicates`, `pointintime`, `readafterwrite`) |
| Contracts whose own law is bound | **0 / 26** |
| Companion (kill matrix + reference test + fuzz target) emitted | 25 / 103 — derived references only (`generator/model/model.go:506-508`) |
| Porcupine concurrent leg emitted | 12 / 103 |

`cas`, `lease`, `saga`, `singleflight`, `transaction`, `workflow`,
`pagination`, `atomic`, `bounded`, `cacheable`, `crdtmerge`, `eventually`,
`idempotent` (as a law), `lifecycleafterclose`, `monotonic`,
`snapshotisolation`, `tamperevident`, `windowed`, and ~25 more certify their
defining property **nowhere**. Both tiers are honest about it in comments —
the suite header says "checked somewhere else", the model header says "Not
bound" — and no gate compares the two lists.
`conformance/gate/annotate_test.go:23-35` measures that every classification is
*stamped* somewhere in the corpus; nothing measures that it is *asserted*.

Why laws do not bind — three independent chokepoints:

1. **The instantiation table has 13 rows for an 83-law catalogue**
   (`generator/tiers/bindings.go:73-91`). 60 distinct laws are selected by real
   classifications and dropped at `generator/model/laws.go:132-138` — 141 `Not
   bound … no instantiation spec` header lines corpus-wide.
2. **Only 9 law-field templates exist** (`generator/model/templates/golang/law/`:
   Collect, Drain, Hash, KeyOf, Keys, Read, Sentinel, Values, Write). Manifests
   naming `Advance`, `Factory`, `History`, `Merge`, `Less`, `Probe`, … have no
   renderer — a second, independent wall behind the first.
3. **`KindHandle` is hardcoded to the key projection**
   (`generator/model/laws.go:270-275`), ignoring `f.From`, which blocks the
   stream laws on any interface without a keyed store even where the template
   needs no projection at all (`handleIdentity`).

Why references stay twins: `referenceOf` (`generator/model/model.go:1010-1143`)
derives only three store models. `engine/model/ref` ships **22 oracles** —
`Lease`, `Pool`, `Cursor`, `Saga`, `Singleflight`, `Txn`, `AtomicCell`,
`AppendOnly`, `SnapshotIsolation`, … — of which the generator can reach **5**
(`MapStore`, `KeyedStore`, `Collection`, `SetCollection`, `StickyStore`). The
oracles that would lift the contract axis off the twin floor exist, are
unit-tested, and are unreachable. Likewise 22 of 43 action constructors
(`CompareAndSwap`, `AcquireLease`, `TwoPhase`, `Saga`, `Cursor`, `Paginator`,
`Watcher`, …) have no dispatch row — `generator/tiers/actions.go:27-48` maps
detector shapes only, contract roles not at all.

## F2 — the model tier's own proofs are the weakest in the repo

- **Kill matrix** (`Test<Iface>ModelKillsInertMutants`): real and load-bearing,
  but (a) it exists only for the 25 derived-reference fixtures — precisely the
  interfaces least in need of it; (b) it asserts `testkit.True(t, f.Failed(),
  …)` — failure *existence*. 0 of 25 assert identity, against 826/826 in the
  suite tier. A mutant reddening via an unrelated panic is indistinguishable
  from a kill. `testkit.FailableTB.Msg()` exists and is unused here.
- **`Test<Iface>ModelReference`** (25 files) runs the derived reference against
  a second instance of itself. A systematically wrong adapter agrees with
  itself; only nondeterminism or shared state can redden it.
- **`Test<Iface>ModelReferenceInertBodies`** calls generated `{ return }`
  bodies and discards both results (`model.companion.tmpl:93-102`). **This
  test cannot fail.** By the repo's own standard it should be deleted or
  replaced with the omission comment.
- **`Fuzz<Iface>Model`**: the wiring is right — the fuzzer's bytes genuinely
  replay as rapid's choice stream — but the shipped target fuzzes the derived
  reference against itself (nothing to find), the seed corpus is one empty
  byte slice, no `testdata/fuzz` corpus is checked in, and fuzz is not in CI.
  The new consumer-facing `<Iface>ModelFuzz` (working tree) is emitted in 103
  files and called in 25.
- **Error identity is never compared.** Every action checks `(sutErr == nil)
  != (refErr == nil)` (`engine/model/action/action.go`, `signature.go`
  throughout); the Porcupine model is built with `sentinel: nil`, so any
  non-nil error reads as a miss (`model.bindings.tmpl:143-146`,
  `linearize/kv.go:69-72`). A subject returning the wrong sentinel agrees with
  the reference everywhere.
- **`Mutator` and `VoidLifecycle` actions assert nothing**
  (`action.go:157-163`, `signature.go:303-308`) — correct only if a law covers
  the shape, and none is ever bound.
- **Porcupine `Unknown` (timeout) is a green test with a log line**
  (`engine/model/concurrent.go:173-175`). No counter, no artifact, no policy.
- **CI model runs are unseeded.** rapid's seed defaults to random per run;
  nothing sets `RAPID_SEED` (`ci.yml` says outright that env overrides would be
  inert); no rapid failfiles are checked in. A CI-only model failure does not
  reproduce from the log.
- **64 `law vacuously holds` returns** across `law/` + `timeaware/`: a subject
  refusing inputs silently neutralises ~30 laws. RFC-0003 commissions a
  not-applicable counter; unbuilt.

## F3 — suite-tier defects

Credit first, because it is earned: the fixture pair (`Key`/`KeyOther`
engineered so a miss cannot accidentally hit), fatal-not-pass on failed
preconditions and seeds, the deadline check on zero time (clock-free), the
`partition` check's recorded two-draft history, and falsification companions
asserting the *reason* (`Rejects` + `.Contains` of the check's own message,
`falsify.go:289-295`) are the standard the rest should be held to. Against it:

1. **`zeroonerror` is unsound as a signature-derived check.** Emitted for any
   `(T, error)` method with input (`suite.go:855`), it *demands* a failure
   path, failing correct total functions — dropped by hand at 13 corpus sites
   across 8 interfaces; `mixin/total`'s fixture states the contradiction
   verbatim; `defaultonerror` collides with it unfixed. And its falsification
   guard proves the wrong claim: the violator succeeds, so the guard matches
   the precondition message (`"supply inputs it misses"`,
   `falsify.go:322-326`) — the named claim ("an error carries the zero value")
   is the one check family with **no falsification at all**.
   `plausibleReturns` already exists to fix it.
2. **Silently-green checks on healthy subjects.** `wrappedvia` early-returns
   when `Cause()` is nil — which it always is against a fresh subject
   (`wrappedviatest/iface_suite.gen.go:265-270`); `batchsize` early-returns
   whenever the derived miss key makes the batch call error, which the
   reference implementation guarantees. Both run zero assertions and report
   green; both should be visible skips at minimum.
3. **Bare `Error` without identity**: `if-absent` (the fixture's own comment: a
   store refusing every write passes), `orderafter`, and the negative branches
   of `validates` / `if-match`. Each directive could carry a sentinel.
4. **The vocabulary is context hygiene.** Of 839 generated checks corpus-wide:
   smoke 207, nil-context 194, deadline 184, cancel 184, zero-on-error 55,
   miss 5, batch 2. **832 of 839** are the signature quartet; 401 (smoke,
   nil-ctx) are recover-only liveness probes. 14 of 20 detectors, 45 of 54
   mixins and 23 of 26 contracts emit no suite check of their own.
5. **The "checked somewhere else" header lies in 7 suites.**
   `Coverage.Elsewhere()` is "a law exists" (`coverage.go:68`), not "a model
   binding was queued" — `scheduled`, `lang/generic`, `lang/genericbound`,
   `lang/embedded`, `lang/nocontext`, `lang/receivercollision`,
   `integration/validated` point readers at model output that does not exist.
   The `Double` wiring (`suite.go:1358-1368`) shows how to gate it on the emit
   queue.
6. Smaller: `sample`'s partner args are never rendered — a builder taking
   arguments after `ctx` emits a non-compiling call (latent, unexercised);
   `Coverage.Checked` is OR-ed across methods so partial coverage reads as
   full (`coverage.go:161`); `FixtureField.OK()` accepts a companion as proof
   of a distinct alternate, so `XOther` can silently be the zero value
   (`fixture.go:128-130`); stale counts in doc comments ("twenty-four
   contracts", "seventy-two", the orphaned `mixinChecks` docblock at
   `suite.go:1005-1014`).

Missing signature-derivable checks the tier's own remit covers: idempotent
`Close` and use-after-close for `lifecycle` (the fixture doc states the law;
nothing emits it), repeat-call agreement for `pure`/`predicate`, a concurrent
smoke (N goroutines × derived value — CI already runs `-race` ×3 and no
generated file contains a goroutine), a goroutine-leak guard (`leakfree` is
registered and unchecked; `CheckGoroutineLeaks` ships with zero callers), and
a `NoError` on the seeded write to give smoke a positive path.

## F4 — per-fixture identity is unenforced (drift is already real)

The gate deliberately measures stamps corpus-wide, not per directory. eidos's
detector vocabulary has moved, and two detector-axis fixtures no longer
exercise their named shape:

- `detector/compositewriter` — doc: "a value in, the stored value out";
  `Store(ctx, v Value) (Value, error)` is stamped **reader** (its model
  sequence runs `Store (reader)`), while two-argument writers elsewhere now
  carry the `compositewriter` stamp (`idempotent`'s `Put`).
- `detector/multiargwriter` — `Set(ctx, key, body string) error` is stamped
  **compositewriter**; three-argument writers elsewhere carry `multiargwriter`.

Coverage stays green because the stamps exist somewhere. The fixture that
exists to pin a detector's dispatch does not pin it — its package doc's claim
("a detector that misfires shows up as wrong generated output") holds only if
someone reads the output.

## F5 — dead capability and dead gates

Engine capability with zero non-test importers, verified workspace-wide:

| Capability | Status |
|---|---|
| `engine/model/bmc` | dead — `Outcome.Violated()` never checked anywhere |
| `engine/model/mutation` | dead — 8 semantic operators (`DropWrites`, `ReturnWrongValue`, `LossyStream`, …); its doc names the model generator as consumer; the generator hand-rolls inert mutants instead |
| `engine/model/shrinker` | dead — its doc claims "the model runner composes them on failure"; the runner does not import it |
| `engine/model/domhint` | dead — `reflect.Type`-keyed design cannot serve a `go/types` generator |
| `engine/model/timeaware` checkers | quarantined `needs-isolation` (`conformance/gate/conduct.go:151-158`); unreachable from any declaration |
| `CheckGoroutineLeaks` | no callers; comment claims wrapper-level use |
| Coverage plumbing (`fired`, `FireRate`, `WeakLaws`, `WithStateHash`, `WithCoverageSink`) | structurally empty from any runner-driven run |
| 17 of 22 `ref` oracles, 22 of 43 action constructors | unreachable from the generator |
| `<Iface>ModelValues` | emitted in 61 files as the documented remedy for narrow pools; **zero callers in the repo** |

Gate hygiene:

- `mutation` disabled in `.ergon.yaml` `checks.disabled`; the `conformance/...`
  thresholds (`score: 100`) are enforced by nothing CI runs.
- Two blanket skips (`func_glob: "*"` over `**/*_suite.gen.go` and
  `**/*_model.gen.go`) exclude the corpus's generated output from the coverage
  and mutation gates. The first skip's own comment says to remove it when the
  falsification companion ships; it shipped.
- `gate.Walk` has no caller outside its own tests — its failure arms protect a
  traversal nothing performs. `TestResolutionReadsTheCorpus` discards the
  result its comment claims to assert.
- No regen gate: the drift checker exists (`cli.ExitCheckDrift`) and nothing
  invokes it; `go:generate` appears nowhere under `conformance/`. The tree is
  currently consistent only because it was regenerated by hand.
- `conformance/doc.go` still says "fixtures are declarations only — no
  implementations"; 113 `inmemory.go` files disagree.
- `mixin/nilsafe/nilsafetest/gens.go` is untracked but required to build.
- The `lang` axis has no model files at all (the model generator is never
  exercised against generics or embedding); `lang/generic` and
  `lang/genericbound` ship suites with no falsification companion;
  `lang/function` generates nothing; `mixin/scheduled` is the one mixin with
  no model file while its suite header still defers to the model tier.

## The corpus, fixture by fixture

Legend: **ref** — twin (subject vs. itself) or derived oracle; **own law** —
whether the fixture's namesake classification is bound in its model file;
**kill** / **porc** — companion kill matrix / Porcupine leg emitted. Laws in
`Not bound` are selected by the classification and dropped for a missing
instantiation spec.

### Detector (20) — all twins, no laws, no kill matrices

| Fixture | Sequences | Suite check beyond the signature quartet |
|---|---|---|
| aggregator | Count (aggregator) | none |
| batchreader | GetAll (batchreader) | batch-size (vacuous in main run, F3.2) |
| compositewriter | Store (**reader** — drifted, F4) | none |
| lifecycle | Close (lifecycle) | none — its own stated law (idempotent Close) unemitted |
| lookup | Inspect (lookup) | miss-flag + zeros |
| multiaggregator | Stats (multiaggregator) | none |
| multiargwriter | Set (**compositewriter** — drifted, F4) | none |
| multireader | GetWithMeta (multireader) | none |
| mutator | Touch (mutator) | none; model action asserts nothing |
| pointerreader | Find (pointerreader) | miss-zero |
| poisonaccessor | Err (poisonaccessor) | none; poison laws unbound |
| predicate | IsEmpty (predicate) | none; repeat-agreement unemitted |
| pure | Describe (pure) | none; determinism unemitted |
| reader | Get (reader) | miss + zero-on-error |
| readernoerror | Lookup (readernoerror) | miss-zero |
| readerwithbool | Load (readerwithbool) | miss-flag |
| streamconsumer | Next (multiaggregator) | none |
| streamreader | List (streamreader) | none; stream laws unbound |
| voidlifecycle | Stop (voidlifecycle) | none; model action asserts nothing |
| writer | Put (writer) | none; `AUTO-WRITE-OBSERVABLE` unbound (no reader to observe through) |

### Mixin (54) — 19 derived, 35 twins; own law bound in 5

| Fixture | Ref | Own law bound? | Notes |
|---|---|---|---|
| associative | twin | no | `AUTO-ASSOCIATIVE` unbound |
| atomic | twin | no | `AUTO-ATOMIC-WRITE` unbound |
| bounded | twin | no | **proven unasserted by break experiment** |
| cacheable | twin | no | `AUTO-CACHEABLE` unbound |
| causal | derived+porc | no | binds `WriteObservable` only |
| commutative | twin | no | |
| concurrent | twin | no | no goroutine in any generated file |
| concurrentreaders | derived+kill | no | binds nothing |
| conservative | twin | no | |
| crdtmerge | twin | no | two-replica law needs an oracle it never gets |
| defaultonerror | derived+kill+porc | **yes** | `DefaultOnError` + `WriteObservable` |
| deleteremoves | derived+kill | **yes** | `DeleteReturnsNotFound` |
| deprecated | twin | n/a (suite-owned) | |
| errors | twin | n/a (suite-owned sentinel check) | |
| eventually | twin | no | convergence law unbound |
| hooks | twin | n/a (suite-owned, real check) | |
| idempotent | derived+kill | no | **break experiment red via differential, not the law** |
| injectionsafe | twin | no | law exists (`InjectionSafe`), unbound |
| integrationonly | twin | n/a (suite guard) | |
| leakfree | twin | no | `CheckGoroutineLeaks` never emitted |
| lifecycleafterclose | twin | no | quarantined `needs-isolation` |
| monotonic | twin | no | |
| monotonicreads | derived+kill+porc | no | binds `WriteObservable` only |
| monotonicwrites | derived+kill+porc | no | same |
| nilsafe | twin | n/a (suite-owned, real check) | `gens.go` untracked |
| noduplicates | derived+kill | **yes** | `StreamNoDuplicates` |
| orderafter | twin | n/a (suite-owned, bare `Error`) | |
| overmatch | derived+kill | no | `StreamOverMatch` blocked on supplied field |
| partition | twin | n/a (suite-owned, strongest check in the tier) | |
| permutation | derived+kill | no | `StreamPermutation` blocked on supplied field |
| pointintime | derived+kill+porc | **yes** | `PointInTime` + `WriteObservable` |
| poisonable | twin | no | all three poison laws unbound |
| pure | twin | no | |
| readafterwrite | derived+kill | **yes** | `ReadAfterWrite` |
| readyourwrites | derived+kill+porc | no | session law unbound; single-client trace would weaken it anyway (`runner.go` stamps `ClientID: -1`) |
| retrysucceeds | twin | n/a (suite-owned per RFC; unemitted) | |
| sample | twin | n/a (suite-owned, real check; latent arg bug F3.6) | |
| scheduled | **no model file** | no | suite header still defers to the model tier |
| scope | twin | n/a (suite-owned per RFC; unemitted) | |
| sideeffect | twin | n/a (suite-owned, real check) | |
| snapshotisolation | derived+kill | no | G0/G1/G2 laws (real Adya implementation) unbound |
| stableorder | derived+kill | no | `StreamStableOrder` blocked on `Less` |
| sticky | derived+kill | no | binds nothing; `WriteObservable` negated by `sticky` (the one negation row) |
| streamreflectsmutations | twin | no | |
| tamperevident | twin | no | quarantined `needs-isolation` |
| timeaware | twin | no | clock laws unreachable (no `clocked` mixin upstream) |
| timeout | twin | n/a (suite-owned, real check) | `AUTO-DEADLINE-RESPECTING` also unbound |
| total | twin | no | `zeroonerror` contradiction documented in fixture |
| ttl | derived+kill+porc | no | `AUTO-TTL-EXPIRY` quarantined |
| validates | derived+kill+porc | n/a (suite-owned, real check) | |
| windowed | twin | no | |
| wrappedvia | twin | n/a (suite-owned, **vacuous on healthy subject**, F3.2) | |
| writesfollowreads | derived+kill+porc | no | binds `WriteObservable` only |
| xsssafe | twin | no | `XSSSafe` law (real string assertion) unbound |

### Contract (26) — own law bound in 0

| Fixture | Ref | What actually runs |
|---|---|---|
| appender | twin | writer error-presence only |
| batch-writer | twin | writer error-presence only |
| cache | twin | two readers vs. themselves |
| cas | twin | `AUTO-CAS-ATOMIC-ONE-WINNER` unbound; oracle `AtomicCell` unreachable |
| chain | derived+kill | differential over Append/Replay; all six chain laws unbound |
| circuit-breaker | twin | no tier owns it (RFC); gate cannot see that |
| codec | twin | `AUTO-ROUNDTRIP` unbound — a lossy codec passes |
| cursor | twin | cursor laws quarantined |
| if-absent | twin | suite check bare `Error` (F3.3) |
| if-match | twin | suite check real on both branches |
| leader-election | twin | no tier owns it |
| lease | derived+kill | lease laws unbound; oracle `Lease` unreachable |
| outbox | twin | suite check real (liveness-guarded subscribe) |
| pagination | twin | paginator laws unbound |
| persister | derived+kill+porc | `WriteObservable` bound; `AUTO-PERSISTER-RETRIEVABLE` unbound |
| pool | twin | pool laws unbound; oracle `Pool` unreachable |
| publisher | twin | delivery laws unbound |
| rate-limit | twin | no tier owns it |
| saga | twin | compensation law unbound; oracle `Saga` unreachable |
| singleflight | twin | coalescing law unbound (and needs concurrency) |
| transaction | twin | rollback + mid-tx-visibility laws unbound |
| tx | twin | no tier owns it |
| updater | derived+kill+porc | `WriteObservable` bound; `AUTO-UPDATER-REPLACES` unbound |
| upserter | derived+kill+porc | `WriteObservable` bound; `AUTO-UPSERTER-IDEMPOTENT` unbound |
| watcher | twin | watch law unbound |
| workflow | twin | transition law unbound |

### Composite (4) and lang (10)

The composite axis exists to prove the three families compose; the generated
files prove context hygiene composes. `batched-mixins` (derived) binds
`ReadAfterWrite`; the other three are twins binding nothing —
`tx-with-retry`'s twelve generated checks are the context quartet three times
over, and every semantic assertion is hand-written in `inmemory_test.go`
(whose null-subject sweep with reason-asserting guards is the pattern the
generators should emit). The lang axis carries `//testkit:suite` only — the
model generator is never exercised against embedding, generics, variadics or
named returns; `generic`/`genericbound` additionally ship no falsification
companion, and `lang/function` produces no output at all.

## Improvement programme

Ordered; each item names its finding. Landed items are struck from the list
rather than annotated — the register and the gates carry their state now.

Landed: the union gate (item 1 — `gate.Emission` + `gate.UnboundLaws`, the
two-way ratchet), the bindings table (item 3 — 38 laws bound, 35 registered
survivors each naming its true chokepoint), kill-matrix identity and the
`InertBodies` deletion (item 5's first half; the `mutation`-operator rows
remain under item 8's wire-or-delete).

1. **Raise the twin floor (F1).** Teach `referenceOf` the contract-role
   oracles that already ship, add contract-role action dispatch, and gate: a
   fixture for a model-owned classification whose reference is a twin fails.
   Interim honesty: report twin runs under `model/twin` so the weaker claim is
   visible in test output.
2. **Suite fixes (F3).** Suppress/repair `zeroonerror` (shape- and
   mixin-aware; falsify its real claim via `plausibleReturns`); visible skips
   for `wrappedvia`/`batchsize`; sentinels for `if-absent`/`orderafter`;
   per-method `Checked`; gate "checked somewhere else" on the queued model
   emit; emit idempotent-Close, use-after-close, pure-repeat and concurrent
   smoke; `NoError` on the seeded write.
3. **Determinism and gates (F2, F5).** Seed rapid in CI or persist the seed on
   failure; a policy for Porcupine `Unknown`; remove the expired `.ergon.yaml`
   skips; wire `ExitCheckDrift` into `make check`; fix `conformance/doc.go`;
   commit `gens.go`.
4. **Per-fixture identity (F4).** A gate row asserting each detector-axis
   fixture's method carries the stamp its directory names.
5. **Delete or wire (F5).** `bmc`, `domhint`, `shrinker`, `mutation`: each
   gains its consumer within a release or goes. Their doc comments currently
   claim integrations that do not exist.

**The argument against this order.** Item 1 goes red on ~60 classifications
the day it lands and stays red until 2–3 finish; a standing red gate is a gate
people learn to ignore, and ADR-0017 already concedes that pressure produces
weak assertions. The alternative is to land 2–3 family-by-family with the
gate's expected set ratcheting alongside — a shrinking, tracked exclusion list
with a reason per entry: debt recorded, not laundered.

## Reproducing this

The break experiments (revert after):

```sh
# bounded: delete the clamp in boundedtest/inmemory.go List, then
go test -count=1 ./conformance/corpus/iface/mixin/bounded/...   # stays green

# idempotent: make Put accumulate (s.items[key] += value), then
go test -count=1 ./conformance/corpus/iface/mixin/idempotent/...  # fails, shrunk
```

The census (run from the repo root; each number above re-derives):

```sh
# reference kinds
grep -l "the subject's own factory" conformance/corpus/iface/*/*/*test/iface_model.gen.go | wc -l   # twins: 78
ls conformance/corpus/iface/*/*/*test/iface_model.gen.go | wc -l                                    # model files: 103
ls conformance/corpus/iface/*/*/*test/iface_model.gen_test.go | wc -l                               # companions: 25

# bound laws, by type and by file
grep -h 'laws\.Add(law\.' conformance/corpus/iface/*/*/*test/iface_model.gen.go | sort | uniq -c
grep -l 'laws\.Add(law\.' conformance/corpus/iface/*/*/*test/iface_model.gen.go | wc -l             # 17

# selected-then-dropped laws
grep -h 'the catalogue carries no instantiation spec' conformance/corpus/iface/*/*/*test/iface_model.gen.go | wc -l

# suite check vocabulary
grep -rhoE 'cfg\.run\(t, "[^"]+", "[^"]+"' conformance/corpus/iface/*/*/*test/iface_suite.gen.go |
  sed 's/.*", "//' | sort | uniq -c | sort -rn

# suite falsification: identity asserted
grep -rl 'Rejects(t,' conformance/corpus/iface/*/*/*test/iface_suite.gen_test.go | wc -l
# model kill matrices: identity not asserted
grep -rh 'f.Failed()' conformance/corpus/iface/*/*/*test/iface_model.gen_test.go | wc -l

# dead engine capability
grep -rln 'engine/model/bmc\|engine/model/mutation\|engine/model/shrinker\|engine/model/domhint' \
  --include='*.go' . | grep -v _test.go | grep -v '^\./engine/model/'                               # empty
```
