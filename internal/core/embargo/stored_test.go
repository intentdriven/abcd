package embargo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeGraft writes lines as the checkout's .git/info/grafts file.
func writeGraft(t *testing.T, root string, lines ...string) {
	t.Helper()
	info := filepath.Join(root, ".git", "info")
	if err := os.MkdirAll(info, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(info, "grafts"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestCheckReadsTheRangeAsStoredUnderAReplaceRef: a refs/replace entry that
// stands a commit with its parent's tree and a different message in for the
// range's tip changes nothing the check reads. The check reads the tip's
// message and patch as stored, which is what a push sends.
func TestCheckReadsTheRangeAsStoredUnderAReplaceRef(t *testing.T) {
	cf := newCheckFixture(t)
	r := cf.f.repo
	base := r.Git("rev-parse", "HEAD")
	r.Write("notes.txt", "mentions "+ghsaOpen+"\n")
	r.Commit("fixes " + ghsaOpen)
	tip := r.Git("rev-parse", "HEAD")
	standIn := r.Git("commit-tree", base+"^{tree}", "-p", base, "-m", "tidy the notes")
	r.Git("replace", tip, standIn)
	if got := r.Git("log", "-1", "--format=%s", tip); got != "tidy the notes" {
		t.Fatalf("the replace ref does not stand in: %s reads as %q", tip, got)
	}

	res, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: r.Root(), Range: base + "..HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	wantKinds(t, res, KindAdvisoryID, KindAdvisoryID)
	var where []string
	for _, f := range res.Findings {
		if f.Source != "commit "+tip {
			t.Fatalf("finding names %q, want the tip %s: %+v", f.Source, tip, f)
		}
		where = append(where, f.Where)
	}
	if got := strings.Join(where, ","); !strings.Contains(got, "message") || !strings.Contains(got, "added line in notes.txt") {
		t.Fatalf("findings read %q, want the message and the added line", got)
	}
}

// TestCheckReadsTheRangeAsStoredUnderAGraft: a .git/info/grafts line that
// gives the range's tip a second parent holding the tip's own tree makes the
// tip look like a merge that adds nothing against that parent. The check
// reads the tip's stored parents, so it reads the tip as the single-parent
// commit it is and sees the line it adds.
func TestCheckReadsTheRangeAsStoredUnderAGraft(t *testing.T) {
	cf := newCheckFixture(t)
	r := cf.f.repo
	parent := r.Git("rev-parse", "HEAD")
	r.Write("notes.txt", "mentions "+ghsaOpen+"\n")
	r.Commit("add notes")
	tip := r.Git("rev-parse", "HEAD")
	cover := r.Git("commit-tree", tip+"^{tree}", "-p", parent, "-m", "a commit with the tip's tree")
	writeGraft(t, r.Root(), tip+" "+parent+" "+cover)
	if got := r.Git("-c", "advice.graftFileDeprecated=false", "log", "-1", "--format=%P", tip); got != parent+" "+cover {
		t.Fatalf("the graft does not stand in: %s's parents read as %q", tip, got)
	}

	res, err := Check(CheckRequest{Home: cf.f.home, RepoRoot: r.Root(), Range: cover + ".." + tip})
	if err != nil {
		t.Fatal(err)
	}
	wantKinds(t, res, KindAdvisoryID)
	if f := res.Findings[0]; f.Source != "commit "+tip || f.Where != "added line in notes.txt" {
		t.Fatalf("finding %+v, want the tip's added line", f)
	}
}

// TestFirstReleaseTagReadsHistoryAsStoredUnderAGraft: a graft that gives an
// earlier release's commit the fix as a parent does not make that release
// the first to hold the fix. Tag containment is read from the commits as
// stored.
func TestFirstReleaseTagReadsHistoryAsStoredUnderAGraft(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	r := pf.f.repo
	pf.commit("README.md", "a project\n", "start")
	pf.tag("v1.0.0")
	start := r.Git("rev-parse", "HEAD")
	r.Git("switch", "-q", "-c", "fix")
	fix := pf.commit("parser.go", "package parser\n", "fix: tighten the parser")
	r.Git("switch", "-q", "main")
	earlier := pf.commit("docs.txt", "a page\n", "docs: add a page")
	pf.tag("v1.1.0")
	r.Git("merge", "-q", "--no-ff", "-m", "merge the fix", "fix")
	pf.resolve(issA, fix)
	pf.tag("v1.2.0")
	pf.remote(t, ghsaOpen, "draft", "the report", vulnJSON(examplePackage, ""))
	writeGraft(t, r.Root(), earlier+" "+start+" "+fix)
	if got := r.Git("-c", "advice.graftFileDeprecated=false", "tag", "--contains", fix, "--list", "v*"); !strings.Contains(got, "v1.1.0") {
		t.Fatalf("the graft does not stand in: the tags holding the fix read as %q", got)
	}

	p := pf.plan(t, "v1.2.0")
	if len(p.Entries) != 1 {
		t.Fatalf("plan %+v", p)
	}
	if e := p.Entries[0]; e.FixCommit != fix || e.FirstTag != "v1.2.0" || e.PatchedVersion != "1.2.0" {
		t.Fatalf("entry %+v, want the fix first released in v1.2.0", e)
	}
}

// TestFixCommitReadsTheRecordAsStoredUnderAReplaceRef: a refs/replace entry
// that stands another blob in for the release's resolved record, naming an
// older commit as the fix, changes neither the fix commit the plan reads nor
// the first release it names.
func TestFixCommitReadsTheRecordAsStoredUnderAReplaceRef(t *testing.T) {
	pf := newPlanFixture(t)
	pf.seed(t, vulnAdvisory(ghsaOpen, issA))
	r := pf.f.repo
	older := pf.commit("README.md", "a project\n", "start")
	pf.tag("v1.0.0")
	fix := pf.commit("parser.go", "package parser\n", "fix: tighten the parser")
	pf.resolve(issA, fix)
	pf.tag("v1.1.0")
	pf.remote(t, ghsaOpen, "draft", "the report", vulnJSON(examplePackage, ""))
	rec := ".abcd/work/issues/resolved/" + issA + "-private-report.md"
	stored := r.Git("rev-parse", "v1.1.0:"+rec)
	body := strings.Replace(readFile(t, filepath.Join(r.Root(), rec)), fix, older, 1)
	standIn := filepath.Join(t.TempDir(), "record.md")
	if err := os.WriteFile(standIn, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	r.Git("replace", stored, r.Git("hash-object", "-w", standIn))
	if got := r.Git("cat-file", "blob", "v1.1.0:"+rec); !strings.Contains(got, older) {
		t.Fatalf("the replace ref does not stand in: the record reads %q", got)
	}

	p := pf.plan(t, "v1.1.0")
	if len(p.Entries) != 1 {
		t.Fatalf("plan %+v", p)
	}
	if e := p.Entries[0]; e.FixCommit != fix || e.FirstTag != "v1.1.0" {
		t.Fatalf("entry %+v, want the stored fix %s first released in v1.1.0", e, fix)
	}
}
