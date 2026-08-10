# Enum

A Go enum is a convention, not a language feature: a defined type and a block of typed constants. Nothing stops a conversion admitting a value outside the set, nothing notices when a variant is added without the switch arm it needs, and nothing relates the type's textual form to the values it was declared with.

Each of those is a one-line mistake that compiles. The `enum` generator writes the textual surface and the checks that hold the type to what its declaration says.

## The directive

```go
//testkit:enum
type Status int
```

| Key | Value | Effect |
|---|---|---|
| `methods` | `off` | Suppresses the generated methods, leaving only the checks. |

A variant overrides its own textual form with `//testkit:value`, for the case where the derived spelling clashes with a protocol's and the derivation cannot be taught about it:

```go
const (
    Draft Status = iota + 1
    Published
    Archived //testkit:value archived-v2
)
```

## The textual form follows the underlying type

This is the one rule worth reading before anything else, because getting it wrong is invisible.

**For a numeric enum the identifier is the only textual form the declaration carries.** `StatusActive` on `type Status int` renders as `Active` — the type name is context wherever the value appears, and repeating it is noise in every log line.

**For a string enum the value _is_ the textual form, and it is already written down.**

```go
//testkit:enum
type Region string

const (
    US Region = "us-east"
    EU Region = "eu-west"
    AP Region = "ap-south"
)
```

`US` renders as `us-east` and parses from it. Deriving `US` instead would discard the only thing the declaration said and break every value arriving from JSON, a database column or a query parameter — while still round-tripping against itself, so the failure would be invisible to a check that only tested the generated pair against itself.

## Where the output goes

Two files, and neither can be routed elsewhere.

| Tag | Suffix | Contents |
|---|---|---|
| _(primary)_ | `.enum_gen.go` | The methods and functions, in the type's own package |
| `test` | `.enum_test.go` | The checks, in the external test package |

The generated API declares methods on the enum's type, which Go permits only in that type's own package. An `//testkit:out` sending it away produces a file naming an undefined type. The checks travel with it and take the external test package the `_test.go` ending gives them.

## What it generates

For `type Status int` with variants `Draft`, `Published`, `Archived`:

```go
func (v Status) String() string
func (v Status) IsValid() bool
func (v Status) MarshalText() ([]byte, error)
func (v *Status) UnmarshalText(text []byte) error

func ParseStatus(s string) (Status, error)
func StatusValues() []Status

var ErrUnknownStatus = errors.New("status: unknown Status value")
```

Two details in the bodies are worth knowing:

**`String` renders an out-of-range value as itself**, `Status(7)`, rather than as any variant. A corrupt value stays visible in a log rather than passing for a good one.

**`StatusValues` returns a fresh slice per call.** A package-level one would let any caller reorder or overwrite the set every other caller reads.

### Text rather than JSON

`MarshalText` over `MarshalJSON`: `encoding/json` reaches for `TextMarshaler` on its own and so does YAML, and it is what makes the type legal as a **map key**, which a JSON marshaller alone does not.

### Anything the type already declares is skipped

An author who wrote their own `String` meant to keep it, and a second declaration is a redeclaration error in their own package against a file they did not write. So the generator stands down — and it stands down for a group, not just the one method:

| Already declared | Also skipped | Why |
|---|---|---|
| `String` | `Parse<T>`, `<T>Values`, `UnmarshalText` | `Parse` and `Values` are package-level rather than methods, so a same-named declaration is invisible to the enum node — a generator emitting them anyway would shadow whatever the author wrote. `UnmarshalText` is written in terms of `Parse`. |

What survives in that case is `IsValid` and `MarshalText`, which depend on neither. `methods=off` suppresses all of them at once and leaves the checks in place.

## What the checks assert

Three functions in `<basename>.enum_test.go`.

### Test\<T\>Variants

| Subtest | What a failure means |
|---|---|
| `declares exactly N variants` | A variant was added or removed without the test being regenerated. |
| `no two variants share a value` | Two names for one value make a `switch` unreachable in one arm. |
| `no two variants share a textual form` | The round trip cannot be a bijection; one of them will not survive it. |
| `a value outside the set is not valid` | `IsValid` admits a value a conversion produced. |
| `every declared variant is valid` | `IsValid` rejects something the declaration says exists. |
| `the zero value is not a declared variant` / `the zero value is <Variant>` | The subtest adapts to what the declaration chose. Starting at `iota + 1` makes an unset value invalid; starting at `iota` makes it mean the first variant. Both are legitimate, and the check pins whichever was written. |

### Test\<T\>Text

| Subtest | What a failure means |
|---|---|
| `every variant survives String and back` | `String` and `Parse` disagree for some variant. |
| `text naming no variant is refused` | `Parse` admits a string the declaration never mentioned. |
| `a value outside the set does not render as one inside it` | `String`'s fallback collides with a declared variant's rendering. |

### Test\<T\>Marshalling

| Subtest | What a failure means |
|---|---|
| `every variant survives a marshal round trip` | `MarshalText` and `UnmarshalText` disagree. |
| `text naming no variant is refused on unmarshal` | Unmarshalling admits an undeclared value, usually straight from a wire payload. |

## Declaring an enum the checks can hold

Two choices in the source make the difference between checks that mean something and checks that pass vacuously.

**Leave no gap in a numeric sequence.** A gap is indistinguishable from an out-of-range value in the fallback check, and the boundary the check probes is the one past the last declared variant.

**For a string enum, do not let one value be a prefix of another.** A prefix makes the fallback indistinguishable from a declared value under a naive match.

## Layout conventions

| File | Owner | Contents |
|---|---|---|
| `iface.go` | Developer | The type, its variants, and the directive |
| `iface.enum_gen.go` | Generator | `String`, `Parse`, `Values`, `IsValid`, the marshallers, the sentinel. Do not edit. |
| `iface.enum_test.go` | Generator | The checks. Do not edit. |

## See also

- [Sentinel](sentinel.md) — for the `ErrUnknown<T>` sentinel the parse failure returns, and the rest of a package's error contract.
- [Assertions](../primitives/assertions.md) — the assertion functions the generated checks call.
