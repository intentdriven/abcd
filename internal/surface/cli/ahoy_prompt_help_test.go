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
	// questions no flag answered: visibility and the docs target. The oracle is
	// not asked while host-delegated is the only reviewer abcd ships
	// (iss-2610031236155833): it is recorded, and the report says so.
	out, errOut, err := runCLIPipedStdinSplit(t, "private\n\n", "ahoy", "install", "--yes", "--adopt", "--json")
	if err != nil {
		t.Fatalf("install exited non-zero: %v\n%s\n%s", err, out, errOut)
	}
	if !json.Valid(out) {
		t.Fatalf("stdout is not a clean JSON envelope:\n%s", out)
	}
	transcript := string(errOut)
	if strings.Contains(transcript, "oracle_backend (") {
		t.Errorf("oracle_backend was asked while only host-delegated has an adapter:\n%s", transcript)
	}
	var res ahoy.InstallResult
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatal(err)
	}
	said := 0
	for _, n := range res.Notes {
		if strings.Contains(n, "host-delegated") && strings.Contains(n, "--oracle-backend") {
			said++
		}
	}
	if said != 1 {
		t.Errorf("the report does not say once what reviewer it recorded and how to choose another: %q", res.Notes)
	}
	for _, key := range []string{"visibility", "docs_target"} {
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

// TestAhoyInstallTextLeadsWithThePlainSummary is iss-164 at the front door: the
// text render opens with core's headline and its plain-language items (what,
// why, what to do) before the exact record of paths, so a person reads what
// changed for them first and the implementer's detail after.
func TestAhoyInstallTextLeadsWithThePlainSummary(t *testing.T) {
	hermeticEnv(t)
	repo := gittest.NewRepo(t).Root()
	t.Chdir(repo)
	args := []string{"ahoy", "install", "--yes", "--adopt", "--visibility", "private", "--docs-target", "both",
		"--oracle-backend", "host-delegated", "--scan-deep", "false"}
	out, errOut, err := runCLIPipedStdinSplit(t, "", append(args, "--json")...)
	if err != nil {
		t.Fatalf("install exited non-zero: %v\n%s\n%s", err, out, errOut)
	}
	var res ahoy.InstallResult
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatal(err)
	}
	if res.Headline == "" || len(res.Summary) == 0 {
		t.Fatalf("the JSON envelope carries no plain summary: %s", out)
	}

	// A second repository, rendered as text, so the first run's writes do not
	// turn this one into an up-to-date no-op.
	repo2 := gittest.NewRepo(t).Root()
	t.Chdir(repo2)
	text, errOut, err := runCLIPipedStdinSplit(t, "", args...)
	if err != nil {
		t.Fatalf("install exited non-zero: %v\n%s\n%s", err, text, errOut)
	}
	s := string(text)
	firstDetail := strings.Index(s, "  wrote: ")
	if firstDetail < 0 {
		t.Fatalf("no write detail in the text render:\n%s", s)
	}
	if at := strings.Index(s, res.Headline); at < 0 || at > firstDetail {
		t.Errorf("the headline does not lead the render:\n%s", s)
	}
	// The machine-level writes (the command entry, the session store) land on
	// the first run only, so the second repository's summary is the first's
	// less those; every item it does print must come whole and before detail.
	shown := 0
	for _, it := range res.Summary {
		at := strings.Index(s, it.What)
		if at < 0 {
			continue
		}
		shown++
		for _, part := range []string{it.What, it.Why, it.Action} {
			if at := strings.Index(s, part); at < 0 || at > firstDetail {
				t.Errorf("summary text %q is not printed before the detail:\n%s", part, s)
			}
		}
	}
	if shown < 5 {
		t.Errorf("only %d summary items were printed:\n%s", shown, s)
	}
}

// TestAhoyInstallYesKeepsValuePromptsOffStdout pins iss-2609012039114508: under
// --yes with no terminal, a value question no flag answered is still asked, and
// it must be asked on the diagnostic stream. A scripted install reads stdout as
// the run's output, so a question there would read as a receipt line.
func TestAhoyInstallYesKeepsValuePromptsOffStdout(t *testing.T) {
	hermeticEnv(t)
	repo := gittest.NewRepo(t).Root()
	t.Chdir(repo)
	out, errOut, err := runCLIPipedStdinSplit(t, "", "ahoy", "install", "--yes", "--adopt")
	if err != nil {
		t.Fatalf("install exited non-zero: %v\n%s\n%s", err, out, errOut)
	}
	if strings.Contains(string(out), "visibility (private/public)") {
		t.Errorf("a value prompt landed on stdout:\n%s", out)
	}
	if !strings.Contains(string(errOut), "visibility (private/public) []: <no answer>") {
		t.Errorf("the value prompt was not asked on stderr:\n%s", errOut)
	}
}

// TestAhoyInstallYesSaysUpFrontThatValuesAreStillAsked is iss-2609120447486547
// at the front door: --yes approves each kind of change but chooses no value, so
// a --yes run that still has a value to ask says so before the first question,
// and every value question names the flag that answers it without being asked.
// The words are core's; a run whose flags answered every value says nothing.
func TestAhoyInstallYesSaysUpFrontThatValuesAreStillAsked(t *testing.T) {
	hermeticEnv(t)
	repo := gittest.NewRepo(t).Root()
	t.Chdir(repo)
	_, errOut, err := runCLIPipedStdinSplit(t, "private\n\n", "ahoy", "install", "--yes", "--adopt", "--json")
	if err != nil {
		t.Fatalf("install exited non-zero: %v\n%s", err, errOut)
	}
	transcript := string(errOut)
	first := strings.Index(transcript, "visibility (")
	if first < 0 {
		t.Fatalf("visibility was not asked:\n%s", transcript)
	}
	notice := strings.Index(transcript, ahoy.YesStillAsksValues)
	if notice < 0 || notice > first {
		t.Errorf("the --yes notice is not printed before the first value question:\n%s", transcript)
	}
	if strings.Count(transcript, ahoy.YesStillAsksValues) != 1 {
		t.Errorf("the --yes notice is printed more than once:\n%s", transcript)
	}
	var help strings.Builder
	if code := Run([]string{"ahoy", "install", "--help"}, &help, &help); code != 0 {
		t.Fatalf("ahoy install --help exited %d", code)
	}
	prev := 0
	for _, key := range []string{"visibility", "docs_target"} {
		h, _ := ahoy.HelpFor(key)
		q := strings.Index(transcript, key+" (")
		if q < 0 {
			t.Fatalf("%s was not asked:\n%s", key, transcript)
		}
		if at := strings.Index(transcript[prev:q], h.FlagHint()); h.Flag == "" || at < 0 {
			t.Errorf("%s: the question does not name its flag %q above it:\n%s", key, h.Flag, transcript)
		}
		if !strings.Contains(help.String(), h.Flag+" ") {
			t.Errorf("%s: core names %q, which ahoy install does not register", key, h.Flag)
		}
		prev = q
	}

	repo2 := gittest.NewRepo(t).Root()
	t.Chdir(repo2)
	_, errOut, err = runCLIPipedStdinSplit(t, "", "ahoy", "install", "--yes", "--adopt", "--json",
		"--visibility", "private", "--docs-target", "skip", "--oracle-backend", "host-delegated", "--scan-deep", "false")
	if err != nil {
		t.Fatalf("install exited non-zero: %v\n%s", err, errOut)
	}
	if strings.Contains(string(errOut), ahoy.YesStillAsksValues) {
		t.Errorf("the --yes notice is printed although the flags answered every value:\n%s", errOut)
	}
}
