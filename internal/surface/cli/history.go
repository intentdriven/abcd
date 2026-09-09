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
	var stagedAllRepos bool
	stagedCmd := &cobra.Command{
		Use:   "staged",
		Short: "List transcripts that ended but are not yet redacted into the store",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// --all-repos answers the question no per-repo listing can
			// (iss-2609090722466403): the repository whose raw transcripts grow
			// without bound is the one nobody opens, so the verb that reports
			// the backlog must have a form that does not need the operator to
			// be standing in it. It is a survey — counts, sizes, names — and
			// never another repository's session ids or paths.
			//
			// It resolves no root SHA at all, which is deliberate: run from
			// anywhere, including outside a git repository, it still answers.
			if stagedAllRepos {
				return renderBacklogSurvey(cmd, *asJSON)
			}
			rootSHA, err := repoRootSHA()
			if err != nil {
				return err
			}
			staged, err := history.ListStaged(rootSHA)
			if err != nil {
				return err
			}
			// Quarantined transcripts are raw bytes on the same disk under the
			// same 0o700, so a listing that omitted them would under-report
			// exactly the transcripts that will never leave on their own.
			quarantined, qerr := history.ListQuarantined(rootSHA)
			if qerr != nil {
				fmt.Fprintf(cmd.ErrOrStderr(),
					"abcd history: the quarantine directory is unreadable (%s)\n",
					termsafe.Sanitize(fsutil.RedactHome(qerr.Error())))
			}
			for k := range quarantined {
				quarantined[k].Path = fsutil.RedactHome(quarantined[k].Path)
				quarantined[k].SidecarPath = fsutil.RedactHome(quarantined[k].SidecarPath)
				quarantined[k].ReasonPath = fsutil.RedactHome(quarantined[k].ReasonPath)
				quarantined[k].Reason = fsutil.RedactHome(quarantined[k].Reason)
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
				if len(staged) == 0 && len(quarantined) == 0 {
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
					if s.Overdue {
						// The age is stated on the entry rather than only in
						// the summary, because the reader deciding what to do
						// needs to know WHICH file has been sitting there.
						state = "OVERDUE (staged more than " + history.StagedTTL.String() + " ago), awaiting redaction"
					}
					if s.Err != "" {
						state = "NOT DRAINABLE: " + termsafe.Sanitize(s.Err)
					}
					fmt.Fprintf(w, "%s  %s  %d bytes  %s\n",
						s.StagedAt.Format("2006-01-02T15:04:05Z"), who, s.Bytes, state)
				}
				if len(staged) > 0 {
					fmt.Fprintf(w, "\n%d staged transcript(s) hold UNREDACTED text until drained; run `abcd history drain`.\n", len(staged))
				}
				if len(quarantined) == 0 {
					return
				}
				// Quarantine is reported as its own block and in its own
				// words. A quarantined transcript is NOT waiting for anything:
				// no drain will retry it, no age will clear it, and the only
				// thing that changes its state is a person deciding. Folding it
				// into the staged list above would file it under "awaiting
				// redaction", which is the one thing it is not.
				fmt.Fprintf(w, "\nQUARANTINED — these can never be redacted; nothing will retry them:\n")
				var qbytes int64
				for _, q := range quarantined {
					who := termsafe.Sanitize(q.SessionID)
					if q.AgentID != "" {
						who += " agent " + termsafe.Sanitize(q.AgentID)
					}
					reason := termsafe.Sanitize(orDefault(q.Reason, q.Err))
					fmt.Fprintf(w, "%s  %s  %d bytes  %s\n    %s\n",
						q.QuarantinedAt.Format("2006-01-02T15:04:05Z"), who, q.Bytes,
						termsafe.Sanitize(filepath.Base(q.Path)), reason)
					qbytes += q.Bytes
				}
				fmt.Fprintf(w, "\n%d quarantined transcript(s) hold %d bytes of UNREDACTED text indefinitely. Read one, then remove it deliberately with `abcd history discard <file> --yes`.\n",
					len(quarantined), qbytes)
			})
		},
	}
	stagedCmd.Flags().BoolVar(&stagedAllRepos, "all-repos", false,
		"survey every repository in the store, not just this one")
	historyCmd.AddCommand(stagedCmd)

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

	// discard — the ONE door in abcd that destroys a transcript nothing has
	// stored (iss-2609090722466403).
	//
	// It exists because quarantine without an exit is a room with no door: a
	// transcript the fail-closed scanner will never pass would otherwise sit
	// unredacted on disk forever with no legitimate way to be rid of it, and an
	// operator with no sanctioned removal reaches for `rm` on a directory whose
	// neighbouring files (the sidecar, the reason note, the lock) they have no
	// reason to know about.
	//
	// The confirmation lives HERE and not in core, and that is the whole shape
	// of the boundary: core.Discard deletes whatever it is told to delete and
	// neither prompts nor prints, and this verb is where the human decision is
	// taken and refused without --yes. Nothing in abcd calls Discard except a
	// person typing this.
	var discardYes bool
	discardCmd := &cobra.Command{
		Use:   "discard <staged-filename>",
		Short: "Permanently delete one staged or quarantined raw transcript (requires --yes)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootSHA, err := repoRootSHA()
			if err != nil {
				return err
			}
			if !discardYes {
				// Named, not merely refused: the operator is being asked to
				// confirm the destruction of the only copy of something, and a
				// refusal that did not say WHAT would be confirmed is not a
				// confirmation.
				return fmt.Errorf("history discard: %q holds the only copy of an unredacted transcript and deleting it is irreversible; pass --yes to confirm (`abcd history staged` shows what each file is)",
					termsafe.Sanitize(args[0]))
			}
			res, err := history.Discard(rootSHA, args[0])
			if err != nil {
				return err
			}
			res.Path = fsutil.RedactHome(res.Path)
			for k := range res.Removed {
				res.Removed[k] = fsutil.RedactHome(res.Removed[k])
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "abcd history discard — deleted %d bytes of unredacted transcript, unrecoverably\n", res.Bytes)
				for _, r := range res.Removed {
					fmt.Fprintf(w, "  removed: %s\n", termsafe.Sanitize(r))
				}
			})
		},
	}
	discardCmd.Flags().BoolVar(&discardYes, "yes", false, "confirm the irreversible deletion of an unredacted transcript")
	historyCmd.AddCommand(discardCmd)

	historyCmd.AddCommand(newHistoryMigrateCommand(asJSON))
	historyCmd.AddCommand(newHistoryIngestCommand(asJSON))
	historyCmd.AddCommand(newHistoryReconstructCommand(asJSON))

	return historyCmd
}

