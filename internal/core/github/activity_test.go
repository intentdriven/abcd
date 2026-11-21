package github

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
// each moves the branch, and is read as a push by its actor. A deletion moves
// the branch to no commit and is passed over. An app's push, the merge
// queue's among them, is read as an app's (App): GitHub credits a merge
// queue's merge to the queue (type Bot), and a pull request's merge to the
// person who merged it. The query names no activity type, so none is lost to
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
		{Before: sha("3"), After: sha("4"), Login: "hosted-app[bot]", App: true},
		{Before: sha("5"), After: sha("6"), Login: "octo"},
		{Before: sha("0"), After: sha("7"), Login: "mallory"},
		{Before: sha("b"), After: sha("c"), Login: "octo"},
		{Before: sha("9"), After: sha("a"), Login: "github-merge-queue[bot]", App: true},
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
	return fakePages(t, body)
}

// fakePages puts a stand-in `gh` on PATH that logs its argv and answers one
// page of pages: the first when the path names no cursor, page n when it
// names after=cursor<n>. Asked with --include, as the forge's activity record
// is, it writes the status line and headers first, as gh does, with a Link
// header naming the next page's cursor while one is left.
func fakePages(t *testing.T, pages ...string) (logPath string) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	dir := t.TempDir()
	logPath = filepath.Join(dir, "argv.log")
	for i, body := range pages {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("page%d.json", i)), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	script := `#!/usr/bin/env bash
printf '%s\n' "$@" >>"` + logPath + `"
inc=0; path=""
for a in "$@"; do [ "$a" = --include ] && inc=1; path="$a"; done
n=0
if [[ "$path" =~ after=cursor([0-9]+) ]]; then n="${BASH_REMATCH[1]}"; fi
if [ "$inc" = 1 ]; then
  printf 'HTTP/2.0 200 OK\n'
  if [ -f "` + dir + `/page$((n+1)).json" ]; then
    printf 'Link: <https://api.github.com/repositories/1/activity?per_page=100&ref=x&after=cursor%d>; rel="next", <https://api.github.com/repositories/1/activity?per_page=100&ref=x&before=cursor0>; rel="prev"\r\n' $((n+1))
  fi
  printf 'Content-Type: application/json; charset=utf-8\r\n\r\n'
fi
cat "` + dir + `/page$n.json"
`
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
	argv := "api\n--hostname\ngithub.com\n--include\nrepos/intentdriven/abcd/activity?per_page=100&ref=refs%2Fheads%2Fabcd-status%2Fmaya\n"
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

