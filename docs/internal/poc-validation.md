# PoC validation: do the derivation rules transfer?

Pack 1 of the second-domain programme. Method: hand-simulate the
generator for a domain the rules were not written from — a topic-keyed
publish/subscribe bus (`gen/example/bus` → `gen/example/bustest`) — by
applying `derivation-rules.md` row by row. Every emission cites its rule
or lands in the gaps ledger; nothing is improvised over silently. The
run is the recomputable evidence:

```
bustest.Bus: 19 checks x 1 subjects = 19 legs (19 passed, 0 failed, 0 did not run)
  of 19 checks that ran: 19 proven able to fail
```

`GOWORK=off go test ./example/...` in `gen/`, race-clean, lint-clean
under the root config. Verdict vocabulary: **survives** (rule applied
unchanged), **branches** (rule needed a domain arm), **new** (no row
existed; one is owed), **broken** (the rule as written would emit
something wrong), n/a (input shape absent — not evidence either way).

Why the bus: it inverts kv's data flow (data leaves through a channel
the subject hands back), delivery order is legally nondeterministic, and
the engine already carries the delivery vocabulary — so the pack tests
the *derivation path*, not engine capability.

## Rule-by-rule verdicts

| derivation-rules.md row | verdict | evidence / ruling |
|---|---|---|
| `//testkit:ctx` → cancel/deadline/nilcontext | **survives**, gap in prose | All three families on Publish and Subscribe. The corpus omits `Close/deadline` (kv and bus both) and the table does not state that exception — the rule as written over-emits by one check. Amendment A5. |
| one smoke per method | **survives** | Three smokes. |
| `//testkit:role` + `//testkit:default` on struct fields | **branches** | Publish and Subscribe take *bare parameters*; there is no request-struct field to stamp. The stamps moved to the named type declaration (`bus.Topic`). Amendment A2. |
| pool provenance (derived blends adversarial; supplied verbatim) | **survives** | `busWideTopics`/`busWideMessages`, value-compare provenance, verbatim. |
| Reader+Writer over one key → `ref` map oracle | **branches** | Publisher+Subscriber derive a *fan-out* reference — topic → open subscriptions, never dropping — written against the shape. Amendment A3. |
| written ≠ read type → Porcupine builder | n/a | No linearizable leg emitted; see the broken row. |
| "safe for concurrent use" → AUTO-LINEARIZABLE | **broken** | Bus declares the sentence, and the rule as written would emit a Porcupine leg that cannot be specified: async fan-out has no per-key register model, and delivery is not a linearizable read. The concurrent claim lowers to the delivery laws under concurrent load — engine work the inline PoC does not drive. This is the pack's weightiest finding: applied mechanically to this domain, the rule emits a wrong check. Amendment A6. |
| mixin ttl / appender / chain bundle / seed seam | n/a | Input shapes absent. |
| sentinel + Induce seam → AUTO-POISON-CONSISTENT | **survives** | Probe maps the sentinel onto the publish surface. |
| sentinel doc prose on Close → AUTO-LIFECYCLE-AFTER-CLOSE | **survives** | `Op` = Publish. |
| chain shape → bundled observational laws | **new ruling by absence** | Every bus law *writes* (a delivery law subscribes and publishes to observe), so the observational bundle is empty and each law rides its own leg. Generalization: observational laws bundle, writing laws ride alone, and a domain with no observational laws has no `model/laws` leg at all. Amendment A4. |
| law IDs always from `lawid` | **survives** | All four AUTO- segments are `lawid` constants. |
| ≥2 model-bearing interfaces → qualified family scopes | **survives — first exercise of the other arm** | One interface → empty qualifier: `model/differential`, not `model/bus/differential`. The kv corpus never reached this arm. |
| Recover seam → sim family | **survives as absence** | No medium, no seam, no sim family — recorded as a ruling, not skipped. |
| Prop sugar only for drawable domain inputs | **branches, small** | `PropPublish` carries *two* drawn parameters (topic, msg) — the kv sugar always drew one. Rule: sugar passes one draw per roled parameter. `PropSubscribe` draws the topic; Close earns nothing. |
| bodies' extra param = the interface's draw source | **survives** | Rows take `BusFixture`. |
| defect sugar follows the constructor seam | **survives** | `BrokenBus` is the plain two-field form. |
| class defaults to the ID family | **survives** | Delivery laws class as `model/laws`; no annotation needed. |
| defect-emitter rules (W1…F1 + signature one-liners) | **survives + two new** | W1 discard-write → the forgetful bus; the signature one-liners transplant verbatim. New: **D1 partial-fanout** (deliver to one subscriber; breaks the per-subscriber bound alone) and the differential's *refusal* defect — a refused publish is `Vacuous` to every delivery law, so only the differential can catch it, which is what makes the defect single-claim. Amendment A7. |

