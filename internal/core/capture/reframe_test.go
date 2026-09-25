package capture

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The reframe record (itd-2609020625402518, spc-2609020626048705).

const (
	fxFraming  = ".abcd/development/brief/01-product/06-framing.md"
	fxScope    = ".abcd/development/brief/01-product/04-scope.md"
	fxGlossary = ".abcd/development/brief/glossary"

	fxItem        = "rdi-11"
	fxDisposition = "dsp-5"
	fxSurprise    = "srp-7"

	reframeGround = "the detection reading showed the construal treated the ledger as the product rather than as the record"

	// A phrase that exists only in the ABANDONED construal. The record must
	// never carry it: its text stays on the local side (adr-55).
	fxOldPhrase = "the repository is a filing cabinet"
)

// framingDoc renders a framing chapter whose Construal section states c. The
// section runs to the next H2, so the H3 under it is part of it and the H2
// after it is not.
func framingDoc(c string) string {
	return "---\nstatus: current\n---\n# Framing\n\nWhat the frame is for.\n\n## Construal\n\n" + c +
		"\n\n### How it is held\n\nA subsection of the construal.\n\n## Consequences\n\nNot the construal.\n"
}

func scopeDoc(s string) string { return "# Scope\n\n" + s + "\n" }

// reframeFixture lays out a repository carrying the three frame surfaces and
// one occasion of each family, all committed, and returns it.
func reframeFixture(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write(fxFraming, framingDoc("We are treating this as a gap: "+fxOldPhrase+"."))
	r.Write(fxScope, scopeDoc("The scope is the eight phases."))
	r.Write(fxGlossary+"/README.md", "# Glossary\n\n- [term](core/term.md)\n")
	r.Write(fxGlossary+"/_template.md", "# Template\n")
	r.Write(fxGlossary+"/core/README.md", "# Core\n")
	r.Write(fxGlossary+"/core/term.md", "# Term\n\nA term.\n")
	r.Write(".abcd/work/issues/readings/rdg-1/"+fxItem+".md", "---\nid: rdi-11\n---\n")
	r.Write(".abcd/work/issues/dispositions/"+fxItem+"/"+fxDisposition+".md", "---\nid: dsp-5\n---\n")
	r.Write(".abcd/work/issues/surprises/"+fxSurprise+".md", "---\nid: srp-7\n---\n")
	r.Commit("base: the frame and its occasions")
	return r
}

// rewriteConstrual commits a new construal.
func rewriteConstrual(r *gittest.Repo, c string) {
	r.Write(fxFraming, framingDoc(c))
	r.Commit("rewrite the construal")
}

func reframeReq(r *gittest.Repo, occasion string) ReframeRequest {
	return ReframeRequest{RepoRoot: r.Root(), OccasionedBy: occasion, Grounds: reframeGround}
}

// readReframe parses a written reframe record.
func readReframe(t *testing.T, r *gittest.Repo, rel string) (map[string]any, string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(r.Root(), filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	fm, _, err := parseFrontmatterAndBody(string(raw))
	if err != nil {
		t.Fatalf("parse %s: %v\n%s", rel, err, raw)
	}
	return fm, string(raw)
}

// frameAt fingerprints the three surfaces as the repository holds them in the
// working tree, through the exported fingerprint functions alone — the test's
// independent derivation of what the verb must have written.
func frameAt(t *testing.T, r *gittest.Repo) Frame {
	t.Helper()
	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(r.Root(), filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	c, err := ConstrualFingerprint(read(fxFraming))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	_ = filepath.Walk(filepath.Join(r.Root(), filepath.FromSlash(fxGlossary)), func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(r.Root(), p)
		b, _ := os.ReadFile(p)
		files[filepath.ToSlash(rel)] = b
		return nil
	})
	return Frame{Construal: c, Glossary: GlossaryFingerprint(files), Scope: ScopeFingerprint(read(fxScope))}
}

func reframeFiles(t *testing.T, r *gittest.Repo) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(r.Root(), ".abcd", "work", "issues", issueschema.ReframesDir))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func ledgerOf(r *gittest.Repo) string { return filepath.Join(r.Root(), ".abcd", "work", "issues") }

// --- the fingerprints ---

