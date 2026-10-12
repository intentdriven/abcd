package embargo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The check fixtures name advisories of the GHSA-xxxx-xxxx-xxxx placeholder
// form and invented titles and CVE ids (examples-use-reserved-identifiers).
const (
	ghsaOpen      = "GHSA-xxxx-xxxx-xxxx" // unpublished, verdict vulnerability, CVE reserved
	ghsaClosed    = "GHSA-wwww-wwww-wwww" // closed on GitHub, never published
	ghsaPublished = "GHSA-vvvv-vvvv-vvvv" // published
	ghsaPublic    = "GHSA-rrrr-rrrr-rrrr" // triaged public-track, unpublished
	ghsaUnlisted  = "GHSA-qqqq-qqqq-qqqq" // no advisory the store holds

	cveOpen       = "CVE-2026-0000"
	cvePublished  = "CVE-2026-0009"
	titleOpen     = "Sample Overflow In The Example Parser"
	titleClosed   = "a sample closed report"
	titlePublic   = "a sample public-track report"
	titlePublishd = "a sample published report"
)

// checkFixture is a store holding one advisory of each kind the deny set
// treats differently.
type checkFixture struct {
	f fixture
	s Store
}

func newCheckFixture(t *testing.T) checkFixture {
	t.Helper()
	f := newFixture(t)
	s, _, err := Open(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	at := fixedNow
	st := emptyState()
	st.Repo = "example/example"
	st.IntakeAt = &at
	st.Advisories = []Advisory{
		{GHSAID: ghsaOpen, IssueID: "iss-2610120000000001", Title: titleOpen, CVEID: strp(cveOpen),
			Severity: "major", ForgeState: "draft", Verdict: strp(VerdictVulnerability), DecidedBy: strp("Alice"), DecidedAt: &at},
		{GHSAID: ghsaClosed, IssueID: "iss-2610120000000002", Title: titleClosed, Severity: "minor", ForgeState: "closed",
			Verdict: strp(VerdictNotAVulnerability), DecidedBy: strp("Alice"), DecidedAt: &at, Reason: strp("documented"), ClosedAt: &at},
		{GHSAID: ghsaPublished, IssueID: "iss-2610120000000003", Title: titlePublishd, CVEID: strp(cvePublished),
			Severity: "major", ForgeState: "published", Verdict: strp(VerdictVulnerability), DecidedBy: strp("Alice"), DecidedAt: &at, PublishedAt: &at},
		{GHSAID: ghsaPublic, IssueID: "iss-2610120000000004", Title: titlePublic, Severity: "minor", ForgeState: "triage",
			Verdict: strp(VerdictPublicTrack), DecidedBy: strp("Alice"), DecidedAt: &at, Reason: strp("no embargo needed")},
	}
	if err := s.SaveState(st); err != nil {
		t.Fatal(err)
	}
	return checkFixture{f: f, s: s}
}

// text checks one stdin text.
func (cf checkFixture) text(t *testing.T, body string, allow ...string) CheckResult {
	t.Helper()
	res, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: cf.f.repo.Root(), Files: []string{"-"},
		Stdin: strings.NewReader(body), Allow: allow, Now: func() time.Time { return fixedNow.Add(time.Hour) }})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return res
}

func kinds(res CheckResult) []string {
	var out []string
	for _, f := range res.Findings {
		out = append(out, f.Kind)
	}
	return out
}

func wantKinds(t *testing.T, res CheckResult, want ...string) {
	t.Helper()
	got := kinds(res)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("finding kinds %q, want %q (%+v)", got, want, res.Findings)
	}
	if res.Clean() != (len(want) == 0) {
		t.Fatalf("Clean() = %v with %d finding(s)", res.Clean(), len(want))
	}
}

