package guard

import (
	"reflect"
	"testing"
)

// TestBranchActsForms: the branch-creating forms of git the created-branch
// recorder reads (spc-2610180914469489 step 3), each read as the branch it
// names, and the forms that create nothing read as nothing.
func TestBranchActsForms(t *testing.T) {
	cases := []struct {
		cmd  string
		want []BranchAct
	}{
		{"git worktree add -b feat/x ../x", []BranchAct{{Kind: BranchCreate, Name: "feat/x"}}},
		{"git worktree add -B feat/x ../x main", []BranchAct{{Kind: BranchCreate, Name: "feat/x"}}},
		{"git worktree add ../x -b feat/x", []BranchAct{{Kind: BranchCreate, Name: "feat/x"}}},
		{"git switch -c feat/y", []BranchAct{{Kind: BranchCreate, Name: "feat/y"}}},
		{"git switch --create=feat/y origin/main", []BranchAct{{Kind: BranchCreate, Name: "feat/y"}}},
		{"git switch -C feat/y", []BranchAct{{Kind: BranchCreate, Name: "feat/y"}}},
		{"git checkout -B feat/z", []BranchAct{{Kind: BranchCreate, Name: "feat/z"}}},
		{"git checkout -b feat/z main", []BranchAct{{Kind: BranchCreate, Name: "feat/z"}}},
		{"git checkout -bfeat/z", []BranchAct{{Kind: BranchCreate, Name: "feat/z"}}},
		{"git branch feat/w", []BranchAct{{Kind: BranchCreate, Name: "feat/w"}}},
		{"git branch --no-track feat/w origin/main", []BranchAct{{Kind: BranchCreate, Name: "feat/w"}}},
		{"git worktree add ../v", []BranchAct{{Kind: BranchCheckout, Name: "v"}}},
		{"git worktree add ../u feat/u", []BranchAct{{Kind: BranchCheckout, Name: "feat/u"}}},
		{"git branch -m feat/w feat/w2", []BranchAct{{Kind: BranchRename, From: "feat/w", Name: "feat/w2"}}},
		{"git branch -M feat/w2", []BranchAct{{Kind: BranchRename, From: "", Name: "feat/w2"}}},
		{"git -C ../other -c core.x=y checkout -b feat/q", []BranchAct{{Kind: BranchCreate, Name: "feat/q"}}},
		{"GIT_TRACE=0 git switch -c feat/e", []BranchAct{{Kind: BranchCreate, Name: "feat/e"}}},
		{"cd ../x && git switch -c feat/a && git branch feat/b", []BranchAct{{Kind: BranchCreate, Name: "feat/a"}, {Kind: BranchCreate, Name: "feat/b"}}},
		// Nothing created.
		{"git branch", nil},
		{"git branch -d feat/w", nil},
		{"git branch --list feat/*", nil},
		{"git branch -u origin/main feat/w", nil},
		{"git checkout feat/w", nil},
		{"git switch feat/w", nil},
		{"git worktree add --detach ../d", nil},
		{"git worktree list", nil},
		{"echo git checkout -b feat/no", nil},
		{"git checkout -b \"$(name)\"", nil},
	}
	for _, c := range cases {
		got := BranchActs(c.cmd)
		if len(got) == 0 && len(c.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("BranchActs(%q) = %+v, want %+v", c.cmd, got, c.want)
		}
	}
}
