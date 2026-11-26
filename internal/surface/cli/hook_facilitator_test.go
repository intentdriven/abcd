package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// hook_facilitator_test.go — iss-2611211700231699: while the technical
// facilitator is on auto, every prompt carries one line saying what the
// setting means, beside the away block, read as the status line reads the
// setting (the default branch's committed .abcd/config.json and the machine
// layer, never the working tree's copy).

// facilitatorWords are the line's load-bearing phrases.
var facilitatorWords = []string{
	"technical facilitator is on auto",
	"decided and recorded with `abcd decide`",
	"reserves to the person",
}

func hasFacilitatorLine(text string) bool {
	return strings.Contains(text, facilitatorWords[0])
}

// TestPromptRouterFacilitatorAutoLineEveryPrompt: under auto, the line rides
// every prompt of a session, the second as well as the first, with no away
// record, and the --json envelope carries it in text too.
func TestPromptRouterFacilitatorAutoLineEveryPrompt(t *testing.T) {
	root := mergingCheckout(t, `{"facilitator": {"mode": "auto"}}`)
	ownHookBudgets(t, 10*time.Second)
	for _, prompt := range []string{"hello", "and again"} {
		so, stderr, err := runSplit(t, awayPromptPayload("f1", root, prompt), "hook", "prompt-router")
		if err != nil {
			t.Fatalf("prompt-router: %v\n%s", err, stderr)
		}
		out := decodePromptHook(t, so)
		for _, want := range facilitatorWords {
			if !strings.Contains(out.Context, want) {
				t.Errorf("prompt %q: the facilitator line misses %q:\n%s", prompt, want, out.Context)
			}
		}
		if strings.Contains(out.Context, "## AWAY") {
			t.Errorf("prompt %q: no away record holds, yet the away block was sent:\n%s", prompt, out.Context)
		}
	}
	so, _, _ := runSplit(t, awayPromptPayload("f1", root, "third"), "hook", "prompt-router", "--json")
	var v routerView
	if err := json.Unmarshal([]byte(so), &v); err != nil {
		t.Fatalf("--json: %v\n%s", err, so)
	}
	if !hasFacilitatorLine(v.Text) {
		t.Errorf("--json text misses the facilitator line:\n%s", v.Text)
	}
}

// TestPromptRouterFacilitatorLineBesideAway: with the away record holding as
// well, both are sent.
func TestPromptRouterFacilitatorLineBesideAway(t *testing.T) {
	root := mergingCheckout(t, `{"facilitator": {"mode": "auto"}}`)
	gitAt(t, root, "config", "--local", "user.name", "Alice")
	ownHookBudgets(t, 10*time.Second)
	setAway(t)
	so, _, err := runSplit(t, awayPromptPayload("f2", root, "hello"), "hook", "prompt-router")
	if err != nil {
		t.Fatal(err)
	}
	out := decodePromptHook(t, so)
	assertAwayBlock(t, "away and auto", out.Context)
	if !hasFacilitatorLine(out.Context) {
		t.Errorf("away and auto: the facilitator line is missing:\n%s", out.Context)
	}
}

// TestPromptRouterNoFacilitatorLineUnlessAuto: human, no configuration, and
// an auto written only in the working tree send no line: the setting is read
// the way the status line reads it, from the last-fetched default branch and
// the machine layer, so a branch cannot switch the line on.
func TestPromptRouterNoFacilitatorLineUnlessAuto(t *testing.T) {
	for name, c := range map[string]struct{ committed, worktree string }{
		"human":                {committed: `{"facilitator": {"mode": "human"}}`},
		"no config":            {},
		"auto in the worktree": {committed: `{"facilitator": {"mode": "human"}}`, worktree: `{"facilitator": {"mode": "auto"}}`},
	} {
		t.Run(name, func(t *testing.T) {
			root := mergingCheckout(t, c.committed)
			ownHookBudgets(t, 10*time.Second)
			if c.worktree != "" {
				if err := os.WriteFile(filepath.Join(root, ".abcd", "config.json"), []byte(c.worktree), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			so, _, err := runSplit(t, awayPromptPayload("f3", root, "hello"), "hook", "prompt-router")
			if err != nil {
				t.Fatal(err)
			}
			if out := decodePromptHook(t, so); hasFacilitatorLine(out.Context) {
				t.Errorf("the facilitator line was sent:\n%s", out.Context)
			}
		})
	}
}

// TestPromptRouterFacilitatorLineSaysWhoseItIs (itd-2611252049031692
// criterion 20): the line is sent under the resolved auto and not under the
// resolved human, and says whose setting it is. The project's committed value
// stands above the computer's file, so a project's auto over the computer's
// human sends it, and a project's human over the computer's auto does not.
func TestPromptRouterFacilitatorLineSaysWhoseItIs(t *testing.T) {
	const auto, human = `{"facilitator": {"mode": "auto"}}`, `{"facilitator": {"mode": "human"}}`
	for _, c := range []struct {
		name, committed, machine, whose string
	}{
		{"the project's auto over the computer's human", auto, human, "(facilitator.mode: the project enforces auto over this computer's human)"},
		{"the project's auto", auto, "", "(facilitator.mode: the project enforces auto)"},
		{"the computer's auto where the project leaves it to each", "", auto, "(facilitator.mode: left to each contributor, and this computer's file sets auto)"},
		{"the project's human over the computer's auto", human, auto, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := mergingCheckout(t, c.committed)
			if c.machine != "" {
				writeMachineConfig(t, c.machine)
			}
			ownHookBudgets(t, 10*time.Second)
			so, stderr, err := runSplit(t, awayPromptPayload("f4", root, "hello"), "hook", "prompt-router")
			if err != nil {
				t.Fatalf("prompt-router: %v\n%s", err, stderr)
			}
			out := decodePromptHook(t, so)
			if c.whose == "" {
				if hasFacilitatorLine(out.Context) {
					t.Errorf("the facilitator line was sent under the project's human:\n%s", out.Context)
				}
				return
			}
			if !hasFacilitatorLine(out.Context) || !strings.Contains(out.Context, "on auto "+c.whose) {
				t.Errorf("the facilitator line misses whose it is, %q:\n%s", c.whose, out.Context)
			}
		})
	}
}
