package cli

import (
	"bufio"
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/surface/cli/ask"
)

// The README question pair at the CLI's doors (spc-2610031704338417 step 4).

// readmeLicenceHelp is the licence part's help as a run builds it, its
// example carrying an escape byte a README could hold.
func readmeLicenceHelp(t *testing.T) ahoy.PromptHelp {
	t.Helper()
	h, ok := ahoy.HelpIn("", "readme.licence")
	if !ok {
		t.Fatal("core has no help for readme.licence")
	}
	h.Now = "not set, so off"
	h.Example = []string{"Licensed under \x1b[31mthe MIT License; see [LICENSE](LICENSE)."}
	return h
}

// TestStdinPrompterRendersAReadmePartsNowAndExample: the stdin door asks a
// helped question as it asks any value question, core's help above the
// unchanged question line, and prints the two per-run fields under that help:
// the current setting and the lines the part would write, each made
// terminal-safe, or the reason it writes none.
func TestStdinPrompterRendersAReadmePartsNowAndExample(t *testing.T) {
	h := readmeLicenceHelp(t)
	var buf strings.Builder
	p := &stdinPrompter{r: bufio.NewReader(strings.NewReader("off\n")), w: &buf}
	if got := p.PromptHelped(h, []string{"on", "off", "later"}, "on"); got != "off" {
		t.Fatalf("answer = %q", got)
	}
	out := buf.String()
	if strings.ContainsRune(out, 0x1b) {
		t.Errorf("an escape byte from the README reached the terminal:\n%q", out)
	}
	q := strings.Index(out, "readme.licence (on/off/later) [on]: ")
	if q < 0 {
		t.Fatalf("the question line is not the one Prompt writes:\n%s", out)
	}
	for _, want := range []string{
		h.About,
		"  on — " + h.Meaning("on"),
		"  (" + question.Default.ChangeLaterPrefix + " " + h.ChangeLater + ")",
		"  " + question.Default.NowPrefix + " not set, so off",
		"  Here it writes:",
		"    Licensed under ?[31mthe MIT License; see [LICENSE](LICENSE).",
	} {
		if at := strings.Index(out, want); at < 0 || at > q {
			t.Errorf("%q is not printed above the question:\n%s", want, out)
		}
	}

	h.Example, h.ExampleLeft = []string{"- [Section 0](#section-0)", "  - [Sub 0](#sub-0)"}, 58
	buf.Reset()
	p = &stdinPrompter{r: bufio.NewReader(strings.NewReader("\n")), w: &buf}
	p.PromptHelped(h, []string{"on", "off", "later"}, "on")
	if out := buf.String(); !strings.Contains(out, "  Here it writes 60 lines, the first 2 of them:\n    - [Section 0](#section-0)\n      - [Sub 0](#sub-0)\n") ||
		strings.Contains(out, "more") {
		t.Errorf("an example cut to its first lines is not printed exactly, its count in its lead:\n%s", out)
	}

	h.Example, h.ExampleLeft, h.ExampleNone = nil, 0, "no LICENSE file \x1b[2J at the root"
	buf.Reset()
	p = &stdinPrompter{r: bufio.NewReader(strings.NewReader("\n")), w: &buf}
	p.PromptHelped(h, []string{"on", "off", "later"}, "on")
	if out := buf.String(); !strings.Contains(out, "  Here it writes nothing: no LICENSE file ?[2J at the root.\n") {
		t.Errorf("the reason it writes none is not printed, terminal-safe:\n%q", out)
	}
}

// TestDrawnPrompterAsksAReadmePartWithItsExample: the drawn door builds the
// helped question from core's words, its current setting the question's Now
// and its example quoted under the material as the exact lines it writes (a
// paragraph of them, never a list that bullets each line), numbers it in
// setup's order and records it.
func TestDrawnPrompterAsksAReadmePartWithItsExample(t *testing.T) {
	swap := drawnPut
	t.Cleanup(func() { drawnPut = swap })
	var put question.Ask
	drawnPut = func(_ ask.Terminal, a question.Ask) ([]ask.Answer, error) {
		put = a
		return []ask.Answer{{Value: "on"}}, nil
	}
	p := &drawnPrompter{setupQuestions: setupQuestions{w: &strings.Builder{}, n: 4}}
	h := readmeLicenceHelp(t)
	if got := p.PromptHelped(h, []string{"on", "off", "later"}, "on"); got != "on" {
		t.Fatalf("answer = %q", got)
	}
	if len(put.Questions) != 1 {
		t.Fatalf("put %d questions", len(put.Questions))
	}
	q := put.Questions[0]
	if q.ID != "readme.licence" || q.Chip != "Setup Q5" || q.Now != "not set, so off" {
		t.Errorf("question id %q chip %q now %q", q.ID, q.Chip, q.Now)
	}
	if len(q.Material) != 2 || q.Material[1].Kind != question.KindParagraph || q.Material[1].Text != h.Example[0] {
		t.Fatalf("the example is not quoted under the material as the line it writes: %+v", q.Material)
	}
	if rec := p.recorded(); len(rec) != 1 || rec[0].ID != "readme.licence" || rec[0].Value != "on" {
		t.Errorf("recorded %+v", rec)
	}
	drawnPut = func(ask.Terminal, question.Ask) ([]ask.Answer, error) {
		return []ask.Answer{{Value: ahoy.SetupLaterValue, Later: true}}, nil
	}
	if got := p.PromptHelped(h, []string{"on", "off", "later"}, "on"); got != "" {
		t.Errorf("deciding later answered %q, want no answer", got)
	}
}