func TestConstrualFingerprintIsStableAcrossLineEndingsAndBlankEdges(t *testing.T) {
	lf := framingDoc("We are treating this as a gap.")
	base, err := ConstrualFingerprint(lf)
	if err != nil {
		t.Fatal(err)
	}
	if !issueschema.ValidFingerprint(base) {
		t.Fatalf("fingerprint %q is not a 64-hex SHA-256", base)
	}
	for name, doc := range map[string]string{
		"CRLF":               strings.ReplaceAll(lf, "\n", "\r\n"),
		"blank lines around": strings.Replace(lf, "## Construal\n\n", "## Construal\n\n\n\n", 1),
		"another section":    strings.Replace(lf, "Not the construal.", "Something else entirely.", 1),
		"no frontmatter":     strings.TrimPrefix(lf, "---\nstatus: current\n---\n"),
	} {
		got, err := ConstrualFingerprint(doc)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got != base {
			t.Errorf("%s moved the construal fingerprint", name)
		}
	}
	// The subsection is part of the section, and so is the not-yet-real marker.
	for name, doc := range map[string]string{
		"subsection": strings.Replace(lf, "A subsection of the construal.", "A rewritten subsection.", 1),
		"statement":  framingDoc("We are treating this as something else."),
		"marker":     framingDoc("> **Status: NOT YET REAL.**\n\nWe are treating this as a gap."),
	} {
		got, err := ConstrualFingerprint(doc)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got == base {
			t.Errorf("a change to the %s did not move the construal fingerprint", name)
		}
	}
}

func TestConstrualFingerprintRefusesAChapterWithoutTheSection(t *testing.T) {
	if _, err := ConstrualFingerprint("# Framing\n\n## Consequences\n\ntext\n"); err == nil || !strings.Contains(err.Error(), "Construal") {
		t.Errorf("no Construal section: err = %v, want a refusal naming it", err)
	}
	two := framingDoc("one") + "\n## Construal\n\ntwo\n"
	if _, err := ConstrualFingerprint(two); err == nil || !strings.Contains(err.Error(), "2 H2 sections titled") {
		t.Errorf("two Construal sections: err = %v, want a refusal naming the count", err)
	}
	// A Construal heading inside a fence is not a heading.
	fenced := "# Framing\n\n```\n## Construal\n```\n"
	if _, err := ConstrualFingerprint(fenced); err == nil {
		t.Error("a fenced Construal heading was read as the section")
	}
	// An H3 titled Construal is not the section either.
	if _, err := ConstrualFingerprint("# Framing\n\n### Construal\n\ntext\n"); err == nil {
		t.Error("an H3 Construal was read as the H2 section")
	}
}

func TestGlossaryFingerprintIgnoresTheIndexAndMovesOnATerm(t *testing.T) {
	files := map[string][]byte{
		fxGlossary + "/README.md":      []byte("# Glossary\n"),
		fxGlossary + "/_template.md":   []byte("# Template\n"),
		fxGlossary + "/core/README.md": []byte("# Core\n"),
		fxGlossary + "/core/term.md":   []byte("# Term\n\nA term.\n"),
	}
	base := GlossaryFingerprint(files)
	if !issueschema.ValidFingerprint(base) {
		t.Fatalf("fingerprint %q is not a 64-hex SHA-256", base)
	}
	clone := func() map[string][]byte {
		out := map[string][]byte{}
		for k, v := range files {
			out[k] = v
		}
		return out
	}
	same := clone()
	same[fxGlossary+"/README.md"] = []byte("# Glossary, regenerated\n\n- a new index line\n")
	same[fxGlossary+"/core/README.md"] = []byte("# Core, regenerated\n")
	same[fxGlossary+"/_template.md"] = []byte("# Template v2\n")
	same[fxGlossary+"/core/term.md"] = []byte("# Term\r\n\r\nA term.\r\n")
	if GlossaryFingerprint(same) != base {
		t.Error("an index regeneration, a template edit or a line-ending change moved the glossary fingerprint")
	}
	edited := clone()
	edited[fxGlossary+"/core/term.md"] = []byte("# Term\n\nA different term.\n")
	added := clone()
	added[fxGlossary+"/core/other.md"] = []byte("# Other\n")
	removed := clone()
	delete(removed, fxGlossary+"/core/term.md")
	renamed := clone()
	delete(renamed, fxGlossary+"/core/term.md")
	renamed[fxGlossary+"/core/renamed.md"] = files[fxGlossary+"/core/term.md"]
	for name, m := range map[string]map[string][]byte{"edited": edited, "added": added, "removed": removed, "renamed": renamed} {
		if GlossaryFingerprint(m) == base {
			t.Errorf("a term %s did not move the glossary fingerprint", name)
		}
	}
}

