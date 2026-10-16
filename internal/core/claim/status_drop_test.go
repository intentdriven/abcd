package claim

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// What the status fetch deletes before it fetches (dropUnlisted): this
// checkout's copy of each status branch origin no longer lists or whose name
// is not read, and nothing else.

// TestOnlyTheStatusCopiesAreDropped: refs outside the status namespace, and a
// local branch under refs/heads/abcd-status/, survive the drop; so does the
// ref a symbolic ref under the namespace points at, since the drop deletes
// the symbolic ref itself, never what it names.
func TestOnlyTheStatusCopiesAreDropped(t *testing.T) {
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	maya.announceAs("merging", "#1012")
	pat.status(nil)
	tip := strings.TrimSpace(pat.git("rev-parse", "HEAD"))
	pat.git("update-ref", "refs/heads/abcd-status/pat", tip)
	pat.git("update-ref", "refs/remotes/origin/other", tip)
	pat.git("update-ref", "refs/abcd/keep-me", tip)
	pat.git("symbolic-ref", StatusTrackingPrefix+"sym", "refs/heads/abcd-status/pat")
	pat.status(nil)
	refs := pat.git("for-each-ref", "--format=%(refname)")
	for _, want := range []string{"refs/heads/abcd-status/pat", "refs/remotes/origin/other", "refs/abcd/keep-me", StatusTrackingPrefix + "maya"} {
		if !strings.Contains(refs, want+"\n") && !strings.HasSuffix(refs, want) {
			t.Fatalf("refs after the status fetch:\n%s\nwant %s kept", refs, want)
		}
	}
	if strings.Contains(refs, StatusTrackingPrefix+"sym") {
		t.Fatalf("refs after the status fetch:\n%s\nwant the symbolic ref under the namespace dropped", refs)
	}
}

// TestADropThatCannotBeMadeIsAGapNotAFailedFetch: a held packed-refs lock (a
// concurrent git) fails the deletion. The fetch still runs, and says which
// copies it could not drop; every reader reads the fetch, not an unread
// status; and a later fetch drops them.
func TestADropThatCannotBeMadeIsAGapNotAFailedFetch(t *testing.T) {
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	zed := w.clone("8+zed@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	maya.announceAs("merging", "#1012")
	zed.announceAs("asking", "#7")
	pat.status(nil)
	runGit(t, w.bare, "update-ref", "-d", StatusRefPrefix+"zed")
	lock := filepath.Join(pat.dir, ".git", "packed-refs.lock")
	if err := os.WriteFile(lock, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	maya.announceAs("merged", "#1012")
	res := pat.status(nil)
	if !strings.Contains(res.NotDropped, "1 copy of a status branch") {
		t.Fatalf("not dropped = %q; want the copy left named by count", res.NotDropped)
	}
	v := pat.read()
	if v.Unread != nil {
		t.Fatalf("unread = %+v; want the fetch read", v.Unread)
	}
	if len(v.Announcements) == 0 || v.Announcements[0].Kind != "merged" {
		t.Fatalf("announcements = %+v; want maya's newer one fetched", v.Announcements)
	}
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	if res := pat.status(nil); res.NotDropped != "" {
		t.Fatalf("with the lock gone, not dropped = %q", res.NotDropped)
	}
	if refs := pat.git("for-each-ref", "--format=%(refname)", StatusTrackingPrefix); strings.Contains(refs, "zed") {
		t.Fatalf("zed's copy survived a later fetch: %s", refs)
	}
}

// TestATipThatIsNotACommitIsReadOnce: a status branch whose tip is not a
// commit is doubtful, and marked read at that tip, so the prompts after it
// do not walk it again.
func TestATipThatIsNotACommitIsReadOnce(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	blob := strings.TrimSpace(runGitStdin(t, pat.dir, "not a commit\n", "hash-object", "-w", "--stdin"))
	ref := StatusTrackingPrefix + "maya"
	pat.git("update-ref", ref, blob)
	v := pat.read()
	if len(v.Doubts) != 1 || v.Doubts[0].Branch != "maya" || !strings.HasPrefix(v.Doubts[0].Reason, "its tip ") {
		t.Fatalf("doubts = %+v; want maya doubtful for its tip", v.Doubts)
	}
	if got := statusBranches.lastRead(pat.dir, ref); got != blob {
		t.Fatalf("maya's read mark = %q; want it marked at its tip %s", got, blob)
	}
	log := traced(t)
	pat.read()
	if log.anyArgContains(ref) {
		t.Fatalf("the next read walked maya again: %v", log.cmds)
	}
}

// TestTheAnnouncementsAreReadWhenTheHoldsAreNot: a read of the release holds
// that fails does not hide the announcements: they are still read, and the
// error names the read that failed.
func TestTheAnnouncementsAreReadWhenTheHoldsAreNot(t *testing.T) {
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	maya.announceAs("merging", "#1012")
	pat.status(nil)
	old := readHolds
	readHolds = func(string, time.Time, Options) ([]Hold, error) {
		return nil, errors.New("the reservations did not read")
	}
	t.Cleanup(func() { readHolds = old })
	v, err := ReadStatus(context.Background(), pat.dir, w.clock(), pat.opts())
	if err == nil || !strings.Contains(err.Error(), "reading the release holds: the reservations did not read") {
		t.Fatalf("ReadStatus error = %v; want the holds' read named", err)
	}
	if len(v.Announcements) != 1 || v.Announcements[0].Item != "#1012" {
		t.Fatalf("announcements = %+v; want maya's read all the same", v.Announcements)
	}
	got := pat.notes()
	for _, want := range []string{"- release holds unread: their read failed; `abcd status` says why\n", "- @maya (claimed): merging #1012, for 30 min\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("the block:\n%s\nwant it to carry %q", got, want)
		}
	}
}

// TestADropThatFailsForOneCopyDeletesTheRest: git deletes a batch of refs
// all or none, so when one copy cannot go (another git holds its lock), the
// drop deletes the rest one at a time, and counts only the one left.
func TestADropThatFailsForOneCopyDeletesTheRest(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	for _, b := range []string{"amy", "bob", "cal", "maya"} {
		w.pushStatus(b, announcementMsg("merging", "#1", b, t0, 30*time.Minute), "")
	}
	pat.status(nil)
	for _, b := range []string{"amy", "bob", "cal"} {
		runGit(t, w.bare, "update-ref", "-d", StatusRefPrefix+b)
	}
	lock := filepath.Join(pat.dir, ".git", filepath.FromSlash(StatusTrackingPrefix+"bob.lock"))
	if err := os.WriteFile(lock, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(lock) })
	res := pat.status(nil)
	if !strings.HasPrefix(res.NotDropped, "1 copy of a status branch") {
		t.Fatalf("not dropped = %q; want the one copy left counted", res.NotDropped)
	}
	refs := pat.git("for-each-ref", "--format=%(refname)", StatusTrackingPrefix)
	for b, kept := range map[string]bool{"amy": false, "bob": true, "cal": false, "maya": true} {
		if got := strings.Contains(refs, StatusTrackingPrefix+b); got != kept {
			t.Fatalf("refs after the drop:\n%s\nwant %s kept %v", refs, b, kept)
		}
	}
}
