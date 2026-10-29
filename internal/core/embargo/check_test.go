package embargo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The check fixtures name advisories of the GHSA-xxxx-xxxx-xxxx placeholder
// form and invented titles and CVE ids (examples-use-reserved-identifiers).
const (
	ghsaOpen      = "GHSA-xxxx-xxxx-xxxx" // unpublished, verdict vulnerability, CVE reserved
	ghsaClosed    = "GHSA-wwww-wwww-wwww" // closed on GitHub, never published
	ghsaPublished = "GHSA-vvvv-vvvv-vvvv" // published
	ghsaPublic    = "GHSA-rrrr-rrrr-rrrr" // triaged public-track, unpublished
	ghsaUnlisted  = "GHSA-qqqq-qqqq-qqqq" // no advisory the store holds, placeholder form

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
	s, _, err := openStore(f.home, f.repo.Root())
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
// GitHub, so its id stays refused. Its title does not: the reporter chose it,
// and a closed report's title held forever ("Merge pull request") would
// refuse every range after it. An advisory not yet closed keeps its title.
func TestCheckRefusesClosedUnpublishedGHSA(t *testing.T) {
	cf := newCheckFixture(t)
	wantKinds(t, cf.text(t, "closes "+ghsaClosed+"\n"), KindAdvisoryID)
	wantKinds(t, cf.text(t, "the report about "+titleClosed+"\n"))
	wantKinds(t, cf.text(t, "the report about "+titleOpen+"\n"), KindAdvisoryTitle)
}

// TestCheckKeepsClosedAdvisorysCVE: closing an advisory drops its title from
// the deny set, never its reserved CVE id.
func TestCheckKeepsClosedAdvisorysCVE(t *testing.T) {
	cf := newCheckFixture(t)
	st, err := cf.s.Verify()
	if err != nil {
		t.Fatal(err)
	}
	st.Advisories[1].CVEID = strp("CVE-2026-0005")
	st.Advisories[1].Title = "Merge pull request"
	if err := cf.s.SaveState(st); err != nil {
		t.Fatal(err)
	}
	wantKinds(t, cf.text(t, "tracked as CVE-2026-0005\n"), KindReservedCVE)
	wantKinds(t, cf.text(t, "Merge pull request #12 from example/branch\n"))
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

// TestCheckRefusesStorePathForms: the store's location on this machine is
// refused absolute, home-relative and as its worktree-store fragment.
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
		"the store is " + cf.s.Root + ".\n",
	} {
		wantKinds(t, cf.text(t, body), KindStorePath)
	}
}

// TestCheckPassesStoreRelativeConstants: the relative paths abcd's own source
// names (the state file and the drafts inside any store) say nothing about
// where this machine keeps its records, so a range that touches the embargo
// package is not refused for them; and a sibling folder whose name merely
// begins with the store's is another place.
func TestCheckPassesStoreRelativeConstants(t *testing.T) {
	cf := newCheckFixture(t)
	wantKinds(t, cf.text(t, "const StateRel = \""+StateRel+"\"\n"))
	wantKinds(t, cf.text(t, "drafts sit in "+DraftDirRel+"/\n"))
	wantKinds(t, cf.text(t, "cd "+cf.s.Root+"-steps-10-11/x\n"))
	wantKinds(t, cf.text(t, "cd "+cf.s.Root+".bak/x\n"))

	r := cf.f.repo
	base := r.Git("rev-parse", "HEAD")
	r.Write("internal/core/embargo/state.go", "package embargo\n\nconst StateRel = \""+StateRel+"\"\n")
	r.Commit("feat: name the state file")
	res, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: r.Root(), Range: base + "..HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	wantKinds(t, res)
}

