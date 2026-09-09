package history

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// mainStage is the StageMeta for a main-thread transcript: a session id and
// nothing else. Sub-agent stages fill the lineage in.
func mainStage(sessionID string) StageMeta {
	return StageMeta{Lineage: CaptureMeta{SessionID: sessionID, Kind: "native"}}
}

// stagedNames lists the staging dir's staged transcripts (the .raw entries; the
// staging lock file is not a transcript) for assertions.
func stagedNames(t *testing.T, home string) []string {
	t.Helper()
	sdir := filepath.Join(home, ".abcd", "history", testRootSHA, "staging")
	entries, err := os.ReadDir(sdir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), stagedSuffix) {
			out = append(out, e.Name())
		}
	}
	return out
}

// TestStageWritesRawWithoutRedacting pins the whole point of staging: the
// SessionEnd half must NOT run the scanner. A secret planted in the transcript
// is expected to survive into the staged file verbatim — that is what makes the
// write cheap, and it is why the directory is 0o700 and drained promptly.
func TestStageWritesRawWithoutRedacting(t *testing.T) {
	_, _ = setupStore(t)
	secret := "ghp_" + strings.Repeat("a", 36)
	raw := []byte("assistant: token is " + secret + "\n")

	res, err := Stage(testRootSHA, mainStage("sess-stage"), raw)
	if err != nil {
		t.Fatalf("Stage failed: %v", err)
	}
	if !res.Wrote {
		t.Fatal("expected Wrote=true on a first stage")
	}
	body, err := os.ReadFile(res.Staged.Path)
	if err != nil {
		t.Fatalf("staged file unreadable: %v", err)
	}
	if !strings.Contains(string(body), secret) {
		t.Fatal("staged file was redacted; staging must write raw bytes or it is not cheap")
	}
	if res.Staged.Bytes != int64(len(raw)) {
		t.Errorf("Bytes = %d, want %d", res.Staged.Bytes, len(raw))
	}
}