// TestAhoyDetectionCarriesTheReadmeQuestions: while the README question is
// open, the detection the plugin page reads carries the README view and the
// same questions as the host's question tool takes them: Q1 alone, then the
// offered parts as tabs of at most four, each tab's text holding its current
// setting and its change-later line, never a preview.
func TestAhoyDetectionCarriesTheReadmeQuestions(t *testing.T) {
	hermeticEnv(t)
	repo := gittest.NewRepo(t).Root()
	t.Chdir(repo)
	if _, err := runCLIStdinErr(t, "", "ahoy", "install", "--yes", "--adopt", "--visibility", "private",
		"--docs-target", "skip", "--scan-deep", "false", "--json"); err != nil {
		t.Fatalf("install: %v", err)
	}
	var out struct {
		Readme     *ahoy.ReadmeView    `json:"readme"`
		ReadmeTool []*hostQuestionCall `json:"readme_tool"`
	}
	raw := runCLI(t, "ahoy", "--json")
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%v\n%s", err, raw)
	}
	if out.Readme == nil || len(out.Readme.Asks) < 2 {
		t.Fatalf("no README view while the question is open:\n%s", raw)
	}
	if len(out.ReadmeTool) != len(out.Readme.Asks) {
		t.Fatalf("%d tool calls for %d asks", len(out.ReadmeTool), len(out.Readme.Asks))
	}
	if len(out.ReadmeTool[0].Questions) != 1 || out.ReadmeTool[0].Questions[0].Header != "Setup Q1" {
		t.Errorf("Q1 is not asked alone: %+v", out.ReadmeTool[0])
	}
	for i, call := range out.ReadmeTool[1:] {
		if n := len(call.Questions); n < 1 || n > 4 {
			t.Errorf("call %d holds %d tabs", i+2, n)
		}
		for _, q := range call.Questions {
			if !strings.Contains(q.Question, question.Default.NowPrefix) || !strings.Contains(q.Question, "--readme-parts") {
				t.Errorf("tab %s lacks its current setting or change-later line:\n%s", q.Header, q.Question)
			}
		}
	}

	if got, err := runCLIStdinErr(t, "", "ahoy", "install", "--yes", "--adopt", "--readme-parts", "none", "--json"); err != nil {
		t.Fatalf("install --readme-parts none: %v\n%s", err, got)
	}
	out.Readme, out.ReadmeTool = nil, nil
	raw = runCLI(t, "ahoy", "--json")
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.Readme != nil || out.ReadmeTool != nil {
		t.Errorf("an answered question is still carried:\n%s", raw)
	}
}

// TestDrawnPrompterStartsAReadmeQuestionOnItsDefault: at the drawn door Enter
// alone takes each README question's default, as an empty answer does at the
// stdin door: off for the managed-with-abcd badge and the table of contents,
// decide later for Q1. The answers keep their fixed order (on, off, later),
// and no text marks the default.
func TestDrawnPrompterStartsAReadmeQuestionOnItsDefault(t *testing.T) {
	swap := drawnPut
	t.Cleanup(func() { drawnPut = swap })
	drawnPut = func(_ ask.Terminal, a question.Ask) ([]ask.Answer, error) {
		pk := ask.NewPicker(a)
		if out := pk.Apply(ask.Key{Kind: ask.KeyEnter}); out != ask.Done {
			t.Fatalf("Enter left the list %v", out)
		}
		return pk.Answers(), nil
	}
	cases := []struct {
		key     string
		choices []string
		def     string
		want    string
		record  string
	}{
		{ahoy.ReadmePromptKey, []string{"yes", "no", ahoy.SetupLaterValue}, ahoy.SetupLaterValue, "", ahoy.SetupLaterValue},
		{"readme.abcd-badge", []string{"on", "off", ahoy.SetupLaterValue}, "off", "off", "off"},
		{"readme.contents", []string{"on", "off", ahoy.SetupLaterValue}, "off", "off", "off"},
		{"readme.licence", []string{"on", "off", ahoy.SetupLaterValue}, "on", "on", "on"},
	}
	for _, c := range cases {
		h, ok := ahoy.HelpIn("", c.key)
		if !ok {
			t.Fatalf("core has no help for %s", c.key)
		}
		p := &drawnPrompter{setupQuestions: setupQuestions{w: &strings.Builder{}}}
		if got := p.PromptHelped(h, c.choices, c.def); got != c.want {
			t.Errorf("%s: Enter answered %q, want %q", c.key, got, c.want)
		}
		rec := p.recorded()
		if len(rec) != 1 || rec[0].Value != c.record {
			t.Fatalf("%s: recorded %+v, want %q", c.key, rec, c.record)
		}
		q := rec[0].Ask.Questions[0]
		var order []string
		for _, o := range q.Options {
			order = append(order, o.Value)
		}
		if want := strings.Join(c.choices[:2], " "); strings.Join(order, " ") != want {
			t.Errorf("%s: options %v, want %s in that order", c.key, order, want)
		}
		text := hostCall(rec[0].Ask).Questions[0].Question
		if strings.Contains(strings.ToLower(text), "default") || strings.Contains(strings.ToLower(text), "recommended") {
			t.Errorf("%s: the question's text marks its default:\n%s", c.key, text)
		}
	}
}
