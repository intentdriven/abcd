package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/occupancy"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/gittest"
)

// sharedCheckout is a main checkout two live sessions started in, on a
// fixture machine whose process table holds both hosts.
func sharedCheckout(t *testing.T) (repo string, f *nameFixture) {
	t.Helper()
	repo, _ = sessionEndRepo(t)
	f = newNameFixture(t)
	f.procs = []machineload.Proc{
		{PID: 7001, PPID: 1, Name: "claude", Age: time.Hour},
		{PID: 7002, PPID: 1, Name: "claude", Age: time.Hour},
	}
	start := time.Now().Add(-time.Hour).Unix()
	tab, _ := f.machine().ReadTable()
	for id, pid := range map[string]int{"s-one": 7001, "s-two": 7002} {
		if err := occupancy.Note(f.home, id, repo, sessionname.Host{PID: pid, StartUnix: start}, tab); err != nil {
			t.Fatal(err)
		}
	}
	return repo, f
}

func issueRecords(t *testing.T, root string) []string {
	t.Helper()
	got, _ := filepath.Glob(filepath.Join(root, ".abcd", "work", "issues", "*", "iss-*.md"))
	return got
}

// TestCaptureRefusesAMainCheckoutTwoLiveSessionsShare is iss-2610240859590333:
// on 2026-10-24 a session filed records into a main checkout another live
// session worked in, and nothing refused. A record verb there now backs off
// (exit 3), names the worktree store, and writes nothing.
func TestCaptureRefusesAMainCheckoutTwoLiveSessionsShare(t *testing.T) {
	repo, _ := sharedCheckout(t)
	t.Chdir(repo)

	_, err := runCLIErr(t, "capture", "a finding filed into a checkout two live sessions share")

	var ee *exitError
	if !errors.As(err, &ee) || ee.Code != 3 {
		t.Fatalf("want exit 3 (back off), got %v", err)
	}
	for _, want := range []string{"2 live sessions work in this main checkout", "git worktree add", "worktrees/<root-sha>/<name>", "Nothing was written"} {
		if !strings.Contains(ee.Msg, want) {
			t.Errorf("the refusal must say %q; got %q", want, ee.Msg)
		}
	}
	if recs := issueRecords(t, repo); len(recs) != 0 {
		t.Errorf("a refused capture wrote %v", recs)
	}
}

// TestDecideRefusesASharedMainCheckout holds the decision store to the same
// rule, through its own front door.
func TestDecideRefusesASharedMainCheckout(t *testing.T) {
	repo, _ := sharedCheckout(t)
	t.Chdir(repo)

	_, err := runCLIErr(t, "decide", "a decision recorded into a checkout two live sessions share")

	var ee *exitError
	if !errors.As(err, &ee) || ee.Code != 3 {
		t.Fatalf("want exit 3, got %v", err)
	}
}

// TestEveryOtherRecordWriterRefusesASharedMainCheckout: the record writers
// beyond the plain create paths refuse too, before they read or write
// anything, so no write through capture or intent lands in a shared main
// checkout.
func TestEveryOtherRecordWriterRefusesASharedMainCheckout(t *testing.T) {
	for _, args := range [][]string{
		{"capture", "migrate", "--apply"},
		{"capture", "resolve", "iss-1", "a note", "--impact", "fix"},
		{"intent", "a draft filed into a checkout two live sessions share"},
		{"intent", "plan", "itd-1"},
		{"intent", "audit", "itd-1"},
		{"intent", "audit", "--owed"},
		{"intent", "interview", "itd-1"},
		{"intent", "consistency", "ingest", "--findings-json", "absent.json"},
		{"spec", "close", "spc-1"},
		{"decide", "--adr", "an ADR minted into a checkout two live sessions share"},
	} {
		t.Run(strings.Join(args[:2], " "), func(t *testing.T) {
			repo, _ := sharedCheckout(t)
			t.Chdir(repo)
			_, err := runCLIErr(t, args...)
			var ee *exitError
			if !errors.As(err, &ee) || ee.Code != 3 || !strings.Contains(ee.Msg, "live sessions work in this main checkout") {
				t.Fatalf("%v: want the shared-checkout refusal (exit 3), got %v", args, err)
			}
		})
	}
}

// TestCaptureWritesInALinkedWorktreeOfASharedCheckout: the remedy the refusal
// names works. A linked worktree is a session's own, however many sessions
// share the main checkout it belongs to.
func TestCaptureWritesInALinkedWorktreeOfASharedCheckout(t *testing.T) {
	repo, f := sharedCheckout(t)
	wt := filepath.Join(t.TempDir(), "lane")
	cmd := exec.Command("git", "worktree", "add", "-q", "-b", "lane", wt)
	cmd.Dir = repo
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v: %s", err, out)
	}
	// Both sessions name the worktree too, so only the main-tree test can be
	// what lets this write through.
	tab, _ := f.machine().ReadTable()
	start := time.Now().Add(-time.Hour).Unix()
	for id, pid := range map[string]int{"s-one": 7001, "s-two": 7002} {
		if err := occupancy.Note(f.home, id+"-wt", wt, sessionname.Host{PID: pid, StartUnix: start}, tab); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(wt)

	if _, err := runCLIErr(t, "capture", "a finding filed from a worktree of its own"); err != nil {
		t.Fatalf("a linked worktree must not be refused: %v", err)
	}
	if recs := issueRecords(t, wt); len(recs) != 1 {
		t.Errorf("want one record in the worktree, got %v", recs)
	}
}

// TestCaptureWritesWhereOneSessionWorks: a person working alone, or one
// session, is never held up by the rule.
func TestCaptureWritesWhereOneSessionWorks(t *testing.T) {
	repo, f := sharedCheckout(t)
	if err := occupancy.Forget(f.home, "s-two"); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)

	if _, err := runCLIErr(t, "capture", "a finding filed where one session works"); err != nil {
		t.Fatalf("one session's checkout must not be refused: %v", err)
	}
}

// TestCaptureWritesOnceTheOtherSessionHasEnded: a session that ended without
// its SessionEnd hook stops counting when its host process is gone.
func TestCaptureWritesOnceTheOtherSessionHasEnded(t *testing.T) {
	repo, f := sharedCheckout(t)
	f.procs = f.procs[:1] // s-two's host has exited
	t.Chdir(repo)

	if _, err := runCLIErr(t, "capture", "a finding filed after a peer session crashed"); err != nil {
		t.Fatalf("an ended session must not count: %v", err)
	}
}

// TestSessionHooksNoteAndForgetTheCheckout: session start writes the marker
// naming the checkout and its host; session end removes it.
func TestSessionHooksNoteAndForgetTheCheckout(t *testing.T) {
	repo, _ := sessionEndRepo(t)
	f := newNameFixture(t)
	f.launch(7101)

	runHook(t, startJSON(t, "s-hooks", repo, nil), "hook", "session-start")

	marker := abcdhome.Path(f.home, "occupancy", "s-hooks.json")
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("session start must write the marker: %v", err)
	}
	real, _ := filepath.EvalSymlinks(repo)
	if !strings.Contains(string(data), `"checkout":"`+real+`"`) || !strings.Contains(string(data), `"host_pid":7101`) {
		t.Errorf("the marker must name the checkout and the host: %s", data)
	}

	runHook(t, endPayload(t, "s-hooks", repo, ""), "hook", "session-end")
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("session end must remove the marker: %v", err)
	}
}
