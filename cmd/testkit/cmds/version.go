// Copyright Thesmos 2026
// SPDX-License-Identifier: MIT

package cmds

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"go.thesmos.sh/eidos/cli"

	"go.thesmos.sh/testkit/cmd/internal/version"
)

// buildKey is the field name the build stamp takes in JSON output. It matches
// the `build:` label the text renderer uses so the two formats agree.
const buildKey = "build"

// versionKernel is the eidos command this subcommand drives. It is
// package-level because flag binding happens at init() while execution happens
// later: RegisterFlags binds cobra's parsed values into this struct's Config
// fields, so the instance that parsing writes to has to be the instance RunE
// executes.
var versionKernel = &cli.VersionCommand{}

func init() {
	bindKernelFlags(versionCmd, "version", versionKernel.RegisterFlags)
	rootCmd.AddCommand(versionCmd)
}

// versionCmd reports what the binary will produce — the eidos emit-contract
// and the embedded generator set — together with what the binary is, the build
// stamp [version.Full] carries.
//
// The generator list is the useful half of the first part: it is how a
// consumer confirms which generators a given build carries, and which of them
// the resolved config has enabled.
//
// The kernel renders the eidos half and owns both output formats, so its
// output is captured rather than streamed: the build stamp is a line appended
// in text mode and a field spliced in under `--diag-format json`. Streaming
// would leave JSON callers without the stamp, which is the format that most
// needs it.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print brand, emit-contract, generator list, and build",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		env, cfg, err := prepare(cmd)
		if err != nil {
			return err
		}
		versionKernel.Config.File = cfg
		versionKernel.Config.Plugins = generators()
		return runVersion(cmd.Context(), env, versionKernel)
	},
}

// runVersion executes the kernel and renders the result with the build stamp
// added. It is separate from RunE so the failure arms — a kernel that reports
// non-zero, a stdout that refuses writes — are reachable without driving the
// whole CLI.
func runVersion(ctx context.Context, env *cli.Env, kernel *cli.VersionCommand) error {
	// The kernel writes to env.Stdout; hand it a buffer so the rendered form
	// can be amended before it reaches the caller.
	captured := *env
	var buf bytes.Buffer
	captured.Stdout = &buf

	if code := kernel.Execute(ctx, &captured); code != cli.ExitOK {
		// Flush whatever the kernel managed to render, then carry its code
		// out: appending a build stamp to a failed run would misreport it.
		if _, err := io.Copy(env.Stdout, &buf); err != nil {
			return fmt.Errorf("cmds: write version: %w", err)
		}
		return exit(code)
	}
	return writeVersion(env.Stdout, buf.Bytes(), kernel.Config.Format)
}

// writeVersion emits the kernel's rendering with the build stamp added, in
// whichever form the format calls for.
func writeVersion(w io.Writer, rendered []byte, format cli.DiagFormat) error {
	if format == cli.DiagFormatJSON {
		return writeVersionJSON(w, rendered)
	}
	if _, err := w.Write(rendered); err != nil {
		return fmt.Errorf("cmds: write version: %w", err)
	}
	if _, err := fmt.Fprintf(w, "%s: %s\n", buildKey, version.Full()); err != nil {
		return fmt.Errorf("cmds: write build stamp: %w", err)
	}
	return nil
}

// writeVersionJSON splices the build stamp into the kernel's document.
//
// Decoding into a map rather than a typed struct is deliberate: the kernel
// owns that schema and adds fields to it. A struct here would silently drop
// any field eidos adds; the map carries them through untouched.
func writeVersionJSON(w io.Writer, rendered []byte) error {
	doc := map[string]any{}
	if err := json.Unmarshal(rendered, &doc); err != nil {
		return fmt.Errorf("cmds: version document is not JSON: %w", err)
	}
	doc[buildKey] = version.Full()

	// doc came out of json.Unmarshal and gained one string, so it holds only
	// JSON-native values and cannot fail to marshal. Discarding the error is
	// deliberate: a guard here would be a branch no input reaches.
	out, _ := json.Marshal(doc) //nolint:errchkjson // values originate from json.Unmarshal
	if _, err := fmt.Fprintf(w, "%s\n", out); err != nil {
		return fmt.Errorf("cmds: write version: %w", err)
	}
	return nil
}
