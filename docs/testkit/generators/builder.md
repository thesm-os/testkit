# Builder

Generates a fluent test-fixture builder for Go structs. Each generation emits two files: the impl (`builder.gen.go`) containing the builders, and a companion test file (`builder_test.gen.go`) verifying the contract every builder honors. Emits one `With*` setter per exported field, plus shape-specific extras: `Append*` for slices, `WithDataString` for `[]byte`, `With<Field>Entry` for maps. Always emits `Mutate`, `Clone`, and `Build`. Generic types are preserved with type parameters intact.

## Directive

```go
//go:generate testkit builder -o storetest/item_builder.gen.go Item

// Multiple types in one file:
//go:generate testkit builder -o storetest/builders.gen.go Item User Order
```

## Default output

`<package>test/<subject>_builder.gen.go`.

## Constructors

The generator emits two constructors per type:

| Constructor | Seed |
|-------------|------|
| `New<Type>()` | Either zero values, or `<Type>Defaults()` from the test package, or directive defaults — see "Defaults" below |
| `New<Type>From(v)` | Caller-supplied value |

## What is generated

For

```go
type Item struct {
    ID       string
    Name     string
    Count    int
    Active   bool
    Tags     []string
    Data     []byte
    Metadata map[string]string
    hidden   int  // unexported — no setter emitted
}
```

the generator produces:

### Builder skeleton

```go
type ItemBuilder struct {
    v basic.Item
}

func NewItem() *ItemBuilder              // zero-value seed (or directive defaults)
func NewItemFrom(v basic.Item) *ItemBuilder

func (b *ItemBuilder) Build() basic.Item
func (b *ItemBuilder) Mutate(fn func(*basic.Item)) *ItemBuilder
func (b *ItemBuilder) Clone() *ItemBuilder
```

### Per-field setters

Setters are sorted alphabetically by field name.

**Scalar fields** — one `With<Field>` returning the receiver:

```go
func (b *ItemBuilder) WithActive(v bool) *ItemBuilder    { b.v.Active = v; return b }
func (b *ItemBuilder) WithCount(v int) *ItemBuilder      { b.v.Count = v; return b }
func (b *ItemBuilder) WithID(v string) *ItemBuilder      { b.v.ID = v; return b }
func (b *ItemBuilder) WithName(v string) *ItemBuilder    { b.v.Name = v; return b }
```

**Slice fields** — variadic `With*` (replaces) plus `Append*`:

```go
func (b *ItemBuilder) WithTags(v ...string) *ItemBuilder    { b.v.Tags = v; return b }
func (b *ItemBuilder) AppendTags(v ...string) *ItemBuilder  { b.v.Tags = append(b.v.Tags, v...); return b }
```

**`[]byte` fields** — both raw and string forms:

```go
func (b *ItemBuilder) WithData(v []byte) *ItemBuilder       { b.v.Data = v; return b }
func (b *ItemBuilder) WithDataString(s string) *ItemBuilder { b.v.Data = []byte(s); return b }
```

**Map fields** — replace and per-entry insert:

```go
func (b *ItemBuilder) WithMetadata(m map[string]string) *ItemBuilder
func (b *ItemBuilder) WithMetadataEntry(k string, v string) *ItemBuilder
```

`WithMetadataEntry` lazily initializes the map if it's nil.

**Unexported fields** are skipped — only exported fields produce setters.

### Mutate and Clone

```go
func (b *ItemBuilder) Mutate(fn func(*basic.Item)) *ItemBuilder { fn(&b.v); return b }
```

`Mutate` runs `fn` against the in-progress value. Used for one-off complex modifications that don't justify a setter.

```go
func (b *ItemBuilder) Clone() *ItemBuilder
```

`Clone` returns a deep copy. Slice and map fields are copied so mutations to the clone do not affect the original. Generated `Clone` walks every reference field — slices via `append([]T(nil), src...)`, maps via per-key copy. Pointer fields are shallow-copied (the generator does not assume value-type ownership of arbitrary pointer targets).

## Defaults

`New<Type>()` seeds with one of three sources, in priority order:

### 1. `<Type>Defaults()` companion function