## Rules the table was missing (engine-documented, no row)

- **A1 — `//testkit:contract publisher role=publish subscribe=<M>
  [redeliver=<M>] mode=<mode>`** (spelling corrected in review). The
  conformance corpus is the directive grammar's home: single-method
  claims are `//testkit:mixin <claim>`, multi-method protocols are
  `//testkit:contract <name>`. The pack initially shipped the ENGINE
  DOCBLOCK's bare spelling (`//testkit:publisher <Subscribe>`) — the
  docblocks are a drifted second home of the vocabulary, and the review
  caught what the hand-simulation reproduced. Upstream docblock fix
  owed (`law/contract.go`, `law/aggregator.go`, `law/reader.go` all
  spell directives bare). The Phase-2 contract before that proposed
  minting `//testkit:mixin delivery=` — the vocabulary existed both
  times. Two misses, one lesson: the grammar needs one home, and it is
  the corpus, not the engine docs.
- **A8 — channel-returning methods.** Lower to Subscriber/Watcher: the
  differential compares the open path only; the delivered stream is the
  delivery laws' observation. The engine owns this split ("channel
  comparison is intractable across impls"); the table must state it so
  a generator does not try to compare channels.
- **A9 — law-private keys.** Delivery laws run their cycles on a key
  *outside* the drawn pools (`busLawTopic`), or the action stream's
  ambient traffic lands in the law's drain between publish and read.

## Open questions the pack surfaced (not resolvable by this corpus)

- **Drain and quiescence.** The bus contract makes delivery synchronous,
  which licenses a non-blocking drain. An *async* bus needs a quiescence
  seam — a harness field? a directive parameter? — before the delivery
  laws can drain honestly. The generator cannot derive this from shape.
- **Harness field derivation** (recorded as ruling A10): the harness
  carries only the capability fields its emitted check set can demand —
  BusHarness has no OnClock and no Recover. Uncontroversial here;
  needs restating as a rule so the generator derives fields from caps,
  not from a fixed template.

## Score

Of 18 applicable rows: **11 survive, 4 branch, 1 breaks, plus 6 new
rows owed** (A1–A9 above, counting the defect rules together). The
branches are all domain arms, not contradictions; the one break
(linearizable) is real and is exactly the kind of wrong emission the
scorecard exists to catch before a generator ships it.

## Residual risks, stated plainly

- This pack is still a hand-simulation: the same judgment that wrote the
  rules applied them. The citation discipline and the one honest
  **broken** verdict are the mitigations; the definitive test remains
  eidos emitting this package from `bus.go`.
- The negative control (`doubleBus`, delivers everything twice) pins
  specificity for duplicates only; ordering-tolerance has no control yet
  because no generated check constrains order — add one when an ordered
  delivery mode lands.
- `InMemory` drops on a full subscription buffer (`subCap`), documented
  as deliberate naivety; the conformance sequences cannot reach it, but
  a soak test would.

---

# Pack 2 — the bounded cache (`gen/example/cachetest`)

The domain chosen to break the differential: a capacity-bounded cache
whose eviction victim is deliberately unspecified. Written against the
A-amended table. The run:

```
cachetest.Cache: 10 checks x 1 subjects = 10 legs (10 passed, 0 failed, 0 did not run)
  of 10 checks that ran: 10 proven able to fail
```

## Rule-by-rule verdicts (rows already scored by pack 1 omitted where
the verdict repeats)

