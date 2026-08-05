// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cmds

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.thesmos.sh/eidos/cli"
)

// The binary reports the generator set so an operator can confirm which
// generators a given build carries. An empty set has to be visible rather than
// silent, which is why this asserts the rendered line and not the slice.
//
//nolint:paralleltest // drives the shared rootCmd
func TestVersionTextOutput(t *testing.T) {
	stdout, _, code := runRoot(t, "version")

	if code != cli.ExitOK {
		t.Fatalf("version must succeed, got exit %d", code)
	}
	for _, want := range []string{"testkit", "emit-contract:", "plugins: (none)", "build: dev"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("version output missing %q:\n%s", want, stdout)
		}
	}
}

// JSON is the format most likely to be consumed by tooling, so it carries the
// build stamp as a field rather than losing it. The document has to stay
// parseable with the field spliced in.
//
//nolint:paralleltest // drives the shared rootCmd
func TestVersionJSONCarriesBuildStamp(t *testing.T) {
	stdout, _, code := runRoot(t, "version", "--diag-format", "json")

	if code != cli.ExitOK {
		t.Fatalf("version must succeed, got exit %d", code)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("JSON output must parse as a document (%v):\n%s", err, stdout)
	}
	if doc["brand"] != brand {
		t.Errorf("expected brand %q in the document, got %v", brand, doc["brand"])
	}
	if doc[buildKey] != "dev" {
		t.Errorf("expected an unstamped build to report %q, got %v", "dev", doc[buildKey])
	}
	// Fields the kernel owns must survive the splice untouched.
	if _, ok := doc["emit_contract"]; !ok {
		t.Errorf("the kernel's fields must pass through, got: %s", stdout)
	}
}

// The splice decodes into a map rather than a typed struct so fields eidos
// adds to its document pass through. A struct would drop them silently.
func TestWriteVersionJSONPreservesUnknownFields(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	rendered := []byte(`{"brand":"testkit","a_field_added_upstream":42}`)
	if err := writeVersionJSON(&out, rendered); err != nil {
		t.Fatalf("splice: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(out.String()), &doc); err != nil {
		t.Fatalf("result must parse: %v", err)
	}
	if doc["a_field_added_upstream"] != float64(42) {
		t.Errorf("an unknown field must survive, got: %s", out.String())
	}
	if doc[buildKey] != "dev" {
		t.Errorf("the build stamp must be added, got: %s", out.String())
	}
}

// A malformed document has to surface rather than being written through as
// broken JSON.
func TestWriteVersionJSONRejectsNonJSON(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	if err := writeVersionJSON(&out, []byte("not json")); err == nil {
		t.Fatal("a non-JSON rendering must be reported, not passed through")
	}
}

// pflag reads a single dash as a shorthand cluster, so the long form is the
// only accepted spelling. That is a deliberate consequence of bridging eidos's
// stdlib flags into cobra, and it has to fail loudly rather than silently
// ignoring the flag.
//
//nolint:paralleltest // drives the shared rootCmd
func TestVersionRejectsSingleDashFlag(t *testing.T) {
	_, _, code := runRoot(t, "version", "-diag-format", "json")

	if code == cli.ExitOK {
		t.Fatal("a single-dash long flag must not be silently accepted")
	}
}

// The subcommand takes no positional arguments. Accepting them silently would
// let a typo'd flag look like a successful run.
//
//nolint:paralleltest // drives the shared rootCmd
func TestVersionRejectsArguments(t *testing.T) {
	_, _, code := runRoot(t, "version", "extra")

	if code == cli.ExitOK {
		t.Fatal("a positional argument must be rejected")
	}
}

// failWriter refuses every write. The version renderers run at the end of a
// command, where a broken stdout — a closed pipe from `testkit version | head`
// — is the realistic failure and has to surface rather than being dropped.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errClosedPipe }

var errClosedPipe = errors.New("cmds: broken pipe")

func TestWriteVersionSurfacesWriteFailures(t *testing.T) {
	t.Parallel()

	rendered := []byte("testkit\nemit-contract: 1.0.0\n")

	t.Run("text: a failing body write is reported", func(t *testing.T) {
		t.Parallel()
		if err := writeVersion(failWriter{}, rendered, cli.DiagFormatText); err == nil {
			t.Fatal("a failed write must not be swallowed")
		}
	})

	// The body write succeeds and the build stamp fails, which is the arm a
	// single failing writer cannot reach.
	t.Run("text: a failing build-stamp write is reported", func(t *testing.T) {
		t.Parallel()
		w := &failAfter{n: 1}
		if err := writeVersion(w, rendered, cli.DiagFormatText); err == nil {
			t.Fatal("a failed build-stamp write must not be swallowed")
		}
	})

	t.Run("json: a failing write is reported", func(t *testing.T) {
		t.Parallel()
		err := writeVersion(failWriter{}, []byte(`{"brand":"testkit"}`), cli.DiagFormatJSON)
		if err == nil {
			t.Fatal("a failed write must not be swallowed")
		}
	})
}

// failAfter accepts n writes then refuses the rest.
type failAfter struct{ n int }

func (f *failAfter) Write(p []byte) (int, error) {
	if f.n <= 0 {
		return 0, errClosedPipe
	}
	f.n--
	return len(p), nil
}

// The kernel reports failures as exit codes, and a failed run must not get a
// build stamp appended: that would present a broken render as a complete one.
// The kernel rejects an env with no brand, which is the only failure it has.
func TestRunVersionCarriesKernelFailure(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	env := &cli.Env{Brand: "", Stdout: &out, Stderr: &out}

	err := runVersion(t.Context(), env, &cli.VersionCommand{})
	if err == nil {
		t.Fatal("a kernel failure must surface")
	}
	var ec exitCodeError
	if !asExitCode(err, &ec) {
		t.Fatalf("the kernel's exit code must be carried out, got %T", err)
	}
	if ec.code != cli.ExitUserError {
		t.Fatalf("expected exit %d, got %d", cli.ExitUserError, ec.code)
	}
	if strings.Contains(out.String(), "build:") {
		t.Errorf("a failed run must not be stamped, got: %s", out.String())
	}
}

// A stdout that refuses writes during the failure flush has to surface too,
// rather than masking the original failure with silence.
func TestRunVersionReportsFlushFailure(t *testing.T) {
	t.Parallel()

	env := &cli.Env{Brand: "", Stdout: failWriter{}, Stderr: failWriter{}}

	if err := runVersion(t.Context(), env, &cli.VersionCommand{}); err == nil {
		t.Fatal("a failed flush must not be swallowed")
	}
}

// The subcommand resolves its config before running, so a named file that is
// absent has to stop the command rather than falling back to defaults.
//
//nolint:paralleltest // drives the shared rootCmd
func TestVersionRejectsUnreadableConfig(t *testing.T) {
	_, _, code := runRoot(t, "version", "--config", filepath.Join(t.TempDir(), "absent.yaml"))

	if code == cli.ExitOK {
		t.Fatal("a config file the caller named must not be silently skipped")
	}
}

// A working directory that vanished under the process leaves nothing to anchor
// config discovery or artifact paths against, and the command must stop.
//
//nolint:paralleltest // t.Chdir cannot be used from a parallel test
func TestVersionReportsUnresolvableWorkdir(t *testing.T) {
	nested := filepath.Join(t.TempDir(), "gone")
	if err := os.Mkdir(nested, 0o750); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Chdir(nested)
	if err := os.Remove(nested); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, _, code := runRoot(t, "version")
	if code == cli.ExitOK {
		t.Fatal("an unresolvable working directory must stop the command")
	}
}
