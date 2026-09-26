package capture

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// A ground that clears the substance floor, used wherever the test is not about
// the ground.
const admitGround = "the configuration engages the frame the comparative reading characterised"

// admitFixture ingests one widening item and commits the comparative run over
// its run, so the item is admissible.
func admitFixture(t *testing.T) (repo, ir, item string) {
	t.Helper()
	repo, ir, item = readingFixture(t, issueschema.PositionWidening)
	commitComparativeRun(t, repo, "rdg-2608300000000002", fixtureRun)
	return repo, ir, item
}

// ledgerDigest hashes every file under the issues root, keyed by relative path,
// so a refusal can be proved to have left the ledger byte-identical.
func ledgerDigest(t *testing.T, ir string) string {
	t.Helper()
	var lines []string
	_ = filepath.Walk(ir, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			t.Fatalf("read %s: %v", p, rerr)
		}
		sum := sha256.Sum256(raw)
		rel, _ := filepath.Rel(ir, p)
		lines = append(lines, rel+" "+hex.EncodeToString(sum[:]))
		return nil
	})
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// admissionFiles lists the admission records filed under run.
func admissionFiles(t *testing.T, ir, run string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(ir, issueschema.AdmissionsDir, run))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

// readFM parses one record's frontmatter.
func readFM(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fm, _, err := parseFrontmatterAndBody(string(raw))
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return fm
}

// ac-1: with no disposition and a comparative run committed over the item's
// run, one act writes the `accepted` disposition and the admission under one
// lock; a second admit finds the admission by (run, proposal) and refuses.
func TestAdmitWritesBothRecordsAndRefusesTwice(t *testing.T) {
	repo, ir, item := admitFixture(t)
	res, err := Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item, Grounds: admitGround})
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if !res.DispositionWritten || res.Run != fixtureRun || res.Item != item {
		t.Fatalf("result = %+v", res)
	}
	disp := readFM(t, filepath.Join(repo, filepath.FromSlash(res.DispositionPath)))
	if disp["state"] != issueschema.DispositionAccepted || disp["item"] != item || disp["id"] != res.Disposition {
		t.Fatalf("disposition = %v", disp)
	}
	adm := readFM(t, filepath.Join(repo, filepath.FromSlash(res.Path)))
	if adm["proposal"] != item || adm["run"] != fixtureRun || adm["id"] != res.Admission || adm["grounds"] != admitGround {
		t.Fatalf("admission = %v", adm)
	}
	if got := admissionFiles(t, ir, fixtureRun); len(got) != 1 || got[0] != res.Admission+".md" {
		t.Fatalf("admissions = %v, want exactly %s.md", got, res.Admission)
	}

	before := ledgerDigest(t, ir)
	_, err = Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item, Grounds: admitGround})
	if !errors.Is(err, ErrInvariantViolation) || !strings.Contains(err.Error(), res.Admission) {
		t.Fatalf("a second admit: err = %v, want ErrInvariantViolation naming %s", err, res.Admission)
	}
	if ledgerDigest(t, ir) != before {
		t.Fatal("a refused second admit changed the ledger")
	}
}

// ac-2: where `accepted` already stands, the admission is written alone and the
// disposition's bytes are untouched.
func TestAdmitWritesTheAdmissionAloneOverAStandingAcceptance(t *testing.T) {
	repo, ir, item := admitFixture(t)
	d, err := Disposition(DispositionRequest{
		RepoRoot: repo, IssuesRoot: ir, Item: item,
		State: issueschema.DispositionAccepted, Grounds: admitGround,
	})
	if err != nil {
		t.Fatalf("Disposition: %v", err)
	}
	dispPath := filepath.Join(repo, filepath.FromSlash(d.Path))
	dispBefore, _ := os.ReadFile(dispPath)

	res, err := Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item, Grounds: admitGround})
	if err != nil {
		t.Fatalf("Admit over a standing acceptance: %v", err)
	}
	if res.DispositionWritten || res.Disposition != d.ID {
		t.Fatalf("result = %+v, want the standing %s and no new disposition", res, d.ID)
	}
	dispAfter, _ := os.ReadFile(dispPath)
	if string(dispAfter) != string(dispBefore) {
		t.Fatal("the standing disposition's bytes changed")
	}
	if got := dispositionFiles(t, ir); len(got) != 1 {
		t.Fatalf("dispositions = %v, want the one standing record", got)
	}
	if got := admissionFiles(t, ir, fixtureRun); len(got) != 1 {
		t.Fatalf("admissions = %v, want one", got)
	}
}

