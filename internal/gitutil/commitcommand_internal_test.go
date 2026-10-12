package gitutil

import (
	"slices"
	"testing"
)

// TestWorkingTreeCommitCommandLeavesTheRepositorysCommitSettings is
// iss-2610111729399920: a commit abcd makes leaves the repository's hooks,
// commit signing and merge signature check to the repository, and keeps every
// other pin WorkingTreeCommand carries; WorkingTreeCommand keeps all of them.
func TestWorkingTreeCommitCommandLeavesTheRepositorysCommitSettings(t *testing.T) {
	commit := WorkingTreeCommitCommand("/repo", "commit").Args
	tree := WorkingTreeCommand("/repo", "status").Args
	for _, pin := range []string{"core.hooksPath=/dev/null", "commit.gpgsign=false", "merge.verifySignatures=false"} {
		if slices.Contains(commit, pin) {
			t.Errorf("the commit command pins %s", pin)
		}
		if !slices.Contains(tree, pin) {
			t.Errorf("the working-tree command lost %s", pin)
		}
	}
	for _, pin := range []string{"core.fsmonitor=false", "log.showSignature=false", "diff.submodule=short", "gc.auto=0", "maintenance.auto=false", "core.quotePath=false"} {
		if !slices.Contains(commit, pin) {
			t.Errorf("the commit command lost %s", pin)
		}
	}
	if got := commit[len(commit)-3:]; !slices.Equal(got, []string{"-C", "/repo", "commit"}) {
		t.Errorf("the commit command ends %q", got)
	}
}
