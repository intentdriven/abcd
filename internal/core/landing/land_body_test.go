package landing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/assistedby"
	"github.com/intentdriven/abcd/internal/core/issuemention"
)

// With no pull request open, land composes the body from the branch's own
// commits (iss-220): their subjects, their Refs:, Resolves: and Delivers: lines
// and their Assisted-by: lines, judged with the checks the shell guard applies
// to a posted body, written to the checkout's local tier, and named by the
// create command's --body-file. A body nobody composed is the one CI refused
// for a missing disclosure line, after the push.

// bodyFileOf is the --body-file path the create command names, read back.
func bodyFileOf(t *testing.T, cmd string) (path, body string) {
	t.Helper()
	_, rest, ok := strings.Cut(cmd, " --body-file ")
	if !ok {
		t.Fatalf("the create command names no body file: %s", cmd)
	}
	path = strings.Trim(strings.Fields(rest)[0], "'")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the body file the command names cannot be read: %v", err)
	}
	return path, string(b)
}

func TestNoPullRequestComposesTheBodyFromTheCommits(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.repo.Git("commit", "-q", "--amend", "-m", "fix: the walk skips a record family\n\nThe walk now reads every family (iss-1).\n\nRefs: iss-1\nResolves: iss-2\nAssisted-by: Claude:claude-opus-5-5")
	f.repo.Write("second.txt", "more\n")
	f.repo.Git("add", "second.txt")
	f.repo.Git("commit", "-q", "-m", "docs: say which families the walk reads\n\nDelivers: itd-3\nAssisted-by: Claude:claude-opus-5-5\nAssisted-by: ExampleVendor:model-b")
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue")
	if s.Status != StatusStopped || !strings.HasPrefix(s.Command, "gh pr create --base main --head feat/x --title ") {
		t.Fatalf("enqueue step %+v, want stopped with the create command", s)
	}
	if !strings.Contains(s.Command, "--title 'fix: the walk skips a record family'") {
		t.Errorf("the title is not the branch's first commit's subject: %s", s.Command)
	}
	path, body := bodyFileOf(t, s.Command)
	// Both resolved: a temporary directory may sit behind a symbolic link
	// (/var on macOS), and land names the checkout git reports.
	tier, err := filepath.EvalSymlinks(filepath.Join(f.repo.Root(), ".abcd", ".work.local"))
	if err != nil {
		t.Fatal(err)
	}
	if real, err := filepath.EvalSymlinks(path); err != nil || !strings.HasPrefix(real, tier+string(filepath.Separator)) {
		t.Errorf("the body file %s is not in the checkout's local tier %s (%v)", path, tier, err)
	}
	for _, want := range []string{
		"- fix: the walk skips a record family\n",
		"- docs: say which families the walk reads\n",
		"\nRefs: iss-1\n", "\nResolves: iss-2\n", "\nDelivers: itd-3\n",
		"\nAssisted-by: Claude:claude-opus-5-5\n", "\nAssisted-by: ExampleVendor:model-b\n",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the composed body lacks %q:\n%s", want, body)
		}
	}
	if n := strings.Count(body, "Assisted-by: Claude:claude-opus-5-5"); n != 1 {
		t.Errorf("a model two commits disclose is named %d times, want once:\n%s", n, body)
	}
	if strings.Contains(body, "The walk now reads") {
		t.Errorf("the body carries a commit's prose, which it does not compose from:\n%s", body)
	}
	if fault := assistedby.JudgeBody(body); fault != assistedby.BodyClean {
		t.Errorf("the composed body fails the disclosure check (%v):\n%s", fault, body)
	}
	if ids := issuemention.Undeclared("fix: the walk skips a record family", body); len(ids) != 0 {
		t.Errorf("the composed body names %v without declaring them:\n%s", ids, body)
	}
	if strings.Contains(s.Note, "fails") {
		t.Errorf("a body that passes its checks is reported as failing one: %s", s.Note)
	}
	if f.forge.called("pr", "create") != 0 {
		t.Fatalf("forge calls %v: land opens no pull request", f.forge.calls)
	}
	if dirty, err := dirtyTree(f.repo.Root()); err != nil || dirty {
		t.Fatalf("writing the body file left the checkout dirty (%v)", err)
	}
}

// TestAComposedBodyWithoutADisclosureSaysSo: a branch whose commits disclose
// nothing composes a body the guard would block, and the stop says which check
// it fails rather than handing over a body that will be refused.
func TestAComposedBodyWithoutADisclosureSaysSo(t *testing.T) {
	f := newFixture(t, queueRuleset)
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue")
	if s.Status != StatusStopped || !strings.Contains(s.Note, "Assisted-by") || !strings.Contains(s.Note, "fails") {
		t.Fatalf("enqueue step %+v, want the stop to name the disclosure the body lacks", s)
	}
}

// TestNoLocalTierWritesNoBody: land makes no local tier, so a worktree with
// none (its receipt minted in another worktree) gets the bare create command
// and a note saying why, and still has no tier afterwards.
func TestNoLocalTierWritesNoBody(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = "[]"
	f.forge.head = f.head()
	wt := filepath.Join(t.TempDir(), "wt")
	f.repo.Git("worktree", "add", "-q", "-b", "feat/y", wt, "feat/x")
	if err := f.mint(f.repo.Root()); err != nil {
		t.Fatal(err)
	}
	res, err := Land(wt, f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue")
	if s.Status != StatusStopped || strings.Contains(s.Command, "--body-file") || !strings.Contains(s.Note, "no local tier") {
		t.Fatalf("enqueue step %+v, want the bare create command and the missing tier named", s)
	}
	if _, err := os.Stat(filepath.Join(wt, ".abcd", ".work.local")); err == nil {
		t.Fatal("land made the worktree's local tier")
	}
}

// TestAllNoneCommitsComposeNone: Assisted-by: None is the body's line only
// when every commit declares it; one assisted commit makes the body an
// assisted one, and one that declares nothing leaves the body without it.
func TestAllNoneCommitsComposeNone(t *testing.T) {
	for name, tc := range map[string]struct {
		second string
		want   string
		unwant string
	}{
		"all none":   {"Assisted-by: None", "\nAssisted-by: None\n", "Assisted-by: Claude"},
		"one helped": {"Assisted-by: Claude:claude-opus-5-5", "\nAssisted-by: Claude:claude-opus-5-5\n", "Assisted-by: None"},
		// A commit that declares nothing has not said no model helped.
		"one silent": {"", "- fix: the walk again\n", "Assisted-by: None"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, queueRuleset)
			f.repo.Git("commit", "-q", "--amend", "-m", "fix: the walk\n\nAssisted-by: None")
			f.repo.Write("second.txt", "more\n")
			f.repo.Git("add", "second.txt")
			f.repo.Git("commit", "-q", "-m", "fix: the walk again\n\n"+tc.second)
			res, err := f.land(f.options())
			if err != nil {
				t.Fatal(err)
			}
			_, body := bodyFileOf(t, step(t, res, "enqueue").Command)
			if !strings.Contains(body, tc.want) || strings.Contains(body, tc.unwant) {
				t.Fatalf("composed body, want %q and no %q:\n%s", tc.want, tc.unwant, body)
			}
		})
	}
}
