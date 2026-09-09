package cli

// The two recovery verbs — `history migrate` and `history ingest` — and the
// one piece of harness knowledge they share.
//
// Both live here rather than in core because both are transport concerns
// wearing a store's clothes. Core decides which repository owns a transcript
// and what may be written; this file decides what to print, what to ask, and
// where on THIS machine the harness happens to keep its files. That last part
// is the reason the seams exist: a vendor's directory layout is the one thing
// core must never learn, because it is the one thing the vendor can change
// without telling anyone.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/history"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// harnessSidecarPrefix, with harnessSidecarSuffix, is the harness's per-agent
// metadata filename. The NAME is used, never the directory structure around it:
// an index built by looking for a file called after the agent works whatever
// nesting the harness chooses, and this machine's store already holds two
// different nestings for it.
const harnessSidecarPrefix = "agent-"

// sidecarIndexDepth bounds the walk that builds the agent index.
const sidecarIndexDepth = 8

// newHistoryMigrateCommand builds `abcd history migrate`.
//
// It reports by default. The store holds the only copy of these records, so
// writing is the explicit ask and never the default.
func newHistoryMigrateCommand(asJSON *bool) *cobra.Command {
	var apply bool
	var sidecarRoots []string
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Repair records filed under a composite session id (reports; writes only with --apply)",
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
			repoRoot := captureRoot(cwd)
			roots, err := resolveSidecarRoots(repoRoot, sidecarRoots)
			if err != nil {
				return err
			}
			res, err := history.Migrate(rootSHA, history.MigrateOptions{
				RepoRoot: repoRoot,
				Apply:    apply,
				Lineage:  harnessLineage(roots),
			})
			if err != nil {
				return err
			}
			redactMigratePaths(&res)
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				renderMigrate(w, res, len(roots))
			})
		},
	}
	cmd.Flags().BoolVar(&apply, "apply", false, "write the repaired records (default: report only)")
	cmd.Flags().StringArrayVar(&sidecarRoots, "sidecar-root", nil,
		"directory to search for the harness's per-agent metadata (repeatable; default: ingest_roots from "+history.ConfigRelPath+")")
	return cmd
}

// renderMigrate writes the human report.
func renderMigrate(w io.Writer, res history.MigrateResult, roots int) {
	mode := "report only — nothing was written; re-run with --apply to write"
	if res.Applied {
		mode = "applied"
	}
	fmt.Fprintf(w, "abcd history migrate — %d record(s) scanned, %d filed under a composite id (%s)\n",
		res.Scanned, res.Composite, mode)
	if res.Composite == 0 {
		fmt.Fprintln(w, "  nothing to migrate")
		return
	}
	if roots == 0 {
		fmt.Fprintf(w, "  no sidecar root declared, so no agent type or spawn depth can be recovered; declare ingest_roots in %s or pass --sidecar-root\n",
			history.ConfigRelPath)
	}
	var enriched int
	for _, e := range res.Migrated {
		if e.SpawnAttribution == "sidecar" {
			enriched++
		}
	}
	fmt.Fprintf(w, "  %d repairable, %d of them with recoverable lineage; %d refused\n",
		len(res.Migrated), enriched, len(res.Refused))
	for _, e := range res.Migrated {
		fmt.Fprintf(w, "  %s -> session %s agent %s (%s)\n",
			termsafe.Sanitize(e.StoredSessionID), termsafe.Sanitize(e.SessionID),
			termsafe.Sanitize(e.AgentID), termsafe.Sanitize(e.SpawnAttribution))
		if e.ExtraSegment != "" {
			fmt.Fprintf(w, "      NOTE: the composite also carried %q, which the schema has no field for and this repair drops\n",
				termsafe.Sanitize(e.ExtraSegment))
		}
	}
	for _, e := range res.Refused {
		fmt.Fprintf(w, "  REFUSED %s: %s\n", termsafe.Sanitize(e.StoredSessionID), termsafe.Sanitize(e.Refused))
	}
}

// redactMigratePaths strips the home root out of every absolute store path in a
// success envelope, before it is rendered or marshalled.
func redactMigratePaths(res *history.MigrateResult) {
	for i := range res.Migrated {
		res.Migrated[i].Path = fsutil.RedactHome(res.Migrated[i].Path)
	}
	for i := range res.Refused {
		res.Refused[i].Path = fsutil.RedactHome(res.Refused[i].Path)
	}
	if res.Migrated == nil {
		res.Migrated = []history.MigrateEntry{}
	}
	if res.Refused == nil {
		res.Refused = []history.MigrateEntry{}
	}
}

