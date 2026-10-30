package history

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStagePendingIsIdempotentOnContent: identical bytes for a session are a
// no-op, different bytes replace the pending copy, and one session has one
// pending transcript whatever fires.
func TestStagePendingIsIdempotentOnContent(t *testing.T) {
	setupStore(t)
	meta := PendingMeta{SessionID: "sess-p", Cwd: "/work/repo", SourcePath: "/host/sess-p.jsonl"}

	first, err := StagePending(meta, []byte("one\n"))
	if err != nil || !first.Wrote || first.Replaced {
		t.Fatalf("first stage = %+v, %v", first, err)
	}
	again, err := StagePending(meta, []byte("one\n"))
	if err != nil || again.Wrote {
		t.Fatalf("identical bytes must be a no-op, got %+v, %v", again, err)
	}
	newer, err := StagePending(meta, []byte("one\ntwo\n"))
	if err != nil || !newer.Replaced || newer.ReplacedBytes != 4 {
		t.Fatalf("different bytes must replace the copy, got %+v, %v", newer, err)
	}
	pending, err := ListPending()
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].Cwd != meta.Cwd || pending[0].SourcePath != meta.SourcePath || pending[0].Bytes != 8 {
		t.Fatalf("want one pending entry carrying its metadata, got %+v", pending)
	}
	if fi, err := os.Stat(pending[0].Path); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("a pending transcript is unredacted and must be 0o600: %v %v", fi.Mode(), err)
	}
}

// TestStagePendingRefusesAnUnsafeSessionID: the session id names a file, so
// one that walks out of the directory is refused and nothing is written.
func TestStagePendingRefusesAnUnsafeSessionID(t *testing.T) {
	setupStore(t)
	for _, id := range []string{"", "..", "../escape", "a/b"} {
		if _, err := StagePending(PendingMeta{SessionID: id}, []byte("x\n")); err == nil {
			t.Errorf("session id %q was accepted", id)
		}
	}
	if pending, _ := ListPending(); len(pending) != 0 {
		t.Fatalf("a refused stage wrote %+v", pending)
	}
}

// TestPlacePendingFilesByCwdAndKeepsWhatItCannotPlace: a transcript whose cwd
// resolves is staged in that store's lane and leaves the pending directory;
// one no store claims stays pending and is reported with its reason.
func TestPlacePendingFilesByCwdAndKeepsWhatItCannotPlace(t *testing.T) {
	repoRoot, _ := setupStore(t)
	if _, err := StagePending(PendingMeta{SessionID: "sess-here", Cwd: repoRoot}, []byte("here\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := StagePending(PendingMeta{SessionID: "sess-lost", Cwd: "/gone"}, []byte("lost\n")); err != nil {
		t.Fatal(err)
	}
	resolve := func(cwd string) (string, string, error) {
		if cwd == repoRoot {
			return repoRoot, testRootSHA, nil
		}
		return "", "", errors.New("its working directory is gone")
	}

	res, err := PlacePending(resolve)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Placed) != 1 || res.Placed[0].SessionID != "sess-here" || res.Placed[0].Via != "cwd" {
		t.Fatalf("placed = %+v", res.Placed)
	}
	if len(res.Unplaced) != 1 || res.Unplaced[0].SessionID != "sess-lost" || !strings.Contains(res.Unplaced[0].Reason, "gone") {
		t.Fatalf("unplaced = %+v", res.Unplaced)
	}
	staged, err := ListStaged(repoRoot, testRootSHA)
	if err != nil || len(staged) != 1 || staged[0].SessionID != "sess-here" {
		t.Fatalf("staged = %+v (err %v)", staged, err)
	}
	pending, _ := ListPending()
	if len(pending) != 1 || pending[0].SessionID != "sess-lost" {
		t.Fatalf("pending after placement = %+v", pending)
	}
}

// TestPlacePendingPrefersTheSessionNote: the note a session wrote when it
// started names its store even when its cwd now resolves to nothing.
func TestPlacePendingPrefersTheSessionNote(t *testing.T) {
	repoRoot, _ := setupStore(t)
	if err := NoteSessionRepo(repoRoot, testRootSHA, "sess-noted"); err != nil {
		t.Fatal(err)
	}
	if _, err := StagePending(PendingMeta{SessionID: "sess-noted", Cwd: "/removed/worktree"}, []byte("noted\n")); err != nil {
		t.Fatal(err)
	}
	res, err := PlacePending(func(string) (string, string, error) { return "", "", errors.New("gone") })
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Placed) != 1 || res.Placed[0].Via != "session-note" || res.Placed[0].RootSHA != testRootSHA {
		t.Fatalf("placed = %+v, unplaced = %+v", res.Placed, res.Unplaced)
	}
}

