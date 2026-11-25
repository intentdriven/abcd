package guard

import (
	"reflect"
	"testing"
)

// TestRecordActsReadsTheGitCommandsThatRecordOrSend: the git commands that
// make a commit or send one, read by position after git's global options, with
// a word nothing fixes allowed as such an option's value and after the
// subcommand and nowhere else (spc-2611251757228271 step 2).
func TestRecordActsReadsTheGitCommandsThatRecordOrSend(t *testing.T) {
	cases := []struct {
		cmd  string
		want []string
	}{
		{"git commit -m x", []string{"commit"}},
		{"git merge feat/x", []string{"merge"}},
		{"git rebase main", []string{"rebase"}},
		{"git cherry-pick abc", []string{"cherry-pick"}},
		{"git revert abc", []string{"revert"}},
		{"git am < x.patch", []string{"am"}},
		{"git pull --rebase", []string{"pull"}},
		{"git push", []string{"push"}},
		{"git commit -m \"$(cat <<'EOF'\nfeat: x\n\nAssisted-by: None\nEOF\n)\"", []string{"commit"}},
		{"git -c abcd.assistedBy=X:y revert abc", []string{"revert"}},
		{`git -C "$wt" commit -m x`, []string{"commit"}},
		{`git -C "$(pwd)" -c user.name="$n" commit -m x`, []string{"commit"}},
		{"cd /wt && git add -A && git commit -m x", []string{"commit"}},
		{"cat AGENTS.md && git commit -m x", []string{"commit"}},
		{"FOO=1 /usr/bin/git --no-pager push origin HEAD", []string{"push"}},
		{`FOO="$x" git push origin HEAD`, []string{"push"}},
		{"git commit -m x && git push", []string{"commit", "push"}},
		// No act.
		{"git status", nil},
		{"git add -A", nil},
		{"git stash push", nil},
		{"git log --grep=commit", nil},
		{"git log --grep commit", nil},
		{"git show push", nil},
		{"echo git commit", nil},
		{"git", nil},
		{"git -C", nil},
		{"git --version", nil},
		{"git $opts commit", nil},
		{`git "$opt" commit`, nil},
		{`git "--git-dir=$d" commit`, nil},
		{`git "$sub"`, nil},
		{"$g commit", nil},
		{"env GIT_AUTHOR_NAME=x git commit -m x", nil},
		{"sh -c 'git commit -m x'", nil},
		{"gitk commit", nil},
		{"git commit -m 'unterminated", nil},
		{"", nil},
	}
	for _, c := range cases {
		got := RecordActs(c.cmd)
		if len(got) == 0 && len(c.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("RecordActs(%q) = %v, want %v", c.cmd, got, c.want)
		}
	}
}
