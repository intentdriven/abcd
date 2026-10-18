package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/claim"
)

// createdBranches is the created-branch list the recorder kept for root's
// copy, by name.
func createdBranches(t *testing.T, root string) []string {
	t.Helper()
	key := claim.OwnCopyKey(root)
	if key == "" {
		return nil
	}
	got, err := awayStore(t, root).CreatedBranches(key)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, b := range got {
		out = append(out, b.Name)
	}
	return out
}

// hookAllows runs the guard hook on command from root and fails the test on a
// deny: the recorder never refuses.
func hookAllows(t *testing.T, root, command string) (stderr string) {
	t.Helper()
	stdout, stderr, code := runGuard(preToolUse(t, "Bash", command, root), "guard", "hook")
	if strings.Contains(stdout, `"deny"`) || code == 2 {
		t.Fatalf("%q was refused: stdout %q stderr %q code %d", command, stdout, stderr, code)
	}
	return stderr
}

// TestBranchRecorderOnCreate: `git worktree add -b feat/x ../x`, `git switch
// -c feat/y` and `git checkout -B feat/z` record the branch under this copy's
// key, whether or not the person is away, only when no branch of that name
// exists, and `git checkout -b main` on an existing main records nothing.
func TestBranchRecorderOnCreate(t *testing.T) {
	root := awayCheckout(t)
	hookAllows(t, root, "git worktree add -b feat/x ../x")
	if got := createdBranches(t, root); !slices.Equal(got, []string{"feat/x"}) {
		t.Fatalf("not away: recorded %v, want [feat/x]", got)
	}
	if _, err := autonomy.SetAway(root, "", false, autonomy.Options{}); err != nil {
		t.Fatal(err)
	}
	hookAllows(t, root, "git switch -c feat/y")
	hookAllows(t, root, "git checkout -B feat/z")
	gitAt(t, root, "branch", "feat/old")
	hookAllows(t, root, "git checkout -b main")
	hookAllows(t, root, "git switch -c feat/old")
	if got := createdBranches(t, root); !slices.Equal(got, []string{"feat/x", "feat/y", "feat/z"}) {
		t.Fatalf("recorded %v, want [feat/x feat/y feat/z]", got)
	}
}

// TestBranchRecorderOtherForms: `git branch feat/w`, a bare `git worktree add
// ../v` recording v, and `git worktree add ../u feat/u` on an existing branch
// are recorded; `git branch -m feat/w feat/w2` carries the record; feat/x is
// not matched inside feat/x2; a tag checked out is no branch; the file stays
// within its cap.
func TestBranchRecorderOtherForms(t *testing.T) {
	root := awayCheckout(t)
	gitAt(t, root, "branch", "feat/u")
	gitAt(t, root, "branch", "feat/x")
	hookAllows(t, root, "git branch feat/w")
	hookAllows(t, root, "git worktree add ../v")
	hookAllows(t, root, "git worktree add ../u feat/u")
	gitAt(t, root, "tag", "v1")
	hookAllows(t, root, "git worktree add ../t v1")
	hookAllows(t, root, "git branch -m feat/w feat/w2")
	hookAllows(t, root, "git checkout -b feat/x2")
	hookAllows(t, root, "git branch -m feat/x feat/q")
	got := createdBranches(t, root)
	want := []string{"feat/w2", "v", "feat/u", "feat/x2"}
	if !slices.Equal(got, want) {
		t.Fatalf("recorded %v, want %v", got, want)
	}

	key := claim.OwnCopyKey(root)
	st := awayStore(t, root)
	var fill []autonomy.BranchChange
	for i := range autonomy.MaxCreatedBranches {
		fill = append(fill, autonomy.BranchChange{Kind: autonomy.BranchCreated, Name: fmt.Sprintf("feat/fill%03d", i)})
	}
	if err := st.RecordBranches(key, fill, time.Now()); err != nil {
		t.Fatal(err)
	}
	hookAllows(t, root, "git switch -c feat/last")
	got = createdBranches(t, root)
	if len(got) != autonomy.MaxCreatedBranches || got[len(got)-1] != "feat/last" {
		t.Fatalf("the list must stay within its cap of %d, newest kept: %d, last %q", autonomy.MaxCreatedBranches, len(got), got[len(got)-1])
	}
}

