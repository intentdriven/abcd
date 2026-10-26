package landing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core"
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
	// The trailers make the landing announce before it pushes. Git's identity
	// names the sender, so no forge is asked whose account it is, and the
	// embargo check, which is not what this test is about, lets it through.
	f.asPat()
	o := f.options()
	o.Announce.Embargo = func(_, _, _ string, _ []string) (bool, error) { return false, nil }
	res, err := f.land(o)
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
		"\nAssisted-by: " + assistedby.ComposedValue(core.Version) + "\n",
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

// TestAComposedBodyThatFailsACheckSaysSo: in a repository that opted into the
// mention check, a branch whose subject names an issue it never declares
// composes a body the guard would block, and the stop says which check it
// fails rather than handing over a body that will be refused.
func TestAComposedBodyThatFailsACheckSaysSo(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.repo.Write(".abcd/config.json", `{"outbound": {"forge": true, "declare_mentions": true}}`+"\n")
	f.repo.Git("add", ".abcd/config.json")
	f.repo.Git("commit", "-q", "--amend", "-m", "fix: the walk behind iss-7\n\nAssisted-by: None")
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue")
	if s.Status != StatusStopped || !strings.Contains(s.Note, "iss-7") || !strings.Contains(s.Note, "fails") {
		t.Fatalf("enqueue step %+v, want the stop to name the mention the body leaves undeclared", s)
	}
}

// TestADisclosureFaultIsNamedForWhatItIs: only a body with no disclosure is
// told to add one; a body carrying a line the guard refuses, a Co-authored-by
// a commit wrote, is told to correct that line.
func TestADisclosureFaultIsNamedForWhatItIs(t *testing.T) {
	dir := t.TempDir()
	note := judgeBody(dir, composedBody{title: "fix: x", body: "fix: x\n\nAssisted-by: None\nCo-authored-by: Alice Example <alice@example.com>\n"})
	if !strings.Contains(note, "Co-authored-by") || !strings.Contains(note, "correct or remove") {
		t.Errorf("a body carrying a Co-authored-by line: %q, want the line named and corrected", note)
	}
	if strings.Contains(note, "disclose none") || strings.Contains(note, "add an Assisted-by") {
		t.Errorf("a body that discloses is told it discloses none: %q", note)
	}
	if note := judgeBody(dir, composedBody{title: "fix: x", body: "fix: x\n"}); !strings.Contains(note, "add an Assisted-by line") {
		t.Errorf("a body with no disclosure: %q, want it told to add one", note)
	}
}

// TestAnUnansweredIgnoreCheckIsNotCalledNotIgnored: check-ignore exits 128
// when git cannot answer (here, no repository at all), which says nothing
// about whether the tier is ignored.
func TestAnUnansweredIgnoreCheckIsNotCalledNotIgnored(t *testing.T) {
	path, _, note := writeBody(t.TempDir(), "feat/x", "0123456789abcdef0123456789abcdef01234567", "fix: x\n")
	if path != "" {
		t.Fatalf("a body was written where git could not answer: %s", path)
	}
	if strings.Contains(note, "is not ignored") || !strings.Contains(note, "could not say") {
		t.Errorf("note %q, want it to say git could not answer", note)
	}
}

// TestAnEditedBodyFileIsKept: a person who edits the body land composed, then
// runs land again before posting, keeps the edit, and the stop judges the text
// the command will post; a new commit names a body file of its own.
func TestAnEditedBodyFileIsKept(t *testing.T) {
	f := newFixture(t, queueRuleset)
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	first, _ := bodyFileOf(t, step(t, res, "enqueue").Command)
	edited := "- fix: the walk, as I would put it\n\nAssisted-by: None\n"
	if err := os.WriteFile(first, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err = f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue")
	again, body := bodyFileOf(t, s.Command)
	if again != first || body != edited {
		t.Fatalf("the rerun named %s holding %q; want the edited %s kept", again, body, first)
	}
	if !strings.Contains(s.Note, "kept") {
		t.Errorf("the note does not say the file was kept: %s", s.Note)
	}
	f.repo.Write("second.txt", "more\n")
	f.repo.Git("add", "second.txt")
	f.repo.Git("commit", "-q", "-m", "fix: the walk again\n\nAssisted-by: None")
	res, err = f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if next, _ := bodyFileOf(t, step(t, res, "enqueue").Command); next == first {
		t.Errorf("a new head reused the body file %s composed for the old one", first)
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

// TestTheComposedBodyDeclaresAbcdsLabel: abcd composed the body's text, so it
// carries abcd's own label whatever the commits declare (ruling R4), then the
// models they name; a commit's None is not carried, since a tool assembled the
// text the body is.
func TestTheComposedBodyDeclaresAbcdsLabel(t *testing.T) {
	label := "\nAssisted-by: " + assistedby.ComposedValue(core.Version) + "\n"
	for name, tc := range map[string]struct {
		second string
		want   string
		unwant string
	}{
		"all none":   {"Assisted-by: None", label, "Assisted-by: None"},
		"one helped": {"Assisted-by: Claude:claude-opus-5-5", label + "Assisted-by: Claude:claude-opus-5-5\n", "Assisted-by: None"},
		"one silent": {"", label, "Assisted-by: None"},
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
