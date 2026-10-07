package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// labCheckout is a one-commit repository under a temp HOME, with the process
// standing in it, so the lab store the verb creates is the test's own. It is
// not abcd's own repository unless asAbcd says so.
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

// asAbcd makes the checkout stand in for abcd's own repository, whose labs the
// dual-binary gate holds.
func asAbcd(t *testing.T, repo string) {
	t.Helper()
	t.Cleanup(lab.SetAbcdRootCommitForTest(gitutil.RootCommit(repo)))
}

// The verb is wired: every sub-verb executes from the CLI, a gate refusal exits
// 1 with its artefact rendered, and no output carries the home path.
func TestLabVerbRunsEverySubverbFromTheCLI(t *testing.T) {
	home, repo := labCheckout(t)
	asAbcd(t, repo)

	code, out, errOut := runLab(t, "lab", "--json", "mint", "does", "the", "procedure", "transfer?")
	if code != 0 {
		t.Fatalf("lab mint exit %d: %s%s", code, out, errOut)
	}
	var m struct {
		ID       string `json:"id"`
		Question string `json:"question"`
		Home     string `json:"home"`
	}
	if err := json.Unmarshal([]byte(out), &m); err != nil || m.Question != "does the procedure transfer?" || !strings.HasPrefix(m.Home, "~/.abcd.noindex/lab/") {
		t.Fatalf("mint JSON = %s (%v)", out, err)
	}

	code, out, _ = runLab(t, "lab")
	if code != 0 || !strings.Contains(out, m.ID) || !strings.Contains(out, "1 lab in ~/.abcd.noindex/lab/") {
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
	if code != 0 || !strings.Contains(out, "written ~/.abcd.noindex/lab/") || !strings.Contains(out, "halted by its preflight") {
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

// A lab of a repository that is not abcd's own passes its preflight, the
// dual-binary group shown not applicable in the text and the JSON alike, and
// mint's next steps never ask for a bin/abcd it cannot build.
func TestLabPreflightShowsTheDualBinaryGroupNotApplicableOutsideAbcd(t *testing.T) {
	_, _ = labCheckout(t)
	code, out, errOut := runLab(t, "lab", "mint", "does", "the", "control", "panel", "load?")
	if code != 0 || strings.Contains(out, "bin/abcd") {
		t.Fatalf("lab mint (exit %d) names bin/abcd outside abcd's own repository:\n%s%s", code, out, errOut)
	}
	id := strings.Fields(out)[1] // "minted <lab-id> at pin <sha>"
	code, out, _ = runLab(t, "lab", "preflight", id)
	if code != 0 || !strings.Contains(out, ": PASSED") || !strings.Contains(out, "n/a   binary.work") ||
		!strings.Contains(out, "not applicable: this repository is not abcd's own") {
		t.Errorf("preflight outside abcd (exit %d):\n%s", code, out)
	}
	code, out, _ = runLab(t, "lab", "--json", "preflight", id)
	var res struct {
		Passed bool `json:"passed"`
		Checks []struct {
			ID            string `json:"id"`
			OK            bool   `json:"ok"`
			NotApplicable bool   `json:"not_applicable"`
		} `json:"checks"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &res) != nil || !res.Passed {
		t.Fatalf("preflight --json (exit %d): %s", code, out)
	}
	for _, c := range res.Checks {
		if strings.HasPrefix(c.ID, "binary.") != c.NotApplicable || !c.OK {
			t.Errorf("check %+v: want the binary checks, and only they, passing as not applicable", c)
		}
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