| rule | verdict | evidence / ruling |
|---|---|---|
| role/default stamps | **survives** | Both homes in one source: type-level on `Key` (A2's arm), field-level on `Value.Body`. |
| stock differential vs a reference | **broken → asymmetric ruling** | The pack's centerpiece. Legal eviction means a correct subject misses where the unbounded reference hits; stock ReaderWithBool (value AND ok must match) reds correct code. Ruling: a subject HIT must agree with the reference and an unexplained hit is invention; a subject MISS is always legal; `Len` is excluded from the differential entirely — the bounded law owns it. Spelled inline via `action.Unknown`; **engine vocabulary owed** (an eviction-aware reader action). Amendment B3. |
| `//testkit:mixin bounded limit=N` | **new** | The literal is the bound, the suite owns the number, and every harness constructor RECEIVES it — the Catalog seed seam generalized to a scalar. Amendment B1. |
| `//testkit:mixin cacheable` on an errorless reader | **new** | `Cacheable` over the packed (value, ok) observation. Amendment B2. |
| no `//testkit:ctx` | **survives + new ruling** | Counter's doctrine at model-bearing scale: smokes only. And `zero-on-error` is NOT emitted — its error source is the cancelled context that only a ctx claim licenses. Amendment B4. |
| observational bundle (A4's other arm) | **survives** | Bounded + cacheable bundle into `model/laws`; the bus had zero observational laws and no bundle; the dividing line holds from both sides. |
| tier honesty in law legs | **new ruling** | kv's laws leg notes a tier because its laws read the reference; the cache bundle observes the subject alone, so the leg notes NO tier — a tier nobody compared against would be an invented measurement. Amendment B5. |
| sentinel + induce / lifecycle prose | **survives** | Probes ride the WRITE surface, because Get has no error slot. |
| Prop sugar per drawable input | **survives** | PropPut draws two, PropGet one, Len and Close none — the pack-1 multi-draw branch reapplied cleanly. |
| defect-emitter rules | **survives + two new + one lesson** | I1 invent-hit (differential-only by construction), G1 exceed-bound. The lesson: the first-draft defect — a never-evicting twin — PASSED its proof, because crossing the bound needs more distinct keys in one drawn sequence than the budget reliably supplies. Two consequences, both recorded: proofs must red deterministically under the proof budget, and a `bounded` literal must sit within the sequence budget's reach or the eviction path ships untested behind a law that reads as coverage (the source moved from limit=4 to limit=2 for exactly this reason). Amendment B7. |
| errorless method on the stub | **new hazard** | An injected fault has no error slot to surface through; it answers the zero return — a miss. Documented at the stub. Amendment B6. |

## Pack-2 score

Of the applicable rows: **6 survive, 1 breaks (the differential — the
break the pack was designed to force), 5 new rows owed (B1–B7,
counting the defect rules together), plus the grammar correction** that
retro-fixed pack 1's A1. The falsifiability harness earned its keep
concretely: it caught its own first-draft defect slipping through,
which produced the sequence-budget rule no amount of a-priori derivation
had surfaced.

---

# Pack 3 — pool + lease (`gen/example/pooltest`)

Concurrency-first resource lifecycles: no keyed reads anywhere, the
lease's central claim lives in a goroutine, and two model-bearing
interfaces share one package. Written against the B-amended table.
The runs:

```
pooltest.Pool:  6 checks x 1 subjects = 6 legs (6 passed, 0 failed, 0 did not run)
pooltest.Lease: 5 checks x 1 subjects = 5 legs (5 passed, 0 failed, 0 did not run)
  of N checks that ran: N proven able to fail   (both suites)
```

## Rule-by-rule verdicts

| rule | verdict | evidence / ruling |
|---|---|---|
| `//testkit:contract lease role=acquire release=<M> timeout=<D> held=<E>` | **survives** | The corpus fixture's own spelling, applied verbatim; `held=` binds the double-acquire law, `timeout=` bounds the cancel poll. |
| pool contract directive | **survives + PROPOSED extension** (erratum) | This row first shipped claiming "the corpus has no pool spelling at all" — FALSE: `//testkit:contract pool role=get put=Put` exists in `conformance/corpus/iface/contract/pool`, and pack 3's recon missed it behind a truncated grep. The erratum stands here because the falsifiability discipline applies to the validator's claims too. What IS new: the corpus fixture has no Stats surface, so the balanced/leak-free laws' observation is unwired there — the `stats=<M>` parameter is the proposed extension. Amendment C1, corrected. |
| quiescence laws | **new ruling** | `PoolLeakFree` is bindable ONLY because the engine's Pool action is cycle-shaped — get-then-put per invocation, so the pool is at quiescence between actions, where laws run. A bare Get action in the stream reds correct code. Amendment C2. |
| contract-owned context semantics | **new ruling** | Release-on-cancel is a ctx behaviour claimed by the CONTRACT while NO `//testkit:ctx` is declared and no signature families emit — the two axes are independent, proved side by side. Amendment C3. |
| contract key parameters | **new ruling** | The lease key is a plain `string` (the corpus's own shape): a contract role's key parameter derives its pool from the contract, no role stamp needed. Amendment C4. |
| lease freedom | **new ruling** | No observer method exists, so the released-on-cancel law's Free probes through the acquire/release pair, self-cleaning. Amendment C5. |
| the polling law's budget | **measured, downgraded** | The feared timeout x iterations explosion does not materialize: an expected red short-circuits the property at its first failing iteration, so the deaf-watcher proof costs shrink-attempts x timeout (~2.5s package total, measured). The leg still binds the law directly on a bare loop with a manual vacuity census — an action stream would multiply cost for no evidence — and the per-leg iteration budget drops from blocker to engine improvement owed. Amendment C6. |
| A10 harness-from-caps | **survives, fully** | First pack where BOTH harnesses carry zero capability fields — no OnClock, no Recover, no Induce, because no check demands any and no sentinel exists. Also the first pack with no lifecycle or poison tier at all: absence derivations on every axis. |
| A18 qualified family scopes | **survives** | Two model-bearing interfaces → `model/pool/...`, `model/lease/...` — the qualified arm re-exercised outside kv. |
| no-config interface (Journal precedent) | **survives + sub-ruling** | Pool has no config, no fixture, no Prop sugar: a method whose input is PRODUCED by a sibling (Put's Conn comes from Get) has no drawable domain input, and its smoke borrows first. Amendment C7. |
| defect-emitter rules | **three new** | A1 lying-accounting (red on the balanced law's first read), L1 grant-always, L2 deaf-watcher (a correct lease acquired under a context nobody watches). Amendment C8. |

## Pack-3 score

Of the applicable rows: **5 survive** (including two arms exercised for
the first time), **0 break**, **7 new rows owed** (C1–C8, counting the
defect rules together), one **proposed directive** awaiting corpus
ratification, and one prior finding **downgraded by measurement** (the
polling-law budget). The negative controls carry the pack's thesis
twice: recycle order is unclaimed (FIFO recycler passes) and the cancel
release is bounded, not instant (a dawdling watcher inside the declared
timeout passes).

---

# Pack 4 — the cursor store (`gen/example/scantest`)

The missing-feature closure: the gen README deferred cursor laws
pending "a cursor-typed sub-harness design". Written against the
C-amended table. The run:

```
scantest.Log: 6 checks x 1 subjects = 6 legs (6 passed, 0 failed, 0 did not run)
  of 6 checks that ran: 6 proven able to fail
```

## Rule-by-rule verdicts

| rule | verdict | evidence / ruling |
|---|---|---|
| the cursor-typed sub-harness | **answered: none exists** | The deferred design question dissolves into two existing rulings composed: a produced secondary interface's laws instantiate at the SECONDARY's type and ride the pack-3 direct-binding leg, with the producing method as the constructor — one fresh cursor per iteration, a refused open counted as unengaged. The harness learns nothing. Amendment D2. |
| cursor contract directive | **new arm — PROPOSED** | The corpus spells the contract on a standalone cursor's own Next (`role=next close=Close sentinel=ErrClosed`); a PRODUCED cursor hosts it on the producer with `role=open next=Next close=Close sentinel=...`. Needs corpus ratification. Amendment D1. |
| cursor-shaped replay | **answered by composition** | The README's "the Journal replays as a slice" gap is drain composition, not a new action shape: open, drain via Next, close, and the result lowers to the engine's Stream action, order compared against the reference. Amendment D3. |
| signature tier for produced interfaces | **new ruling** | The produced Cursor gets no smokes, no index entries, no stub: it is covered by its contract laws, reached through the producer; the opener's own smoke closes what it opens. Stub emission follows `//testkit:stub` on the suite-bearing interface — cursor defects are small hand types over a real cursor (the Journal precedent). Amendment D4. |
| Journal-shape derivations (no config, PropAppend, A10 minimal harness, empty qualifier, B4 no-ctx) | **survive** | All reapply without friction — the fourth domain in a row for most of them. |
| defect-emitter rules | **two new** | K1 refuse-teardown (the second Close errors) and K2 outlive-close (Close acknowledged and ignored; Next answers where the sentinel is owed), both as hand cursor types the LogStub's Scan override returns. W1 discard-write reapplies as drop-every-other-append, caught by the drained differential. Amendment D5. |
| negative control | **strategy, not policy** | A cursor reading lazily behind a boundary captured at open is a genuinely different implementation strategy that every claim tolerates — while the consumer's `own/scan-snapshot` check pins the isolation sentence only this implementation makes, its defect a boundary-less live cursor. |

## Pack-4 score

Of the applicable rows: **6 survive, 0 break, 5 new rows owed**
(D1–D5), one **proposed directive arm** awaiting corpus ratification —
and the deferred design question the pack existed to answer closed with
a ruling instead of a mechanism.

---

# Programme verdict

Four packs, five domains beside kv (pub/sub, bounded cache, pool,
lease, cursor log), 15,000+ lines of hand-simulated corpus, every leg
green under `-race`, every check proven able to fail, lint-clean, and
every emission citing a rule or landing in an amendment. Across the
programme:

- **Rule table**: ~28 rows survived contact, ~5 branched into domain
  arms, **2 broke** (Porcupine-on-async-fan-out, stock-differential-
  under-eviction — both would have shipped wrong emissions), and the
  A/B/C/D series added ~26 rows the kv corpus alone could never have
  produced.
- **The falsifiability discipline caught its own operators**: a
  first-draft defect that slipped its proof (pack 2, sequence-budget
  rule), a truncated-grep erratum (pack 3), and two directive
  spellings I got wrong until review or the corpus corrected them.
- **Standing gaps, honestly held**: the harness `Provides` registry has
  zero validation evidence in any pack; the async-bus quiescence seam
  is not derivable from shape; three engine items are owed (directive
  docblock drift fix, an eviction-aware reader action, a per-leg
  iteration budget); two directive arms await corpus ratification
  (`pool stats=`, `cursor role=open`); and the pilot measurement —
  time-to-unblock-red with a team that didn't build this — remains the
  one gate no amount of corpus can pass.

What the generator consumes now exists in one place: the amended
`derivation-rules.md`. What would falsify the programme's conclusion —
a domain whose emissions cannot cite the table — has been hunted
through five domains and found twice, both times before a generator
could ship it. That is the PoC validated to the limit of self-applied
evidence; eidos emitting these packages from their sources is the step
that retires the rest.

---

# Post-programme closes (the gap sweep)

The gap ledger drawn after pack 4 was worked down in one pass; what
each close proved, and what remains, recorded here so the ledger stays
one document.

**Closed with run evidence:**

- **The `Provides` open door** — `bustest`'s hard-mode run declares an
  open capability (`bus.fanout-limit`, a number only the deployment
  knows), both harnesses answer it through the new `Provide` field, the
  consumer's `own/fanout-capacity` check reads it off the subject, and
  its planted defect carries the door so it reds for the claim rather
  than the wiring. The `Provide` field currently exists on BusHarness
  only; consumer rows make open needs declarable everywhere, so the
  generator emits it uniformly — the corpus exercises one.
- **Oracle, Serial, Excused beyond kv** — the same hard-mode run:
  `21 checks x 2 subjects = 42 legs`, `oracle: "in-memory"`,
  `model reference: derived 1 | differential 1`, one Serial subject,
  one leg `did not run: excused`.
- **Prop\* sugar zero-callers** — every pack's consumer table now
  carries a sugar row with a planted defect that only the drawn hostile
  pool member can expose (refuses-hostile-inputs defects; the lease's
  acquire-release-reacquire cycle). The sugar earns its surface or it
  would have been cut, per the corpus's own veneer precedent.
- **Engine docblock directive drift** — ten sites across
  `engine/model/law` respelled to the corpus grammar (mixin/contract
  families). Found in passing: `//testkit:mixin lifecycleafterclose`
  exists in the corpus canon, so kv's prose-derivation of the
  after-close law contradicted "prose licenses nothing" — the main
  table row now carries the correction, and the gen corpus owes the
  directive on its sources.
- **Lint carve-outs** — the four config classes the lint-posture
  section documents are now root-config exclusions; the corpus lints
  clean, and the four residual findings the sweep surfaced were real
  defects in the sweep's own code, fixed.
- **Artifact-name collisions, fully** — the package token now carries
  the MODULE PATH (the pair Go guarantees unique) plus the
  module-relative directory; the hashed fallback stands outside
  modules.
- **The scrub's external assumption** — a subprocess canary stages a
  genuinely failing rapid property and asserts the reproduction file
  lands under `testdata/rapid/<TestName>`, where `scrubBucket` looks;
  rapid moving its layout now reds a test instead of silently
  re-littering the tree.
- **Grammar and renderer gates** — `FuzzValidateID` (never panics,
  judges deterministically, composer round-trip) and
  `BenchmarkReportText` (~3.2µs, 23 allocs per render at 60-leg scale;
  the report prints once per run, so the budget is honest).
- **The gen README** — go.work sentence, stub/builder placement, the
  23-vs-29 drift, the closed cursor bullet, and the corpus-beyond-kv
  table.
- **Rule-table double answers** — the main rows the amendments
  corrected now carry the corrections inline, with the amendment-wins
  rule stated at the table head.

**Closed after the sweep — the engine round:**

- **Three of the four engine features are DELIVERED**:
  `action.EvictingReader` (the asymmetric comparison, one constructor
  call in cachetest), `law.Budget` (the polling-law ceiling — per
  property iteration, refilled by the runner's reset, which is what
  keeps rapid's shrinking deterministic; pooltest's cancel leg rides
  it), and `law.Produced` (the produced-secondary lift; scantest's
  cursor legs ride it and the direct-binding pattern is retired from
  the corpus). Each shipped with its own law/action tests.
- **The docblock sweep is complete**: the remaining 41 bare directive
  spellings across `engine/model/law` and `action` now read corpus
  canon, including the three chain-role prose references. Found in
  passing: this table's own Inputs list still spells bare `appender`
  where the corpus canon is `contract appender role=fn` — the same
  drift class, on the rule-table side.

**Still open, unchanged in kind:**

- Engine: the async fan-out concurrency lowering and the quiescence
  seam — RFC-0005-shaped design, not a mechanical fix.
- ~~Corpus ratification~~ — DONE end to end. The mixin half is
  DONE: `mixin lifecycleafterclose` now sits on every sentinel-reporting
  operation of the gen sources (kv Put/Get/Len, cache Put/Len), the
  prose-derivation mentions are retired, and the rule row reads
  RATIFIED. The two contract spellings are upstream vocabulary — the
  registry lives in eidos (`plugins/annotator/shape/contracts`), which
  testkit deliberately does not curate (ADR-0004) — filed as eidos#39
  (pool `stats` role) and eidos#40 (cursor `open` producer arm, riding
  the returned-handle resolver scope eidos#29 delivered). Both registrations
  landed in eidos main (the cursor arm via the Param.Role scoping the
  #40 thread converged on); the three testkit modules ride the
  pseudo-version, the conformance fixtures spell the canon (pool
  stats=Stats with a struct-shaped accounting role; the cursor-open
  producer fixture), the corpus regenerated clean (drift gate: 673
  outputs), and the pool fixture's concrete-type Stats() supplied
  wiring is retired in favor of the role. Deliberately deferred: the
  cursor-open fixture carries no //testkit:model — the
  produced-secondary lowering is the next-generation model plugin's,
  proven in gen/scantest. The gate's law-manifest census also caught
  the engine's new Ops field, now manifested KindDefault with the
  emitter rule recorded beside it.
- Single-domain evidence: clock/timeaware, the sim tier, and Porcupine
  still rest on kv alone.
- ~~The `x/` vs `own/` family word~~ — DECIDED and applied: the family
  is `own/` (`own/no-replay`, `own/hand-written`), renamed while zero
  consumer lock files existed. The mechanics were one constant
  (`FamilyHand`) plus literal mentions — the one-home rule's own proof.
- Eidos emission against this corpus, and the pilot measurement.

## The stdlib promotion round — closed

The DRY/SOLID pass over all five packs asked: which idioms does the
generator re-emit per package that a library should hold once? Three
tiers shipped, and the full corpus (bustest, cachetest, pooltest,
scantest, kvtest) migrated onto them — every gate green under `-race`,
lint at zero, every proof still able to fail.

- **Tier 1 — `gen/legs`** (new package): the sanctioned suite↔engine
  bridge. `Reference`, `Law`, `Differential`, `Blend`, `NoteVacuity`,
  and the `CompatV1` witness each generated model file pins. This ends
  the five-copies-drift problem the doctrine ("suite imports testing
  and clock and nothing else") used to force onto the emitter.
- **Tier 2 — `gen/suite` + `prove`**: `ProvenCheck`, `Inductions` +
  `LowerInductions`, `ExcuseSet`, `Must`, `DropHinter`,
  `Bundle.ConfigOnce`, `prove.One`, `Defect.Reasoned`.
- **Tier 3 — `stub.Group`** (ROOT module): the fan-out and
  cleanup-verify tail every generated double restated. This one is a
  published-surface addition and rides the release train.

Deliberately NOT promoted: `CheckFresh` (zero net line savings), the
tb-less `one` adapters (they wrap package-owned builders), and the
typed Recover downcast (names the harness's own type parameter).

Emission consequence recorded in `derivation-rules.md` ("the emission
diet"): the emitter targets these surfaces instead of expanding their
bodies.

## Correction — negative controls reclassified consumer-authored

The packs' negative controls (doubleBus, mruCache, fifoPool,
delayedLease, boundedLazyLog, and kv's four) sat in `*_proofs.gen_test.go`
files, implicitly claiming generated provenance. The audit question "how
would a generator produce an MRU cache?" has no honest answer: a control
is a legal-but-different implementation, and different-but-legal is
invented, not derived. Contrast the derived reference, which survives
the same audit — role lowering plus the bounded-mixin asymmetry rule
plus signature-derived degradation, each recorded.

Applied: all nine controls moved to the consumer package
(`example/*_control_test.go`) with consumer provenance headers, riding
only exported surface — which itself surfaced one seam finding: a
corpus-seeded interface's control cannot use `prove.Green` from outside
(the corpus is unexported, by design), so Catalog's control rides
`RunCatalog` with a seed harness, the same green assertion through the
run's own wiring. The consumer-stated constants (`limit=2`,
`timeout=100ms`) are restated in the control files because a consumer
knows their own directives. Ownership split and the two PROPOSED
emitter duties (scaffolded control slot, missing-control report note)
recorded in `derivation-rules.md`.

## The second optimization round — closed

Follow-through on the "optimize further" audit: the two levers worth
pulling existed mostly as unused engine surface.

- **Derived references now ride `ref` primitives.** The audit's
  credibility argument made concrete: cachetest (`KeyedStore`),
  scantest (`MonotonicLog` + `BoundedCursor` — the cursor satisfies
  `scan.Cursor` as-is, so the hand-rolled refCursor is gone), pooltest
  (`BalancedPool`, `LeaseTracker`), kvtest journal (`MonotonicLog`).
  What remains emitted is signature adaptation only. bustest keeps its
  hand-rolled fan-out: no channel-shaped topic-broker primitive exists,
  recorded as the second-consumer rule rather than filled speculatively.
  Two semantic checks cleared before migrating: conn identity is never
  compared (the FIFO negative control is the proof), so BalancedPool's
  recycle-reuse carries no claim; LeaseTracker's nil free-release error
  reproduces release-of-unheld-is-a-no-op.
- **Delegation closures became method expressions.** Nine `*Of`
  constructors added engine-side (136 lines incl. docs), 19 corpus
  sites migrated; the three closures that do real work stay.
- **`suite.LowerRecover`** closes the harness-lowering tail.

Net: model files 1789 → 1730 lines with the wrong-method defect class
removed and every derived ref visibly primitive-plus-adaptation. The
emission diet is now declared closed per the audit's boundary: the
remaining mass is the typed surface itself.

## The remediation sweep — closed

"Fix all, not only P0": every valid finding from both audits is fixed,
declined-with-reason, or recorded as an emitter duty — the full table is
`remediation-ledger.md`. The structural class got the deepest cut: the
lifecycle law now probes every stamped method (with a planted
partial-outlive defect that passes the old single-probe law and reds the
new), nilcontext asserts the error its claim records (with an
accepts-nil defect for the new arm), partial vacuity is reported
per-leg, and the lock's v2 binds column makes probe-set narrowing diff.
Skew witnesses now cover all four modules. The seed headers state the
truth. One audit finding was reclassified during the fix: adoption.md
documents the incumbent toolchain accurately — the "stale sim status"
reading came from the audit premise (generators built) colliding with
the tree it read.
