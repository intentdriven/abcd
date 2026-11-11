package cli

// ahoy_worktree.go is the worktree store's front door (itd-2609091014076309,
// spc-2609301811532881): `add` makes a worktree in the repository's lane,
// bare `abcd ahoy worktree` and `list` render the repository's worktrees and
// its lane, `list --all` every lane on the machine, and `prune` reclaims the
// lane's merged, clean worktrees and lists its quiet ones, reclaiming a quiet
// one only when --yes names it. The core judges each row and masks every string in it, since a
// branch, a lock reason and a path are another checkout's bytes; this door
// lays the rows out and maps a refusal or a fault to exit 2.

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// worktreeForge is the forge client the worktree verbs ask, once a run, for
// the merged judgement: the repository's own bounded `gh`. A test replaces it.
var worktreeForge worktree.GH = landing.GH

// worktreeNow is prune's clock: the quiet window is measured back from it,
// and the notes archive is stamped with it. A test moves it on.
var worktreeNow = time.Now

// worktreeTrimCache is the trim of the Go build cache prune and trim-cache run.
// TestMain stands in for it across this package's tests, so none trims the
// machine's own cache.
var worktreeTrimCache = worktree.TrimGoCache

// worktreeAddTimeout bounds add's `git worktree add`: gitutil.CheckoutBound,
// the bound the build loop's lane checkout has, since a checkout's time scales
// with the tree and a Git LFS download. A var so a test can shorten it.
var worktreeAddTimeout = gitutil.CheckoutBound

// worktreeGit is the git `add` hands the store for `git worktree add`, which
// checks a working tree out as the build loop's lanes do:
// gitutil.WorkingTreeCommitCommand, so the content filters in the person's
// global config apply (iss-2610111729375108) and the repository's
// post-checkout hook, its signing and its signature checks run as on the
// person's own `git worktree add` (iss-2610111729399920). Since those
// programs run, it is bounded, and a hook that waits is stopped with git, so
// git removes its lock files. The add's reads, whose answers abcd parses,
// stay isolated in the core.
var worktreeGit worktree.GitRunner = func(dir string, args ...string) (string, error) {
	out, err := gitutil.RunCommitCommandBounded(dir, worktreeAddTimeout, gitutil.ErrCheckoutStalled, args...)
	return strings.TrimSpace(out), err
}

// worktreeFault is a refusal or a fault of a worktree verb: exit 2, its text
// home-redacted and masked.
func worktreeFault(verb string, err error) error {
	return &exitError{Code: 2, Msg: "abcd ahoy worktree " + verb + ": " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
}

// worktreeCheckout is the checkout the caller stands in.
func worktreeCheckout() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return gitutil.Toplevel(cwd)
}

// newAhoyWorktreeCommand builds `ahoy worktree [list [--all]]`.
func newAhoyWorktreeCommand(asJSON *bool) *cobra.Command {
	list := func(cmd *cobra.Command, all bool) error {
		if all {
			// The registry's labels have no exported reader yet, so every
			// lane is shown under its root commit.
			got, err := worktree.ListAll(worktree.AllOptions{Here: hereOrNone(), Forge: worktreeForge})
			if err != nil {
				return worktreeFault("list", err)
			}
			return render(cmd.OutOrStdout(), *asJSON, got, func(w io.Writer) { renderAllLanes(w, got) })
		}
		root, err := worktreeCheckout()
		if err != nil {
			return worktreeFault("list", err)
		}
		got, err := worktree.List(root, worktree.ListOptions{Forge: worktreeForge})
		if err != nil {
			return worktreeFault("list", err)
		}
		return render(cmd.OutOrStdout(), *asJSON, got, func(w io.Writer) {
			fmt.Fprintf(w, "lane %s\n", got.Lane)
			renderWorktreeRows(w, got.Worktrees)
			if got.ForgeNote != "" {
				fmt.Fprintf(w, "note: %s\n", got.ForgeNote)
			}
		})
	}
	cmd := &cobra.Command{
		Use:   "worktree",
		Short: "List the worktrees of this repository and its lane in the worktree store",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return list(cmd, false) },
	}
	var all bool
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List the worktrees of this repository and its lane, or with --all every lane",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return list(cmd, all) },
	}
	listCmd.Flags().BoolVar(&all, "all", false, "walk every lane in the worktree store on this machine")
	cmd.AddCommand(newAhoyWorktreeAddCommand(asJSON))
	cmd.AddCommand(listCmd)
	cmd.AddCommand(newAhoyWorktreePruneCommand(asJSON))
	cmd.AddCommand(newAhoyWorktreeTrimCacheCommand(asJSON))
	return cmd
}

