package embargo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/tools"
)

// forge_state_test.go — iss-2610171250479046: the store learns from GitHub
// that an advisory it holds was published or closed outside `abcd embargo
// publish`. Advisory ids are of the GHSA-xxxx-xxxx-xxxx placeholder form.

// endedAt is when the fake forge says an advisory was published or closed.
var endedAt = time.Date(2026, 10, 12, 8, 0, 0, 0, time.UTC)

// heldOnly answers every read of ghsa with the advisory in state, stamped
// with at as its published_at or closed_at ("" leaves both null; any other
// string is sent as written).
func heldOnly(t *testing.T, g *ghFake, ghsa, state, at string) {
	t.Helper()
	a := advisoryJSON(ghsa, state, "high", "a sample summary", "the report")
	a["vulnerabilities"] = []map[string]any{}
	a["published_at"], a["closed_at"] = nil, nil
	if at != "" {
		a[state+"_at"] = at
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	g.answer("GET", listPath+"/"+ghsa, "", 0, string(b))
}

// heldFixture is an intake fixture whose store already holds advisory A
// (judged a vulnerability, in draft) and an untriaged B, neither listed in
// triage or draft any more.
func heldFixture(t *testing.T) (planFixture, fixture) {
	t.Helper()
	f, g := intakeFixture(t)
	s, _, err := openStore(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	pf := planFixture{f: f, s: s, g: g}
	untriaged := Advisory{GHSAID: ghsaB, IssueID: issB, Title: "a sample untriaged report", Severity: "minor", ForgeState: "triage"}
	pf.seed(t, vulnAdvisory(ghsaA, issA), untriaged)
	return pf, f
}

func (pf planFixture) held(t *testing.T, ghsa string) Advisory {
	t.Helper()
	st, err := pf.s.Verify()
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range st.Advisories {
		if a.GHSAID == ghsa {
			return a
		}
	}
	t.Fatalf("%s not held", ghsa)
	return Advisory{}
}

func (pf planFixture) checkText(t *testing.T, text string) CheckResult {
	t.Helper()
	res, err := Check(CheckRequest{Home: pf.f.home, RepoRoot: pf.f.repo.Root(), Files: []string{"-"}, Stdin: strings.NewReader(text)})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// TestIntakeRecordsAdvisoryPublishedOnGitHub: an advisory the store holds
// that GitHub reports published, by a hand on GitHub, gets the published_at
// GitHub gives, and the outbound check no longer refuses its id.
func TestIntakeRecordsAdvisoryPublishedOnGitHub(t *testing.T) {
	pf, f := heldFixture(t)
	heldOnly(t, pf.g, ghsaA, "published", endedAt.Format(time.RFC3339))
	heldOnly(t, pf.g, ghsaB, "triage", "")
	if c := pf.checkText(t, "fixes "+ghsaA+"\n"); c.Clean() {
		t.Fatal("the check does not refuse a held advisory before intake")
	}
	res := runIntake(t, f)
	a := pf.held(t, ghsaA)
	if a.PublishedAt == nil || !a.PublishedAt.Equal(endedAt) || a.ForgeState != "published" || a.ClosedAt != nil {
		t.Fatalf("held %+v", a)
	}
	if len(res.Ended) != 1 || res.Ended[0].GHSAID != ghsaA || res.Ended[0].State != "published" || len(res.Unread) != 0 {
		t.Fatalf("result %+v", res)
	}
	if b := pf.held(t, ghsaB); b.PublishedAt != nil || b.ClosedAt != nil {
		t.Fatalf("an advisory GitHub reports in triage was ended: %+v", b)
	}
	if c := pf.checkText(t, "fixes "+ghsaA+"\n"); !c.Clean() {
		t.Fatalf("the check still refuses an advisory GitHub reports published: %+v", c.Findings)
	}
	noWrites(t, pf.g)
}

// TestIntakeRecordsAdvisoryClosedOnGitHub: one the store judged with a
// closing verdict that GitHub reports closed gets its closed_at, and stays
// unpublished.
func TestIntakeRecordsAdvisoryClosedOnGitHub(t *testing.T) {
	pf, f := heldFixture(t)
	pf.seed(t, judged(strp(VerdictNotAVulnerability)), Advisory{GHSAID: ghsaB, IssueID: issB, Title: "a sample untriaged report", Severity: "minor", ForgeState: "triage"})
	heldOnly(t, pf.g, ghsaA, "closed", endedAt.Format(time.RFC3339))
	heldOnly(t, pf.g, ghsaB, "triage", "")
	res := runIntake(t, f)
	a := pf.held(t, ghsaA)
	if a.ClosedAt == nil || !a.ClosedAt.Equal(endedAt) || a.ForgeState != "closed" || a.PublishedAt != nil {
		t.Fatalf("held %+v", a)
	}
	if len(res.Ended) != 1 || res.Ended[0].State != "closed" {
		t.Fatalf("result %+v", res)
	}
}

// TestIntakeKeepsStateWhenGitHubCannotSay: a read that fails, a state this
// abcd does not know, or a timestamp not of GitHub's shape (or later than
// the clock) leaves the advisory as the store holds it, under embargo, and
// the result says which and why; the intake itself completes.
func TestIntakeKeepsStateWhenGitHubCannotSay(t *testing.T) {
	for name, set := range map[string]func(t *testing.T, g *ghFake){
		"read fails":     func(_ *testing.T, g *ghFake) { g.fail("GET", listPath+"/"+ghsaA, "", "HTTP 502: Bad Gateway") },
		"unknown state":  func(t *testing.T, g *ghFake) { heldOnly(t, g, ghsaA, "withdrawn", "") },
		"malformed time": func(t *testing.T, g *ghFake) { heldOnly(t, g, ghsaA, "published", "yesterday; rm -rf /") },
		"future time": func(t *testing.T, g *ghFake) {
			heldOnly(t, g, ghsaA, "published", fixedNow.Add(48*time.Hour).Format(time.RFC3339))
		},
	} {
		t.Run(name, func(t *testing.T) {
			pf, f := heldFixture(t)
			set(t, pf.g)
			heldOnly(t, pf.g, ghsaB, "triage", "")
			before := readFile(t, pf.s.StatePath())
			res := runIntake(t, f)
			a := pf.held(t, ghsaA)
			if a.PublishedAt != nil || a.ClosedAt != nil || a.ForgeState != "draft" {
				t.Fatalf("held %+v", a)
			}
			if len(res.Unread) != 1 || res.Unread[0].GHSAID != ghsaA || !strings.Contains(res.Unread[0].Reason, "unchanged") || len(res.Ended) != 0 {
				t.Fatalf("result %+v", res)
			}
			if strings.Contains(res.Unread[0].Reason, "rm -rf") {
				t.Fatalf("the reason repeats GitHub's value: %q", res.Unread[0].Reason)
			}
			after := readFile(t, pf.s.StatePath())
			if !strings.Contains(before, `"forge_state": "draft"`) || !strings.Contains(after, `"forge_state": "draft"`) {
				t.Fatalf("the held advisory's state changed:\n%s", after)
			}
			if c := pf.checkText(t, "fixes "+ghsaA+"\n"); c.Clean() {
				t.Fatal("the check no longer refuses an advisory GitHub could not be read for")
			}
		})
	}
}

// TestPlanRecordsHeldAdvisoryPublishedOnGitHub: a plan reads every held
// advisory, judged or not, and records the end GitHub reports for each; one
// it cannot read is listed with the reason and left as it is.
func TestPlanRecordsHeldAdvisoryPublishedOnGitHub(t *testing.T) {
	pf, _ := heldFixture(t)
	pf.tag("v0.9.0")
	heldOnly(t, pf.g, ghsaB, "published", endedAt.Format(time.RFC3339))
	pf.g.fail("GET", listPath+"/"+ghsaA, "", "HTTP 502: Bad Gateway")
	p := pf.plan(t, "")
	if b := pf.held(t, ghsaB); b.PublishedAt == nil || !b.PublishedAt.Equal(endedAt) || b.ForgeState != "published" {
		t.Fatalf("held %+v", b)
	}
	if a := pf.held(t, ghsaA); a.PublishedAt != nil || a.ClosedAt != nil {
		t.Fatalf("an advisory GitHub could not be read for was ended: %+v", a)
	}
	if len(p.Unread) != 1 || p.Unread[0].GHSAID != ghsaA || !strings.Contains(p.Unread[0].Reason, "unchanged") {
		t.Fatalf("plan %+v", p)
	}
	if len(p.Entries) != 0 || !strings.Contains(leftOutReason(t, p, ghsaA), "not resolved in v0.9.0") {
		t.Fatalf("plan %+v", p)
	}
	noWrites(t, pf.g)
}

// withoutGH leaves PATH holding git alone, so gh is not found.
func withoutGH(t *testing.T) {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	dir := t.TempDir()
	if err := os.Symlink(git, filepath.Join(dir, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

// signedOut makes gh answer a read of ghsa as it does with no account
// signed in: its exit status 4, authentication required, and its own words.
func signedOut(g *ghFake, ghsa string) {
	g.hook(`if [ "$path" = '` + listPath + "/" + ghsa + `' ]; then ` +
		`printf '%s\n' 'To get started with GitHub CLI, please run:  gh auth login' ` +
		`'Alternatively, populate the GH_TOKEN environment variable with a GitHub API authentication token.' >&2; exit 4; fi`)
}

// forgeAccessFaults are the failed reads every other read would meet too,
// each with what its refusal must name: a token without the advisory scope,
// a gh signed in to no account, a token GitHub rejects, and gh
// not installed ("" wants gh's missing-tool refusal).
var forgeAccessFaults = map[string]struct {
	set  func(t *testing.T, g *ghFake)
	want string
}{
	"scope": {func(_ *testing.T, g *ghFake) {
		g.fail("GET", listPath+"/"+ghsaA, "", "HTTP 403: Resource not accessible by integration")
	}, "gh auth refresh"},
	"signed out":      {func(_ *testing.T, g *ghFake) { signedOut(g, ghsaA) }, "gh auth login"},
	"bad credentials": {func(_ *testing.T, g *ghFake) { g.fail("GET", listPath+"/"+ghsaA, "", "HTTP 401: Bad credentials") }, "gh auth login"},
	"no gh":           {func(t *testing.T, _ *ghFake) { withoutGH(t) }, ""},
}

// mustRefuseForAccess holds err to the refusal fault wants.
func mustRefuseForAccess(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		var missing *tools.MissingError
		if !errors.As(err, &missing) {
			t.Fatalf("err = %v, want gh's missing-tool refusal", err)
		}
		return
	}
	mustRefuse(t, err, want)
	if strings.Contains(err.Error(), "GH_TOKEN") || strings.Contains(err.Error(), "Bad credentials") {
		t.Fatalf("the refusal repeats gh's own words: %v", err)
	}
}

// TestForgeAccessRefusesWholePlan: a held advisory GitHub refuses to show
// for want of a scope, or for want of a signed-in account or an accepted
// token, or a gh that is not installed, refuses the whole plan with its
// remedy, and records nothing, even an end read for another held advisory;
// only a transient fault leaves one advisory unread.
func TestForgeAccessRefusesWholePlan(t *testing.T) {
	for name, tc := range forgeAccessFaults {
		t.Run(name, func(t *testing.T) {
			pf, _ := heldFixture(t)
			pf.tag("v0.9.0")
			heldOnly(t, pf.g, ghsaB, "published", endedAt.Format(time.RFC3339))
			tc.set(t, pf.g)
			before := readFile(t, pf.s.StatePath())
			_, err := planPublication(PublishRequest{Home: pf.f.home, RepoRoot: pf.f.repo.Root(), Now: func() time.Time { return fixedNow.Add(time.Hour) }})
			mustRefuseForAccess(t, err, tc.want)
			if readFile(t, pf.s.StatePath()) != before {
				t.Fatal("a refused plan recorded an end")
			}
		})
	}
}

// TestForgeAccessRefusesWholeIntake: the same faults on the read of a held
// advisory refuse the whole intake with its remedy, recording nothing. (A
// missing gh refuses at the listing, before any held advisory is read.)
func TestForgeAccessRefusesWholeIntake(t *testing.T) {
	for name, tc := range forgeAccessFaults {
		if tc.want == "" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			pf, f := heldFixture(t)
			tc.set(t, pf.g)
			heldOnly(t, pf.g, ghsaB, "published", endedAt.Format(time.RFC3339))
			before := readFile(t, pf.s.StatePath())
			_, err := Intake(IntakeRequest{Home: f.home, RepoRoot: f.repo.Root(), Now: func() time.Time { return fixedNow }})
			mustRefuseForAccess(t, err, tc.want)
			if readFile(t, pf.s.StatePath()) != before {
				t.Fatal("a refused intake recorded something")
			}
		})
	}
}

// judged is advisory A of heldFixture with verdict (nil: untriaged).
func judged(verdict *string) Advisory {
	a := vulnAdvisory(ghsaA, issA)
	a.Verdict = verdict
	if verdict == nil {
		a.DecidedBy, a.DecidedAt = nil, nil
	} else if *verdict != VerdictVulnerability {
		a.Reason = strp("a sample reason")
	}
	return a
}

// TestClosedOnGitHubRecordedOnlyForClosingVerdict: GitHub's closed is
// recorded only for an advisory whose verdict already ends its private
// track: a closing verdict, or the public track. One the store holds as a
// vulnerability, or untriaged, keeps its state, under embargo and still a
// candidate for publication, and is listed with a fixed reason naming what
// the store holds, for a person to settle.
func TestClosedOnGitHubRecordedOnlyForClosingVerdict(t *testing.T) {
	for name, tc := range map[string]struct {
		verdict *string
		record  bool
		holds   string
	}{
		"untriaged":           {nil, false, "untriaged"},
		"vulnerability":       {strp(VerdictVulnerability), false, "as a vulnerability"},
		"public-track":        {strp(VerdictPublicTrack), true, ""},
		"not-abcds":           {strp(VerdictNotAbcds), true, ""},
		"not-a-vulnerability": {strp(VerdictNotAVulnerability), true, ""},
	} {
		t.Run(name, func(t *testing.T) {
			f, g := intakeFixture(t)
			s, _, err := openStore(f.home, f.repo.Root())
			if err != nil {
				t.Fatal(err)
			}
			pf := planFixture{f: f, s: s, g: g}
			pf.seed(t, judged(tc.verdict))
			heldOnly(t, g, ghsaA, "closed", endedAt.Format(time.RFC3339))
			res := runIntake(t, f)
			a := pf.held(t, ghsaA)
			if tc.record {
				if a.ClosedAt == nil || !a.ClosedAt.Equal(endedAt) || a.ForgeState != "closed" || len(res.Ended) != 1 || len(res.Unread) != 0 {
					t.Fatalf("held %+v, result %+v", a, res)
				}
				return
			}
			if a.ClosedAt != nil || a.ForgeState != "draft" || len(res.Ended) != 0 {
				t.Fatalf("closed recorded for a %s advisory: held %+v, result %+v", name, a, res)
			}
			if len(res.Unread) != 1 || res.Unread[0].GHSAID != ghsaA {
				t.Fatalf("unread %+v", res.Unread)
			}
			r := res.Unread[0].Reason
			for _, want := range []string{"closed on GitHub", tc.holds, "not recorded", "a person settles it"} {
				if !strings.Contains(r, want) {
					t.Fatalf("reason %q does not say %q", r, want)
				}
			}
		})
	}
}

// TestClosedOnGitHubReadAgain: an advisory judged a vulnerability that
// GitHub reports closed is read again by the next plan, so a reopen is seen:
// it then plans as before, and is no longer listed.
func TestClosedOnGitHubReadAgain(t *testing.T) {
	pf, _ := heldFixture(t)
	pf.tag("v0.9.0")
	heldOnly(t, pf.g, ghsaB, "triage", "")
	a := advisoryJSON(ghsaA, "closed", "high", "a sample summary", "the report")
	a["closed_at"] = endedAt.Format(time.RFC3339)
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	pf.g.answer("GET", listPath+"/"+ghsaA, "", 1, string(b))
	pf.remote(t, ghsaA, "draft", "the report")
	pf.g.answer("GET", listPath+"/"+ghsaA, "", 2, readGHAnswer(t, pf.g, ghsaA))

	first := pf.plan(t, "")
	if len(first.Unread) != 1 || first.Unread[0].GHSAID != ghsaA || !strings.Contains(first.Unread[0].Reason, "as a vulnerability") {
		t.Fatalf("first plan %+v", first)
	}
	if h := pf.held(t, ghsaA); h.ClosedAt != nil {
		t.Fatalf("closed recorded: %+v", h)
	}
	second := pf.plan(t, "")
	if len(second.Unread) != 0 {
		t.Fatalf("a reopened advisory is still listed: %+v", second.Unread)
	}
	reads := 0
	for _, c := range pf.g.calls() {
		if c.has(listPath + "/" + ghsaA) {
			reads++
		}
	}
	if reads != 2 {
		t.Fatalf("%d reads of the advisory, want one per plan", reads)
	}
}

// readGHAnswer is the fake's unnumbered answer for a read of ghsa.
func readGHAnswer(t *testing.T, g *ghFake, ghsa string) string {
	t.Helper()
	return readFile(t, filepath.Join(g.dir, "resp", ghKey("GET", listPath+"/"+ghsa, "")))
}

// TestMissingForgeTimeLeavesUnread: every timestamp the store records from
// GitHub is GitHub's own, so an end GitHub reports with no published_at or
// closed_at is not recorded at the clock's time: the advisory is listed,
// with a fixed reason, and left as the store holds it.
func TestMissingForgeTimeLeavesUnread(t *testing.T) {
	for _, state := range []string{"published", "closed"} {
		t.Run(state, func(t *testing.T) {
			pf, f := heldFixture(t)
			pf.seed(t, judged(strp(VerdictNotAVulnerability)))
			heldOnly(t, pf.g, ghsaA, state, "")
			res := runIntake(t, f)
			if a := pf.held(t, ghsaA); a.PublishedAt != nil || a.ClosedAt != nil || a.ForgeState != "draft" {
				t.Fatalf("held %+v", a)
			}
			if len(res.Ended) != 0 || len(res.Unread) != 1 || !strings.Contains(res.Unread[0].Reason, "gives no "+state+"_at") {
				t.Fatalf("result %+v", res)
			}
		})
	}
}

// TestPlanListsWhatItRecords: a plan that records an end GitHub reports for
// a held advisory says so in Ended, and the ends are not bound into the
// digest, which stays the one over the repository, the release and the
// entries alone.
func TestPlanListsWhatItRecords(t *testing.T) {
	pf, _ := heldFixture(t)
	pf.tag("v0.9.0")
	heldOnly(t, pf.g, ghsaB, "published", endedAt.Format(time.RFC3339))
	pf.g.fail("GET", listPath+"/"+ghsaA, "", "HTTP 502: Bad Gateway")
	p := pf.plan(t, "")
	if len(p.Ended) != 1 || p.Ended[0].GHSAID != ghsaB || p.Ended[0].State != "published" || !p.Ended[0].At.Equal(endedAt) {
		t.Fatalf("ended %+v", p.Ended)
	}
	data, err := json.Marshal(struct {
		Repo    string      `json:"repo"`
		Release string      `json:"release"`
		Entries []PlanEntry `json:"entries"`
	}{p.Repo, p.Release, p.Entries})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if p.Digest != hex.EncodeToString(sum[:]) {
		t.Fatal("the digest is not the one over the repository, the release and the entries")
	}
	bare := p
	bare.Ended, bare.Unread, bare.Restores, bare.LeftOut = nil, nil, nil, nil
	if planDigest(bare) != p.Digest {
		t.Fatal("the ends, or another list outside the entries, are bound into the digest")
	}
	again := pf.plan(t, "")
	if len(again.Ended) != 0 || again.Digest != p.Digest {
		t.Fatalf("a plan with nothing left to record: ended %+v, digest changed %v", again.Ended, again.Digest != p.Digest)
	}
}

// notShown are the answers gh gives when GitHub shows one advisory to no
// one this token speaks for: a deleted advisory, or one this account cannot
// see. Neither names a scope.
var notShown = map[string]string{
	"404": "gh: Not Found (HTTP 404)",
	"403": "gh: Forbidden (HTTP 403)",
}

// TestOneAdvisoryNotShownLeavesItUnread: a 404 or a 403 on one held
// advisory's read is no fault of the run: the plan and the intake go on,
// that advisory is listed unread with a fixed reason, and the other's end
// is recorded.
func TestOneAdvisoryNotShownLeavesItUnread(t *testing.T) {
	for code, stderr := range notShown {
		check := func(t *testing.T, pf planFixture, unread []UnreadAdvisory, ended []ForgeEnd) {
			t.Helper()
			if len(unread) != 1 || unread[0].GHSAID != ghsaA || !strings.Contains(unread[0].Reason, "no such advisory to this account") ||
				!strings.Contains(unread[0].Reason, "unchanged") || strings.Contains(unread[0].Reason, "Forbidden") {
				t.Fatalf("unread %+v", unread)
			}
			if len(ended) != 1 || ended[0].GHSAID != ghsaB {
				t.Fatalf("ended %+v", ended)
			}
			if b := pf.held(t, ghsaB); b.PublishedAt == nil || !b.PublishedAt.Equal(endedAt) {
				t.Fatalf("held %+v", b)
			}
			if a := pf.held(t, ghsaA); a.PublishedAt != nil || a.ClosedAt != nil {
				t.Fatalf("held %+v", a)
			}
		}
		t.Run("plan "+code, func(t *testing.T) {
			pf, _ := heldFixture(t)
			pf.tag("v0.9.0")
			heldOnly(t, pf.g, ghsaB, "published", endedAt.Format(time.RFC3339))
			pf.g.fail("GET", listPath+"/"+ghsaA, "", stderr)
			p := pf.plan(t, "")
			check(t, pf, p.Unread, p.Ended)
			if !strings.Contains(leftOutReason(t, p, ghsaA), "not resolved in v0.9.0") {
				t.Fatalf("plan %+v", p)
			}
		})
		t.Run("intake "+code, func(t *testing.T) {
			pf, f := heldFixture(t)
			heldOnly(t, pf.g, ghsaB, "published", endedAt.Format(time.RFC3339))
			pf.g.fail("GET", listPath+"/"+ghsaA, "", stderr)
			res := runIntake(t, f)
			check(t, pf, res.Unread, res.Ended)
		})
	}
}

// TestEveryAdvisoryNotShownListingDecides: when every held advisory's read
// a plan makes answers 404 or 403, the plan asks for the repository's
// advisory listing once, which tells a missing scope from advisories GitHub
// no longer shows. A listing that answers proves the scope: the advisories
// are unread and the plan goes on, so a store whose one advisory was
// deleted on GitHub still plans. A listing GitHub refuses is a missing
// scope, refused whole with its remedy; a listing that fails otherwise
// refuses whole with its own fixed remedy. Either refusal records nothing
// and repeats none of gh's words. An intake, whose own listing has just
// answered, goes on without asking again.
func TestEveryAdvisoryNotShownListingDecides(t *testing.T) {
	for code, stderr := range notShown {
		t.Run("one deleted "+code, func(t *testing.T) {
			pf, _ := heldFixture(t)
			pf.seed(t, vulnAdvisory(ghsaA, issA))
			pf.tag("v0.9.0")
			pf.g.fail("GET", listPath+"/"+ghsaA, "", stderr)
			p := pf.plan(t, "")
			if len(p.Unread) != 1 || p.Unread[0].GHSAID != ghsaA || !strings.Contains(p.Unread[0].Reason, "no such advisory to this account") {
				t.Fatalf("plan %+v", p)
			}
			if !strings.Contains(leftOutReason(t, p, ghsaA), "not resolved in v0.9.0") {
				t.Fatalf("plan %+v", p)
			}
			listed := 0
			for _, c := range pf.g.calls() {
				if c.has(listPath) {
					listed++
				}
			}
			if listed != 1 {
				t.Fatalf("%d listings, want one", listed)
			}
			noWrites(t, pf.g)
		})
		for name, tc := range map[string]struct{ listing, want string }{
			"listing refused for scope " + code: {stderr, "gh auth refresh"},
			"listing refused in words " + code:  {"gh: Resource not accessible by personal access token (HTTP 403)", "gh auth refresh"},
			"listing fails transiently " + code: {"gh: Server Error (HTTP 502)", "the listing that tells"},
		} {
			t.Run(name, func(t *testing.T) {
				pf, _ := heldFixture(t)
				pf.tag("v0.9.0")
				pf.g.fail("GET", listPath+"/"+ghsaA, "", stderr)
				pf.g.fail("GET", listPath+"/"+ghsaB, "", stderr)
				pf.g.fail("GET", listPath, "triage", tc.listing)
				before := readFile(t, pf.s.StatePath())
				_, err := planPublication(PublishRequest{Home: pf.f.home, RepoRoot: pf.f.repo.Root(), Now: func() time.Time { return fixedNow.Add(time.Hour) }})
				mustRefuse(t, err, tc.want)
				for _, words := range []string{"Forbidden", "Not Found", "Server Error", "personal access token"} {
					if strings.Contains(err.Error(), words) {
						t.Fatalf("the refusal repeats gh's own words: %v", err)
					}
				}
				if readFile(t, pf.s.StatePath()) != before {
					t.Fatal("a refused plan recorded something")
				}
			})
		}
		t.Run("intake "+code, func(t *testing.T) {
			pf, f := heldFixture(t)
			pf.g.fail("GET", listPath+"/"+ghsaA, "", stderr)
			pf.g.fail("GET", listPath+"/"+ghsaB, "", stderr)
			res := runIntake(t, f)
			if len(res.Unread) != 2 || len(res.Ended) != 0 {
				t.Fatalf("result %+v", res)
			}
		})
	}
}

// TestHTTP401IsSignedOut: GitHub's HTTP 401, which gh reports when GitHub
// rejects the token it holds, is read as signed out on its own, whatever
// gh's exit status, and refuses with the sign-in remedy, repeating none of
// gh's words.
func TestHTTP401IsSignedOut(t *testing.T) {
	err := errors.New("gh: Bad credentials (HTTP 401)")
	if !ghSignedOut(err) {
		t.Fatal("an HTTP 401 is not read as signed out")
	}
	got := remoteError(err, false)
	mustRefuse(t, got, "gh auth login")
	if !refusesWhole(got) || strings.Contains(got.Error(), "Bad credentials") {
		t.Fatalf("remoteError(401) = %v", got)
	}
	if ghSignedOut(errors.New("gh: Server Error (HTTP 500)")) {
		t.Fatal("an HTTP 500 is read as signed out")
	}
}
