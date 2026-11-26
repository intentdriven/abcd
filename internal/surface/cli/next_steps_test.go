package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/spf13/cobra"
)

// next_steps_test.go — what usually comes next after a successful command
// (itd-2610301428378830, spc-2610301431108175 step 6). Every visible command
// has an entry in the table (internal/core/surface/next.go), every template
// names a command the tree has, and under --json a successful result carries
// its command's steps as a top-level next_steps key, spliced in with no other
// byte of the document moved.

// nextStepsBareArray are the visible commands whose --json document is not a
// JSON object, so a key cannot join it without breaking its readers. Their
// documents are left exactly as they are; their facts still carry the steps,
// so `abcd explain` and the page show them. How such a command should name
// its next step in its result is a product question the run recorded for the
// product thinker.
var nextStepsBareArray = map[string]string{
	"abcd decide":         "its --json document is a bare array",
	"abcd history list":   "its --json document is a bare array",
	"abcd history staged": "its --json document is a bare array",
	"abcd library ledger": "its --json document is a bare array",
}

// TestEveryVisibleCommandHasANextEntry holds the table to the tree both
// ways: every visible command has an entry, and every entry names a visible
// command, never a hidden one or one the tree no longer has.
func TestEveryVisibleCommandHasANextEntry(t *testing.T) {
	visible := map[string]bool{}
	for _, c := range visibleCommands(NewRootCommand()) {
		visible[c.CommandPath()] = true
		if _, ok := surface.NextFor(c.CommandPath()); !ok {
			t.Errorf("`%s` is visible and has no entry in internal/core/surface/next.go", c.CommandPath())
		}
	}
	for _, path := range surface.NextPathsForTest() {
		if !visible[path] {
			t.Errorf("internal/core/surface/next.go names %q, which is not a visible command", path)
		}
	}
}

// TestEveryNextTemplateNamesARealCommand reads each step's template the way
// a caller would run it: its leading words must reach a visible command (a
// bare word left over is a positional only where the command has no
// sub-command it could be a misspelling of), and every flag it passes must be
// one that command registers or inherits.
func TestEveryNextTemplateNamesARealCommand(t *testing.T) {
	root := NewRootCommand()
	byPath := map[string]*cobra.Command{}
	for _, c := range visibleCommands(root) {
		byPath[c.CommandPath()] = c
	}
	for _, path := range surface.NextPathsForTest() {
		n, _ := surface.NextFor(path)
		for _, s := range n.Steps {
			if problem := nextTemplateProblem(byPath, s.Command); problem != "" {
				t.Errorf("%s: step %q %s", path, s.Command, problem)
			}
		}
	}
}

// TestNoNextStepAnswersAConsentPromptAhead holds the steps to the worked
// examples' rule (TestNoWorkedExampleAnswersAConsentPromptAhead): an agent
// reads a step as a call it may run next, so a step passing one of
// consentFlags that the command's Use line does not require says in its when
// that it waits on the person.
func TestNoNextStepAnswersAConsentPromptAhead(t *testing.T) {
	byPath := map[string]*cobra.Command{}
	for _, c := range visibleCommands(NewRootCommand()) {
		byPath[c.CommandPath()] = c
	}
	for _, path := range surface.NextPathsForTest() {
		n, _ := surface.NextFor(path)
		for _, s := range n.Steps {
			c := nextTemplateCommand(byPath, s.Command)
			if c == nil {
				continue // TestEveryNextTemplateNamesARealCommand names it
			}
			required := map[string]bool{}
			for _, group := range usageRequirements(c.Use) {
				for _, f := range group {
					required[f] = true
				}
			}
			for _, tok := range exampleTokens(s.Command) {
				for _, flag := range consentFlags {
					if (tok == flag || strings.HasPrefix(tok, flag+"=")) && !required[flag] && !strings.Contains(s.When, "the person") {
						t.Errorf("%s: step %q passes %s, and its when %q does not say it waits on the person", path, s.Command, flag, s.When)
					}
				}
			}
		}
	}
}

// nextTemplateCommand is the visible command a template's leading words
// reach, or nil.
func nextTemplateCommand(byPath map[string]*cobra.Command, template string) *cobra.Command {
	toks := exampleTokens(template)
	if len(toks) == 0 || toks[0] != "abcd" {
		return nil
	}
	path := "abcd"
	for _, tok := range toks[1:] {
		if _, ok := byPath[path+" "+tok]; !ok {
			break
		}
		path += " " + tok
	}
	return byPath[path]
}

