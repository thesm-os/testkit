# Remediation ledger — the two audits, deduplicated

One row per finding from the distinguished-engineer audit (B/S/3.x series)
and the 30-gap analysis (G series), after source verification. Status is
the truth; the gap documents stay as written. Items marked DECLINED carry
the reason — a declined finding resurfacing without new evidence should
be closed against this table.

## The structural class (one root cause, three instances)

| Item | Finding | Status |
|---|---|---|
| B1 | lifecycle claim says "every method", law probes one | FIX: `law.LifecycleAfterCloseSentinel` gains `Ops` (probe per stamped method, error names the method); corpus binds the stamped set; kv proof upgraded to a partial-outlive defect that passes the old law and reds the new |
| G-17 | nilcontext claim says "returns an error", code asserts only no-panic | FIX: `ToleratesNilContext` asserts a non-nil error |
| G-09 | a bundle with one engaged law masks four vacuous ones | FIX: `legs.NoteVacuity` notes partial vacuity; report renders it |
| root cause | nothing ties claim text to assertion shape | FIX (partial): lock v2 gains a `binds` column naming the law IDs/probes a check binds, so a probe-set change diffs; full fix (claims derived from probe data) is emitter-phase |

## Trust and skew (B2)

| Item | Finding | Status |
|---|---|---|
| B2a | assertion-body libraries unwitnessed | FIX: `model.CompatV1` (engine module witness, referenced per generated model file), `stub.CompatV1` (root module witness, referenced per stub file) — module granularity, since packages inside a module version together |
| B2b | lock cannot see a body change | FIX: lock v2 `binds` column (above); body-hash fingerprinting is emitter-phase, recorded in derivation-rules |

## Seed truth and CI economics (B3)

| Item | Finding | Status |
|---|---|---|
| B3a | `Draws: seed 0x…` headers read by no code | FIX: headers now state the truth (run-time seed; TESTKIT_RAPID_SEED pins) |
| G-23 | report JSON records seed "0" on unseeded runs | FIX: renders "randomized"; the replay seed's one home is the failure log, stated in the docblock. Capturing rapid's internal seed needs upstream rapid API — none exists in v1.3 |
| B3b | no presubmit/postsubmit split | FIX (docs): CI recipe documents TESTKIT_RAPID_CHECKS presubmit budget + postsubmit -run gating for proofs/controls. Library adds no skip mechanism — the no-skip doctrine holds |
| lease 100ms | compiled-in poll budget, no knob | FIX: TESTKIT_TIMEOUT_SCALE multiplier in the lease law |
| porcupine timeout | undecided-within-budget reads as nonlinearizable | FIX: distinct actionable message + TESTKIT_LINEARIZE_TIMEOUT knob; still red — undecided must not pass |
| free-immediately assertion | zero-tolerance negative flagged as flake risk for async acquires | DECLINED to weaken: an acquire that returns before the grant is observable is the defect, not the flake; docblock states the position |

## Artifact integrity (B4)

| Item | Finding | Status |
|---|---|---|
| G-04 | WriteArtifact fails on missing report dir | FIX: MkdirAll |
| B4a | -count overwrites the report silently | FIX: existing-file suffix (-2, -3…) |
| G-30 | Class field unvalidated for tabs | FIX |
| B4b | absence-as-signal has no manifest | DEFERRED to emitter (rules row): the expected-artifact manifest needs whole-package knowledge only the generator has |
| G-22 | report lacks module/package fields for fleet scraping | FIX: additive ModulePath/PackagePath fields |
| G-13 | Bazel sandbox paths | FIX: TESTKIT_TESTDATA_DIR override for lock + scrub |
| G-03/S7 | scrub is CWD-relative RemoveAll of a dir a consumer may own | FIX: scoped to `*.fail` files + the override above |

## ID space and naming

| Item | Finding | Status |
|---|---|---|
| G-24 | adding interface #2 renames every family ID | FIX: qualification is uniform — every family-scoped ID carries the interface segment regardless of interface count. Reopens the A18 ruling; locks regenerate once, now, while zero consumers hold them |
| S6 | first interface gets unqualified unexported names | FIX: kvtest unexported types take the store prefix |
| S11 | generated banners not machine-uniform | FIX: one header shape |

## Consumer seat

