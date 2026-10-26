package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/shellpath"
)

// The shell-settings offer (itd-2610240919030303): when the folder abcd
// installs into is not on the PATH, a person at a terminal is asked once
// whether to add it to their shell settings, naming the file and the line.

// zshUser points the shell-settings seam at a zsh login shell in the test's
// home, reading the PATH the test set.
func zshUser(t *testing.T) {
	t.Helper()
	orig := shellEnv
	shellEnv = func() shellpath.Env {
		e := shellpath.CurrentEnv()
		e.Shell, e.ZDotDir, e.GOOS = "/bin/zsh", "", "darwin"
		return e
	}
	t.Cleanup(func() { shellEnv = orig })
}

// shellPathPrompter approves every category, answers the shell-settings
// offer as told, and declines the rest, at a terminal when terminal is set.
func shellPathPrompter(yes, terminal bool) Prompter {
	sp := &scriptedPrompter{confirm: func(q string) bool {
		switch {
		case askedCategory(q) != "":
			return true
		case isShellPathQuestion(q):
			return yes
		}
		return false
	}}
	if terminal {
		return terminalScripted{sp}
	}
	return sp
}

func askedOf(p Prompter) []string {
	switch v := p.(type) {
	case terminalScripted:
		return v.asked
	case *scriptedPrompter:
		return v.asked
	}
	return nil
}