// TestCheckRefusesUnpublishedGHSA: an unpublished advisory's id is refused in
// any case, and the finding says where it is.
func TestCheckRefusesUnpublishedGHSA(t *testing.T) {
	cf := newCheckFixture(t)
	for _, body := range []string{
		"fixes " + ghsaOpen + "\n",
		"line one\nsee " + strings.ToLower(ghsaOpen) + ".\n",
	} {
		res := cf.text(t, body)
		wantKinds(t, res, KindAdvisoryID)
		f := res.Findings[0]
		if f.Source != "stdin" || f.Line != strings.Count(body[:strings.Index(strings.ToLower(body), "ghsa")], "\n")+1 {
			t.Fatalf("finding %+v", f)
		}
	}
}

// TestCheckRefusesClosedUnpublishedGHSA: a closed advisory stays private on
// GitHub, so its id stays refused.
func TestCheckRefusesClosedUnpublishedGHSA(t *testing.T) {
	cf := newCheckFixture(t)
	wantKinds(t, cf.text(t, "closes "+ghsaClosed+"\n"), KindAdvisoryID)
	wantKinds(t, cf.text(t, "the report about "+titleClosed+"\n"), KindAdvisoryTitle)
}

// TestCheckPassesPublishedGHSA: a published advisory's id, CVE id and title
// are public, and pass without --allow.
func TestCheckPassesPublishedGHSA(t *testing.T) {
	cf := newCheckFixture(t)
	res := cf.text(t, "fixed in "+ghsaPublished+" ("+cvePublished+"): "+titlePublishd+"\n")
	wantKinds(t, res)
}

// TestCheckRefusesReservedCVE: a CVE id reserved for an unpublished advisory
// is refused; a longer id that merely starts with it is another id.
func TestCheckRefusesReservedCVE(t *testing.T) {
	cf := newCheckFixture(t)
	wantKinds(t, cf.text(t, "tracked as "+strings.ToLower(cveOpen)+"\n"), KindReservedCVE)
	wantKinds(t, cf.text(t, "unrelated "+cveOpen+"1\n"))
}

// TestCheckRefusesTitleCaseInsensitive: an unpublished advisory's title is
// refused in any case and however its whitespace is wrapped; a public-track
// advisory's title is not, since that issue is fixed in the open.
func TestCheckRefusesTitleCaseInsensitive(t *testing.T) {
	cf := newCheckFixture(t)
	wantKinds(t, cf.text(t, "fix: "+strings.ToUpper(titleOpen)+"\n"), KindAdvisoryTitle)
	wrapped := "fix: sample   overflow in\n  the example\tparser\n"
	res := cf.text(t, wrapped)
	wantKinds(t, res, KindAdvisoryTitle)
	if res.Findings[0].Line != 1 {
		t.Fatalf("wrapped title found on line %d, want 1", res.Findings[0].Line)
	}
	wantKinds(t, cf.text(t, "fix: "+titlePublic+"\n"))
	// The public-track advisory's id is still unpublished.
	wantKinds(t, cf.text(t, "see "+ghsaPublic+"\n"), KindAdvisoryID)
	// A title glued into a longer word is not the title.
	wantKinds(t, cf.text(t, "Xsample overflow in the example parsers\n"))
}

// TestCheckRefusesStorePathForms: the store's location is refused absolute,
// home-relative, as its worktree-store fragment, and as the local-tier folder
// the state sits in.
func TestCheckRefusesStorePathForms(t *testing.T) {
	cf := newCheckFixture(t)
	sha := cf.f.repo.Git("rev-list", "--max-parents=0", "HEAD")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(home, cf.s.Root)
	if err != nil || strings.HasPrefix(rel, "..") {
		t.Fatalf("the fixture store %s is not under HOME %s", cf.s.Root, home)
	}
	for _, body := range []string{
		"cd " + cf.s.Root + "/x\n",
		"cd ~/" + filepath.ToSlash(rel) + "\n",
		"under worktrees/" + sha + "/embargo/.abcd\n",
		"see .abcd/.work.local/embargo/state.json\n",
	} {
		wantKinds(t, cf.text(t, body), KindStorePath)
	}
}