// renderBacklogSurvey renders `history staged --all-repos`: every repository in
// the store that is holding unredacted transcript text.
//
// It reports COUNTS, SIZES and repository NAMES and nothing else. The reader is
// standing in one repository and being told about others, and a store whose
// premise is that transcripts are redacted before they are readable should not
// spill one repository's session ids into another repository's terminal to make
// a summary richer. Whoever needs the detail runs the verb in that repository.
func renderBacklogSurvey(cmd *cobra.Command, asJSON bool) error {
	repos, err := history.SurveyBacklog()
	if err != nil {
		return err
	}
	if repos == nil {
		repos = []history.RepoBacklog{}
	}
	return render(cmd.OutOrStdout(), asJSON, repos, func(w io.Writer) {
		if len(repos) == 0 {
			fmt.Fprintln(w, "abcd history — no repository in this store is holding unredacted transcript text")
			return
		}
		var total int64
		var overdue, quarantined int
		for _, b := range repos {
			name := termsafe.Sanitize(b.Name)
			if name == "" {
				name = "(unregistered)"
			}
			line := fmt.Sprintf("%s  %s  %d staged (%s)", b.RootSHA[:12], name, b.Staged, humanBytes(int(b.StagedBytes)))
			if b.Overdue > 0 {
				line += fmt.Sprintf("  %d OVERDUE", b.Overdue)
			}
			if b.Quarantined > 0 {
				line += fmt.Sprintf("  %d quarantined (%s)", b.Quarantined, humanBytes(int(b.QuarantinedBytes)))
			}
			if !b.OldestStagedAt.IsZero() {
				line += "  oldest " + b.OldestStagedAt.Format("2006-01-02")
			}
			fmt.Fprintln(w, line)
			total += b.Total()
			overdue += b.Overdue
			quarantined += b.Quarantined
		}
		fmt.Fprintf(w, "\n%d repositor(y/ies) hold %s of UNREDACTED transcript text.\n", len(repos), humanBytes(int(total)))
		if overdue > 0 {
			fmt.Fprintf(w, "%d staged transcript(s) are past the %s limit. A drain only runs in the repository that owns them: open a session there, or run `abcd history drain` from it.\n",
				overdue, history.StagedTTL)
		}
		if quarantined > 0 {
			fmt.Fprintf(w, "%d transcript(s) can never be redacted and will never leave on their own; `abcd history discard` is the only thing that removes them.\n", quarantined)
		}
	})
}
