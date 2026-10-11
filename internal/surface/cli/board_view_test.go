package cli

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/gittest"
)

// setBannerTTY forces the bare board's Terminal seam for one test.
func setBannerTTY(t *testing.T, on bool) {
	t.Helper()
	old := bannerTTY
	bannerTTY = func(io.Writer) bool { return on }
	t.Cleanup(func() { bannerTTY = old })
}

// TestBareBoardOpensOnTheProductView is A1 at the front door
// (spc-2610031844142274): on a Terminal, whichever role the stored status
// names, bare abcd draws the product thinker's view, and no line of it holds a
// record handle, abcd followed by a verb, or an owed answer.
func TestBareBoardOpensOnTheProductView(t *testing.T) {
	root := managedCheckout(t)
	statusRecord(t, root)
	writeRunState(t, root, "itd-7")
	setBannerTTY(t, true)
	setBoardWidth(t, 80)
	t.Setenv("NO_COLOR", "1")
	command := regexp.MustCompile(`\babcd (build|capture|intent|implement|ahoy|peers|inbox|mode|spec|drain|lint|decide|update)\b`)
	for _, role := range []string{"product-thinker", "facilitator"} {
		runCLI(t, "mode", role)
		text := string(runCLI(t))
		if !strings.Contains(text, "\nview for the product thinker\n") || strings.Contains(text, "view for the facilitator") {
			t.Fatalf("with the stored status %s the board is not the product thinker's view:\n%s", role, text)
		}
		board := text[strings.Index(text, "view for the product thinker"):]
		for _, l := range strings.Split(board, "\n") {
			if h, ok := recordid.HandleInText(l); ok {
				t.Errorf("the product thinker's view carries the handle %s: %q", h, l)
			}
			if command.MatchString(l) || strings.Contains(l, "waiting on") {
				t.Errorf("the product thinker's view carries a command or an owed answer: %q", l)
			}
		}
	}
}

// TestPipedBoardCarriesNoEscape is A2: piped, with a Terminal that could show
// true colour, bare abcd draws the product thinker's view with no escape
// byte, and --json carries no escape byte and names the view.
func TestPipedBoardCarriesNoEscape(t *testing.T) {
	root := managedCheckout(t)
	statusRecord(t, root)
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-256color")
	text := string(runCLI(t))
	if strings.ContainsRune(text, '\x1b') {
		t.Errorf("the piped board carries an escape byte:\n%q", text)
	}
	if first, _, _ := strings.Cut(text, "\n"); first != "view for the product thinker" {
		t.Errorf("the piped board opens on %q, want the product thinker's view", first)
	}
	raw := runCLI(t, "--json")
	if strings.ContainsRune(string(raw), '\x1b') {
		t.Errorf("--json carries an escape byte")
	}
	var got struct {
		View string `json:"view"`
	}
	if err := json.Unmarshal(raw, &got); err != nil || got.View != "product-thinker" {
		t.Errorf("--json view = %q (%v), want product-thinker", got.View, err)
	}
	if err := json.Unmarshal(runCLI(t, "--json", "--view", "facilitator"), &got); err != nil || got.View != "facilitator" {
		t.Errorf("--json --view facilitator names view %q (%v)", got.View, err)
	}
}