// TestCheckShortTitleNotMatchedAlone: a title shorter than the minimum is
// not matched on its own, so a very short title cannot match everywhere; the
// advisory's id still is.
func TestCheckShortTitleNotMatchedAlone(t *testing.T) {
	cf := newCheckFixture(t)
	st, err := cf.s.Verify()
	if err != nil {
		t.Fatal(err)
	}
	short := "Parser bug"  // 10 characters: not matched alone
	edge := "Parser crash" // 12 characters: matched
	st.Advisories = append(st.Advisories,
		Advisory{GHSAID: "GHSA-mmmm-mmmm-mmmm", IssueID: "iss-2610120000000005", Title: short, Severity: "minor", ForgeState: "triage"},
		Advisory{GHSAID: "GHSA-jjjj-jjjj-jjjj", IssueID: "iss-2610120000000006", Title: edge, Severity: "minor", ForgeState: "triage"})
	if err := cf.s.SaveState(st); err != nil {
		t.Fatal(err)
	}
	wantKinds(t, cf.text(t, "fix: a parser bug in the example\n"))
	wantKinds(t, cf.text(t, "see GHSA-mmmm-mmmm-mmmm\n"), KindAdvisoryID)
	wantKinds(t, cf.text(t, "fix: parser crash on empty input\n"), KindAdvisoryTitle)
}

// TestCheckReadsMergeCommitDiffs: a merge commit's own diff is read against
// its first parent, so a line a merge adds that no merged commit carries (an
// evil merge, a conflict resolution) is not let through.
func TestCheckReadsMergeCommitDiffs(t *testing.T) {
	cf := newCheckFixture(t)
	r := cf.f.repo
	base := r.Git("rev-parse", "HEAD")
	r.Git("switch", "-q", "-c", "side")
	r.Write("side.txt", "clean\n")
	r.Commit("side work")
	r.Git("switch", "-q", "main")
	r.Write("main.txt", "clean\n")
	r.Commit("main work")
	r.Git("merge", "-q", "--no-ff", "--no-commit", "side")
	r.Write("merged.txt", "names "+ghsaOpen+"\n")
	r.Git("add", "merged.txt")
	r.Git("commit", "-q", "-m", "merge side")
	merge := r.Git("rev-parse", "HEAD")

	res, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: r.Root(), Range: base + "..HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	wantKinds(t, res, KindAdvisoryID)
	if f := res.Findings[0]; f.Source != "commit "+merge || f.Where != "added line in merged.txt" || f.Line != 1 {
		t.Fatalf("merge finding %+v", f)
	}
}

// ghsaRealShaped is an advisory id of a real one's shape that the store does
// not hold. It is assembled at run time so this file's own text carries no
// such id for the check to refuse over the commit that adds it.
var ghsaRealShaped = "GHSA-" + strings.Join([]string{"2c3f", "4g5h", "6j7m"}, "-")

// TestCheckRefusesUnknownGHSAUnlessAllowed: an advisory id the store does not
// hold is refused unless --allow names it, and every allowed id is listed; an
// unpublished advisory cannot be allowed.
func TestCheckRefusesUnknownGHSAUnlessAllowed(t *testing.T) {
	cf := newCheckFixture(t)
	body := "upstream " + ghsaRealShaped + "\n"
	wantKinds(t, cf.text(t, body), KindUnlistedAdvisoryID)
	res := cf.text(t, body, strings.ToLower(ghsaRealShaped))
	wantKinds(t, res)
	if len(res.Allowed) != 1 || res.Allowed[0] != ghsaRealShaped {
		t.Fatalf("allowed %q", res.Allowed)
	}
	_, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: cf.f.repo.Root(), Files: []string{"-"},
		Stdin: strings.NewReader(body), Allow: []string{ghsaOpen}})
	mustRefuse(t, err, "unpublished")
	_, err = Check(CheckRequest{Home: cf.f.home, RepoRoot: cf.f.repo.Root(), Files: []string{"-"},
		Stdin: strings.NewReader(body), Allow: []string{"not-an-id"}})
	mustRefuse(t, err, "--allow")
}

