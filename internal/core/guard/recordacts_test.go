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
		{"if ! git diff --cached --quiet; then git commit -m wip; fi", []string{"commit"}},
		{"if true; then git commit -m x; fi", []string{"commit"}},
		{"if git pull --rebase; then true; else git merge x; fi", []string{"pull", "merge"}},
		{"if true; then :; elif git rebase main; then :; fi", []string{"rebase"}},
		{"{ git commit -m x; }", []string{"commit"}},
		{"! git commit -m x", []string{"commit"}},
		{`for b in a b; do git push origin "$b"; done`, []string{"push"}},
		{"while ! git push; do sleep 1; done", []string{"push"}},
		{"until git pull; do FOO=1 git am x.patch; done", []string{"pull", "am"}},
		{"function f { git commit -m x; }; f", []string{"commit"}},
		{"function f() { git push; }", []string{"push"}},
		{"function f if true; then git pull; fi", []string{"pull"}},
		{"FOO+=1 git commit -m x", []string{"commit"}},
		{"a[1]=x git commit -m x", []string{"commit"}},
		{"coproc git push", []string{"push"}},
		{"coproc P { git push; }", []string{"push"}},
		{"repeat 3 git pull", []string{"pull"}},
		{`case "$x" in a) git commit -m x;; esac`, []string{"commit"}},
		// No act.
		{"git status", nil},
		{"function git { :; }", nil},
		// A function in any other form is held by Check (function-form-unread),
		// and is an act whose body is not read: the hold asks.
		{"function f g { git commit -m x; }; f", []string{"function"}},
		{"function -T f { git push; }", []string{"function"}},
		{"function if g { git commit -m x; }; g", []string{"function"}},
		{"function f if git pull {; then :; fi", []string{"function"}},
		{"function 'f' { git commit -m x; }; f", []string{"function"}},
		{`function f { :; }; function \g { :; }`, []string{"function"}},
		{"if git diff --quiet; then git status; fi", nil},
		{"then commit", nil},
		{"{ $g commit; }", nil},
		{"if", nil},
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
