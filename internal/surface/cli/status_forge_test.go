package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/shellquote"
)

// forgeAnswering puts a stand-in `gh` on PATH that answers the activity
// record in GitHub's shape: the fixture internal/core/github records
// (testdata/activity.json, a real answer trimmed, every login a placeholder),
// each {{name}} replaced by subs[name]. Asked for one activity type, as
// GitHub is, it answers only the entries of that type. fixture is the
// fixture's text, read before a test changes directory.
func forgeAnswering(t *testing.T, fixture string, subs map[string]string) {
	t.Helper()
	s := fixture
	for k, v := range subs {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	if strings.Contains(s, "{{") {
		t.Fatalf("the fixture keeps a placeholder: %s", s)
	}
	var all []map[string]any
	if err := json.Unmarshal([]byte(s), &all); err != nil {
		t.Fatal(err)
	}
	var pushes []map[string]any
	for _, a := range all {
		if a["activity_type"] == "push" {
			pushes = append(pushes, a)
		}
	}
	dir := t.TempDir()
	write := func(name string, v any) string {
		t.Helper()
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, b, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	every, onlyPush := write("every.json", all), write("push.json", pushes)
	script := "#!/bin/sh\ncase \"$*\" in\n*activity_type=push*) cat " + shellquote.Single(onlyPush) + " ;;\n*) cat " +
		shellquote.Single(every) + " ;;\nesac\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestTheForgesRecordJudgesEachPushLikeActivity is C17 through the forge
// adapter, against an answer in GitHub's own shape: the branch abcd-status/octo
// was made by another account's first push (branch_creation), then added to by
// octo, by a pull request's merge octo made, by an app, and by octo's forced
// push, beside a deletion that moves nothing and a merge queue's merge,
// started by another account, that moved the branch to the same tip as the
// forced push. The first announcement reads doubtful, naming the account that
// made the branch; octo's own pushes read verified; the app's reads claimed;
// the last reads doubtful, naming the account the merge queue moved the
// branch for; and the deletion judges no commit.
func TestTheForgesRecordJudgesEachPushLikeActivity(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "core", "github", "testdata", "activity.json"))
	if err != nil {
		t.Fatal(err)
	}
	w := newReserveWorld(t)
	octo := w.clone(t, "octo", octoMail)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	var tips []string
	for _, item := range []string{"1", "2", "3", "4"} {
		if code, _, stderr := reserveRun(t, octo, "announce", "merging", item); code != 0 {
			t.Fatalf("announce: exit %d, %s", code, stderr)
		}
		tips = append(tips, gitCmd(t, w.origin, "rev-parse", claim.StatusRefPrefix+"octo"))
	}
	c1, c2, c3, c4 := tips[0], tips[1], tips[2], tips[3]
	forgeAnswering(t, string(fixture), map[string]string{
		"before-force": c3, "after-force": c4,
		"before-bot": c2, "after-bot": c3,
		"before-push": c1, "after-push": c2,
		"after-creation":  c1,
		"before-deletion": c2,
		"before-queue":    c3, "after-queue": c4,
		"before-pr": c1, "after-pr": c2,
	})
	// origin is a local bare repository, so the adapter is asked from a
	// checkout whose origin names GitHub, as `abcd status` asks in one.
	gh := t.TempDir()
	gitInitAt(t, gh)
	gitCmd(t, gh, "remote", "add", "origin", "https://github.com/intentdriven/abcd.git")
	real := statusPushers
	statusPushers = func(_, branch string) ([]claim.Push, error) { return real(gh, branch) }
	t.Cleanup(func() { statusPushers = real })

	code, stdout, stderr := reserveRun(t, pat, "status", "--json")
	if code != 0 {
		t.Fatalf("status: exit %d, %s", code, stderr)
	}
	var got statusJSON
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("--json: %v\n%s", err, stdout)
	}
	checks := map[string]string{}
	for _, a := range got.Announcements {
		checks[a.Commit] = a.Check + " " + a.Text
	}
	for c, want := range map[string]string{
		c1: "doubtful @octo (doubtful: the forge's record names @mallory as its pusher): merging #1",
		c2: "verified @octo (verified): merging #2",
		c3: "claimed @octo (claimed): merging #3",
		c4: "doubtful @octo (doubtful: the forge's record names @mallory as its pusher): merging #4",
	} {
		if !strings.HasPrefix(checks[c], want) {
			t.Fatalf("checks = %v; want %s to read %q", checks, c, want)
		}
	}
	if got.SendersUnchecked != "" {
		t.Fatalf("senders unchecked = %q; want the forge asked in full", got.SendersUnchecked)
	}
}
