package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/shellquote"
)

// TestAFullPageOfTheForgesRecordKeepsNoMarkForWhatItDidNotCover is C17 when
// the forge's record runs past the one page abcd reads: a hundred pushes,
// the newest, fill it, and the commit beneath them is covered by none of
// them. That commit is never kept as asked, so the next `abcd status` asks
// again, and each says the record was not read in full.
func TestAFullPageOfTheForgesRecordKeepsNoMarkForWhatItDidNotCover(t *testing.T) {
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	// The branch is named by octo's login, as a release before the holder id
	// named it: only such a branch names an account the forge is asked about.
	var tips []string
	for _, item := range []string{"1", "2"} {
		tips = append(tips, w.legacyAnnounce(t, "octo", item))
	}
	older, newer := tips[0], tips[1]
	var page []map[string]any
	for range 100 {
		page = append(page, map[string]any{"before": newer, "after": newer, "activity_type": "push",
			"ref": claim.StatusRefPrefix + "octo", "actor": map[string]any{"login": "octo", "type": "User"}})
	}
	page[len(page)-1]["before"] = older
	body, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	answer, calls := filepath.Join(dir, "answer.json"), filepath.Join(dir, "calls")
	if err := os.WriteFile(answer, body, 0o600); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\necho call >>" + shellquote.Single(calls) + "\ncat " + shellquote.Single(answer) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	gh := t.TempDir()
	gitInitAt(t, gh)
	gitCmd(t, gh, "remote", "add", "origin", "https://github.com/intentdriven/abcd.git")
	real := statusPushers
	statusPushers = func(ctx context.Context, _, branch string) ([]claim.Push, error) { return real(ctx, gh, branch) }
	t.Cleanup(func() { statusPushers = real })

	for run := 1; run <= 2; run++ {
		code, stdout, stderr := reserveRun(t, pat, "status", "--json")
		if code != 0 {
			t.Fatalf("status run %d: exit %d, %s", run, code, stderr)
		}
		var got statusJSON
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("--json: %v\n%s", err, stdout)
		}
		if !strings.Contains(got.SendersUnchecked, "abcd-status/octo: the forge's record was not read in full") {
			t.Fatalf("status run %d: senders unchecked = %q; want the record said to be read in part", run, got.SendersUnchecked)
		}
		for _, a := range got.Announcements {
			if a.Commit == older && a.Check != claim.SenderClaimed {
				t.Fatalf("status run %d: the commit no push covered reads %q; want claimed", run, a.Check)
			}
		}
		data, err := os.ReadFile(calls)
		if err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(string(data), "call\n"); n != run {
			t.Fatalf("after status run %d the forge was asked %d times; want %d, each run asking again", run, n, run)
		}
	}
}