// TestFormatAndJSONAreRefusedTogether: --json beside --format is two answers to
// one question and is refused with exit 2 naming both flags; so is either new
// flag beside a record id, whose answer has one form; an unknown value of
// either flag is refused too.
func TestFormatAndJSONAreRefusedTogether(t *testing.T) {
	root := managedCheckout(t)
	statusRecord(t, root)
	for _, tc := range []struct {
		args  []string
		names []string
	}{
		{[]string{"--json", "--format", "markdown"}, []string{"--json", "--format"}},
		{[]string{"--json", "--format", "text"}, []string{"--json", "--format"}},
		{[]string{"itd-7", "--view", "facilitator"}, []string{"--view"}},
		{[]string{"itd-7", "--format", "markdown"}, []string{"--format"}},
		{[]string{"--view", "manager"}, []string{"--view", "product", "facilitator"}},
		{[]string{"--format", "html"}, []string{"--format", "text", "markdown"}},
		{[]string{"--version", "--view", "facilitator"}, []string{"--version", "--view"}},
		{[]string{"--version", "--format", "markdown"}, []string{"--version", "--format"}},
	} {
		out, err := runCLIStdinErr(t, "", tc.args...)
		var ee *exitError
		if !errors.As(err, &ee) || ee.Code != 2 {
			t.Errorf("%v: err = %v, want exit 2", tc.args, err)
			continue
		}
		for _, n := range tc.names {
			if !strings.Contains(ee.Msg, n) {
				t.Errorf("%v: the refusal %q does not name %s", tc.args, ee.Msg, n)
			}
		}
		if strings.Contains(string(out), "view for the") {
			t.Errorf("%v: a refused call drew the board:\n%s", tc.args, out)
		}
	}
}

// TestTheTerminalBoardTakesTheColourLadderAndTheLocale is A6 at the front
// door: on a Terminal that shows true colour the view label is painted;
// NO_COLOR and --no-color paint nothing; without a UTF-8 locale the board
// draws its plain-text symbols.
func TestTheTerminalBoardTakesTheColourLadderAndTheLocale(t *testing.T) {
	root := managedCheckout(t)
	statusRecord(t, root)
	setBannerTTY(t, true)
	setBoardWidth(t, 80)
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LANG", "en_GB.UTF-8")
	t.Setenv("NO_COLOR", "")
	painted := string(runCLI(t))
	if !strings.Contains(painted, "\x1b[38;2;") || !strings.Contains(painted, " view for the product thinker \x1b[39;49m") {
		t.Errorf("at true colour the view label is not painted in the role's pair:\n%q", painted)
	}
	if !strings.Contains(painted, "● building:") {
		t.Errorf("a UTF-8 locale draws the UTF-8 symbols:\n%s", painted)
	}
	board := func(out string) string { return out[strings.Index(out, "view for the product thinker"):] }
	t.Setenv("NO_COLOR", "1")
	if out := board(string(runCLI(t))); strings.ContainsRune(out, '\x1b') {
		t.Errorf("under NO_COLOR the board carries an escape byte:\n%q", out)
	}
	t.Setenv("NO_COLOR", "")
	if out := board(string(runCLI(t, "--no-color"))); strings.ContainsRune(out, '\x1b') {
		t.Errorf("under --no-color the board carries an escape byte:\n%q", out)
	}
	t.Setenv("LANG", "C")
	if out := string(runCLI(t, "--no-color")); !strings.Contains(out, "* building:") || strings.Contains(out, "●") {
		t.Errorf("without a UTF-8 locale the board does not draw its plain-text symbols:\n%s", out)
	}
}

// TestALaneOnItsBranchIsInFlightOnTheBoard is A5 end to end: a run's lane whose
// recorded branch exists, building an intent whose spec is open, is marked in
// flight on the facilitator's view and in --json.
func TestALaneOnItsBranchIsInFlightOnTheBoard(t *testing.T) {
	root := managedCheckout(t)
	statusRecord(t, root)
	gitCommitAt(t, root, "c0")
	const branch = "build/run-2609290000000001-lane-1"
	cmd := exec.Command("git", "-C", root, "branch", branch)
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git branch: %v\n%s", err, out)
	}
	writeRunStateOnBranch(t, root, "itd-2609010000000001", branch)
	setBoardWidth(t, wideBoard)
	text := string(runCLI(t, "--view", "facilitator"))
	if !strings.Contains(text, "[lane-1: implement (run-2609290000000001); in flight]") {
		t.Errorf("the lane on its branch is not marked in flight:\n%s", text)
	}
	var got struct {
		Status struct {
			Now []struct {
				SpecID string `json:"spec_id"`
				Lane   *struct {
					Branch   string `json:"branch"`
					InFlight bool   `json:"in_flight"`
				} `json:"lane"`
			} `json:"now"`
		} `json:"status"`
	}
	if err := json.Unmarshal(runCLI(t, "--json"), &got); err != nil {
		t.Fatal(err)
	}
	if n := got.Status.Now; len(n) == 0 || n[0].Lane == nil || n[0].Lane.Branch != branch || !n[0].Lane.InFlight || n[0].SpecID != "spc-2609010000000011" {
		t.Errorf("--json Now = %+v, want the lane's branch, in_flight and its spec", n)
	}
}

