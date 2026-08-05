# testkit Sub-packages

Optional sub-packages with isolated dependencies. Each sub-package is a separate Go module so projects opt in to additional deps only when needed — the core `testkit` package never carries them transitively.

> **Status: not implemented.** None of the modules below exist in the tree.
> They are a design sketch retained for reference. The modules that do ship are
> `go.thesmos.sh/testkit`, `go.thesmos.sh/testkit/engine`, and
> `go.thesmos.sh/testkit/tool` — see
> [ADR-0005](../adr/0005-split-into-published-modules.md).

## testkit/container

Container lifecycle management for integration tests via [`testcontainers-go`](https://github.com/testcontainers/testcontainers-go). Planned: a `SharedContainer` primitive that starts one container per test binary via `TestMain` with automatic cleanup, eliminating per-test startup overhead and orphaned containers. Will support Docker, Podman (rootless), and containerd; the Ryuk reaper handled automatically based on runtime detection.

```go
import "go.thesmos.sh/testkit/container"

func TestMain(m *testing.M) {
    pool := container.Shared(m, container.Config{
        Image:   "postgres:17-alpine",
        Port:    "5432/tcp",
        WaitFor: "database system is ready",
        Env:     map[string]string{"POSTGRES_PASSWORD": "test"},
    })
    os.Exit(m.Run())
}
```

## testkit/httptest

HTTP response assertion helpers. stdlib `net/http` only.

| Planned function | Description |
|----------|-------------|
| `AssertStatus(t, resp, code, msg)` | Fails if `resp.StatusCode != code` |
| `AssertJSON(t, resp, &target, msg)` | Decodes JSON body into target |
| `AssertHeader(t, resp, key, want, msg)` | Checks response header value |

## testkit/oteltest

OpenTelemetry metric assertions. Depends on `go.opentelemetry.io/otel/sdk`.

| Planned function | Description |
|----------|-------------|
| `AssertCounterDelta(t, meter, name, delta, msg)` | Counter incremented by delta |
| `AssertGaugeValue(t, meter, name, value, msg)` | Gauge at expected value |
| `AssertHistogramCount(t, meter, name, count, msg)` | Histogram recorded count samples |

## testkit/clitest

CLI binary testing via `os/exec`. The substrate the [`smoke`](generators/smoke.md) generator targets.

Planned `RunBinary` runs a compiled binary, captures stdout, stderr, exit code:

```go
import "go.thesmos.sh/testkit/clitest"

result := clitest.RunBinary(t, "./bin/myapp",
    []string{"migrate", "--dry-run"},
    clitest.WithEnv("DATABASE_URL", dsn),
    clitest.WithTimeout(30*time.Second),
    clitest.WithStdin(strings.NewReader("yes\n")),
)
testkit.Equal(t, result.ExitCode, 0, "migrate must succeed")
```

| Field | Type | Description |
|-------|------|-------------|
| `ExitCode` | `int` | Process exit code |
| `Stdout` | `string` | Captured standard output |
| `Stderr` | `string` | Captured standard error |

| Option | Description |
|--------|-------------|
| `WithEnv(key, val)` | Set environment variable |
| `WithTimeout(d)` | Kill process after `d` |
| `WithStdin(r)` | Pipe reader to stdin |
| `WithDir(path)` | Set working directory |

`AssertExitCode` will be a convenience that includes stdout/stderr in the failure message on mismatch:

```go
clitest.AssertExitCode(t, result, 0, "server must start cleanly")
// On failure:
//   server must start cleanly
//     got exit code 1, want 0
//     stderr: "listen tcp :8080: address already in use"
```