// TestCheckRefusesUnknownGHSAUnlessAllowed: an advisory id the store does not
// hold is refused unless --allow names it, and every allowed id is listed; an
// unpublished advisory cannot be allowed.
func TestCheckRefusesUnknownGHSAUnlessAllowed(t *testing.T) {
	cf := newCheckFixture(t)
	body := "upstream " + ghsaUnlisted + "\n"
	wantKinds(t, cf.text(t, body), KindUnlistedAdvisoryID)
	res := cf.text(t, body, strings.ToLower(ghsaUnlisted))
	wantKinds(t, res)
	if len(res.Allowed) != 1 || res.Allowed[0] != ghsaUnlisted {
		t.Fatalf("allowed %q", res.Allowed)
	}
	_, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: cf.f.repo.Root(), Files: []string{"-"},
		Stdin: strings.NewReader(body), Allow: []string{ghsaOpen}})
	mustRefuse(t, err, "unpublished")
	_, err = Check(CheckRequest{Home: cf.f.home, RepoRoot: cf.f.repo.Root(), Files: []string{"-"},
		Stdin: strings.NewReader(body), Allow: []string{"not-an-id"}})
	mustRefuse(t, err, "--allow")
}

// TestCheckReadsRangeMessagesAndAddedLines: over a commit range the check
// reads each commit's message and the lines it adds, naming the commit, the
// file and the line; a line a commit removes is not read.
func TestCheckReadsRangeMessagesAndAddedLines(t *testing.T) {
	cf := newCheckFixture(t)
	r := cf.f.repo
	r.Write("notes.txt", "one\nmentions "+ghsaOpen+"\nthree\n")
	r.Commit("add notes")
	base := r.Git("rev-parse", "HEAD")
	r.Write("notes.txt", "one\nthree\n")
	r.Commit("drop a line")
	r.Write("notes.txt", "one\nthree\nfour\ntracked as "+cveOpen+"\n")
	r.Commit("add more")
	added := r.Git("rev-parse", "HEAD")
	r.Write("other.txt", "clean\n")
	r.Commit("subject line\n\nbody names " + titleOpen)
	msgCommit := r.Git("rev-parse", "HEAD")

	res, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: r.Root(), Range: base + "..HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	wantKinds(t, res, KindReservedCVE, KindAdvisoryTitle)
	f := res.Findings[0]
	if f.Source != "commit "+added || f.Where != "added line in notes.txt" || f.Line != 4 {
		t.Fatalf("added-line finding %+v", f)
	}
	f = res.Findings[1]
	if f.Source != "commit "+msgCommit || f.Where != "message" || f.Line != 3 {
		t.Fatalf("message finding %+v", f)
	}
	if res.Commits != 3 {
		t.Fatalf("read %d commits, want 3", res.Commits)
	}
}