| Item | Finding | Status |
|---|---|---|
| 3.1 | controls force a second harness vocabulary | FIX: harnesses export `Subject()`; controls rewritten onto harness vocabulary |
| 3.2 | consumer rows have no specificity path | FIX: veneer `SuiteWith(cfg, rows)` exposes generated+consumer checks to `prove.Green` |
| 3.3 | control restates the declared limit | FIX: veneer `DeclaredLimit()` — the directive's one exported home |
| S3 | fix message names nonexistent `Provides` field; comment claims uniform emission | FIX: `Provide` emitted on every harness (an open capability is consumer-declared, so need-driven emission cannot predict it); message and comment corrected |
| S4 | `Prove*` has two arities | FIX: variadic opts mirroring `Run*` — proofs still bind what the run binds |
| S8 | sealed Defect interfaces documented open | FIX: docs state the seal and the reason |
| S10 | Suite-as-data path panics where the run path reports | FIX: `TrySuite` returns the error; `Suite` keeps the panic for the assert-shaped call sites, docblock says which to use when |
| 3.9 | anonymous harness error identifies nothing | FIX: error carries the option's position |
| 3.10 | Excuse forces a suite import for the slice type | DECLINED: `Checks.<Family>.All()` already returns the slice where excusing whole families is idiomatic; a bare alias adds a name for a `[]suite.ID` literal — documented instead |
| 3.11 | NoteTier omission is silent | FIX: report renders "tier unstated" instead of blank |
| G-14 | typed downcast rejects decorator-wrapped subjects | FIX: `Unwrap() T` support in LowerInductions/LowerRecover |
| G-15 | no per-subject setup/teardown for heavy fixtures | FIX: `Subject.Setup`/`Teardown`, run once per subject around its legs |
| G-16 | `RunCtx` body variant | DECLINED: `tb.Context()` is the idiom; a sixth body arity multiplies every row's exclusivity rule for zero new capability |
| G-29 | miss checks vs pre-populated stores | ALREADY ADDRESSED: `KeyPool` config steers the keyspace; documented in adoption notes |

## Runtime hardening

| Item | Finding | Status |
|---|---|---|
| G-05 | bidirectional errors.Is + map-order fallback in Inducer | FIX: unidirectional (`errors.Is(key, sentinel)` asks "does this registered trigger cover the asked sentinel"), deterministic sorted fallback |
| G-06 | clone shares Needs maps | FIX: deep copy |
| G-07 | Provides values unvalidated | FIX: a `Caps` value may be a `func(any) error` validator, consulted by CanRun |
| G-02 | child-goroutine panic crashes the process | FIX: `FailableTB.Go` managed spawner; exposure otherwise equals any Go test |
| G-01 | proof can block on a deadlocked defect | DECLINED: `go test -timeout` owns process deadlines and dumps all stacks; a library wall-clock timer adds the flake class B3 removes |
| G-11 | rapid env applied once, process-global | DECLINED: iteration budget is process policy by design; per-suite budgets would make two suites in one binary disagree about what a PR ran |
| G-12 | scheduler nondeterminism under pinned seeds | FIX (docs): stated where seeds are documented |
| G-19 | -run escaping for slashed IDs | FIX (docs) |

## Wrong or stale in the 30-gap document — closed without action

| Item | Why |
|---|---|
| G-08 | `ctx.Err()` derives from wall time in the context package; an injected TestClock never sees the deadline. The described subject (comparing `ctx.Deadline()` to its own clock) is itself broken |
| G-10 | `t.Parallel()` is throttled by `-parallel` per binary and `-p` across packages; the EMFILE scenario cannot occur |
| G-27 | MultiReader/BatchReader/MultiArgWriter/MultiAggregator/PureVar exist in `engine/model/action` |
| G-18 | eidos#35/#36 closed the mistyped-param and mistyped-name cases; prescreen rejects undeclared directives. Residual (`//testkti:` prefix typos) recorded as a future `testkit vet` rule |
| G-25 | mechanically true (`sub.reference = oracle.New`) but clocked checks compare against no reference by design; recorded as an emitter-phase note should a clocked differential leg ever exist |

## Emitter-phase rows (recorded in derivation-rules, not built here)

G-20 nested builders · G-21 cancel-inflight family · G-26 generic
interfaces · G-28 per-method zero-on-error opt-out · B4b expected-artifact
manifest · claim-from-probe derivation (the full structural fix) ·
G-25 clocked-oracle bundle.

## The suite generator's DRY/SOLID audit

Run when the plugin passed 4,400 production lines and the growth stopped
looking like the trade the design intended. Findings and what each cost.

### Fixed

| Finding | Was | Now |
|---|---|---|
| `segAccessor` | split on `-` and exported-name'd each part | `golang.ExportedName` — the platform's splitter already treats `-` as a separator |
| differential sequence word | `strings.ToLower(a) + "-" + strings.ToLower(b)` | `naming.Kebab`; the hand form spelled a two-word method `putall-nextpage` |
| `companionFor` | a token-for-token copy of builder's lookup | `generator/source.Companion`, taking the suffix as an argument so the convention stays with the directive that spells it |
| `resolveMethods` | copied verbatim between suite and stub | `generator/source.MethodSet`, with the one clause that differs passed as a `Consequence` |
| `lawBackedStamps` | rebuilt an index from `tiers.Rules()` per Derive call | `tiers.LawsFor`, which answers the question and says so in its docblock |
| `mixinParamsOf` | hand-enumerated twelve `(mixin, param)` pairs | the pairs come from the live registry through `mixinParamKeys` |
| `LawParams` | re-read the stamps off `m.Source.Meta()` | reads `Method.MixinParam`, the projection that already holds them |
| the Argued→Proven flip | four hand-written copies | `proveOrArgue`, settling both fields together as the parity gate requires |
| `seedOf` | recomputed `fixtureArgs` for a method already carrying `ArgFields` | reads what the method carries |