func TestScopeFingerprintIsStableAcrossLineEndings(t *testing.T) {
	lf := "---\nstatus: current\n---\n# Scope\n\nThe eight phases.\n"
	base := ScopeFingerprint(lf)
	if !issueschema.ValidFingerprint(base) {
		t.Fatalf("fingerprint %q is not a 64-hex SHA-256", base)
	}
	if ScopeFingerprint(strings.ReplaceAll(lf, "\n", "\r\n")) != base {
		t.Error("CRLF moved the scope fingerprint")
	}
	if ScopeFingerprint(strings.TrimPrefix(lf, "---\nstatus: current\n---\n")) != base {
		t.Error("the frontmatter moved the scope fingerprint")
	}
	if ScopeFingerprint(strings.Replace(lf, "eight", "nine", 1)) == base {
		t.Error("a scope edit did not move the scope fingerprint")
	}
}

// --- the whole write ---

// ac-1: one record carrying the occasion, the before fingerprints of the
// previously committed state, the after fingerprints of HEAD, which surfaces
// changed, and the ground — and nothing of the abandoned text.
func TestReframeRecordsACommittedRewrite(t *testing.T) {
	r := reframeFixture(t)
	before := frameAt(t, r)
	rewriteConstrual(r, "We are treating this as a record of judgement.")
	after := frameAt(t, r)

	res, err := Reframe(reframeReq(r, fxItem))
	if err != nil {
		t.Fatalf("Reframe: %v", err)
	}
	if res.Half != ReframeHalfWhole || res.Before != before || res.After != after {
		t.Fatalf("result = %+v\nwant before %+v after %+v", res, before, after)
	}
	if !slices.Equal(res.Changed, []string{"construal"}) || res.Commits != 1 {
		t.Fatalf("changed = %v commits = %d, want [construal] across 1", res.Changed, res.Commits)
	}
	if got := reframeFiles(t, r); len(got) != 1 || got[0] != res.ID+".md" {
		t.Fatalf("reframes = %v, want exactly %s.md", got, res.ID)
	}
	fm, raw := readReframe(t, r, res.Path)
	want := map[string]any{
		"schema_version": 1, "id": res.ID, "occasioned_by": fxItem, "grounds": reframeGround,
		"construal_before": before.Construal, "glossary_before": before.Glossary, "scope_before": before.Scope,
		"construal_after": after.Construal, "glossary_after": after.Glossary, "scope_after": after.Scope,
	}
	for k, v := range want {
		if fm[k] != v {
			t.Errorf("%s = %v, want %v", k, fm[k], v)
		}
	}
	if ch, _ := fm["changed"].([]string); !slices.Equal(ch, []string{"construal"}) {
		t.Errorf("changed = %v, want [construal]", fm["changed"])
	}
	if len(fm) != len(issueschema.ReframeKnown) {
		t.Errorf("the record carries %d keys, want the complete record's %d", len(fm), len(issueschema.ReframeKnown))
	}
	if strings.Contains(raw, fxOldPhrase) || strings.Contains(raw, "record of judgement") {
		t.Fatalf("the record carries a surface's text:\n%s", raw)
	}
	if !strings.HasPrefix(res.Path, ".abcd/work/issues/reframes/rfm-") {
		t.Errorf("path = %q", res.Path)
	}
}

func TestReframeRecordsAGlossaryRewrite(t *testing.T) {
	r := reframeFixture(t)
	r.Write(fxGlossary+"/core/term.md", "# Term\n\nA sharper term.\n")
	r.Commit("rewrite a glossary term")
	res, err := Reframe(reframeReq(r, fxDisposition))
	if err != nil {
		t.Fatalf("Reframe: %v", err)
	}
	if !slices.Equal(res.Changed, []string{"glossary"}) {
		t.Fatalf("changed = %v, want [glossary]", res.Changed)
	}
	if res.Before.Construal != res.After.Construal || res.Before.Scope != res.After.Scope {
		t.Errorf("an unmoved surface's fingerprints differ: %+v", res)
	}
}

func TestReframeRecordsAScopeRewrite(t *testing.T) {
	r := reframeFixture(t)
	r.Write(fxScope, scopeDoc("The scope is the nine phases."))
	r.Commit("rewrite the scope")
	res, err := Reframe(reframeReq(r, fxSurprise))
	if err != nil {
		t.Fatalf("Reframe: %v", err)
	}
	if !slices.Equal(res.Changed, []string{"scope"}) {
		t.Fatalf("changed = %v, want [scope]", res.Changed)
	}
}