// worktreeAdded is what `add` reports: the worktree's name, its branch and the
// path to work in, home-redacted.
type worktreeAdded struct {
	Name   string `json:"name"`
	Branch string `json:"branch"`
	Path   string `json:"path"`
}

// newAhoyWorktreeAddCommand builds `ahoy worktree add <name> [--branch
// <existing>]`. Without --branch it makes a new branch named <name>, cut from
// the default branch as the build loop's lanes are; with it, it checks that
// existing local branch out and makes none. It prints the path alone, so a
// caller can work in it at once, and exits 2 on any refusal or fault.
func newAhoyWorktreeAddCommand(asJSON *bool) *cobra.Command {
	var branch string
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a worktree in this repository's lane of the worktree store",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := worktreeCheckout()
			if err != nil {
				return worktreeFault("add", err)
			}
			name := args[0]
			opts := worktree.AddOptions{Name: name, Git: worktreeGit}
			if cmd.Flags().Changed("branch") {
				opts.Branch, opts.Existing = branch, true
			} else {
				if err := branchTaken(root, name); err != nil {
					return worktreeFault("add", err)
				}
				// No default branch, or none git can name, is no base: the
				// core refuses that, after the repository's own refusals.
				if def := gitutil.DefaultRef(root); def != "" {
					opts.Base, _ = gitutil.Run(root, "rev-parse", "--verify", "--quiet", def+"^{commit}", "--")
				}
			}
			got, err := worktree.Add(root, opts)
			if err != nil {
				return worktreeFault("add", err)
			}
			out := worktreeAdded{
				Name:   termsafe.Sanitize(name),
				Branch: termsafe.Sanitize(got.Branch),
				Path:   termsafe.Sanitize(fsutil.RedactHome(got.Path)),
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { fmt.Fprintln(w, out.Path) })
		},
	}
	cmd.Flags().StringVar(&branch, "branch", "", "check out this existing local branch instead of making a new one")
	return cmd
}

// branchTaken refuses a new branch whose name a local branch already holds,
// naming the flag that checks it out. The core refuses the same case without
// knowing this door's flags, and refuses a taken path in the lane first, so
// this answers only for a valid name whose place in the lane is free.
func branchTaken(root, name string) error {
	if !worktree.ValidName(name) {
		return nil
	}
	if _, err := gitutil.Run(root, "rev-parse", "--verify", "--quiet", "refs/heads/"+name, "--"); err != nil {
		return nil
	}
	lane, err := worktree.LaneFor(root)
	if err != nil {
		return nil
	}
	if _, err := os.Lstat(lane.Path(name)); err == nil {
		return nil
	}
	return fmt.Errorf("a branch named %s already exists; check it out with `--branch %s`", name, name)
}