// The admission-alone branch requires the ground the standing acceptance states,
// so the disposition and the admission cannot give two reasons for one act by
// any path. The refusal names both texts.
func TestAdmissionAloneRequiresTheStandingGround(t *testing.T) {
	repo, ir, item := admitFixture(t)
	if _, err := Disposition(DispositionRequest{
		RepoRoot: repo, IssuesRoot: ir, Item: item,
		State: issueschema.DispositionAccepted, Grounds: admitGround,
	}); err != nil {
		t.Fatalf("Disposition: %v", err)
	}
	before := ledgerDigest(t, ir)
	other := "a different reason entirely for taking this configuration forward"
	_, err := Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item, Grounds: other})
	if !errors.Is(err, ErrInvariantViolation) {
		t.Fatalf("a second ground: err = %v, want ErrInvariantViolation", err)
	}
	for _, want := range []string{admitGround, other} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must name %q; got %v", want, err)
		}
	}
	if ledgerDigest(t, ir) != before {
		t.Fatal("a refused admit changed the ledger")
	}
	// The same ground, spaced differently, is the same ground once folded.
	if _, err := Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item,
		Grounds: "  the configuration engages the frame\n the comparative reading   characterised "}); err != nil {
		t.Fatalf("the standing ground, refolded: %v", err)
	}
}

// ac-3's admission half: before any comparative run names the item's run, the
// admit refuses, names what it waits for, and writes nothing.
func TestAdmitRefusesBeforeTheComparativeRun(t *testing.T) {
	repo, ir, item := readingFixture(t, issueschema.PositionWidening)
	before := ledgerDigest(t, ir)
	_, err := Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item, Grounds: admitGround})
	if !errors.Is(err, ErrNotCharacterised) {
		t.Fatalf("admit before the comparative run: err = %v, want ErrNotCharacterised", err)
	}
	if !strings.Contains(err.Error(), fixtureRun) || !strings.Contains(err.Error(), "comparative") {
		t.Errorf("the refusal must name the run and the comparative run it waits for; got %v", err)
	}
	if ledgerDigest(t, ir) != before {
		t.Fatal("a refused admit changed the ledger")
	}

	// The admission-alone branch waits on the same gate: an acceptance written by
	// hand before characterisation does not open it.
	writeFile(t, filepath.Join(ir, issueschema.DispositionsDir, item, "dsp-2608300000000001.md"),
		"---\nschema_version: 1\nid: \"dsp-2608300000000001\"\nitem: \""+item+"\"\nstate: \"accepted\"\n"+
			"disposition_grounds: \""+admitGround+"\"\n---\n\n")
	_, err = Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item, Grounds: admitGround})
	if !errors.Is(err, ErrNotCharacterised) {
		t.Fatalf("the admission-alone branch before the comparative run: err = %v, want ErrNotCharacterised", err)
	}
}

// A comparative run committed with an EMPTY item set is the not-exercised
// outcome (a widening run of fewer than two candidates), and it satisfies the
// gate exactly as a characterising run does.
func TestAdmitProceedsOnAnEmptyComparativeRun(t *testing.T) {
	repo, ir, item := readingFixture(t, issueschema.PositionWidening)
	writeFile(t, filepath.Join(repo, filepath.FromSlash(issueschema.ReadingsRecordDir), "rdg-2608300000000003", issueschema.RunRecordFileName),
		`{"run_id":"rdg-2608300000000003","position":"comparative","candidate_run":"`+fixtureRun+`","candidates":1,"exercised":false,"records":[]}`)
	if _, err := Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item, Grounds: admitGround}); err != nil {
		t.Fatalf("Admit after an empty comparative run: %v", err)
	}
}

// ac-4: a standing answer in any state but `accepted` refuses, naming its id and
// state.
func TestAdmitRefusesAStandingNonAcceptance(t *testing.T) {
	for _, tc := range []DispositionRequest{
		{State: issueschema.DispositionDeclined, Grounds: "the proposal repeats a standing candidate"},
		{State: issueschema.DispositionHeld, ExitCondition: "the next widening run returns it again"},
	} {
		t.Run(tc.State, func(t *testing.T) {
			repo, ir, item := admitFixture(t)
			tc.RepoRoot, tc.IssuesRoot, tc.Item = repo, ir, item
			d, err := Disposition(tc)
			if err != nil {
				t.Fatalf("Disposition: %v", err)
			}
			before := ledgerDigest(t, ir)
			_, err = Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item, Grounds: admitGround})
			if !errors.Is(err, ErrInvariantViolation) {
				t.Fatalf("admit over %s: err = %v, want ErrInvariantViolation", tc.State, err)
			}
			if !strings.Contains(err.Error(), d.ID) || !strings.Contains(err.Error(), tc.State) {
				t.Errorf("the refusal must name %s and %s; got %v", d.ID, tc.State, err)
			}
			if ledgerDigest(t, ir) != before {
				t.Fatal("a refused admit changed the ledger")
			}
		})
	}
}