// TestCheckPassesPlaceholderAsUnlisted: an id of the placeholder form, each
// four-character group one character repeated, is a documentation or test
// placeholder and is not refused as unlisted; an id of a real one's shape
// still is, and a placeholder-form id the store holds unpublished is still
// refused as that advisory's id.
func TestCheckPassesPlaceholderAsUnlisted(t *testing.T) {
	cf := newCheckFixture(t)
	for _, id := range []string{ghsaUnlisted, "ghsa-mmmm-mmmm-mmmm", "GHSA-2222-cccc-xxxx"} {
		wantKinds(t, cf.text(t, "see "+id+"\n"))
	}
	for _, id := range []string{ghsaRealShaped, "GHSA-" + strings.Join([]string{"xxxx", "xxxx", "xxx2"}, "-")} {
		wantKinds(t, cf.text(t, "see "+id+"\n"), KindUnlistedAdvisoryID)
	}
	wantKinds(t, cf.text(t, "fixes "+ghsaOpen+"\n"), KindAdvisoryID)
	wantKinds(t, cf.text(t, "closes "+strings.ToLower(ghsaClosed)+"\n"), KindAdvisoryID)
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

// TestCheckFoldsConfusablesAndLineWraps: an id or title spelled with
// fullwidth letters, a dash other than the hyphen-minus, an invisible
// character inside it, or a hyphenated token broken at a line end is the same
// id or title, and is refused. The snippet repeats it in no spelling: folded
// as the check folds, it holds no folded needle.
func TestCheckFoldsConfusablesAndLineWraps(t *testing.T) {
	cf := newCheckFixture(t)
	for name, tc := range map[string]struct {
		body   string
		kind   string
		needle string
	}{
		"wrapped id":         {"fixes GHSA-xxxx-xxxx-\nxxxx today\n", KindAdvisoryID, ghsaOpen},
		"wrapped id, indent": {"fixes GHSA-xxxx-\n    xxxx-xxxx today\n", KindAdvisoryID, ghsaOpen},
		"wrapped cve":        {"tracked as CVE-2026-\n0000\n", KindReservedCVE, cveOpen},
		"fullwidth id":       {"fixes ＧＨＳＡ－ｘｘｘｘ－ｘｘｘｘ－ｘｘｘｘ\n", KindAdvisoryID, ghsaOpen},
		"unicode hyphens":    {"fixes GHSA‐xxxx‑xxxx–xxxx\n", KindAdvisoryID, ghsaOpen},
		"minus sign":         {"fixes GHSA−xxxx−xxxx−xxxx\n", KindAdvisoryID, ghsaOpen},
		"zero-width inside":  {"fixes GHSA-xx\u200bxx-xxxx-xx\u00adxx\n", KindAdvisoryID, ghsaOpen},
		"fullwidth cve":      {"tracked as ＣＶＥ-２０２６-００００\n", KindReservedCVE, cveOpen},
		"fullwidth title":    {"about the ｓample overflow in the example parser\n", KindAdvisoryTitle, titleOpen},
	} {
		t.Run(name, func(t *testing.T) {
			res := cf.text(t, tc.body)
			wantKinds(t, res, tc.kind)
			snippet, _ := normalise(res.Findings[0].Snippet)
			needle, _ := normalise(tc.needle)
			if strings.Contains(snippet, strings.TrimSpace(needle)) {
				t.Fatalf("the snippet repeats the match: %q", res.Findings[0].Snippet)
			}
			// A wrapped match's snippet is only the line where it starts, so
			// the whole folded needle can never appear there; the masked
			// fragment before the break must still show none of its letters.
			if strings.Contains(snippet, "xxxx") || strings.Contains(snippet, "2026-") {
				t.Fatalf("the snippet repeats a fragment of the match: %q", res.Findings[0].Snippet)
			}
		})
	}
}

// TestCheckLineJoinOnlyJoinsWrappedTokens: a dash standing alone at a line's
// end is not a broken token, and joining nothing else makes no finding.
func TestCheckLineJoinOnlyJoinsWrappedTokens(t *testing.T) {
	cf := newCheckFixture(t)
	wantKinds(t, cf.text(t, "a list -\nsample overflow in the parser\nwell-\nknown\n"))
}

// TestCheckRefusesARecordTheStoreHoldsUnderEmbargo: a record id a caller is
// about to name in public (an announcement's item, itd-2609151838312703 C5)
// is refused when the store holds it for an advisory not yet published,
// closed or public-track ones included, as their advisory ids are; a
// published advisory's record, and a record the store does not hold, pass.
// The finding names the record by its place among those asked, never by the
// id. A record that is not a record id refuses the check, and records alone
// are something to check.
func TestCheckRefusesARecordTheStoreHoldsUnderEmbargo(t *testing.T) {
	cf := newCheckFixture(t)
	records := func(ids ...string) (CheckResult, error) {
		return Check(CheckRequest{Home: cf.f.home, RepoRoot: cf.f.repo.Root(), Records: ids,
			Now: func() time.Time { return fixedNow.Add(time.Hour) }})
	}
	for _, id := range []string{"iss-2610120000000001", "iss-2610120000000002", "iss-2610120000000004"} {
		res, err := records("itd-7", id)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		wantKinds(t, res, KindEmbargoedRecord)
		if f := res.Findings[0]; f.Source != "record 2" || strings.Contains(f.Source+f.Where+f.Snippet, id) {
			t.Fatalf("%s: finding %+v; want it named as the second record, without the id", id, f)
		}
	}
	for _, id := range []string{"iss-2610120000000003", "iss-5", "itd-2610120000000001"} {
		res, err := records(id)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		wantKinds(t, res)
	}
	_, err := records("not-a-record")
	mustRefuse(t, err, "record id")

	f := newFixture(t)
	res, err := Check(CheckRequest{Home: f.home, RepoRoot: f.repo.Root(), Records: []string{"iss-2610120000000001"}})
	if err != nil || !res.NoStore || !res.Clean() {
		t.Fatalf("with no store: %+v, %v; want a clean check saying there is none", res, err)
	}
}

// TestCheckFindsTheStoreWithHEADOnAnOrphanBranch: the store is keyed on the
// root commit of origin's default branch, so a checkout whose HEAD is on an
// orphan branch, or whose history a replace ref cuts short, still finds
// it, and a record it holds is refused rather than read as nothing under
// embargo.
func TestCheckFindsTheStoreWithHEADOnAnOrphanBranch(t *testing.T) {
	for _, name := range []string{"HEAD on an orphan branch", "a replace ref that cuts the history short"} {
		t.Run(name, func(t *testing.T) {
			cf := newCheckFixture(t)
			r := cf.f.repo
			r.Write("NEXT.md", "more\n")
			r.Commit("second")
			r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
			r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
			if name == "HEAD on an orphan branch" {
				r.Git("checkout", "-q", "--orphan", "elsewhere")
				r.Commit("an unrelated root")
			} else {
				// The tip is stood in for by a commit with no parent, so read
				// through the replace ref the tip itself is the root.
				r.Git("replace", r.Git("rev-parse", "HEAD"), r.Git("commit-tree", r.Git("rev-parse", "HEAD^{tree}"), "-m", "a stand-in root"))
			}
			res, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: r.Root(), Records: []string{"iss-2610120000000001"},
				Now: func() time.Time { return fixedNow.Add(time.Hour) }})
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			if res.NoStore {
				t.Fatalf("the store was not found: %+v", res)
			}
			wantKinds(t, res, KindEmbargoedRecord)
		})
	}
}

