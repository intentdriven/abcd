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
		{"if true; then git checkout -b feat/i; fi", []BranchAct{{Kind: BranchCreate, Name: "feat/i"}}},
		{"if ! git switch -c feat/j; then git branch feat/k; fi", []BranchAct{{Kind: BranchCreate, Name: "feat/j"}, {Kind: BranchCreate, Name: "feat/k"}}},
		{"{ git worktree add -b feat/l ../l; }", []BranchAct{{Kind: BranchCreate, Name: "feat/l"}}},
		{"for i in 1; do git branch -m feat/m; done", []BranchAct{{Kind: BranchRename, From: "", Name: "feat/m"}}},
		{"function f { git checkout -b feat/n; }", []BranchAct{{Kind: BranchCreate, Name: "feat/n"}}},
		{"FOO+=1 git switch -c feat/o", []BranchAct{{Kind: BranchCreate, Name: "feat/o"}}},
		{"coproc git branch feat/p", []BranchAct{{Kind: BranchCreate, Name: "feat/p"}}},
		// Nothing created.
		{"git branch", nil},
		{`for b in a b; do git checkout -b "$b"; done`, nil},
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