// ac-2: a frame with no distinct prior committed state has no reframe to
// record, and the refusal says so.
func TestReframeRefusesAFrameWithNoPriorState(t *testing.T) {
	r := reframeFixture(t)
	before := ledgerDigest(t, ledgerOf(r))
	_, err := Reframe(reframeReq(r, fxItem))
	if err == nil || !strings.Contains(err.Error(), "matches no prior committed state") {
		t.Fatalf("err = %v, want the no-prior-state refusal", err)
	}
	if ledgerDigest(t, ledgerOf(r)) != before {
		t.Fatal("a refused reframe changed the ledger")
	}
}

func TestReframeRefusesUncommittedChangesWithoutOpen(t *testing.T) {
	r := reframeFixture(t)
	rewriteConstrual(r, "A committed rewrite.")
	for name, edit := range map[string]func(){
		"construal": func() { r.Write(fxFraming, framingDoc("An uncommitted rewrite.")) },
		"glossary":  func() { r.Write(fxGlossary+"/core/new-term.md", "# New\n") },
		"scope":     func() { r.Write(fxScope, scopeDoc("An uncommitted scope.")) },
	} {
		t.Run(name, func(t *testing.T) {
			r.Git("checkout", "--", ".")
			r.Git("clean", "-fdq", "--", fxGlossary)
			edit()
			_, err := Reframe(reframeReq(r, fxItem))
			if err == nil || !strings.Contains(err.Error(), "the "+name+" has uncommitted changes") || !strings.Contains(err.Error(), "--open") {
				t.Fatalf("err = %v, want a refusal naming the %s and --open", err, name)
			}
		})
	}
	if got := reframeFiles(t, r); len(got) != 0 {
		t.Fatalf("a refused reframe wrote %v", got)
	}
}

// --- the two halves ---

func TestReframeOpensAHalfBeforeTheCommit(t *testing.T) {
	r := reframeFixture(t)
	head := frameAt(t, r)
	r.Write(fxFraming, framingDoc("The rewrite, not yet committed."))
	req := reframeReq(r, fxItem)
	req.Open = true
	res, err := Reframe(req)
	if err != nil {
		t.Fatalf("Reframe --open: %v", err)
	}
	if res.Half != ReframeHalfOpen || res.Before != head || res.After != (Frame{}) || len(res.Changed) != 0 {
		t.Fatalf("result = %+v, want the before triple of HEAD and nothing after", res)
	}
	fm, _ := readReframe(t, r, res.Path)
	for _, k := range issueschema.ReframeAfter {
		if _, ok := fm[k]; ok {
			t.Errorf("an open record carries %q", k)
		}
	}
	if fm["construal_before"] != head.Construal {
		t.Errorf("construal_before = %v, want HEAD's %s", fm["construal_before"], head.Construal)
	}
}

func TestCompleteFinishesAnOpenRecord(t *testing.T) {
	r := reframeFixture(t)
	head := frameAt(t, r)
	req := reframeReq(r, fxItem)
	req.Open = true
	opened, err := Reframe(req)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	r.Write(fxFraming, framingDoc("The rewrite, now committed."))
	r.Write(fxScope, scopeDoc("A scope that moved with it."))
	r.Git("add", fxFraming, fxScope)
	r.Git("commit", "-m", "rewrite the construal and the scope")
	after := frameAt(t, r)

	res, err := Reframe(ReframeRequest{RepoRoot: r.Root(), Complete: opened.ID})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if res.Half != ReframeHalfCompleted || res.ID != opened.ID || res.Before != head || res.After != after {
		t.Fatalf("result = %+v", res)
	}
	if !slices.Equal(res.Changed, []string{"construal", "scope"}) || res.Commits != 1 {
		t.Fatalf("changed = %v commits = %d", res.Changed, res.Commits)
	}
	fm, _ := readReframe(t, r, opened.Path)
	if fm["construal_after"] != after.Construal || fm["scope_after"] != after.Scope || fm["grounds"] != reframeGround {
		t.Fatalf("completed record = %v", fm)
	}
	if got := reframeFiles(t, r); len(got) != 1 {
		t.Fatalf("completion wrote a second record: %v", got)
	}
	// A complete record is not completed twice.
	if _, err := Reframe(ReframeRequest{RepoRoot: r.Root(), Complete: opened.ID}); err == nil || !strings.Contains(err.Error(), "already complete") {
		t.Fatalf("second completion: err = %v", err)
	}
}