// TestTheStoreOfAnOrdinaryCloneStaysWhereItWas: in a clone whose HEAD shares
// origin's default branch's root, the store sits where it always has,
// worktrees/<root commit>/embargo in abcd's home.
func TestTheStoreOfAnOrdinaryCloneStaysWhereItWas(t *testing.T) {
	f := newFixture(t)
	r := f.repo
	r.Git("update-ref", "refs/remotes/origin/main", "HEAD")
	r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	root := r.Git("rev-list", "--max-parents=0", "HEAD")
	loc, err := locate(f.home, r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(abcdhome.Path(f.home, "worktrees", root, StoreName)); loc.Path != want {
		t.Fatalf("store path %s; want %s", loc.Path, want)
	}
}

// TestAddEmbargoedTakesANamedMode: the set addEmbargoed builds is stated by a
// named mode, never a bare bool, so a caller cannot pass true or false and
// silently build the other set (a closed advisory's title leaked into a
// publication text, or every range refused on a closed report's title).
func TestAddEmbargoedTakesANamedMode(t *testing.T) {
	fn := reflect.TypeOf((*denySet).addEmbargoed)
	if fn.NumIn() != 4 {
		t.Fatalf("addEmbargoed takes %d arguments with its receiver, want 4", fn.NumIn())
	}
	mode := fn.In(3)
	if mode.Kind() == reflect.Bool || mode.Name() == "" || mode.Name() == "bool" {
		t.Fatalf("addEmbargoed's mode is %s, want a named mode type the compiler holds apart from bool", mode)
	}
	if mode != reflect.TypeOf(denyForRange) {
		t.Fatalf("addEmbargoed's mode is %s, want denyMode", mode)
	}
}

// TestAddEmbargoedModes: each mode builds its own set, and they differ only
// in a closed advisory's title. A mode neither constant names (the zero
// denyMode) keeps the title: it holds more, never less.
func TestAddEmbargoedModes(t *testing.T) {
	for _, tc := range []struct {
		name       string
		mode       denyMode
		closedHeld bool
	}{
		{"range", denyForRange, false},
		{"publication", denyForPublication, true},
		{"unnamed", denyMode(0), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ds := denySet{}
			ds.addEmbargoed(closedAdvisoryState(), "", tc.mode)
			if got := hasNeedle(ds, KindAdvisoryTitle, titleClosed); got != tc.closedHeld {
				t.Fatalf("closed advisory's title held = %v, want %v: %+v", got, tc.closedHeld, ds.needles)
			}
			for _, want := range []struct{ kind, text string }{
				{KindAdvisoryTitle, titleOpen}, {KindAdvisoryID, ghsaOpen}, {KindAdvisoryID, ghsaClosed}, {KindReservedCVE, cveOpen},
			} {
				if !hasNeedle(ds, want.kind, want.text) {
					t.Fatalf("the %s set lacks %s %q: %+v", tc.name, want.kind, want.text, ds.needles)
				}
			}
			if ds.embargo != 2 {
				t.Fatalf("embargo count %d, want 2", ds.embargo)
			}
		})
	}
}

