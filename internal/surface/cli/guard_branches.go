package cli

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// awayCopyRoot is the working copy a hook acts for: the worktree root of the
// session's directory. The stop check reads the copy's owed work from the same
// root (spc-2610180914469489, "Which copy").
func awayCopyRoot(cwd string) (string, bool) {
	root, err := gitutil.CheckoutRoot(cwd, autonomy.StoreName)
	if err != nil {
		return "", false
	}
	return root, true
}

// recordCreatedBranches is the created-branch recorder of the away mode
// (spc-2610180914469489 step 3): before the guard judges command, the
// branches it creates, checks out into a new worktree or renames are recorded
// under the session copy's key in the away store, whether or not the person
// is away. It refuses nothing: a store it cannot write leaves one line on w
// and the command goes on to the guard's own judgement.
func recordCreatedBranches(w io.Writer, cwd string, workdir json.RawMessage, command string) {
	acts := guard.BranchActs(command)
	if len(acts) == 0 {
		return
	}
	copyRoot, ok := awayCopyRoot(cwd)
	if !ok {
		return
	}
	runDir := cwd
	if wd, err := hookWorkdir(workdir, cwd); err == nil && wd.Exists {
		runDir = wd.Path
	}
	changes := make([]autonomy.BranchChange, 0, len(acts))
	var names []string
	for _, a := range acts {
		c := autonomy.BranchChange{Name: a.Name, From: a.From}
		switch a.Kind {
		case guard.BranchCreate:
			c.Kind = autonomy.BranchCreated
		case guard.BranchCheckout:
			c.Kind = autonomy.BranchCheckedOut
		case guard.BranchRename:
			c.Kind = autonomy.BranchMoved
		default:
			continue
		}
		changes = append(changes, c)
		names = append(names, a.Name)
	}
	if err := autonomy.RecordCreated(copyRoot, runDir, changes, autonomy.Options{}); err != nil {
		diagnosticLine(w, "abcd guard: the created branch %s was not recorded for the away mode (%s); the command is not refused for it",
			strings.Join(names, ", "), scrubPaths(err))
	}
}
