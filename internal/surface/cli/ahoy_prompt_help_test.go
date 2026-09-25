package cli

import (
	"bufio"
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAhoyInstallRendersCoreHelpAboveEachValueQuestion is iss-163 at the front
// door: a value question is shown with core's canonical explanation of what is
// being decided and what every answer means, printed above the question line,
// so a person (or a host agent relaying the question) never has to invent it.
// The help travels on the diagnostic stream with the question, so the --json
// envelope on stdout stays clean.
func TestAhoyInstallRendersCoreHelpAboveEachValueQuestion(t *testing.T) {
	hermeticEnv(t)
	repo := gittest.NewRepo(t).Root()
	t.Chdir(repo)
	// --yes approves the categories, so the only questions left are the value
	// questions no flag answered: visibility, the docs target, the oracle.
	out, errOut, err := runCLIPipedStdinSplit(t, "private\n\n\n", "ahoy", "install", "--yes", "--adopt", "--json")
	if err != nil {
		t.Fatalf("install exited non-zero: %v\n%s\n%s", err, out, errOut)
	}
	if !json.Valid(out) {
		t.Fatalf("stdout is not a clean JSON envelope:\n%s", out)
	}
	transcript := string(errOut)
	for _, key := range []string{"visibility", "docs_target", "oracle_backend"} {
		h, ok := ahoy.HelpFor(key)
		if !ok {
			t.Fatalf("core has no help for %s", key)
		}
		q := strings.Index(transcript, key+" (")
		if q < 0 {
			t.Fatalf("%s was not asked:\n%s", key, transcript)
		}
		about := strings.Index(transcript, h.About)
		if about < 0 || about > q {
			t.Errorf("%s: core's explanation is not printed above the question:\n%s", key, transcript)
		}
		for _, c := range h.Choices {
			line := c.Value + " — " + c.Meaning
			at := strings.Index(transcript, line)
			if at < 0 || at > q {
				t.Errorf("%s: the meaning of %q is not printed above the question", key, c.Value)
			}
		}
	}
}

// TestStdinPrompterRendersNoHelpForAnUnknownKey keeps the front door from
// inventing help: a key core has no help for is asked as the bare question.
func TestStdinPrompterRendersNoHelpForAnUnknownKey(t *testing.T) {
	var buf strings.Builder
	p := &stdinPrompter{r: bufio.NewReader(strings.NewReader("x\n")), w: &buf}
	if got := p.Prompt("no_such_question", []string{"x", "y"}, "x"); got != "x" {
		t.Fatalf("answer = %q", got)
	}
	if want := "no_such_question (x/y) [x]: x\n"; buf.String() != want {
		t.Fatalf("rendered %q, want only the question and the echoed answer %q", buf.String(), want)
	}
}