// nextTemplateProblem says what is wrong with one template, or "" when it
// names a visible command with flags it takes.
func nextTemplateProblem(byPath map[string]*cobra.Command, template string) string {
	toks := exampleTokens(template)
	if len(toks) == 0 || toks[0] != "abcd" {
		return "does not start with the binary's name"
	}
	path, rest := "abcd", toks[1:]
	for len(rest) > 0 {
		if _, ok := byPath[path+" "+rest[0]]; !ok {
			break
		}
		path, rest = path+" "+rest[0], rest[1:]
	}
	c, ok := byPath[path]
	if !ok {
		return "names no visible command"
	}
	if len(rest) > 0 && nextBareWord(rest[0]) && hasVisibleSub(c) {
		return fmt.Sprintf("reaches `%s`, and %q names none of its sub-commands", path, rest[0])
	}
	for _, tok := range rest {
		if tok == "--" {
			break
		}
		if !strings.HasPrefix(tok, "--") {
			continue
		}
		name, _, _ := strings.Cut(strings.TrimPrefix(tok, "--"), "=")
		if c.Flags().Lookup(name) == nil && c.InheritedFlags().Lookup(name) == nil {
			return fmt.Sprintf("passes --%s, which `%s` does not register", name, path)
		}
	}
	return ""
}

// nextBareWord reports whether tok reads as a sub-command's name: a word that
// is no placeholder, flag or value.
func nextBareWord(tok string) bool {
	if tok == "" || strings.ContainsAny(tok, "<>-./=") {
		return false
	}
	return strings.ToLower(tok) == tok
}

// hasVisibleSub reports whether c has a visible sub-command.
func hasVisibleSub(c *cobra.Command) bool {
	for _, sub := range c.Commands() {
		if !sub.Hidden {
			return true
		}
	}
	return false
}

// TestTheFactsCarryTheNextSteps: the facts are the one source the result,
// `explain` and the page read, so each visible command's facts carry its
// entry whole, and a hidden command's carry none.
func TestTheFactsCarryTheNextSteps(t *testing.T) {
	facts := CommandFacts()
	visible := map[string]bool{}
	for _, f := range visibleFacts(facts) {
		visible[f.Path] = true
	}
	for _, f := range facts {
		n, ok := surface.NextFor(f.Path)
		if !visible[f.Path] {
			if len(f.Next) > 0 || f.NoNext != "" || f.Streaming {
				t.Errorf("the hidden command %s carries next steps: %+v", f.Path, f.Next)
			}
			continue
		}
		if !ok {
			continue // TestEveryVisibleCommandHasANextEntry names it
		}
		if !reflect.DeepEqual(f.Next, n.Steps) || f.NoNext != n.None || f.Streaming != n.Streaming {
			t.Errorf("%s: the facts carry next %+v / %q / %v, the table %+v / %q / %v",
				f.Path, f.Next, f.NoNext, f.Streaming, n.Steps, n.None, n.Streaming)
		}
	}
}

// TestOnlyTheRootDeclaresAPreRunHook: cobra runs the nearest persistent
// pre-run hook alone, so one declared below the root would silently take the
// next steps off every command under it.
func TestOnlyTheRootDeclaresAPreRunHook(t *testing.T) {
	root := NewRootCommand()
	if root.PersistentPreRunE == nil {
		t.Fatal("the root declares no pre-run hook to carry the next steps")
	}
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if sub.PersistentPreRun != nil || sub.PersistentPreRunE != nil {
				t.Errorf("`%s` declares a persistent pre-run hook, which replaces the root's", sub.CommandPath())
			}
			walk(sub)
		}
	}
	walk(root)
}

// fixtureResult is a fixture verb's result: a nested object, a number with a
// fraction, HTML-escaped text and a key of its own named next, so a splice
// that moved, re-encoded or replaced anything would show.
type fixtureResult struct {
	Name   string            `json:"name"`
	Next   string            `json:"next,omitempty"`
	Ratio  float64           `json:"ratio"`
	Nested map[string]string `json:"nested"`
	Text   string            `json:"text"`
}

func newFixtureResult(next string) fixtureResult {
	return fixtureResult{Name: "fixture", Next: next, Ratio: 0.1 + 0.2, Nested: map[string]string{"b": "2", "a": "1"}, Text: "a <b> & c"}
}

// fixtureSteps is the entry the fixture verbs answer with.
var fixtureSteps = []surface.NextStep{
	{Command: "abcd capture resolve <iss-N> \"<what fixed it>\" --impact <impact>", When: "once the fix lands"},
	{Command: "abcd land", When: "to push & queue"},
}

