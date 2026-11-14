package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestAReadOnlyMergeRunsNoFilterUnderRenormalize: merge.renormalize makes a
// three-way content merge run each side through the content filters the
// attributes select (git-config(1), merge.renormalize; gitattributes(5)), and
// merge-tree reads it, so a read-only merge abcd computes under
// MergeDriverOverrides alone started the repository's clean filter with no
// check asked. MergeDriverOverrides pins it off, and the command-line judge
// takes a merge-tree as starting nothing only when it carries that pin too.
func TestAReadOnlyMergeRunsNoFilterUnderRenormalize(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	repo := t.TempDir()
	marker := filepath.Join(t.TempDir(), "ran")
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.invalid", "-C", repo}, args...)...)
		cmd.Env = gitEnv()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(repo, "x.txt"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(repo, ".gitattributes"), []byte("*.txt filter=f\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	write("a\n1\n2\n3\n4\nb\n")
	git("add", ".")
	git("commit", "-q", "-m", "base")
	git("switch", "-q", "-c", "side")
	write("a\n1\n2\n3\n4\nB\n")
	git("commit", "-q", "-a", "-m", "side")
	git("switch", "-q", "main")
	write("A\n1\n2\n3\n4\nb\n")
	git("commit", "-q", "-a", "-m", "main")
	git("config", "filter.f.clean", "touch '"+marker+"'; cat")
	git("config", "filter.f.smudge", "cat")
	git("config", "merge.renormalize", "true")

	overrides, err := MergeDriverOverrides(repo)
	if err != nil {
		t.Fatal(err)
	}
	args := append(slices.Clip(overrides), "merge-tree", "--write-tree", "main", "side")
	if _, err := Run(repo, args...); err != nil {
		t.Fatalf("merge-tree: %v", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatalf("the read-only merge ran the repository's clean filter under merge.renormalize (overrides %q)", overrides)
	}
	if LeavesContentProgramsOn(args) {
		t.Errorf("a merge-tree carrying MergeDriverOverrides must start nothing: %q", args)
	}
	if !LeavesContentProgramsOn([]string{"-c", "merge.default=text", "merge-tree", "--write-tree", "main", "side"}) {
		t.Error("a merge-tree whose renormalize is not pinned off can start a content filter, so it must ask")
	}
}

// TestALogOrShowWhoseFormatIsNotFixedAsks: format.pretty is the default
// format of log, show and whatchanged, and pretty.<name> an alias a
// --pretty=<name> or --format=<name> takes (git-config(1)); either can name
// a format holding %G, whose signature check starts gpg.program. So a log
// or show whose format the command line does not fix (a user format holding
// %, an empty one, a format: or tformat: one, or a built-in name, which an
// alias cannot shadow) asks, and one that fixes it does not. A show of a
// blob alone is not formatted.
func TestALogOrShowWhoseFormatIsNotFixedAsks(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"log", "-1", "HEAD"}, true},
		{[]string{"log", "--format=changelog", "-1"}, true},
		{[]string{"log", "--pretty=changelog", "-1"}, true},
		{[]string{"log", "-n", "--format=%H"}, true}, // the format is -n's value
		{[]string{"log", "--format=%H", "-1"}, false},
		{[]string{"log", "--format=", "-1"}, false},
		{[]string{"log", "--pretty=format:%x00%ad"}, false},
		{[]string{"log", "--pretty=tformat:x"}, false},
		{[]string{"log", "--pretty=oneline"}, false},
		{[]string{"log", "--oneline"}, false},
		{[]string{"log", "-p", "--no-textconv", "--no-ext-diff"}, true},
		{[]string{"log", "-p", "--no-textconv", "--no-ext-diff", "--format=commit %H"}, false},
		{[]string{"whatchanged", "-1"}, true},
		{[]string{"show", "-s", "HEAD"}, true},
		{[]string{"show", "--no-textconv", "--no-ext-diff", "HEAD"}, true},
		{[]string{"show", "-s", "--format=%ct", "HEAD"}, false},
		{[]string{"show", "HEAD:a.txt"}, false},
		{[]string{"reflog", "show", "-n", "1", "--format=%ct", "main", "--"}, false},
	} {
		if got := LeavesContentProgramsOn(tc.args); got != tc.want {
			t.Errorf("LeavesContentProgramsOn(%q) = %v, want %v", tc.args, got, tc.want)
		}
	}
}

// TestNoAliasStandsForASubcommandAbcdRuns: git ignores an alias that hides
// one of its own commands (git-config(1), alias.*), so alias.* changes no git
// command abcd runs while every subcommand the table of what an isolated
// command can start knows is one of git's own: the hold leaves alias.* out
// for that reason (remoteid.go), and this fails when the table gains a
// subcommand git does not ship.
func TestNoAliasStandsForASubcommandAbcdRuns(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	cmd := exec.Command("git", "--list-cmds=main,nohelpers")
	cmd.Env = gitEnv()
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git --list-cmds: %v", err)
	}
	own := strings.Fields(string(out))
	var verbs []string
	for v := range startsNothing {
		verbs = append(verbs, v)
	}
	for v := range contentJudges {
		verbs = append(verbs, v)
	}
	// The networked runners' subcommands, which neither list holds.
	verbs = append(verbs, "fetch", "push", "ls-remote")
	slices.Sort(verbs)
	for _, v := range verbs {
		if !slices.Contains(own, v) {
			t.Errorf("%q is not one of git's own commands, so an alias of that name would stand for it", v)
		}
	}
}