Three bugs surfaced under those, each fixed with its finding:
`borrowedCall` silently dropped an argument when its two lists
disagreed, emitting a call shorter than the method takes;
`Inventory.Token` was documented as the ID qualifier while holding the
Go one; and the kebab join above. Deleted with them: a 31-line docblock
for `partnerArgs`, a function this package does not have, and two
references to `Method.MixinPartner`, a method it does not have either.

`generator/source` is where both landed, and the first attempt got it
wrong twice: a package named for one function, justified by a
constraint that does not exist — plugins import each other freely here,
fault reads stub and model reads suite — and shipped without a test.
The package earns its name by holding what no plugin owns: questions
about the source tree a generator has to ask. Both functions' refusal
paths are now covered, which neither copy ever was.

### Declined, with the reason

**The fixture's whole-value sample, derived for composed fields that
never render it.** True as stated and not separable: `sampleFor`
answers the sample and the alternate together, and a composed field
still renders the alternate — the miss value a reader check needs.
Splitting the pair to save one struct resolve at build time would buy
microseconds for a seam where two values stop being derived together.

### Deferred, with an owner

**~800 production lines derive artefacts no template consumes.** The
templates reference four projection paths; `poolsOf`, `HarnessOf` and
`LockLines` have no production caller at all. This is the derive layer
built ahead of its emission rather than dead code, and the honest
resolution is to finish the bodies and let them prove which projections
are real. Anything still unconsumed when the emission lands is a
deletion list written from evidence.

**The SRP splits.** `suite.go` carries seven jobs — plugin declaration,
emit types, pure signature queries over `golang.Sig`, an eidos
vocabulary alias table, stamp reading, method-set resolution and two
derivation rules. `fixture.go` carries five, of which seed derivation is
a separate concept from the fixture. `derive_laws.go` hides the
cross-plugin stamp API `generator/model` imports. Each is a move rather
than a rewrite, and each wants doing when the file it lands in is not
also being rewritten by the emission arc.

## The depguard rule that enforced the opposite of its comment

The `generator` rule read "vocabulary only: engine/model and the
property dependencies stay denied", and did neither. In
`list-mode: strict` a deeper allow entry shadows a broader one, so
listing `go.thesmos.sh/testkit/engine/suite` beside
`go.thesmos.sh/testkit` refused **every** intra-testkit import in the
module — 89 findings, every one a false positive — while the broad
entry went on admitting `engine/model`, the one import the comment
said was denied.

Fixed by saying it the way depguard says it: the broad allow stands
alone, and `engine/model` is a `deny` with the reason beside it. The
rule now refuses what it always claimed to and permits what the module
has always legitimately done.

## The claim the ledger recorded wrong

Under "transitional", the design doc said the corpus's 126 undefined
`Assert*` references would clear when the rewrite's rows landed. The
rows landed and they did not, because they could not: the orphaned
companions call the incumbent's exported `Assert<Iface><Method><Seg>`,
the rewrite emits the packs' unexported `<token>Assert<Method><Word>`,
and nothing had regenerated a companion since the sweep deleted its
template. Two different names and no writer between them.

What closed it was writing the companion the plugin's second output has
declared since the rewrite began. Recorded here because the entry read
as self-healing for several sessions, and an item nobody owns because
everybody believes it is about to resolve itself is worse than one
nobody has started.

## The falsifiability tier, and where its evidence lives

`prove.All` holds a suite's claims and its proofs to each other in both
directions: Proven with no defect fails, a defect naming no check
fails, and a defect the check tolerated fails. That makes the stamp on
an emitted row a load-bearing claim rather than a label, which forced
two rules.

A row is stamped Proven only where the companion can SPELL its defect,
not merely where a deriver named one — `suite.ArguedCheck` exists for
the difference. Emitting Proven and letting the parity gate find the
hole would report a generator gap against generated code, which is the
wrong file and the wrong reader.

A proof quotes the red it expects by CONSTANT, not by text.
`suite.RedPanicked` and its three siblings are declared where the
message is written and the message is built from them, so rewording a
primitive breaks the generated file's compile instead of quietly
weakening every proof that quoted it. The families whose failure prose
lives in a body template have no constant to name and get no
`Reasoned` — a weaker proof, said out loud in the generated comment
rather than left looking identical to a strong one.