// TestPushersMarksAnAppsPush is C17's app identity: a push made by an app (an
// actor of type Bot, or a login ending in [bot]) says nothing of who sent the
// announcement, so it is marked as an app's, which judges no commit and
// leaves the sender claimed rather than reading as another account's push.
func TestPushersMarksAnAppsPush(t *testing.T) {
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
	want := []Push{
		{Before: b, After: a, Login: "claude", App: true},
		{Before: c, After: b, Login: "some-app[bot]", App: true},
		{Before: zero, After: c, Login: "maya"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("pushers = %+v; want %+v: the apps' pushes marked, maya's not", got, want)
	}
}

// TestPushersMarksAMergeQueuesMerge: GitHub credits a merge queue's merge
// (merge_queue_merge) to the queue itself, github-merge-queue[bot] of type
// Bot, which says nothing of who sent an announcement, so it is marked as an
// app's and the commits it carried stay claimed; a pull request's merge
// (pr_merge) names the person who merged it, and is read as theirs.
func TestPushersMarksAMergeQueuesMerge(t *testing.T) {
	a := strings.Repeat("a", 40)
	b := strings.Repeat("b", 40)
	c := strings.Repeat("c", 40)
	fakeActivity(t, `[{"before":"`+b+`","after":"`+a+`","activity_type":"merge_queue_merge","actor":{"login":"github-merge-queue[bot]","type":"Bot"}},`+
		`{"before":"`+c+`","after":"`+b+`","activity_type":"pr_merge","actor":{"login":"maya","type":"User"}}]`)
	got, _, err := Pushers(context.Background(), originAt(t, "https://github.com/intentdriven/abcd.git"), "refs/heads/abcd-status/maya")
	if err != nil {
		t.Fatal(err)
	}
	want := []Push{
		{Before: b, After: a, Login: "github-merge-queue[bot]", App: true},
		{Before: c, After: b, Login: "maya"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("pushers = %+v; want %+v: the queue's merge marked, maya's not", got, want)
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

// TestPushersPageReadsOnByTheCursorThePageNames: a page of the forge's
// activity record names the page past it in its Link header; the next ask
// sends that cursor back as after, and the last page names none.
func TestPushersPageReadsOnByTheCursorThePageNames(t *testing.T) {
	a := strings.Repeat("a", 40)
	b := strings.Repeat("b", 40)
	zero := strings.Repeat("0", 40)
	logPath := fakePages(t,
		`[{"before":"`+b+`","after":"`+a+`","activity_type":"push","actor":{"login":"maya"}}]`,
		`[{"before":"`+zero+`","after":"`+b+`","activity_type":"branch_creation","actor":{"login":"maya"}}]`)
	dir := originAt(t, "https://github.com/intentdriven/abcd.git")
	first, err := PushersPage(context.Background(), dir, "refs/heads/abcd-reserved/rel-v1.0.0", "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Next != "cursor1" || !first.More() || len(first.Pushes) != 1 || first.Pushes[0].After != a {
		t.Fatalf("first page = %+v; want maya's push and the cursor of the page past it", first)
	}
	second, err := PushersPage(context.Background(), dir, "refs/heads/abcd-reserved/rel-v1.0.0", first.Next)
	if err != nil {
		t.Fatal(err)
	}
	if second.Next != "" || second.More() || len(second.Pushes) != 1 || second.Pushes[0].Before != zero {
		t.Fatalf("second page = %+v; want the push that made the branch, and no page past it", second)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "activity?after=cursor1&per_page=100&ref=refs%2Fheads%2Fabcd-reserved%2Frel-v1.0.0\n") {
		t.Fatalf("argv = %q; want the second ask to send the cursor back", data)
	}
}

// TestPushersPageRefusesACursorItDoesNotRead: a cursor outside the forge's
// token alphabet is never sent, and gh is not run for it.
func TestPushersPageRefusesACursorItDoesNotRead(t *testing.T) {
	logPath := fakePages(t, `[]`)
	dir := originAt(t, "https://github.com/intentdriven/abcd.git")
	if _, err := PushersPage(context.Background(), dir, "refs/heads/abcd-reserved/rel-v1.0.0", "x&ref=refs/heads/main"); err == nil {
		t.Fatal("a cursor outside the token alphabet was sent")
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("gh ran for a cursor it does not read: %v", err)
	}
}

// TestNextCursorTakesOnlyTheNextLinksAfterParameter: of the Link header,
// only the rel="next" link's after parameter is read; a link of another
// relation, a header of another name, or a cursor outside the alphabet names
// no page past it.
func TestNextCursorTakesOnlyTheNextLinksAfterParameter(t *testing.T) {
	for _, c := range []struct {
		header []string
		want   string
	}{
		{[]string{"HTTP/2.0 200 OK", `Link: <https://api.github.com/x?before=p>; rel="prev", <https://elsewhere.example/y?after=Y3Vy_c29y-MQ%3D%3D>; rel="next"`}, "Y3Vy_c29y-MQ=="},
		{[]string{"HTTP/2.0 200 OK", `link: <https://api.github.com/x?after=abc>; rel="next"`}, "abc"},
		{[]string{"HTTP/2.0 200 OK", `Link: <https://api.github.com/x?after=abc>; rel="prev"`}, ""},
		{[]string{"HTTP/2.0 200 OK", `X-Link: <https://api.github.com/x?after=abc>; rel="next"`}, ""},
		{[]string{"HTTP/2.0 200 OK", `Link: <https://api.github.com/x?after=a%26b>; rel="next"`}, ""},
		{[]string{"HTTP/2.0 200 OK", `Link: https://api.github.com/x?after=abc; rel="next"`}, ""},
	} {
		if got := nextCursor(c.header); got != c.want {
			t.Fatalf("nextCursor(%q) = %q; want %q", c.header, got, c.want)
		}
	}
}

// TestPushersPageRefusesAnAnswerWithoutHeaders: an answer that does not
// open with gh's status line and headers names no page past it, so it is
// refused rather than read as a whole record.
func TestPushersPageRefusesAnAnswerWithoutHeaders(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte("#!/usr/bin/env bash\nprintf '[]\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if _, err := PushersPage(context.Background(), originAt(t, "https://github.com/intentdriven/abcd.git"), "refs/heads/abcd-reserved/rel-v1.0.0", ""); err == nil {
		t.Fatal("an answer with no headers was read")
	}
}
