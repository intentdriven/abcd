package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSessionRepoRoundTrip: the note is the whole fallback. A hook holding only
// a session id has to reach the store without a working directory, because the
// worktree the payload names may have been removed before the hook ran.
func TestSessionRepoRoundTrip(t *testing.T) {
	repoRoot, _ := setupStore(t)
	if err := NoteSessionRepo(repoRoot, testRootSHA, "sess-note"); err != nil {
		t.Fatalf("NoteSessionRepo: %v", err)
	}
	got, err := SessionRepo("sess-note")
	if err != nil {
		t.Fatalf("SessionRepo: %v", err)
	}
	if got != testRootSHA {
		t.Errorf("SessionRepo = %q, want %q", got, testRootSHA)
	}
}

// TestSessionRepoUnknownSessionIsAnError: an unknown session must not resolve to
// "some store". The caller's next move is to stage nothing and say so.
func TestSessionRepoUnknownSessionIsAnError(t *testing.T) {
	_, _ = setupStore(t)
	if _, err := SessionRepo("sess-never-seen"); err == nil {
		t.Fatal("an unknown session resolved to a store")
	}
}

// TestSessionRepoRefusesAnAmbiguousSession: a transcript filed against the wrong
// repository is redacted by the wrong repository's scanner configuration. That
// is a privacy fault, so two claimants refuse rather than pick.
func TestSessionRepoRefusesAnAmbiguousSession(t *testing.T) {
	repoRoot, home := setupStore(t)
	other := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if err := os.MkdirAll(filepath.Join(home, ".abcd", "transcripts", other, "records"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, sha := range []string{testRootSHA, other} {
		if err := NoteSessionRepo(repoRoot, sha, "sess-both"); err != nil {
			t.Fatalf("NoteSessionRepo(%s): %v", sha, err)
		}
	}
	if got, err := SessionRepo("sess-both"); err == nil {
		t.Fatalf("an ambiguous session resolved to %q instead of refusing", got)
	}
}

// TestSessionRepoRefusesADirectoryReference: sessionIDRe admits "." and "..",
// which as a whole path segment would walk out of the directory being written.
func TestSessionRepoRefusesADirectoryReference(t *testing.T) {
	repoRoot, _ := setupStore(t)
	for _, id := range []string{".", ".."} {
		if err := NoteSessionRepo(repoRoot, testRootSHA, id); err == nil {
			t.Errorf("NoteSessionRepo accepted %q as a session id", id)
		}
		if _, err := SessionRepo(id); err == nil {
			t.Errorf("SessionRepo accepted %q as a session id", id)
		}
	}
}

// TestSessionNotesArePruned keeps the directory bounded: a note is an inode, and
// a machine that runs sessions for years should not accumulate them forever.
func TestSessionNotesArePruned(t *testing.T) {
	repoRoot, home := setupStore(t)
	if err := NoteSessionRepo(repoRoot, testRootSHA, "sess-old"); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".abcd", "transcripts", testRootSHA, "sessions")
	old := time.Now().Add(-2 * sessionNoteTTL)
	if err := os.Chtimes(filepath.Join(dir, "sess-old"), old, old); err != nil {
		t.Fatal(err)
	}
	if err := NoteSessionRepo(repoRoot, testRootSHA, "sess-new"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sess-old")); !os.IsNotExist(err) {
		t.Error("a note well past its TTL survived a later write")
	}
	if _, err := SessionRepo("sess-new"); err != nil {
		t.Errorf("the fresh note was pruned too: %v", err)
	}
}

// TestSubagentGapMarker: on a harness that fires SubagentStop without an
// agent_transcript_path there is nothing to stage, and an empty sub-agent corpus
// would look exactly like a session that delegated nothing. The marker is what
// tells those apart.
func TestSubagentGapMarker(t *testing.T) {
	repoRoot, _ := setupStore(t)
	if _, ok, err := SubagentGap(repoRoot, testRootSHA); err != nil || ok {
		t.Fatalf("a fresh store reports a gap: ok=%v err=%v", ok, err)
	}
	if err := NoteSubagentGap(repoRoot, testRootSHA, "SubagentStop"); err != nil {
		t.Fatalf("NoteSubagentGap: %v", err)
	}
	note, ok, err := SubagentGap(repoRoot, testRootSHA)
	if err != nil || !ok {
		t.Fatalf("marker not readable: ok=%v err=%v", ok, err)
	}
	if note.Count != 1 || note.Event != "SubagentStop" || note.FirstSeen.IsZero() {
		t.Errorf("marker = %+v, want one sighting of SubagentStop with a first-seen time", note)
	}
	if err := NoteSubagentGap(repoRoot, testRootSHA, "SubagentStop"); err != nil {
		t.Fatal(err)
	}
	again, _, err := SubagentGap(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if again.Count != 2 {
		t.Errorf("count = %d, want 2", again.Count)
	}
	if !again.FirstSeen.Equal(note.FirstSeen) {
		t.Error("a second sighting moved first_seen; it must record when the gap was FIRST seen")
	}
}
