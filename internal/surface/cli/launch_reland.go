package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/board"
	"github.com/intentdriven/abcd/internal/core/launch/reland"
	"github.com/intentdriven/abcd/internal/core/launch/scaffold"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// relandForge is the forge `abcd launch reland` asks; a test stands in its own.
var relandForge = reland.DefaultForge()

// relandFetch is the networked git the reland fetches with; nil is gitutil.Net.
var relandFetch func(dir string, args ...string) (string, error)

// newLaunchRelandCommand builds `abcd launch reland` (iss-2611220952509874):
// the person's way to land a bot's dependency bump while the
// dependency-reauthor workflow cannot re-author it. Bare, it lists the open
// bumps waiting on it; with a pull request, it relands that one as the
// checkout's own commit on a local branch, inside the workflow's own bound.
func newLaunchRelandCommand(asJSON *bool) *cobra.Command {
	var assisted string
	cmd := &cobra.Command{
		Use: "reland [<pull request>] [--assisted-by <Vendor>:<model-version>]",
		Long: "Reland a bot's dependency pull request as your own commit, the hand step the\n" +
			"dependency-reauthor workflow leaves to a person while the owner in\n" +
			scaffold.ReauthorConfPath + " or its App is unset.\n\n" +
			"Bare, it asks the forge for the open pull requests a declared bot opened from a\n" +
			"declared branch prefix, and lists them; it writes nothing.\n\n" +
			"With a pull request, it fetches origin's default branch and the pull request's\n" +
			"branch, and judges the pull request with the workflow's own script against the\n" +
			"declaration the default branch stores: the bot, the branch prefix, the run's\n" +
			"actor (the account whose push last moved the branch, from the forge's activity\n" +
			"record), exactly one commit by the bot, and only the row's manifests at the row's\n" +
			"directory. What passes is applied onto the default branch's tip as stored, writing\n" +
			"no working tree, and committed under this checkout's git identity on the local\n" +
			"branch reland/pr-<number>, with \"Supersedes #<number>\" in its message. A machine\n" +
			"identity is refused. A change that does not apply cleanly is refused: redo the bump\n" +
			"with the ecosystem's own tool (go get, for example).\n\n" +
			"The commit carries Assisted-by: None, or Assisted-by: <Vendor>:<model-version>\n" +
			"from --assisted-by, which an agent's session must pass. It pushes nothing and opens\n" +
			"nothing: switch to the branch and run abcd land. It is a person's act, so the\n" +
			"workflow never runs it.\n\n" +
			"Exit 0: relanded, or listed. Exit 1: refused or the forge could not answer;\n" +
			"nothing was written. Exit 2: usage.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := gitutil.CheckoutRoot(cwd, "the reland")
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd launch reland: " + scrubPaths(err)}
			}
			if len(args) == 0 {
				return relandList(cmd.OutOrStdout(), *asJSON, root)
			}
			n, err := strconv.Atoi(strings.TrimPrefix(args[0], "#"))
			if err != nil || n <= 0 {
				return &exitError{Code: 2, Msg: "abcd launch reland: the pull request is a number, such as 1618"}
			}
			if assisted == "" {
				for _, k := range driveSessionMarkers {
					if os.Getenv(k) != "" {
						return &exitError{Code: 2, Msg: "abcd launch reland: " + k + " is set, so an agent is running this; " +
							"pass --assisted-by <Vendor>:<model-version> so the commit discloses it (nothing was written)"}
					}
				}
			}
			res, err := reland.Reland(reland.Options{Root: root, PR: n, AssistedBy: assisted, Forge: relandForge, Fetch: relandFetch})
			var refused *reland.Refusal
			switch {
			case errors.As(err, &refused):
				return &exitError{Code: 1, Msg: "abcd launch reland: refused: " +
					termsafe.Sanitize(fsutil.RedactHome(refused.Reason)) + " (nothing was written)"}
			case err != nil && strings.Contains(err.Error(), "--assisted-by"):
				return &exitError{Code: 2, Msg: "abcd launch reland: " + termsafe.Sanitize(err.Error())}
			case err != nil:
				return &exitError{Code: 1, Msg: "abcd launch reland: " + scrubPaths(err) + " (nothing was written)"}
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintf(w, "abcd launch reland — relanded #%d as your own commit\n", res.PR)
				fmt.Fprintf(w, "  bound:   %s\n", termsafe.Sanitize(res.Bound))
				fmt.Fprintf(w, "  onto:    %s at %s\n", termsafe.Sanitize(res.Default), shortSHA(res.Base))
				fmt.Fprintf(w, "  branch:  %s at %s, local only: nothing was pushed\n", res.Branch, shortSHA(res.Commit))
				fmt.Fprintf(w, "  author:  %s\n", termsafe.Sanitize(res.Author))
				fmt.Fprintf(w, "  next:    %s\n", termsafe.Sanitize(res.Next))
			})
		},
	}
	cmd.Flags().StringVar(&assisted, "assisted-by", "",
		"the model that assisted, as <Vendor>:<model-version>, for the commit's Assisted-by trailer (an agent's session must pass it)")
	return cmd
}

