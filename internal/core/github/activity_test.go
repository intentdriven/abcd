package github

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// activityFixture is the forge's activity record as GitHub answers it
// (testdata/activity.json: a real answer's shape, trimmed, with every login
// a placeholder), each {{name}} in it replaced by subs[name].
func activityFixture(t *testing.T, subs map[string]string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "activity.json"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for k, v := range subs {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	if strings.Contains(s, "{{") {
		t.Fatalf("the fixture keeps a placeholder: %s", s)
	}
	return s
}

// TestPushersReadsEveryPushLikeActivity: GitHub records a branch's first push
// as branch_creation, a forced one as force_push, and a merge made by a pull
// request or a merge queue as pr_merge and merge_queue_merge, beside push:
// each moves the branch, and its actor is the account that moved it, so all
// five are read. A deletion moves the branch to no commit and is passed over,
// as is an app's push. The query names no activity type, so none is lost to
// it.
func TestPushersReadsEveryPushLikeActivity(t *testing.T) {
	sha := func(c string) string { return strings.Repeat(c, 40) }
	logPath := fakeActivity(t, activityFixture(t, map[string]string{
		"before-force": sha("1"), "after-force": sha("2"), "before-bot": sha("3"), "after-bot": sha("4"),
		"before-push": sha("5"), "after-push": sha("6"), "after-creation": sha("7"), "before-deletion": sha("8"),
		"before-queue": sha("9"), "after-queue": sha("a"), "before-pr": sha("b"), "after-pr": sha("c"),
	}))
	got, partial, err := Pushers(context.Background(), originAt(t, "https://github.com/intentdriven/abcd.git"), "refs/heads/abcd-status/octo")
	if err != nil || partial {
		t.Fatalf("Pushers: partial %v, %v; want a record read in full", partial, err)
	}
	want := []Push{
		{Before: sha("1"), After: sha("2"), Login: "octo"},
		{Before: sha("5"), After: sha("6"), Login: "octo"},
		{Before: sha("0"), After: sha("7"), Login: "mallory"},
		{Before: sha("b"), After: sha("c"), Login: "octo"},
		{Before: sha("9"), After: sha("a"), Login: "mallory"},
	}
	if len(got) != len(want) {
		t.Fatalf("pushers = %+v; want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pushers = %+v; want %+v", got, want)
		}
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "activity_type") {
		t.Fatalf("the query names an activity type, so the others are never answered: %q", data)
	}
}

