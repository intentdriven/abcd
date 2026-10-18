package embargo

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/termsafe"
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
	s, _, err := openStore(f.home, f.repo.Root())
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

// tag makes a published release: the tag in the checkout, pushed to the
// fixture's origin, and GitHub reporting its release published.
func (pf planFixture) tag(name string) {
	pf.localTag(name)
	pf.push(name)
	pf.release(name, releaseJSON(name, false, false))
}

// localTag makes the tag in the checkout alone.
func (pf planFixture) localTag(name string) { pf.f.repo.Git("tag", name) }

// originPath is the fixture's origin: a bare repository kept inside the
// checkout's .git, made on first use and named as origin's URL (for a
// fixture whose origin names GitHub, in place of that URL, which the plan
// does not read).
func (pf planFixture) originPath() string {
	bare := filepath.Join(pf.f.repo.Root(), ".git", "fixture-origin.git")
	if _, err := os.Stat(bare); err != nil {
		gittest.InitBare(pf.g.t, bare)
		pf.f.repo.Git("config", "remote.origin.url", bare)
	}
	return bare
}

// push pushes the tag to the fixture's origin.
func (pf planFixture) push(name string) {
	pf.originPath()
	pf.f.repo.Git("push", "-q", "origin", "refs/tags/"+name)
}

// releasePath is the API path of the GitHub release of tag.
func releasePath(tag string) string { return "repos/example/example/releases/tags/" + tag }

// release sets GitHub's answer for the release of tag.
func (pf planFixture) release(tag, body string) { pf.g.answer("GET", releasePath(tag), "", 0, body) }