// runFixture runs a fixture verb under a fresh root whose next-steps lookup
// answers entry for it, and returns its stdout. The root is built with its
// own lookup rather than by swapping the binary's, so the fixture runs beside
// every other test.
func runFixture(t *testing.T, entry surface.Next, hidden bool, args []string, run func(cmd *cobra.Command, asJSON bool) error) string {
	t.Helper()
	root := newRootCommand(func(path string) (surface.Next, bool) {
		if path == "abcd fixture-next" {
			return entry, true
		}
		return surface.NextFor(path)
	})
	fixture := &cobra.Command{
		Use:    "fixture-next",
		Hidden: hidden,
		RunE: func(cmd *cobra.Command, _ []string) error {
			asJSON, _ := cmd.Flags().GetBool("json")
			return run(cmd, asJSON)
		},
	}
	root.AddCommand(fixture)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(io.Discard)
	root.SetArgs(append([]string{"fixture-next"}, args...))
	if err := root.Execute(); err != nil {
		t.Fatalf("the fixture verb failed: %v", err)
	}
	return out.String()
}

// fixtureWriter is a visible fixture command's stdout under --json as the
// root's pre-run hook wraps it, with a check of its own, for the documents
// the binary's check would rightly record.
func fixtureWriter(out io.Writer, entry surface.Next, check func(string, []byte)) io.Writer {
	return &nextStepsWriter{Writer: out, path: "abcd fixture-next", next: entry, check: check}
}

