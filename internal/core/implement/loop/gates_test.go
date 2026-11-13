package loop

import (
	"strings"
	"testing"
)

// TestAHooksPathThatCannotBeReadIsNotNone: the hooks path a drive's gate
// check reads is none only where git says the key is not set; a read git
// fails is an error naming it, never "no in-tree hooks directory". Watched
// failing: the failed read answered none.
func TestAHooksPathThatCannotBeReadIsNotNone(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	if p, err := DriveHooksPath(repo.Root()); p != "" || err != nil {
		t.Fatalf("no core.hooksPath is none: %q %v", p, err)
	}
	repo.Git("config", "core.hooksPath", ".githooks")
	if p, err := DriveHooksPath(repo.Root()); p != ".githooks" || err != nil {
		t.Fatalf("a relative core.hooksPath is named: %q %v", p, err)
	}
	standInGit(t, `*"--get-all core.hooksPath"*) echo "fatal: bad config line 3 in file .git/config" >&2; exit 128 ;;`)
	p, err := DriveHooksPath(repo.Root())
	if err == nil || p != "" || !strings.Contains(err.Error(), "git config --type=path --get-all core.hooksPath") {
		t.Fatalf("a read git failed is an error naming it: %q %v", p, err)
	}
}