// newHistoryIngestCommand builds `abcd history ingest`.
//
// The destination is named, not inferred: --into takes a repository root, and
// the verb always prints which repository it wrote into. Core refuses a
// destination it was not given, because a transcript stored in the wrong
// repository is redacted by the wrong repository's scanner configuration — a
// privacy fault rather than a misfiling.
func newHistoryIngestCommand(asJSON *bool) *cobra.Command {
	var into string
	var adopt []string
	cmd := &cobra.Command{
		Use:   "ingest [<path>...]",
		Short: "Redact and store transcripts already on disk into a named destination repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			dest, err := resolveDestination(into)
			if err != nil {
				return err
			}
			cfg, err := history.LoadConfig(dest.RepoRoot)
			if err != nil {
				return err
			}
			sources := args
			if len(sources) == 0 {
				sources = cfg.IngestRoots
			}
			if len(sources) == 0 {
				return fmt.Errorf("history ingest: name at least one source path, or declare ingest_roots in %s", history.ConfigRelPath)
			}
			opts := history.IngestOptions{
				Adopt:   append(append([]string(nil), cfg.AdoptProjects...), adopt...),
				Lineage: harnessLineage(sources),
			}
			res, err := history.Ingest(dest, sources, opts)
			if err != nil {
				return err
			}
			// `prompt` is handled HERE and nowhere else: core returned the
			// orphan list and wrote nothing, because an interactive question is
			// a transport concern and core has no transport.
			if cfg.OnOrphan == history.OnOrphanPrompt && len(res.Orphans) > 0 {
				chosen := askAdoptions(cmd, res.Orphans)
				if len(chosen) > 0 {
					opts.Adopt = append(opts.Adopt, chosen...)
					if res, err = history.Ingest(dest, sources, opts); err != nil {
						return err
					}
				}
			}
			redactIngestPaths(&res)
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				renderHistoryIngest(w, dest, res)
			})
		},
	}
	cmd.Flags().StringVar(&into, "into", "",
		"destination repository root (REQUIRED, no default; its own redaction configuration governs everything stored)")
	cmd.Flags().StringArrayVar(&adopt, "adopt", nil,
		"project directory name to claim for this run, in addition to adopt_projects (repeatable)")
	return cmd
}

// resolveDestination turns --into into the explicit destination core requires.
//
// There is deliberately NO default. Defaulting to the working directory would
// put back exactly the derivation the seam removes, one layer up: an operator
// recovering a backlog is not standing in the repository the transcripts belong
// to, and a transcript stored in the wrong repository is redacted by the wrong
// repository's scanner configuration. `--into .` is a fine answer; an unasked
// question is not.
func resolveDestination(into string) (history.Destination, error) {
	if into == "" {
		return history.Destination{}, fmt.Errorf("history ingest: --into <repo-root> is required and has no default — the destination repository's own configuration governs how its transcripts are redacted, so it is named, never inferred from the working directory (pass --into . to mean this one)")
	}
	abs, err := filepath.Abs(into)
	if err != nil {
		return history.Destination{}, err
	}
	det, err := ahoy.Detect(abs)
	if err != nil {
		return history.Destination{}, err
	}
	if det.RootSHA == "" {
		return history.Destination{}, fmt.Errorf("history ingest: %s is not a git repository with commits, so it has no store key",
			termsafe.Sanitize(fsutil.RedactHome(abs)))
	}
	return history.Destination{RepoRoot: abs, RootSHA: det.RootSHA}, nil
}

