package guard

import (
	"strings"
	"testing"
)

// TestApproveRefusesSettingCoreWorktree: core.worktree moves the working tree
// git reads and writes to another directory, outside the one the run's
// commands are judged in, so setting it, in the repository's file or for one
// command, is refused; reading it is not.
func TestApproveRefusesSettingCoreWorktree(t *testing.T) {
	r, _ := gitRig(t)
	refused := refusedAs(gitConfigWriteEntryID)
	r.run(t, []approveCase{
		{`git config core.worktree /tmp/elsewhere`, refused},
		{`git config set core.worktree ..`, refused},
		{`git config --local core.worktree ../x`, refused},
		{`git config CORE.WORKTREE /tmp/x`, refused},
		{`git -c core.worktree=/tmp/x checkout main`, refused},
		{`git config --get core.worktree`, nil},
		{`git config --unset core.worktree`, nil},
	})
	a := approveShell(clean(), r.files(), `git config core.worktree /tmp/elsewhere`)
	if !strings.Contains(a.Reason, "working tree") || strings.Contains(a.Reason, "starts a program") {
		t.Errorf("the refusal does not say what core.worktree does: %s", a.Reason)
	}
}
