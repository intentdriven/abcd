package cli

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/guard"
)

// awayCopyRoot is the working copy a hook acts for: the session's start copy,
// as the session-start hook noted it (autonomy.ResolveCopy), else the worktree
// root of the payload's cwd; and nothing at all outside a checkout abcd
// manages (managedRoot). The stop check reads the copy's owed work from the
// same root (spc-2610180914469489, "Which copy"), so a branch a session cuts
// after changing into a worktree it made is recorded where the check looks.
func awayCopyRoot(cwd, sessionID string) (string, bool) {
	root, ok := managedRoot(cwd)
	if !ok {
		return "", false
	}
	if c, _, err := autonomy.ResolveCopy(root, sessionID, autonomy.Options{}); err == nil {
		return c, true
	}
	return root, true
}

// recordCreatedBranches is the created-branch recorder of the away mode
// (spc-2610180914469489 step 3): before the guard judges command, the
// branches it creates, checks out into a new worktree or renames are recorded
// under the session copy's key (awayCopyRoot) in the away store, whether or not the person
// is away. It refuses nothing: a store it cannot write leaves one line on w
// and the command goes on to the guard's own judgement.
func recordCreatedBranches(w io.Writer, cwd, sessionID string, workdir json.RawMessage, command string) {
	acts := guard.BranchActs(command)
	if len(acts) == 0 {
		return
	}
	copyRoot, ok := awayCopyRoot(cwd, sessionID)
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
