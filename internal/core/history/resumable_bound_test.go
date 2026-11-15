package history

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// A pass under its hook's bound, a part that cannot be kept, a commit that
// loses the records lock, the reuse stamp, and the clean-up of kept parts. The
// transcripts here are made up, and no test prints a transcript, a part or a
// record.

// TestDrainRedactsAnUnsplittableTranscriptWholeUnderASpentBound: a
// transcript with no safe cut is redacted whole in the first pass that reaches
// it, under a spent bound too, as every transcript was before parts existed.
func TestDrainRedactsAnUnsplittableTranscriptWholeUnderASpentBound(t *testing.T) {
	repoRoot, _ := setupStore(t)
	smallParts(t, 1024)
	huge := `{"text":"` + strings.Repeat("made up words ", 400) + `"}` + "\n"
	stage(t, repoRoot, "sess-huge", huge)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := DrainContext(ctx, repoRoot, testRootSHA, DrainBudget{MaxEntries: 1})
	if err != nil || len(res.Captured) != 1 || len(res.Unfinished) != 0 {
		t.Fatalf("an unsplittable transcript was not stored whole: %v captured=%d unfinished=%d",
			err, len(res.Captured), len(res.Unfinished))
	}
}

// TestDrainFinishesAnEntryItBeganInThatPass: while its bound lasts, a pass
// finishes the entry it begins, part after part, whatever the size of a later
// part, so a transcript the drain could redact whole within the bound is
// stored at the same pass as when it was redacted whole.
func TestDrainFinishesAnEntryItBeganInThatPass(t *testing.T) {
	repoRoot, home := setupStore(t)
	smallParts(t, 1024)
	huge := `{"text":"` + strings.Repeat("made up words ", 400) + `"}` + "\n"
	text, _ := partsTranscript(home, 60)
	text += huge + fillerLines(100, 20)
	stage(t, repoRoot, "sess-one-pass", text)
	kept := 0
	onStep(t, func(step string, _ int) {
		if step == stepPartKept {
			kept++
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	res, err := DrainContext(ctx, repoRoot, testRootSHA, DrainBudget{MaxEntries: 1})
	if err != nil || len(res.Captured) != 1 || len(res.Unfinished) != 0 {
		t.Fatalf("a pass with its bound to spare did not finish the entry it began: %v captured=%d unfinished=%d",
			err, len(res.Captured), len(res.Unfinished))
	}
	if kept < 3 {
		t.Fatalf("the test needs a transcript of several parts, got %d", kept)
	}
	if got, want := sessionBody(t, repoRoot, "sess-one-pass"), wholeBody(t, repoRoot, text); got != want {
		t.Error("the record differs from a whole capture")
	}
}

// TestDrainStartsNoFurtherEntryOnceItsBoundHasPassed: past its bound a pass
// still runs its first entry's first part, as a pass did before it had a
// bound, but begins no further entry; what it did not reach is counted.
func TestDrainStartsNoFurtherEntryOnceItsBoundHasPassed(t *testing.T) {
	repoRoot, _ := setupStore(t)
	stage(t, repoRoot, "sess-first", `{"text":"made up one"}`+"\n")
	stage(t, repoRoot, "sess-second", `{"text":"made up two"}`+"\n")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := DrainContext(ctx, repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Captured) != 1 || res.Remaining != 1 {
		t.Errorf("a spent bound: captured %d, remaining %d; want 1 and 1", len(res.Captured), res.Remaining)
	}
}

// TestDrainKeepsAOnePartTranscriptWhoseCommitLosesTheLock: a transcript
// redacted whole in one part, whose record cannot be written because another
// capture holds the records lock past the bound, keeps that part, says all of
// it is done, and the next pass stores it without redacting it again.
func TestDrainKeepsAOnePartTranscriptWhoseCommitLosesTheLock(t *testing.T) {
	repoRoot, home := setupStore(t)
	text, leaks := partsTranscript(home, 4)
	stage(t, repoRoot, "sess-lock", text)
	store, err := Resolve(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	held, release := make(chan struct{}), make(chan struct{})
	var holder sync.WaitGroup
	onStep(t, func(step string, _ int) {
		if step != stepPartsDone {
			return
		}
		holder.Add(1)
		go func() {
			defer holder.Done()
			_ = withRepoLock(store.Records, func() error {
				close(held)
				<-release
				return nil
			})
		}()
		<-held
	})
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()
	res, err := DrainContext(ctx, repoRoot, testRootSHA, DrainBudget{})
	close(release)
	holder.Wait()
	if err != nil || len(res.Failed) != 0 || len(res.Captured) != 0 || len(res.Unfinished) != 1 {
		t.Fatalf("a commit that lost the lock: %v failed=%d captured=%d unfinished=%d", err, len(res.Failed), len(res.Captured), len(res.Unfinished))
	}
	if u := res.Unfinished[0]; u.Done != u.Total || partFiles(home) != 1 {
		t.Fatalf("the redacted part was not kept: %d of %d bytes, %d part(s)", u.Done, u.Total, partFiles(home))
	}
	if got := storeHolds(t, home, leaks, true); len(got) != 0 {
		t.Fatalf("unredacted text in %v", got)
	}
	onStep(t, func(string, int) {})
	redone := countRedactions(t)
	if res, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil || len(res.Captured) != 1 {
		t.Fatalf("the pass after: %v captured=%d", err, len(res.Captured))
	}
	if *redone != 0 {
		t.Errorf("the pass after redacted the kept part again (%d redaction(s))", *redone)
	}
	if got, want := sessionBody(t, repoRoot, "sess-lock"), wholeBody(t, repoRoot, text); got != want {
		t.Error("the record differs from a whole capture")
	}
}

// countRedactions counts the parts redacted for the rest of the test.
func countRedactions(t *testing.T) *int {
	t.Helper()
	n := 0
	old := redactPart
	redactPart = func(sc *scanner.Scanner, text, label string) (Redacted, error) {
		n++
		return old(sc, text, label)
	}
	t.Cleanup(func() { redactPart = old })
	return &n
}

// TestDrainCarriesOnWhenAPartCannotBeKept: a progress directory the drain
// cannot write in costs the transcript nothing: the pass redacts in memory to
// the record, as a whole capture would, past its bound too, since nothing it
// did could be kept for a later pass; it removes the staged copy, and names
// the fault once however many entries met it.
func TestDrainCarriesOnWhenAPartCannotBeKept(t *testing.T) {
	repoRoot, home := setupStore(t)
	smallParts(t, 1024)
	if err := os.MkdirAll(progressPath(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(progressPath(home), 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(progressPath(home), 0o700) })
	one, _ := partsTranscript(home, 60)
	two := strings.Replace(one, "made-up session", "made-up meeting", -1)
	three := strings.Replace(one, "made-up session", "made-up workshop", -1)
	stage(t, repoRoot, "sess-keep-one", one)
	stage(t, repoRoot, "sess-keep-two", two)
	stage(t, repoRoot, "sess-keep-three", three)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := DrainContext(ctx, repoRoot, testRootSHA, DrainBudget{MaxEntries: 1})
	if err != nil || len(res.Unfinished) != 0 || len(res.Failed) != 0 || len(res.Captured) != 1 {
		t.Fatalf("a pass past its bound that could keep nothing: %v unfinished=%d failed=%d captured=%d",
			err, len(res.Unfinished), len(res.Failed), len(res.Captured))
	}
	if res.KeepFault == "" {
		t.Error("the pass past its bound did not name the fault")
	}

	res, err = Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil || len(res.Failed) != 0 || len(res.Captured) != 2 {
		t.Fatalf("a part that cannot be kept failed the entry: %v failed=%d captured=%d", err, len(res.Failed), len(res.Captured))
	}
	if res.KeepFault == "" || strings.Count(res.KeepFault, "keep a redacted part") != 1 {
		t.Error("the fault keeping a part was not named once")
	}
	if names := stagedNames(t, home); len(names) != 0 {
		t.Errorf("staged copies left: %v", names)
	}
	if got, want := sessionBody(t, repoRoot, "sess-keep-one"), wholeBody(t, repoRoot, one); got != want {
		t.Error("the record differs from a whole capture")
	}
}

// TestADrainThatFindsItsSourceStoredUnderTheLockWritesNothing: a peer stores
// the same staged source while a pass is between its last part and the
// records lock; the pass then writes nothing, supersedes nothing, and leaves
// the peer's record the only one. The peer here redacts under a pattern added
// meanwhile, so its record differs from the one the pass assembled and only
// the check under the lock keeps the two from standing side by side.
func TestADrainThatFindsItsSourceStoredUnderTheLockWritesNothing(t *testing.T) {
	repoRoot, home := setupStore(t)
	smallParts(t, 1024)
	text := fillerLines(0, 2) + `{"text":"house value HOUSE-123456 early on"}` + "\n" + fillerLines(10, 60)
	stage(t, repoRoot, "sess-twice", text)
	var peer DrainResult
	var peerErr error
	inPeer := false
	onStep(t, func(step string, _ int) {
		if step != stepPartsDone || inPeer {
			return
		}
		inPeer = true
		writeHouseToken(t, repoRoot)
		peer, peerErr = DrainContext(context.Background(), repoRoot, testRootSHA, DrainBudget{})
	})
	res, err := DrainContext(context.Background(), repoRoot, testRootSHA, DrainBudget{})
	if err != nil || peerErr != nil || len(res.Failed) != 0 || len(peer.Failed) != 0 {
		t.Fatalf("drains: %v / %v failed=%d/%d", err, peerErr, len(res.Failed), len(peer.Failed))
	}
	if !inPeer || len(peer.Captured) != 1 {
		t.Fatalf("the peer did not store the transcript first: %d", len(peer.Captured))
	}
	if len(res.Captured) != 0 {
		t.Error("the pass wrote a record for a source the peer had stored")
	}
	recs, _ := List(repoRoot, testRootSHA)
	if len(recs) != 1 || recs[0].Path != peer.Captured[0].Path {
		t.Fatalf("want the peer's record alone, got %d record(s)", len(recs))
	}
	if strings.Contains(sessionBody(t, repoRoot, "sess-twice"), "HOUSE-123456") {
		t.Error("the stored record holds the value the peer's pattern masks")
	}
	if names := stagedNames(t, home); len(names) != 0 {
		t.Errorf("staged copy left: %v", names)
	}
	if partFiles(home) != 0 {
		t.Error("parts left")
	}
}

// writeHouseToken adds a pattern for a made-up house token to the
// repository's scanner config.
func writeHouseToken(t *testing.T, repoRoot string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(repoRoot, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"patterns":{"house_token":{"regex":"HOUSE-[0-9]{6}","kind":"house_token","label":"house token","severity":"hard_fail"}}}`
	if err := os.WriteFile(filepath.Join(repoRoot, ".abcd", "config", "pii.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
}

// withBinaryStamp makes partsScanner see stamp as the running binary.
func withBinaryStamp(t *testing.T, stamp string) {
	t.Helper()
	old := binaryStamp
	binaryStamp = func() string { return stamp }
	t.Cleanup(func() { binaryStamp = old })
}

// firstKeptEnd runs one unbounded pass and returns where the first part it
// kept ended, 0 when it kept none.
func firstKeptEnd(t *testing.T, repoRoot string) int {
	t.Helper()
	first := 0
	onStep(t, func(step string, end int) {
		if step == stepPartKept && first == 0 {
			first = end
		}
	})
	if res, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil || len(res.Captured) != 1 {
		t.Fatalf("drain: %v captured=%d", err, len(res.Captured))
	}
	return first
}

// TestDrainRedoesPartsKeptUnderAnotherBinaryOrHome: parts kept by another
// binary, or under another home path, are redone from byte zero rather than
// reused; under the same ones they are reused.
func TestDrainRedoesPartsKeptUnderAnotherBinaryOrHome(t *testing.T) {
	setup := func(t *testing.T) (repoRoot string, ends []int) {
		t.Helper()
		repoRoot, home := setupStore(t)
		smallParts(t, 1024)
		withBinaryStamp(t, "binary-one")
		text, _ := partsTranscript(home, 60)
		stage(t, repoRoot, "sess-stamp", text)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		stopAfterParts(t, 2, cancel, &ends)
		if _, err := DrainContext(ctx, repoRoot, testRootSHA, DrainBudget{MaxEntries: 1}); err != nil {
			t.Fatal(err)
		}
		if len(ends) != 2 {
			t.Fatalf("the test needs two parts kept, got %d", len(ends))
		}
		return repoRoot, ends
	}
	t.Run("same binary and home", func(t *testing.T) {
		repoRoot, ends := setup(t)
		if got := firstKeptEnd(t, repoRoot); got <= ends[1] {
			t.Errorf("parts kept under the same binary and home were redone (first part ended at %d, kept to %d)", got, ends[1])
		}
	})
	t.Run("another binary", func(t *testing.T) {
		repoRoot, ends := setup(t)
		withBinaryStamp(t, "binary-two")
		if got := firstKeptEnd(t, repoRoot); got != ends[0] {
			t.Errorf("parts kept by another binary were reused (first part ended at %d, want %d)", got, ends[0])
		}
	})
	t.Run("another home", func(t *testing.T) {
		repoRoot, ends := setup(t)
		t.Setenv(abcdhome.StageHomeEnv, os.Getenv("HOME"))
		t.Setenv(abcdhome.PersonHomeEnv, filepath.Join(t.TempDir(), "made-up-person"))
		if got := firstKeptEnd(t, repoRoot); got != ends[0] {
			t.Errorf("parts kept under another home were reused (first part ended at %d, want %d)", got, ends[0])
		}
	})
}

// TestTwoBinariesDrainingInTurnStoreOneCorrectRecord: two sessions on two
// builds drain one transcript in turn, each under a bound that stops it after
// a few parts and each redoing what the other kept; the transcript still ends
// as one record, the one a whole capture writes, with nothing kept or staged.
func TestTwoBinariesDrainingInTurnStoreOneCorrectRecord(t *testing.T) {
	repoRoot, home := setupStore(t)
	smallParts(t, 1024)
	text, leaks := partsTranscript(home, 80)
	stage(t, repoRoot, "sess-turns", text)
	for pass := 0; pass < 4; pass++ {
		withBinaryStamp(t, fmt.Sprintf("binary-%d", pass%2))
		ctx, cancel := context.WithCancel(context.Background())
		var ends []int
		stopAfterParts(t, 2, cancel, &ends)
		res, err := DrainContext(ctx, repoRoot, testRootSHA, DrainBudget{MaxEntries: 1})
		cancel()
		if err != nil || len(res.Failed) != 0 || len(res.Captured) != 0 {
			t.Fatalf("pass %d: %v failed=%d captured=%d", pass, err, len(res.Failed), len(res.Captured))
		}
		if got := storeHolds(t, home, leaks, true); len(got) != 0 {
			t.Fatalf("pass %d: unredacted text in %v", pass, got)
		}
	}
	onStep(t, func(string, int) {})
	withBinaryStamp(t, "binary-0")
	if res, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil || len(res.Captured) != 1 {
		t.Fatalf("the unbounded pass: %v captured=%d", err, len(res.Captured))
	}
	if recs, _ := List(repoRoot, testRootSHA); len(recs) != 1 {
		t.Fatalf("want one record, got %d", len(recs))
	}
	if got, want := sessionBody(t, repoRoot, "sess-turns"), wholeBody(t, repoRoot, text); got != want {
		t.Error("the record differs from a whole capture")
	}
	if names := stagedNames(t, home); len(names) != 0 || partFiles(home) != 0 {
		t.Errorf("left %d staged and %d part(s)", len(names), partFiles(home))
	}
}

// TestExecutableStampIsTheBinarysBytes: the stamp of a readable executable is
// the sha256 of its bytes; one that cannot be read, or is no longer the file
// the process started from, gets a stamp unique to the call, which matches no
// other.
func TestExecutableStampIsTheBinarysBytes(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "made-up-binary")
	if err := os.WriteFile(exe, []byte("made-up binary bytes"), 0o700); err != nil {
		t.Fatal(err)
	}
	at, err := os.Stat(exe)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := executableStamp(exe, at), "sha256:"+digest("made-up binary bytes"); got != want {
		t.Errorf("the stamp of a readable binary is not the sha256 of its bytes")
	}
	other := filepath.Join(dir, "made-up-other")
	if err := os.WriteFile(other, []byte("made-up binary bytes"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, stamp := range map[string]func() string{
		"unreadable": func() string { return executableStamp(filepath.Join(dir, "absent"), at) },
		"swapped":    func() string { return executableStamp(other, at) },
		"unknown":    func() string { return executableStamp(exe, nil) },
	} {
		a, b := stamp(), stamp()
		if a == b || strings.HasPrefix(a, "sha256:") {
			t.Errorf("%s: the stamp is not unique to the call", name)
		}
	}
}

// TestDiscardRemovesTheKeptParts: discarding a staged transcript a drain had
// part-redacted removes its kept parts too, so nothing of it is left.
func TestDiscardRemovesTheKeptParts(t *testing.T) {
	repoRoot, home := setupStore(t)
	smallParts(t, 1024)
	text, _ := partsTranscript(home, 60)
	stage(t, repoRoot, "sess-discard", text)
	keepParts(t, repoRoot, 2)
	if _, err := Discard(repoRoot, testRootSHA, filepath.Base(stagedFor(t, repoRoot, "sess-discard").Path)); err != nil {
		t.Fatal(err)
	}
	if !noParts(t, home) {
		t.Error("a discard left the parts a drain kept of the transcript")
	}
}

// TestSweepRetiresItsLocksSkipsAContendedOrphanAndStopsWithItsPass: the
// sweep removes the lock file it took for an orphan, leaves an orphan whose
// lock someone holds for a later sweep, and removes nothing once its pass's
// context has ended.
func TestSweepRetiresItsLocksSkipsAContendedOrphanAndStopsWithItsPass(t *testing.T) {
	_, home := setupStore(t)
	sdir := stagingPath(home)
	orphan := func(t *testing.T, staged string) string {
		t.Helper()
		dir := filepath.Join(progressPath(home), staged+partsDirSuffix)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, partName(0)), []byte("{}\nredacted"), 0o600); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	gone := orphan(t, "20200101T000000Z-gone.raw")
	held := orphan(t, "20200101T000000Z-held.raw")

	ended, cancel := context.WithCancel(context.Background())
	cancel()
	sweepProgress(ended, sdir)
	for _, d := range []string{gone, held} {
		if _, err := os.Lstat(d); err != nil {
			t.Fatal("a sweep whose pass had ended removed parts")
		}
	}

	heldKey := stagedLockKey(filepath.Join(sdir, "20200101T000000Z-held.raw"))
	if err := os.MkdirAll(filepath.Join(sdir, stagingLocksDirName), 0o700); err != nil {
		t.Fatal(err)
	}
	locked, release, swept := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		_ = fsutil.WithFileLock(stagingLockPath(sdir, heldKey), time.Minute, func() error {
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked
	go func() {
		defer close(swept)
		sweepProgress(context.Background(), sdir)
	}()
	<-swept
	close(release)
	if _, err := os.Lstat(gone); err == nil {
		t.Error("the sweep left the parts of a staged file that is gone")
	}
	if _, err := os.Lstat(held); err != nil {
		t.Error("the sweep removed parts whose lock someone held")
	}
	goneKey := stagedLockKey(filepath.Join(sdir, "20200101T000000Z-gone.raw"))
	if _, err := os.Lstat(stagingLockPath(sdir, goneKey)); err == nil {
		t.Error("the sweep left the lock file it took")
	}
}

// TestPartsScannerFollowsTheHomeAndTheBinary: the stamp a part carries
// changes with the caller's home and with the running binary, one scanner
// held fixed, so neither change can be hidden behind the scanner's own
// fingerprint.
func TestPartsScannerFollowsTheHomeAndTheBinary(t *testing.T) {
	repoRoot, home := setupStore(t)
	sc, err := redactionScanner(context.Background(), repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	withBinaryStamp(t, "binary-one")
	base := partsScanner(sc)
	if partsScanner(sc) != base {
		t.Fatal("one scanner, home and binary gave two stamps")
	}
	withBinaryStamp(t, "binary-two")
	if partsScanner(sc) == base {
		t.Error("another binary left the stamp unchanged")
	}
	withBinaryStamp(t, "binary-one")
	t.Setenv(abcdhome.StageHomeEnv, home)
	t.Setenv(abcdhome.PersonHomeEnv, filepath.Join(t.TempDir(), "made-up-person"))
	if partsScanner(sc) == base {
		t.Error("another home left the stamp unchanged")
	}
}

// TestDrainRunsAnEntryToItsRecordWhenNoLaterPassCouldUseItsParts: under a
// binary whose stamp is unique to the process, nothing a pass keeps is used
// by a later one, so a pass past its bound neither stops before a later part
// nor gives up on the records lock at the bound: it stores the transcript, as
// a capture did before parts existed.
func TestDrainRunsAnEntryToItsRecordWhenNoLaterPassCouldUseItsParts(t *testing.T) {
	t.Run("a later part past the bound", func(t *testing.T) {
		repoRoot, home := setupStore(t)
		smallParts(t, 1024)
		withBinaryStamp(t, processStampPrefix+"made-up")
		text, leaks := partsTranscript(home, 60)
		stage(t, repoRoot, "sess-process", text)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		res, err := DrainContext(ctx, repoRoot, testRootSHA, DrainBudget{MaxEntries: 1})
		if err != nil || len(res.Captured) != 1 || len(res.Unfinished) != 0 {
			t.Fatalf("a pass whose parts no later pass could use stopped: %v captured=%d unfinished=%d",
				err, len(res.Captured), len(res.Unfinished))
		}
		if got := storeHolds(t, home, leaks, true); len(got) != 0 {
			t.Fatalf("unredacted text in %v", got)
		}
		if got, want := sessionBody(t, repoRoot, "sess-process"), wholeBody(t, repoRoot, text); got != want {
			t.Error("the record differs from a whole capture")
		}
	})
	t.Run("the records lock past the bound", func(t *testing.T) {
		repoRoot, home := setupStore(t)
		withBinaryStamp(t, processStampPrefix+"made-up")
		text, _ := partsTranscript(home, 4)
		stage(t, repoRoot, "sess-process-lock", text)
		store, err := Resolve(repoRoot, testRootSHA)
		if err != nil {
			t.Fatal(err)
		}
		held := make(chan struct{})
		var holder sync.WaitGroup
		onStep(t, func(step string, _ int) {
			if step != stepPartsDone {
				return
			}
			holder.Add(1)
			go func() {
				defer holder.Done()
				_ = withRepoLock(store.Records, func() error {
					close(held)
					// Held past the least wait a bounded pass gives the lock.
					time.Sleep(4 * minLockWait)
					return nil
				})
			}()
			<-held
		})
		ctx, cancel := context.WithDeadline(context.Background(), time.Now())
		defer cancel()
		res, err := DrainContext(ctx, repoRoot, testRootSHA, DrainBudget{})
		holder.Wait()
		if err != nil || len(res.Failed) != 0 || len(res.Captured) != 1 || len(res.Unfinished) != 0 {
			t.Fatalf("a pass whose part no later pass could use gave up on the lock: %v failed=%d captured=%d unfinished=%d",
				err, len(res.Failed), len(res.Captured), len(res.Unfinished))
		}
	})
}

// TestDrainFinishesASlowLaterPartStartedBeforeTheBound: a later part started
// before the bound runs to its end however long it takes, and once the bound
// has passed no part starts after it. So a slow last part is stored in that
// pass, and a slow part with more after it is kept, with what follows it left
// for the next pass. The bound passes inside the long part's redaction (the
// redactPart seam ends the pass's context there), so no test waits on a clock.
func TestDrainFinishesASlowLaterPartStartedBeforeTheBound(t *testing.T) {
	long := `{"text":"` + strings.Repeat("made up words ", 3000) + `"}` + "\n"
	head, tail := fillerLines(0, 30), fillerLines(100, 30)
	// pass drains text under a bound that passes while its long part is
	// redacted, and returns what it did and how many parts started after.
	pass := func(t *testing.T, repoRoot, session, text string) (DrainResult, int) {
		t.Helper()
		smallParts(t, 1024)
		stage(t, repoRoot, session, text)
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Hour))
		defer cancel()
		slow, after := 0, 0
		old := redactPart
		redactPart = func(sc *scanner.Scanner, part, label string) (Redacted, error) {
			if slow > 0 {
				after++
			}
			red, err := old(sc, part, label)
			if len(part) >= len(long) {
				slow++
				cancel()
			}
			return red, err
		}
		res, err := DrainContext(ctx, repoRoot, testRootSHA, DrainBudget{MaxEntries: 1})
		redactPart = old
		if err != nil || len(res.Failed) != 0 {
			t.Fatalf("the pass: %v failed=%d", err, len(res.Failed))
		}
		if slow != 1 {
			t.Fatalf("the test needs the long line redacted as one later part, got %d such part(s)", slow)
		}
		return res, after
	}
	t.Run("the last part", func(t *testing.T) {
		repoRoot, _ := setupStore(t)
		text := head + long
		res, after := pass(t, repoRoot, "sess-slow-last", text)
		if len(res.Captured) != 1 || len(res.Unfinished) != 0 {
			t.Fatalf("a slow last part started before the bound was not stored in that pass: captured=%d unfinished=%d",
				len(res.Captured), len(res.Unfinished))
		}
		if after != 0 {
			t.Errorf("%d part(s) started after the bound", after)
		}
		if got, want := sessionBody(t, repoRoot, "sess-slow-last"), wholeBody(t, repoRoot, text); got != want {
			t.Error("the record differs from a whole capture")
		}
	})
	t.Run("a part with more after it", func(t *testing.T) {
		repoRoot, _ := setupStore(t)
		res, after := pass(t, repoRoot, "sess-slow-more", head+long+tail)
		if len(res.Captured) != 0 || len(res.Unfinished) != 1 {
			t.Fatalf("captured=%d unfinished=%d, want the entry left unfinished", len(res.Captured), len(res.Unfinished))
		}
		if after != 0 {
			t.Errorf("%d part(s) started after the bound", after)
		}
		if u := res.Unfinished[0]; u.Total-u.Done != int64(len(tail)) {
			t.Errorf("the pass left %d bytes, want what follows the slow part (%d)", u.Total-u.Done, len(tail))
		}
	})
}
