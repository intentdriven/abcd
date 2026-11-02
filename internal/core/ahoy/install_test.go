package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// limitsRefusedRepo is a git repository abcd does not manage yet, whose
// .abcd/config.json sets question.rows outside its bounds.
func limitsRefusedRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), []byte(`{"question": {"rows": 100}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}

// TestInstallRefusesBeforeItsFirstQuestionOnARefusedValue: a question value
// outside its bounds stops setup before its first question and its first
// write, naming the key, the value, the bounds and the file, and the run is
// refused, which the front door exits 2 on (itd-2610301914350541 decision 7).
func TestInstallRefusesBeforeItsFirstQuestionOnARefusedValue(t *testing.T) {
	setupHermetic(t)
	repo := limitsRefusedRepo(t)
	p := &recordingPrompter{confirm: true}
	res, err := Install(repo, InstallOptions{}, p)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "refused" {
		t.Fatalf("status = %q, want refused (notes %v)", res.Status, res.Notes)
	}
	if len(p.asked) != 0 {
		t.Errorf("setup asked %d question(s) before refusing: %q", len(p.asked), p.asked)
	}
	notes := strings.Join(res.Notes, "\n")
	for _, w := range []string{"question.rows", "100", "24 to 60", ".abcd/config.json"} {
		if !strings.Contains(notes, w) {
			t.Errorf("the refusal does not name %q:\n%s", w, notes)
		}
	}
	if len(res.Writes) != 0 {
		t.Errorf("a refused setup wrote %v", res.Writes)
	}
	if _, err := os.Stat(filepath.Join(repo, "AGENTS.md")); !os.IsNotExist(err) {
		t.Errorf("a refused setup left AGENTS.md (%v)", err)
	}
}

// TestDryRunRefusesARefusedValueWhileTheReadmeQuestionIsOpen: the dry run
// relays the README pair cut to the limits in force, so a refused value
// refuses it, naming the key, rather than relaying a pair cut to figures that
// are not in force.
func TestDryRunRefusesARefusedValueWhileTheReadmeQuestionIsOpen(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	det, err := DryRun(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !hasGapID(det.Gaps, ReadmeOfferGapID) {
		t.Skip("the README question is not open in an installed repository here")
	}
	cfg := filepath.Join(repo, ".abcd", "config.json")
	raw, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	body := strings.Replace(string(raw), "{", `{"question": {"label_words": 9},`, 1)
	if err := os.WriteFile(cfg, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = DryRun(repo)
	if err == nil || !strings.Contains(err.Error(), "question.label_words") || !strings.Contains(err.Error(), "4 to 8") {
		t.Fatalf("err = %v, want a refusal naming question.label_words and its bounds", err)
	}
}
