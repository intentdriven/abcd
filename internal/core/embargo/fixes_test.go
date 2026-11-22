package embargo

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

// fixes_test.go: the fix commits recorded on an advisory under embargo, and
// the bare status that lists each advisory whose recorded fix is in a
// published release as ready to publish. Origin is a bare repository standing
// in for the remote; GitHub is the fake gh.

func (pf planFixture) recordFix(t *testing.T, target string, replace bool, commits ...string) FixResult {
	t.Helper()
	res, err := RecordFix(FixRequest{Home: pf.f.home, RepoRoot: pf.f.repo.Root(), Target: target, Commits: commits, Replace: replace})
	if err != nil {
		t.Fatalf("RecordFix: %v", err)
	}
	return res
}

func (pf planFixture) status(t *testing.T) StatusResult {
	t.Helper()
	res, err := Status(StatusRequest{Home: pf.f.home, RepoRoot: pf.f.repo.Root()})
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	return res
}

func (pf planFixture) stateBytes(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(pf.s.StatePath())
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestRecordFixWritesOnlyTheStore: a fix commit named by an abbreviation is
// kept in the store's state by its full id, and nothing in the checkout, its
// public ledger included, changes.
func TestRecordFixWritesOnlyTheStore(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	fix := pf.commit("parser.go", "package parser\n", "fix: tighten the parser")

	res := pf.recordFix(t, issA, false, fix[:10])
	if res.GHSAID != ghsaOpen || res.IssueID != issA || !slices.Equal(res.FixCommits, []string{fix}) || !slices.Equal(res.Added, []string{fix}) {
		t.Fatalf("result %+v", res)
	}
	st, err := pf.s.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Advisories[0].FixCommits; !slices.Equal(got, []string{fix}) {
		t.Fatalf("state's fix commits %v, want [%s]", got, fix)
	}
	if out := pf.f.repo.Git("status", "--porcelain", "--untracked-files=all"); out != "" {
		t.Fatalf("the checkout changed:\n%s", out)
	}
}

// TestRecordFixAppendsOnceAndReplaces: a second call adds only commits not
// already recorded, in the order named; --replace swaps the whole list.
func TestRecordFixAppendsOnceAndReplaces(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	one := pf.commit("a.go", "package a\n", "fix: one")
	two := pf.commit("b.go", "package b\n", "fix: two")

	pf.recordFix(t, ghsaOpen, false, one)
	res := pf.recordFix(t, issA, false, one, two, two)
	if !slices.Equal(res.FixCommits, []string{one, two}) || !slices.Equal(res.Added, []string{two}) {
		t.Fatalf("append: %+v", res)
	}
	res = pf.recordFix(t, issA, true, two)
	if !slices.Equal(res.FixCommits, []string{two}) {
		t.Fatalf("replace: %+v", res)
	}
}

// TestRecordFixRefusesWritingNothing: each refusal leaves the state as it was.
func TestRecordFixRefusesWritingNothing(t *testing.T) {
	untriaged := vulnAdvisory(ghsaOpen, issA)
	untriaged.Verdict, untriaged.DecidedBy, untriaged.DecidedAt = nil, nil, nil
	closed := vulnAdvisory(ghsaOpen, issA)
	at := fixedNow
	closed.ClosedAt = &at
	notVuln := vulnAdvisory(ghsaOpen, issA)
	notVuln.Verdict = strp(VerdictNotAVulnerability)

	for name, c := range map[string]struct {
		a       Advisory
		target  string
		commits func(fix string) []string
		want    string
	}{
		"no commit":         {vulnAdvisory(ghsaOpen, issA), issA, func(string) []string { return nil }, "at least one fix commit"},
		"unknown advisory":  {vulnAdvisory(ghsaOpen, issA), issB, func(f string) []string { return []string{f} }, "no advisory by that id"},
		"untriaged":         {untriaged, issA, func(f string) []string { return []string{f} }, "judged a vulnerability"},
		"not vulnerability": {notVuln, issA, func(f string) []string { return []string{f} }, "judged a vulnerability"},
		"closed":            {closed, issA, func(f string) []string { return []string{f} }, "closed or published"},
		"option-shaped":     {vulnAdvisory(ghsaOpen, issA), issA, func(string) []string { return []string{"--output=x"} }, "not a commit id"},
		"a ref name":        {vulnAdvisory(ghsaOpen, issA), issA, func(string) []string { return []string{"HEAD"} }, "not a commit id"},
		"absent commit":     {vulnAdvisory(ghsaOpen, issA), issA, func(string) []string { return []string{strings.Repeat("ab", 20)} }, "not a commit of this repository"},
		"a tree's id": {vulnAdvisory(ghsaOpen, issA), issA, func(string) []string {
			return []string{"4b825dc642cb6eb9a060e54bf8d69288fbee4904"} // the empty tree
		}, "not a commit of this repository"},
	} {
		t.Run(name, func(t *testing.T) {
			pf := newPlanFixture(t)
			pf.seed(t, c.a)
			fix := pf.commit("parser.go", "package parser\n", "fix: tighten the parser")
			before := pf.stateBytes(t)
			_, err := RecordFix(FixRequest{Home: pf.f.home, RepoRoot: pf.f.repo.Root(), Target: c.target, Commits: c.commits(fix)})
			var r *RefusalError
			if !errors.As(err, &r) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err %v, want a refusal naming %q", err, c.want)
			}
			if string(pf.stateBytes(t)) != string(before) {
				t.Fatal("a refusal wrote the state")
			}
		})
	}
}

