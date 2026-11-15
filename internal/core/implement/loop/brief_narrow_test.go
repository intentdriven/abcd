package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement"
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
		"issue": string(renderIssueBrief(State{RunID: "run-1", Key: "iss-2609300000000101"}, lane, "lane-dir", issueBriefSources{
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

// briefsForTest renders the intent brief and the issue brief for a fixture lane.
func briefsForTest() map[string]string {
	st := State{RunID: "run-1", Key: "itd-1", Intent: "itd-1", Spec: "spc-1"}
	lane := Lane{ID: "lane-1", Branch: "build/x", BaseSHA: strings.Repeat("a", 40), Worktree: "wt", SpecStep: 1}
	return map[string]string{
		"intent": string(renderBrief(st, lane, "lane-dir", briefSources{
			intentPath: "intent.md", specPath: "spec.md", conventionsFrom: "the whole file",
		})),
		"issue": string(renderIssueBrief(State{RunID: "run-1", Key: "iss-2609300000000101"}, lane, "lane-dir", issueBriefSources{
			issuePath: "issue.md", conventionsFrom: "the whole file",
		})),
	}
}

// TestEveryBriefKeepsARaceRunWhereThePreflightHasNone: a brief tells its
// implementer to leave -race out of the narrow form, so where the repository's
// pre-push gate runs no race detector it spells a gated race run instead of
// letting the race tests drop: a plain gated wait, never marked narrow, since
// the gate reads any race run as a whole run.
func TestEveryBriefKeepsARaceRunWhereThePreflightHasNone(t *testing.T) {
	for name, brief := range briefsForTest() {
		var race bool
		for _, m := range narrowCommandLine.FindAllStringSubmatch(brief, -1) {
			words := strings.Fields(m[1])
			sep := slices.Index(words, "--")
			if sep < 0 || !slices.Contains(words[sep+1:], "-race") {
				continue
			}
			race = true
			if slices.Contains(words[:sep], machineload.NarrowMark) || !slices.Contains(words[:sep], "--wait") {
				t.Errorf("the %s brief's race run is not a plain gated wait: %s", name, m[1])
			}
		}
		if !race {
			t.Errorf("the %s brief spells no race run for a repository whose pre-push gate has no race stage:\n%s", name, brief)
		}
		if flat := strings.Join(strings.Fields(brief), " "); !strings.Contains(flat, "runs none and the conventions ask for race tests") {
			t.Errorf("the %s brief does not say when the race run applies", name)
		}
	}
}

// TestTheNarrowFlagsABriefNamesReadAsNarrow: every flag a brief lists as
// narrow keeps a run narrow to the gate, and the brief names each one.
func TestTheNarrowFlagsABriefNamesReadAsNarrow(t *testing.T) {
	brief := briefsForTest()["intent"]
	for _, f := range narrowTestFlags {
		cmd := []string{"go", "test", f}
		if f == "-run" || f == "-skip" || f == "-timeout" {
			cmd = append(cmd, "1m")
		}
		cmd = append(cmd, "./internal/core/x")
		if !machineload.IsNarrowCommand(cmd) {
			t.Errorf("the brief names %s as narrow, but the gate reads %v as a whole run", f, cmd)
		}
		if !strings.Contains(brief, "`"+f+"`") {
			t.Errorf("the brief does not name %s", f)
		}
	}
}

// v0162NarrowFlags are the flags v0.16.2's gate reads as narrow, without their
// dashes, copied from its machineload narrowBoolFlags and narrowValueFlags (the
// value flag count only as count=1). They are copied, not read, because the
// released binary is what a lane on another repository runs.
var v0162NarrowFlags = []string{"v", "short", "failfast", "json", "run", "skip", "timeout", "count=1"}

// TestTheNarrowFlagsABriefNamesReadAsNarrowToV0162: the brief tells a lane in
// any repository which flags keep a run narrow, and that lane runs whichever
// abcd its machine has installed, v0.16.2 among them. So every flag the brief
// lists as narrow must be one v0.16.2's gate reads as narrow too: a flag only a
// newer gate reads, -trimpath, is named apart, and one added to the list would
// make a run on v0.16.2 wait behind every preflight on the machine.
func TestTheNarrowFlagsABriefNamesReadAsNarrowToV0162(t *testing.T) {
	for _, f := range narrowTestFlags {
		if !slices.Contains(v0162NarrowFlags, strings.TrimLeft(f, "-")) {
			t.Errorf("the brief names %s as narrow, but v0.16.2's gate reads it as a whole run", f)
		}
	}
}

// TestEveryBriefStatesTheNarrowGiveWay: a narrow run gives way to an earlier
// whole run that is overdue, for implement.WaitNarrowYieldCap at most, so a
// brief that told its implementer the narrow form never waits behind a whole
// preflight would mislead it about a wait it will see. Each brief gives the cap
// the gate holds to, and says which gate reads -trimpath as narrow. It says so
// without the word "aside", which a brief never holds: a restart saves a lane's
// edits aside, and the brief never names that (restart.go).
func TestEveryBriefStatesTheNarrowGiveWay(t *testing.T) {
	giveWay := fmt.Sprintf("gives way to a whole preflight for %g minutes at most", implement.WaitNarrowYieldCap.Minutes())
	for name, brief := range briefsForTest() {
		flat := strings.Join(strings.Fields(brief), " ")
		if strings.Contains(flat, "never behind a whole preflight") {
			t.Errorf("the %s brief says a narrow run never waits behind a whole preflight", name)
		}
		if strings.Contains(brief, "aside") {
			t.Errorf("the %s brief names an aside", name)
		}
		for _, want := range []string{
			giveWay,
			"a gate newer than v0.16.2 reads `-trimpath` as narrow too",
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