func shellPathRun(t *testing.T, yes, terminal bool, opts InstallOptions) (home string, res InstallResult, asked []string) {
	t.Helper()
	home, _ = setupUserScope(t, filepath.Join(t.TempDir(), "elsewhere"))
	zshUser(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := shellPathPrompter(yes, terminal)
	res, err := Install(repo, opts, p)
	if err != nil {
		t.Fatal(err)
	}
	return home, res, askedOf(p)
}

func shellPathAsked(asked []string) []string {
	var out []string
	for _, q := range asked {
		if isShellPathQuestion(q) {
			out = append(out, q)
		}
	}
	return out
}

const shellLine = `export PATH="$HOME/.local/bin:$PATH" # added by abcd ahoy install`

func interactiveOpts() InstallOptions {
	opts := installOpts()
	opts.Yes = false
	return opts
}

// TestShellPathOfferAddsTheLineOnYes is criterion 1: asked once, naming the
// file and the line; a yes keeps a copy, adds one line, and says to open a new
// terminal.
func TestShellPathOfferAddsTheLineOnYes(t *testing.T) {
	home, res, asked := shellPathRun(t, true, true, interactiveOpts())
	qs := shellPathAsked(asked)
	if len(qs) != 1 {
		t.Fatalf("the offer was asked %d times, want once; asked %q", len(qs), asked)
	}
	for _, want := range []string{"~/.zshrc", shellLine} {
		if !strings.Contains(qs[0], want) {
			t.Errorf("the offer does not name %q:\n%s", want, qs[0])
		}
	}
	raw, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil || string(raw) != shellLine+"\n" {
		t.Fatalf("~/.zshrc = %q, %v; want the one line", raw, err)
	}
	joined := notesJoined(res.Notes)
	if !strings.Contains(joined, "open a new terminal") {
		t.Errorf("the run does not say to open a new terminal; notes %v", res.Notes)
	}
	if strings.Contains(joined, "Add it to your shell settings") {
		t.Errorf("the run still asks the person to add the line it added; notes %v", res.Notes)
	}
	if strings.Contains(joined, home) {
		t.Errorf("a note leaks the absolute home: %v", res.Notes)
	}

	// A second run finds the line and asks nothing.
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := shellPathPrompter(true, true)
	if _, err := Install(repo, interactiveOpts(), p); err != nil {
		t.Fatal(err)
	}
	if again := shellPathAsked(askedOf(p)); len(again) != 0 {
		t.Errorf("a second run asked again: %q", again)
	}
	if raw, _ := os.ReadFile(filepath.Join(home, ".zshrc")); string(raw) != shellLine+"\n" {
		t.Errorf("a second run changed ~/.zshrc: %q", raw)
	}

	// Criterion 3, second half: removal takes out the line abcd added.
	rcpt, err := Uninstall(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if !rcpt.ShellPath.Removed {
		t.Errorf("uninstall did not remove the line: %+v", rcpt.ShellPath)
	}
	if _, err := os.Lstat(filepath.Join(home, ".zshrc")); !os.IsNotExist(err) {
		t.Errorf("the ~/.zshrc abcd created for its line alone is still there: %v", err)
	}
}

// TestShellPathDeclinedPutsTheLineOnTheChecklist is criterion 3, first half.
func TestShellPathDeclinedPutsTheLineOnTheChecklist(t *testing.T) {
	home, res, asked := shellPathRun(t, false, true, interactiveOpts())
	if len(shellPathAsked(asked)) != 1 {
		t.Fatalf("the offer was not asked once; asked %q", asked)
	}
	if _, err := os.Lstat(filepath.Join(home, ".zshrc")); !os.IsNotExist(err) {
		t.Errorf("a no wrote ~/.zshrc: %v", err)
	}
	if joined := notesJoined(res.Notes); !strings.Contains(joined, `export PATH="$HOME/.local/bin:$PATH"`) {
		t.Errorf("the line is not listed for the person; notes %v", res.Notes)
	}
}

// TestShellPathNeverAskedWithoutAPerson: --yes and a prompter not at a
// terminal never put the offer and never edit the file.
func TestShellPathNeverAskedWithoutAPerson(t *testing.T) {
	for name, run := range map[string]func(t *testing.T) (string, InstallResult, []string){
		"--yes": func(t *testing.T) (string, InstallResult, []string) {
			return shellPathRun(t, true, true, installOpts())
		},
		"no terminal": func(t *testing.T) (string, InstallResult, []string) {
			return shellPathRun(t, true, false, interactiveOpts())
		},
	} {
		t.Run(name, func(t *testing.T) {
			home, res, asked := run(t)
			if qs := shellPathAsked(asked); len(qs) != 0 {
				t.Errorf("the offer was put: %q", qs)
			}
			if _, err := os.Lstat(filepath.Join(home, ".zshrc")); !os.IsNotExist(err) {
				t.Errorf("~/.zshrc was written: %v", err)
			}
			if !strings.Contains(notesJoined(res.Notes), `export PATH="$HOME/.local/bin:$PATH"`) {
				t.Errorf("the line is not listed; notes %v", res.Notes)
			}
		})
	}
}

// TestShellPathLinkedFileIsNotEdited is criterion 2, second case: a startup
// file kept elsewhere is never edited; the line goes on the checklist.
func TestShellPathLinkedFileIsNotEdited(t *testing.T) {
	home, _ := setupUserScope(t, filepath.Join(t.TempDir(), "elsewhere"))
	zshUser(t)
	kept := filepath.Join(t.TempDir(), "zshrc")
	if err := os.WriteFile(kept, []byte("alias g=git\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(kept, filepath.Join(home, ".zshrc")); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := shellPathPrompter(true, true)
	res, err := Install(repo, interactiveOpts(), p)
	if err != nil {
		t.Fatal(err)
	}
	if qs := shellPathAsked(askedOf(p)); len(qs) != 0 {
		t.Errorf("a linked file was offered: %q", qs)
	}
	if raw, _ := os.ReadFile(kept); string(raw) != "alias g=git\n" {
		t.Errorf("the linked file changed: %q", raw)
	}
	joined := notesJoined(res.Notes)
	if !strings.Contains(joined, "link") || !strings.Contains(joined, `export PATH="$HOME/.local/bin:$PATH"`) {
		t.Errorf("the checklist does not say why and what to add; notes %v", res.Notes)
	}
}

// TestShellPathAlreadyAddedDoesNothing is criterion 2, first case.
func TestShellPathAlreadyAddedDoesNothing(t *testing.T) {
	home, _ := setupUserScope(t, filepath.Join(t.TempDir(), "elsewhere"))
	zshUser(t)
	body := "export PATH=\"$HOME/.local/bin:$PATH\"\n"
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := shellPathPrompter(true, true)
	if _, err := Install(repo, interactiveOpts(), p); err != nil {
		t.Fatal(err)
	}
	if qs := shellPathAsked(askedOf(p)); len(qs) != 0 {
		t.Errorf("settings that already add the folder were offered a second line: %q", qs)
	}
	if raw, _ := os.ReadFile(filepath.Join(home, ".zshrc")); string(raw) != body {
		t.Errorf("~/.zshrc changed: %q", raw)
	}
}

// TestShellPathOfferGapIsRaised: detection names the offer beside the PATH
// gap, which still carries the line.
func TestShellPathOfferGapIsRaised(t *testing.T) {
	home, pluginRoot := setupUserScope(t, filepath.Join(t.TempDir(), "elsewhere"))
	zshUser(t)
	linkOwned(t, filepath.Join(home, ".local", "bin", "abcd"), pluginRoot)
	det, err := Detect(managedRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	g := gapByID(det.Gaps, ShellPathOfferGapID)
	if g == nil {
		t.Fatalf("no %s gap: %+v", ShellPathOfferGapID, det.Gaps)
	}
	if g.Required || !g.Resolvable || g.Category != ShellPath {
		t.Errorf("the offer gap must be optional and resolvable in its own category: %+v", g)
	}
	if !strings.Contains(g.Detail, "~/.zshrc") {
		t.Errorf("the offer does not name the file: %+v", g)
	}
}
