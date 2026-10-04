package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/tools"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestTerminalToolConfirmAsksOnlyAPersonAtATerminal is the gh offer's
// confirmation at the remote write (the DQ3 ruling): only an answer typed at
// a terminal is a yes. A piped y, --yes and a prompter that is no terminal all
// decline, and each decline carries the command to run by hand. The question
// shows the explanation and the exact step before anything runs.
func TestTerminalToolConfirmAsksOnlyAPersonAtATerminal(t *testing.T) {
	e := tools.Explain("gh", tools.GitHubSettings)
	var w bytes.Buffer

	piped := &stdinPrompter{r: bufio.NewReader(strings.NewReader("y\n")), w: &w}
	if ans := terminalToolConfirm(piped, false, &w)(e); ans.Yes ||
		!strings.Contains(ans.Why, "no terminal to ask at") || !strings.Contains(ans.Why, "brew install gh") {
		t.Fatalf("a piped y: %+v", ans)
	}
	tty := &stdinPrompter{r: bufio.NewReader(strings.NewReader("y\n")), w: &w, tty: true}
	if ans := terminalToolConfirm(tty, true, &w)(e); ans.Yes ||
		!strings.Contains(ans.Why, "--yes") || !strings.Contains(ans.Why, "brew install gh") {
		t.Fatalf("--yes at a terminal: %+v", ans)
	}
	if ans := terminalToolConfirm(ahoy.RefusingPrompter{}, false, &w)(e); ans.Yes {
		t.Fatal("the refusing prompter installed gh")
	}
	if w.Len() != 0 {
		t.Fatalf("a decline asked a question anyway:\n%s", w.String())
	}

	tty = &stdinPrompter{r: bufio.NewReader(strings.NewReader("n\n")), w: &w, tty: true}
	if ans := terminalToolConfirm(tty, false, &w)(e); ans.Yes {
		t.Fatal("an n typed at a terminal was a yes")
	}
	w.Reset()
	tty = &stdinPrompter{r: bufio.NewReader(strings.NewReader("y\n")), w: &w, tty: true}
	ans := terminalToolConfirm(tty, false, &w)(e)
	if !ans.Yes {
		t.Fatalf("a y typed at a terminal was not a yes: %+v", ans)
	}
	asked := w.String()
	q := strings.Index(asked, "Install gh now by running brew install gh? [y/N]")
	if q < 0 {
		t.Fatalf("the question does not show the exact step:\n%s", asked)
	}
	for _, want := range []string{"what abcd uses it for:", "without it:", "gh auth login"} {
		if i := strings.Index(asked, want); i < 0 || i > q {
			t.Errorf("%q is not shown before the question:\n%s", want, asked)
		}
	}
	if !strings.Contains(asked[q:], "running brew install gh") {
		t.Errorf("the step is not announced as it starts:\n%s", asked)
	}
}

// TestAhoyRemoteApplyNeverInstallsGhOnAScriptedYes is the end-to-end no: a
// piped y and --yes each reach the gh offer and decline it, so the step is
// never attempted (brew's absence would otherwise be the reason), and the
// refusal carries the command to run.
func TestAhoyRemoteApplyNeverInstallsGhOnAScriptedYes(t *testing.T) {
	hermeticEnv(t)
	repo := gittest.NewRepo(t)
	repo.Git("remote", "add", "origin", "https://github.com/example-org/example-repo.git")
	t.Chdir(repo.Root())
	if _, err := runCLIErr(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "agents_md",
		"--oracle-backend", "host-delegated", "--scan-deep", "false"); err != nil {
		t.Fatalf("install: %v", err)
	}
	toolFreePath(t)

	for _, tc := range []struct {
		args []string
		why  string
	}{
		{[]string{"ahoy", "remote", "apply", "--json"}, "no terminal to ask at"},
		{[]string{"ahoy", "remote", "apply", "--yes", "--json"}, "--yes"},
	} {
		out, errOut, err := runCLIPipedStdinSplit(t, "y\ny\n", tc.args...)
		if err == nil {
			t.Fatalf("%v exited zero with gh missing:\n%s", tc.args, out)
		}
		var res ahoy.RemoteResult
		if jerr := json.Unmarshal(out, &res); jerr != nil {
			t.Fatalf("%v: not JSON: %v\n%s\n%s", tc.args, jerr, out, errOut)
		}
		notes := strings.Join(res.Notes, "\n")
		if res.Status != "refused" || !strings.Contains(notes, tc.why) || !strings.Contains(notes, "brew install gh") {
			t.Errorf("%v: status %q, notes lack %q or the step:\n%s", tc.args, res.Status, tc.why, notes)
		}
		if strings.Contains(notes, "is not on PATH, so the step cannot run") {
			t.Errorf("%v: a scripted yes reached the install step:\n%s", tc.args, notes)
		}
		if strings.Contains(string(errOut), "[y/N]") {
			t.Errorf("%v: the install was asked of a pipe:\n%s", tc.args, errOut)
		}
	}
}