// TestPlacePendingKeepsAnEntryWithoutMetadata: a transcript whose sidecar is
// gone carries no cwd to trust, so it is reported and never placed.
func TestPlacePendingKeepsAnEntryWithoutMetadata(t *testing.T) {
	setupStore(t)
	if _, err := StagePending(PendingMeta{SessionID: "sess-bare", Cwd: "/x"}, []byte("bare\n")); err != nil {
		t.Fatal(err)
	}
	pending, _ := ListPending()
	_, side := pendingPaths(filepath.Dir(pending[0].Path), "sess-bare")
	if err := os.Remove(side); err != nil {
		t.Fatal(err)
	}
	res, err := PlacePending(func(string) (string, string, error) { return "", testRootSHA, nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Placed) != 0 || len(res.Unplaced) != 1 || !strings.Contains(res.Unplaced[0].Reason, "metadata") {
		t.Fatalf("placed = %+v, unplaced = %+v", res.Placed, res.Unplaced)
	}
}

// TestTakePendingMissReportsOnce: a stage the session-end hook could not make
// is counted, and the start that reads the count clears it.
func TestTakePendingMissReportsOnce(t *testing.T) {
	setupStore(t)
	for range 2 {
		if err := NotePendingMiss("session s1", "disk full"); err != nil {
			t.Fatal(err)
		}
	}
	note, ok, err := TakePendingMiss()
	if err != nil || !ok || note.Count != 2 || note.LastWho != "session s1" {
		t.Fatalf("take = %+v %v %v", note, ok, err)
	}
	if _, ok, err := TakePendingMiss(); err != nil || ok {
		t.Fatalf("a taken marker must be gone, got ok=%v err=%v", ok, err)
	}
}

// TestPlacePendingDropsASessionOutsideAnyRepository: no store noted the
// session and the resolver says definitely that its directory is in no
// repository, so no start will ever place it; the raw copy is deleted rather
// than kept unredacted forever, and the drop is reported.
func TestPlacePendingDropsASessionOutsideAnyRepository(t *testing.T) {
	setupStore(t)
	if _, err := StagePending(PendingMeta{SessionID: "sess-nowhere", Cwd: "/downloads"}, []byte("x\n")); err != nil {
		t.Fatal(err)
	}
	res, err := PlacePending(func(string) (string, string, error) {
		return "", "", fmt.Errorf("its working directory is %w", ErrNoRepository)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Dropped) != 1 || res.Dropped[0].SessionID != "sess-nowhere" || len(res.Unplaced) != 0 {
		t.Fatalf("dropped = %+v, unplaced = %+v", res.Dropped, res.Unplaced)
	}
	if pending, _ := ListPending(); len(pending) != 0 {
		t.Fatalf("a session outside any repository is still pending: %+v", pending)
	}
}

// TestPlacePendingKeepsWhatAnUncertainAnswerCannotSettle: a git that did not
// answer is not a definite "no repository", so the copy stays pending.
func TestPlacePendingKeepsWhatAnUncertainAnswerCannotSettle(t *testing.T) {
	setupStore(t)
	if _, err := StagePending(PendingMeta{SessionID: "sess-unsure", Cwd: "/busy"}, []byte("x\n")); err != nil {
		t.Fatal(err)
	}
	res, err := PlacePending(func(string) (string, string, error) {
		return "", "", errors.New("git gave no root commit (signal: killed)")
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Dropped) != 0 || len(res.Unplaced) != 1 {
		t.Fatalf("dropped = %+v, unplaced = %+v", res.Dropped, res.Unplaced)
	}
}

// TestPlacePendingDoesNotDropACopyRestagedSinceTheListing: a SessionEnd that
// re-fires between the listing and the lock wrote a newer copy, and the drop
// judged on the older one must leave it alone.
func TestPlacePendingDoesNotDropACopyRestagedSinceTheListing(t *testing.T) {
	setupStore(t)
	meta := PendingMeta{SessionID: "sess-resumed", Cwd: "/downloads"}
	if _, err := StagePending(meta, []byte("old\n")); err != nil {
		t.Fatal(err)
	}
	res, err := PlacePending(func(string) (string, string, error) {
		// The resolver runs after the listing and before the lock: the
		// session resumes and ends again here.
		if _, err := StagePending(meta, []byte("old\nnewer\n")); err != nil {
			t.Fatal(err)
		}
		return "", "", fmt.Errorf("its working directory is %w", ErrNoRepository)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Dropped) != 0 {
		t.Fatalf("a re-staged copy was dropped on the older copy's judgement: %+v", res.Dropped)
	}
	pending, _ := ListPending()
	if len(pending) != 1 || pending[0].Bytes != int64(len("old\nnewer\n")) {
		t.Fatalf("the newer copy is gone: %+v", pending)
	}
}

// TestDiscardPendingRemovesOneCopy: the operator's way to remove a pending
// transcript no start can place; a name that is not one is refused.
func TestDiscardPendingRemovesOneCopy(t *testing.T) {
	setupStore(t)
	if _, err := StagePending(PendingMeta{SessionID: "sess-d", Cwd: "/gone"}, []byte("bytes\n")); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "../sess-d.raw", "sess-d", "absent.raw"} {
		if _, err := DiscardPending(bad); err == nil {
			t.Errorf("DiscardPending(%q) was accepted", bad)
		}
	}
	res, err := DiscardPending("sess-d.raw")
	if err != nil || res.Bytes != 6 || len(res.Removed) != 2 {
		t.Fatalf("discard = %+v, %v", res, err)
	}
	if pending, _ := ListPending(); len(pending) != 0 {
		t.Fatalf("still pending: %+v", pending)
	}
}
