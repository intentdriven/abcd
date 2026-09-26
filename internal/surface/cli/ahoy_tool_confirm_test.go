package cli

import (
	"bufio"
	"bytes"
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
