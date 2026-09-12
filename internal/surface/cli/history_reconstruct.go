package cli

// The `history reconstruct` verb: one session out of the store as one readable
// artefact and one machine-readable telemetry file.
//
// Core builds both and knows no path; this file is where they land. The two
// files are written together or not at all — a telemetry file describing an
// artefact that was never written would be a measure of nothing.

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// defaultMaxBlockBytes caps one rendered tool input or tool result.
//
// A cap is needed because the artefact's consumer is a model being handed the
// session as context and the corpus does not respect that: the largest stored
// main thread on this machine is 38 MB. 8 KiB keeps a tool result readable —
// long enough to carry a diagnostic or a file excerpt whole — while stopping a
// single dumped file from being most of the document. Whatever it removes is
// marked in place and counted in the telemetry's completeness block, so a short
// artefact is never mistaken for a short session. `--max-block-bytes 0` turns
// it off.
const defaultMaxBlockBytes = 8 << 10

// reconstructStdout is the --out value that writes to stdout instead of files.
const reconstructStdout = "-"

// newHistoryReconstructCommand builds `abcd history reconstruct`.
func newHistoryReconstructCommand(asJSON *bool) *cobra.Command {
	var out, mode string
	var maxBlock int
	cmd := &cobra.Command{
		Use:   "reconstruct <session-id>",
		Short: "Render one session — the main thread and every sub-agent — as one artefact plus telemetry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, rootSHA, err := historyStore(cmd)
			if err != nil {
				return err
			}
			res, err := history.Reconstruct(repoRoot, rootSHA, history.ReconstructOptions{
				SessionID:     args[0],
				Mode:          history.ReconstructMode(mode),
				MaxBlockBytes: maxBlock,
			})
			if err != nil {
				return err
			}
			if out == reconstructStdout {
				return writeReconstructionToStdout(cmd.OutOrStdout(), *asJSON, res)
			}
			written, err := writeReconstruction(out, res)
			if err != nil {
				return err
			}
			// The written paths are real paths on this machine, so they are
			// home-redacted before they are rendered or marshalled — the same
			// discipline every other history verb's success envelope follows.
			for k := range written {
				written[k] = fsutil.RedactHome(written[k])
			}
			envelope := struct {
				history.Reconstruction
				Written []string `json:"written"`
			}{Reconstruction: res, Written: written}
			return render(cmd.OutOrStdout(), *asJSON, envelope, func(w io.Writer) {
				renderReconstruct(w, res, written)
			})
		},
	}
	cmd.Flags().StringVar(&out, "out", ".",
		"directory to write <session>.md and <session>.telemetry.json into, or - for stdout")
	cmd.Flags().StringVar(&mode, "mode", string(history.ModeFull),
		"full (every turn of every agent) | spine (the main thread whole, each sub-agent reduced to its instruction and its conclusion)")
	cmd.Flags().IntVar(&maxBlock, "max-block-bytes", defaultMaxBlockBytes,
		"truncate one rendered tool input or result at this many bytes (0 disables); what is removed is marked and counted")
	return cmd
}

// writeReconstruction writes the artefact and its telemetry into dir and
// returns the two paths, artefact first.
//
// The directory must already exist and be a real directory. Creating one would
// mean guessing that a mistyped path was meant, and this verb writes a document
// whose whole value is being findable afterwards.
func writeReconstruction(dir string, res history.Reconstruction) ([]string, error) {
	if !fsutil.IsRealDir(dir) {
		return nil, fmt.Errorf("history reconstruct: --out %s is not an existing directory", dir)
	}
	tel, err := marshalTelemetry(res)
	if err != nil {
		return nil, err
	}
	artefactPath := filepath.Join(dir, res.ArtefactName)
	telemetryPath := filepath.Join(dir, res.TelemetryName)
	if err := fsutil.WriteFileAtomic(artefactPath, res.Artefact, 0o644); err != nil {
		return nil, fmt.Errorf("history reconstruct: write artefact: %w", err)
	}
	if err := fsutil.WriteFileAtomic(telemetryPath, tel, 0o644); err != nil {
		return nil, fmt.Errorf("history reconstruct: write telemetry: %w", err)
	}
	return []string{artefactPath, telemetryPath}, nil
}

