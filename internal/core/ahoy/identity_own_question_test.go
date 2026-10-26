package ahoy

import (
	"strings"
	"testing"
)

// TestIdentityQuestionStandsWithoutTheSettingsApproval is iss-2610240824005655:
// the commit identity is a question only the person can answer, so it is asked
// on its own, naming the identity and where it is written, whatever the answer
// to the settings change was. A no to the settings used to mean the identity
// question was never asked.
func TestIdentityQuestionStandsWithoutTheSettingsApproval(t *testing.T) {
	clearIdentityEnv(t)
	dir := idGitRepo(t, "Test User", "test@example.com")
	idWritePin(t, dir, pinAlex)
	p := &identityPrompter{terminal: true, answer: true}
	a := establishCtx(dir, p, MismatchGapID)
	a.approved = map[GapCategory]bool{} // the settings change declined

	a.stepGitIdentity()

	if len(p.asked) != 1 {
		t.Fatalf("asked %d questions, want the one identity question: %q", len(p.asked), p.asked)
	}
	q := p.asked[0]
	for _, want := range []string{"Alex Reppel <alex@example.com>", ".git/config"} {
		if !strings.Contains(q, want) {
			t.Errorf("the identity question does not name %q: %q", want, q)
		}
	}
	if got := localConfig(t, dir, "user.name"); got != "Alex Reppel" {
		t.Errorf("user.name = %q after a yes, want the pinned name", got)
	}
}

// TestSettingsApprovalDoesNotListTheIdentity: the identity is its own question,
// so the settings change neither lists it nor is asked for it alone.
func TestSettingsApprovalDoesNotListTheIdentity(t *testing.T) {
	gaps := []Gap{{ID: MismatchGapID, Category: ConfigChange, Title: "git identity differs from the pin", Required: true, Resolvable: true}}
	p := &identityPrompter{terminal: true, answer: true}
	resolveApproval(gaps, InstallOptions{}, nil, p)
	for _, q := range p.asked {
		if strings.Contains(q, "git identity differs") {
			t.Errorf("the settings approval lists the identity, which is its own question: %q", q)
		}
	}
	if len(p.asked) != 0 {
		t.Errorf("an approval was asked for the identity alone: %q", p.asked)
	}
}