// TestStagingDirIsOwnerOnly guards the one place abcd stores unredacted
// transcript text. A group- or world-readable staging dir would leak every
// secret the store exists to keep out.
func TestStagingDirIsOwnerOnly(t *testing.T) {
	_, home := setupStore(t)
	if _, err := Stage(testRootSHA, mainStage("sess-perm"), []byte("hello\n")); err != nil {
		t.Fatalf("Stage failed: %v", err)
	}
	sdir := filepath.Join(home, ".abcd", "history", testRootSHA, "staging")
	fi, err := os.Stat(sdir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o700 {
		t.Errorf("staging dir mode = %o, want 700 (it holds unredacted transcripts)", perm)
	}
	names := stagedNames(t, home)
	if len(names) != 1 {
		t.Fatalf("expected exactly one staged file, got %v", names)
	}
	ffi, err := os.Stat(filepath.Join(sdir, names[0]))
	if err != nil {
		t.Fatal(err)
	}
	if perm := ffi.Mode().Perm(); perm != 0o600 {
		t.Errorf("staged file mode = %o, want 600", perm)
	}
}

// TestStageIsIdempotentPerSession proves a re-fired SessionEnd carrying the SAME
// bytes cannot stage the same session twice, which would drain into two records
// for one session. Idempotency is keyed on content, not on the session id alone:
// the different-bytes case is TestStageReplacesStaleCopyOnDifferentContent.
func TestStageIsIdempotentPerSession(t *testing.T) {
	_, home := setupStore(t)
	if _, err := Stage(testRootSHA, mainStage("sess-dup"), []byte("one\n")); err != nil {
		t.Fatal(err)
	}
	second, err := Stage(testRootSHA, mainStage("sess-dup"), []byte("one\n"))
	if err != nil {
		t.Fatalf("second Stage failed: %v", err)
	}
	if second.Wrote || second.Replaced {
		t.Errorf("expected Wrote=false, Replaced=false when the session is already staged with identical bytes, got %+v", second)
	}
	if names := stagedNames(t, home); len(names) != 1 {
		t.Errorf("expected 1 staged file after a duplicate stage, got %v", names)
	}
}

// TestStageReplacesStaleCopyOnDifferentContent is the GHSA-xq36-hcgf-9wrj
// data-loss limb. A second SessionEnd for one session carrying DIFFERENT bytes (a
// harness retry, a later snapshot) must replace the staged copy — last-writer-
// wins, since the fresher end-of-session bytes are the ones worth keeping — not
// be dropped as a no-op that then drains the stale prefix and deletes the only
// copy of the newer transcript.
func TestStageReplacesStaleCopyOnDifferentContent(t *testing.T) {
	repoRoot, home := setupStore(t)
	if _, err := Stage(testRootSHA, mainStage("sess-restage"), []byte("one\n")); err != nil {
		t.Fatal(err)
	}
	second, err := Stage(testRootSHA, mainStage("sess-restage"), []byte("one\ntwo\n"))
	if err != nil {
		t.Fatalf("second Stage failed: %v", err)
	}
	if !second.Wrote {
		t.Error("expected Wrote=true when the session is re-staged with different bytes")
	}
	if !second.Replaced || second.ReplacedBytes != int64(len("one\n")) {
		t.Errorf("expected Replaced=true with ReplacedBytes=%d, got %+v", len("one\n"), second)
	}
	names := stagedNames(t, home)
	if len(names) != 1 {
		t.Fatalf("expected exactly 1 staged file after a re-stage, got %v", names)
	}
	sdir := filepath.Join(home, ".abcd", "history", testRootSHA, "staging")
	body, err := os.ReadFile(filepath.Join(sdir, names[0]))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "one\ntwo\n" {
		t.Errorf("staged file holds %q, want the newer bytes %q", body, "one\ntwo\n")
	}

	if _, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil {
		t.Fatalf("Drain: %v", err)
	}
	_, stored, err := Read(testRootSHA, "sess-restage")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !strings.Contains(string(stored), "two") {
		t.Errorf("the store holds the stale transcript, not the re-staged one:\n%s", stored)
	}
}

// TestStageConcurrentSameSessionYieldsOneCopy is the GHSA-xq36-hcgf-9wrj
// duplicate-records limb. Concurrent SessionEnds for one session id must leave
// exactly one staged copy, so the drain cannot store two records claiming the
// same session. flock is per open-file-description, so in-process goroutines
// contend through the staging lock exactly as separate hook processes do — the
// barrier-race shape of ahoy's lockrace_test.
func TestStageConcurrentSameSessionYieldsOneCopy(t *testing.T) {
	repoRoot, _ := setupStore(t)
	const writers = 8
	// Three rounds, not twenty: the unlocked Stage loses this race on the first
	// round every time (five runs out of five with the lock removed), and every
	// round costs ~1 s here and twice that under -race, on every CI leg.
	const rounds = 3
	var last string
	for round := 0; round < rounds; round++ {
		id := "sess-race-" + strconv.Itoa(round)
		last = id
		start := make(chan struct{})
		var wg sync.WaitGroup
		errs := make(chan error, writers)
		for i := 0; i < writers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				if _, err := Stage(testRootSHA, mainStage(id), []byte("copy "+strconv.Itoa(i)+"\n")); err != nil {
					errs <- err
				}
			}(i)
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Errorf("round %d: Stage: %v", round, err)
		}
		staged, err := ListStaged(testRootSHA)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, s := range staged {
			if s.SessionID == id {
				n++
			}
		}
		if n != 1 {
			t.Fatalf("round %d: %d staged copies for one session id; the stage handshake is not exclusive", round, n)
		}
	}

	res, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if len(res.Failed) != 0 {
		t.Fatalf("drain failures: %+v", res.Failed)
	}
	recs, err := List(testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, r := range recs {
		if r.SessionID == last {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("%d records claim session %s, want exactly 1", n, last)
	}
}

// TestDrainCapturesRedactedAndRemovesStaged is the round trip: what staging
// wrote raw must reach the store redacted, and the raw copy must be gone.
func TestDrainCapturesRedactedAndRemovesStaged(t *testing.T) {
	repoRoot, home := setupStore(t)
	secret := "ghp_" + strings.Repeat("b", 36)
	if _, err := Stage(testRootSHA, mainStage("sess-drain"), []byte("assistant: "+secret+"\n")); err != nil {
		t.Fatal(err)
	}

	res, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatalf("Drain failed: %v", err)
	}
	if len(res.Failed) != 0 {
		t.Fatalf("unexpected drain failures: %+v", res.Failed)
	}
	if len(res.Captured) != 1 {
		t.Fatalf("expected 1 captured record, got %d", len(res.Captured))
	}
	if names := stagedNames(t, home); len(names) != 0 {
		t.Errorf("staged copy survived a successful drain: %v", names)
	}
	recs, err := List(testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].SessionID != "sess-drain" {
		t.Fatalf("store does not hold the drained session: %+v", recs)
	}
	body, err := os.ReadFile(recs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), secret) {
		t.Fatal("drained record contains the live secret; the store invariant is broken")
	}
}