// writeRunStateOnBranch writes one run's state file with a lane in progress on
// intentID whose worktree stage recorded branch.
func writeRunStateOnBranch(t *testing.T, root, intentID, branch string) {
	t.Helper()
	writeRunState(t, root, intentID)
	dir := filepath.Join(root, filepath.FromSlash(loop.RunRelDir), "run-2609290000000001")
	p := filepath.Join(dir, loop.StateFileName)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var st loop.State
	if err := json.Unmarshal(data, &st); err != nil {
		t.Fatal(err)
	}
	st.Lanes[0].Branch = branch
	if data, err = json.Marshal(st); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestAbcdPageRelaysTheMarkdownForm is A3's page (spc-2610031844142274): the
// /abcd page runs the markdown form exactly and tells the agent to paste its
// output unchanged in one fenced block, adding nothing inside it and retelling
// none of it outside; the full board is the same with --view facilitator.
func TestAbcdPageRelaysTheMarkdownForm(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRootFromTest(t), "commands", "abcd.md"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(raw)
	for _, want := range []string{
		"```bash\n\"${CLAUDE_PLUGIN_ROOT}/abcd\" --format markdown\n```",
		"```bash\n\"${CLAUDE_PLUGIN_ROOT}/abcd\" --view facilitator --format markdown\n```",
		"unchanged, in one fenced block",
		"add nothing inside the fence",
		"retell none of it outside",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the /abcd page lacks %q", want)
		}
	}
	// The bare board is drawn, not summarised from JSON: the first command the
	// page runs is the markdown form.
	if i, j := strings.Index(page, "--format markdown"), strings.Index(page, "\" --json"); i < 0 || (j >= 0 && j < i) {
		t.Errorf("the /abcd page runs --json before the markdown form")
	}
}

// TestBoardShowsTheVersion is A9 at the front door (spc-2610100613109045,
// decision 4): the installed version is the board's last line in both views
// and both forms, read from the same core.VersionInfo `abcd --version`
// reports; --json carries it as `version`; and the first line stays the view
// label alone.
func TestBoardShowsTheVersion(t *testing.T) {
	root := managedCheckout(t)
	statusRecord(t, root)
	orig := core.Version
	core.Version = "v9.9.9"
	t.Cleanup(func() { core.Version = orig })
	setBoardWidth(t, 80)
	t.Setenv("NO_COLOR", "1")
	for _, tc := range []struct {
		args        []string
		first, last string
	}{
		{nil, "view for the product thinker", "abcd v9.9.9"},
		{[]string{"--view", "facilitator"}, "view for the facilitator", "abcd v9.9.9"},
		{[]string{"--format", "markdown"}, "view for the product thinker", "- abcd v9.9.9"},
		{[]string{"--view", "facilitator", "--format", "markdown"}, "view for the facilitator", "- abcd v9.9.9"},
	} {
		lines := strings.Split(strings.TrimRight(string(runCLI(t, tc.args...)), "\n"), "\n")
		if lines[0] != tc.first {
			t.Errorf("%v: the first line is %q, want the view label %q alone", tc.args, lines[0], tc.first)
		}
		if got := lines[len(lines)-1]; got != tc.last {
			t.Errorf("%v: the last line is %q, want the version %q", tc.args, got, tc.last)
		}
	}
	var got struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(runCLI(t, "--json"), &got); err != nil || got.Version != "v9.9.9" {
		t.Errorf("--json version = %q (%v), want the version --version reports, v9.9.9", got.Version, err)
	}
}
