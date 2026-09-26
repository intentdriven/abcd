package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

const surpriseText = "the comparative reading ranked the proposal nobody expected to survive first"

// surpriseFiles lists the surprise records.
func surpriseFiles(t *testing.T, ir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(ir, issueschema.SurprisesDir))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

// dispositionsDigest hashes the dispositions tree alone.
func dispositionsDigest(t *testing.T, ir string) string {
	t.Helper()
	return ledgerDigest(t, filepath.Join(ir, issueschema.DispositionsDir))
}

// ac-6: a surprise is its own record, written to surprises/srp-N.md, and no
// disposition file is touched on the way — "never a field on a disposition"
// in code.
func TestSurpriseIsItsOwnRecord(t *testing.T) {
	repo, ir, item := admitFixture(t)
	adm, err := Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item, Grounds: admitGround})
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	before := dispositionsDigest(t, ir)
	res, err := Surprise(SurpriseRequest{RepoRoot: repo, IssuesRoot: ir, OccasionedBy: adm.Disposition, Text: surpriseText})
	if err != nil {
		t.Fatalf("Surprise: %v", err)
	}
	if res.OccasionedBy != adm.Disposition {
		t.Fatalf("result = %+v", res)
	}
	if got := surpriseFiles(t, ir); len(got) != 1 || got[0] != res.ID+".md" {
		t.Fatalf("surprises = %v, want exactly %s.md", got, res.ID)
	}
	raw, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(res.Path)))
	if err != nil {
		t.Fatal(err)
	}
	fm, body, err := parseFrontmatterAndBody(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(fm) != len(issueschema.SurpriseRequired) || fm["id"] != res.ID || fm["occasioned_by"] != adm.Disposition {
		t.Fatalf("frontmatter = %v", fm)
	}
	if strings.TrimSpace(body) != surpriseText {
		t.Fatalf("body = %q, want the surprise itself", body)
	}
	if dispositionsDigest(t, ir) != before {
		t.Fatal("writing a surprise touched the dispositions tree")
	}
}

// The occasion is a closed form: an rdi-N, adm-N or dsp-N that resolves in this
// ledger, and nothing else. Each family resolves; an unknown id, a malformed
// id, prose, and an id of a fourth family refuse, naming it, before anything is
// minted.
func TestSurpriseRefusesAnUnresolvedOccasion(t *testing.T) {
	repo, ir, item := admitFixture(t)
	adm, err := Admit(AdmitRequest{RepoRoot: repo, IssuesRoot: ir, Item: item, Grounds: admitGround})
	if err != nil {
		t.Fatalf("Admit: %v", err)
	}
	for _, occ := range []string{item, adm.Admission, adm.Disposition} {
		if _, err := Surprise(SurpriseRequest{RepoRoot: repo, IssuesRoot: ir, OccasionedBy: occ, Text: surpriseText}); err != nil {
			t.Errorf("occasion %s: %v", occ, err)
		}
	}
	before := ledgerDigest(t, ir)
	for _, occ := range []string{
		"rdi-9999", "adm-9999", "dsp-9999", // unknown
		"rdi-", "rdi-x", "RDI-1", " " + item, "", // malformed
		"a consequence nobody predicted", // prose
		"itd-1", "iss-1", "srp-1",        // a fourth family
	} {
		_, err := Surprise(SurpriseRequest{RepoRoot: repo, IssuesRoot: ir, OccasionedBy: occ, Text: surpriseText})
		if err == nil {
			t.Errorf("occasion %q: accepted, want a refusal", occ)
			continue
		}
		if occ != "" && !strings.Contains(err.Error(), strings.TrimSpace(occ)) {
			t.Errorf("occasion %q: the refusal must name it; got %v", occ, err)
		}
	}
	if ledgerDigest(t, ir) != before {
		t.Fatal("a refused surprise changed the ledger")
	}
}

// The surprise's text is held to the same floor every grounds-shaped field in
// this workstream applies.
func TestSurpriseRefusesADegenerateText(t *testing.T) {
	repo, ir, item := readingFixture(t, "detection")
	before := ledgerDigest(t, ir)
	for _, text := range []string{"", "  ", "odd", "no time now"} {
		_, err := Surprise(SurpriseRequest{RepoRoot: repo, IssuesRoot: ir, OccasionedBy: item, Text: text})
		if !errors.Is(err, ErrGroundsRefused) || !strings.Contains(err.Error(), "nothing written") {
			t.Errorf("text %q: err = %v, want ErrGroundsRefused saying nothing was written", text, err)
		}
	}
	if ledgerDigest(t, ir) != before {
		t.Fatal("a refused surprise changed the ledger")
	}
}

// The body is redacted before it is written: a surprise lands in the committed
// ledger exactly as a capture does.
func TestSurpriseRedactsItsBody(t *testing.T) {
	repo, ir, item := readingFixture(t, "detection")
	home := t.TempDir()
	t.Setenv("HOME", home)
	res, err := Surprise(SurpriseRequest{RepoRoot: repo, IssuesRoot: ir, OccasionedBy: item,
		Text: "the reading cited " + filepath.Join(home, "notes", "draft.md") + " which nobody had shown it"})
	if err != nil {
		t.Fatalf("Surprise: %v", err)
	}
	if res.Redacted == 0 {
		t.Fatal("Redacted = 0, want the home path counted")
	}
	raw, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(res.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), home) {
		t.Fatalf("the committed surprise carries the caller's home root:\n%s", raw)
	}
}

// Its id is minted under the ledger lock.
func TestSurpriseMintsUnderTheLock(t *testing.T) {
	repo, ir, item := readingFixture(t, "detection")
	held := mintLockProbe(t, repo, ir)
	if _, err := Surprise(SurpriseRequest{RepoRoot: repo, IssuesRoot: ir, OccasionedBy: item, Text: surpriseText}); err != nil {
		t.Fatalf("Surprise: %v", err)
	}
	if len(*held) != 1 || !(*held)[0] {
		t.Fatalf("mint lock probe = %v, want one mint under the lock", *held)
	}
}
