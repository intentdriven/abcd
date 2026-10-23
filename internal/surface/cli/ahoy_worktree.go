package cli

// ahoy_worktree.go is the worktree store's front door (itd-2609091014076309,
// spc-2609301811532881): `add` makes a worktree in the repository's lane,
// bare `abcd ahoy worktree` and `list` render the repository's worktrees and
// its lane, `list --all` every lane on the machine, and `prune` reclaims the
// lane's merged, clean worktrees. The core judges each row and masks every string in it, since a
// branch, a lock reason and a path are another checkout's bytes; this door
// lays the rows out and maps a refusal or a fault to exit 2.

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// worktreeForge is the forge client the worktree verbs ask, once a run, for
// the merged judgement: the repository's own bounded `gh`. A test replaces it.
var worktreeForge worktree.GH = landing.GH

// worktreeGit is the git `add` hands the store: git through gitutil's
// isolated environment, as the spec's add runs it.
var worktreeGit worktree.GitRunner = func(dir string, args ...string) (string, error) {
	return gitutil.Run(dir, args...)
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

// newAhoyWorktreePruneCommand builds `ahoy worktree prune [--dry-run]`. It
// exits 0 when it reclaimed a worktree, 1 when the run completed and
// reclaimed nothing, and 2 on a refusal or a fault (decision 8); a dry run
// answers the code the real run would.
func newAhoyWorktreePruneCommand(asJSON *bool) *cobra.Command {
	var dryRun bool
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
			got, err := worktree.Prune(root, worktree.PruneOptions{DryRun: dryRun, Forge: worktreeForge, Held: loop.HeldWorktrees, Cwd: cwd})
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
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "archive and remove nothing; report what a real run would")
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
	}
	if got.ForgeNote != "" {
		fmt.Fprintf(w, "note: %s\n", got.ForgeNote)
	}
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