// writeReconstructionToStdout emits both halves on one stream. In --json that
// is the envelope with the artefact carried as a field; otherwise it is the
// artefact, then a fenced telemetry block, so a reader piping this into a file
// still gets both.
func writeReconstructionToStdout(w io.Writer, asJSON bool, res history.Reconstruction) error {
	tel, err := marshalTelemetry(res)
	if err != nil {
		return err
	}
	if asJSON {
		envelope := struct {
			history.Reconstruction
			Artefact string `json:"artefact"`
		}{Reconstruction: res, Artefact: string(res.Artefact)}
		return render(w, true, envelope, nil)
	}
	// The artefact is a stored transcript rendered: untrusted text that may
	// have ingested hostile pages or files. Capture redacted secrets and home
	// paths, not terminal control bytes, so those are neutralised here.
	fmt.Fprint(w, termsafe.SanitizeBlock(string(res.Artefact)))
	fmt.Fprintf(w, "\n<!-- %s -->\n\n```json\n%s\n```\n", res.TelemetryName, tel)
	return nil
}

// marshalTelemetry renders the telemetry file's bytes, newline-terminated so
// the file is a well-formed text file rather than a bare JSON blob.
func marshalTelemetry(res history.Reconstruction) ([]byte, error) {
	data, err := json.MarshalIndent(res.Telemetry, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("history reconstruct: marshal telemetry: %w", err)
	}
	return append(data, '\n'), nil
}

// renderReconstruct writes the human summary. It reports the artefact's SIZE
// alongside its path, because the one thing a caller cannot tell from a
// filename is whether what they just produced will fit where they meant to put
// it.
func renderReconstruct(w io.Writer, res history.Reconstruction, written []string) {
	tel := res.Telemetry
	fmt.Fprintf(w, "abcd history reconstruct — session %s (%s mode)\n",
		termsafe.Sanitize(tel.SessionID), termsafe.Sanitize(tel.Mode))
	for _, p := range written {
		fmt.Fprintf(w, "  wrote:      %s\n", termsafe.Sanitize(p))
	}
	fmt.Fprintf(w, "  artefact:   %s\n", humanBytes(res.ArtefactBytes))
	fmt.Fprintf(w, "  agents:     %d (%d sub-agent transcript(s))\n",
		len(tel.Agents), max(len(tel.Agents)-1, 0))
	fmt.Fprintf(w, "  turns:      %d (%d user, %d assistant)\n",
		tel.Turns.Total, tel.Turns.User, tel.Turns.Assistant)
	fmt.Fprintf(w, "  tokens:     %d over %d API response(s)\n",
		tel.Tokens.Total, tel.Tokens.APIResponses)
	fmt.Fprintf(w, "  duration:   %.0fs\n", tel.WallClockSeconds)
	if len(tel.ToolCalls) > 0 {
		fmt.Fprintf(w, "  tool calls: %s\n", topTools(tel.ToolCalls))
	}
	c := tel.Completeness
	if !c.MainThreadPresent {
		fmt.Fprintln(w, "  NOTE: no main-thread record is stored for this session; no spawn point could be established for any agent")
	}
	if c.AgentsWithoutSpawnPoint > 0 {
		fmt.Fprintf(w, "  NOTE: %d sub-agent(s) could not be placed and are listed under \"Unattributed sub-agents\"\n",
			c.AgentsWithoutSpawnPoint)
	}
	if len(c.DroppedRecords) > 0 {
		fmt.Fprintf(w, "  NOTE: %d record(s) found for this session were not used; the artefact names each one\n",
			len(c.DroppedRecords))
	}
	if c.UsageWithoutMessageID > 0 {
		fmt.Fprintf(w, "  NOTE: %d response(s) carried usage that could not be de-duplicated; the token total is an upper bound to that extent\n",
			c.UsageWithoutMessageID)
	}
}

// topTools renders the tool-call histogram most-used first, deterministically.
func topTools(calls map[string]int) string {
	names := make([]string, 0, len(calls))
	for n := range calls {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		if calls[names[i]] != calls[names[j]] {
			return calls[names[i]] > calls[names[j]]
		}
		return names[i] < names[j]
	})
	var b strings.Builder
	for i, n := range names {
		if i > 0 {
			b.WriteString(", ")
		}
		if i == 6 {
			fmt.Fprintf(&b, "and %d more", len(names)-i)
			break
		}
		fmt.Fprintf(&b, "%s=%d", termsafe.Sanitize(n), calls[n])
	}
	return b.String()
}