// TestStateRefusesAMalformedFixCommit: a state whose fix commit is not a full
// object name, or is listed twice, is refused as no state abcd writes.
func TestStateRefusesAMalformedFixCommit(t *testing.T) {
	full := strings.Repeat("ab", 20)
	for name, fixes := range map[string][]string{
		"abbreviated": {"abcdef1"},
		"twice":       {full, full},
		"upper case":  {strings.ToUpper(full)},
	} {
		st := emptyState()
		a := vulnAdvisory(ghsaOpen, issA)
		a.FixCommits = fixes
		st.Advisories = []Advisory{a}
		if err := st.validate(); err == nil {
			t.Errorf("%s: a state with fix commits %v validated", name, fixes)
		}
	}
}

// TestStatusListsReadyWhenTheFixIsInAPublishedRelease: an advisory whose
// recorded fix commit is in a published release is ready to publish, naming
// the first such release and the command that shows its plan.
func TestStatusListsReadyWhenTheFixIsInAPublishedRelease(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA), vulnAdvisory(ghsaPublished, issB))
	fix := pf.commit("parser.go", "package parser\n", "fix: tighten the parser")
	pf.tag("v0.9.0")
	pf.commit("docs.txt", "a page\n", "docs: add a page")
	pf.tag("v0.10.0")
	pf.recordFix(t, issA, false, fix)

	res := pf.status(t)
	if res.NoStore || len(res.Ready) != 1 || len(res.Waiting) != 0 {
		t.Fatalf("status %+v", res)
	}
	r := res.Ready[0]
	if r.GHSAID != ghsaOpen || r.IssueID != issA || r.Release != "v0.9.0" || !slices.Equal(r.FixCommits, []string{fix}) ||
		r.PublishWith != "abcd embargo publish --release v0.9.0" {
		t.Fatalf("ready %+v", r)
	}
	if !slices.ContainsFunc(res.NoFix, func(a HeldAdvisory) bool { return a.IssueID == issB }) {
		t.Fatalf("no-fix list %+v, want %s", res.NoFix, issB)
	}
	for _, c := range pf.g.writes() {
		t.Fatalf("status wrote to GitHub: %v", c)
	}
}

// TestStatusWaitsUntilTheReleaseIsPublished: a fix in no release tag, or only
// in a tag that is not a published release, waits, saying why.
func TestStatusWaitsUntilTheReleaseIsPublished(t *testing.T) {
	for name, c := range map[string]struct {
		make func(pf planFixture)
		want string
	}{
		"no tag":     {func(planFixture) {}, "no release tag yet"},
		"local only": {func(pf planFixture) { pf.localTag("v0.9.0") }, "no published release"},
		"draft": {func(pf planFixture) {
			pf.localTag("v0.9.0")
			pf.push("v0.9.0")
			pf.release("v0.9.0", releaseJSON("v0.9.0", true, false))
		}, "no published release"},
	} {
		t.Run(name, func(t *testing.T) {
			pf := newPlanFixture(t)
			pf.seed(t, vulnAdvisory(ghsaOpen, issA))
			fix := pf.commit("parser.go", "package parser\n", "fix: tighten the parser")
			pf.originPath()
			c.make(pf)
			pf.recordFix(t, issA, false, fix)
			res := pf.status(t)
			if len(res.Ready) != 0 || len(res.Waiting) != 1 || !strings.Contains(res.Waiting[0].Reason, c.want) {
				t.Fatalf("status %+v, want one waiting naming %q", res, c.want)
			}
		})
	}
}

