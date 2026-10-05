package cli

import (
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/recordid"
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