// newAhoyWorktreePruneCommand builds `ahoy worktree prune [--dry-run] [--yes
// <name>]…`. It lists each quiet candidate with its dossier, and reclaims one
// only when --yes names it. It exits 0 when it reclaimed a worktree, 1 when
// the run completed and reclaimed nothing, and 2 on a refusal or a fault, a
// quiet window the configuration refuses and a --yes name the lane does not
// hold among them (decision 8); a dry run answers the code the real run
// would. When it removed a worktree it also trims the Go build cache and says
// what went (worktree.TrimGoCache); what the trim finds never changes the code.
func newAhoyWorktreePruneCommand(asJSON *bool) *cobra.Command {
	var dryRun bool
	var yes []string
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Reclaim the merged, clean worktrees of this repository's lane, archiving their notes first",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return worktreeFault("prune", err)
			}
			root, err := gitutil.Toplevel(cwd)
			if err != nil {
				return worktreeFault("prune", err)
			}
			roots, notes := layered.RootsFor(root)
			for _, n := range notes {
				fmt.Fprintln(cmd.ErrOrStderr(), noteLine(n))
			}
			days, err := worktree.QuietDays(roots)
			if err != nil {
				return worktreeFault("prune", err)
			}
			got, err := worktree.Prune(root, worktree.PruneOptions{
				DryRun: dryRun, Forge: worktreeForge, Held: loop.HeldWorktrees, Cwd: cwd, Now: worktreeNow,
				QuietDays: days, Yes: yes, TrimCache: worktreeTrimCache,
			})
			if err != nil {
				return worktreeFault("prune", err)
			}
			if err := render(cmd.OutOrStdout(), *asJSON, got, func(w io.Writer) { renderPruned(w, got) }); err != nil {
				return err
			}
			if got.Reclaimed() == 0 {
				return &exitError{Code: 1}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "archive, remove and trim nothing; report what a real run would")
	cmd.Flags().StringArrayVar(&yes, "yes", nil, "reclaim this quiet candidate too (repeatable); a name not in the lane refuses the run")
	return cmd
}

// renderPruned lays out one line a worktree: its outcome, then why it was
// kept or where its notes went.
func renderPruned(w io.Writer, got worktree.Pruned) {
	head := "lane " + got.Lane
	if got.DryRun {
		head += " (dry run: nothing archived or removed)"
	}
	fmt.Fprintln(w, head)
	if len(got.Rows) == 0 {
		fmt.Fprintln(w, "  (no worktrees)")
	}
	for _, r := range got.Rows {
		line := "  " + r.Outcome + "  " + r.Name + "  " + r.Path
		switch {
		case r.Reason != "":
			line += "; " + r.Reason
		case r.Archive != "":
			line += "; notes (" + r.Notes + ") archived to " + r.Archive
		case r.Notes != "":
			line += "; notes: " + r.Notes
		}
		fmt.Fprintln(w, line)
		if r.Dossier != nil {
			renderDossier(w, *r.Dossier)
		}
	}
	if got.ForgeNote != "" {
		fmt.Fprintf(w, "note: %s\n", got.ForgeNote)
	}
	if got.CacheTrim != nil {
		fmt.Fprintln(w, cacheTrimLine(*got.CacheTrim, got.DryRun))
	}
}

