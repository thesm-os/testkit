# Suite

A conformance harness is what an implementation is held to. The `suite`
generator reads an annotated interface and emits everything the interface's
shape and classifications imply about how an implementation must behave —
derived rather than configured, with a typed extension point for what
derivation cannot reach. Passing the harness is the claim that one
implementation can stand in for another.

The unit is the method, not the classification. A method's checks come from
four sources, and only the third involves a directive:

1. **The signature.** A context parameter earns cancellation, deadline and
   nil-context checks; an error return earns a zero-value-on-error check;
   every method earns a smoke call. This is most of the volume and needs no
   annotation.
2. **The detector.** The shape stamp adds what is specific to it — a
   `reader` misses, a `lookup`'s `ok == false` comes with zero values.
3. **Each mixin and contract the method carries** adds the direct form of
   its declared law.
4. **The extension point**, typed, whether or not anything filled it.

The interface as a whole earns cross-method checks a fixed sequence can
witness — read-after-write is the writer followed by the reader. Law-backed
classifications belong to the model tier, not here; see
[which tier owns what](#which-classifications-this-tier-asserts).

## The directive

```go
//testkit:suite
type Mixed interface { ... }
```

No keys, no positional argument, and the negated form is denied — a suite
exists where one is declared, so deleting the line is the suppression.
Everything else the generator needs it reads: the shape stamps for what to
check, `//testkit:out` and `pkg=` for where the file lands, and the source
signature for the rest.

## Where the output goes

| Tag | Suffix | Contents |
|---|---|---|
| *(primary)* | `_suite.gen.go` | The harness: entry point, per-method checks and check types, fixture, options. |
| `test` | `_suite.gen_test.go` | The self-check: every generated check driven against a stand-in built to break it. |

Both route with `//testkit:out`, usually once at package scope. The corpus
convention pairs the suite with the same interface's `//testkit:stub` in one
`<pkg>test` package.

## Proving a check can fail

A check that cannot fail is worse than none, and no gate reports one — a vacuous
check passes. The companion output is that gate.

For every generated check it configures the double into an implementation that
violates exactly that check, drives the check against it, and asserts the
rejection names the right reason:

```go
func TestAssertContractPutRefusesADuplicateCanFail(t *testing.T) {
    t.Parallel()

    fixture := ifabsenttest.DefaultContractFixture()
    subject := ifabsenttest.NewContractStub(t,
        ifabsenttest.WithContractPut(func(context.Context, ifabsent.Value) error {
            return nil // accepts the duplicate, which is the violation
        }))

    got := testkit.Rejects(t, "a store that accepts every write",
        func(tb testing.TB) {
            ifabsenttest.AssertContractPutRefusesADuplicate(tb, subject, fixture.VOther)
        })
    testkit.Assert(t, got).Contains("must be refused",
        "and rejects it for the reason the check is about")
}
```

The message is asserted as well as the rejection. A stand-in that failed for
some unrelated reason would satisfy a guard reading only the fact, which is the
vacuity the file exists to catch, one level up.

`go test -run CanFail` selects the whole family, so the proof can be a CI stage
of its own. [`testkit.Rejects`](../primitives/assertions.md) is the same
primitive for a check you write yourself.

Two shapes get no companion, and the harness header says which: an interface
declaring no `//testkit:stub` has no stand-in to configure, and a generic one
whose constraints derive no witnesses and whose stub pins none has no
concrete instantiation to prove against — a `Test` function cannot carry
type arguments. A witnessed generic interface *is* proved: the guards
instantiate every entry point at the double's own witnesses — pinned by
`//testkit:stub witness=` or derived from an open constraint — so the two
companions run at one instantiation.

## What it generates

The worked example throughout is the conformance fixture
`conformance/corpus/iface/mixin/validates`:

```go
type Payload struct{ Key, Body string }

//testkit:out validatestest/ pkg=validatestest
//testkit:stub
//testkit:suite
type Mixed interface {
    //testkit:mixin validates fn=Validate
    Store(ctx context.Context, v Payload) error

    Validate(v Payload) error

    Read(ctx context.Context, key string) (Payload, error)
}
```

One directive beyond `suite` itself, and the generated harness carries ten
checks across the three methods — the signature family supplies nine of
them.

### The check types

```go
type MixedStoreCheck func(tb testing.TB, subject validates.Mixed, v validates.Payload)
type MixedReadCheck  func(tb testing.TB, subject validates.Mixed, key string)
```

One named type per method, over the failure-recording interface and the
method's own concrete parameters. Every generated check is a value of it,
and so is a consumer's — which is what lets them compose, reorder, and run
standalone. `testing.TB` rather than `*testing.T` is what makes a check
drivable by a stand-in such as [`testkit.FailableTB`](../primitives/failure.md).

### The exported checks

```go
func AssertMixedStoreSmoke(tb testing.TB, subject validates.Mixed, v validates.Payload)
func AssertMixedStoreCancels(tb testing.TB, subject validates.Mixed, v validates.Payload)
func AssertMixedStoreHonoursDeadline(tb testing.TB, subject validates.Mixed, v validates.Payload)
func AssertMixedStoreToleratesNilContext(tb testing.TB, subject validates.Mixed, v validates.Payload)
func AssertMixedReadZeroOnError(tb testing.TB, subject validates.Mixed, key string)
// ...
```

Exported so a consumer can run one in isolation, and so a self-check can
drive it against a violating stand-in and prove it fails. Generated beside
the subject at concrete types, there are no type parameters to erase
([ADR-0012](../../adr/0012-generate-per-shape-helpers-into-the-consumer.md)).
Subtest names carry the classification they assert
([ADR-0015](../../adr/0015-subtest-names-carry-the-classification.md)).

### The fixture

```go
type MixedFixture struct {
    V        validates.Payload
    VOther   validates.Payload
    Key      string
    KeyOther string
}

func DefaultMixedFixture() MixedFixture
```

One field per parameter, derived from the parameter's name and type, with a
paired alternate: a check comparing a result against a single input passes
whenever the subject happened to be seeded with it, and a miss check whose
key happens to hit asserts nothing. Struct parameters compose per field. A
type the source declares a `<Type>Defaults()` companion for uses it for the
sample half.

A field whose type admits no literal — a func, a channel, a type the run
never read — is declared and left at its zero value, and every check that
needed it is **dropped rather than emitted against a value nobody could
write**. Supply one through `MixedWithFixture` to restore them.

### The seed

The interface seeds itself: where it declares a writer (`writer`,
`compositewriter` or `multiargwriter`), each fresh subject is populated
through it with fixture values before the checks run — which is also what
makes read-after-write free. A writer-less interface has no derivable seed;
`MixedSeed` supplies one, and it returns an error rather than swallowing
one, because a seed that failed quietly leaves every later check asserting
against an empty subject.

### The entry point

```go
func AssertMixedContract(t *testing.T, opts ...MixedOption)
```

Runs every check against every declared subject, each subtest against a
fresh subject. The generated docblock is a report: how many checks, across
how many methods, which options extend or drop them, and whether the double
run applies.

## Options

```go
validatestest.AssertMixedContract(t,
    validatestest.MixedSubject("in-memory", newInMemory),
    validatestest.MixedSubject("postgres", newPostgres),
)
```

| Option | Effect |
|---|---|
| `MixedSubject(name, factory)` | Declare an implementation. The one option a consumer always writes — which implementations exist is the single fact derivation cannot recover. |
| `MixedWithFixture(f)` | Replace any subset of the derived inputs; unset fields keep their derived values. |
| `MixedSeed(fn)` | Supply the seed a writer-less interface cannot derive. |
| `MixedOn<Method>(name, fn)` | Add a named check of the method's check type. It reports beside the generated ones. |
| `MixedWithout(paths...)` | Drop generated checks by path — `"Store/reports a cancelled context"` — without abandoning the rest. |
| `MixedWithoutDouble()` | Decline the second, double-wrapped run. |

The acceptance measure this generator is held to: a suite that needs only
`Subject` is the design working — every further option is either a fact the
source cannot state or a derivation not yet done.

## What a harness says it does not check

The header names every classification the interface declares that the file does
not assert, in two lists. What a consumer can write themselves goes in one, with
the extension point to write it with; what needs a reference implementation to
compare against goes in the other, because telling somebody to hand-write it
would be telling them to state a property their run cannot reach.

An absent check is a stated boundary rather than a suspected defect. Without it
a harness reads as a list of checks with its declared classifications invisible,
and nothing distinguishes one covered elsewhere from one nobody handled.

## Integration-only methods

A method carrying `//testkit:mixin integrationonly` reaches something outside
the process, so its whole check group sits behind `TESTKIT_INTEGRATION`. Unset
is a skip rather than a pass: a check that succeeded because its dependency was
absent reports coverage it did not earn.

## The double run

Where the interface also carries `//testkit:stub`, every subject runs twice:
once plain, once wrapped in the generated double in delegate mode. Anything
the wrapper fails that the subject passes is the double lying — which is
what makes a generated double trustworthy. The double is read off the stub
generator's queued output, never re-derived from its directive. An
interface without `//testkit:stub` generates neither the run nor the
option.

## Which classifications this tier asserts

[ADR-0018](../../adr/0018-one-tier-owns-each-classification.md) assigns each
classification to exactly one tier by rule: **the suite tier implements no
property `engine/model/law` already carries.** Where a law exists the
classification is the model tier's; what stays here is the
signature-derived family, the shapes the law catalogue does not reach, and
the classifications whose direct form is a fixed call sequence. The full
per-classification tables live in
[RFC-0002](../../rfc/0002-the-suite-generator.md); the generated header
records which tier owns each check the file carries.

One detector claim lives in a success value rather than in an absence: an
`answeringwriter` beside a keyed reader of the same state earns
`Assert<Iface><Method>AnswersWhatItStored` — the write's answer, read back
under the fixture's own key, must be what the read observes. Nothing else
can state it: the derived seed discards the answer, and the model tier's
twin floor compares two subjects wearing the same lie. A lone answering
writer derives nothing and stays listed as unchecked, which is the truth
of it.

## Generic interfaces

The harness is generic where the interface is — the entry point and check
types carry the interface's type parameters, and the consumer's factory
fixes them at the call site. The self-check half, which must instantiate at
concrete witnesses, lands with the reserved `test` output.

## Layout conventions

| File | Owner | Contents |
|---|---|---|
| `iface.go` | Developer | The interface, its directives, the package-scope routing |
| `<pkg>test/iface_suite.gen.go` | Generator | The harness. Do not edit. |
| `<pkg>test/iface_stub.gen.go` | Generator | The double the second run wraps subjects in. |
| `<pkg>test/inmemory_test.go` | Developer | The wiring: subjects, extensions, drops. |

## Planned additions

[RFC-0003](../../rfc/0003-the-projection-consumers.md) commissions two
additions to this generator's surface: an init-time **subject registry**
(`RegisterMixedSubject`) so the bench, fuzz and model consumers can generate
their own entry points, and a generic **extension seam** those consumers
register runners through. Neither is emitted yet.

## See also

- [Stub](stub.md) — the double the second run wraps subjects in.
- [Bench](bench.md), [Fuzz](fuzz.md), [Model](model.md) — the consumers of
  this generator's projection, designed in RFC-0003.
- [RFC-0002](../../rfc/0002-the-suite-generator.md) — the design record and
  the full classification-to-tier tables.