// closedAdvisoryState holds one closed, unpublished advisory with a reserved
// CVE id, and one still open, both titled long enough to be held.
func closedAdvisoryState() State {
	at := fixedNow
	st := emptyState()
	st.Advisories = []Advisory{
		{GHSAID: ghsaOpen, IssueID: "iss-2610120000000001", Title: titleOpen, Severity: "major", ForgeState: "draft",
			Verdict: strp(VerdictVulnerability), DecidedBy: strp("Alice"), DecidedAt: &at},
		{GHSAID: ghsaClosed, IssueID: "iss-2610120000000002", Title: titleClosed, CVEID: strp(cveOpen), Severity: "minor",
			ForgeState: "closed", Verdict: strp(VerdictNotAVulnerability), DecidedBy: strp("Alice"), DecidedAt: &at,
			Reason: strp("documented"), ClosedAt: &at},
	}
	return st
}

// hasNeedle reports whether ds holds text as a needle of kind.
func hasNeedle(ds denySet, kind, text string) bool {
	n, _ := normalise(text)
	n = strings.TrimSpace(n)
	for _, have := range ds.needles {
		if have.kind == kind && have.text == n {
			return true
		}
	}
	return false
}

// TestRangeDenySetDropsClosedTitles: the range and file check's set
// (buildDenySet) drops a closed advisory's title and keeps its id and CVE id;
// an open advisory keeps its title.
func TestRangeDenySetDropsClosedTitles(t *testing.T) {
	ds, err := buildDenySet([]heldStore{{root: t.TempDir(), st: closedAdvisoryState()}}, map[string]bool{})
	if err != nil {
		t.Fatal(err)
	}
	if hasNeedle(ds, KindAdvisoryTitle, titleClosed) {
		t.Fatalf("the range set holds a closed advisory's title: %+v", ds.needles)
	}
	for _, want := range []struct{ kind, text string }{
		{KindAdvisoryTitle, titleOpen}, {KindAdvisoryID, ghsaClosed}, {KindReservedCVE, cveOpen},
	} {
		if !hasNeedle(ds, want.kind, want.text) {
			t.Fatalf("the range set lacks %s %q: %+v", want.kind, want.text, ds.needles)
		}
	}
}