// newAhoyWorktreeTrimCacheCommand builds `ahoy worktree trim-cache [--dry-run]`:
// the same trim prune runs, with no worktree to remove first. It is what a lane's
// start runs detached (kickCacheTrim), so a real run saves its answer for the next
// lane to read. It exits 0 when the cache was walked with nothing refused, a cache
// left over its limit by what it must keep among them, and 1 when it was skipped
// or an entry could not be read or removed.
func newAhoyWorktreeTrimCacheCommand(asJSON *bool) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "trim-cache",
		Short: "Keep the Go build cache under its limit, oldest entries first",
		Long: "Delete the oldest entries of the Go build cache until the rest fit its limit, keeping\n" +
			"every entry used in the last 3 hours whatever the limit; if those alone exceed it,\n" +
			"they stay and the line says so. The limit is `" + worktree.CacheLimitKey + "` in\n" +
			"`" + abcdhome.Display("config.json") + "`, a whole number of GiB, 100 by default, and is never read from a\n" +
			"repository; a value it cannot use, a file it cannot read, or a key under `load`\n" +
			"that nothing reads (a misspelt one among them) deletes nothing.\n\n" +
			"It acts only on the directory `go env GOCACHE` reports, and only when that is a real\n" +
			"directory this account owns, has a Go build cache's shape and is not the home or above\n" +
			"it, and there only on the entries Go's own layout defines. A build running beside it\n" +
			"finds a miss and rebuilds, and an entry a build used since the walk stays, since each\n" +
			"one is read again just before it goes; a build that looks one up in the instant\n" +
			"between that read and the unlink can still fail to open it, and a rerun rebuilds it.\n" +
			"Judge it by the cache's size: a local snapshot can hold\n" +
			"what it deleted, so free disk can lag.\n\n" +
			"The build loop runs this detached when it makes a lane, at most once an hour\n" +
			"(`" + abcdhome.Display("go-cache-trim.json") + "`), and the next lane says on stderr what it could not do.\n" +
			"A dry run deletes nothing and records nothing. Exits 0 when the cache was walked and\n" +
			"nothing refused, 1 when it was skipped or an entry could not be read or removed.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			started := worktreeNow()
			got := worktreeTrimCache(dryRun, started)
			if !dryRun {
				home, err := abcdhome.MachineHome()
				if err == nil {
					err = worktree.SaveCacheTrim(home, started, got)
				}
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "abcd: the Go build cache's trim could not be recorded: %s\n", shownCacheTrimText(err.Error()))
				}
			}
			if err := render(cmd.OutOrStdout(), *asJSON, got, func(w io.Writer) { fmt.Fprintln(w, cacheTrimLine(got, dryRun)) }); err != nil {
				return err
			}
			if got.Skipped != "" || got.Failed > 0 {
				return &exitError{Code: 1}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "delete nothing; report what a real run would")
	return cmd
}

// cacheTrimLine is the one line saying what the Go build cache's trim did, or
// would do under a dry run, or why it was skipped.
func cacheTrimLine(t worktree.CacheTrim, dryRun bool) string {
	if t.Skipped != "" {
		return "go build cache not trimmed: " + t.Skipped
	}
	entries := fmt.Sprintf("%d entries", t.EntriesRemoved)
	if t.EntriesRemoved == 1 {
		entries = "1 entry"
	}
	freed, left, limit := cacheSizeText(t.BytesFreed), cacheSizeText(t.LeftBytes), cacheSizeText(t.LimitBytes)
	line := fmt.Sprintf("go build cache %s: removed %s, %s freed, %s left of a limit of %s", t.Path, entries, freed, left, limit)
	if dryRun {
		line = fmt.Sprintf("go build cache %s: would remove %s, freeing %s, leaving %s of a limit of %s", t.Path, entries, freed, left, limit)
	}
	if t.OverLimit {
		line += fmt.Sprintf("; still over the limit, since entries used in the last %d hours stay", t.KeepHours)
	}
	if t.Failed > 0 {
		line += fmt.Sprintf("; %d could not be read or removed", t.Failed)
	}
	return line
}

// cacheSizeText renders a size in GiB, as the limit is set, or small ones as they
// are.
func cacheSizeText(n uint64) string {
	if n < 1<<30 {
		return humanBytes(int(n))
	}
	return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
}

// shownCacheTrimText is text bound for the terminal: home-redacted and masked.
func shownCacheTrimText(s string) string { return termsafe.Sanitize(fsutil.RedactHome(s)) }

