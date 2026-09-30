package ahoy

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/drainrule"
)

// The setup offer of the drain eligibility record (ruling BX2: "the PROJECT
// MUST HOLD the eligibility decision in its own record (e.g. added at setup);
// drain refuses there until it does"). The offer writes abcd's strict baseline
// as an accepted decision record, and only on the person's own yes.

// drainRulePrompter approves every category question and answers the drain
// rule offer as told; every other offer is declined.
func drainRulePrompter(yes bool) *scriptedPrompter {
	return &scriptedPrompter{confirm: func(q string) bool {
		switch {
		case strings.HasPrefix(q, "Apply "):
			return true
		case strings.Contains(q, drainRuleQuestionTail):
			return yes
		}
		return false
	}}
}

// TestDrainRuleOfferWritesTheBaselineOnYes: a managed repository without the
// record is offered it, the question states the rule, and a yes writes an
// accepted record that the drain's own reader reads as the strict baseline,
// after which the offer is not made again.
func TestDrainRuleOfferWritesTheBaselineOnYes(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !hasGap(det.Gaps, DrainRuleOfferGapID) {
		t.Fatalf("a repository without the record is not offered it: gaps %v", det.Gaps)
	}

	p := drainRulePrompter(true)
	res, err := Install(repo, InstallOptions{}, p)
	if err != nil {
		t.Fatal(err)
	}
	asked := false
	for _, q := range p.asked {
		if strings.Contains(q, drainRuleQuestionTail) {
			asked = true
			for _, want := range []string{"nitpick", "minor", "security", "remedy", "abcd drain"} {
				if !strings.Contains(q, want) {
					t.Errorf("the offer does not state %q:\n%s", want, q)
				}
			}
		}
	}
	if !asked {
		t.Fatalf("the drain rule was not offered; asked %q", p.asked)
	}
	r, err := drainrule.Load(repo)
	if err != nil {
		t.Fatalf("the written record does not load: %v (writes %v, notes %v)", err, res.Writes, res.Notes)
	}
	if len(r.Loosened) != 0 || r.Security != drainrule.SecurityHandBack || strings.Join(r.Severities, ",") != "nitpick,minor" {
		t.Errorf("the offer wrote a rule that is not the baseline: %+v", r)
	}
	raw, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(r.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "\nstatus: accepted\n") || !strings.Contains(string(raw), "## Decision") {
		t.Errorf("the record is not an accepted decision record:\n%s", raw)
	}
	if !containsString(res.Writes, r.Path) {
		t.Errorf("the write is not reported: %v", res.Writes)
	}
	det, _ = Detect(repo)
	if hasGap(det.Gaps, DrainRuleOfferGapID) {
		t.Error("a repository holding the record is offered it again")
	}
}

// TestDrainRuleOfferDeclineWritesNothing: a no writes nothing and is not
// persisted, so the next install offers again.
func TestDrainRuleOfferDeclineWritesNothing(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	if _, err := Install(repo, InstallOptions{}, drainRulePrompter(false)); err != nil {
		t.Fatal(err)
	}
	if _, err := drainrule.Load(repo); !errors.Is(err, drainrule.ErrUnrecorded) {
		t.Fatalf("a declined offer left a rule: %v", err)
	}
	det, _ := Detect(repo)
	if !hasGap(det.Gaps, DrainRuleOfferGapID) {
		t.Error("a decline was persisted; the offer must stand for the next install")
	}
}

// TestDrainRuleOfferYesFlagSkipsAndSaysSo: --yes approves the category but
// never writes the record, which decides what an unattended agent may do, and
// names it under optional_skipped, as the other consent offers are.
func TestDrainRuleOfferYesFlagSkipsAndSaysSo(t *testing.T) {
	setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(res.OptionalSkipped, DrainRuleOfferGapID) {
		t.Errorf("optional_skipped = %v, want it to name %s", res.OptionalSkipped, DrainRuleOfferGapID)
	}
	if _, err := drainrule.Load(repo); !errors.Is(err, drainrule.ErrUnrecorded) {
		t.Errorf("--yes wrote a drain rule: %v", err)
	}
}

// TestDrainRuleOfferLeavesAMalformedRecordAlone: a repository that states a
// rule, however badly, is not offered a second one; the drain names what is
// wrong with the record it has.
func TestDrainRuleOfferLeavesAMalformedRecordAlone(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	dir := filepath.Join(repo, filepath.FromSlash(drainrule.ADRsRelDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: adr-7\nstatus: accepted\ndrain_categories: [bug]\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "0007-rule.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if hasGap(det.Gaps, DrainRuleOfferGapID) {
		t.Error("a repository stating a malformed rule is offered a second record")
	}
}