func TestCompleteRefusesWhenNoSurfaceMoved(t *testing.T) {
	r := reframeFixture(t)
	req := reframeReq(r, fxItem)
	req.Open = true
	opened, err := Reframe(req)
	if err != nil {
		t.Fatal(err)
	}
	r.Write("unrelated.md", "an unrelated change\n")
	r.Commit("unrelated")
	before := ledgerDigest(t, ledgerOf(r))
	_, err = Reframe(ReframeRequest{RepoRoot: r.Root(), Complete: opened.ID})
	if err == nil || !strings.Contains(err.Error(), "still the state the record opened against") {
		t.Fatalf("err = %v, want the nothing-rewritten refusal", err)
	}
	if ledgerDigest(t, ledgerOf(r)) != before {
		t.Fatal("a refused completion changed the ledger")
	}
}

// A rewrite committed in two commits, and one brought in by a merge commit,
// pair the same way: the merge strategy does not decide the outcome.
func TestCompleteCrossesATwoCommitRewrite(t *testing.T) {
	t.Run("two commits", func(t *testing.T) {
		r := reframeFixture(t)
		req := reframeReq(r, fxItem)
		req.Open = true
		opened, err := Reframe(req)
		if err != nil {
			t.Fatal(err)
		}
		rewriteConstrual(r, "First half of the rewrite.")
		r.Write(fxGlossary+"/core/term.md", "# Term\n\nSecond half of the rewrite.\n")
		r.Commit("rewrite a term")
		res, err := Reframe(ReframeRequest{RepoRoot: r.Root(), Complete: opened.ID})
		if err != nil {
			t.Fatalf("complete: %v", err)
		}
		if res.Commits != 2 || !slices.Equal(res.Changed, []string{"construal", "glossary"}) {
			t.Fatalf("commits = %d changed = %v, want 2 and [construal glossary]", res.Commits, res.Changed)
		}
	})
	t.Run("a merge commit", func(t *testing.T) {
		r := reframeFixture(t)
		req := reframeReq(r, fxItem)
		req.Open = true
		opened, err := Reframe(req)
		if err != nil {
			t.Fatal(err)
		}
		// The open record is uncommitted ledger state; keep it out of the
		// branch switch.
		r.Git("checkout", "-q", "-b", "rewrite")
		rewriteConstrual(r, "The rewrite on its own branch.")
		r.Write(fxScope, scopeDoc("And a scope on that branch."))
		r.Git("add", fxScope)
		r.Git("commit", "-q", "-m", "rewrite the scope")
		r.Git("checkout", "-q", "main")
		r.Write("unrelated.md", "main moved meanwhile\n")
		r.Git("add", "unrelated.md")
		r.Git("commit", "-q", "-m", "unrelated")
		r.Git("merge", "-q", "--no-ff", "-m", "merge the rewrite", "rewrite")
		res, err := Reframe(ReframeRequest{RepoRoot: r.Root(), Complete: opened.ID})
		if err != nil {
			t.Fatalf("complete across a merge: %v", err)
		}
		if res.Commits < 2 || !slices.Equal(res.Changed, []string{"construal", "scope"}) {
			t.Fatalf("commits = %d changed = %v", res.Commits, res.Changed)
		}
	})
}