// TestCheckRunsNoRepoDiffDriver: a textconv or external diff the repository
// configures never runs, and the added line is read as git stores it.
func TestCheckRunsNoRepoDiffDriver(t *testing.T) {
	cf := newCheckFixture(t)
	r := cf.f.repo
	marker := filepath.Join(t.TempDir(), "ran")
	driver := filepath.Join(t.TempDir(), "driver.sh")
	if err := os.WriteFile(driver, []byte("#!/bin/sh\ntouch '"+marker+"'\necho clean\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("config", "diff.evil.textconv", driver)
	r.Git("config", "diff.evil.command", driver)
	r.Git("config", "diff.external", driver)
	r.Write(".gitattributes", "*.txt diff=evil\n")
	r.Commit("attributes")
	base := r.Git("rev-parse", "HEAD")
	r.Write("leak.txt", "names "+ghsaOpen+"\n")
	r.Commit("add a file")

	res, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: r.Root(), Range: base + "..HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("a repository-configured diff driver ran")
	}
	wantKinds(t, res, KindAdvisoryID)
}

// TestCheckMasksSpan: no part of a finding, in its fields or its JSON,
// repeats what was matched; the rest of the line is kept to find it by.
func TestCheckMasksSpan(t *testing.T) {
	cf := newCheckFixture(t)
	res := cf.text(t, "see "+ghsaOpen+" and "+titleOpen+" and "+cveOpen+" here\n")
	if len(res.Findings) != 3 {
		t.Fatalf("findings %+v", res.Findings)
	}
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	low := strings.ToLower(string(data))
	for _, s := range []string{ghsaOpen, titleOpen, cveOpen, "overflow", "xxxx", "0000"} {
		if strings.Contains(low, strings.ToLower(s)) {
			t.Fatalf("the result repeats %q: %s", s, data)
		}
	}
	if sn := res.Findings[0].Snippet; !strings.HasPrefix(sn, "see ***") || !strings.HasSuffix(sn, " here") {
		t.Fatalf("snippet %q", sn)
	}
}

// TestCheckNoStorePasses: with no store on this machine nothing is under
// embargo, and the check passes saying so.
func TestCheckNoStorePasses(t *testing.T) {
	f := newFixture(t)
	res, err := Check(CheckRequest{Home: f.home, RepoRoot: f.repo.Root(), Files: []string{"-"},
		Stdin: strings.NewReader("fixes " + ghsaOpen + "\n")})
	if err != nil {
		t.Fatal(err)
	}
	if !res.NoStore || !res.Clean() || res.Note == "" {
		t.Fatalf("result %+v", res)
	}
}

// TestCheckClean: text holding none of the deny set passes, and the intake's
// age is reported, stale past a day.
func TestCheckClean(t *testing.T) {
	cf := newCheckFixture(t)
	res := cf.text(t, "fix: tighten the example parser\n")
	wantKinds(t, res)
	if res.UnderEmbargo != 3 || res.IntakeAt == nil || res.IntakeStale {
		t.Fatalf("result %+v", res)
	}
	res, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: cf.f.repo.Root(), Files: []string{"-"},
		Stdin: strings.NewReader("clean\n"), Now: func() time.Time { return fixedNow.Add(25 * time.Hour) }})
	if err != nil || !res.IntakeStale {
		t.Fatalf("a day-old intake is not stale: %+v %v", res, err)
	}
}

// TestCheckFileInputs: a named file is read, a missing one refuses, and a
// file name that itself carries the deny set is not repeated.
func TestCheckFileInputs(t *testing.T) {
	cf := newCheckFixture(t)
	dir := t.TempDir()
	body := filepath.Join(dir, "pr-body.md")
	if err := os.WriteFile(body, []byte("clean\n\nfixes "+ghsaOpen+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: cf.f.repo.Root(), Files: []string{body}})
	if err != nil {
		t.Fatal(err)
	}
	wantKinds(t, res, KindAdvisoryID)
	if res.Findings[0].Source != "file "+body || res.Findings[0].Line != 3 {
		t.Fatalf("finding %+v", res.Findings[0])
	}
	_, err = Check(CheckRequest{Home: cf.f.home, RepoRoot: cf.f.repo.Root(), Files: []string{filepath.Join(dir, "absent")}})
	mustRefuse(t, err, "could not be read")
	named := filepath.Join(dir, ghsaOpen+".md")
	if err := os.WriteFile(named, []byte("names "+ghsaOpen+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err = Check(CheckRequest{Home: cf.f.home, RepoRoot: cf.f.repo.Root(), Files: []string{named}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Findings) != 1 || strings.Contains(res.Findings[0].Source, "xxxx") {
		t.Fatalf("findings %+v", res.Findings)
	}
	_, err = Check(CheckRequest{Home: cf.f.home, RepoRoot: cf.f.repo.Root()})
	mustRefuse(t, err, "nothing to check")
}