// TestBranchRecorderNeverRefuses: the command is allowed whatever the
// recorder does, and a store it cannot write leaves a stderr line.
func TestBranchRecorderNeverRefuses(t *testing.T) {
	root := awayCheckout(t)
	st := awayStore(t, root)
	// A plain file where the store's folder goes: nothing can be written.
	parent := filepath.Dir(st.Dir())
	if err := os.MkdirAll(parent, abcdhome.DirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(st.Dir(), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	stderr := hookAllows(t, root, "git switch -c feat/y")
	if !strings.Contains(stderr, "created branch") || !strings.Contains(stderr, "feat/y") {
		t.Fatalf("a store the recorder cannot write must leave a stderr line naming the branch; stderr %q", stderr)
	}
	// A command with no branch act is silent as before.
	if stderr := hookAllows(t, root, "git status --porcelain"); stderr != "" {
		t.Fatalf("no branch act, no line: %q", stderr)
	}
}

// TestBranchRecorderUsesSessionStartCopy: the recorder keys a branch on the
// session's start copy, the one the session-start hook noted, so a session
// that changed into a worktree it made still records under the copy the stop
// check reads; with no note for the session it keys on the payload cwd's
// worktree.
func TestBranchRecorderUsesSessionStartCopy(t *testing.T) {
	root := awayCheckout(t)
	wt := filepath.Join(t.TempDir(), "wt")
	gitAt(t, root, "worktree", "add", "-q", "-b", "feat/wt", wt)
	wt = realPath(t, wt)
	if err := os.MkdirAll(filepath.Join(wt, ".abcd", ".work.local"), 0o700); err != nil {
		t.Fatal(err)
	}
	payload := func(session, command string) string {
		b, err := json.Marshal(map[string]any{
			"session_id": session, "cwd": wt, "hook_event_name": "PreToolUse", "tool_name": "Bash",
			"tool_input": map[string]any{"command": command, "description": "x"},
		})
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	run := func(session, command string) {
		t.Helper()
		stdout, stderr, code := runGuard(payload(session, command), "guard", "hook")
		if strings.Contains(stdout, `"deny"`) || code == 2 {
			t.Fatalf("%q was refused: stdout %q stderr %q code %d", command, stdout, stderr, code)
		}
	}

	if _, err := awayStore(t, root).NoteSessionCopy("noted", root, time.Now()); err != nil {
		t.Fatal(err)
	}
	run("noted", "git switch -c feat/from-noted")
	if got := createdBranches(t, root); !slices.Contains(got, "feat/from-noted") {
		t.Fatalf("a noted session's branch is recorded under its start copy: start copy holds %v", got)
	}
	if got := createdBranches(t, wt); slices.Contains(got, "feat/from-noted") {
		t.Fatalf("a noted session's branch is not recorded under the worktree it ran in: worktree holds %v", got)
	}

	run("unnoted", "git switch -c feat/from-unnoted")
	if got := createdBranches(t, wt); !slices.Contains(got, "feat/from-unnoted") {
		t.Fatalf("with no note the branch is recorded under the payload cwd's worktree: worktree holds %v", got)
	}
	if got := createdBranches(t, root); slices.Contains(got, "feat/from-unnoted") {
		t.Fatalf("with no note nothing is recorded under another copy: start copy holds %v", got)
	}
}

// TestBranchRecorderSkipsBranchCheckedOutElsewhere: `git worktree add <path>
// <branch>` naming a branch another worktree has checked out fails in git,
// so the recorder, which runs before the command, records nothing for it;
// the same command naming a branch no worktree has checked out is recorded.
func TestBranchRecorderSkipsBranchCheckedOutElsewhere(t *testing.T) {
	root := awayCheckout(t)
	sib := filepath.Join(t.TempDir(), "b")
	gitAt(t, root, "worktree", "add", "-q", "-b", "feat/sib", sib)
	gitAt(t, root, "branch", "feat/free")
	hookAllows(t, root, "git worktree add ../r feat/sib")
	if got := createdBranches(t, root); slices.Contains(got, "feat/sib") {
		t.Errorf("a branch another worktree has checked out was recorded: %v", got)
	}
	hookAllows(t, root, "git worktree add ../f feat/free")
	if got := createdBranches(t, root); !slices.Contains(got, "feat/free") {
		t.Errorf("a branch no worktree has checked out was not recorded: %v", got)
	}
}
