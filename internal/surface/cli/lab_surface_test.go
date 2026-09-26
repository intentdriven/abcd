package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// labCheckout is a one-commit repository under a temp HOME, with the process
// standing in it, so the lab store the verb creates is the test's own.
func labCheckout(t *testing.T) (home, repo string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	repo = filepath.Join(home, "repo")
	gitInitAt(t, repo)
	writeRel(t, repo, "README.md", "root\n")
	gitCmd(t, repo, "add", "-A")
	gitCommit(t, repo, "commit", "-q", "-m", "root")
	t.Chdir(repo)
	return home, repo
}

func runLab(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb)
	return code, out.String(), errb.String()
}

// The verb is wired: every sub-verb executes from the CLI, a gate refusal exits
// 1 with its artefact rendered, and no output carries the home path.
func TestLabVerbRunsEverySubverbFromTheCLI(t *testing.T) {
	home, repo := labCheckout(t)

	code, out, errOut := runLab(t, "lab", "--json", "mint", "does", "the", "procedure", "transfer?")
	if code != 0 {
		t.Fatalf("lab mint exit %d: %s%s", code, out, errOut)
	}
	var m struct {
		ID       string `json:"id"`
		Question string `json:"question"`
		Home     string `json:"home"`
	}
	if err := json.Unmarshal([]byte(out), &m); err != nil || m.Question != "does the procedure transfer?" || !strings.HasPrefix(m.Home, "~/.abcd/lab/") {
		t.Fatalf("mint JSON = %s (%v)", out, err)
	}

	code, out, _ = runLab(t, "lab")
	if code != 0 || !strings.Contains(out, m.ID) || !strings.Contains(out, "1 lab in ~/.abcd/lab/") {
		t.Errorf("bare lab (exit %d):\n%s", code, out)
	}

	code, out, _ = runLab(t, "lab", "preflight", m.ID)
	if code != 1 || !strings.Contains(out, "preflight "+m.ID+": HALTED") || !strings.Contains(out, "FAIL  binary.work") ||
		!strings.Contains(out, "recorded as F-1") {
		t.Errorf("preflight on a lab with no work binary (exit %d):\n%s", code, out)
	}
	code, _, errOut = runLab(t, "lab", "record", m.ID, "p1")
	if code != 1 || !strings.Contains(errOut, "halted") {
		t.Errorf("record on a halted lab: exit %d, stderr %q", code, errOut)
	}
	code, out, _ = runLab(t, "lab", "sweep", m.ID)
	if code != 0 || !strings.Contains(out, "PASSED") {
		t.Errorf("sweep (exit %d):\n%s", code, out)
	}
	code, out, _ = runLab(t, "lab", "harvest", m.ID)
	if code != 0 || !strings.Contains(out, "written ~/.abcd/lab/") || !strings.Contains(out, "halted by its preflight") {
		t.Errorf("harvest (exit %d):\n%s", code, out)
	}

	for _, args := range [][]string{{"lab"}, {"lab", "--json"}, {"lab", "--json", "preflight", m.ID}, {"lab", "--json", "harvest", m.ID}} {
		_, out, errOut := runLab(t, args...)
		if strings.Contains(out+errOut, home) {
			t.Errorf("%v leaked the home path:\n%s%s", args, out, errOut)
		}
	}
	if got := gitCmd(t, repo, "status", "--porcelain", "--ignored"); got != "" {
		t.Errorf("the lab verbs wrote into the repository: %q", got)
	}
}

func TestLabVerbRefusesARequestItCannotServe(t *testing.T) {
	_, _ = labCheckout(t)
	for _, args := range [][]string{
		{"lab", "preflight", "../../etc"},
		{"lab", "sweep", "lab-260925101112-abcdef0"},
		{"lab", "mint", "  "},
		{"lab", "mint", "q?", "--pin", "no-such-rev"},
	} {
		if code, _, errOut := runLab(t, args...); code != 2 {
			t.Errorf("%v: exit %d (%s), want 2", args, code, errOut)
		}
	}
	t.Chdir(t.TempDir())
	if code, _, errOut := runLab(t, "lab"); code != 2 || !strings.Contains(errOut, "not inside a git repository") {
		t.Errorf("lab outside a checkout: exit %d, %q", code, errOut)
	}
}
