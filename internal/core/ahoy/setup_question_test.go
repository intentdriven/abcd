package ahoy

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
)

// TestSetupValueQuestionIsCoresHelp builds a value question the way every
// front door that draws or records one does (spc-2610030911534855, "The
// interviews whose questions abcd fixes"): the key the id, About the
// material, the choices the prompt offers as options with core's meanings,
// the change-later line from ChangeLaterLine, and a decide-later answer whose
// meaning says what core does with no answer.
func TestSetupValueQuestionIsCoresHelp(t *testing.T) {
	h, _ := HelpIn("", "visibility")
	q := SetupValueQuestion(1, "", "visibility", []string{"private", "public"}, "")
	if q.ID != "visibility" || q.Chip != "Setup Q1" {
		t.Fatalf("id %q chip %q", q.ID, q.Chip)
	}
	if len(q.Material) != 1 || q.Material[0].Text != h.About {
		t.Fatalf("material %+v, want About", q.Material)
	}
	if len(q.Options) != 2 || q.Options[1].Value != "public" || q.Options[1].Meaning != h.Meaning("public") {
		t.Fatalf("options %+v", q.Options)
	}
	if q.ChangeLater != h.ChangeLaterLine() || !strings.Contains(q.ChangeLater, "--visibility") {
		t.Fatalf("change later %q, want %q", q.ChangeLater, h.ChangeLaterLine())
	}
	if q.Later.Value != SetupLaterValue || !strings.Contains(q.Later.Meaning, "unset") {
		t.Fatalf("later %+v", q.Later)
	}
	if fs := question.Check(question.Ask{Questions: []question.Question{q}}); len(fs) > 0 {
		t.Fatalf("structural findings: %v", fs)
	}

	// The prompt's choices are the options, in its order: docs_target offers
	// only the values setup writes.
	d := SetupValueQuestion(2, "", "docs_target", docsTargetWritable, docsTargetDefault)
	if len(d.Options) != len(docsTargetWritable) {
		t.Fatalf("docs_target options %+v, want %v", d.Options, docsTargetWritable)
	}
	// A question no flag answers takes its default for no answer, and Later says so.
	a := SetupValueQuestion(3, "", artefactKindKey, []string{"plugin", "binary", "application"}, artefactKindDefault)
	if !strings.Contains(a.Later.Meaning, artefactKindDefault) || a.ChangeLater == "" {
		t.Fatalf("artefact kind later %+v change later %q", a.Later, a.ChangeLater)
	}
	// A key core has no help for is asked bare: no material of the door's own.
	b := SetupValueQuestion(4, "", "no.such.key", []string{"x", "y"}, "x")
	if len(b.Material) != 0 || len(b.Options) != 2 {
		t.Fatalf("bare question %+v", b)
	}
}

// TestSetupConfirmQuestionNamesEveryApproval gives every approval setup asks a
// stable id, so an answers file can answer it, and splits its text into the
// material and the one plain question.
func TestSetupConfirmQuestionNamesEveryApproval(t *testing.T) {
	cases := []struct {
		text, id, ask string
		material      bool
	}{
		{"Adopt this unmanaged repo into abcd?", "adopt", "Adopt this unmanaged repo into abcd?", false},
		{"Apply config-change changes?", "approve.config-change", "Apply config-change changes?", false},
		{"Apply oracle-routing changes?", "approve.oracle-routing", "Apply oracle-routing changes?", false},
		{machineRoutingQuestion(), OracleRoutingMachineGapID, oracleRoutingMachineQuestionTail, true},
		{repoRoutingQuestion(), OracleRoutingRepoGapID, oracleRoutingRepoQuestionTail, true},
		{statusLineOfferQuestion, StatusLineOfferGapID, "Install abcd's status line?", true},
		{drainRuleQuestion(), DrainRuleOfferGapID, drainRuleQuestionTail, true},
		{"Re-founded from abc1234? Link lineage?", "history.lineage", "Re-founded from abc1234? Link lineage?", false},
		{"Commit to this repository as Jan B. Doe, from the pin? (sets user.name and user.email in this repository's .git/config only)",
			"git_identity.establish", "", false},
		{"Change GitHub settings on o/r? (secret scanning on)", "remote.apply", "", false},
	}
	for _, c := range cases {
		q := SetupConfirmQuestion(7, c.text)
		if q.ID != c.id {
			t.Errorf("%.40q: id %q, want %q", c.text, q.ID, c.id)
		}
		if c.ask != "" && q.Ask != c.ask {
			t.Errorf("%s: ask %q, want %q", c.id, q.Ask, c.ask)
		}
		if c.ask == "" && q.Ask != c.text {
			t.Errorf("%s: a text with no question to split off is asked whole; got %q", c.id, q.Ask)
		}
		if (len(q.Material) > 0) != c.material {
			t.Errorf("%s: material %+v", c.id, q.Material)
		}
		if len(q.Options) != 2 || q.Options[0].Value != "yes" || q.Options[1].Value != "no" || q.Later.Value != SetupLaterValue {
			t.Errorf("%s: answers %+v later %+v", c.id, q.Options, q.Later)
		}
		if fs := question.Check(question.Ask{Questions: []question.Question{q}}); len(fs) > 0 {
			t.Errorf("%s: structural findings: %v", c.id, fs)
		}
	}
	// The routing offer is said in counts, one paragraph, not a table.
	m := SetupConfirmQuestion(1, machineRoutingQuestion())
	if m.Material[0].Kind != question.KindParagraph || !strings.Contains(m.Material[0].Text, " agents: ") {
		t.Fatalf("machine routing material %+v", m.Material)
	}
	// A text abcd does not own has no setup id: the door numbers it.
	if q := SetupConfirmQuestion(5, "Change the host on example? (x)"); q.ID != "" {
		t.Fatalf("a foreign text got id %q", q.ID)
	}
}

// TestSetupMachineWideQuestions names the questions whose answers change the
// machine rather than the repository: their record goes to the home.
func TestSetupMachineWideQuestions(t *testing.T) {
	for _, id := range []string{StatusLineOfferGapID, elementPromptPrefix + "repo", OracleRoutingMachineGapID} {
		if !SetupMachineWide(id) {
			t.Errorf("%s is machine-wide", id)
		}
	}
	for _, id := range []string{"visibility", "adopt", "approve.status-line", OracleRoutingRepoGapID, DrainRuleOfferGapID} {
		if SetupMachineWide(id) {
			t.Errorf("%s is the repository's", id)
		}
	}
}
