# Migration Chain

Validates SQL migration files form a contiguous,
correctly-ordered chain with no gaps, no duplicates, and
no schema anti-patterns.

## Command

```
testkit validate migration
```

## What it checks

1. Migration files follow the naming convention
   `NNN-<description>.sql` (zero-padded, monotonic)
2. No sequence gaps (e.g., 001, 002, 004 is an error)
3. No duplicate sequence numbers
4. `RequiredSchemaVersion` constant in Go source matches
   the highest migration version number
5. No `DEFAULT CURRENT_TIMESTAMP` or `now()` in up
   migrations (invariant: no database-side timestamps on
   data tables)
6. Every table has a `PRIMARY KEY` declaration
7. Every `.up.sql` has a matching `.down.sql` (when the
   project convention requires reversible migrations)

## Failure output

```
migration: FAIL

  plugins/postgres/schema/
    gap: 003 missing between 002-snapshots.sql and 004-active-tasks.sql

  plugins/postgres/schema/005-user-updates.sql
    line 12: DEFAULT now() violates no-database-timestamps invariant

  plugins/postgres/pool.go
    RequiredSchemaVersion = 5 but highest migration is 006
```

## Why

Migration chain errors are catastrophic in production —
a gap means `migrate.Up()` skips a version, a duplicate
means one version overwrites another, and stale
`RequiredSchemaVersion` means the boot assertion passes
on an incomplete schema. Catching these at PR time is
orders of magnitude cheaper than catching them at deploy
time.

## Configuration

```yaml
# .testkit.yml
validators:
  migration:
    enabled: true
    dirs:
      - plugins/postgres/schema
    naming: "{seq:03d}-{description}.sql"
    reversible: false
    # Go source file containing RequiredSchemaVersion constant.
    version_source: plugins/postgres/pool.go
```

---
