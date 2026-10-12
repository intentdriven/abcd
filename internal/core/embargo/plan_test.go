package embargo

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// planFixture is a checkout with an embargo store, a fake gh, and helpers to
// make the history a release is cut from: fix commits, resolved records that
// name them, and tags. Advisory ids are of the GHSA-xxxx-xxxx-xxxx placeholder
// form and the package is the example module (examples-use-reserved-identifiers).
type planFixture struct {
	f fixture
	s Store
	g *ghFake
}

const examplePackage = "example.com/example"

func newPlanFixture(t *testing.T) planFixture {
	t.Helper()
	f := newFixture(t)
	s, _, err := Open(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	return planFixture{f: f, s: s, g: newGHFake(t)}
}

// vulnAdvisory is an advisory held with the vulnerability verdict.
func vulnAdvisory(ghsa, issID string) Advisory {
	at := fixedNow
	return Advisory{GHSAID: ghsa, IssueID: issID, Title: "a sample report " + issID, Severity: "major", ForgeState: "draft",
		Verdict: strp(VerdictVulnerability), DecidedBy: strp("Alice"), DecidedAt: &at}
}

func (pf planFixture) seed(t *testing.T, advisories ...Advisory) {
	t.Helper()
	st := emptyState()
	st.Repo = "example/example"
	at := fixedNow
	st.IntakeAt = &at
	st.Advisories = advisories
	if err := pf.s.SaveState(st); err != nil {
		t.Fatal(err)
	}
}

// commit makes one commit changing path and returns its id.
func (pf planFixture) commit(path, content, msg string) string {
	pf.f.repo.Write(path, content)
	pf.f.repo.Commit(msg)
	return pf.f.repo.Git("rev-parse", "HEAD")
}

// resolve commits issID's record in the public ledger's resolved/ folder,
// naming fix as the commit that resolved it (empty: no resolved_by).
func (pf planFixture) resolve(issID, fix string) {
	rec := "---\nschema_version: 1\nid: \"" + issID + "\"\nslug: \"private-report\"\nseverity: \"major\"\n" +
		"category: \"security\"\nsource: \"security-advisory\"\nresolution: \"fixed\"\n"
	if fix != "" {
		rec += "resolved_by:\n  commit: \"" + fix + "\"\n"
	}
	rec += "---\n\nA private report, redacted.\n"
	pf.commit(".abcd/work/issues/resolved/"+issID+"-private-report.md", rec, "resolve a record")
}

func (pf planFixture) tag(name string) { pf.f.repo.Git("tag", name) }

func vulnJSON(name, rng string) map[string]any {
	v := map[string]any{
		"package":                  map[string]any{"ecosystem": "go", "name": name},
		"vulnerable_version_range": nil,
		"patched_versions":         nil,
		"vulnerable_functions":     []string{},
	}
	if name == "" {
		v["package"] = nil
	}
	if rng != "" {
		v["vulnerable_version_range"] = rng
	}
	return v
}

// remote answers every read of ghsa with the advisory in state, carrying
// description and vulns.
func (pf planFixture) remote(t *testing.T, ghsa, state, description string, vulns ...map[string]any) {
	t.Helper()
	a := advisoryJSON(ghsa, state, "high", "a sample summary", description)
	if vulns == nil {
		vulns = []map[string]any{}
	}
	a["vulnerabilities"] = vulns
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	pf.g.answer("GET", listPath+"/"+ghsa, "", 0, string(b))
}

func (pf planFixture) plan(t *testing.T, release string) PublicationPlan {
	t.Helper()
	p, err := PlanPublication(PublishRequest{Home: pf.f.home, RepoRoot: pf.f.repo.Root(), Release: release,
		Now: func() time.Time { return fixedNow.Add(time.Hour) }})
	if err != nil {
		t.Fatalf("PlanPublication: %v", err)
	}
	return p
}

func leftOutReason(t *testing.T, p PublicationPlan, ghsa string) string {
	t.Helper()
	for _, l := range p.LeftOut {
		if l.GHSAID == ghsa {
			return l.Reason
		}
	}
	t.Fatalf("%s is not left out of the plan: %+v", ghsa, p)
	return ""
}

const (
	issA = "iss-2610120000000001"
	issB = "iss-2610120000000002"
)

// TestFirstReleaseTagContainingFix: a fix first released in v0.9.0 publishes
// as patched in 0.9.0 though the release being cut, which holds the resolved
// record, is v0.10.0 (a hand-run typed the later version at first).
func TestFirstReleaseTagContainingFix(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	fix := pf.commit("parser.go", "package parser\n", "fix: tighten the parser")
	pf.tag("v0.9.0")
	pf.resolve(issA, fix)
	pf.tag("v0.10.0")
	pf.remote(t, ghsaOpen, "draft", "the report", vulnJSON(examplePackage, ""))

	p := pf.plan(t, "")
	if p.Release != "v0.10.0" || len(p.Entries) != 1 || len(p.LeftOut) != 0 {
		t.Fatalf("plan %+v", p)
	}
	e := p.Entries[0]
	if e.GHSAID != ghsaOpen || e.IssueID != issA || e.FixCommit != fix || e.FirstTag != "v0.9.0" || e.PatchedVersion != "0.9.0" {
		t.Fatalf("entry %+v", e)
	}
	if len(e.Vulnerabilities) != 1 || e.Vulnerabilities[0].PatchedVersions != "0.9.0" ||
		e.Vulnerabilities[0].VulnerableVersionRange != "< 0.9.0" || e.Vulnerabilities[0].Package.Name != examplePackage {
		t.Fatalf("vulnerabilities %+v", e.Vulnerabilities)
	}
	if !hex64.MatchString(p.Digest) || p.ApproveWith != "abcd embargo publish --release v0.10.0 --approve "+p.Digest {
		t.Fatalf("digest %q, approve_with %q", p.Digest, p.ApproveWith)
	}
}

// TestPrereleaseTagsIgnored: only strict vMAJOR.MINOR.PATCH tags are
// releases, for the first tag holding a fix and for the default release.
func TestPrereleaseTagsIgnored(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	fix := pf.commit("parser.go", "package parser\n", "fix: tighten the parser")
	pf.tag("v0.9.0-rc.1")
	pf.tag("v0.8")
	pf.tag("release-0.8.0")
	pf.resolve(issA, fix)
	pf.tag("v0.9.0")
	pf.commit("other.go", "package other\n", "later")
	pf.tag("v0.9.1-rc.1")
	pf.tag("v01.0.0")
	pf.remote(t, ghsaOpen, "draft", "the report", vulnJSON(examplePackage, ""))

	p := pf.plan(t, "")
	if p.Release != "v0.9.0" || len(p.Entries) != 1 || p.Entries[0].FirstTag != "v0.9.0" || p.Entries[0].PatchedVersion != "0.9.0" {
		t.Fatalf("plan %+v", p)
	}
	_, err := PlanPublication(PublishRequest{Home: pf.f.home, RepoRoot: pf.f.repo.Root(), Release: "v0.9.0-rc.1"})
	mustRefuse(t, err, "vMAJOR.MINOR.PATCH")
	_, err = PlanPublication(PublishRequest{Home: pf.f.home, RepoRoot: pf.f.repo.Root(), Release: "v7.7.7"})
	mustRefuse(t, err, "no tag")
}

// TestFixInNoTagLeftOut: a fix no release tag holds is left out, saying so.
func TestFixInNoTagLeftOut(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	r := pf.f.repo
	r.Git("switch", "-q", "-c", "side")
	fix := pf.commit("parser.go", "package parser\n", "fix on a branch no tag holds")
	r.Git("switch", "-q", "main")
	pf.resolve(issA, fix)
	pf.tag("v0.9.0")
	pf.remote(t, ghsaOpen, "draft", "the report", vulnJSON(examplePackage, ""))

	p := pf.plan(t, "")
	if len(p.Entries) != 0 || !strings.Contains(leftOutReason(t, p, ghsaOpen), "no release tag") {
		t.Fatalf("plan %+v", p)
	}
	if p.ApproveWith != "" {
		t.Fatalf("an empty plan offers an approval: %q", p.ApproveWith)
	}
}

// TestUnresolvedRecordLeftOut: a record not in the release's resolved/ folder,
// or resolved there without naming its fix commit, is left out with the
// reason; an advisory without the vulnerability verdict is not planned.
func TestUnresolvedRecordLeftOut(t *testing.T) {
	pf := newPlanFixture(t)
	untriaged := Advisory{GHSAID: ghsaClosed, IssueID: "iss-2610120000000003", Title: "a sample untriaged report", Severity: "minor", ForgeState: "triage"}
	pf.seed(t, vulnAdvisory(ghsaOpen, issA), vulnAdvisory(ghsaPublic, issB), untriaged)
	pf.resolve(issB, "")
	pf.tag("v0.9.0")
	p := pf.plan(t, "")
	if len(p.Entries) != 0 || len(p.LeftOut) != 2 {
		t.Fatalf("plan %+v", p)
	}
	if r := leftOutReason(t, p, ghsaOpen); !strings.Contains(r, "not resolved in v0.9.0") {
		t.Fatalf("reason %q", r)
	}
	if r := leftOutReason(t, p, ghsaPublic); !strings.Contains(r, "names no fix commit") {
		t.Fatalf("reason %q", r)
	}
	if len(pf.g.calls()) != 0 {
		t.Fatal("an advisory left out by the release was read from GitHub")
	}
}

// TestMissingPackageLeftOut: an advisory with no package on GitHub is left
// out, asking the person to set it there.
func TestMissingPackageLeftOut(t *testing.T) {
	for name, vulns := range map[string][]map[string]any{
		"no vulnerabilities": nil,
		"null package":       {vulnJSON("", "")},
	} {
		t.Run(name, func(t *testing.T) {
			pf := newPlanFixture(t)
			pf.seed(t, vulnAdvisory(ghsaOpen, issA))
			fix := pf.commit("parser.go", "package parser\n", "fix")
			pf.resolve(issA, fix)
			pf.tag("v0.9.0")
			pf.remote(t, ghsaOpen, "draft", "the report", vulns...)
			p := pf.plan(t, "")
			if len(p.Entries) != 0 || !strings.Contains(leftOutReason(t, p, ghsaOpen), "set the package on GitHub") {
				t.Fatalf("plan %+v", p)
			}
		})
	}
}

// TestRangeKeepsLowerBound: the affected range ends below the patched version
// and keeps a lower bound the advisory already carries.
func TestRangeKeepsLowerBound(t *testing.T) {
	for existing, want := range map[string]string{
		"":                     "< 0.9.0",
		"<= 0.8.9":             "< 0.9.0",
		"= 0.8.8":              "< 0.9.0",
		">= 0.5.0":             ">= 0.5.0, < 0.9.0",
		">= 0.5.0, <= 0.8.9":   ">= 0.5.0, < 0.9.0",
		"> 0.4.1, < 0.8.0":     "> 0.4.1, < 0.9.0",
		">= 0.5.0, < 0.9.0":    ">= 0.5.0, < 0.9.0",
		">= v0.5.0, <= v0.8.9": ">= v0.5.0, < 0.9.0",
	} {
		var p *string
		if existing != "" {
			p = &existing
		}
		got, err := affectedRange(p, "0.9.0")
		if err != nil || got != want {
			t.Errorf("affectedRange(%q) = %q, %v; want %q", existing, got, err, want)
		}
	}
	upper := ">= 0.9.0"
	if _, err := affectedRange(&upper, "0.9.0"); err == nil {
		t.Error("a lower bound at the patched version was kept")
	}

	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	fix := pf.commit("parser.go", "package parser\n", "fix")
	pf.resolve(issA, fix)
	pf.tag("v0.9.0")
	pf.remote(t, ghsaOpen, "draft", "the report", vulnJSON(examplePackage, ">= 0.5.0, <= 0.8.9"))
	p := pf.plan(t, "v0.9.0")
	if len(p.Entries) != 1 || p.Entries[0].Vulnerabilities[0].VulnerableVersionRange != ">= 0.5.0, < 0.9.0" {
		t.Fatalf("plan %+v", p)
	}
}

// TestRangeGrammarValidated: only GitHub's range form enters a plan; an
// advisory whose range on GitHub is in another form is left out rather than
// overwritten.
func TestRangeGrammarValidated(t *testing.T) {
	for r, ok := range map[string]bool{
		"< 0.9.0":               true,
		">= 0.5.0, < 0.9.0":     true,
		"= 0.6.8":               true,
		"<= 1.0.0-rc.1":         true,
		"":                      false,
		"<0.9.0":                false,
		"< 0.9.0,>= 0.5.0":      false,
		">= 0.6.2 and <= 0.6.8": false,
		"all":                   false,
		"~> 0.9":                false,
		"< 0.9.0, ":             false,
	} {
		if validRange(r) != ok {
			t.Errorf("validRange(%q) = %v, want %v", r, !ok, ok)
		}
	}
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	fix := pf.commit("parser.go", "package parser\n", "fix")
	pf.resolve(issA, fix)
	pf.tag("v0.9.0")
	pf.remote(t, ghsaOpen, "draft", "the report", vulnJSON(examplePackage, ">= 0.6.2 and <= 0.6.8"))
	p := pf.plan(t, "")
	if len(p.Entries) != 0 || !strings.Contains(leftOutReason(t, p, ghsaOpen), "range") {
		t.Fatalf("plan %+v", p)
	}
}

// TestScanFindingLeavesAdvisoryOut: an advisory whose text carries a secret,
// another account's home path or a banned private name is left out, naming
// the kind of finding and never the finding.
func TestScanFindingLeavesAdvisoryOut(t *testing.T) {
	token := "gh" + "p_" + strings.Repeat("D", 36) // synthetic, split so no secret scanner reads it as one
	for name, desc := range map[string]string{
		"token":      "reproduce with " + token,
		"other home": "the file at /Users/zed-example/project/x.go",
	} {
		t.Run(name, func(t *testing.T) {
			pf := newPlanFixture(t)
			pf.seed(t, vulnAdvisory(ghsaOpen, issA))
			fix := pf.commit("parser.go", "package parser\n", "fix")
			pf.resolve(issA, fix)
			pf.tag("v0.9.0")
			pf.remote(t, ghsaOpen, "draft", "line one\n"+desc, vulnJSON(examplePackage, ""))
			p := pf.plan(t, "")
			r := leftOutReason(t, p, ghsaOpen)
			if len(p.Entries) != 0 || !strings.Contains(r, "description") || !strings.Contains(r, "line 2") ||
				strings.Contains(r, token) || strings.Contains(r, "zed-example") {
				t.Fatalf("plan %+v", p)
			}
		})
	}
}

// TestPlanLeavesOutAdvisoryNamingAnother: an advisory whose text names
// another advisory still under embargo (its id, reserved CVE id, or its
// title by the check's title rule) is left out, naming why and never the
// other advisory, since publishing it would disclose the other one. Naming
// itself, an advisory already published, or a closed advisory's title, does
// not.
func TestPlanLeavesOutAdvisoryNamingAnother(t *testing.T) {
	at := fixedNow
	other := Advisory{GHSAID: ghsaClosed, IssueID: "iss-2610120000000003", Title: titleClosed, CVEID: strp(cveOpen), Severity: "minor",
		ForgeState: "closed", Verdict: strp(VerdictNotAVulnerability), DecidedBy: strp("Alice"), DecidedAt: &at, Reason: strp("documented"), ClosedAt: &at}
	published := Advisory{GHSAID: ghsaPublished, IssueID: "iss-2610120000000004", Title: titlePublishd, Severity: "major",
		ForgeState: "published", Verdict: strp(VerdictVulnerability), DecidedBy: strp("Alice"), DecidedAt: &at, PublishedAt: &at}
	short := Advisory{GHSAID: "GHSA-mmmm-mmmm-mmmm", IssueID: "iss-2610120000000005", Title: "Parser bug", Severity: "minor", ForgeState: "triage"}
	const siblingTitle = "a sample sibling report title"
	sibling := Advisory{GHSAID: "GHSA-hhhh-hhhh-hhhh", IssueID: "iss-2610120000000006", Title: siblingTitle, Severity: "minor", ForgeState: "triage"}
	for name, tc := range map[string]struct {
		desc    string
		leftOut bool
	}{
		"other id":        {"also see " + strings.ToLower(ghsaClosed), true},
		"other cve":       {"tracked with " + cveOpen, true},
		"other title":     {"related to " + strings.ToUpper(siblingTitle), true},
		"closed title":    {"related to " + strings.ToUpper(titleClosed), false},
		"same-plan id":    {"found with " + ghsaPublic, true},
		"own id":          {"this is " + ghsaOpen, false},
		"published id":    {"like " + ghsaPublished + ": " + titlePublishd, false},
		"short title":     {"a parser bug in the example", false},
		"unlisted public": {"upstream " + ghsaUnlisted, false},
	} {
		t.Run(name, func(t *testing.T) {
			pf := newPlanFixture(t)
			pf.seed(t, vulnAdvisory(ghsaOpen, issA), vulnAdvisory(ghsaPublic, issB), other, published, short, sibling)
			fix := pf.commit("parser.go", "package parser\n", "fix")
			pf.resolve(issA, fix)
			pf.resolve(issB, fix)
			pf.tag("v0.9.0")
			pf.remote(t, ghsaOpen, "draft", "line one\n"+tc.desc, vulnJSON(examplePackage, ""))
			pf.remote(t, ghsaPublic, "draft", "another report", vulnJSON(examplePackage, ""))
			p := pf.plan(t, "")
			if !tc.leftOut {
				if len(p.Entries) != 2 {
					t.Fatalf("plan %+v", p)
				}
				return
			}
			r := leftOutReason(t, p, ghsaOpen)
			if len(p.Entries) != 1 || p.Entries[0].GHSAID != ghsaPublic ||
				!strings.Contains(r, "another advisory still under embargo") || !strings.Contains(r, "description") || !strings.Contains(r, "line 2") {
				t.Fatalf("plan %+v", p)
			}
			low := strings.ToLower(r)
			for _, s := range []string{ghsaClosed, ghsaPublic, cveOpen, siblingTitle} {
				if strings.Contains(low, strings.ToLower(s)) {
					t.Fatalf("the reason repeats %q: %q", s, r)
				}
			}
		})
	}
}

// TestPlanSendsOnlyGets: building the plan reads, and writes nothing on
// GitHub or in the store.
func TestPlanSendsOnlyGets(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA), vulnAdvisory(ghsaPublic, issB))
	fix := pf.commit("parser.go", "package parser\n", "fix")
	pf.resolve(issA, fix)
	pf.resolve(issB, fix)
	pf.tag("v0.9.0")
	pf.remote(t, ghsaOpen, "draft", "the report", vulnJSON(examplePackage, ""))
	pf.remote(t, ghsaPublic, "triage", "another report", vulnJSON(examplePackage, ""))
	before := pf.s.StatePath()
	stateBefore := readFile(t, before)
	p := pf.plan(t, "")
	if len(p.Entries) != 2 || p.Entries[0].GHSAID > p.Entries[1].GHSAID {
		t.Fatalf("plan %+v", p)
	}
	noWrites(t, pf.g)
	if got := len(pf.g.calls()); got != 2 {
		t.Fatalf("%d calls, want one read per advisory", got)
	}
	if readFile(t, before) != stateBefore {
		t.Fatal("building the plan changed the store's state")
	}
}

