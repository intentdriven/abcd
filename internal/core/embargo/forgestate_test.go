package embargo

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
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

// TestIntakeRecordsAdvisoryClosedOnGitHub: one GitHub reports closed gets
// its closed_at, and stays unpublished.
func TestIntakeRecordsAdvisoryClosedOnGitHub(t *testing.T) {
	pf, f := heldFixture(t)
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
