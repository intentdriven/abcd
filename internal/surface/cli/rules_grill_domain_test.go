package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGrillDomainCarriesQuestionVisibilityAndAddresseeRules pins this
// repository's GRILL domain to the two rules the product thinker's
// 2026-09-29 captures asked for: the example that makes a question answerable
// sits in the question and in each option's preview (iss-2609291925134691),
// and the addressee is classified and the mode label set before every question
// (iss-2609291925149138). The count is pinned too, so the ten rules the domain
// already carried are added to, never rewritten away: a change to the count is
// a deliberate edit to this pin.
func TestGrillDomainCarriesQuestionVisibilityAndAddresseeRules(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(testRepoRoot(), ".abcd", "rules.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rules struct {
		Domains map[string]struct {
			Rules []string `json:"rules"`
		} `json:"domains"`
	}
	if err := json.Unmarshal(data, &rules); err != nil {
		t.Fatalf("parse .abcd/rules.json: %v", err)
	}
	dom, ok := rules.Domains["GRILL"]
	if !ok {
		t.Fatal(".abcd/rules.json declares no GRILL domain")
	}
	const wantCount = 12
	if len(dom.Rules) != wantCount {
		t.Errorf("GRILL carries %d rules, want %d (ten standing rules plus the two from the 2026-09-29 captures)", len(dom.Rules), wantCount)
	}
	for _, want := range []struct {
		issue  string
		phrase []string
	}{
		{"iss-2609291925134691", []string{"IN the question text", "each option's preview", "invisible while the question shows"}},
		{"iss-2609291925149138", []string{"addressee", "FIRST", "mode verb", "mixed interview"}},
	} {
		found := false
		for _, r := range dom.Rules {
			all := true
			for _, p := range want.phrase {
				if !strings.Contains(r, p) {
					all = false
					break
				}
			}
			if all {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GRILL has no rule carrying %q (%s)", want.phrase, want.issue)
		}
	}
}

// TestGrillQuotingRuleSaysProseIsInvisibleWhileTheQuestionShows holds the
// quoting rule (what the human accepts, edits or strikes is quoted in the
// question) to the same claim the visibility rule makes: prose written between
// tool calls is invisible while the question shows. A softer "does not
// reliably reach" beside the firmer claim reads as two rules disagreeing about
// one fact (review of iss-2609291925134691, MINOR 4).
func TestGrillQuotingRuleSaysProseIsInvisibleWhileTheQuestionShows(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(testRepoRoot(), ".abcd", "rules.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rules struct {
		Domains map[string]struct {
			Rules []string `json:"rules"`
		} `json:"domains"`
	}
	if err := json.Unmarshal(data, &rules); err != nil {
		t.Fatalf("parse .abcd/rules.json: %v", err)
	}
	found := false
	for _, r := range rules.Domains["GRILL"].Rules {
		if strings.Contains(r, "does not reliably reach") {
			t.Errorf("a GRILL rule still says prose %q; the domain states it as invisible while the question shows: %s", "does not reliably reach", r)
		}
		if strings.Contains(r, "quoted IN the question itself") {
			found = true
			if !strings.Contains(r, "invisible while the question shows") {
				t.Errorf("the GRILL quoting rule does not say prose is %q: %s", "invisible while the question shows", r)
			}
		}
	}
	if !found {
		t.Error("GRILL carries no rule that quotes what the human is asked to accept IN the question itself")
	}
}

// TestIntentInterviewPageCarriesGrillVisibilityAndAddresseeLines holds the
// planning interview's "How every question is asked" paragraph in
// commands/intent.md to the two GRILL rules the product thinker's 2026-09-29
// captures asked for, since that page ships to every repository abcd is
// installed in and is where an adopter's agent reads how to ask: the example
// sits in the question text and in each option's preview
// (iss-2609291925134691), and a mixed interview re-sets the addressee per
// question (iss-2609291925149138).
func TestIntentInterviewPageCarriesGrillVisibilityAndAddresseeLines(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(testRepoRoot(), "commands", "intent.md"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(data)
	start := strings.Index(page, "**How every question is asked")
	if start < 0 {
		t.Fatal("commands/intent.md has no \"How every question is asked\" paragraph")
	}
	end := strings.Index(page[start:], "\n\n")
	if end < 0 {
		end = len(page) - start
	}
	para := strings.Join(strings.Fields(page[start:start+end]), " ")
	for _, want := range []struct {
		issue, phrase string
	}{
		{"iss-2609291925134691", "the example that makes a question answerable goes in the question text"},
		{"iss-2609291925134691", "each option's preview"},
		{"iss-2609291925134691", "invisible while a question shows"},
		{"iss-2609291925149138", "classify each question's addressee first"},
		{"iss-2609291925149138", "a mixed interview re-sets it per question"},
	} {
		if !strings.Contains(para, want.phrase) {
			t.Errorf("the interview paragraph does not say %q (%s)", want.phrase, want.issue)
		}
	}
	if strings.Contains(para, "does not reliably reach") {
		t.Error("the interview paragraph still says prose \"does not reliably reach\" the human; GRILL states it as invisible while a question shows")
	}
}
