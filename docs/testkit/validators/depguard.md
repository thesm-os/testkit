# Depguard Sync

Enforces that the import graph matches layer boundary
rules. Catches cross-layer imports that violate the
architecture before they reach code review.

## Command

```
testkit validate depguard
```

## What it checks

1. Model packages never import service or plugin packages
2. Service packages never import plugin packages
3. Plugin packages never import other plugin packages
4. Test packages (`*test`) may import their parent but not
   sibling test packages
5. No circular dependencies between packages at the same
   layer

## Failure output

```
depguard: FAIL

  model/snapshot.go
    imports example.com/myapp/service/dispatcher
    violation: model -> service

  plugins/postgres/store_backend.go
    imports example.com/myapp/plugins/sqlite/keyspace
    violation: plugin -> sibling plugin
```

## Why

Layer violations are the most common architectural
regression. A single `import` in the wrong direction
creates a coupling that is expensive to undo — downstream
modules inherit transitive dependencies, build times
increase, and the dependency graph becomes a hairball.

The validator operates on the testkit config as the single
source of truth. A future generator can emit the
`.golangci.yml` `depguard` section from the same config,
eliminating the need to maintain rules in two places.

## Configuration

```yaml
# .testkit.yml
validators:
  depguard:
    enabled: true
    layers:
      - name: model
        pattern: "example.com/myapp/model/..."
        deny:
          - "example.com/myapp/service/..."
          - "example.com/myapp/plugins/..."
      - name: service
        pattern: "example.com/myapp/service/..."
        deny:
          - "example.com/myapp/plugins/..."
      - name: plugins
        pattern: "example.com/myapp/plugins/*"
        deny_siblings: true
```

---
