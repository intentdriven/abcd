package loop

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/machineload"
)

// narrowCommandLine finds the gated test command a brief spells, in a code span.
var narrowCommandLine = regexp.MustCompile("`(abcd implement load [^`]*)`")

// TestEveryBriefSpellsTheNarrowTestForm: an implementer's scoped test waits
// behind every whole preflight on the machine unless it runs in the gate's
// narrow form, so the intent brief (and the fix round, which renders it) and
// the issue brief each spell that form outright, the command reads as narrow to
// the gate that classes it, and the brief says why -race is left out and how
// many packages one run names.
func TestEveryBriefSpellsTheNarrowTestForm(t *testing.T) {
	st := State{RunID: "run-1", Key: "itd-1", Intent: "itd-1", Spec: "spc-1"}
	lane := Lane{ID: "lane-1", Branch: "build/x", BaseSHA: strings.Repeat("a", 40), Worktree: "wt", SpecStep: 1}
	briefs := map[string]string{
		"intent": string(renderBrief(st, lane, "lane-dir", briefSources{
			intentPath: "intent.md", specPath: "spec.md", conventionsFrom: "the whole file",
		})),
		"issue": string(renderIssueBrief(State{RunID: "run-1", Key: "iss-2609300000000101"}, lane, "repo-root", "lane-dir", issueBriefSources{
			issuePath: "issue.md", conventionsFrom: "the whole file",
		})),
	}
	for name, brief := range briefs {
		m := narrowCommandLine.FindStringSubmatch(brief)
		if m == nil {
			t.Errorf("the %s brief spells no gated test command:\n%s", name, brief)
			continue
		}
		words := strings.Fields(strings.ReplaceAll(m[1], "<package>", "internal/core/x"))
		sep := slices.Index(words, "--")
		if sep < 0 || !slices.Contains(words[:sep], "--wait") || !slices.Contains(words[:sep], machineload.NarrowMark) {
			t.Errorf("the %s brief's command waits without the narrow mark: %s", name, m[1])
			continue
		}
		if !machineload.IsNarrowCommand(words[sep+1:]) {
			t.Errorf("the %s brief's command is a whole run to the gate: %s", name, m[1])
		}
		flat := strings.Join(strings.Fields(brief), " ")
		for _, want := range []string{
			"use theirs, still without `-race`",
			"Leave out `-race`",
			"at most 3 packages",
			"waits behind every preflight",
		} {
			if !strings.Contains(flat, want) {
				t.Errorf("the %s brief does not say %q:\n%s", name, want, brief)
			}
		}
	}
}

// conventionsNarrowSpan finds a code span that spells a narrow gated run.
var conventionsNarrowSpan = regexp.MustCompile("`([^`]*implement load[^`]*" + regexp.QuoteMeta(machineload.NarrowMark) + "[^`]*)`")

// TestTheConventionsNarrowFormsReadAsNarrow: the brief defers to the narrow
// form this repository's conventions spell, so each one they spell (AGENTS.md,
// which the brief quotes, and the rules the loader injects) must read as narrow
// to the gate: one carrying -race, as the drive lane's line once did, waits
// behind every preflight on the machine however it is marked.
func TestTheConventionsNarrowFormsReadAsNarrow(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, name := range []string{ConventionsFile, filepath.Join(".abcd", "rules.json")} {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range conventionsNarrowSpan.FindAllStringSubmatch(string(raw), -1) {
			words := strings.Fields(strings.ReplaceAll(m[1], `\n`, " "))
			sep := slices.Index(words, "--")
			if sep < 0 {
				continue
			}
			found++
			cmd := append(slices.Clone(words[sep+1:]), "./internal/core/x")
			if !machineload.IsNarrowCommand(cmd) {
				t.Errorf("%s spells a narrow form the gate reads as a whole run: %s", name, m[1])
			}
		}
	}
	if found == 0 {
		t.Fatal("no narrow form found in the conventions: the test reads nothing")
	}
}
