# Error Prefix

Verifies that every `errors.New` call uses the correct
package name as its prefix. Catches sentinel errors that
don't follow the project's error-handling convention.

## Command

```
testkit validate error-prefix
```

## What it checks

For every `.go` file (excluding `*_test.go`):

1. Extract the `package <name>` declaration
2. Find every `errors.New("...")` call
3. Verify the string literal starts with `"<pkg>: "`

## Examples

```go
// PASS — prefix matches package name
package store

var ErrNotFound = errors.New("store: item not found")

// FAIL — missing prefix
var ErrNotFound = errors.New("item not found")

// FAIL — wrong prefix
var ErrNotFound = errors.New("cache: item not found")  // in package store
```

## Failure output

```
error-prefix: FAIL

  store/errors.go:12: errors.New("item not found")
    expected prefix: "store: "

  store/errors.go:15: errors.New("cache: item not found")
    expected prefix: "store: "
```

## Why

Consistent error prefixes enable log grep-ability — when
a production error surfaces as `"store: item not found"`,
the operator immediately knows which package produced it.
Without enforcement, prefixes drift: some errors have the
prefix, some don't, some have the wrong one.

The sentinel generator's prefix test catches this at test
time for packages with generated sentinel tests. This
validator catches it at CI time for all packages,
including those that haven't adopted the sentinel
generator yet.

## Configuration

```yaml
# .testkit.yml
validators:
  error_prefix:
    enabled: true
    # Directories to scan (default: all Go source dirs).
    dirs:
      - "."
    # Exclude test files (default: true).
    exclude_tests: true
```