// askAdoptions asks the operator, once per distinct project, whether this
// repository claims it. An answer that is not an explicit yes is a no: the
// default for an orphan is to leave it alone.
func askAdoptions(cmd *cobra.Command, orphans []history.Orphan) []string {
	seen := map[string]int{}
	var order []string
	for _, o := range orphans {
		if _, ok := seen[o.Project]; !ok {
			order = append(order, o.Project)
		}
		seen[o.Project]++
	}
	err := cmd.ErrOrStderr()
	fmt.Fprintf(err, "abcd history ingest: %d transcript(s) belong to %d project(s) whose repository is not on this machine.\n",
		len(orphans), len(order))
	fmt.Fprintln(err, "Adopting one stores its transcripts HERE, redacted under THIS repository's configuration.")
	reader := bufio.NewReader(cmd.InOrStdin())
	var chosen []string
	for _, project := range order {
		fmt.Fprintf(err, "  adopt %q (%d transcripts)? [y/N] ", termsafe.Sanitize(project), seen[project])
		line, readErr := reader.ReadString('\n')
		answer := strings.ToLower(strings.TrimSpace(line))
		if answer == "y" || answer == "yes" {
			chosen = append(chosen, project)
		}
		if readErr != nil {
			// No more input: everything unanswered stays unadopted.
			fmt.Fprintln(err)
			break
		}
	}
	return chosen
}

// renderHistoryIngest writes the human report. The destination leads it, because the
// destination is the fact an operator most needs to be sure of.
func renderHistoryIngest(w io.Writer, dest history.Destination, res history.IngestResult) {
	var wrote int
	for _, c := range res.Captured {
		if c.Wrote {
			wrote++
		}
	}
	fmt.Fprintf(w, "abcd history ingest — into %s (root %s)\n",
		termsafe.Sanitize(fsutil.RedactHome(dest.RepoRoot)), dest.RootSHA)
	fmt.Fprintf(w, "  stored %d of %d owned transcript(s); %d skipped, %d orphaned, %d failed\n",
		wrote, len(res.Captured), len(res.Skipped), len(res.Orphans), len(res.Failed))
	for _, c := range res.Captured {
		state := "already stored"
		if c.Wrote {
			state = "stored"
		}
		who := termsafe.Sanitize(c.SessionID)
		if c.AgentID != "" {
			who += " agent " + termsafe.Sanitize(c.AgentID)
		}
		fmt.Fprintf(w, "  %-14s %s (via %s)\n", state, who, termsafe.Sanitize(c.Via))
	}
	for _, s := range res.Skipped {
		line := fmt.Sprintf("  skipped        %s: %s", termsafe.Sanitize(s.SessionID), termsafe.Sanitize(s.Reason))
		if s.RootSHA != "" {
			line += " (" + s.RootSHA + ")"
		}
		fmt.Fprintln(w, line)
	}
	for _, o := range res.Orphans {
		fmt.Fprintf(w, "  orphan         %s under project %s (recorded cwd %s) — ignored\n",
			termsafe.Sanitize(o.SessionID), termsafe.Sanitize(o.Project), termsafe.Sanitize(o.Cwd))
	}
	if len(res.Orphans) > 0 {
		fmt.Fprintf(w, "  an orphan's repository is not on this machine; claim its project in adopt_projects (%s) or pass --adopt to store it here\n",
			history.ConfigRelPath)
	}
	for _, f := range res.Failed {
		fmt.Fprintf(w, "  FAILED         %s: %s\n", termsafe.Sanitize(f.Path), termsafe.Sanitize(f.Err))
	}
}

// redactIngestPaths strips the home root out of every path in the envelope.
func redactIngestPaths(res *history.IngestResult) {
	for i := range res.Captured {
		res.Captured[i].Path = fsutil.RedactHome(res.Captured[i].Path)
		res.Captured[i].RecordPath = fsutil.RedactHome(res.Captured[i].RecordPath)
	}
	for i := range res.Skipped {
		res.Skipped[i].Path = fsutil.RedactHome(res.Skipped[i].Path)
	}
	for i := range res.Orphans {
		res.Orphans[i].Path = fsutil.RedactHome(res.Orphans[i].Path)
	}
	for i := range res.Failed {
		res.Failed[i].Path = fsutil.RedactHome(res.Failed[i].Path)
	}
	if res.Captured == nil {
		res.Captured = []history.Ingested{}
	}
	if res.Skipped == nil {
		res.Skipped = []history.IngestSkip{}
	}
	if res.Orphans == nil {
		res.Orphans = []history.Orphan{}
	}
	if res.Failed == nil {
		res.Failed = []history.IngestFailure{}
	}
}