// TestStatusNeedsEveryFixCommitInTheRelease: with two fix commits, the release
// named is the first published one holding both.
func TestStatusNeedsEveryFixCommitInTheRelease(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	one := pf.commit("a.go", "package a\n", "fix: one")
	pf.tag("v0.8.0")
	two := pf.commit("b.go", "package b\n", "fix: two")
	pf.tag("v0.9.0")
	pf.recordFix(t, issA, false, one, two)

	res := pf.status(t)
	if len(res.Ready) != 1 || res.Ready[0].Release != "v0.9.0" {
		t.Fatalf("status %+v, want ready in v0.9.0", res)
	}
}

// TestStatusTakesTheStoresResolvedRecord: with no fix commit recorded, the
// fix commit the private record resolved in the store names is the evidence,
// as the publication plan takes it.
func TestStatusTakesTheStoresResolvedRecord(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	fix := pf.commit("parser.go", "package parser\n", "fix: tighten the parser")
	pf.tag("v0.9.0")
	pf.storeResolve(t, issA, fix)

	res := pf.status(t)
	if len(res.Ready) != 1 || res.Ready[0].Release != "v0.9.0" || !slices.Equal(res.Ready[0].FixCommits, []string{fix}) {
		t.Fatalf("status %+v", res)
	}
}

// TestStatusReadsHistoryAsStoredUnderAGraft: a graft that gives an earlier
// release's commit the fix as a parent does not make that release hold the
// fix.
func TestStatusReadsHistoryAsStoredUnderAGraft(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	r := pf.f.repo
	pf.commit("README.md", "a project\n", "start")
	start := r.Git("rev-parse", "HEAD")
	r.Git("switch", "-q", "-c", "fix")
	fix := pf.commit("parser.go", "package parser\n", "fix: tighten the parser")
	r.Git("switch", "-q", "main")
	earlier := pf.commit("docs.txt", "a page\n", "docs: add a page")
	pf.tag("v1.1.0")
	pf.recordFix(t, issA, false, fix)
	writeGraft(t, r.Root(), earlier+" "+start+" "+fix)

	res := pf.status(t)
	if len(res.Ready) != 0 || len(res.Waiting) != 1 || !strings.Contains(res.Waiting[0].Reason, "no release tag yet") {
		t.Fatalf("status %+v, want the fix in no release tag as stored", res)
	}
}

// TestStatusWithoutAStore: a checkout with no embargo store reports so and
// makes none.
func TestStatusWithoutAStore(t *testing.T) {
	f := newFixture(t)
	res, err := Status(StatusRequest{Home: f.home, RepoRoot: f.repo.Root()})
	if err != nil || !res.NoStore {
		t.Fatalf("status %+v, err %v", res, err)
	}
	loc, err := locate(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(loc.Path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("status made a store: %v", err)
	}
}

// TestPlanTakesRecordedFixCommits: an advisory with no record resolved in the
// release or in the store, whose fix commits are recorded, is planned from
// them, the patched version given by the first published release holding
// every one.
func TestPlanTakesRecordedFixCommits(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	one := pf.commit("a.go", "package a\n", "fix: one")
	pf.tag("v0.8.0")
	two := pf.commit("b.go", "package b\n", "fix: two")
	pf.tag("v0.9.0")
	pf.recordFix(t, issA, false, one, two)
	pf.remote(t, ghsaOpen, "draft", "the report", vulnJSON(examplePackage, ""))

	p := pf.plan(t, "")
	if len(p.Entries) != 1 {
		t.Fatalf("plan %+v", p)
	}
	e := p.Entries[0]
	if e.FirstTag != "v0.9.0" || e.PatchedVersion != "0.9.0" || !e.StoreEvidence || !slices.Equal(e.FixCommits, []string{one, two}) || e.FixCommit != one {
		t.Fatalf("entry %+v", e)
	}
}