// TestPublicationDenySetKeepsClosedTitles: the set a publication or closing
// text is held to (othersDenySet) keeps a closed advisory's title with its id
// and CVE id, since the advisory stays private on GitHub, and leaves out the
// advisory being published.
func TestPublicationDenySetKeepsClosedTitles(t *testing.T) {
	ds := othersDenySet(closedAdvisoryState(), ghsaOpen)
	for _, want := range []struct{ kind, text string }{
		{KindAdvisoryTitle, titleClosed}, {KindAdvisoryID, ghsaClosed}, {KindReservedCVE, cveOpen},
	} {
		if !hasNeedle(ds, want.kind, want.text) {
			t.Fatalf("the publication set lacks %s %q: %+v", want.kind, want.text, ds.needles)
		}
	}
	if hasNeedle(ds, KindAdvisoryTitle, titleOpen) || hasNeedle(ds, KindAdvisoryID, ghsaOpen) {
		t.Fatalf("the publication set holds the advisory being published: %+v", ds.needles)
	}
}

// TestCheckSkipsWhatAMergeBringsInFromOutsideTheRange: a branch that has
// merged the default branch is judged by its own lines. A line the merge
// brings in from a parent behind the range's start is already there and is
// not read, while a line the merge adds against every parent still is.
func TestCheckSkipsWhatAMergeBringsInFromOutsideTheRange(t *testing.T) {
	cf := newCheckFixture(t)
	r := cf.f.repo
	r.Git("switch", "-q", "-c", "feature")
	r.Write("feature.txt", "clean\n")
	r.Commit("feature work")
	r.Git("switch", "-q", "main")
	r.Write("upstream.txt", "already public: "+ghsaRealShaped+"\n")
	r.Commit("main work")
	main := r.Git("rev-parse", "HEAD")
	r.Git("switch", "-q", "feature")
	r.Git("merge", "-q", "--no-ff", "--no-commit", "main")
	r.Write("resolved.txt", "names "+ghsaOpen+"\n")
	r.Git("add", "resolved.txt")
	r.Git("commit", "-q", "-m", "merge main")
	merge := r.Git("rev-parse", "HEAD")

	res, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: r.Root(), Range: main + "..HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	wantKinds(t, res, KindAdvisoryID)
	if f := res.Findings[0]; f.Source != "commit "+merge || f.Where != "added line in resolved.txt" || f.Line != 1 {
		t.Fatalf("merge finding %+v", f)
	}
}

// ghsaOtherShaped is a second advisory id of a real one's shape that no store
// holds, assembled at run time as ghsaRealShaped is.
var ghsaOtherShaped = "GHSA-" + strings.Join([]string{"3c4f", "5g6h", "7j8m"}, "-")