// renderDossier lays out a quiet candidate's dossier under its row, in the
// spec's order: dirty state first, so uncommitted work is the first thing
// read.
func renderDossier(w io.Writer, d worktree.Dossier) {
	dirty := "none (the tree is clean)"
	if n := d.Dirty.Modified + d.Dirty.Untracked; n > 0 {
		dirty = fmt.Sprintf("%d modified, %d untracked: %s", d.Dirty.Modified, d.Dirty.Untracked, strings.Join(d.Dirty.Paths, ", "))
		if more := n - len(d.Dirty.Paths); more > 0 {
			dirty += fmt.Sprintf(" and %d more", more)
		}
	}
	fmt.Fprintf(w, "      dirty: %s\n", dirty)
	if d.Ignored.Unread {
		fmt.Fprintf(w, "      ignored: could not be listed; any go with it, unarchived\n")
	} else if n := d.Ignored.Count; n > 0 {
		line := fmt.Sprintf("%d removed with it, unarchived: %s", n, strings.Join(d.Ignored.Paths, ", "))
		if more := n - len(d.Ignored.Paths); more > 0 {
			line += fmt.Sprintf(" and %d more", more)
		}
		fmt.Fprintf(w, "      ignored: %s\n", line)
	}
	fmt.Fprintf(w, "      commits: %d not on the default branch\n", d.Commits)
	for _, s := range d.Subjects {
		fmt.Fprintf(w, "        %s\n", s)
	}
	if more := d.Commits - len(d.Subjects); more > 0 {
		fmt.Fprintf(w, "        and %d more\n", more)
	}
	if d.Diff != "" {
		fmt.Fprintf(w, "      diff: %s\n", d.Diff)
	}
	if len(d.Records) > 0 {
		fmt.Fprintf(w, "      records: %s\n", strings.Join(d.Records, ", "))
	}
	fmt.Fprintf(w, "      last activity: %s\n", d.LastActivity)
	fmt.Fprintf(w, "      next: %s\n", d.Recommendation)
}

// hereOrNone is the checkout the caller stands in, or "" outside one: --all
// reads the whole store from anywhere.
func hereOrNone() string {
	root, err := worktreeCheckout()
	if err != nil {
		return ""
	}
	return root
}

// renderWorktreeRows lays out one line a row: a judged row's state columns,
// or the reason it is not judged, then git's lock and prune reasons.
func renderWorktreeRows(w io.Writer, rows []worktree.Row) {
	if len(rows) == 0 {
		fmt.Fprintln(w, "  (no worktrees)")
	}
	for _, r := range rows {
		branch := r.Branch
		if branch == "" {
			branch = "-"
		}
		cols := []string{r.Name, branch}
		if r.Reason == "" {
			merged := r.Merged
			if r.JudgedBy != "" {
				merged += " (" + r.JudgedBy + ")"
			}
			cols = append(cols, r.State, merged)
		}
		cols = append(cols, r.Where, r.Path)
		line := "  " + strings.Join(cols, "  ")
		if r.Reason != "" {
			line += "; " + r.Reason
		}
		if r.Locked {
			line += "  locked"
			if r.LockReason != "" {
				line += ": " + r.LockReason
			}
		}
		if r.Prunable && r.PrunableReason != "" {
			line += "  prunable: " + r.PrunableReason
		}
		fmt.Fprintln(w, line)
	}
}

// renderAllLanes lays out --all: each lane under its label, its reason when
// it is not read, and its rows.
func renderAllLanes(w io.Writer, got worktree.AllListing) {
	fmt.Fprintf(w, "store %s\n", got.Store)
	if len(got.Lanes) == 0 {
		fmt.Fprintln(w, "  (no lanes)")
	}
	for _, l := range got.Lanes {
		head := l.Label
		if l.Label != l.Key {
			head += " (" + l.Key + ")"
		}
		if l.Reason != "" {
			fmt.Fprintf(w, "%s: %s\n", head, l.Reason)
		} else {
			fmt.Fprintln(w, head)
		}
		if len(l.Worktrees) > 0 {
			renderWorktreeRows(w, l.Worktrees)
		}
	}
	if got.ForgeNote != "" {
		fmt.Fprintf(w, "note: %s\n", got.ForgeNote)
	}
}
