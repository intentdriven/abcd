package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	admitRun  = "rdg-2608300000000001"
	admitItem = "rdi-2608300000000002"
)

// writeWideningFixture lays down one widening item and, when characterised, the
// committed comparative run over its run — the marker the ordering gate reads.
func writeWideningFixture(t *testing.T, repo string, characterised bool) {
	t.Helper()
	dir := filepath.Join(repo, ".abcd", "work", "issues", "readings", admitRun)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, admitItem+".md"), []byte("---\n"+
		"schema_version: 1\nid: \""+admitItem+"\"\nrun: \""+admitRun+"\"\nmanifest: \"sha256:beef\"\n"+
		"position: \"widening\"\nregime: \"generative\"\npattern: \"a stated constraint\"\n"+
		"configuration: \"a third arrangement the frame does not hold\"\n"+
		"what_admits_it: \"the constraint the record already states\"\n---\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !characterised {
		return
	}
	comp := filepath.Join(repo, ".abcd", "development", "readings", "rdg-2608300000000009")
	if err := os.MkdirAll(comp, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(comp, "run.json"),
		[]byte(`{"run_id":"rdg-2608300000000009","position":"comparative","candidate_run":"`+admitRun+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
}

// `capture admit` is a front door: reachable from the CLI, rendering both ids,
// reporting `redacted` when the ground carried something the ledger never
// commits, and refusing before characterisation with nothing written.
func TestCaptureAdmitRendersAndRedacts(t *testing.T) {
	repo := captureLedgerRepo(t)
	writeWideningFixture(t, repo, false)
	out, err := runCLIErr(t, "capture", "admit", admitItem, "--grounds", "the configuration engages the frame the reading characterised")
	if err == nil || !strings.Contains(err.Error(), "comparative") {
		t.Fatalf("an admit before the comparative run must refuse naming what it waits for; err = %v\n%s", err, out)
	}
	if _, serr := os.Stat(filepath.Join(repo, ".abcd", "work", "issues", "admissions")); !os.IsNotExist(serr) {
		t.Fatalf("a refused admit wrote into the admissions store: %v", serr)
	}

	writeWideningFixture(t, repo, true)
	home := t.TempDir()
	t.Setenv("HOME", home)
	out = runCLI(t, "capture", "admit", admitItem, "--grounds",
		"the configuration engages the frame noted in "+filepath.Join(home, "notes", "frame.md"), "--json")
	var r struct {
		Admission          string `json:"admission"`
		Disposition        string `json:"disposition"`
		DispositionWritten bool   `json:"disposition_written"`
		Path               string `json:"path"`
		Run                string `json:"run"`
		Redacted           int    `json:"redacted"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("admit output not JSON: %v\n%s", err, out)
	}
	if !strings.HasPrefix(r.Admission, "adm-") || !strings.HasPrefix(r.Disposition, "dsp-") ||
		!r.DispositionWritten || r.Run != admitRun || r.Redacted == 0 {
		t.Fatalf("admit result = %+v", r)
	}
	raw, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(r.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), home) {
		t.Fatalf("the admission carries the caller's home root:\n%s", raw)
	}

	// The board carries the run's count.
	board := runCLI(t, "capture", "--json")
	var b struct {
		Outstanding struct {
			WideningRuns []struct {
				Run         string   `json:"run"`
				Items       int      `json:"items"`
				Admitted    int      `json:"admitted"`
				Outstanding []string `json:"outstanding"`
			} `json:"widening_runs"`
		} `json:"reading_outstanding"`
	}
	if err := json.Unmarshal(board, &b); err != nil {
		t.Fatalf("capture board not JSON: %v\n%s", err, board)
	}
	if len(b.Outstanding.WideningRuns) != 1 || b.Outstanding.WideningRuns[0].Admitted != 1 ||
		b.Outstanding.WideningRuns[0].Outstanding == nil {
		t.Fatalf("widening_runs = %+v\n%s", b.Outstanding.WideningRuns, board)
	}
	text := string(runCLI(t, "capture"))
	if !strings.Contains(text, "widening "+admitRun+" — 1 proposal(s): 1 admitted, 0 declined, 0 held, 0 outstanding") {
		t.Fatalf("the board must render the run's count:\n%s", text)
	}
}

// `capture surprise` needs its occasion: without --occasioned-by it refuses and
// writes nothing; with a resolving occasion it writes one record.
func TestCaptureSurpriseRequiresAnOccasion(t *testing.T) {
	repo := captureLedgerRepo(t)
	writeWideningFixture(t, repo, false)
	surprises := filepath.Join(repo, ".abcd", "work", "issues", "surprises")

	out, err := runCLIErr(t, "capture", "surprise", "the reading ranked the unexpected proposal first")
	if err == nil || !strings.Contains(err.Error(), "--occasioned-by") {
		t.Fatalf("a surprise with no occasion must refuse naming the flag; err = %v\n%s", err, out)
	}
	if _, serr := os.Stat(surprises); !os.IsNotExist(serr) {
		t.Fatalf("a refused surprise wrote into the surprise store: %v", serr)
	}

	out = runCLI(t, "capture", "surprise", "--occasioned-by", admitItem,
		"the reading ranked the unexpected proposal first", "--json")
	var r struct {
		ID           string `json:"id"`
		OccasionedBy string `json:"occasioned_by"`
		Path         string `json:"path"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("surprise output not JSON: %v\n%s", err, out)
	}
	if !strings.HasPrefix(r.ID, "srp-") || r.OccasionedBy != admitItem {
		t.Fatalf("surprise result = %+v", r)
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(r.Path))); err != nil {
		t.Fatalf("the surprise record must exist: %v", err)
	}
	// And `abcd srp-N` dispatches to it.
	d := string(runCLI(t, r.ID))
	if !strings.Contains(d, "surprise, recorded") || !strings.Contains(d, "occasioned_by: "+admitItem) {
		t.Fatalf("abcd %s:\n%s", r.ID, d)
	}
}
