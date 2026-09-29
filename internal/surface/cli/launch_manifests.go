package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// newLaunchLockstepCommand builds `abcd launch manifests`, the manifest lockstep
// checker's front door (itd-69): the check the preview and the cut run at the
// dev polarity and the payload render runs at the public one, over a tree the
// caller names, so a public checkout (a marketplace install, a release source
// archive) is checkable too. It reads and never writes, and it has no flag that
// waives a finding: manifest consistency cannot be waved through at its own
// layer.
//
// Exit codes: 0 consistent; 1 drift, one line per field; 2 an input that
// cannot be read (the version-location contract, a manifest, the artefact
// declaration) or an operand it does not know.
func newLaunchManifestsCommand(asJSON *bool) *cobra.Command {
	var tree, root string
	cmd := &cobra.Command{
		Use: "manifests --tree public|dev [--root <dir>]",
		Long: "Run the manifest lockstep check over a tree. --tree public requires the\n" +
			"version-location primary present as strict SemVer and every pinned secondary\n" +
			"to agree with it; --tree dev requires every version key absent (adr-19). The\n" +
			"tree is the working directory, or --root. Exit 0 consistent, 1 drift (one\n" +
			"line per field), 2 unreadable. Nothing is written.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if tree == "" {
				return &exitError{Code: 2, Msg: "abcd launch manifests: name the polarity with --tree public or --tree dev"}
			}
			t, err := launch.ParseLockstepTree(tree)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd launch manifests: " + err.Error()}
			}
			dir := root
			if dir == "" {
				if dir, err = os.Getwd(); err != nil {
					return &exitError{Code: 2, Msg: "abcd launch manifests: " + scrubPaths(err)}
				}
			}
			if dir, err = filepath.Abs(dir); err != nil {
				return &exitError{Code: 2, Msg: "abcd launch manifests: " + scrubPaths(err)}
			}
			res := launch.CheckTree(t, dir)
			if rerr := render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				renderLockstep(w, res)
			}); rerr != nil {
				return rerr
			}
			if res.ExitCode != 0 {
				return &exitError{Code: res.ExitCode}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tree, "tree", "", "the polarity to check: public (versions present and agreeing) or dev (versions absent)")
	cmd.Flags().StringVar(&root, "root", "", "the tree to check (default: the working directory)")
	return cmd
}

// renderLockstep prints one lockstep result for a person. Drift lines and the
// unreadable detail quote manifest values, so each is sanitised.
func renderLockstep(w io.Writer, res launch.LockstepResult) {
	switch {
	case res.Unreadable:
		fmt.Fprintf(w, "abcd launch manifests — tree %s: UNREADABLE\n  %s\n", res.Tree, termsafe.Sanitize(scrubMessage(res.Detail)))
	case len(res.Drifts) > 0:
		fmt.Fprintf(w, "abcd launch manifests — tree %s: DRIFT\n", res.Tree)
		for _, d := range res.Drifts {
			fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(scrubMessage(d)))
		}
	default:
		fmt.Fprintf(w, "abcd launch manifests — tree %s: consistent\n", res.Tree)
	}
}
