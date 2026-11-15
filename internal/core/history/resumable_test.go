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
	"github.com/intentdriven/abcd/internal/adapter/scanner/augmenttest"
)

// The transcripts here are made up: line-delimited JSON records with planted
// fake tokens and the test home's path, never text from a real session. No
// test prints a transcript, a part or a record.

// partsSecret is a fake token of the shape the scanner masks.
func partsSecret(i int) string {
	return fmt.Sprintf("ghp_%036d", i)
}

// partsTranscript is n record-shaped lines, every seventh carrying a fake
// token and every fifth the caller's home path.
func partsTranscript(home string, n int) (string, []string) {
	var b strings.Builder
	var leaks []string
	for i := 0; i < n; i++ {
		text := fmt.Sprintf("filler line %d of a long made-up session", i)
		if i%7 == 3 {
			s := partsSecret(i)
			leaks = append(leaks, s)
			text += " token " + s
		}
		if i%5 == 1 {
			text += " at " + home + "/work/file.go"
		}
		fmt.Fprintf(&b, "{\"type\":\"assistant\",\"n\":%d,\"text\":%q}\n", i, text)
	}
	return b.String(), append(leaks, home+"/work")
}

// fillerLines is n record-shaped lines carrying nothing the scanner masks.
func fillerLines(from, n int) string {
	var b strings.Builder
	for i := from; i < from+n; i++ {
		fmt.Fprintf(&b, "{\"n\":%d,\"text\":\"filler line %d of a made-up session\"}\n", i, i)
	}
	return b.String()
}

// The two halves of a made-up key block: a header line, and a line holding
// body bytes and the END marker. Only a redaction that sees both consumes the
// body.
var (
	pemHeaderLine = `{"text":"-----BEGIN ` + `OPENSSH PRIVATE KEY-----"}` + "\n"
	pemBody       = strings.Repeat("Q", 64)
	pemEndLine    = `{"text":"` + pemBody + ` -----END ` + `OPENSSH PRIVATE KEY-----"}` + "\n"
)

// smallParts makes parts tiny so a short transcript needs many of them.
func smallParts(t *testing.T, n int) {
	t.Helper()
	old := redactionPartBytes
	redactionPartBytes = n
	t.Cleanup(func() { redactionPartBytes = old })
}

// onStep replaces drainStep for the rest of the test.
func onStep(t *testing.T, f func(step string, covered int)) {
	t.Helper()
	old := drainStep
	drainStep = f
	t.Cleanup(func() { drainStep = old })
}

// stopAfterParts cancels ctx once a pass has kept k parts, and records the
// byte offset every kept part ended at.
func stopAfterParts(t *testing.T, k int, cancel context.CancelFunc, ends *[]int) {
	t.Helper()
	n := 0
	onStep(t, func(step string, end int) {
		if step != stepPartKept {
			return
		}
		*ends = append(*ends, end)
		n++
		if n%k == 0 {
			cancel()
		}
	})
}

// keepParts runs one pass that keeps k parts of what is staged and stops.
func keepParts(t *testing.T, repoRoot string, k int) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var ends []int
	stopAfterParts(t, k, cancel, &ends)
	res, err := DrainContext(ctx, repoRoot, testRootSHA, DrainBudget{MaxEntries: 1})
	drainStep = func(string, int) {}
	if err != nil || len(res.Failed) != 0 || len(res.Captured) != 0 || len(ends) != k {
		t.Fatalf("the pass meant to keep %d parts: %v failed=%d captured=%d kept=%d", k, err, len(res.Failed), len(res.Captured), len(ends))
	}
}

// killedAt runs a drain that a simulated kill ends at the nth time it reaches
// step, and reports whether the kill landed.
func killedAt(t *testing.T, repoRoot, step string, nth int) bool {
	t.Helper()
	n, landed := 0, false
	old := drainStep
	drainStep = func(s string, covered int) {
		old(s, covered)
		if s == step {
			n++
			if n == nth {
				landed = true
				panic("killed")
			}
		}
	}
	defer func() { drainStep = old }()
	func() {
		defer func() { _ = recover() }()
		_, _ = Drain(repoRoot, testRootSHA, DrainBudget{})
	}()
	return landed
}

