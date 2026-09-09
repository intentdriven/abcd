package history

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// ageStage backdates a staged entry's sidecar and mtime so the entry reads as
// having been staged `age` ago. Both are moved: listStaged prefers the
// sidecar's stamp and falls back to the mtime, and a fixture that moved only
// one would pass for the wrong reason on the other path.
func ageStage(t *testing.T, rawPath string, age time.Duration) {
	t.Helper()
	when := time.Now().UTC().Add(-age)
	side := strings.TrimSuffix(rawPath, stagedSuffix) + stageSidecarSuffix
	if data, err := os.ReadFile(side); err == nil {
		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		doc["staged_at"] = when.Format(time.RFC3339Nano)
		out, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(side, out, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(rawPath, when, when); err != nil {
		t.Fatal(err)
	}
}

// TestStagedEntryPastTheLimitReportsOverdue is the age half of
// iss-2609090722466403. staging.go promised a staged file lived "only until the
// next session starts"; nothing measured that, so nothing could contradict it,
// and four files aged up to a fortnight on the author's own disk with every
// listing calling them "awaiting redaction".
func TestStagedEntryPastTheLimitReportsOverdue(t *testing.T) {
	_, _ = setupStore(t)
	fresh, err := Stage(testRootSHA, mainStage("sess-fresh"), []byte("fresh\n"))
	if err != nil {
		t.Fatal(err)
	}
	old, err := Stage(testRootSHA, mainStage("sess-old"), []byte("old\n"))
	if err != nil {
		t.Fatal(err)
	}
	ageStage(t, old.Staged.Path, StagedTTL+48*time.Hour)

	staged, err := ListStaged(testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, s := range staged {
		seen[s.SessionID] = s.Overdue
	}
	if len(staged) != 2 {
		t.Fatalf("want 2 staged entries, got %d", len(staged))
	}
	if !seen["sess-old"] {
		t.Errorf("a transcript staged %s ago is not reported overdue; the limit says nothing and the pile stays invisible", StagedTTL+48*time.Hour)
	}
	if seen["sess-fresh"] {
		t.Errorf("a transcript staged moments ago was reported overdue")
	}
	_ = fresh
}

// TestOverdueEntryIsNeverDeletedByAge is the constraint the age mechanism must
// not break. Losing the only copy of a transcript is worse than keeping it —
// that is the premise staging is built on — so an expiry that discarded raw
// text to make its own warning go away would be a regression dressed as a fix.
func TestOverdueEntryIsNeverDeletedByAge(t *testing.T) {
	repoRoot, home := setupStore(t)
	// A transcript whose capture cannot succeed, so nothing but expiry could
	// remove it: the store's transcripts dir is taken away after the stage.
	res, err := Stage(testRootSHA, mainStage("sess-ancient"), []byte("keep me\n"))
	if err != nil {
		t.Fatal(err)
	}
	ageStage(t, res.Staged.Path, 400*24*time.Hour)
	if err := os.RemoveAll(filepath.Join(home, ".abcd", "history", testRootSHA, "transcripts")); err != nil {
		t.Fatal(err)
	}
	if _, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if _, err := os.Stat(res.Staged.Path); err != nil {
		t.Fatalf("a year-old staged transcript was removed by age with nothing storing it: %v", err)
	}
}

// TestDrainTakesOverdueEntriesFirst pins the only thing age buys: priority. A
// repository whose sessions stage faster than one budgeted pass drains can
// starve its own oldest entry forever, and the oldest entry is precisely the
// raw text that has been on disk longest.
//
// The fixture is adversarial on the EXISTING order: the overdue entry is a
// sub-agent's and the fresh one is a main thread's, so main-thread-first
// ordering alone puts the fresh one at the head.
func TestDrainTakesOverdueEntriesFirst(t *testing.T) {
	repoRoot, _ := setupStore(t)
	if _, err := Stage(testRootSHA, mainStage("sess-new"), []byte("new spine\n")); err != nil {
		t.Fatal(err)
	}
	oldSub, err := Stage(testRootSHA, subAgentStage("sess-gone", "agent-gone"), []byte("old branch\n"))
	if err != nil {
		t.Fatal(err)
	}
	ageStage(t, oldSub.Staged.Path, StagedTTL+time.Hour)

	dr, err := Drain(repoRoot, testRootSHA, DrainBudget{MaxEntries: 1})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if len(dr.Captured) != 1 {
		t.Fatalf("want exactly 1 captured under a 1-entry budget, got %d", len(dr.Captured))
	}
	if dr.Captured[0].SessionID != "sess-gone" {
		t.Errorf("the one attempted entry was %q; the overdue transcript must go first or a busy repo starves its oldest raw file",
			dr.Captured[0].SessionID)
	}
	if dr.Overdue != 1 {
		t.Errorf("DrainResult.Overdue = %d, want 1 — the count is what a notice says out loud", dr.Overdue)
	}
}

// forceResidualRefusal makes Capture refuse every transcript with a
// *RedactionResidualError, the deterministic failure. The stub is the same one
// TestCaptureRefusesWhenAugmentedSpanIsNotMasked uses: a gitleaks finding whose
// span Redact cannot apply, so the stage-two re-scan finds it unmasked.
func forceResidualRefusal(t *testing.T) {
	t.Helper()
	restore := scanGitleaks
	t.Cleanup(func() { scanGitleaks = restore })
	scanGitleaks = func(_, _, logical string) ([]scanner.Finding, error) {
		return []scanner.Finding{{
			File: logical, Line: 999, Column: 1,
			Kind:     "gitleaks:generic-api-key",
			Severity: scanner.SeverityHardFail,
			// A value the NATIVE scanner does not recognise, so it survives
			// stage one unmasked and the stage-two re-scan finds it — which is
			// what makes the refusal deterministic rather than incidental.
			Matched: gitleaksResidueSecret,
		}}, nil
	}
}

// TestDeterministicRefusalIsQuarantinedNotRetriedForever is the fourth limb of
// iss-2609090722466403, and the one the review added. The drain left the staged
// file in place on ANY Capture failure, which is right for a transient fault
// and wrong for a *RedactionResidualError: the same bytes are refused by every
// future drain, so the raw copy stays unredacted forever AND every later pass
// re-reads and re-scans it to reach the same answer.
func TestDeterministicRefusalIsQuarantinedNotRetriedForever(t *testing.T) {
	repoRoot, home := setupStore(t)
	forceResidualRefusal(t)
	res, err := Stage(testRootSHA, mainStage("sess-refused"), []byte("api_key = "+gitleaksResidueSecret+"\n"))
	if err != nil {
		t.Fatal(err)
	}

	dr, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if len(dr.Failed) != 1 {
		t.Fatalf("want 1 reported failure, got %+v", dr.Failed)
	}
	f := dr.Failed[0]
	if !f.Permanent {
		t.Errorf("a redaction refusal was reported as retryable; nothing then distinguishes it from a transient fault: %+v", f)
	}
	if !f.Quarantined {
		t.Errorf("a permanently unstorable transcript was not quarantined: %+v", f)
	}
	if _, err := os.Stat(res.Staged.Path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the refused transcript is still in staging (%v); every later drain will re-read and re-refuse it", err)
	}
	qdir := filepath.Join(home, ".abcd", "history", testRootSHA, "quarantine")
	qpath := filepath.Join(qdir, filepath.Base(res.Staged.Path))
	body, err := os.ReadFile(qpath)
	if err != nil {
		t.Fatalf("the quarantined transcript is not readable at %s: %v", qpath, err)
	}
	if !strings.Contains(string(body), gitleaksResidueSecret) {
		t.Error("the quarantined copy lost its bytes; quarantine preserves the transcript, it does not discard it")
	}

	// The point of the terminal state: the next pass has nothing to do.
	second, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatalf("second Drain: %v", err)
	}
	if len(second.Failed) != 0 {
		t.Errorf("the quarantined transcript was retried: %+v", second.Failed)
	}
}

// TestQuarantineHoldsUnredactedTextAtOwnerOnlyModes: quarantine is a change of
// STATE, not of exposure. The bytes are as raw as they were in staging, so the
// directory and the files must be exactly as closed.
func TestQuarantineHoldsUnredactedTextAtOwnerOnlyModes(t *testing.T) {
	repoRoot, home := setupStore(t)
	forceResidualRefusal(t)
	if _, err := Stage(testRootSHA, mainStage("sess-modes"), []byte("api_key = "+gitleaksResidueSecret+"\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil {
		t.Fatal(err)
	}
	qdir := filepath.Join(home, ".abcd", "history", testRootSHA, "quarantine")
	fi, err := os.Stat(qdir)
	if err != nil {
		t.Fatalf("quarantine dir: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o700 {
		t.Errorf("quarantine dir mode = %o, want 700 (it holds unredacted transcripts)", perm)
	}
	entries, err := os.ReadDir(qdir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("%s mode = %o, want 600", e.Name(), perm)
		}
	}
}

// TestRetryableFailureStaysStagedAndRetryable is the other side of the split. A
// store path that is momentarily unusable is not a property of the transcript,
// so the entry must stay exactly where it is and stay queued.
func TestRetryableFailureStaysStagedAndRetryable(t *testing.T) {
	repoRoot, home := setupStore(t)
	res, err := Stage(testRootSHA, mainStage("sess-transient"), []byte("body\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(home, ".abcd", "history", testRootSHA, "transcripts")); err != nil {
		t.Fatal(err)
	}
	dr, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if len(dr.Failed) != 1 {
		t.Fatalf("want 1 reported failure, got %+v", dr.Failed)
	}
	if dr.Failed[0].Permanent || dr.Failed[0].Quarantined {
		t.Errorf("a missing store directory was treated as a permanent property of the transcript: %+v", dr.Failed[0])
	}
	if _, err := os.Stat(res.Staged.Path); err != nil {
		t.Errorf("a retryable failure moved the staged transcript: %v", err)
	}
	// And it really is retryable.
	if err := os.MkdirAll(filepath.Join(home, ".abcd", "history", testRootSHA, "transcripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	again, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Captured) != 1 {
		t.Errorf("the retryable entry did not capture on the retry: %+v", again)
	}
}

// TestCorruptSidecarIsNotTreatedAsPermanent: an operator can rewrite or delete
// a broken sidecar and the same bytes then drain normally, so it is a
// repairable fault and must NOT be parked in the terminal state. Getting this
// wrong would quarantine transcripts that were only ever one file rewrite from
// being stored.
func TestCorruptSidecarIsNotTreatedAsPermanent(t *testing.T) {
	repoRoot, _ := setupStore(t)
	res, err := Stage(testRootSHA, subAgentStage("sess-cs", "agent-cs"), []byte("body\n"))
	if err != nil {
		t.Fatal(err)
	}
	side := strings.TrimSuffix(res.Staged.Path, stagedSuffix) + stageSidecarSuffix
	if err := os.WriteFile(side, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	dr, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatal(err)
	}
	if len(dr.Failed) != 1 {
		t.Fatalf("want 1 failure, got %+v", dr.Failed)
	}
	if dr.Failed[0].Permanent {
		t.Error("a corrupt sidecar was called permanent; rewriting the sidecar fixes it, so nothing is terminal about it")
	}
	if _, err := os.Stat(res.Staged.Path); err != nil {
		t.Errorf("the staged transcript was moved for a repairable fault: %v", err)
	}
}

// TestSurveyBacklogSeesEveryRepositoryInTheStore is the third limb. `abcd
// history staged` answers for the repository the operator is standing in, which
// is by construction one whose staged files are being drained. The pile that
// grows without bound is in the repository nobody opens, and until this nothing
// in abcd could see it from anywhere.
func TestSurveyBacklogSeesEveryRepositoryInTheStore(t *testing.T) {
	_, home := setupStore(t)
	const otherSHA = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if err := os.MkdirAll(filepath.Join(home, ".abcd", "history", otherSHA, "transcripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".abcd", "history", otherSHA, "meta.json"),
		[]byte(`{"root_commit":"`+otherSHA+`","name":"a-quiet-repo"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Stage(testRootSHA, mainStage("sess-here"), []byte("here\n")); err != nil {
		t.Fatal(err)
	}
	quiet, err := Stage(otherSHA, mainStage("sess-there"), []byte("over there, forever\n"))
	if err != nil {
		t.Fatal(err)
	}
	ageStage(t, quiet.Staged.Path, StagedTTL+14*24*time.Hour)

	repos, err := SurveyBacklog()
	if err != nil {
		t.Fatalf("SurveyBacklog: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("want both repositories reported, got %d: %+v", len(repos), repos)
	}
	var other *RepoBacklog
	for i := range repos {
		if repos[i].RootSHA == otherSHA {
			other = &repos[i]
		}
	}
	if other == nil {
		t.Fatalf("the quiet repository is absent from the survey: %+v", repos)
	}
	if other.Name != "a-quiet-repo" {
		t.Errorf("Name = %q, want the repository's name from meta.json — a bare SHA is a riddle, not a notice", other.Name)
	}
	if other.Staged != 1 || other.Overdue != 1 {
		t.Errorf("staged=%d overdue=%d, want 1 and 1", other.Staged, other.Overdue)
	}
	if other.StagedBytes == 0 {
		t.Error("StagedBytes = 0; the size is the number that says whether to care")
	}
}

// TestSurveyBacklogSkipsRepositoriesHoldingNothing keeps the notice honest: a
// survey that listed every key in the store would report a row for every
// repository the operator has ever used, and a warning that always fires is
// read as noise and then not read at all.
func TestSurveyBacklogSkipsRepositoriesHoldingNothing(t *testing.T) {
	_, _ = setupStore(t)
	repos, err := SurveyBacklog()
	if err != nil {
		t.Fatalf("SurveyBacklog: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("a store with nothing staged anywhere reported %+v", repos)
	}
}

// TestDiscardRemovesOneTranscriptAndItsMetadata pins the only door that
// deletes. Quarantine without an exit is a room with no door: an operator with
// no sanctioned removal reaches for rm on a directory whose sidecar, reason
// note and lock file they have no reason to know about.
func TestDiscardRemovesOneTranscriptAndItsMetadata(t *testing.T) {
	repoRoot, home := setupStore(t)
	forceResidualRefusal(t)
	res, err := Stage(testRootSHA, subAgentStage("sess-d", "agent-d"), []byte("api_key = "+gitleaksResidueSecret+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Drain(repoRoot, testRootSHA, DrainBudget{}); err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(res.Staged.Path)
	qdir := filepath.Join(home, ".abcd", "history", testRootSHA, "quarantine")

	out, err := Discard(testRootSHA, name)
	if err != nil {
		t.Fatalf("Discard: %v", err)
	}
	if out.Bytes == 0 {
		t.Error("Discard reported 0 bytes; the operator is being told what was destroyed")
	}
	for _, leftover := range []string{
		filepath.Join(qdir, name),
		strings.TrimSuffix(filepath.Join(qdir, name), stagedSuffix) + stageSidecarSuffix,
		strings.TrimSuffix(filepath.Join(qdir, name), stagedSuffix) + quarantineSuffix,
	} {
		if _, err := os.Stat(leftover); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s survived the discard (%v); a half-removed entry is worse than an untouched one", leftover, err)
		}
	}
}

// TestDiscardRefusesAnythingButABareStagedFilename: this function deletes, and
// its argument comes from a command line. A path would let it delete outside
// the store entirely.
func TestDiscardRefusesAnythingButABareStagedFilename(t *testing.T) {
	_, _ = setupStore(t)
	for _, name := range []string{
		"../transcripts/20260101T000000.000000000Z-sess.md",
		"/etc/passwd",
		"sub/dir.raw",
		"notes.md",
		"",
		".lock",
	} {
		if _, err := Discard(testRootSHA, name); err == nil {
			t.Errorf("Discard(%q) was accepted; it must take a bare staged filename and nothing else", name)
		}
	}
}
