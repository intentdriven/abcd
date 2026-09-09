package cli

// The `history` sub-tree: the front door onto internal/core/history, abcd's
// native session-transcript store (adr-29).
//
// It lives in its own file because the store now has seven sub-verbs across
// three shapes — read (`list`, `show`, `staged`), redacting write (`capture`,
// `drain`, `ingest`) and repair (`migrate`) — and a sub-tree that size buried
// in the root command file is a file about several things.
//
// Everything here is formatting and operator interaction. The store's rules —
// what is redacted, which repository owns a transcript, whether an orphan is
// adopted — are core's, and the one thing this file must never do is decide any
// of them.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// newHistoryCommand builds the `history` sub-tree over internal/core/history —
// the native session-transcript store (adr-29). `list`/`show` read; `capture`
// is the redacting write path. The per-repo store is keyed on the root-commit
// SHA resolved from cwd.
func newHistoryCommand(asJSON *bool) *cobra.Command {
	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "Manage the native session-transcript store",
		Args:  cobra.NoArgs,
		RunE:  helpRunE,
	}

	// capture — the redacting write path: read a raw transcript from a file
	// argument (or stdin with "-"/no arg), sanitise it through the scanner
	// (two-stage, fail-closed), and store the record. This is the ONLY path that
	// writes to the store; list/show never mutate.
	var session, kind string
	captureCmd := &cobra.Command{
		Use:   "capture [<transcript-file>|-]",
		Short: "Redact and store a raw session transcript (reads a file or stdin)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootSHA, err := repoRootSHA()
			if err != nil {
				return err
			}
			src := "-"
			if len(args) == 1 {
				src = args[0]
			}
			// The transcript cap, not the JSON-operand cap: this verb recovers
			// what the hooks store, so it must accept what they accept.
			raw, err := readSourceCapped(cmd, src, maxTranscriptBytes)
			if err != nil {
				return fmt.Errorf("history capture: cannot read transcript: %w", err)
			}
			sess := session
			if sess == "" && src != "-" {
				// Derive a session id from the file basename (sans extension).
				base := filepath.Base(src)
				sess = strings.TrimSuffix(base, filepath.Ext(base))
			}
			if sess == "" {
				return fmt.Errorf("history capture: --session <id> is required when reading from stdin")
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := history.Capture(captureRoot(cwd), rootSHA, raw,
				history.CaptureMeta{SessionID: sess, Kind: orDefault(kind, "native")})
			if err != nil {
				return err
			}
			// The stored path is absolute and home-rooted; this is a success
			// envelope the CLI error scrub never sees, so redact the home root to
			// ~ before it is rendered or marshalled. Callers re-derive the file
			// handle from disk, never from this rendered value. A superseded
			// record's path is the same absolute path from the same store.
			res.Record.Path = fsutil.RedactHome(res.Record.Path)
			if res.Superseded != nil {
				res.Superseded.Path = fsutil.RedactHome(res.Superseded.Path)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				if !res.Wrote {
					fmt.Fprintf(w, "abcd history capture — %s already stored (no-op); redacted secrets=%d home=%d\n",
						res.Record.SessionID, res.Record.Secrets, res.Record.HomePaths)
					return
				}
				fmt.Fprintf(w, "abcd history capture — stored %s (%s)\n", res.Record.SessionID, res.Record.SourceKind)
				fmt.Fprintf(w, "  path:     %s\n", termsafe.Sanitize(res.Record.Path))
				fmt.Fprintf(w, "  redacted: secrets=%d home=%d\n", res.Record.Secrets, res.Record.HomePaths)
			})
		},
	}
	captureCmd.Flags().StringVar(&session, "session", "", "session id for the record (default: transcript filename; required for stdin)")
	captureCmd.Flags().StringVar(&kind, "kind", "", "source kind: native | specstory-import (default native)")
	historyCmd.AddCommand(captureCmd)

	// list — records newest-first for this repo.
	historyCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List stored transcripts for this repo, newest first",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rootSHA, err := repoRootSHA()
			if err != nil {
				return err
			}
			records, err := history.List(rootSHA)
			if err != nil {
				return err
			}
			// An empty store is an empty LIST in JSON, not bare `null`: the
			// command doc promises "an empty list means no transcripts", and a
			// consumer that iterates the value should get [], as every other
			// --json verb's collection does.
			if records == nil {
				records = []history.Record{}
			}
			// The path field is absolute and home-rooted; redact the home root to ~
			// in this success envelope (JSON and text) before it is marshalled.
			for k := range records {
				records[k].Path = fsutil.RedactHome(records[k].Path)
			}
			return render(cmd.OutOrStdout(), *asJSON, records, func(w io.Writer) {
				if len(records) == 0 {
					fmt.Fprintln(w, "abcd history — no transcripts stored for this repo")
					return
				}
				for _, r := range records {
					fmt.Fprintf(w, "%s  %s  %s  redacted secrets=%d home=%d\n",
						r.CapturedAt.Format("2006-01-02T15:04:05Z"), termsafe.Sanitize(r.SessionID), termsafe.Sanitize(r.SourceKind), r.Secrets, r.HomePaths)
				}
			})
		},
	})

	// staged — what ended but is not yet stored. This is the outcome axis the
	// store never had: before staging existed, "absent from the store" spanned
	// never-ended, ended-before-the-store-existed and ended-and-lost, and nothing
	// could tell them apart. A staged entry says exactly one thing.
	historyCmd.AddCommand(&cobra.Command{
		Use:   "staged",
		Short: "List transcripts that ended but are not yet redacted into the store",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rootSHA, err := repoRootSHA()
			if err != nil {
				return err
			}
			staged, err := history.ListStaged(rootSHA)
			if err != nil {
				return err
			}
			// The gap marker is the answer to a question an empty listing
			// cannot: a harness that fires SubagentStop without an
			// agent_transcript_path stages nothing, and "no sub-agent
			// transcripts" then reads as "this session delegated nothing"
			// rather than "this harness cannot deliver them".
			gap, hasGap, gapErr := history.SubagentGap(rootSHA)
			if gapErr != nil {
				fmt.Fprintf(cmd.ErrOrStderr(),
					"abcd history: the sub-agent payload marker is unreadable (%s)\n",
					termsafe.Sanitize(gapErr.Error()))
			}
			if staged == nil {
				staged = []history.Staged{}
			}
			// Every path in the envelope is absolute and home-rooted — the
			// staged copy, the sidecar beside it, and the harness file the
			// bytes were read from — so all three are redacted, not just the
			// one that existed when this verb was written.
			for k := range staged {
				staged[k].Path = fsutil.RedactHome(staged[k].Path)
				staged[k].SidecarPath = fsutil.RedactHome(staged[k].SidecarPath)
				staged[k].SourcePath = fsutil.RedactHome(staged[k].SourcePath)
			}
			// The JSON envelope stays the array it has always been: a
			// consumer that iterates it must keep working. The marker is a
			// per-repo fact rather than a staged entry, so it is reported in
			// the human render, where the reader who needs it is.
			return render(cmd.OutOrStdout(), *asJSON, staged, func(w io.Writer) {
				defer func() {
					if hasGap {
						fmt.Fprintf(w, "\nNOTE: this harness fired %s %d time(s) without an agent_transcript_path (first %s). No sub-agent transcript can be captured on it, so an empty sub-agent corpus here is the harness, not the sessions.\n",
							termsafe.Sanitize(orDefault(gap.Event, "SubagentStop")), gap.Count,
							gap.FirstSeen.Format("2006-01-02T15:04:05Z"))
					}
				}()
				if len(staged) == 0 {
					fmt.Fprintln(w, "abcd history — nothing staged; every ended session is stored")
					return
				}
				for _, s := range staged {
					who := termsafe.Sanitize(s.SessionID)
					if s.AgentID != "" {
						who += " agent " + termsafe.Sanitize(s.AgentID)
						if s.AgentType != "" {
							who += " (" + termsafe.Sanitize(s.AgentType) + ")"
						}
					}
					state := "awaiting redaction"
					if s.Err != "" {
						state = "NOT DRAINABLE: " + termsafe.Sanitize(s.Err)
					}
					fmt.Fprintf(w, "%s  %s  %d bytes  %s\n",
						s.StagedAt.Format("2006-01-02T15:04:05Z"), who, s.Bytes, state)
				}
				fmt.Fprintf(w, "\n%d staged transcript(s) hold UNREDACTED text until drained; run `abcd history drain`.\n", len(staged))
			})
		},
	})

	// drain — finish the capture SessionStart bounded. Unbudgeted by design: the
	// interactive budget exists to protect a session start, and this verb is the
	// explicit ask, so it runs the backlog to completion.
	historyCmd.AddCommand(&cobra.Command{
		Use:   "drain",
		Short: "Redact and store every staged transcript for this repo",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rootSHA, err := repoRootSHA()
			if err != nil {
				return err
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := history.Drain(captureRoot(cwd), rootSHA, history.DrainBudget{})
			if err != nil {
				return err
			}
			if res.Captured == nil {
				res.Captured = []history.Record{}
			}
			for k := range res.Captured {
				res.Captured[k].Path = fsutil.RedactHome(res.Captured[k].Path)
			}
			for k := range res.Failed {
				res.Failed[k].Path = fsutil.RedactHome(res.Failed[k].Path)
			}
			renderErr := render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				if len(res.Captured) == 0 && len(res.Failed) == 0 {
					fmt.Fprintln(w, "abcd history — nothing staged; nothing to drain")
					return
				}
				for _, r := range res.Captured {
					fmt.Fprintf(w, "stored  %s  redacted secrets=%d home=%d\n",
						termsafe.Sanitize(r.SessionID), r.Secrets, r.HomePaths)
				}
				for _, f := range res.Failed {
					fmt.Fprintf(w, "FAILED  %s  %s\n  raw transcript kept (unredacted): %s\n",
						termsafe.Sanitize(f.SessionID), termsafe.Sanitize(f.Err), termsafe.Sanitize(f.Path))
				}
			})
			if renderErr != nil {
				return renderErr
			}
			// A drain that could not store something must not exit 0: this verb is
			// the remedy the SessionStart notice points at, and a silent success
			// here would leave the user believing the backlog cleared.
			if len(res.Failed) > 0 {
				return fmt.Errorf("history: %d staged transcript(s) could not be stored", len(res.Failed))
			}
			return nil
		},
	})

	// show <session-id-or-filename> — metadata + redacted body of one record.
	historyCmd.AddCommand(&cobra.Command{
		Use:   "show <session-id-or-filename>",
		Short: "Show one stored transcript's metadata and redacted body",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootSHA, err := repoRootSHA()
			if err != nil {
				return err
			}
			rec, body, err := history.Read(rootSHA, args[0])
			if err != nil {
				return err
			}
			// Redact the home root out of the absolute stored path in this success
			// envelope (JSON and the text path line) before it is rendered.
			rec.Path = fsutil.RedactHome(rec.Path)
			out := struct {
				history.Record
				Body string `json:"body"`
			}{Record: rec, Body: string(body)}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				// The stored transcript body is untrusted (it may have ingested
				// hostile fetched pages or target-repo files); capture redacts
				// only secrets/home paths, so neutralise terminal-control bytes
				// here before they reach the terminal. SanitizeBlock keeps the
				// transcript's line structure. The metadata fields are validated
				// at write time but not re-validated on the read path, so pass
				// them through too.
				fmt.Fprintf(w, "session:    %s\n", termsafe.Sanitize(rec.SessionID))
				fmt.Fprintf(w, "captured:   %s\n", rec.CapturedAt.Format("2006-01-02T15:04:05Z"))
				fmt.Fprintf(w, "source:     %s\n", termsafe.Sanitize(rec.SourceKind))
				fmt.Fprintf(w, "path:       %s\n", termsafe.Sanitize(rec.Path))
				fmt.Fprintf(w, "redacted:   secrets=%d home=%d\n", rec.Secrets, rec.HomePaths)
				fmt.Fprintln(w, "---")
				fmt.Fprint(w, termsafe.SanitizeBlock(string(body)))
			})
		},
	})

	historyCmd.AddCommand(newHistoryMigrateCommand(asJSON))
	historyCmd.AddCommand(newHistoryIngestCommand(asJSON))

	return historyCmd
}