// relandList renders the bare listing: the open bot bumps the forge names
// under a declared row.
func relandList(out io.Writer, asJSON bool, root string) error {
	w, err := reland.List(root, relandForge)
	if errors.Is(err, reland.ErrNoDeclaration) {
		empty := reland.Waiting{Bumps: []reland.PullRequest{}}
		return render(out, asJSON, empty, func(w io.Writer) {
			fmt.Fprintf(w, "abcd launch reland — this repository declares no dependency re-authoring (%s), so no bump waits on a reland\n",
				scaffold.ReauthorConfPath)
		})
	}
	if err != nil {
		return &exitError{Code: 1, Msg: "abcd launch reland: " + scrubPaths(err)}
	}
	return render(out, asJSON, w, func(out io.Writer) {
		owner := "the owner is unset in " + scaffold.ReauthorConfPath + ", so the workflow refuses each"
		if w.Declaration.OwnerSet {
			owner = "the workflow re-authors each once its App's two secrets are set"
		}
		fmt.Fprintf(out, "abcd launch reland — %s open from a declared bot (%s)\n", countOf(len(w.Bumps), "bump"), owner)
		for _, p := range w.Bumps {
			fmt.Fprintf(out, "  #%d %s — %s\n", p.Number, termsafe.Sanitize(p.HeadRef), termsafe.Sanitize(p.Title))
			fmt.Fprintf(out, "    abcd launch reland %d\n", p.Number)
		}
		if w.Truncated {
			fmt.Fprintln(out, "  the forge's first page was full, so more may be open than are listed")
		}
	})
}

// boardBumpsMax is how many waiting branches the board names.
const boardBumpsMax = 5

// boardBumps is the board's dependency-bumps member, from disk alone
// (adr-38): present when the repository declares re-authoring with no owner
// and origin's branches under a declared prefix, as last fetched, wait on a
// person. A declaration that cannot be read omits it and says why on stderr.
func boardBumps(cwd string, stderr io.Writer) *reland.Fetched {
	root, err := gitutil.CheckoutRoot(cwd, "the board")
	if err != nil {
		return nil
	}
	f, err := reland.AsFetched(root)
	if errors.Is(err, reland.ErrNoDeclaration) {
		return nil
	}
	if err != nil {
		fmt.Fprintf(stderr, "abcd: the dependency bumps line is omitted — %s\n", termsafe.Sanitize(fsutil.RedactHome(err.Error())))
		return nil
	}
	if f.OwnerSet || len(f.Branches) == 0 {
		return nil
	}
	return &f
}

// bumpsRow is the facilitator's row for the waiting bumps, nil when none wait.
func bumpsRow(f *reland.Fetched) *board.Row {
	if f == nil {
		return nil
	}
	noun := "bot dependency branches"
	if len(f.Branches) == 1 {
		noun = "bot dependency branch"
	}
	r := &board.Row{Label: "bumps", Text: fmt.Sprintf("%d %s as last fetched, with no re-authoring owner — "+
		"`abcd launch reland` lists and relands them", len(f.Branches), noun)}
	for i, b := range f.Branches {
		if i == boardBumpsMax {
			r.Items = append(r.Items, fmt.Sprintf("and %d more", len(f.Branches)-boardBumpsMax))
			break
		}
		r.Items = append(r.Items, termsafe.Sanitize(b))
	}
	return r
}
