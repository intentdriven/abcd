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

// TestForgeAccessRefusesWholePlan: a held advisory GitHub refuses to show
// for want of a scope, or a gh that is not installed, refuses the whole plan
// with its remedy, and records nothing, even an end read for another held
// advisory; only a transient fault leaves one advisory unread.
func TestForgeAccessRefusesWholePlan(t *testing.T) {
	for name, set := range map[string]func(t *testing.T, g *ghFake){
		"scope": func(_ *testing.T, g *ghFake) {
			g.fail("GET", listPath+"/"+ghsaA, "", "HTTP 403: Resource not accessible by integration")
		},
		"no gh": func(t *testing.T, _ *ghFake) { withoutGH(t) },
	} {
		t.Run(name, func(t *testing.T) {
			pf, _ := heldFixture(t)
			pf.tag("v0.9.0")
			heldOnly(t, pf.g, ghsaB, "published", endedAt.Format(time.RFC3339))
			set(t, pf.g)
			before := readFile(t, pf.s.StatePath())
			_, err := planPublication(PublishRequest{Home: pf.f.home, RepoRoot: pf.f.repo.Root(), Now: func() time.Time { return fixedNow.Add(time.Hour) }})
			var missing *tools.MissingError
			switch name {
			case "scope":
				mustRefuse(t, err, "repository_advisories")
				mustRefuse(t, err, "gh auth refresh")
			default:
				if !errors.As(err, &missing) {
					t.Fatalf("err = %v, want gh's missing-tool refusal", err)
				}
			}
			if readFile(t, pf.s.StatePath()) != before {
				t.Fatal("a refused plan recorded an end")
			}
		})
	}
}

// TestForgeAccessRefusesWholeIntake: a held advisory GitHub refuses to show
// for want of a scope refuses the whole intake with its remedy, recording
// nothing.
func TestForgeAccessRefusesWholeIntake(t *testing.T) {
	pf, f := heldFixture(t)
	pf.g.fail("GET", listPath+"/"+ghsaA, "", "HTTP 403: Resource not accessible by integration")
	heldOnly(t, pf.g, ghsaB, "published", endedAt.Format(time.RFC3339))
	before := readFile(t, pf.s.StatePath())
	_, err := Intake(IntakeRequest{Home: f.home, RepoRoot: f.repo.Root(), Now: func() time.Time { return fixedNow }})
	mustRefuse(t, err, "gh auth refresh")
	if readFile(t, pf.s.StatePath()) != before {
		t.Fatal("a refused intake recorded something")
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
