package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The live brief session questions at the CLI's doors (spc-2610141709430146
// step 6).

// TestAhoyDetectionCarriesTheSessionsQuestions: while the live-session
// question is open, the detection the plugin page reads carries both
// questions as the host's question tool takes them, each alone, with its
// current setting and change-later line; --live-sessions answers it, and an
// answered question is no longer carried.
func TestAhoyDetectionCarriesTheSessionsQuestions(t *testing.T) {
	hermeticEnv(t)
	repo := gittest.NewRepo(t).Root()
	t.Chdir(repo)
	if _, err := runCLIStdinErr(t, "", "ahoy", "install", "--yes", "--adopt", "--visibility", "private",
		"--docs-target", "skip", "--scan-deep", "false", "--json"); err != nil {
		t.Fatalf("install: %v", err)
	}
	var out struct {
		Sessions     *ahoy.SessionsView  `json:"sessions"`
		SessionsTool []*hostQuestionCall `json:"sessions_tool"`
		Skipped      []string            `json:"optional_skipped"`
	}
	raw := runCLI(t, "ahoy", "--json")
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%v\n%s", err, raw)
	}
	if out.Sessions == nil || len(out.SessionsTool) != 2 {
		t.Fatalf("no live-session questions while the question is open:\n%s", raw)
	}
	for i, call := range out.SessionsTool {
		if len(call.Questions) != 1 {
			t.Fatalf("call %d holds %d questions", i+1, len(call.Questions))
		}
		q := call.Questions[0]
		if !strings.Contains(q.Question, question.Default.NowPrefix) || !strings.Contains(q.Question, "--live-sessions") {
			t.Errorf("call %d lacks its current setting or change-later line:\n%s", i+1, q.Question)
		}
		last := q.Options[len(q.Options)-1].Label
		if last != question.Default.LaterLabels[0] {
			t.Errorf("call %d's last option is %q", i+1, last)
		}
	}
	if !strings.Contains(out.SessionsTool[0].Questions[0].Question, "Set up live brief sessions?") ||
		!strings.Contains(out.SessionsTool[1].Questions[0].Question, "Where may sessions run?") {
		t.Errorf("the questions are not the two setup asks:\n%s", raw)
	}

	text, err := runCLIStdinErr(t, "", "ahoy", "install", "--yes", "--adopt")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, text)
	}
	if !strings.Contains(string(text), "--live-sessions "+ahoy.LiveSessionsChoices) {
		t.Errorf("a run that skipped the question does not name the flag:\n%s", text)
	}

	if got, err := runCLIStdinErr(t, "", "ahoy", "install", "--yes", "--adopt", "--live-sessions", "everywhere"); exitCode(err) != 2 ||
		!strings.Contains(err.Error(), "local, cloud, both or none") {
		t.Errorf("an unknown answer: exit %d, %v\n%s", exitCode(err), err, got)
	}
	if got, err := runCLIStdinErr(t, "", "ahoy", "install", "--yes", "--adopt", "--live-sessions", "both"); err != nil ||
		!strings.Contains(string(got), "sessions.live: not set -> both") {
		t.Fatalf("install --live-sessions both: %v\n%s", err, got)
	}
	cfg, err := os.ReadFile(filepath.Join(repo, ".abcd", "config.json"))
	if err != nil || !strings.Contains(string(cfg), `"live": "both"`) {
		t.Fatalf("config.json does not record the answer (%v):\n%s", err, cfg)
	}
	out.Sessions, out.SessionsTool = nil, nil
	raw = runCLI(t, "ahoy", "--json")
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.Sessions != nil || out.SessionsTool != nil {
		t.Errorf("an answered question is still carried:\n%s", raw)
	}
}
