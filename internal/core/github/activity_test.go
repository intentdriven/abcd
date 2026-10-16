package github

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
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
		`{"before":"`+zero+`","after":"`+b+`","activity_type":"push","actor":null},{"before":"`+zero+`","after":"nope","actor":{"login":"x"}}]`)
	dir := originAt(t, "git@github.com:intentdriven/abcd.git")
	got, err := Pushers(dir, "refs/heads/abcd-status/maya")
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
	argv := "api\n--hostname\ngithub.com\nrepos/intentdriven/abcd/activity?activity_type=push&per_page=100&ref=refs%2Fheads%2Fabcd-status%2Fmaya\n"
	if string(data) != argv {
		t.Fatalf("argv = %q, want %q", data, argv)
	}
}

// TestPushersRefusesAnOriginNotOnGitHub: another forge cannot say who
// pushed, and gh is never run for it.
func TestPushersRefusesAnOriginNotOnGitHub(t *testing.T) {
	logPath := fakeActivity(t, `[]`)
	if _, err := Pushers(originAt(t, "https://gitlab.example.com/a/b.git"), "refs/heads/abcd-status/maya"); err == nil {
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
	fakeActivity(t, `[{"before":"`+b+`","after":"`+a+`","actor":{"login":"claude","type":"Bot"}},`+
		`{"before":"`+c+`","after":"`+b+`","actor":{"login":"some-app[bot]","type":"User"}},`+
		`{"before":"`+zero+`","after":"`+c+`","actor":{"login":"maya","type":"User"}}]`)
	got, err := Pushers(originAt(t, "https://github.com/intentdriven/abcd.git"), "refs/heads/abcd-status/maya")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != (Push{Before: zero, After: c, Login: "maya"}) {
		t.Fatalf("pushers = %+v; want only maya's push", got)
	}
}