// plainJSON is v as render wrote it before next steps existed.
func plainJSON(t *testing.T, v any) string {
	t.Helper()
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// stepsJSON is the steps as the splice writes them, indented one level in.
func stepsJSON(t *testing.T, steps []surface.NextStep) string {
	t.Helper()
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetIndent("  ", "  ")
	if err := enc.Encode(steps); err != nil {
		t.Fatal(err)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// TestASuccessfulJSONResultCarriesItsNextSteps runs a fixture verb under
// --json: its result carries the verb's steps as a top-level next_steps key,
// last, and every byte before it is the document render wrote before the
// key existed. A verb whose own result has a next keeps it where it was.
func TestASuccessfulJSONResultCarriesItsNextSteps(t *testing.T) {
	for _, own := range []string{"", "the loop's own next action"} {
		v := newFixtureResult(own)
		got := runFixture(t, surface.Next{Steps: fixtureSteps}, false, []string{"--json"}, func(cmd *cobra.Command, asJSON bool) error {
			return render(cmd.OutOrStdout(), asJSON, v, func(io.Writer) {})
		})
		plain := plainJSON(t, v)
		want := strings.TrimSuffix(plain, "\n}\n") + ",\n  \"next_steps\": " + stepsJSON(t, fixtureSteps) + "\n}\n"
		if got != want {
			t.Fatalf("the result is not the plain document with next_steps added last:\ngot\n%s\nwant\n%s", got, want)
		}
		var doc map[string]json.RawMessage
		if err := json.Unmarshal([]byte(got), &doc); err != nil {
			t.Fatalf("the result does not decode: %v", err)
		}
		var steps []surface.NextStep
		if err := json.Unmarshal(doc["next_steps"], &steps); err != nil || !reflect.DeepEqual(steps, fixtureSteps) {
			t.Errorf("next_steps decodes as %+v (%v), want %+v", steps, err, fixtureSteps)
		}
		if own != "" && string(doc["next"]) != `"the loop's own next action"` {
			t.Errorf("the verb's own next was not kept: %s", doc["next"])
		}
	}
}

// TestNextStepsAreAddedOnlyUnderJSONAndOnlyOnAVisibleCommand: the text form
// is untouched, and so is a hidden command's --json, the hook plane's.
func TestNextStepsAreAddedOnlyUnderJSONAndOnlyOnAVisibleCommand(t *testing.T) {
	v := newFixtureResult("")
	emit := func(cmd *cobra.Command, asJSON bool) error {
		return render(cmd.OutOrStdout(), asJSON, v, func(w io.Writer) { fmt.Fprintln(w, "text form") })
	}
	if got := runFixture(t, surface.Next{Steps: fixtureSteps}, false, nil, emit); got != "text form\n" {
		t.Errorf("the text form changed: %q", got)
	}
	if got := runFixture(t, surface.Next{Steps: fixtureSteps}, true, []string{"--json"}, emit); got != plainJSON(t, v) {
		t.Errorf("a hidden command's --json changed:\n%s", got)
	}
}

// TestAnEmptyEntryCarriesAnEmptyList: a command after which nothing usually
// follows says so with an empty list, and an empty object takes the key alone.
func TestAnEmptyEntryCarriesAnEmptyList(t *testing.T) {
	got := runFixture(t, surface.Next{None: "nothing usually follows"}, false, []string{"--json"}, func(cmd *cobra.Command, asJSON bool) error {
		return render(cmd.OutOrStdout(), asJSON, struct{}{}, func(io.Writer) {})
	})
	if got != "{\n  \"next_steps\": []\n}\n" {
		t.Errorf("an empty object with no steps reads %q", got)
	}
}

// TestAStreamingVerbCarriesNextStepsOnlyOnItsSummary: a streaming command's
// documents before its last carry no key, and the closing summary, written
// through renderFinal, does.
func TestAStreamingVerbCarriesNextStepsOnlyOnItsSummary(t *testing.T) {
	first, summary := newFixtureResult("first"), newFixtureResult("summary")
	got := runFixture(t, surface.Next{Steps: fixtureSteps, Streaming: true}, false, []string{"--json"}, func(cmd *cobra.Command, asJSON bool) error {
		if err := render(cmd.OutOrStdout(), asJSON, first, func(io.Writer) {}); err != nil {
			return err
		}
		return renderFinal(cmd.OutOrStdout(), asJSON, summary, func(io.Writer) {})
	})
	head := plainJSON(t, first)
	if !strings.HasPrefix(got, head) {
		t.Fatalf("the first document changed:\n%s", got)
	}
	tail := strings.TrimPrefix(got, head)
	want := strings.TrimSuffix(plainJSON(t, summary), "\n}\n") + ",\n  \"next_steps\": " + stepsJSON(t, fixtureSteps) + "\n}\n"
	if tail != want {
		t.Errorf("the summary is not the plain document with next_steps added:\ngot\n%s\nwant\n%s", tail, want)
	}
}

// TestABareArrayResultIsLeftAsItIs: a document that is not an object takes no
// key, and its bytes are those render always wrote.
func TestABareArrayResultIsLeftAsItIs(t *testing.T) {
	v := []fixtureResult{newFixtureResult("")}
	var out bytes.Buffer
	var seen []string
	w := fixtureWriter(&out, surface.Next{Steps: fixtureSteps}, checkNextStepsForTest(func(line string) { seen = append(seen, line) }))
	if err := render(w, true, v, func(io.Writer) {}); err != nil {
		t.Fatal(err)
	}
	if out.String() != plainJSON(t, v) {
		t.Errorf("a bare array changed:\n%s", out.String())
	}
	if len(seen) != 1 {
		t.Errorf("a bare array from a command the exemption list does not name was recorded %q", seen)
	}
}

// TestEveryBareArrayExemptionIsAVisibleCommandWithAnEntry: an exemption names
// a command the table still shows steps for, in explain and on the page.
func TestEveryBareArrayExemptionIsAVisibleCommandWithAnEntry(t *testing.T) {
	for path, why := range nextStepsBareArray {
		if why != "its --json document is a bare array" {
			t.Errorf("%s: the exemption's reason is %q", path, why)
		}
		if _, ok := surface.NextFor(path); !ok {
			t.Errorf("%s is exempted and has no entry in the table", path)
		}
	}
}

// TestNextStepsCheckRefusesAResultAlreadyCarryingTheKey: the check the test
// binary runs over every --json document a test renders reads the encoded
// bytes, so a verb whose own result already has a top-level next_steps is
// recorded, and the splice adds no second one. A nested next_steps is the
// result's own business and passes, and so does a bare array the exemption
// list names.
func TestNextStepsCheckRefusesAResultAlreadyCarryingTheKey(t *testing.T) {
	var got []string
	check := checkNextStepsForTest(func(line string) { got = append(got, line) })
	check("abcd fixture-next", []byte("{\n  \"next_steps\": 1\n}\n"))
	if len(got) != 1 || !strings.Contains(got[0], "already carries a top-level next_steps") {
		t.Fatalf("a result carrying the key was not recorded: %q", got)
	}
	got = nil
	check("abcd fixture-next", []byte("{\n  \"inner\": {\"next_steps\": 1}\n}\n"))
	check("abcd decide", []byte("[]\n"))
	if len(got) != 0 {
		t.Fatalf("the check recorded a nested key or an exempted bare array: %q", got)
	}
	check("abcd fixture-next", []byte("[]\n"))
	if len(got) != 1 || !strings.Contains(got[0], "not a JSON object") {
		t.Fatalf("a bare array from a command not exempted was not recorded: %q", got)
	}

	type carrying struct {
		NextSteps int `json:"next_steps"`
	}
	var out bytes.Buffer
	var seen []string
	w := fixtureWriter(&out, surface.Next{Steps: fixtureSteps}, checkNextStepsForTest(func(line string) { seen = append(seen, line) }))
	if err := render(w, true, carrying{NextSteps: 7}, func(io.Writer) {}); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 {
		t.Errorf("rendering a result that carries the key recorded %q", seen)
	}
	if out.String() != plainJSON(t, carrying{NextSteps: 7}) {
		t.Errorf("the splice changed a result that already carries the key:\n%s", out.String())
	}
}

// TestExplainShowsTheNextSteps: the text answer lists the steps, or says
// nothing usually follows and why, and the --json answer carries them in the
// command's facts.
func TestExplainShowsTheNextSteps(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	code, out, errOut := explainRun(t, "capture", "resolve")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	n, _ := surface.NextFor("abcd capture resolve")
	for _, s := range n.Steps {
		if !strings.Contains(out, "\n  "+s.Command+" — "+s.When+"\n") {
			t.Errorf("the answer does not list the step %q:\n%s", s.Command, out)
		}
	}
	code, out, _ = explainRun(t, "explain")
	if n, _ := surface.NextFor("abcd explain"); code != 0 || !strings.Contains(out, "\nNext: nothing usually follows: "+n.None+"\n") {
		t.Errorf("the answer for a command with no steps does not give the reason:\n%s", out)
	}
	_, out, _ = explainRun(t, "capture", "resolve", "--json")
	var doc struct {
		Commands  []surface.Facts    `json:"commands"`
		NextSteps []surface.NextStep `json:"next_steps"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("--json does not decode: %v", err)
	}
	if len(doc.Commands) != 1 || !reflect.DeepEqual(doc.Commands[0].Next, n.Steps) {
		t.Errorf("--json does not carry the command's next steps: %+v", doc.Commands)
	}
	en, _ := surface.NextFor("abcd explain")
	if doc.NextSteps == nil || len(doc.NextSteps) != len(en.Steps) {
		t.Errorf("explain's own result does not carry its next_steps: %s", out)
	}
}

// TestThePageBlockListsTheNextSteps: the command-facts block of a page lists
// each command's steps under its usage line.
func TestThePageBlockListsTheNextSteps(t *testing.T) {
	block := CommandFactsBlock(CommandFacts(), "capture", "abcd")
	n, _ := surface.NextFor("abcd capture resolve")
	for _, s := range n.Steps {
		if !strings.Contains(block, "    - "+codeSpan(s.Command)+", "+s.When+".\n") {
			t.Errorf("the block does not list the step %q:\n%s", s.Command, block)
		}
	}
	block = CommandFactsBlock(CommandFacts(), "explain", "abcd")
	if en, _ := surface.NextFor("abcd explain"); !strings.Contains(block, "  - Nothing usually follows: "+en.None+".\n") {
		t.Errorf("the block does not give the reason nothing follows:\n%s", block)
	}
}

// nextStepsViolations collects, for TestMain, every --json document the
// package's tests saw rendered that a next_steps key could not join.
var nextStepsViolations struct {
	sync.Mutex
	lines []string
}

// checkNextStepsForTest is nextStepsCheck under the package's tests: a
// document already carrying a top-level next_steps is recorded, and so is a
// document that is not an object from a command the exemption list does not
// name.
func checkNextStepsForTest(record func(string)) func(path string, doc []byte) {
	return func(path string, doc []byte) {
		body := bytes.TrimSpace(doc)
		if len(body) == 0 || body[0] != '{' {
			if _, ok := nextStepsBareArray[path]; !ok {
				record(fmt.Sprintf("%s: its --json document is not a JSON object, so next_steps cannot join it; "+
					"name it in nextStepsBareArray (next_steps_test.go) with its reason", path))
			}
			return
		}
		var top map[string]json.RawMessage
		if json.Unmarshal(body, &top) == nil {
			if _, dup := top["next_steps"]; dup {
				record(fmt.Sprintf("%s: its result already carries a top-level next_steps, which the splice would shadow", path))
			}
		}
	}
}

// installNextStepsCheck sets the check for the whole test binary; the
// returned function reports what it recorded and says whether the binary
// must fail.
func installNextStepsCheck() (report func() bool) {
	nextStepsCheck = checkNextStepsForTest(func(line string) {
		nextStepsViolations.Lock()
		defer nextStepsViolations.Unlock()
		nextStepsViolations.lines = append(nextStepsViolations.lines, line)
	})
	return func() bool {
		nextStepsViolations.Lock()
		defer nextStepsViolations.Unlock()
		sort.Strings(nextStepsViolations.lines)
		for _, line := range nextStepsViolations.lines {
			fmt.Fprintln(os.Stderr, "next steps check:", line)
		}
		return len(nextStepsViolations.lines) > 0
	}
}