If a function `<Type>Defaults() <Type>` exists in the **test** package (e.g. `RequestDefaults` in `package defaultstest`), the generator uses it as the seed:

```go
// defaultstest/defaults.go — hand-written
func RequestDefaults() defaults.Request {
    return defaults.Request{
        RunID: "test-run-id",
        Token: 42,
        Data:  []byte("test-data"),
    }
}
```

```go
// defaultstest/builders.gen.go — generated
func NewRequest() *RequestBuilder {
    return &RequestBuilder{v: RequestDefaults()}  // seed from companion
}
```

### 2. `//testkit:default` field directives

Per-field directives bake literal defaults into the generated constructor:

```go
type Config struct {
    Host    string //testkit:default "localhost"
    Port    int    //testkit:default 8080
    Verbose bool   //testkit:default true
    Name    string // no default — uses zero value
}
```

```go
// generated:
func NewConfig() *ConfigBuilder {
    return &ConfigBuilder{v: fielddefaults.Config{
        Host:    "localhost",
        Port:    8080,
        Verbose: true,
    }}
}
```

`//testkit:default` accepts Go literal values (strings, ints, bools, nil). Field directives are useful when the defaults are stable enough to live with the type definition; the companion function is better when defaults need test-package context.

### 3. Zero values

If neither a companion nor field directives are present, `New<Type>()` returns a builder over the zero value:

```go
func NewItem() *ItemBuilder { return &ItemBuilder{} }
```

## Generic types

Generic structs preserve their type parameters end-to-end:

```go
type Container[T any] struct {
    Label string
    Items []T
    Limit int
}

type Pair[A, B any] struct {
    First  A
    Second B
}
```

produces

```go
type ContainerBuilder[T any] struct{ v generics.Container[T] }
func NewContainer[T any]() *ContainerBuilder[T]
func NewContainerFrom[T any](v generics.Container[T]) *ContainerBuilder[T]
func (b *ContainerBuilder[T]) WithItems(v ...T) *ContainerBuilder[T]
func (b *ContainerBuilder[T]) AppendItems(v ...T) *ContainerBuilder[T]
// ...

type PairBuilder[A any, B any] struct{ v generics.Pair[A, B] }
// ...
```

Type-parameter constraints are propagated. The companion function for a generic type, if present, must also be generic.

## Usage

```go
// Vanilla — zero-seeded.
item := basictest.NewItem().
    WithID("item-1").
    WithName("Widget").
    AppendTags("alpha", "beta").
    Build()

// From defaults companion.
req := defaultstest.NewRequest().
    WithToken(99).  // override one field
    Build()

// From directive defaults.
cfg := fielddefaultstest.NewConfig().
    WithName("override").
    Build()

// Mutate for complex modifications.
item := basictest.NewItem().
    Mutate(func(i *basic.Item) {
        i.Metadata = make(map[string]string)
        i.Metadata["a"] = "1"
    }).
    Build()

// Clone for variants from a shared base.
base := basictest.NewItem().WithName("base")
a := base.Clone().WithID("a").Build()
b := base.Clone().WithID("b").Build()
```

## Why

Builders eliminate brittle inline struct construction. When a field is added, only `<Type>Defaults` (or the field's `//testkit:default`) needs updating — every test using the builder continues to compile and run. The generator produces deterministic output: setters sorted by name, deep-copy logic emitted only for fields that need it (slices, maps, byte slices), and generic type parameters preserved exactly as written.

## Layout Conventions

A typical domain object generates its builder into a `<pkg>test/` sub-package. This ensures the builder is accessible to integration tests across your codebase without leaking test-infrastructure dependencies into your production binary.

**What goes where:**

| File | Owner | Contents |
|------|-------|----------|
| `types.go` | Developer | The source file containing the struct definition. |
| `*_builder.gen.go` | Generator | The fluent builder implementation (DO NOT EDIT). |
| `*_builder.gen_test.go` | Generator | The self-verifying test suite for the builder itself (DO NOT EDIT). |
| `defaults.go` | Developer | Hand-written `func <Type>Defaults() <Type>` factories. |

## See also
- [Generators / Overview](README.md)