// storeHolds reports which of leaks appear in any file under the store's
// records, and, when withParts, under staging/progress.
func storeHolds(t *testing.T, home string, leaks []string, withParts bool) []string {
	t.Helper()
	dirs := []string{abcdhome.Path(home, "transcripts", testRootSHA, "records")}
	if withParts {
		dirs = append(dirs, progressPath(home))
	}
	var found []string
	for _, d := range dirs {
		_ = filepath.WalkDir(d, func(p string, e os.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			for _, l := range leaks {
				if strings.Contains(string(data), l) {
					found = append(found, filepath.Base(p))
				}
			}
			return nil
		})
	}
	return found
}

func progressPath(home string) string {
	return abcdhome.Path(home, "transcripts", testRootSHA, "staging", progressDirName)
}

func stagingPath(home string) string {
	return abcdhome.Path(home, "transcripts", testRootSHA, "staging")
}

// sessionBody returns the body of the one record stored for session.
func sessionBody(t *testing.T, repoRoot, session string) string {
	t.Helper()
	recs, err := ListForSession(repoRoot, testRootSHA, session)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("want one record for %s, got %d", session, len(recs))
	}
	data, err := os.ReadFile(recs[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	_, body, err := parseRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// wholeBody captures text whole under a session of its own in the same
// store, for comparison with a record assembled from parts.
func wholeBody(t *testing.T, repoRoot, text string) string {
	t.Helper()
	session := fmt.Sprintf("sess-whole-%d", time.Now().UnixNano())
	if _, err := Capture(repoRoot, testRootSHA, []byte(text), CaptureMeta{SessionID: session, Kind: "native"}); err != nil {
		t.Fatal(err)
	}
	return sessionBody(t, repoRoot, session)
}

// partFiles counts the part files kept anywhere under staging/progress.
func partFiles(home string) int {
	n := 0
	_ = filepath.WalkDir(progressPath(home), func(p string, e os.DirEntry, err error) error {
		if err == nil && !e.IsDir() && strings.HasSuffix(p, partSuffix) {
			n++
		}
		return nil
	})
	return n
}

// noParts reports whether the progress directory holds anything at all.
func noParts(t *testing.T, home string) bool {
	t.Helper()
	entries, err := os.ReadDir(progressPath(home))
	return err != nil || len(entries) == 0
}

// stagedFor returns the one staged entry for session.
func stagedFor(t *testing.T, repoRoot, session string) Staged {
	t.Helper()
	staged, err := ListStaged(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range staged {
		if s.SessionID == session {
			return s
		}
	}
	t.Fatalf("nothing staged for %s", session)
	return Staged{}
}

// framedStaged is the framed text a drain splits for session's staged entry.
func framedStaged(t *testing.T, repoRoot, session, body string) string {
	t.Helper()
	meta, _, err := prepareCapture(repoRoot, testRootSHA, stagedFor(t, repoRoot, session).captureMeta())
	if err != nil {
		t.Fatal(err)
	}
	framed, err := frameLineage(meta, []byte(body))
	if err != nil {
		t.Fatal(err)
	}
	return framed
}

func stage(t *testing.T, repoRoot, session, text string) {
	t.Helper()
	if _, err := Stage(repoRoot, testRootSHA, mainStage(session), []byte(text)); err != nil {
		t.Fatal(err)
	}
}

// TestDrainFinishesATranscriptOverSeveralPasses: a transcript that needs more
// than one pass's time is finished over several passes, each continuing at the
// first byte no kept part covers, so no part is redacted twice; nothing
// unredacted reaches the store or the parts on the way; and the record is the
// one a whole capture of the same bytes writes.
func TestDrainFinishesATranscriptOverSeveralPasses(t *testing.T) {
	repoRoot, home := setupStore(t)
	smallParts(t, 1024)
	text, leaks := partsTranscript(home, 120)
	stage(t, repoRoot, "sess-long", text)
	var ends []int
	passes := 0
	for ; passes < 100; passes++ {
		ctx, cancel := context.WithCancel(context.Background())
		stopAfterParts(t, 2, cancel, &ends)
		res, err := DrainContext(ctx, repoRoot, testRootSHA, DrainBudget{MaxEntries: 1})
		cancel()
		if err != nil || len(res.Failed) != 0 {
			t.Fatalf("pass %d: %v %+v", passes, err, res.Failed)
		}
		if got := storeHolds(t, home, leaks, true); len(got) != 0 {
			t.Fatalf("pass %d: unredacted text in %v", passes, got)
		}
		if len(res.Captured) == 1 {
			break
		}
		if len(res.Unfinished) != 1 {
			t.Fatalf("pass %d stopped without naming what it left: %+v", passes, res)
		}
		if recs, _ := List(repoRoot, testRootSHA); len(recs) != 0 {
			t.Fatalf("pass %d stored a record before every part was done", passes)
		}
		if names := stagedNames(t, home); len(names) != 1 {
			t.Fatalf("pass %d: the staged original went before its record: %v", passes, names)
		}
	}
	if passes < 3 {
		t.Fatalf("finished in %d passes; the test needs a transcript longer than one pass", passes)
	}
	for i := 1; i < len(ends); i++ {
		if ends[i] <= ends[i-1] {
			t.Fatalf("part ends %v: a later pass redid work an earlier one kept", ends)
		}
	}
	if names := stagedNames(t, home); len(names) != 0 {
		t.Errorf("staged copy survived the record: %v", names)
	}
	if !noParts(t, home) {
		t.Error("parts survived the record")
	}
	if got, want := sessionBody(t, repoRoot, "sess-long"), wholeBody(t, repoRoot, text); got != want {
		t.Errorf("the record assembled from parts differs from a whole capture of the same bytes")
	}
}

// TestDrainPartsMatchAWholeRedactionAcrossAPEMBlock: a PEM header on the
// last line of a part closes on an END marker on the next line, which only a
// redaction of both lines together consumes; the part runs on to the next cut,
// so the record matches a whole capture and keeps no key line.
func TestDrainPartsMatchAWholeRedactionAcrossAPEMBlock(t *testing.T) {
	repoRoot, _ := setupStore(t)
	text := fillerLines(0, 6) + pemHeaderLine + pemEndLine + fillerLines(100, 6)
	stage(t, repoRoot, "sess-pem-parts", text)
	// The first part is sized to end on the header's line, just before END,
	// in the text the drain splits: the transcript behind its lineage frame.
	framed := framedStaged(t, repoRoot, "sess-pem-parts", text)
	split := strings.Index(framed, pemEndLine)
	smallParts(t, split)
	if got := nextCut(framed, 0, split); got != split {
		t.Fatalf("the first part ends at %d, want %d: the test needs it to end on the header", got, split)
	}
	res, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil || len(res.Captured) != 1 {
		t.Fatalf("drain: %v failed=%d captured=%d", err, len(res.Failed), len(res.Captured))
	}
	got := sessionBody(t, repoRoot, "sess-pem-parts")
	if strings.Contains(got, pemBody) {
		t.Error("a key line split across two parts reached the store")
	}
	if want := wholeBody(t, repoRoot, text); got != want {
		t.Errorf("parts redacted the key block differently from a whole capture")
	}
}

// TestDrainReusesNoFinalPartOnceTheTextGrew: a pass that kept every part,
// the last ending at the end of the text, and stopped before the record; the
// transcript is then staged again longer, and the new line closes a key block
// the old last line opened. The old last part is not reused: the record is the
// one a whole capture of the longer text writes, and holds no key bytes.
func TestDrainReusesNoFinalPartOnceTheTextGrew(t *testing.T) {
	repoRoot, home := setupStore(t)
	smallParts(t, 1024)
	short := fillerLines(0, 40) + pemHeaderLine
	stage(t, repoRoot, "sess-grow", short)
	if !killedAt(t, repoRoot, stepPartsDone, 1) {
		t.Fatal("the pass did not reach the end of its parts")
	}
	if partFiles(home) < 2 {
		t.Fatalf("the test needs the short text kept in several parts, got %d", partFiles(home))
	}
	long := short + pemEndLine + fillerLines(100, 5)
	stage(t, repoRoot, "sess-grow", long)
	if framed := framedStaged(t, repoRoot, "sess-grow", long); !safeCut(framed, len(framedStaged(t, repoRoot, "sess-grow", short))) {
		t.Fatal("the test needs the old end of text to be a safe cut of the longer text")
	}
	res, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil || len(res.Captured) != 1 {
		t.Fatalf("drain: %v failed=%d captured=%d", err, len(res.Failed), len(res.Captured))
	}
	got := sessionBody(t, repoRoot, "sess-grow")
	if strings.Contains(got, pemBody) {
		t.Error("key bytes after a kept final part reached the store")
	}
	if want := wholeBody(t, repoRoot, long); got != want {
		t.Error("the record of a grown text differs from a whole capture of it")
	}
}

// TestDrainRedoesPartsOfAChangedStagedFile: parts kept for one text are not
// reused for the bytes staged in their place, whether the staged file was
// re-staged with other bytes or a directory of parts belongs to another
// transcript that had the same staged name.
func TestDrainRedoesPartsOfAChangedStagedFile(t *testing.T) {
	t.Run("re-staged with other bytes", func(t *testing.T) {
		repoRoot, home := setupStore(t)
		smallParts(t, 1024)
		text, _ := partsTranscript(home, 60)
		stage(t, repoRoot, "sess-change", text)
		keepParts(t, repoRoot, 2)
		changed := strings.Replace(text, "filler line 1 ", "filler line X ", 1)
		if len(changed) != len(text) || changed == text {
			t.Fatal("the test needs a same-length change in the first part")
		}
		stage(t, repoRoot, "sess-change", changed)
		if _, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil {
			t.Fatal(err)
		}
		if got, want := sessionBody(t, repoRoot, "sess-change"), wholeBody(t, repoRoot, changed); got != want {
			t.Error("parts of the old bytes went into the record of the new ones")
		}
	})
	t.Run("a stale directory of the same name", func(t *testing.T) {
		repoRoot, home := setupStore(t)
		smallParts(t, 1024)
		old, _ := partsTranscript(home, 60)
		stage(t, repoRoot, "sess-old", old)
		keepParts(t, repoRoot, 2)
		oldName, _ := partsDirName(stagedFor(t, repoRoot, "sess-old").Path)
		parked := filepath.Join(t.TempDir(), "parked")
		if err := os.Rename(filepath.Join(progressPath(home), oldName), parked); err != nil {
			t.Fatal(err)
		}
		if _, err := Discard(repoRoot, testRootSHA, filepath.Base(stagedFor(t, repoRoot, "sess-old").Path)); err != nil {
			t.Fatal(err)
		}
		text := strings.Replace(old, "made-up session", "made-up meeting", -1)
		stage(t, repoRoot, "sess-new", text)
		newName, _ := partsDirName(stagedFor(t, repoRoot, "sess-new").Path)
		if err := os.Rename(parked, filepath.Join(progressPath(home), newName)); err != nil {
			t.Fatal(err)
		}
		if _, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil {
			t.Fatal(err)
		}
		if got, want := sessionBody(t, repoRoot, "sess-new"), wholeBody(t, repoRoot, text); got != want {
			t.Error("another transcript's parts went into the record")
		}
	})
}

// TestDrainUnderAnAugmenterRedactsTheWholeText: with an augmenter armed the
// transcript is one part, read by the augmenter whole; armed after parts were
// kept without it, those parts are not used and the pass starts from byte
// zero. Either way the record is the one a whole capture under the augmenter
// writes, and holds nothing the augmenter flags.
func TestDrainUnderAnAugmenterRedactsTheWholeText(t *testing.T) {
	text := fillerLines(0, 3) + `{"text":"a value ` + augmenttest.Value + ` early on"}` + "\n" + fillerLines(10, 60)
	armed := func(t *testing.T) *[]int {
		t.Helper()
		var mu sync.Mutex
		var seen []int
		augmenttest.Install(t, &augmenttest.Func{F: func(in, file string) ([]scanner.Finding, error) {
			mu.Lock()
			seen = append(seen, len(in))
			mu.Unlock()
			return augmenttest.Locate(in, file, augmenttest.Value, augmenttest.Kind), nil
		}})
		return &seen
	}
	check := func(t *testing.T, repoRoot, home, session string, seen *[]int) {
		t.Helper()
		framed := framedStaged(t, repoRoot, session, text)
		res, err := DrainContext(context.Background(), repoRoot, testRootSHA, DrainBudget{MaxEntries: 1})
		if err != nil || len(res.Captured) != 1 {
			t.Fatalf("drain: %v failed=%d captured=%d", err, len(res.Failed), len(res.Captured))
		}
		if len(*seen) == 0 || (*seen)[0] != len(framed) {
			t.Errorf("the augmenter was not handed the whole text first: %v, want %d", *seen, len(framed))
		}
		if partFiles(home) != 0 {
			t.Error("parts were kept under an augmenter")
		}
		got := sessionBody(t, repoRoot, session)
		if strings.Contains(got, augmenttest.Value) {
			t.Error("a value the augmenter flags reached the store")
		}
		if want := wholeBody(t, repoRoot, text); got != want {
			t.Error("the record differs from a whole capture under the augmenter")
		}
	}
	t.Run("armed from the start", func(t *testing.T) {
		repoRoot, home := setupStore(t)
		smallParts(t, 1024)
		seen := armed(t)
		stage(t, repoRoot, "sess-aug", text)
		check(t, repoRoot, home, "sess-aug", seen)
	})
	t.Run("armed after parts were kept", func(t *testing.T) {
		repoRoot, home := setupStore(t)
		smallParts(t, 1024)
		stage(t, repoRoot, "sess-aug-late", text)
		keepParts(t, repoRoot, 2)
		seen := armed(t)
		check(t, repoRoot, home, "sess-aug-late", seen)
	})
}

// TestDrainRedoesPartsKeptUnderAnotherPatternSet: a pattern added to the
// repository's scanner config after parts were kept reaches the text those
// parts cover, as a whole capture under the new config would.
func TestDrainRedoesPartsKeptUnderAnotherPatternSet(t *testing.T) {
	repoRoot, _ := setupStore(t)
	smallParts(t, 1024)
	text := fillerLines(0, 2) + `{"text":"house value HOUSE-123456 early on"}` + "\n" + fillerLines(10, 60)
	stage(t, repoRoot, "sess-pattern", text)
	keepParts(t, repoRoot, 2)
	if err := os.MkdirAll(filepath.Join(repoRoot, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"patterns":{"house_token":{"regex":"HOUSE-[0-9]{6}","kind":"house_token","label":"house token","severity":"hard_fail"}}}`
	if err := os.WriteFile(filepath.Join(repoRoot, ".abcd", "config", "pii.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil {
		t.Fatal(err)
	}
	got := sessionBody(t, repoRoot, "sess-pattern")
	if strings.Contains(got, "HOUSE-123456") {
		t.Error("a part kept before the pattern existed went into the record")
	}
	if want := wholeBody(t, repoRoot, text); got != want {
		t.Error("the record differs from a whole capture under the new pattern set")
	}
}

// TestDrainKilledAtEachStepLeavesTheOriginalOrTheRecord: a pass killed at
// each step boundary of one entry's drain leaves the staged original or the
// whole record, never neither, and nothing unredacted in the store or the
// parts; the next pass then stores the transcript once, as a whole capture
// would, and leaves nothing staged or kept.
func TestDrainKilledAtEachStepLeavesTheOriginalOrTheRecord(t *testing.T) {
	for _, kill := range []struct {
		step string
		nth  int
	}{
		{stepPartKept, 1}, {stepPartKept, 3}, {stepPartsDone, 1}, {stepRecordWritten, 1}, {stepStagedRemoved, 1},
	} {
		t.Run(fmt.Sprintf("%s %d", kill.step, kill.nth), func(t *testing.T) {
			repoRoot, home := setupStore(t)
			smallParts(t, 1024)
			text, leaks := partsTranscript(home, 80)
			stage(t, repoRoot, "sess-kill", text)
			if !killedAt(t, repoRoot, kill.step, kill.nth) {
				t.Fatal("the simulated kill did not land")
			}
			recs, _ := List(repoRoot, testRootSHA)
			if len(stagedNames(t, home)) == 0 && len(recs) == 0 {
				t.Fatal("the kill left neither the staged original nor the record")
			}
			if got := storeHolds(t, home, leaks, true); len(got) != 0 {
				t.Fatalf("unredacted text in %v", got)
			}
			if res, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil || len(res.Failed) != 0 {
				t.Fatalf("the pass after the kill: %v failed=%d", err, len(res.Failed))
			}
			if names := stagedNames(t, home); len(names) != 0 {
				t.Errorf("staged copy left after the pass that followed the kill: %v", names)
			}
			if !noParts(t, home) {
				t.Error("parts left after the pass that followed the kill")
			}
			if got, want := sessionBody(t, repoRoot, "sess-kill"), wholeBody(t, repoRoot, text); got != want {
				t.Error("the record after a kill differs from a whole capture")
			}
		})
	}
}

// TestDrainResumesWithoutRedoingKeptParts: the pass after a kill between
// parts starts at the first byte the kept parts do not cover.
func TestDrainResumesWithoutRedoingKeptParts(t *testing.T) {
	repoRoot, home := setupStore(t)
	smallParts(t, 1024)
	text, _ := partsTranscript(home, 80)
	stage(t, repoRoot, "sess-resume", text)
	lastKept := 0
	onStep(t, func(step string, end int) {
		if step == stepPartKept {
			lastKept = end
		}
	})
	if !killedAt(t, repoRoot, stepPartKept, 3) {
		t.Fatal("the simulated kill did not land")
	}
	first := 0
	onStep(t, func(step string, end int) {
		if step == stepPartKept && first == 0 {
			first = end
		}
	})
	if res, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil || len(res.Captured) != 1 {
		t.Fatalf("drain: %v captured=%d", err, len(res.Captured))
	}
	if first == 0 || first <= lastKept {
		t.Errorf("the pass after the kill redid kept parts (its first part ended at %d, kept to %d)", first, lastKept)
	}
}

// TestDrainDoesNotTrustAPartCutShort: a part whose text was cut short on disk
// is redone, never assembled into the record.
func TestDrainDoesNotTrustAPartCutShort(t *testing.T) {
	repoRoot, home := setupStore(t)
	smallParts(t, 1024)
	text, _ := partsTranscript(home, 60)
	stage(t, repoRoot, "sess-cut", text)
	keepParts(t, repoRoot, 2)
	var first string
	_ = filepath.WalkDir(progressPath(home), func(p string, e os.DirEntry, err error) error {
		if err == nil && !e.IsDir() && first == "" {
			first = p
		}
		return nil
	})
	if first == "" {
		t.Fatal("no part was kept")
	}
	data, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first, data[:len(data)-40], 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil {
		t.Fatal(err)
	}
	if got, want := sessionBody(t, repoRoot, "sess-cut"), wholeBody(t, repoRoot, text); got != want {
		t.Error("a part cut short on disk went into the record")
	}
}

// TestDrainUnderASpentContextRunsEachEntrysFirstPart: a pass whose deadline
// has already passed still redacts the first part of the entry it reaches, as
// the whole transcript was before parts existed, and starts no part after
// that one (a one-part transcript is stored:
// TestDrainStartsNoFurtherEntryOnceItsBoundHasPassed).
func TestDrainUnderASpentContextRunsEachEntrysFirstPart(t *testing.T) {
	repoRoot, home := setupStore(t)
	smallParts(t, 1024)
	text, _ := partsTranscript(home, 60)
	stage(t, repoRoot, "sess-late", text)
	kept := 0
	onStep(t, func(step string, _ int) {
		if step == stepPartKept {
			kept++
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := DrainContext(ctx, repoRoot, testRootSHA, DrainBudget{MaxEntries: 1})
	if err != nil {
		t.Fatal(err)
	}
	if kept != 1 || len(res.Unfinished) != 1 || len(res.Captured) != 0 {
		t.Errorf("a spent context: kept %d part(s), unfinished %d, captured %d; want 1, 1, 0", kept, len(res.Unfinished), len(res.Captured))
	}
	if names := stagedNames(t, home); len(names) != 1 {
		t.Errorf("the staged original went before its record: %v", names)
	}
}

// TestSweepRemovesOnlyThePartsOfStagedFilesThatAreGone: the sweep keeps the
// parts of a file that is staged, whenever it was staged, removes the parts of
// one that is gone, also when nothing at all is staged, and leaves alone
// anything in the progress directory that is not a parts directory.
func TestSweepRemovesOnlyThePartsOfStagedFilesThatAreGone(t *testing.T) {
	repoRoot, home := setupStore(t)
	smallParts(t, 1024)
	text, _ := partsTranscript(home, 60)
	stage(t, repoRoot, "sess-sweep", text)
	keepParts(t, repoRoot, 2)
	sdir := stagingPath(home)
	sweepProgress(context.Background(), sdir)
	if partFiles(home) != 2 {
		t.Fatalf("the sweep removed the parts of a staged file: %d left", partFiles(home))
	}
	stray := filepath.Join(progressPath(home), "stray-file")
	if err := os.WriteFile(stray, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(progressPath(home), "not-parts")
	if err := os.Mkdir(other, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Discard(repoRoot, testRootSHA, filepath.Base(stagedFor(t, repoRoot, "sess-sweep").Path)); err != nil {
		t.Fatal(err)
	}
	if staged, _ := ListStaged(repoRoot, testRootSHA); len(staged) != 0 {
		t.Fatal("the test needs nothing staged")
	}
	if _, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil {
		t.Fatal(err)
	}
	if partFiles(home) != 0 {
		t.Error("a drain with nothing staged left the parts of a file that is gone")
	}
	for _, p := range []string{stray, other} {
		if _, err := os.Lstat(p); err != nil {
			t.Errorf("the sweep removed something that is not a parts directory: %s", filepath.Base(p))
		}
	}
}

// TestPartsDirectoriesAreRealDirectoriesOfTheirOwn: a progress directory, or a
// parts directory, that is a link is never read, written or emptied through,
// and costs the transcript nothing (the pass carries on in memory and names the
// fault); the parts directory's name can be neither empty nor a dot name.
func TestPartsDirectoriesAreRealDirectoriesOfTheirOwn(t *testing.T) {
	for _, c := range []struct {
		staged string
		ok     bool
	}{
		{"20261111T000000Z-sess.raw", true}, {".raw", true}, {"..raw", true}, {"x.txt", false}, {"", false},
	} {
		name, ok := partsDirName(filepath.Join("staging", c.staged))
		if ok != c.ok {
			t.Errorf("partsDirName(%q) ok=%v, want %v", c.staged, ok, c.ok)
			continue
		}
		if ok && (name == "" || name == "." || name == ".." || strings.Contains(name, "/") || filepath.Base(name) != name) {
			t.Errorf("partsDirName(%q) = %q, which is not a plain name", c.staged, name)
		}
	}
	for _, n := range []string{".", "..", "", "x", "x.parts", ".parts", "a/b.raw.parts"} {
		if _, ok := stagedNameForParts(n); ok {
			t.Errorf("stagedNameForParts accepted %q", n)
		}
	}

	// A sentinel outside the store stands for what a link would aim the
	// store at; the name it carries is one the sweep would remove.
	outside := func(t *testing.T) (dir, sentinel string) {
		t.Helper()
		dir = t.TempDir()
		sentinel = filepath.Join(dir, "20200101T000000Z-gone.raw.parts", partName(0))
		if err := os.MkdirAll(filepath.Dir(sentinel), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(sentinel, []byte("{}\nkept"), 0o600); err != nil {
			t.Fatal(err)
		}
		return dir, sentinel
	}
	intact := func(t *testing.T, dir, sentinel string, want int) {
		t.Helper()
		if _, err := os.Stat(sentinel); err != nil {
			t.Error("the store removed a file through a link")
		}
		n := 0
		_ = filepath.WalkDir(dir, func(_ string, e os.DirEntry, err error) error {
			if err == nil && !e.IsDir() {
				n++
			}
			return nil
		})
		if n != want {
			t.Errorf("the store wrote through a link: %d file(s) behind it, want %d", n, want)
		}
	}

	t.Run("progress is a link", func(t *testing.T) {
		repoRoot, home := setupStore(t)
		smallParts(t, 1024)
		dir, sentinel := outside(t)
		if err := os.MkdirAll(stagingPath(home), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(dir, progressPath(home)); err != nil {
			t.Fatal(err)
		}
		if _, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil {
			t.Fatal(err)
		}
		intact(t, dir, sentinel, 1)
		text, _ := partsTranscript(home, 60)
		stage(t, repoRoot, "sess-link", text)
		res, err := Drain(repoRoot, testRootSHA, DrainBudget{})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Failed) != 0 || len(res.Captured) != 1 || res.KeepFault == "" {
			t.Errorf("a drain that could keep no part: failed=%d captured=%d fault named %v; want 0, 1, true",
				len(res.Failed), len(res.Captured), res.KeepFault != "")
		}
		intact(t, dir, sentinel, 1)
	})
	t.Run("a parts directory is a link", func(t *testing.T) {
		repoRoot, home := setupStore(t)
		smallParts(t, 1024)
		dir, sentinel := outside(t)
		text, _ := partsTranscript(home, 60)
		stage(t, repoRoot, "sess-link", text)
		name, _ := partsDirName(stagedFor(t, repoRoot, "sess-link").Path)
		if err := os.MkdirAll(progressPath(home), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Dir(sentinel), filepath.Join(progressPath(home), name)); err != nil {
			t.Fatal(err)
		}
		staged := stagedFor(t, repoRoot, "sess-link").Path
		res, err := Drain(repoRoot, testRootSHA, DrainBudget{})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Failed) != 0 || len(res.Captured) != 1 || res.KeepFault == "" {
			t.Errorf("a drain whose parts directory is a link: failed=%d captured=%d fault named %v; want 0, 1, true",
				len(res.Failed), len(res.Captured), res.KeepFault != "")
		}
		intact(t, dir, sentinel, 1)
		removeProgress(stagingPath(home), staged)
		sweepProgress(context.Background(), stagingPath(home))
		intact(t, dir, sentinel, 1)
	})
}

// TestNextCutEndsBetweenRecords pins where a part may end.
func TestNextCutEndsBetweenRecords(t *testing.T) {
	rec := `{"a":1}` + "\n"
	text := rec + rec + rec + "plain line\n" + rec + rec + rec + rec
	if got := nextCut(text, 0, 1); got != 2*len(rec) {
		t.Errorf("first cut %d, want %d (two records behind it, one ahead)", got, 2*len(rec))
	}
	// Past "plain line" the next safe cut needs two record lines behind it.
	after := 3*len(rec) + len("plain line\n")
	if got := nextCut(text, 3*len(rec), 1); got != after+2*len(rec) {
		t.Errorf("cut after the plain line %d, want %d", got, after+2*len(rec))
	}
	if got := nextCut("no records\nhere\n", 0, 1); got != len("no records\nhere\n") {
		t.Errorf("text with no safe cut is one part, got %d", got)
	}
}
