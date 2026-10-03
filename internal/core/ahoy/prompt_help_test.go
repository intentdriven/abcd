package ahoy

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// choiceRecordingPrompter approves every confirm, answers the visibility
// question private (so the conditional deep-scan question is reached), takes
// the default everywhere else, and keeps every value question it was asked
// together with the choices it was offered.
type choiceRecordingPrompter struct {
	asked map[string][]string
}

func (p *choiceRecordingPrompter) Confirm(string) bool { return true }

func (p *choiceRecordingPrompter) Prompt(key string, choices []string, def string) string {
	p.asked[key] = append([]string(nil), choices...)
	if key == "visibility" {
		return "private"
	}
	return def
}

// TestEveryInstallQuestionCarriesPlainLanguageHelp is iss-163's detector: a
// question core asks with bare enum values leaves the front door to invent
// what each answer means, and an invented description can be wrong or
// circular. It drives a real first install that reaches every value question
// core has — the three required configuration values, the conditional
// deep-scan question, the house-style question and the status-line element
// switches — and holds each one to canonical help: what is being decided, and
// a meaning for every choice offered, that is more than the value restated.
func TestEveryInstallQuestionCarriesPlainLanguageHelp(t *testing.T) {
	setupHermetic(t)
	harnessFixture(t, harnessSettingsWith(""))
	// Deep scanning is only asked about when trufflehog is on PATH; keep the
	// rest of PATH so git still resolves.
	th := t.TempDir()
	if err := os.WriteFile(filepath.Join(th, "trufflehog"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", th+string(os.PathListSeparator)+os.Getenv("PATH"))
	repo := t.TempDir()
	idMustGit(t, repo, "init")

	p := &choiceRecordingPrompter{asked: map[string][]string{}}
	if _, err := Install(repo, InstallOptions{}, p); err != nil {
		t.Fatal(err)
	}

	// The drive must actually reach every family of question, or a passing run
	// would say nothing about the one it skipped. The oracle question is asked
	// only once a second answer has an adapter (iss-2610031236155833); until
	// then a first install records host-delegated without asking it.
	wants := []string{"visibility", "docs_target", "scan_deep", emDashPromptKey, elementPromptPrefix + "repo"}
	if oracleBackendAsked() {
		wants = append(wants, "oracle_backend")
	} else if _, asked := p.asked["oracle_backend"]; asked {
		t.Errorf("the install asked oracle_backend while only host-delegated has an adapter")
	}
	for _, want := range wants {
		if _, ok := p.asked[want]; !ok {
			keys := make([]string, 0, len(p.asked))
			for k := range p.asked {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			t.Fatalf("the install never asked %q, so its help is untested; asked %v", want, keys)
		}
	}

	for key, choices := range p.asked {
		h, ok := helpFor(key)
		if !ok {
			t.Errorf("%s: no canonical help, so a front door must invent what the question means", key)
			continue
		}
		if h.Key != key {
			t.Errorf("%s: help is keyed %q", key, h.Key)
		}
		if len(strings.Fields(h.About)) < 8 {
			t.Errorf("%s: About %q does not say what is being decided", key, h.About)
		}
		if len(h.Choices) != len(choices) {
			t.Errorf("%s: help explains %d choices, the question offers %d (%v)", key, len(h.Choices), len(choices), choices)
		}
		for _, c := range choices {
			m := h.Meaning(c)
			if len(strings.Fields(m)) < 5 {
				t.Errorf("%s=%s: meaning %q is missing or too thin to explain the answer", key, c, m)
			}
			if strings.EqualFold(strings.TrimSpace(m), c) {
				t.Errorf("%s=%s: meaning only restates the value", key, c)
			}
		}
	}
}

// TestOracleHelpDefinesTheOracleAndItsCosts pins the specific gap iss-163
// names: the backend question must say what an oracle is, and every answer
// must state what it asks of the person (keys, tools, cost), not only its name.
func TestOracleHelpDefinesTheOracleAndItsCosts(t *testing.T) {
	h, ok := helpFor("oracle_backend")
	if !ok {
		t.Fatal("no help for oracle_backend")
	}
	if !strings.Contains(h.About, "oracle") || !strings.Contains(h.About, "AI model") {
		t.Errorf("About does not define an oracle: %q", h.About)
	}
	for _, c := range oracleBackendChoices {
		m := h.Meaning(c)
		if !strings.Contains(m, "cost") && !strings.Contains(m, "billed") && !strings.Contains(m, "key") {
			t.Errorf("%s: meaning states no consequence (cost, credentials): %q", c, m)
		}
	}
}

// TestHelpForUnknownKeyIsAbsent keeps the lookup honest: a key core does not
// ask about has no help, so a front door renders nothing rather than a guess.
func TestHelpForUnknownKeyIsAbsent(t *testing.T) {
	if _, ok := helpFor("no_such_question"); ok {
		t.Fatal("helpFor invented help for an unknown key")
	}
	if _, ok := helpFor(elementPromptPrefix + "no_such_element"); ok {
		t.Fatal("helpFor invented help for an unknown status-line element")
	}
}

// TestHelpInShowsTheTrackedCaveatOnlyWhereItApplies holds the visibility
// question to the repository it is asked in (iss-2610031236155833): public's
// caveat, that git cannot hide the records it already tracks under .abcd/, is
// part of the help only where .abcd/ holds tracked files, the case in which the
// install narrows the public block. Elsewhere it would describe a repository
// the person does not have, at the cost of rows the question cannot spare.
func TestHelpInShowsTheTrackedCaveatOnlyWhereItApplies(t *testing.T) {
	base, ok := helpFor("visibility")
	if !ok {
		t.Fatal("no help for visibility")
	}
	if strings.Contains(base.Meaning("public"), visibilityTrackedCaveat) {
		t.Fatalf("the repository-independent help carries the tracked caveat: %q", base.Meaning("public"))
	}

	untracked := gittest.NewRepo(t)
	h, ok := HelpIn(untracked.Root(), "visibility")
	if !ok {
		t.Fatal("HelpIn has no help for visibility")
	}
	if strings.Contains(h.Meaning("public"), visibilityTrackedCaveat) {
		t.Errorf("a repository whose .abcd/ holds nothing tracked is shown the caveat: %q", h.Meaning("public"))
	}

	tracked := gittest.NewRepo(t)
	tracked.Write(".abcd/work/DECISIONS.md", "- a decision\n")
	tracked.Commit("record tier")
	h, ok = HelpIn(tracked.Root(), "visibility")
	if !ok {
		t.Fatal("HelpIn has no help for visibility")
	}
	if !strings.HasSuffix(h.Meaning("public"), " "+visibilityTrackedCaveat) {
		t.Errorf("a repository whose .abcd/ holds tracked records is not shown the caveat: %q", h.Meaning("public"))
	}
	// The caveat says what the narrowed block still ignores, so it names every
	// entry the install writes there beyond .abcd/'s own scratch (the memory/
	// fence): saying less than the write does reads as negating it.
	entries, narrowed := effectiveVisibilityEntries(tracked.Root(), "public")
	if !narrowed {
		t.Fatal("the tracked repository's public block was not narrowed")
	}
	for _, e := range entries {
		name := strings.TrimPrefix(e, "/")
		if strings.HasPrefix(name, ".abcd/") {
			continue
		}
		if !strings.Contains(visibilityTrackedCaveat, name) {
			t.Errorf("the narrowed block keeps %s ignored, but the caveat does not say so: %q", e, visibilityTrackedCaveat)
		}
	}
	if h.Meaning("private") != base.Meaning("private") || h.About != base.About || h.Flag != base.Flag {
		t.Errorf("the caveat changed more than public's meaning: %+v", h)
	}
	if again, _ := helpFor("visibility"); again.Meaning("public") != base.Meaning("public") {
		t.Errorf("HelpIn wrote its variant into the canonical help: %q", again.Meaning("public"))
	}

	// Every other question reads the same in every repository.
	for _, key := range []string{"docs_target", "scan_deep", artefactKindKey, elementPromptPrefix + "repo", "no_such_question"} {
		want, wantOK := helpFor(key)
		got, gotOK := HelpIn(tracked.Root(), key)
		if gotOK != wantOK || got.About != want.About || got.ChangeLaterLine() != want.ChangeLaterLine() {
			t.Errorf("%s: HelpIn differs from helpFor", key)
		}
	}
}
