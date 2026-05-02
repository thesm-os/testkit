# Proto-Sync

Verifies that proto definitions and their generated Go
counterparts are in sync. Catches silent drift between
`.proto` files and `.codec.go` files that only surfaces
as wire-format bugs at integration time.

## Command

```
testkit validate proto-sync
```

## What it checks

1. Every `.proto` message has a corresponding Go struct
   with matching field names (after case conversion)
2. Proto field numbers are stable (no renumbering between
   the committed `.codec.go` and the current `.proto`)
3. No proto field was removed without a `reserved`
   declaration
4. Every `(codec.field)` annotation matches the Go field
   name
5. Every `(codec.cast)` annotation references an existing
   Go type
6. Every proto field has a doc comment
7. Re-running `buf generate` produces no diff against the
   committed `.codec.go` files

## Failure output

```
proto-sync: FAIL

  api/proto/myapp/types/v1/model/snapshot.proto
    field 2 "entries": generated codec is stale
    re-run: make generate

  api/proto/myapp/types/v1/store/item.proto
    field 1 "items": undocumented — add a comment
```

## Why

Stale codecs silently break wire compatibility. A renamed
proto field that is not regenerated produces a codec that
reads the old wire layout — tests pass (they use the old
codec too) but production traffic on the new wire format
fails. Catching this in CI prevents a class of bugs that
only surface at integration time.

## Configuration

```yaml
# .testkit.yml
validators:
  proto_sync:
    enabled: true
    proto_root: api/proto
    # Packages that are allowed to have proto messages
    # without corresponding Go structs (e.g., third-party
    # protos that are consumed but not generated locally).
    skip_packages: []
```

---