// TestDrainBudgetLeavesRemainderLoudly pins that a bounded pass reports what it
// did not do. A silent cap would read as "everything is captured" while a
// backlog sat undrained — the exact silence this mechanism exists to end.
func TestDrainBudgetLeavesRemainderLoudly(t *testing.T) {
	repoRoot, home := setupStore(t)
	for _, id := range []string{"sess-a", "sess-b", "sess-c"} {
		if _, err := Stage(testRootSHA, mainStage(id), []byte("body "+id+"\n")); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Millisecond) // distinct stamps so ordering is stable
	}

	res, err := Drain(repoRoot, testRootSHA, DrainBudget{MaxEntries: 2})
	if err != nil {
		t.Fatalf("Drain failed: %v", err)
	}
	if len(res.Captured) != 2 {
		t.Fatalf("expected 2 captured under a budget of 2, got %d", len(res.Captured))
	}
	if res.Remaining != 1 {
		t.Errorf("Remaining = %d, want 1 — a budgeted pass must report the remainder", res.Remaining)
	}
	if names := stagedNames(t, home); len(names) != 1 {
		t.Errorf("expected 1 staged file left, got %v", names)
	}
}

// TestDrainKeepsStagedOnCaptureFailure is the fail-closed guarantee. If
// redaction cannot produce a clean record, the staged copy is the only copy abcd
// holds; deleting it would turn a reported failure into permanent silent loss.
func TestDrainKeepsStagedOnCaptureFailure(t *testing.T) {
	repoRoot, home := setupStore(t)
	if _, err := Stage(testRootSHA, mainStage("sess-fail"), []byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	// Break the store's transcripts dir so Capture cannot write.
	tdir := filepath.Join(home, ".abcd", "history", testRootSHA, "transcripts")
	if err := os.RemoveAll(tdir); err != nil {
		t.Fatal(err)
	}

	res, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatalf("Drain returned a hard error rather than a per-item failure: %v", err)
	}
	if len(res.Failed) != 1 {
		t.Fatalf("expected 1 recorded failure, got %+v", res.Failed)
	}
	if names := stagedNames(t, home); len(names) != 1 {
		t.Fatal("staged copy was removed despite a failed capture — the only copy would be gone")
	}
}

// TestListStagedIsTheEndedSignal pins the outcome axis the store never had. A
// staged entry means exactly one thing: this session ended and its capture has
// not completed. Absence of a record no longer has to be guessed at.
func TestListStagedIsTheEndedSignal(t *testing.T) {
	repoRoot, _ := setupStore(t)
	if got, err := ListStaged(testRootSHA); err != nil || len(got) != 0 {
		t.Fatalf("empty staging should list nothing: %v %v", got, err)
	}
	if _, err := Stage(testRootSHA, mainStage("sess-ended"), []byte("body\n")); err != nil {
		t.Fatal(err)
	}
	got, err := ListStaged(testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].SessionID != "sess-ended" {
		t.Fatalf("ListStaged = %+v, want one entry for sess-ended", got)
	}
	if _, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil {
		t.Fatal(err)
	}
	if got, err := ListStaged(testRootSHA); err != nil || len(got) != 0 {
		t.Fatalf("a drained session must leave staging: %+v %v", got, err)
	}
}