// TestCheckPassesAnIdThisRepositoryPublished: an advisory id the store does
// not hold passes when GitHub lists it among this repository's published
// advisories, asked once a check finds such an id and never before; an id it
// does not list as published, one it lists in another state included, stays
// refused as unlisted.
func TestCheckPassesAnIdThisRepositoryPublished(t *testing.T) {
	cf := newCheckFixture(t)
	cf.f.repo.Git("remote", "add", "origin", "https://github.com/example/example.git")
	g := newGHFake(t)
	b, err := json.Marshal([]map[string]any{
		advisoryJSON(ghsaRealShaped, "published", "high", "an upstream report", "its text"),
		advisoryJSON(ghsaOtherShaped, "draft", "high", "a report not yet published", "its text"),
	})
	if err != nil {
		t.Fatal(err)
	}
	g.answer("GET", listPath, "published", 0, string(b))

	wantKinds(t, cf.text(t, "clean text\n"))
	if n := len(g.calls()); n != 0 {
		t.Fatalf("a check that found no unlisted id asked GitHub %d time(s)", n)
	}
	wantKinds(t, cf.text(t, "see "+strings.ToLower(ghsaRealShaped)+"\n"))
	wantKinds(t, cf.text(t, "see "+ghsaRealShaped+" and "+ghsaOtherShaped+"\n"), KindUnlistedAdvisoryID)
	if n := len(g.calls()); n != 2 {
		t.Fatalf("two checks asked GitHub %d time(s), want once each", n)
	}
}

// TestCheckKeepsAnUnlistedIdGitHubCannotConfirm: with no github.com origin,
// or when GitHub cannot be read, an unlisted id stays refused and the note
// says GitHub could not be asked.
func TestCheckKeepsAnUnlistedIdGitHubCannotConfirm(t *testing.T) {
	cf := newCheckFixture(t)
	res := cf.text(t, "see "+ghsaRealShaped+"\n")
	wantKinds(t, res, KindUnlistedAdvisoryID)
	if !strings.Contains(res.Note, "published") {
		t.Fatalf("note %q does not say GitHub could not be asked", res.Note)
	}
	cf.f.repo.Git("remote", "add", "origin", "https://github.com/example/example.git")
	g := newGHFake(t)
	g.fail("GET", listPath, "published", "HTTP 502: Bad Gateway")
	res = cf.text(t, "see "+ghsaRealShaped+"\n")
	wantKinds(t, res, KindUnlistedAdvisoryID)
	if !strings.Contains(res.Note, "published") || strings.Contains(res.Note, "502") {
		t.Fatalf("note %q", res.Note)
	}
}