// TestDigestStable: the same plan has the same digest; another release, or
// an advisory changed on GitHub, has another.
func TestDigestStable(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	fix := pf.commit("parser.go", "package parser\n", "fix")
	pf.tag("v0.9.0")
	pf.resolve(issA, fix)
	pf.tag("v0.9.1")
	pf.remote(t, ghsaOpen, "draft", "the report", vulnJSON(examplePackage, ""))
	a, b := pf.plan(t, ""), pf.plan(t, "")
	if a.Digest != b.Digest {
		t.Fatal("the same plan has two digests")
	}
	if c := pf.plan(t, "v0.9.1"); c.Digest != a.Digest {
		t.Fatal("naming the default release changed the digest")
	}
	pf.commit("other.go", "package other\n", "later")
	pf.tag("v0.9.2")
	if c := pf.plan(t, ""); c.Digest == a.Digest {
		t.Fatal("another release has the same digest")
	}
	pf.remote(t, ghsaOpen, "draft", "the report, edited", vulnJSON(examplePackage, ""))
	if c := pf.plan(t, "v0.9.1"); c.Digest == a.Digest {
		t.Fatal("an advisory edited on GitHub has the same digest")
	}
}

// TestPatchedVersionForm: the patched version is the tag without its v, the
// form this repository's published advisories carry.
func TestPatchedVersionForm(t *testing.T) {
	if patchedVersionKeepsV {
		t.Fatal("the pinned form keeps the v")
	}
	if got := patchedVersion("v0.9.0"); got != "0.9.0" {
		t.Fatalf("patchedVersion = %q", got)
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
