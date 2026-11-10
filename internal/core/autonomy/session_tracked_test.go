package autonomy

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestASessionStateACommitCarriesIsRefused: a session's wake state is this
// copy's own, so one git tracks came with a commit and is refused rather than
// read as the session's.
func TestASessionStateACommitCarriesIsRefused(t *testing.T) {
	root := t.TempDir()
	if err := SaveSession(root, "sess-1", SessionState{SchemaVersion: SchemaVersion}); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSession(context.Background(), root, "sess-1"); err != nil {
		t.Fatalf("this copy's own state: %v", err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-f", "--", path.Join(SessionsDir, "sess-1.json")}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, err := LoadSession(context.Background(), root, "sess-1"); err == nil || !strings.Contains(err.Error(), "git tracks") {
		t.Fatalf("LoadSession over a tracked state = %v; want it refused as tracked", err)
	}
}

// TestALaneCheckTheBudgetEndedIsACutReading: the question whether git tracks
// the run tier, asked once the check's budget is spent, fails for the budget
// and not for anything tracked, so the reading is cut short, never whole with
// no lanes.
func TestALaneCheckTheBudgetEndedIsACutReading(t *testing.T) {
	f := newFx(t)
	f.writeRun("run-0000000000000001", "itd-1", false, map[string]any{"id": "lane-1", "branch": "feat/lane", "stage": "validate"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := &owedReader{ctx: ctx, root: f.root}
	r.readLanes()
	if !r.cut || len(r.lanes) != 0 {
		t.Fatalf("a lane check the budget ended: cut %v, %d lanes; want a cut reading", r.cut, len(r.lanes))
	}
}

// TestTheStopAndPromptReadsRunNoGitOnASpentDeadline: the session's state, the
// behind note, the copy's resolution and the owed reading all run their git
// under the hook's ctx, so a spent deadline runs none and each read refuses.
func TestTheStopAndPromptReadsRunNoGitOnASpentDeadline(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Commit("seed")
	root := repo.Root()
	if err := SaveSession(root, "sess-1", SessionState{SchemaVersion: SchemaVersion, ChecklistNote: notePrefix + "x"}); err != nil {
		t.Fatal(err)
	}
	calls := countGit(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := LoadSession(ctx, root, "sess-1"); err == nil {
		t.Error("LoadSession read the state on a spent deadline")
	}
	if note, err := TakeChecklistNote(ctx, root, "sess-1"); err == nil || note != "" {
		t.Errorf("TakeChecklistNote took %q on a spent deadline", note)
	}
	if _, _, err := ResolveCopyContext(ctx, root, "sess-1", Options{Home: t.TempDir()}); err == nil {
		t.Error("ResolveCopyContext resolved on a spent deadline")
	}
	if _, err := ReadOwed(ctx, ReadOptions{Root: root, Home: t.TempDir()}); err == nil {
		t.Error("ReadOwed read on a spent deadline")
	}
	if n := calls(); n != 0 {
		t.Errorf("a spent deadline still ran git %d time(s)", n)
	}
}

// hangingIdentityGit puts a git first on PATH that logs every call and hangs
// on any that reads a user.* identity key, handing the rest to the real git.
func hangingIdentityGit(t *testing.T) (log string) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	bin := t.TempDir()
	log = filepath.Join(t.TempDir(), "calls")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '" + log + "'\ncase \"$*\" in *user*) exec sleep 60;; esac\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

// TestTheOwedReadsLaterGitCallsAreBound: the owed read's identity reads run
// under its ctx after the root lookup. The person's email is read first and
// its git hangs until the deadline kills it; the reader's holder, read later
// while away, then forks no git at all. Unbound, the first would hang the
// read past its deadline and the second would start a git after it.
func TestTheOwedReadsLaterGitCallsAreBound(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Commit("seed")
	log := hangingIdentityGit(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := ReadOwed(ctx, ReadOptions{Root: repo.Root(), Home: t.TempDir(), Record: &Record{Start: time.Now()},
			Forge: func(context.Context, string, ...string) (string, error) { return "", errors.New("no forge here") }})
		done <- err
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("the owed read hung past its deadline on an identity read")
	}
	raw, _ := os.ReadFile(log)
	var identityReads []string
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, "user") {
			identityReads = append(identityReads, line)
		}
	}
	if len(identityReads) != 1 {
		t.Errorf("identity reads %q, want only the email's, killed at the deadline", identityReads)
	}
}

// TestTheFetchStampIsJudgedUnderTheDeadline: the fetch time the owed read
// names is the stamp's, asked of git only under the read's ctx, so a spent
// deadline forks no git for it.
func TestTheFetchStampIsJudgedUnderTheDeadline(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Commit("seed")
	stamp := filepath.Join(repo.Root(), ".abcd", ".work.local", "reservation", "fetched-at")
	if err := os.MkdirAll(filepath.Dir(stamp), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stamp, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	calls := countGit(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := &owedReader{ctx: ctx, root: repo.Root()}
	if got := r.fetchedAt(); got != "an unknown time" {
		t.Errorf("a spent deadline still named the fetch: %q", got)
	}
	if n := calls(); n != 0 {
		t.Errorf("a spent deadline still ran git %d time(s)", n)
	}
}