// Admission is the widening position's warm act alone.
func TestAdmitRefusesANonWideningItem(t *testing.T) {
	for _, position := range []string{"detection", "entailment", "comparative"} {
		repo, ir, item := readingFixture(t, position)
		before := ledgerDigest(t, ir)
		_, err := Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item, Grounds: admitGround})
		if !errors.Is(err, ErrInvariantViolation) || !strings.Contains(err.Error(), position) {
			t.Fatalf("admit at %s: err = %v, want ErrInvariantViolation naming the position", position, err)
		}
		if ledgerDigest(t, ir) != before {
			t.Fatalf("a refused admit at %s changed the ledger", position)
		}
	}
}

// ac-5: a blank, whitespace or degenerate ground refuses before any mint, and
// the ledger is byte-identical afterwards.
func TestAdmitRefusesADegenerateGround(t *testing.T) {
	repo, ir, item := admitFixture(t)
	before := ledgerDigest(t, ir)
	for _, g := range []string{"", "   ", "\t\n", "ok", "admit admit admit", "no time now", "fine\vby me okay then"} {
		_, err := Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item, Grounds: g})
		if !errors.Is(err, ErrGroundsRefused) {
			t.Errorf("ground %q: err = %v, want ErrGroundsRefused", g, err)
			continue
		}
		if !strings.Contains(err.Error(), "nothing written") {
			t.Errorf("ground %q: the refusal must say nothing was written; got %v", g, err)
		}
	}
	if ledgerDigest(t, ir) != before {
		t.Fatal("a refused ground changed the ledger")
	}
}

// A contested item — two standing answers — refuses as Disposition does.
func TestAdmitRefusesAContestedItem(t *testing.T) {
	repo, ir, item := readingFixture(t, issueschema.PositionWidening)
	commitComparativeRun(t, repo, "rdg-2608300000000002", fixtureRun)
	writeRawDisposition(t, ir, item, "dsp-2608300000000001", issueschema.DispositionAccepted, "")
	writeRawDisposition(t, ir, item, "dsp-2608300000000002", issueschema.DispositionDeclined, "")
	before := ledgerDigest(t, ir)
	_, err := Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item, Grounds: admitGround})
	if !errors.Is(err, ErrInvariantViolation) || !strings.Contains(err.Error(), "dsp-2608300000000001") ||
		!strings.Contains(err.Error(), "dsp-2608300000000002") {
		t.Fatalf("admit under contest: err = %v", err)
	}
	if ledgerDigest(t, ir) != before {
		t.Fatal("a refused admit changed the ledger")
	}
}

// Both records or neither: a failure writing the admission removes the
// disposition this act wrote.
func TestAdmitRemovesTheDispositionWhenTheAdmissionWriteFails(t *testing.T) {
	repo, ir, item := admitFixture(t)
	before := dispositionFiles(t, ir)
	orig := readingWriteHook
	t.Cleanup(func() { readingWriteHook = orig })
	readingWriteHook = func(path string, data []byte) error {
		if strings.Contains(filepath.ToSlash(path), "/"+issueschema.AdmissionsDir+"/") {
			return errors.New("injected admission write failure")
		}
		return writeContained(ledgerBase(repo, ir), path, data)
	}
	_, err := Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item, Grounds: admitGround})
	if err == nil || !strings.Contains(err.Error(), "injected admission write failure") {
		t.Fatalf("err = %v, want the injected failure", err)
	}
	if got := dispositionFiles(t, ir); len(got) != len(before) {
		t.Fatalf("the disposition survived a failed admission: %v", got)
	}
	if got := admissionFiles(t, ir, fixtureRun); len(got) != 0 {
		t.Fatalf("admissions = %v, want none", got)
	}
}

// Both ids are minted under the ledger lock.
func TestAdmitMintsUnderTheLock(t *testing.T) {
	repo, ir, item := admitFixture(t)
	held := mintLockProbe(t, repo, ir)
	if _, err := Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item, Grounds: admitGround}); err != nil {
		t.Fatalf("Admit: %v", err)
	}
	if len(*held) != 2 {
		t.Fatalf("mints = %d, want 2 (the disposition and the admission)", len(*held))
	}
	for i, h := range *held {
		if !h {
			t.Fatalf("mint %d ran outside the ledger lock", i+1)
		}
	}
}

// The folded ground lands in both records: the disposition's
// disposition_grounds and the admission's grounds are one text.
func TestAdmissionAndDispositionCarryOneGround(t *testing.T) {
	repo, ir, item := admitFixture(t)
	res, err := Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item,
		Grounds: "  the configuration engages\n\tthe frame the comparative   reading characterised\n"})
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	disp := readFM(t, filepath.Join(repo, filepath.FromSlash(res.DispositionPath)))
	adm := readFM(t, filepath.Join(repo, filepath.FromSlash(res.Path)))
	if disp["disposition_grounds"] != admitGround || adm["grounds"] != admitGround {
		t.Fatalf("grounds: disposition %q, admission %q, want both %q",
			disp["disposition_grounds"], adm["grounds"], admitGround)
	}
}
