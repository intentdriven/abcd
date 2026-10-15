package embargo

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// seeded is a store holding one private record linked to one advisory,
// seeded directly so triage and close run without intake.
type seeded struct {
	f     fixture
	s     Store
	ghsa  string
	issID string
}

const seededTitle = "a sample report title"

func seedStore(t *testing.T) seeded {
	t.Helper()
	f := newFixture(t)
	s, _, err := openStore(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	res, err := capture.Capture(capture.CaptureRequest{
		RepoRoot: s.Root, Text: seededTitle + "\n\nthe sample body", Severity: capture.SeverityMajor,
		Category: "security", Source: "security-advisory", Slug: "private-report",
		FoundDuring: "embargo intake 2026-10-12", Remedy: issueschema.MachineRemedy,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Intake closes each record it mints to this account alone; so does the seed.
	if err := os.Chmod(filepath.Join(s.Root, filepath.FromSlash(res.Path)), 0o600); err != nil {
		t.Fatal(err)
	}
	st := emptyState()
	st.Repo = "example/example"
	st.Advisories = []Advisory{{GHSAID: "GHSA-xxxx-xxxx-xxxx", IssueID: res.ID, Title: seededTitle, Severity: "major", ForgeState: "triage"}}
	if err := s.SaveState(st); err != nil {
		t.Fatal(err)
	}
	return seeded{f: f, s: s, ghsa: "GHSA-xxxx-xxxx-xxxx", issID: res.ID}
}

func (sd seeded) triage(req TriageRequest) (TriageResult, error) {
	req.Home, req.RepoRoot = sd.f.home, sd.f.repo.Root()
	if req.Now == nil {
		req.Now = func() time.Time { return fixedNow }
	}
	return Triage(req)
}

func (sd seeded) state(t *testing.T) Advisory {
	t.Helper()
	st, err := sd.s.Verify()
	if err != nil {
		t.Fatal(err)
	}
	return st.Advisories[0]
}

func (sd seeded) recordIn(status string) bool {
	m, _ := filepath.Glob(filepath.Join(sd.s.Root, ".abcd", "work", "issues", status, sd.issID+"-*.md"))
	return len(m) == 1
}

func (sd seeded) draftPath() string {
	return filepath.Join(sd.s.Root, filepath.FromSlash(DraftDirRel), sd.ghsa+".md")
}

// snapshotStore is the bytes of the state file and the list of the store's
// ledger and local-tier files: what a refused triage must leave unchanged.
func (sd seeded) snapshot(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	data, _ := os.ReadFile(sd.s.StatePath())
	b.Write(data)
	for _, dir := range []string{".abcd/work/issues", ".abcd/.work.local"} {
		_ = filepath.Walk(filepath.Join(sd.s.Root, filepath.FromSlash(dir)), func(p string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && !strings.HasSuffix(p, ".lock") {
				b.WriteString(p + "\n")
			}
			return nil
		})
	}
	return b.String()
}

// TestTriageEachVerdict: each of the four verdicts is kept with who decided it
// and when, from the clock; the three verdicts that end the private track move
// the record to wontfix/ in the store, and the two closing ones draft text.
func TestTriageEachVerdict(t *testing.T) {
	for _, tc := range []struct {
		req       TriageRequest
		wontfix   bool
		draft     bool
		reasonHas string
	}{
		{TriageRequest{Verdict: VerdictVulnerability, By: "Alice"}, false, false, ""},
		{TriageRequest{Verdict: VerdictNotAbcds, By: "Alice", ExistsWithout: "plain git runs it", CannotHelp: "only by switching off the repository's own hooks"}, true, true, "plain git runs it"},
		{TriageRequest{Verdict: VerdictNotAVulnerability, By: "Alice", Reason: "the behaviour is documented"}, true, true, "the behaviour is documented"},
		{TriageRequest{Verdict: VerdictPublicTrack, By: "Alice", Reason: "it discloses nothing"}, true, false, "it discloses nothing"},
	} {
		t.Run(tc.req.Verdict, func(t *testing.T) {
			sd := seedStore(t)
			tc.req.Target = sd.issID
			res, err := sd.triage(tc.req)
			if err != nil {
				t.Fatalf("Triage: %v", err)
			}
			a := sd.state(t)
			if a.Verdict == nil || *a.Verdict != tc.req.Verdict || a.DecidedBy == nil || *a.DecidedBy != "Alice" ||
				a.DecidedAt == nil || !a.DecidedAt.Equal(fixedNow) {
				t.Fatalf("state entry %+v", a)
			}
			if tc.reasonHas != "" && (a.Reason == nil || !strings.Contains(*a.Reason, tc.reasonHas)) {
				t.Fatalf("reason %v does not carry %q", a.Reason, tc.reasonHas)
			}
			if sd.recordIn("wontfix") != tc.wontfix || sd.recordIn("open") == tc.wontfix {
				t.Fatalf("record in wontfix/ = %v, want %v", sd.recordIn("wontfix"), tc.wontfix)
			}
			if tc.wontfix {
				m, _ := filepath.Glob(filepath.Join(sd.s.Root, ".abcd", "work", "issues", "wontfix", sd.issID+"-*.md"))
				if fi, err := os.Stat(m[0]); err != nil || fi.Mode().Perm() != 0o600 {
					t.Fatalf("the moved private record's mode is %v (%v), want 0600", fi.Mode().Perm(), err)
				}
			}
			_, statErr := os.Stat(sd.draftPath())
			if (statErr == nil) != tc.draft || (res.Draft != "") != tc.draft {
				t.Fatalf("draft on disk %v, in result %q; want %v", statErr == nil, res.Draft, tc.draft)
			}
			if tc.draft {
				data, _ := os.ReadFile(sd.draftPath())
				if string(data) != res.Draft {
					t.Fatal("the printed draft is not the one on disk")
				}
				if fi, _ := os.Stat(sd.draftPath()); fi.Mode().Perm() != 0o600 {
					t.Fatalf("draft written %v", fi.Mode().Perm())
				}
			}
			if res.GHSAID != sd.ghsa || res.IssueID != sd.issID {
				t.Fatalf("result names %s / %s", res.GHSAID, res.IssueID)
			}
		})
	}
}

// TestTriageByAdvisoryID: the advisory id addresses the same entry as the
// record's id.
func TestTriageByAdvisoryID(t *testing.T) {
	sd := seedStore(t)
	if _, err := sd.triage(TriageRequest{Target: sd.ghsa, Verdict: VerdictVulnerability, By: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if a := sd.state(t); a.Verdict == nil {
		t.Fatal("no verdict recorded")
	}
}

// TestTriageRefusesUnknownVerdictWritingNothing: a verdict outside the four is
// refused before the store is touched.
func TestTriageRefusesUnknownVerdictWritingNothing(t *testing.T) {
	sd := seedStore(t)
	before := sd.snapshot(t)
	_, err := sd.triage(TriageRequest{Target: sd.issID, Verdict: "not-ours", By: "Alice", Reason: "x"})
	mustRefuse(t, err, "verdict")
	if after := sd.snapshot(t); after != before {
		t.Fatalf("a refused triage wrote:\nbefore\n%s\nafter\n%s", before, after)
	}
}

// TestTriageRefusesMissingBy: who decided is required, and so is the reason
// of a verdict that has one; either missing writes nothing.
func TestTriageRefusesMissingBy(t *testing.T) {
	for name, req := range map[string]TriageRequest{
		"no by":                       {Verdict: VerdictVulnerability, By: "  "},
		"not-a-vulnerability, reason": {Verdict: VerdictNotAVulnerability, By: "Alice"},
		"public-track, reason":        {Verdict: VerdictPublicTrack, By: "Alice", Reason: " "},
	} {
		t.Run(name, func(t *testing.T) {
			sd := seedStore(t)
			before := sd.snapshot(t)
			req.Target = sd.issID
			_, err := sd.triage(req)
			var r *RefusalError
			if !errors.As(err, &r) {
				t.Fatalf("err = %v, want a refusal", err)
			}
			if after := sd.snapshot(t); after != before {
				t.Fatal("a refused triage wrote")
			}
		})
	}
}

// TestNotAbcdsNeedsBothHalves: "not abcd's" is the closing route of
// abcd-fixes-what-it-adds, so it takes both halves of that rule's reason.
func TestNotAbcdsNeedsBothHalves(t *testing.T) {
	for name, req := range map[string]TriageRequest{
		"no exists-without": {Verdict: VerdictNotAbcds, By: "Alice", CannotHelp: "only by changing its hooks"},
		"no cannot-help":    {Verdict: VerdictNotAbcds, By: "Alice", ExistsWithout: "plain git runs it"},
	} {
		t.Run(name, func(t *testing.T) {
			sd := seedStore(t)
			before := sd.snapshot(t)
			req.Target = sd.issID
			_, err := sd.triage(req)
			mustRefuse(t, err, "abcd")
			if after := sd.snapshot(t); after != before {
				t.Fatal("a refused triage wrote")
			}
		})
	}
}

// TestClosingDraftNamesBothHalves: the "not abcd's" draft says why the problem
// would exist without abcd and why abcd cannot help without changing how the
// repository works, in the person's words; neither draft names the advisory.
func TestClosingDraftNamesBothHalves(t *testing.T) {
	sd := seedStore(t)
	res, err := sd.triage(TriageRequest{Target: sd.issID, Verdict: VerdictNotAbcds, By: "Alice",
		ExistsWithout: "a plain git commit runs the configured filter", CannotHelp: "only by switching the filter off for the repository"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"would exist without abcd", "a plain git commit runs the configured filter",
		"without changing how the repository works", "only by switching the filter off for the repository",
	} {
		if !strings.Contains(res.Draft, want) {
			t.Errorf("the draft does not say %q:\n%s", want, res.Draft)
		}
	}
	if strings.Contains(res.Draft, sd.ghsa) || strings.Contains(res.Draft, "Alice") {
		t.Errorf("the draft names the advisory or the decider:\n%s", res.Draft)
	}

	sd2 := seedStore(t)
	res2, err := sd2.triage(TriageRequest{Target: sd2.issID, Verdict: VerdictNotAVulnerability, By: "Alice", Reason: "the behaviour is documented and intended"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res2.Draft, "not a security vulnerability") || !strings.Contains(res2.Draft, "the behaviour is documented and intended") {
		t.Errorf("the not-a-vulnerability draft:\n%s", res2.Draft)
	}
}

// TestRetriageAppendsHistory: a second verdict replaces the first in place and
// keeps it, with who decided it and when, in the entry's history.
func TestRetriageAppendsHistory(t *testing.T) {
	sd := seedStore(t)
	if _, err := sd.triage(TriageRequest{Target: sd.issID, Verdict: VerdictVulnerability, By: "Alice", Reason: "first look"}); err != nil {
		t.Fatal(err)
	}
	later := fixedNow.Add(time.Hour)
	if _, err := sd.triage(TriageRequest{Target: sd.issID, Verdict: VerdictNotAVulnerability, By: "Bob", Reason: "on a closer look, documented",
		Now: func() time.Time { return later }}); err != nil {
		t.Fatal(err)
	}
	a := sd.state(t)
	if *a.Verdict != VerdictNotAVulnerability || *a.DecidedBy != "Bob" || !a.DecidedAt.Equal(later) {
		t.Fatalf("current verdict %+v", a)
	}
	if len(a.History) != 1 {
		t.Fatalf("history holds %d entries, want 1", len(a.History))
	}
	h := a.History[0]
	if h.Verdict != VerdictVulnerability || h.DecidedBy != "Alice" || !h.DecidedAt.Equal(fixedNow) || h.Reason != "first look" {
		t.Fatalf("history entry %+v", h)
	}
	if !sd.recordIn("wontfix") {
		t.Fatal("the record did not move to wontfix/")
	}

	// A third verdict between the closing ones keeps the record where it is
	// and redrafts the text.
	res, err := sd.triage(TriageRequest{Target: sd.issID, Verdict: VerdictNotAbcds, By: "Bob",
		ExistsWithout: "plain git runs it", CannotHelp: "only by changing its hooks"})
	if err != nil {
		t.Fatal(err)
	}
	if a := sd.state(t); len(a.History) != 2 {
		t.Fatalf("history holds %d entries, want 2", len(a.History))
	}
	data, _ := os.ReadFile(sd.draftPath())
	if !bytes.Equal(data, []byte(res.Draft)) || !strings.Contains(res.Draft, "plain git runs it") {
		t.Fatal("the draft was not redrafted")
	}
}

// TestRetriageRefusesReopeningAClosedRecord: a record triage moved to wontfix/
// cannot be brought back by a later "vulnerability" verdict, since the ledger
// has no reopen; the refusal writes nothing.
func TestRetriageRefusesReopeningAClosedRecord(t *testing.T) {
	sd := seedStore(t)
	if _, err := sd.triage(TriageRequest{Target: sd.issID, Verdict: VerdictNotAVulnerability, By: "Alice", Reason: "documented"}); err != nil {
		t.Fatal(err)
	}
	before := sd.snapshot(t)
	_, err := sd.triage(TriageRequest{Target: sd.issID, Verdict: VerdictVulnerability, By: "Alice"})
	mustRefuse(t, err, "wontfix")
	if after := sd.snapshot(t); after != before {
		t.Fatal("a refused triage wrote")
	}
}

// TestTriageRefusesClosedOrPublishedAdvisory: an advisory already closed or
// published on GitHub takes no new verdict.
func TestTriageRefusesClosedOrPublishedAdvisory(t *testing.T) {
	sd := seedStore(t)
	st, _ := sd.s.Verify()
	at := fixedNow
	st.Advisories[0].ClosedAt = &at
	if err := sd.s.SaveState(st); err != nil {
		t.Fatal(err)
	}
	_, err := sd.triage(TriageRequest{Target: sd.issID, Verdict: VerdictVulnerability, By: "Alice"})
	mustRefuse(t, err, "closed")
}

// TestTriageRefusesWithoutStore: triage makes no store; with none it says to
// run intake.
func TestTriageRefusesWithoutStore(t *testing.T) {
	f := newFixture(t)
	_, err := Triage(TriageRequest{Home: f.home, RepoRoot: f.repo.Root(), Target: "iss-2610120000000001", Verdict: VerdictVulnerability, By: "Alice"})
	mustRefuse(t, err, "intake")
	p, _ := storePath(f.home, f.repo.Root())
	if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("triage made a store")
	}
}

// TestPublicTrackPrintsCaptureCommand: moving an advisory to the public track
// prints the capture command for the person to run in the public checkout,
// carrying the record's severity and none of the advisory's id or title.
func TestPublicTrackPrintsCaptureCommand(t *testing.T) {
	sd := seedStore(t)
	res, err := sd.triage(TriageRequest{Target: sd.issID, Verdict: VerdictPublicTrack, By: "Alice", Reason: "it discloses nothing"})
	if err != nil {
		t.Fatal(err)
	}
	cmd := res.CaptureCommand
	for _, want := range []string{"abcd capture ", "--severity major", "--category security", "--remedy "} {
		if !strings.Contains(cmd, want) {
			t.Errorf("the command %q lacks %q", cmd, want)
		}
	}
	if strings.Contains(cmd, sd.ghsa) || strings.Contains(cmd, seededTitle) || strings.Contains(cmd, "security-advisory") {
		t.Errorf("the command carries the advisory: %q", cmd)
	}
	a := sd.state(t)
	if a.Reason == nil || !strings.Contains(*a.Reason, "moved to the public track") {
		t.Fatalf("reason %v", a.Reason)
	}
}
