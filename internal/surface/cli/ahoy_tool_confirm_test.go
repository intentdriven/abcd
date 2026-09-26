package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/tools"
)

func gitleaksExplained() tools.Explanation {
	return tools.Explain("gitleaks", tools.TranscriptScan)
}

// TestToolConfirmAsksOnlyAtATerminal is itd-63's non-interactive rule at the
// CLI: a piped "y" (the `yes | abcd ahoy install` a host reaches for) answers
// the category questions, never the install of a program. Only an answer typed
// at a terminal, or the tool named with --install-tool, is a yes.
func TestToolConfirmAsksOnlyAtATerminal(t *testing.T) {
	e := gitleaksExplained()

	var w bytes.Buffer
	piped := &stdinPrompter{r: bufio.NewReader(strings.NewReader("y\n")), w: &w}
	ans := toolConfirm(piped, nil, false, &w)(e)
	if ans.Yes {
		t.Fatal("a piped y installed a tool")
	}
	if !strings.Contains(ans.Why, "terminal") || !strings.Contains(ans.Why, "--install-tool gitleaks") {
		t.Errorf("the decline does not say how to say yes: %q", ans.Why)
	}

	w.Reset()
	tty := &stdinPrompter{r: bufio.NewReader(strings.NewReader("y\n")), w: &w, tty: true}
	ans = toolConfirm(tty, nil, false, &w)(e)
	if !ans.Yes {
		t.Fatalf("a y typed at a terminal was not a yes: %+v", ans)
	}
	asked := w.String()
	for _, want := range []string{"what abcd uses it for:", "without it:", "brew install gitleaks", "[y/N]"} {
		if !strings.Contains(asked, want) {
			t.Errorf("the terminal question lacks %q:\n%s", want, asked)
		}
	}

	w.Reset()
	tty = &stdinPrompter{r: bufio.NewReader(strings.NewReader("\n")), w: &w, tty: true}
	if ans = toolConfirm(tty, nil, false, &w)(e); ans.Yes {
		t.Fatal("a bare Enter at a terminal was a yes; the default is no")
	}
}

// TestToolConfirmNamedAndYes: --install-tool is the explicit answer a host's
// question tool relays; --yes never installs a tool, even at a terminal.
func TestToolConfirmNamedAndYes(t *testing.T) {
	e := gitleaksExplained()
	var w bytes.Buffer
	tty := &stdinPrompter{r: bufio.NewReader(strings.NewReader("y\n")), w: &w, tty: true}

	if ans := toolConfirm(tty, map[string]bool{"gitleaks": true}, false, &w)(e); !ans.Yes {
		t.Fatalf("a tool named with --install-tool was not a yes: %+v", ans)
	}
	ans := toolConfirm(tty, nil, true, &w)(e)
	if ans.Yes || !strings.Contains(ans.Why, "--yes") {
		t.Fatalf("--yes installed a tool or said nothing: %+v", ans)
	}
	if ans := toolConfirm(ahoy.RefusingPrompter{}, nil, false, &w)(e); ans.Yes {
		t.Fatal("the refusing prompter installed a tool")
	}
}

// TestInstallToolRefusesAnUnknownName: an --install-tool name ahoy install
// does not check for is refused before anything runs, naming the ones it does.
// gh is in the registry but no install gap names it, so it is refused too: a
// flag that would be accepted and then do nothing is a silent no.
func TestInstallToolRefusesAnUnknownName(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, name := range []string{"frobnicate", "gh"} {
		_, err := runCLIStdinErr(t, "", "ahoy", "install", "--install-tool", name, "--refuse-adopt")
		if err == nil {
			t.Fatalf("--install-tool %s was accepted", name)
		}
		for _, want := range append([]string{name}, ahoy.DependencyTools...) {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("refusal lacks %q: %v", want, err)
			}
		}
	}
}

// toolFreePath leaves git on PATH (the install reads the checkout) and nothing
// else, so neither gitleaks nor a package manager is found and no install
// step can ever run from this test.
func toolFreePath(t *testing.T) {
	t.Helper()
	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	if err := os.Symlink(gitBin, filepath.Join(dir, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("CI", "")
}

func installNotes(t *testing.T, out []byte) (notes, declined []string) {
	t.Helper()
	var res struct {
		Notes              []string `json:"notes"`
		DeclinedCategories []string `json:"declined_categories"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("install output not JSON: %v\n%s", err, out)
	}
	return res.Notes, res.DeclinedCategories
}

// TestAhoyInstallPipedYesExplainsAndKeepsTheNativeScanner is the end-to-end
// no: `yes | abcd ahoy install` approves every category, and the missing
// gitleaks is explained and left uninstalled, loudly, on the native scanner.
func TestAhoyInstallPipedYesExplainsAndKeepsTheNativeScanner(t *testing.T) {
	hermeticRepo(t)
	toolFreePath(t)
	out, errOut, err := runCLIPipedStdinSplit(t, strings.Repeat("y\n", 12), "ahoy", "install", "--allow-stale-binary", "--json")
	if err != nil {
		t.Fatalf("install exited non-zero: %v\n%s\n%s", err, out, errOut)
	}
	notes, _ := installNotes(t, out)
	joined := strings.Join(notes, "\n")
	for _, want := range []string{
		"dependency: gitleaks — optional for",
		"install step (Homebrew): brew install gitleaks",
		"gitleaks not installed (no terminal to ask at",
		"continuing on the native secret scanner",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("notes lack %q:\n%s", want, joined)
		}
	}
}

// TestAhoyInstallNamedToolReachesTheStep is the host's relayed yes end to
// end: with stdin closed, --install-tool answers the dependency question and
// the install is attempted; here the package manager is absent, so the
// result says so and nothing ran.
func TestAhoyInstallNamedToolReachesTheStep(t *testing.T) {
	hermeticRepo(t)
	toolFreePath(t)
	out, errOut, err := runCLIPipedStdinSplit(t, "", "ahoy", "install", "--adopt", "--allow-stale-binary", "--install-tool", "gitleaks", "--json")
	if err != nil {
		t.Fatalf("install exited non-zero: %v\n%s\n%s", err, out, errOut)
	}
	notes, declined := installNotes(t, out)
	for _, c := range declined {
		if c == "dependency" {
			t.Fatalf("--install-tool did not answer the dependency question: declined %v", declined)
		}
	}
	joined := strings.Join(notes, "\n")
	if !strings.Contains(joined, "Homebrew (brew) is not on PATH") || !strings.Contains(joined, "continuing on the native secret scanner") {
		t.Fatalf("the named install did not reach the step, or was silent:\n%s", joined)
	}
}