func TestCompleteRefusesARewriteItCannotPair(t *testing.T) {
	t.Run("a before state the history never held", func(t *testing.T) {
		r := reframeFixture(t)
		req := reframeReq(r, fxItem)
		req.Open = true
		opened, err := Reframe(req)
		if err != nil {
			t.Fatal(err)
		}
		// Hand-edit the before fingerprint to a state no commit ever held.
		path := filepath.Join(r.Root(), filepath.FromSlash(opened.Path))
		raw, _ := os.ReadFile(path)
		sum := sha256.Sum256([]byte("a frame nobody committed"))
		phantom := hex.EncodeToString(sum[:])
		edited := strings.Replace(string(raw), opened.Before.Construal, phantom, 1)
		if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
			t.Fatal(err)
		}
		rewriteConstrual(r, "A rewrite.")
		_, err = Reframe(ReframeRequest{RepoRoot: r.Root(), Complete: opened.ID})
		if err == nil || !strings.Contains(err.Error(), "no longer contains") || !strings.Contains(err.Error(), phantom) {
			t.Fatalf("err = %v, want the unpairable refusal naming the before triple", err)
		}
	})
	t.Run("a surface rewritten beyond the bound", func(t *testing.T) {
		orig := frameHistoryBound
		frameHistoryBound = 2
		t.Cleanup(func() { frameHistoryBound = orig })
		r := reframeFixture(t)
		req := reframeReq(r, fxItem)
		req.Open = true
		opened, err := Reframe(req)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range []string{"one", "two", "three"} {
			rewriteConstrual(r, "Rewrite "+c+".")
		}
		before := ledgerDigest(t, ledgerOf(r))
		_, err = Reframe(ReframeRequest{RepoRoot: r.Root(), Complete: opened.ID})
		if err == nil || !strings.Contains(err.Error(), "bound") || !strings.Contains(err.Error(), opened.Before.Construal) {
			t.Fatalf("err = %v, want a refusal naming the bound and the before triple", err)
		}
		if ledgerDigest(t, ledgerOf(r)) != before {
			t.Fatal("a refused completion changed the ledger")
		}
	})
}

func TestReframeRefusesASecondOpenRecord(t *testing.T) {
	r := reframeFixture(t)
	req := reframeReq(r, fxItem)
	req.Open = true
	opened, err := Reframe(req)
	if err != nil {
		t.Fatal(err)
	}
	req.OccasionedBy = fxSurprise
	_, err = Reframe(req)
	if err == nil || !strings.Contains(err.Error(), opened.ID) || !strings.Contains(err.Error(), "open") {
		t.Fatalf("err = %v, want a refusal naming the open %s", err, opened.ID)
	}
	if got := reframeFiles(t, r); len(got) != 1 {
		t.Fatalf("reframes = %v, want the one open record", got)
	}
}

// --- the ground ---

func TestReframeHoldsTheGroundToTheFloor(t *testing.T) {
	r := reframeFixture(t)
	rewriteConstrual(r, "A rewrite.")
	for _, g := range []string{"", "   ", "ok", "because"} {
		req := reframeReq(r, fxItem)
		req.Grounds = g
		if _, err := Reframe(req); !errors.Is(err, ErrGroundsRefused) {
			t.Errorf("ground %q: err = %v, want ErrGroundsRefused", g, err)
		}
	}
	if got := reframeFiles(t, r); len(got) != 0 {
		t.Fatalf("a refused ground wrote %v", got)
	}
}

func TestReframeRedactsTheGround(t *testing.T) {
	r := reframeFixture(t)
	rewriteConstrual(r, "A rewrite.")
	home := os.Getenv("HOME")
	req := reframeReq(r, fxItem)
	req.Grounds = "the reading cited " + filepath.Join(home, "notes", "draft.md") + " which recast the whole construal"
	res, err := Reframe(req)
	if err != nil {
		t.Fatalf("Reframe: %v", err)
	}
	if res.Redacted == 0 {
		t.Fatal("Redacted = 0, want the home path counted")
	}
	_, raw := readReframe(t, r, res.Path)
	if strings.Contains(raw, home) {
		t.Fatalf("the committed reframe carries the caller's home root:\n%s", raw)
	}
}

// --- the occasion ---

// ac-3: an occasion that does not resolve to a reading item, a disposition or
// a surprise this ledger holds refuses, writing nothing.
func TestReframeRefusesAnUnresolvableOccasion(t *testing.T) {
	r := reframeFixture(t)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "srp-9.md"), []byte("---\nid: srp-9\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "srp-9.md"), filepath.Join(ledgerOf(r), "surprises", "srp-9.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	r.Commit("a symlinked surprise")
	rewriteConstrual(r, "A rewrite.")
	for _, occ := range []string{"rdi-99", "dsp-99", "srp-9", "iss-1", "adm-1", "a reading", ""} {
		before := ledgerDigest(t, ledgerOf(r))
		if _, err := Reframe(reframeReq(r, occ)); err == nil {
			t.Errorf("occasion %q: resolved, want a refusal", occ)
		}
		if ledgerDigest(t, ledgerOf(r)) != before {
			t.Errorf("occasion %q: a refused reframe changed the ledger", occ)
		}
	}
}