// resolveSidecarRoots picks the directories a migration searches for the
// harness's per-agent metadata: what the operator passed, else what the
// repository declared. Both are configuration; neither is a path this binary
// knows.
func resolveSidecarRoots(repoRoot string, flagged []string) ([]string, error) {
	if len(flagged) > 0 {
		return flagged, nil
	}
	cfg, err := history.LoadConfig(repoRoot)
	if err != nil {
		return nil, err
	}
	return cfg.IngestRoots, nil
}

// harnessLineage builds the attribution ladder's first rung over a set of
// declared roots.
//
// Two ways in, cheapest first. When the caller holds the transcript file — as
// ingest does — the metadata file is derived from it by SUBSTITUTING the
// extension, which is the same one-step derivation the SubagentStop hook uses
// and involves no directory at all. When it does not — as a migration does not,
// because a stored record has no source path — an index of the declared roots
// is built once, by FILE NAME, so it finds the metadata wherever the harness
// nests it.
//
// It returns nil when no root was declared and there is nothing to build, so
// the caller can say so rather than silently recovering nothing.
func harnessLineage(roots []string) history.LineageLookup {
	var index map[string]string
	built := false
	return func(ref history.LineageRef) (history.HarnessLineage, bool) {
		if ref.SourcePath != "" {
			if side, ok := readHarnessAgentSidecar(ref.SourcePath); ok {
				return toHarnessLineage(side), true
			}
		}
		if ref.AgentID == "" {
			return history.HarnessLineage{}, false
		}
		if !built {
			index = indexHarnessSidecars(roots)
			built = true
		}
		path, ok := index[ref.AgentID]
		if !ok {
			return history.HarnessLineage{}, false
		}
		side, ok := parseHarnessAgentSidecar(path)
		if !ok {
			return history.HarnessLineage{}, false
		}
		return toHarnessLineage(side), true
	}
}

// toHarnessLineage converts the harness's file into the shape core takes.
func toHarnessLineage(side harnessAgentSidecar) history.HarnessLineage {
	return history.HarnessLineage{
		AgentType:      safeTextScalar(side.AgentType),
		ParentAgentID:  side.ParentAgentID,
		SpawnDepth:     side.SpawnDepth,
		SpawnToolUseID: safeTextScalar(side.ToolUseID),
	}
}

// indexHarnessSidecars maps agent id to metadata file across the declared
// roots, by filename and to a bounded depth.
//
// An agent id that two files claim is DROPPED rather than resolved to either:
// the id is the whole key, so two claimants mean the index cannot say which
// agent a record's lineage would come from, and a wrong agent type written into
// a record reads exactly like a right one.
func indexHarnessSidecars(roots []string) map[string]string {
	index := map[string]string{}
	ambiguous := map[string]struct{}{}
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return nil
			}
			segments := strings.Split(filepath.ToSlash(rel), "/")
			if d.IsDir() {
				if rel != "." && len(segments) >= sidecarIndexDepth {
					return fs.SkipDir
				}
				return nil
			}
			name := d.Name()
			if !d.Type().IsRegular() || !strings.HasPrefix(name, harnessSidecarPrefix) ||
				!strings.HasSuffix(name, harnessSidecarSuffix) {
				return nil
			}
			id := strings.TrimSuffix(strings.TrimPrefix(name, harnessSidecarPrefix), harnessSidecarSuffix)
			if id == "" {
				return nil
			}
			if prior, seen := index[id]; seen && prior != path {
				ambiguous[id] = struct{}{}
			}
			index[id] = path
			return nil
		})
	}
	for id := range ambiguous {
		delete(index, id)
	}
	return index
}

// parseHarnessAgentSidecar reads one metadata file. Every failure — absent,
// unreadable, not a regular file, unparseable, or carrying no spawn depth —
// means the rung did not answer, and none of them is an error.
func parseHarnessAgentSidecar(path string) (harnessAgentSidecar, bool) {
	data, err := fsutil.ReadGuarded(path, maxHarnessSidecarBytes)
	if err != nil {
		return harnessAgentSidecar{}, false
	}
	var side harnessAgentSidecar
	if err := json.Unmarshal(data, &side); err != nil {
		return harnessAgentSidecar{}, false
	}
	// A sub-agent is at depth 1 or deeper by definition. A file that does not
	// say so is not one this rung can read, whatever else it contains.
	if side.SpawnDepth <= 0 {
		return harnessAgentSidecar{}, false
	}
	return side, true
}
