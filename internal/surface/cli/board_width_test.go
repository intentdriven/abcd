package cli

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/textwidth"
)

// boardLongTitle runs past any row an 80-column window holds.
const boardLongTitle = "A board row whose title runs well past the eighty columns a narrow window offers, so it has to wrap"

// widthBoard lays a managed checkout whose READY intent carries a long title
// and whose reviews tree holds one release receipt, the two rows the product
// thinker saw hard-wrap to column 0 (iss-2610031207397996).
func widthBoard(t *testing.T) string {
	t.Helper()
	root := managedCheckout(t)
	statusRecord(t, root)
	p := filepath.Join(root, ".abcd", "development", "intents", "planned", "itd-2609010000000001-ready.md")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(string(data), "# The ready one", "# "+boardLongTitle, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCommitAt(t, root, "c0")
	cmd := exec.Command("git", "-C", root, "rev-parse", "HEAD")
	cmd.Env = gittest.Env(t)
	head, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	receipt := filepath.Join(root, ".abcd", "work", "reviews", strings.TrimSpace(string(head)))
	if err := os.MkdirAll(receipt, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(receipt, "docs-currency-reviewer.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// wideBoard is a window no fixture row reaches, for the tests that count a
// section's lines and are not about the width.
const wideBoard = 1000

// setBoardWidth pins the window width the board reads for one test.
func setBoardWidth(t *testing.T, cols int) {
	t.Helper()
	old := boardWidth
	boardWidth = func(io.Writer) int { return cols }
	t.Cleanup(func() { boardWidth = old })
}

// TestBoardWrapsEveryRowAtTheWindowWidth is iss-2610031207397996 at the bare
// board: every line fits the window, a row too long for it breaks at a space
// and continues four columns in from where the row began, never at column 0,
// and the first three rows are words, not Go's printing of a value.
func TestBoardWrapsEveryRowAtTheWindowWidth(t *testing.T) {
	widthBoard(t)
	for _, cols := range []int{80, 60} {
		setBoardWidth(t, cols)
		text := string(runCLI(t, "--view", "facilitator"))
		for _, l := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
			if n := textwidth.Columns(l); n > cols {
				t.Errorf("at %d columns a line takes %d:\n%q\nin\n%s", cols, n, l, text)
			}
			if l != "" && !strings.HasPrefix(l, " ") && !strings.HasPrefix(l, "abcd — ") && l != "view for the facilitator" {
				t.Errorf("at %d columns a line starts at column 0:\n%q\nin\n%s", cols, l, text)
			}
		}
		if cols != 80 {
			continue
		}
		for _, want := range []string{
			"  git repo:   yes\n  record:     yes\n",
			"      itd-2609010000000001  spc-2609010000000011  A board row whose title runs\n" +
				"          well past the eighty columns a narrow window offers, so it has to wrap\n" +
				"          [next up]\n",
			"    receipts: 1 release receipt — the release it gated is 0 commits behind main;\n" +
				"        a receipt is not re-run, and --json lists each\n",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("the 80-column board lacks\n%s\nin\n%s", want, text)
			}
		}
		for _, raw := range []string{"true", "false", "["} {
			for _, l := range strings.Split(text, "\n")[2:5] {
				if strings.Contains(l, raw) {
					t.Errorf("the first rows print a Go value (%q):\n%s", raw, text)
				}
			}
		}
	}
}

// TestPipedBoardWrapsAt80WithNoEscape: off a terminal the width is 80 whatever
// COLUMNS says, and the board carries no escape byte (adr-49).
func TestPipedBoardWrapsAt80WithNoEscape(t *testing.T) {
	widthBoard(t)
	t.Setenv("COLUMNS", "200")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	if got := boardWidth(w); got != 80 {
		t.Errorf("a pipe reads %d columns, want 80", got)
	}
	text := string(runCLI(t, "--view", "facilitator"))
	if strings.ContainsRune(text, '\x1b') {
		t.Errorf("the piped board carries an escape byte:\n%q", text)
	}
	for _, l := range strings.Split(text, "\n") {
		if n := textwidth.Columns(l); n > 80 {
			t.Errorf("a piped line takes %d columns:\n%q", n, l)
		}
	}
}