// releaseJSON is a release as GitHub reports it.
func releaseJSON(tag string, draft, prerelease bool) string {
	b, _ := json.Marshal(map[string]any{"tag_name": tag, "draft": draft, "prerelease": prerelease, "name": "a sample release"})
	return string(b)
}

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
	p, err := planPublication(PublishRequest{Home: pf.f.home, RepoRoot: pf.f.repo.Root(), Release: release,
		Now: func() time.Time { return fixedNow.Add(time.Hour) }})
	if err != nil {
		t.Fatalf("planPublication: %v", err)
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
	_, err := planPublication(PublishRequest{Home: pf.f.home, RepoRoot: pf.f.repo.Root(), Release: "v0.9.0-rc.1"})
	mustRefuse(t, err, "vMAJOR.MINOR.PATCH")
	_, err = planPublication(PublishRequest{Home: pf.f.home, RepoRoot: pf.f.repo.Root(), Release: "v7.7.7"})
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
	// Every plan reads each held advisory's state (iss-2610171250479046),
	// and sends nothing else.
	noWrites(t, pf.g)
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
// other advisory, since publishing it would disclose the other one. A closed
// advisory stays private on GitHub, so its title is refused too. Naming
// itself, or an advisory already published, does not.
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
		"closed title":    {"related to " + strings.ToUpper(titleClosed), true},
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
			for _, s := range []string{ghsaClosed, ghsaPublic, cveOpen, siblingTitle, titleClosed} {
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
	calls := pf.g.calls()
	if len(calls) != 3 || !calls[0].has(releasePath("v0.9.0")) {
		t.Fatalf("%d calls, want one read of the release, then one per advisory", len(calls))
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

// storeResolve writes issID's private record into the store's own resolved/
// folder, this account's alone, naming fix as its resolved_by.commit (empty:
// no resolved_by), and returns its path inside the store. It writes the
// value as given, so a test can hold the plan to a store value capture's
// shape check would not have let through.
func (pf planFixture) storeResolve(t *testing.T, issID, fix string) string {
	t.Helper()
	rel := ".abcd/work/issues/resolved/" + issID + "-private-report.md"
	rec := "---\nschema_version: 1\nid: \"" + issID + "\"\nslug: \"private-report\"\nseverity: \"major\"\n" +
		"category: \"security\"\nsource: \"security-advisory\"\nresolution: \"fixed\"\nimpact: fix\n"
	if fix != "" {
		rec += "resolved_by:\n  commit: \"" + fix + "\"\n"
	}
	rec += "---\n\na sample private report.\n"
	p := filepath.Join(pf.s.Root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(rec), 0o600); err != nil {
		t.Fatal(err)
	}
	return rel
}

// TestStoreRecordedFixPlans: an advisory whose record is held out of every
// committed tree, as the security rule asks, and resolved in the store's own
// ledger through capture's resolve with its fix commit, is planned with the
// first release tag holding that commit as its patched version, though no
// release carries a public record of it.
func TestStoreRecordedFixPlans(t *testing.T) {
	pf := newPlanFixture(t)
	res, err := capture.Capture(capture.CaptureRequest{
		RepoRoot: pf.s.Root, Text: "a sample report title\n\nthe sample body", Severity: capture.SeverityMajor,
		Category: "security", Source: "security-advisory", Slug: "private-report",
		FoundDuring: "embargo intake 2026-10-12", Remedy: issueschema.MachineRemedy,
	})
	if err != nil {
		t.Fatal(err)
	}
	pf.seed(t, vulnAdvisory(ghsaOpen, res.ID))
	fix := pf.commit("parser.go", "package parser\n", "fix: tighten the parser")
	pf.tag("v0.9.0")
	pf.commit("other.go", "package other\n", "later")
	pf.tag("v0.10.0")
	moved, err := capture.Resolve(capture.ResolveRequest{RepoRoot: pf.s.Root, ID: res.ID, Resolution: "fixed by tightening the parser",
		Impact: "fix", ByCommit: fix})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(pf.s.Root, filepath.FromSlash(moved.Path)), 0o600); err != nil {
		t.Fatal(err)
	}
	if out := pf.f.repo.Git("ls-tree", "-r", "--name-only", "v0.10.0", "--", ".abcd/work/issues/resolved/"); out != "" {
		t.Fatalf("the release carries a public record: %q", out)
	}
	pf.remote(t, ghsaOpen, "draft", "the report", vulnJSON(examplePackage, ""))

	p := pf.plan(t, "")
	if p.Release != "v0.10.0" || len(p.Entries) != 1 || len(p.LeftOut) != 0 {
		t.Fatalf("plan %+v", p)
	}
	e := p.Entries[0]
	if e.GHSAID != ghsaOpen || e.IssueID != res.ID || e.FixCommit != fix || e.FirstTag != "v0.9.0" || e.PatchedVersion != "0.9.0" || !e.StoreEvidence {
		t.Fatalf("entry %+v", e)
	}
	if len(e.Vulnerabilities) != 1 || e.Vulnerabilities[0].VulnerableVersionRange != "< 0.9.0" || e.Vulnerabilities[0].PatchedVersions != "0.9.0" {
		t.Fatalf("vulnerabilities %+v", e.Vulnerabilities)
	}
	if p.ApproveWith == "" {
		t.Fatal("a plan holding an entry offers no approval")
	}
}

// TestStoreRecordedFixInNoTagLeftOut: a fix the store records that no release
// tag holds is left out, saying so, and GitHub is not asked about it.
func TestStoreRecordedFixInNoTagLeftOut(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	r := pf.f.repo
	pf.tag("v0.9.0")
	r.Git("switch", "-q", "-c", "side")
	fix := pf.commit("parser.go", "package parser\n", "fix on a branch no tag holds")
	r.Git("switch", "-q", "main")
	pf.storeResolve(t, issA, fix)
	p := pf.plan(t, "")
	if len(p.Entries) != 0 || !strings.Contains(leftOutReason(t, p, ghsaOpen), "no release tag") {
		t.Fatalf("plan %+v", p)
	}
	// Every plan reads each held advisory's state (iss-2610171250479046),
	// and sends nothing else.
	noWrites(t, pf.g)
}

// TestStoreRecordedFixMustBeFullCommitID: the store is a trust boundary, so
// the fix commit it names must be a full object name of this repository's
// hash, naming a commit itself: an abbreviation (which the public record's
// path accepts), an option-shaped value, another case, a name of the other
// hash's length, a name of no object here, or a tag object's name is left
// out with the reason and never reaches git as an option. A store record that
// is not resolved, or names no fix, leaves the advisory out as before.
func TestStoreRecordedFixMustBeFullCommitID(t *testing.T) {
	for name, tc := range map[string]struct {
		fix    func(pf planFixture, fix string) string
		reason string
	}{
		"abbreviated":    {func(_ planFixture, fix string) string { return fix[:12] }, "not a full commit id"},
		"option-shaped":  {func(pf planFixture, _ string) string { return "--output=" + pf.f.home + "/written" }, "not a full commit id"},
		"upper case":     {func(_ planFixture, fix string) string { return strings.ToUpper(fix) }, "not a full commit id"},
		"other length":   {func(_ planFixture, fix string) string { return fix + strings.Repeat("0", 64-len(fix)) }, "not a commit of this repository"},
		"no such object": {func(_ planFixture, _ string) string { return strings.Repeat("0", 40) }, "not a commit of this repository"},
		"tag object": {func(pf planFixture, _ string) string {
			pf.f.repo.Git("tag", "-a", "-m", "an annotated tag", "annotated")
			return pf.f.repo.Git("rev-parse", "refs/tags/annotated")
		}, "not a commit of this repository"},
		"no fix named": {func(_ planFixture, _ string) string { return "" }, "names no fix commit"},
	} {
		t.Run(name, func(t *testing.T) {
			pf := newPlanFixture(t)
			pf.seed(t, vulnAdvisory(ghsaOpen, issA))
			fix := pf.commit("parser.go", "package parser\n", "fix")
			pf.tag("v0.9.0")
			pf.storeResolve(t, issA, tc.fix(pf, fix))
			p := pf.plan(t, "")
			r := leftOutReason(t, p, ghsaOpen)
			if len(p.Entries) != 0 || !strings.Contains(r, tc.reason) || !strings.Contains(r, "not resolved in v0.9.0") {
				t.Fatalf("reason %q, plan %+v", r, p)
			}
			if _, err := os.Stat(filepath.Join(pf.f.home, "written")); err == nil {
				t.Fatal("a value from the store reached git as an option")
			}
			noWrites(t, pf.g)
		})
	}

	// A private record still open in the store is no evidence of a fix.
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	pf.tag("v0.9.0")
	p := pf.plan(t, "")
	if r := leftOutReason(t, p, ghsaOpen); len(p.Entries) != 0 || !strings.Contains(r, "not resolved in v0.9.0") || !strings.Contains(r, "store") {
		t.Fatalf("reason %q", r)
	}
}

// TestPublicRecordPathUnchanged: an advisory whose record is resolved in the
// release plans from that record exactly as before, whatever the store also
// records: its fix commit is the public record's, and the entry is not
// marked as taken from the store.
func TestPublicRecordPathUnchanged(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	fix := pf.commit("parser.go", "package parser\n", "fix")
	pf.tag("v0.9.0")
	other := pf.commit("other.go", "package other\n", "another fix")
	pf.resolve(issA, fix)
	pf.tag("v0.10.0")
	pf.storeResolve(t, issA, other)
	pf.remote(t, ghsaOpen, "draft", "the report", vulnJSON(examplePackage, ""))
	p := pf.plan(t, "")
	if len(p.Entries) != 1 {
		t.Fatalf("plan %+v", p)
	}
	if e := p.Entries[0]; e.FixCommit != fix || e.FirstTag != "v0.9.0" || e.StoreEvidence {
		t.Fatalf("entry %+v", e)
	}
	data, err := json.Marshal(p.Entries[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "store") {
		t.Fatalf("a public-record entry carries a store field, which would change its digest: %s", data)
	}
}

// storeRecordGuards are the ways a private record in the store's resolved/
// folder can fail the checks the plan holds it to before taking anything
// from it, each with the reason the plan gives. set lays the record (or
// records) down in place of storeResolve's.
var storeRecordGuards = map[string]struct {
	set    func(t *testing.T, pf planFixture, fix string)
	reason string
}{
	"another id": {func(t *testing.T, pf planFixture, fix string) {
		rel := pf.storeResolve(t, issA, fix)
		p := filepath.Join(pf.s.Root, filepath.FromSlash(rel))
		data := strings.Replace(readFile(t, p), `id: "`+issA+`"`, `id: "`+issB+`"`, 1)
		if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}, "carries another id"},
	"symlinked record": {func(t *testing.T, pf planFixture, fix string) {
		rel := pf.storeResolve(t, issA, fix)
		p := filepath.Join(pf.s.Root, filepath.FromSlash(rel))
		elsewhere := filepath.Join(t.TempDir(), "record.md")
		if err := os.Rename(p, elsewhere); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(elsewhere, p); err != nil {
			t.Fatal(err)
		}
	}, "could not be read as a regular file within its size limit"},
	"over the cap": {func(t *testing.T, pf planFixture, fix string) {
		rel := pf.storeResolve(t, issA, fix)
		p := filepath.Join(pf.s.Root, filepath.FromSlash(rel))
		data := readFile(t, p) + strings.Repeat("x", maxRecordBytes)
		if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}, "could not be read as a regular file within its size limit"},
	"held twice": {func(t *testing.T, pf planFixture, fix string) {
		rel := pf.storeResolve(t, issA, fix)
		p := filepath.Join(pf.s.Root, filepath.FromSlash(rel))
		twin := filepath.Join(filepath.Dir(p), issA+"-another-report.md")
		if err := os.WriteFile(twin, []byte(readFile(t, p)), 0o600); err != nil {
			t.Fatal(err)
		}
	}, "holds its private record twice"},
}

// TestStoreRecordGuardsLeaveOut: a private record in the store that carries
// another advisory's id, is a link, is over its cap, or is held twice in
// resolved/ is no evidence of a fix: the advisory is left out with the
// reason, and nothing is taken from the record.
func TestStoreRecordGuardsLeaveOut(t *testing.T) {
	for name, tc := range storeRecordGuards {
		t.Run(name, func(t *testing.T) {
			pf := newPlanFixture(t)
			pf.seed(t, vulnAdvisory(ghsaOpen, issA))
			fix := pf.commit("parser.go", "package parser\n", "fix")
			pf.tag("v0.9.0")
			tc.set(t, pf, fix)
			pf.remote(t, ghsaOpen, "draft", "the report", vulnJSON(examplePackage, ""))
			p := pf.plan(t, "")
			if r := leftOutReason(t, p, ghsaOpen); len(p.Entries) != 0 || !strings.Contains(r, tc.reason) || !strings.Contains(r, "not resolved in v0.9.0") {
				t.Fatalf("reason %q, plan %+v", r, p)
			}
		})
	}
}

// TestPublishedStoreRecordGuardsNameNoRestore: an advisory GitHub reports
// published whose private record in the store fails the same checks names
// no restore command: copying a link, or another advisory's record, into
// the public ledger is never offered.
func TestPublishedStoreRecordGuardsNameNoRestore(t *testing.T) {
	for name, tc := range storeRecordGuards {
		t.Run(name, func(t *testing.T) {
			pf := newPlanFixture(t)
			pf.seed(t, vulnAdvisory(ghsaOpen, issA))
			fix := pf.commit("parser.go", "package parser\n", "fix")
			pf.tag("v0.9.0")
			tc.set(t, pf, fix)
			a := advisoryJSON(ghsaOpen, "published", "high", "a sample summary", "the report")
			a["published_at"] = endedAt.Format(time.RFC3339)
			b, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			pf.g.answer("GET", listPath+"/"+ghsaOpen, "", 0, string(b))
			p := pf.plan(t, "")
			if len(p.Restores) != 1 || p.Restores[0].RestoreWith != "" || !strings.Contains(p.Restores[0].Reason, tc.reason) {
				t.Fatalf("restores %+v; want none named, saying %q", p.Restores, tc.reason)
			}
		})
	}
}

// TestRestoreCommandQuoting: the restore command names the store's record
// under ~ whenever the store sits under home, in home's own spelling or
// through a link to it, with the rest single-quoted when a shell would not
// read it as written; one outside home is quoted whole and left whole by
// the surface's home redaction. A name holding a character the terminal
// would not show as written names no command, saying why. Each command,
// run by a shell with that home, names exactly the record.
func TestRestoreCommandQuoting(t *testing.T) {
	const key = "0123456789abcdef0123456789abcdef01234567"
	storeUnder := func(home string) string {
		return abcdhome.Path(home, "worktrees", key, "embargo")
	}
	for name, tc := range map[string]struct {
		setup func(t *testing.T) (home, root string)
		file  string
		want  string // "~": the ~/ form; "abs": quoted whole; "": none
	}{
		"plain":         {func(t *testing.T) (string, string) { h := t.TempDir(); return h, storeUnder(h) }, "iss-1-private-report.md", "~"},
		"space":         {func(t *testing.T) (string, string) { h := t.TempDir(); return h, storeUnder(h) }, "iss-1-a report.md", "~"},
		"single quote":  {func(t *testing.T) (string, string) { h := t.TempDir(); return h, storeUnder(h) }, "iss-1-it's.md", "~"},
		"dollar":        {func(t *testing.T) (string, string) { h := t.TempDir(); return h, storeUnder(h) }, "iss-1-$HOME.md", "~"},
		"newline":       {func(t *testing.T) (string, string) { h := t.TempDir(); return h, storeUnder(h) }, "iss-1-a\nb.md", ""},
		"bidi":          {func(t *testing.T) (string, string) { h := t.TempDir(); return h, storeUnder(h) }, "iss-1-a‮b.md", ""},
		"control":       {func(t *testing.T) (string, string) { h := t.TempDir(); return h, storeUnder(h) }, "iss-1-a\x1bb.md", ""},
		"outside home":  {func(t *testing.T) (string, string) { return t.TempDir(), storeUnder(t.TempDir()) }, "iss-1-private-report.md", "abs"},
		"outside quote": {func(t *testing.T) (string, string) { return t.TempDir(), storeUnder(t.TempDir()) }, "iss-1-it's.md", "abs"},
		"symlinked home": {func(t *testing.T) (string, string) {
			real := t.TempDir()
			link := filepath.Join(t.TempDir(), "home")
			if err := os.Symlink(real, link); err != nil {
				t.Fatal(err)
			}
			return link, storeUnder(real)
		}, "iss-1-private-report.md", "~"},
		"symlinked abcd home": {func(t *testing.T) (string, string) {
			h, elsewhere := t.TempDir(), t.TempDir()
			if err := os.Symlink(elsewhere, abcdhome.Path(h)); err != nil {
				t.Fatal(err)
			}
			return h, storeUnder(h)
		}, "iss-1-private-report.md", "~"},
	} {
		t.Run(name, func(t *testing.T) {
			home, root := tc.setup(t)
			t.Setenv("HOME", home)
			dir := filepath.Join(root, ".abcd", "work", "issues", "resolved")
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			src := filepath.Join(dir, tc.file)
			if err := os.WriteFile(src, []byte("a record\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd, why := restoreCommand(home, Store{Root: root}, resolvedRel+"/"+tc.file)
			if tc.want == "" {
				if cmd != "" || !strings.Contains(why, "terminal") {
					t.Fatalf("command %q, why %q; want none, saying why", cmd, why)
				}
				return
			}
			const dst = " .abcd/work/issues/resolved/"
			shown, ok := strings.CutPrefix(cmd, "cp -- ")
			if !ok || why != "" || !strings.HasSuffix(shown, dst) {
				t.Fatalf("command %q, why %q", cmd, why)
			}
			shown = strings.TrimSuffix(shown, dst)
			switch tc.want {
			case "~":
				if !strings.HasPrefix(shown, "~/") {
					t.Fatalf("command %q does not name the record under ~", cmd)
				}
			case "abs":
				if !strings.HasPrefix(shown, "'/") {
					t.Fatalf("command %q does not quote the record's path whole", cmd)
				}
			}
			if strings.Contains(cmd, home) || strings.Contains(cmd, storepath.Resolve(home)) {
				t.Fatalf("command %q names the home directory", cmd)
			}
			if fsutil.RedactHome(cmd) != cmd {
				t.Fatalf("home redaction changes the command %q to %q", cmd, fsutil.RedactHome(cmd))
			}
			if termsafe.Sanitize(cmd) != cmd {
				t.Fatalf("the terminal would not show the command as written: %q", cmd)
			}
			sh := exec.Command("sh", "-c", "printf '%s' "+shown)
			sh.Env = append(os.Environ(), "HOME="+home)
			out, err := sh.Output()
			if err != nil {
				t.Fatal(err)
			}
			if storepath.Resolve(string(out)) != storepath.Resolve(src) {
				t.Fatalf("a shell reads %q as %q, not the record %q", shown, out, src)
			}
		})
	}
}
