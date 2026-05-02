# testkit Sub-packages

Optional sub-packages with additional dependencies.
Import only what you need — the core `testkit` package
has no transitive dependency on any of these.

## testkit/container

Container lifecycle management via `testcontainers-go`.

### SharedContainer

Starts one container for the entire test binary via
`TestMain`, with automatic cleanup. Eliminates per-test
container startup overhead and orphaned containers.

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

Supports Docker, Podman (rootless), and containerd.
The Ryuk reaper is handled automatically based on
runtime detection.

## testkit/httptest

HTTP response assertion helpers. stdlib `net/http` only.

| Function | Description |
|----------|-------------|
| `AssertStatus(t, resp, code, msg)` | Fails if resp.StatusCode != code |
| `AssertJSON(t, resp, &target, msg)` | Decodes JSON body into target |
| `AssertHeader(t, resp, key, want, msg)` | Checks response header value |

## testkit/oteltest

OpenTelemetry metric assertions. Depends on
`go.opentelemetry.io/otel/sdk`.

| Function | Description |
|----------|-------------|
| `AssertCounterDelta(t, meter, name, delta, msg)` | Counter incremented by delta |
| `AssertGaugeValue(t, meter, name, value, msg)` | Gauge at expected value |
| `AssertHistogramCount(t, meter, name, count, msg)` | Histogram recorded count samples |

## testkit/clitest

CLI binary testing via `os/exec`.

### RunBinary

Runs a compiled binary, captures stdout, stderr, and exit
code.

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
| `WithTimeout(d)` | Kill process after d |
| `WithStdin(r)` | Pipe reader to stdin |
| `WithDir(path)` | Set working directory |

### AssertExitCode

Convenience assertion that includes stdout/stderr in the
failure message on mismatch.

```go
clitest.AssertExitCode(t, result, 0,
    "server must start cleanly")
// On failure:
//   server must start cleanly
//     got exit code 1, want 0
//     stderr: "listen tcp :8080: address already in use"
```
