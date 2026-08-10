# Builder

A test that constructs a value by composite literal restates every field each time. Add a field to the struct and every literal breaks at once; read the test back and you cannot tell which fields it actually cares about, because all of them are written down.

A builder inverts that. The constructor supplies the rest, and the test states only what it varies.

```go
cfg := configtest.NewConfig().WithPort(0).Build()
```

## The directive

```go
//testkit:builder
type Config struct { ... }
```

| Key | Value | Effect |
|---|---|---|
| `defaults` | A function name | Seeds the constructor from that function instead of from the `<Type>Defaults()` convention. |

The value names a function returning the struct, in either of two notations:

```go
//testkit:builder defaults=seed.ConfigDefaults
//testkit:builder defaults=example.com/seed.ConfigDefaults
```

The first resolves its qualifier against the imports of the file that declared the struct, which is what an author writes for a package the file already uses. The second carries its own import path, and exists because an import written solely to feed a directive is an unused import — which does not compile.

Repeating the directive takes the last `defaults=` written, matching `//testkit:default`.

Per-field, two more declarations apply:

```go
type Config struct {
    Host     string `` //testkit:default "localhost"
    Internal string `builder:"-"`
}
```

`//testkit:default <expression>` seeds one field. The directive is owned by the shared `defaults` package rather than by this generator, so a later generator can read the same stamp.

`builder:"-"` drops a field's setter entirely, for one a test should never set but which cannot be unexported.

## Where the output goes

| Tag | Suffix | Contents |
|---|---|---|
| *(primary)* | `_builder.gen.go` | The builder |
| `test` | `_builder.gen_test.go` | The companion checks |

Both land beside the source unless routed. Declare routing once at package scope — every builder in a package lands in the same companion package, so a per-struct directive is the same statement written N times and the Nth copy is the one that gets forgotten:

```go
//testkit:out configtest/ pkg=configtest
package config
```

## What it generates

Given:

```go
//testkit:out plaintest/ pkg=plaintest
package plain

//testkit:builder
type Item struct {
    ID    string
    Count int
    Tags  []string
}
```

the generator writes:

```go
type ItemBuilder struct{ v plain.Item }

func NewItem() *ItemBuilder                     { return &ItemBuilder{} }
func NewItemFrom(v plain.Item) *ItemBuilder     { return &ItemBuilder{v: v} }

func (b *ItemBuilder) WithID(v string) *ItemBuilder       { b.v.ID = v; return b }
func (b *ItemBuilder) WithCount(v int) *ItemBuilder       { b.v.Count = v; return b }
func (b *ItemBuilder) WithTags(v ...string) *ItemBuilder  { b.v.Tags = v; return b }
func (b *ItemBuilder) AppendTags(v ...string) *ItemBuilder

func (b *ItemBuilder) Mutate(fn func(*plain.Item)) *ItemBuilder
func (b *ItemBuilder) Clone() *ItemBuilder
func (b *ItemBuilder) Build() plain.Item
```

`New<T>From` exists for the case where a test varies one field of a value it already has. `Mutate` reaches the field a setter does not — an unexported one, or a shape the builder does not model.

## The setter follows the field's type

A setter that took `any` would defeat the point; a setter for `Weekday int` takes `Weekday`, or the declaration was pointless. What each field owes:

| Field type | Setters |
|---|---|
| Scalar, struct, interface, func, channel, `any`, `error` | `With<F>(v T)` |
| Slice | `With<F>(v ...T)` replacing, `Append<F>(v ...T)` adding |
| Fixed-length array | `With<F>(v [N]T)` only — an append has nowhere to go |
| `[]byte`, and named aliases of it | `With<F>(v []byte)` and `With<F>String(v string)`, so a caller with a string need not convert |
| Map | `With<F>(m)`, `With<F>Entry(k, v)`, `With<F>Entries(m)` |
| Set — `map[K]struct{}` | `With<F>(m)`, `With<F>Entry(k)`, `With<F>Entries(m)` |
| Pointer | `With<F>(v T)` taking the pointee and addressing it |

The set case is the one worth calling out: `With<F>Entry` takes **no value parameter**. Every value in a set is the same one, so a setter asking for it asks the caller for the one thing they cannot vary.

A channel gets a setter but is never constructed by the builder — capacity is a decision the caller owns.

## Defaults

Three sources, applied in order. Each produces a different constructor, so the one you get depends on what the source declares.

**1. A companion function** — a `<Type>Defaults()` beside the struct by convention, or whatever `defaults=` names. It is written by hand and its signature is checked, not only its name: one taking arguments or returning something else is a different function that happens to collide.

**2. Per-field `//testkit:default` directives**, which apply on top of whatever the companion seeded:

```go
//testkit:builder
type User struct {
    Username string //testkit:default "anonymous"
}
```

```go
func NewUser() *UserBuilder {
    v := domain.UserDefaults()
    v.Username = "anonymous"
    return &UserBuilder{v: v}
}
```

**3. The zero value**, when neither is declared:

```go
func NewItem() *ItemBuilder { return &ItemBuilder{} }
```

A default of `0`, `false` or `nil` is still a default. The directive is read, not inferred from whether the value differs from the zero — so `//testkit:default 0` produces an explicit assignment, and a generator that treated "zero" as "no directive" would be caught by it.

The literal is carried verbatim rather than parsed per kind. `"localhost"`, `8080`, `true` and `nil` all reach the generated file as themselves, which avoids a parser that would have to know every literal form Go admits and be told the field's type to tell `0` from `0.0`. What is checked is that the value cannot swallow the rest of the line: an unterminated string, raw string or rune fails at the directive rather than as a syntax error somewhere else.

A value that carries a dot is a symbol, not a literal, and resolves through the same two notations `defaults=` takes — so `//testkit:default time.Second` needs the file to import `time`, and `//testkit:default example.com/seed.Region` does not. A leading dot is a decimal point: `.5` is a number.

## Clone shares what a pointer means

`Clone` copies the slice, byte-slice and map fields, so appending through one clone is not visible through another.

Values held *inside* those are shared — a struct in a slice that owns a slice of its own — as are pointer fields. That is what a pointer means, and it is what stops a self-referential struct sending the copy into a loop it cannot leave.

```go
base := NewItem().WithTags("a")
other := base.Clone().AppendTags("b")

base.Build().Tags   // ["a"]
other.Build().Tags  // ["a", "b"]
```

## Generic structs

Type parameters are preserved, and a setter acquires them only if its field uses them:

```go
type Container[T any] struct {
    Value T
    Items []T
    Label string   // not parameterised
}
```

`WithValue` and `AppendItems` thread `T`; `WithLabel` does not. A map keyed by a comparable parameter threads both.

## Layout conventions

| File | Owner | Contents |
|---|---|---|
| `iface.go` | Developer | The struct, its directives, and the package-scope routing |
| `<pkg>test/iface_builder.gen.go` | Generator | The builder. Do not edit. |
| `<pkg>test/iface_builder.gen_test.go` | Generator | The checks for the builder itself. Do not edit. |
| `<pkg>test/defaults.go` | Developer | The `<Type>Defaults()` companion, when `defaults=companion` is declared |

## See also

- [Stub](stub.md) — for the doubles that return the values a builder constructs.
- [Assertions](../primitives/assertions.md) — the assertion functions the generated checks call.