func TestReframeRefusesAnUncommittedOccasion(t *testing.T) {
	r := reframeFixture(t)
	rewriteConstrual(r, "A rewrite.")
	r.Write(".abcd/work/issues/readings/rdg-1/rdi-12.md", "---\nid: rdi-12\n---\n")
	_, err := Reframe(reframeReq(r, "rdi-12"))
	if err == nil || !strings.Contains(err.Error(), "rdi-12 is not committed") {
		t.Fatalf("err = %v, want the uncommitted-occasion refusal", err)
	}
	// The first half asks the same: the occasion must be committed at HEAD.
	req := reframeReq(r, "rdi-12")
	req.Open = true
	if _, err := Reframe(req); err == nil || !strings.Contains(err.Error(), "rdi-12 is not committed") {
		t.Fatalf("open: err = %v, want the uncommitted-occasion refusal", err)
	}
}

func TestReframeRefusesAnOccasionCommittedAfterTheRewrite(t *testing.T) {
	t.Run("a later commit", func(t *testing.T) {
		r := reframeFixture(t)
		rewriteConstrual(r, "A rewrite.")
		r.Write(".abcd/work/issues/readings/rdg-1/rdi-12.md", "---\nid: rdi-12\n---\n")
		r.Commit("the occasion, after the rewrite")
		_, err := Reframe(reframeReq(r, "rdi-12"))
		if err == nil || !strings.Contains(err.Error(), "what came later") {
			t.Fatalf("err = %v, want the predate refusal", err)
		}
	})
	t.Run("the same commit", func(t *testing.T) {
		r := reframeFixture(t)
		r.Write(fxFraming, framingDoc("A rewrite committed with its occasion."))
		r.Write(".abcd/work/issues/readings/rdg-1/rdi-12.md", "---\nid: rdi-12\n---\n")
		r.Commit("the rewrite and the occasion at once")
		_, err := Reframe(reframeReq(r, "rdi-12"))
		if err == nil || !strings.Contains(err.Error(), "what came later") {
			t.Fatalf("err = %v, want the predate refusal", err)
		}
	})
	t.Run("a completion", func(t *testing.T) {
		r := reframeFixture(t)
		req := reframeReq(r, fxItem)
		req.Open = true
		opened, err := Reframe(req)
		if err != nil {
			t.Fatal(err)
		}
		// The occasion's record is removed and re-added AFTER the rewrite, so
		// the commit that added it no longer predates the rewrite.
		r.Git("rm", "-q", ".abcd/work/issues/readings/rdg-1/"+fxItem+".md")
		r.Git("commit", "-q", "-m", "drop the occasion")
		rewriteConstrual(r, "A rewrite.")
		r.Write(".abcd/work/issues/readings/rdg-1/"+fxItem+".md", "---\nid: rdi-11\n---\n")
		r.Git("add", ".abcd/work/issues/readings")
		r.Git("commit", "-q", "-m", "restore the occasion")
		_, err = Reframe(ReframeRequest{RepoRoot: r.Root(), Complete: opened.ID})
		if err == nil || !strings.Contains(err.Error(), "what came later") {
			t.Fatalf("err = %v, want the predate refusal", err)
		}
	})
}

// --- the lock ---

func TestReframeSerialisesOnTheLedgerLock(t *testing.T) {
	// Two first halves raced: the open-record check and the write are one
	// decision under the lock, so exactly one lands and the other is refused
	// naming the record that did.
	r2 := reframeFixture(t)
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, occ := range []string{fxItem, fxSurprise} {
		wg.Add(1)
		go func(i int, occ string) {
			defer wg.Done()
			q := reframeReq(r2, occ)
			q.Open = true
			_, errs[i] = Reframe(q)
		}(i, occ)
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case !strings.Contains(err.Error(), "is open"):
			t.Errorf("the losing open was refused for another reason: %v", err)
		}
	}
	if ok != 1 || len(reframeFiles(t, r2)) != 1 {
		t.Fatalf("raced opens: errs = %v, records = %v, want exactly one open record", errs, reframeFiles(t, r2))
	}

	// And the id is minted under the lock.
	r := reframeFixture(t)
	held := mintLockProbe(t, r.Root(), ledgerOf(r))
	req := reframeReq(r, fxItem)
	req.Open = true
	if _, err := Reframe(req); err != nil {
		t.Fatalf("Reframe: %v", err)
	}
	if len(*held) != 1 || !(*held)[0] {
		t.Fatalf("mint lock probe = %v, want one mint under the lock", *held)
	}
}
