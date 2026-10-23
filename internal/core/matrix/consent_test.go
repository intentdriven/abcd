package matrix

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// countingStarter records each start.
type countingStarter struct {
	starts int
	run    RunRecord
}

func (c *countingStarter) Start(_ Plan, rec RunRecord) error {
	c.starts++
	c.run = rec
	return nil
}

// consentSetup is a lab, a plan of the run tests' declaration at the lab's pin,
// and the inputs its consent writes the record from.
func consentSetup(t *testing.T) (root string, runPath string, p Plan, in ConsentInputs) {
	t.Helper()
	root, home, m, ri := matrixLab(t)
	p, err := BuildPlan(PlanInputs{Lab: m.ID, FrozenCommit: m.Pin, Declaration: []byte(goodDeclaration),
		API: planAPI(t, "https://api.example.com/v1"), CheckRoute: anyRoute,
		Cap: ResolvedCap{Cap: &Money{Amount: 25, Unit: "usd"}, Origin: CapFromRun}, Cores: 16})
	if err != nil {
		t.Fatal(err)
	}
	in = ConsentInputs{RunID: ri.RunID, RootSHA: ri.RootSHA, Binary: ri.Binary, Declaration: []byte(goodDeclaration),
		PromptSHA256: ri.PromptSHA256, ToolDefinitionsSHA256: ri.ToolDefinitionsSHA256, Machine: Machine{OS: "darwin", Arch: "arm64", Cores: 16}}
	return root, runDir(home, m, ri.RunID), p, in
}

func TestNothingStartsWithoutConsent(t *testing.T) {
	root, run, p, in := consentSetup(t)
	s := &countingStarter{}
	if _, err := Consent(root, p, "", in, s); !errors.Is(err, ErrNotConsented) {
		t.Fatalf("err = %v; want ErrNotConsented", err)
	}
	if s.starts != 0 {
		t.Fatalf("the run started %d times without consent", s.starts)
	}
	if _, err := os.Stat(run); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a run with no consent left %s: %v", run, err)
	}
}

func TestAStaleDigestIsRefusedWithNothingWritten(t *testing.T) {
	root, run, p, in := consentSetup(t)
	s := &countingStarter{}
	stale := strings.Repeat("0", 64)
	if _, err := Consent(root, p, stale, in, s); err == nil || !strings.Contains(err.Error(), "not the plan") {
		t.Fatalf("err = %v; want a refusal: the approval names another plan", err)
	}
	// A plan changed after it was shown, its digest left as shown, is refused
	// as well: the approval holds for what was shown, never for an edit.
	edited := p
	edited.Concurrency++
	if _, err := Consent(root, edited, p.Digest, in, s); err == nil {
		t.Fatal("an edited plan carrying the shown digest was started")
	}
	// The declaration consented to is the one the plan was built from.
	other := in
	other.Declaration = []byte(strings.Replace(goodDeclaration, "routing check", "another check", 1))
	if _, err := Consent(root, p, p.Digest, other, s); err == nil || !strings.Contains(err.Error(), "declaration") {
		t.Fatalf("err = %v; want a refusal naming the declaration", err)
	}
	if s.starts != 0 {
		t.Fatalf("a refused consent started the run %d times", s.starts)
	}
	if _, err := os.Stat(run); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused consent left %s: %v", run, err)
	}
}

func TestConsentWritesTheRunRecordAndStartsOnce(t *testing.T) {
	root, run, p, in := consentSetup(t)
	s := &countingStarter{}
	rec, err := Consent(root, p, p.Digest, in, s)
	if err != nil {
		t.Fatal(err)
	}
	if s.starts != 1 || s.run.RunID != rec.RunID {
		t.Fatalf("starts = %d, run %q; want one start of %q", s.starts, s.run.RunID, rec.RunID)
	}
	for _, f := range []string{"run.json", "declaration.json"} {
		if _, err := os.Stat(filepath.Join(run, f)); err != nil {
			t.Fatalf("consent did not write %s: %v", f, err)
		}
	}
	if rec.SpendCap == nil || rec.SpendCap.Amount != 25 || rec.SpendCapOrigin != CapFromRun || rec.FrozenCommit != p.FrozenCommit {
		t.Fatalf("record = %+v", rec)
	}
	for _, c := range rec.Cells {
		if c.Source == SourceOpenRouter && (c.ProviderPin == nil || c.ProviderPin.Order[0] != "deepinfra") {
			t.Fatalf("the OpenRouter cell's record pin = %+v; want the plan's", c.ProviderPin)
		}
	}
	// A second consent to the same plan and run id writes nothing over it.
	if _, err := Consent(root, p, p.Digest, in, s); err == nil || s.starts != 1 {
		t.Fatalf("a second consent to the same run: err %v, starts %d; want a refusal and one start", err, s.starts)
	}
}