// TestSessionIDFromStagedRoundTrips guards the filename parse against session
// ids containing dashes, which the host's uuid-shaped ids always do.
func TestSessionIDFromStagedRoundTrips(t *testing.T) {
	id := "7d884491-9773-4621-b438-77e2ef3d3ed5"
	name := stagedFilename(time.Now().UTC(), id)
	if got := sessionIDFromStaged(name); got != id {
		t.Errorf("sessionIDFromStaged(%q) = %q, want %q", name, got, id)
	}
}

// TestDrainLeavesAReStagedCopyForTheNextPass is the mid-drain limb of
// GHSA-xq36-hcgf-9wrj, and the only test that exercises removeStagedIfUnchanged's
// content check: Drain reads the staged bytes, spends the redaction budget in
// Capture, and only then removes the file — so a SessionEnd that re-stages the
// session in that window would have its newer transcript deleted by a bare
// os.Remove, with the store holding only the stale prefix. That is permanent
// loss of the sole copy, so the removal compares the bytes on disk against the
// bytes it captured and leaves a replacement for the next pass.
//
// The interleave is deterministic rather than raced: scanGitleaks is a seam
// inside Capture, which is exactly the window between Drain's read and its
// removal, so staging the newer bytes from the seam replays the sequence with
// no timing dependency.
func TestDrainLeavesAReStagedCopyForTheNextPass(t *testing.T) {
	repoRoot, home := setupStore(t)
	const older = "user: older snapshot\n"
	const newer = "user: older snapshot\nuser: newer snapshot\n"
	if _, err := Stage(testRootSHA, mainStage("sess-middrain"), []byte(older)); err != nil {
		t.Fatal(err)
	}

	restore := scanGitleaks
	t.Cleanup(func() { scanGitleaks = restore })
	var restaged bool
	var restageErr error
	scanGitleaks = func(_, _, _ string) ([]scanner.Finding, error) {
		if !restaged {
			restaged = true
			if _, err := Stage(testRootSHA, mainStage("sess-middrain"), []byte(newer)); err != nil {
				restageErr = err
			}
		}
		return nil, nil
	}

	res, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if restageErr != nil {
		t.Fatalf("mid-drain Stage: %v", restageErr)
	}
	if !restaged {
		t.Fatal("the seam never fired; the interleave did not happen")
	}
	if len(res.Failed) != 0 {
		t.Fatalf("drain failures: %+v", res.Failed)
	}
	if len(res.Captured) != 1 {
		t.Fatalf("expected the older snapshot captured, got %d records", len(res.Captured))
	}

	names := stagedNames(t, home)
	if len(names) != 1 {
		t.Fatalf("the mid-drain re-stage was removed: staging holds %v; the newer transcript is gone for good", names)
	}
	sdir := filepath.Join(home, ".abcd", "history", testRootSHA, "staging")
	body, err := os.ReadFile(filepath.Join(sdir, names[0]))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != newer {
		t.Fatalf("staged file holds %q, want the re-staged bytes %q", body, newer)
	}

	// The next pass stores what the first one left behind.
	second, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatalf("second Drain: %v", err)
	}
	if len(second.Failed) != 0 {
		t.Fatalf("second drain failures: %+v", second.Failed)
	}
	if len(second.Captured) != 1 {
		t.Fatalf("the second pass captured %d records, want the re-staged transcript", len(second.Captured))
	}
	_, stored, err := Read(testRootSHA, "sess-middrain")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !strings.Contains(string(stored), "newer snapshot") {
		t.Errorf("the store never received the re-staged transcript:\n%s", stored)
	}
	if names := stagedNames(t, home); len(names) != 0 {
		t.Errorf("staging still holds %v after the transcript reached the store", names)
	}
}