// TestParseAddedLinesKeepsLeadingEmptyLines: a hunk's empty added lines are
// kept, leading ones too, so each line of a run stays at its place in the
// file.
func TestParseAddedLinesKeepsLeadingEmptyLines(t *testing.T) {
	patch := "diff --git a/f.txt b/f.txt\n--- a/f.txt\n+++ b/f.txt\n@@ -1,0 +2,3 @@\n+\n+\n+names it\n"
	got, err := parseAddedLines([]byte(patch))
	if err != nil {
		t.Fatal(err)
	}
	want := []fileAdds{{path: "f.txt", runs: []addedRun{{line: 2, text: "\n\nnames it"}}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseAddedLines = %+v, want %+v", got, want)
	}
}

// TestCheckReadsAMergesLineAfterABlankItBringsIn: a merge that adds a line
// against every parent is read even when a blank line before it comes from
// one parent and is new against the other, so the two parents' diffs start
// their runs at different lines.
func TestCheckReadsAMergesLineAfterABlankItBringsIn(t *testing.T) {
	cf := newCheckFixture(t)
	r := cf.f.repo
	r.Write("f.txt", "a\n")
	r.Commit("base")
	r.Git("switch", "-q", "-c", "feature")
	r.Write("feature.txt", "clean\n")
	r.Commit("feature work")
	r.Git("switch", "-q", "main")
	r.Write("f.txt", "a\n\n")
	r.Commit("main adds a blank line")
	main := r.Git("rev-parse", "HEAD")
	r.Git("switch", "-q", "feature")
	r.Git("merge", "-q", "--no-ff", "--no-commit", "main")
	r.Write("f.txt", "a\n\n"+ghsaOpen+"\n")
	r.Git("add", "f.txt")
	r.Git("commit", "-q", "-m", "merge main")
	merge := r.Git("rev-parse", "HEAD")

	res, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: r.Root(), Range: main + "..HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	wantKinds(t, res, KindAdvisoryID)
	if f := res.Findings[0]; f.Source != "commit "+merge || f.Where != "added line in f.txt" || f.Line != 3 {
		t.Fatalf("merge finding %+v", f)
	}
}

// TestCheckReadsAnOctopusMergeOnce: in an octopus merge of branches in the
// range, a line one merged branch adds is read in that branch's commit and not
// again in the merge, and a line the merge adds against every parent is read
// in the merge.
func TestCheckReadsAnOctopusMergeOnce(t *testing.T) {
	cf := newCheckFixture(t)
	r := cf.f.repo
	base := r.Git("rev-parse", "HEAD")
	for _, b := range []struct{ name, file, body string }{
		{"one", "one.txt", "clean\n"},
		{"two", "two.txt", "names " + ghsaOpen + "\n"},
	} {
		r.Git("switch", "-q", "-c", b.name, base)
		r.Write(b.file, b.body)
		r.Commit(b.name + " work")
	}
	two := r.Git("rev-parse", "HEAD")
	r.Git("switch", "-q", "main")
	r.Write("main.txt", "clean\n")
	r.Commit("main work")
	r.Git("merge", "-q", "--no-ff", "--no-commit", "one", "two")
	r.Write("own.txt", "names "+ghsaClosed+"\n")
	r.Git("add", "own.txt")
	r.Git("commit", "-q", "-m", "merge one and two")
	merge := r.Git("rev-parse", "HEAD")

	res, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: r.Root(), Range: base + "..HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	wantKinds(t, res, KindAdvisoryID, KindAdvisoryID)
	got := map[string]bool{}
	for _, f := range res.Findings {
		got[f.Source+" "+f.Where] = true
	}
	for _, want := range []string{"commit " + two + " added line in two.txt", "commit " + merge + " added line in own.txt"} {
		if !got[want] {
			t.Fatalf("no finding at %q: %+v", want, res.Findings)
		}
	}
}

// TestCheckReadsARootCommitAMergeBringsIn: a commit with no parent in the
// range is read whatever log.showRoot says, so a line an unrelated history
// brings into a merge is read in its root commit.
func TestCheckReadsARootCommitAMergeBringsIn(t *testing.T) {
	cf := newCheckFixture(t)
	r := cf.f.repo
	r.Git("config", "log.showRoot", "false")
	base := r.Git("rev-parse", "HEAD")
	r.Git("switch", "-q", "--orphan", "other")
	r.Write("other.txt", "names "+ghsaOpen+"\n")
	r.Git("add", "other.txt")
	r.Git("commit", "-q", "-m", "other root")
	root := r.Git("rev-parse", "HEAD")
	r.Git("switch", "-q", "main")
	r.Git("merge", "-q", "--no-ff", "--allow-unrelated-histories", "-m", "merge other", "other")

	res, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: r.Root(), Range: base + "..HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	wantKinds(t, res, KindAdvisoryID)
	if f := res.Findings[0]; f.Source != "commit "+root || f.Where != "added line in other.txt" {
		t.Fatalf("root finding %+v", f)
	}
}
