// Command scaffold-sync propagates the pinned action refs in the committed
// release workflows back into the scaffold templates they were rendered from,
// keeping TestSelfScaffoldParity satisfied after a dependency bump.
//
// It exists because the parity is one-directional in practice. Dependabot's
// github-actions ecosystem discovers `.github/workflows/*` and composite action
// manifests only — no ecosystem scans a .tmpl under internal/ — so every action
// bump it opens lands on the rendered workflow alone, breaks parity, and can
// never go green on its own (iss-209). This command is the other half of that
// loop, run by hand via `make scaffold-sync`. Nothing in CI invokes it: the
// drift is gated by TestSelfScaffoldParity and TestSyncRepoPinsIsCleanToday
// under preflight, and zero-touch propagation onto the dependabot branch is a
// recorded dead end (a workflow_run implementation was built and rejected in
// review — see iss-209).
//
// Re-rendering the template over the workflow would REVERT the bump and turn a
// red PR into a silently useless green one, so the flow is one-way by design.
//
// Exit codes: 0 nothing to do (or, with no -check, the sync was applied);
// 1 a fault; 3 with -check, templates are out of date and nothing was written.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/launch/scaffold"
	"github.com/intentdriven/abcd/internal/gitutil"
)

func main() {
	check := flag.Bool("check", false, "report drift and write nothing; exit 3 when a template is out of date")
	rootPath := flag.String("root", "", "repo root to sync (default: git toplevel, or cwd)")
	flag.Parse()

	root := *rootPath
	if root == "" {
		root = resolveRoot()
	}

	changed, err := scaffold.SyncRepoPins(root, !*check)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scaffold-sync:", scrubRoot(err.Error(), root))
		os.Exit(1)
	}
	if len(changed) == 0 {
		fmt.Println("scaffold-sync: templates already carry the committed workflow pins; nothing to do.")
		return
	}
	for _, rel := range changed {
		if *check {
			fmt.Printf("scaffold-sync: %s is out of date with the committed workflow pins\n", rel)
			continue
		}
		fmt.Printf("scaffold-sync: synced %s to the committed workflow pins\n", rel)
	}
	if *check {
		fmt.Fprintln(os.Stderr, "scaffold-sync: run `make scaffold-sync` to propagate the workflow pins into the templates.")
		os.Exit(3)
	}
}

// resolveRoot returns the git toplevel, falling back to the working directory.
// The env is scrubbed (gitutil.IsolatedEnv) so an inherited GIT_DIR/GIT_WORK_TREE
// cannot redirect discovery onto another tree; the os.Getwd fallback is the
// correct root under the Makefile/CI contract, so scrubbing introduces no
// regression.
func resolveRoot() string {
	if wd, err := os.Getwd(); err == nil {
		if top, err := gitutil.Toplevel(wd); err == nil {
			return top
		}
		return wd
	}
	return "."
}

// scrubRoot keeps an absolute checkout path out of the reported message, the way
// record-lint does: a fault names the repo-relative file, never where this clone
// happens to live on someone's disk.
func scrubRoot(msg, root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		return msg
	}
	return strings.ReplaceAll(strings.ReplaceAll(msg, abs+string(filepath.Separator), ""), abs, ".")
}
