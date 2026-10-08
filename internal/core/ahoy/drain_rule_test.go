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

// terminalScripted is a scriptedPrompter a person answers at a terminal, the
// only prompter the drain rule offer is put to.
type terminalScripted struct{ *scriptedPrompter }

func (terminalScripted) AtTerminal() bool { return true }

// drainRulePrompter approves every category question and answers the drain
// rule offer as told, at a terminal; every other offer is declined.
func drainRulePrompter(yes bool) terminalScripted {
	return terminalScripted{&scriptedPrompter{confirm: func(q string) bool {
		switch {
		case askedCategory(q) != "":
			return true
		case strings.Contains(q, drainRuleQuestionTail):
			return yes
		}
		return false
	}}}
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

// pipedPrompter is a scripted answer stream off a pipe: each confirm takes the
// next answer, and past the end reads EOF, a no, as the CLI's stdin prompter
// does. It is not a TerminalPrompter, so no person is at a terminal.
type pipedPrompter struct {
	answers []bool
	asked   []string
}

func (p *pipedPrompter) Confirm(q string) bool {
	p.asked = append(p.asked, q)
	if len(p.asked) > len(p.answers) {
		return false
	}
	return p.answers[len(p.asked)-1]
}

func (p *pipedPrompter) Prompt(_ string, _ []string, def string) string { return def }

// TestAPipedInstallStreamIsNotShiftedByTheDrainRuleOffer: a scripted `ahoy
// install` answers its questions positionally, so a question added to the
// sequence hands every later answer to the wrong question, and a yes meant for
// another question would write the drain rule, which decides what an unattended
// agent may change. Off a terminal the offer is never asked (the itd-131
// precedent): a stream of the answer count a repository holding the record is
// asked gets the same questions in the same order, the last of them still gets
// its answer, no record is written, and optional_skipped names the offer.
func TestAPipedInstallStreamIsNotShiftedByTheDrainRuleOffer(t *testing.T) {
	var control []string
	t.Run("control: the repository holds the record", func(t *testing.T) {
		setupHermetic(t)
		repo := installedRepo(t)
		dir := filepath.Join(repo, filepath.FromSlash(drainrule.ADRsRelDir))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		body := "---\nid: adr-2609300000000009\nstatus: accepted\n" + drainrule.ProposalFrontmatter() + "---\n"
		if err := os.WriteFile(filepath.Join(dir, "2609300000000009-drain-rule.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		p := &pipedPrompter{answers: make([]bool, 64)}
		for i := range p.answers {
			p.answers[i] = true
		}
		if _, err := Install(repo, InstallOptions{}, p); err != nil {
			t.Fatal(err)
		}
		control = append(control, p.asked...)
	})
	if len(control) == 0 {
		t.Fatal("the control install asked nothing; the stream has nothing to shift")
	}
	setupHermetic(t)
	repo := installedRepo(t)
	p := &pipedPrompter{answers: make([]bool, len(control))}
	for i := range p.answers {
		p.answers[i] = true
	}
	res, err := Install(repo, InstallOptions{}, p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(p.asked, "\n") != strings.Join(control, "\n") {
		t.Errorf("the piped stream was shifted:\n got %q\nwant %q", p.asked, control)
	}
	for _, q := range p.asked {
		if strings.Contains(q, string(DrainRule)) || strings.Contains(q, drainRuleQuestionTail) {
			t.Errorf("a pipe was asked the drain rule: %q", q)
		}
	}
	if _, err := drainrule.Load(repo); !errors.Is(err, drainrule.ErrUnrecorded) {
		t.Errorf("a piped stream wrote a drain rule: %v", err)
	}
	if !containsString(res.OptionalSkipped, DrainRuleOfferGapID) {
		t.Errorf("optional_skipped = %v, want it to name %s", res.OptionalSkipped, DrainRuleOfferGapID)
	}
}
