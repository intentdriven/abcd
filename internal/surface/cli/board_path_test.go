package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// boardCheckout makes a git checkout at dir under a fresh HOME and moves the
// test into it, so bare `abcd` renders the board for dir.
func boardCheckout(t *testing.T, home, dir string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	gitInitAt(t, dir)
	t.Chdir(dir)
}

// boardDir runs the board in both forms and returns the directory the text
// form's first line names and the JSON form's dir, with the whole text output.
func boardDir(t *testing.T) (textDir, jsonDir, text string) {
	t.Helper()
	text = string(runCLI(t))
	first, _, _ := strings.Cut(text, "\n")
	textDir, ok := strings.CutPrefix(first, "abcd — ")
	if !ok {
		t.Fatalf("the board's first line is not `abcd — <dir>`:\n%s", text)
	}
	var got struct {
		Dir string `json:"dir"`
	}
	if err := json.Unmarshal(runCLI(t, "--json"), &got); err != nil {
		t.Fatal(err)
	}
	return textDir, got.Dir, text
}

// absSpellings is p in the spelling given and its symlink-resolved one (the
// one os.Getwd and git report back on macOS, where /var is /private/var).
func absSpellings(p string) []string {
	out := []string{p}
	if r, err := filepath.EvalSymlinks(p); err == nil && r != p {
		out = append(out, r)
	}
	return out
}

// The board is the output most often pasted, and it named the checkout by its
// absolute path, /Users/<account>/… included (iss-2609281613094952). Under
// HOME it names it home-relative, in the text form and in --json.
func TestBoardNamesACheckoutUnderHomeHomeRelative(t *testing.T) {
	home := t.TempDir()
	boardCheckout(t, home, filepath.Join(home, "code", "the-repo"))

	textDir, jsonDir, text := boardDir(t)
	want := "~" + string(filepath.Separator) + filepath.Join("code", "the-repo")
	if textDir != want {
		t.Errorf("the board's first line names %q, want %q", textDir, want)
	}
	if jsonDir != want {
		t.Errorf("--json dir = %q, want %q", jsonDir, want)
	}
	noHomePath(t, home, text)
	noHomePath(t, home, jsonDir)
}

// Outside HOME the home redaction leaves the path whole, so the board names
// the checkout by its directory name, fsutil.DisplayPath's rule.
func TestBoardNamesACheckoutOutsideHomeByItsDirectoryName(t *testing.T) {
	outside := t.TempDir()
	boardCheckout(t, t.TempDir(), filepath.Join(outside, "the-repo"))

	textDir, jsonDir, text := boardDir(t)
	if textDir != "the-repo" {
		t.Errorf("the board's first line names %q, want the directory name %q", textDir, "the-repo")
	}
	if jsonDir != "the-repo" {
		t.Errorf("--json dir = %q, want the directory name %q", jsonDir, "the-repo")
	}
	for _, abs := range absSpellings(outside) {
		if strings.Contains(text, abs) || strings.Contains(jsonDir, abs) {
			t.Errorf("the board prints the absolute checkout path under %s:\n%s\ndir: %s", abs, text, jsonDir)
		}
	}
}

// The board's peers notice carried the reader's error through the home
// redaction alone, and an unreadable record folder's error names the folder by
// its absolute path, so a checkout outside HOME was printed whole on stderr.
// The notice names it by the display rule, like the board's first line.
func TestBoardPeersNoticeNamesACheckoutOutsideHomeByItsDirectoryName(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("chmod 0 does not deny root, so the record folder stays readable and no notice is printed")
	}
	outside := t.TempDir()
	repo := filepath.Join(outside, "the-repo")
	boardCheckout(t, t.TempDir(), repo)
	locked := filepath.Join(repo, ".abcd", "work", "issues", "open")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	_, stderr, err := runCLISplit(t)
	if err != nil {
		t.Fatalf("the board failed: %v\n%s", err, stderr)
	}
	if !strings.Contains(stderr, "the peers line is omitted") {
		t.Fatalf("precondition: an unreadable record folder omits the peers line with a notice, got stderr:\n%s", stderr)
	}
	if !strings.Contains(stderr, filepath.Join("the-repo", ".abcd", "work", "issues", "open")) {
		t.Errorf("the notice no longer names the folder it could not read:\n%s", stderr)
	}
	for _, abs := range absSpellings(outside) {
		if strings.Contains(stderr, abs) {
			t.Errorf("the board's notice prints the absolute checkout path under %s:\n%s", abs, stderr)
		}
	}
}

// The board's first line is the one board value that reached the terminal
// unsanitised: every other line goes through termsafe.Sanitize, but the
// checkout's display name was written raw, so a directory whose name carries an
// ESC sequence or a bidi override recoloured or reordered the board a person
// reads and pastes (iss-2609281736483740). The text line masks each control;
// --json carries the directory's true name, which the encoder escapes where it
// is a control byte and a reader renders on its own terms, as it does every
// other board field.
func TestBoardFirstLineMasksControlsInTheCheckoutName(t *testing.T) {
	for _, tc := range []struct{ label, name, masked, raw string }{
		{"an ESC sequence", "the\x1b[31mrepo", "the?[31mrepo", "\x1b"},
		{"a right-to-left override", "the\u202erepo", "the?repo", "\u202e"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			outside := t.TempDir()
			boardCheckout(t, t.TempDir(), filepath.Join(outside, tc.name))

			textDir, jsonDir, text := boardDir(t)
			if textDir != tc.masked {
				t.Errorf("the board's first line names %q, want the masked %q", textDir, tc.masked)
			}
			if strings.Contains(text, tc.raw) {
				t.Errorf("the board's text carries the raw control %q:\n%q", tc.raw, text)
			}
			if jsonDir != tc.name {
				t.Errorf("--json dir = %q, want the directory's true name %q", jsonDir, tc.name)
			}
			if raw := runCLI(t, "--json"); strings.Contains(string(raw), "\x1b") {
				t.Errorf("--json carries a raw ESC byte; the encoder escapes it:\n%q", raw)
			}
		})
	}
}
