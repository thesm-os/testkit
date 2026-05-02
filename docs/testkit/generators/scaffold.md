# Scaffold command

`testkit scaffold` generates a one-time companion file
for a stub — the hand-written part with TODO markers.
Unlike generators, scaffold runs once (not on every
`go generate`) and produces a file intended for editing.

## Command

```bash
testkit scaffold stub storetest Store
```

## Output

`storetest/in_memory_store.go` (if it doesn't already
exist — scaffold never overwrites):

```go
// in_memory_store.go — scaffolded by testkit, safe to edit.

package storetest

import (
    "context"
    "sync"

    "example.com/myapp/store"
)

type storeState struct {
    mu sync.Mutex
    // TODO: add internal state fields
}

// NewInMemoryStore constructs a StoreStub backed by
// in-memory state.
func NewInMemoryStore(opts ...StoreOption) *StoreStub {
    state := &storeState{}
    return NewStoreStub(
        state.put,
        state.get,
        state.list,
        state.count,
        state.delete,
        opts...,
    )
}

func (s *storeState) put(ctx context.Context, req store.PutRequest) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    // TODO: implement
    return nil
}

func (s *storeState) get(ctx context.Context, key string) (string, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    // TODO: implement
    return "", nil
}

// ... one method per interface method, all returning zero values
```

## Why

The stub generator produces the mechanical skeleton. But
the developer still has to create the companion file from
scratch, knowing the exact constructor signature, one
method per interface method, and the correct types. This
is a cold-start problem.

`testkit scaffold` eliminates it: run once, get a
compilable file with TODOs, fill in the domain logic.
The file is not regenerated — it's the developer's from
that point on.

## What else can be scaffolded

| Command | Creates |
|---------|---------|
| `testkit scaffold stub <pkg> <Interface>` | `in_memory_<subject>.go` with state struct + defaults |
| `testkit scaffold model <pkg> <Interface>` | `<subject>_model_oracle.go` with oracle interface impl |
| `testkit scaffold builder <pkg> <Struct>` | `fixtures.go` entry with defaults constructor |

---