// fakeActivity puts a stand-in `gh` on PATH that logs its argv and answers
// body.
func fakeActivity(t *testing.T, body string) (logPath string) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	dir := t.TempDir()
	logPath = filepath.Join(dir, "argv.log")
	answer := filepath.Join(dir, "answer.json")
	if err := os.WriteFile(answer, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	script := "#!/usr/bin/env bash\nprintf '%s\\n' \"$@\" >>\"" + logPath + "\"\ncat \"" + answer + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

// originAt is a repository whose origin is url.
func originAt(t *testing.T, url string) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", url}} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

// TestPushersReadsTheForgesRecordOfWhoPushed (itd-2609151838312703 C17): the
// activity record of pushes to one status branch lists each push, newest
// first, with the tip it moved from and to and the pusher's login, read from
// github.com for the repository origin names; an entry with no actor or a
// commit name that is not one is passed over.
func TestPushersReadsTheForgesRecordOfWhoPushed(t *testing.T) {
	a := strings.Repeat("a", 40)
	b := strings.Repeat("b", 40)
	zero := strings.Repeat("0", 40)
	logPath := fakeActivity(t, `[{"before":"`+b+`","after":"`+a+`","activity_type":"push","actor":{"login":"Maya"}},`+
		`{"before":"`+zero+`","after":"`+b+`","activity_type":"push","actor":{"login":"mallory"}},`+
		`{"before":"`+zero+`","after":"`+b+`","activity_type":"push","actor":null},{"before":"`+zero+`","after":"nope","activity_type":"push","actor":{"login":"x"}}]`)
	dir := originAt(t, "git@github.com:intentdriven/abcd.git")
	got, _, err := Pushers(context.Background(), dir, "refs/heads/abcd-status/maya")
	if err != nil {
		t.Fatal(err)
	}
	want := []Push{{Before: b, After: a, Login: "Maya"}, {Before: zero, After: b, Login: "mallory"}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("pushers = %+v; want %+v", got, want)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	argv := "api\n--hostname\ngithub.com\nrepos/intentdriven/abcd/activity?per_page=100&ref=refs%2Fheads%2Fabcd-status%2Fmaya\n"
	if string(data) != argv {
		t.Fatalf("argv = %q, want %q", data, argv)
	}
}

// TestPushersRefusesAnOriginNotOnGitHub: another forge cannot say who
// pushed, and gh is never run for it.
func TestPushersRefusesAnOriginNotOnGitHub(t *testing.T) {
	logPath := fakeActivity(t, `[]`)
	if _, _, err := Pushers(context.Background(), originAt(t, "https://gitlab.example.com/a/b.git"), "refs/heads/abcd-status/maya"); err == nil {
		t.Fatal("an origin off github.com was asked")
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("gh ran for an origin off github.com: %v", err)
	}
}

// TestPushersPassesOverAnAppsPush is C17's app identity: a push made by an
// app (an actor of type Bot, or a login ending in [bot], as a hosted session
// pushes) says nothing of who sent the announcement, so it is passed over and
// the sender stays claimed rather than reading as another account's push.
func TestPushersPassesOverAnAppsPush(t *testing.T) {
	a := strings.Repeat("a", 40)
	b := strings.Repeat("b", 40)
	c := strings.Repeat("c", 40)
	zero := strings.Repeat("0", 40)
	fakeActivity(t, `[{"before":"`+b+`","after":"`+a+`","activity_type":"push","actor":{"login":"claude","type":"Bot"}},`+
		`{"before":"`+c+`","after":"`+b+`","activity_type":"push","actor":{"login":"some-app[bot]","type":"User"}},`+
		`{"before":"`+zero+`","after":"`+c+`","activity_type":"branch_creation","actor":{"login":"maya","type":"User"}}]`)
	got, _, err := Pushers(context.Background(), originAt(t, "https://github.com/intentdriven/abcd.git"), "refs/heads/abcd-status/maya")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != (Push{Before: zero, After: c, Login: "maya"}) {
		t.Fatalf("pushers = %+v; want only maya's push", got)
	}
}

// TestPushersSaysWhenItsPageCameBackFull: one ask reads the forge's one page
// of 100 activities, so a page that came back full, of any type, may have
// older activities past it, and the record is said to be partial; one page
// short of full is the whole record.
func TestPushersSaysWhenItsPageCameBackFull(t *testing.T) {
	a := strings.Repeat("a", 40)
	b := strings.Repeat("b", 40)
	entry := `{"before":"` + b + `","after":"` + a + `","activity_type":"push","actor":{"login":"maya"}}`
	deletion := `{"before":"` + a + `","after":"` + strings.Repeat("0", 40) + `","activity_type":"branch_deletion","actor":{"login":"maya"}}`
	for _, c := range []struct {
		pushes, deletions int
		partial           bool
	}{{99, 0, false}, {100, 0, true}, {60, 40, true}, {60, 39, false}} {
		var es []string
		for range c.pushes {
			es = append(es, entry)
		}
		for range c.deletions {
			es = append(es, deletion)
		}
		fakeActivity(t, "["+strings.Join(es, ",")+"]")
		got, partial, err := Pushers(context.Background(), originAt(t, "https://github.com/intentdriven/abcd.git"), "refs/heads/abcd-status/maya")
		if err != nil {
			t.Fatal(err)
		}
		if partial != c.partial || len(got) != c.pushes {
			t.Fatalf("%d pushes and %d deletions: %d pushes, partial %v; want %d, partial %v", c.pushes, c.deletions, len(got), partial, c.pushes, c.partial)
		}
	}
}

// TestPushersEndsAtItsCallersDeadline: a forge that does not answer holds
// the call no longer than its caller's deadline, well inside Timeout, and
// the error says gh did not answer.
func TestPushersEndsAtItsCallersDeadline(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte("#!/usr/bin/env bash\nsleep 20\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	repo := originAt(t, "https://github.com/intentdriven/abcd.git")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, _, err := Pushers(ctx, repo, "refs/heads/abcd-status/maya")
	took := time.Since(start)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("Pushers = %v; want gh's silence named", err)
	}
	if took > 3*time.Second {
		t.Fatalf("a silent forge held the call %s past a 300 ms deadline", took)
	}
}
