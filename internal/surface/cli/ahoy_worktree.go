package cli

// ahoy_worktree.go is the worktree store's front door (itd-2609091014076309,
// spc-2609301811532881): bare `abcd ahoy worktree` and `list` render the
// repository's worktrees and its lane, `list --all` every lane on the
// machine. The core judges each row and masks every string in it, since a
// branch, a lock reason and a path are another checkout's bytes; this door
// lays the rows out and maps a refusal or a fault to exit 2.

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// worktreeForge is the forge client the worktree verbs ask, once a run, for
// the merged judgement: the repository's own bounded `gh`. A test replaces it.
var worktreeForge worktree.GH = landing.GH

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
	cmd.AddCommand(listCmd)
	return cmd
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
