package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
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
// examples' rule (TestNoWorkedExampleAnswersAConsentPromptAhead), and more
// tightly: an agent reads a step as a call it may run next, so a step that
// passes one of consentFlags, or that the guard refuses an agent (the acts
// only the person takes, internal/core/guard/personacts.go, among them), says
// in its when that it waits on the person's word. No Use line excuses one: a
// command's own worked example may need the flag to be written at all, but a
// step offered after another command is a call the agent would make next.
func TestNoNextStepAnswersAConsentPromptAhead(t *testing.T) {
	reg := guard.Defaults()
	for _, path := range surface.NextPathsForTest() {
		n, _ := surface.NextFor(path)
		for _, s := range n.Steps {
			if why := nextStepNeedsThePerson(reg, s.Command); why != "" && !strings.Contains(s.When, nextStepPersonsWord) {
				t.Errorf("%s: step %q %s, and its when %q does not say it waits %s", path, s.Command, why, s.When, nextStepPersonsWord)
			}
		}
	}
}

// nextStepPersonsWord is what the when of a step that waits on the person
// says.
const nextStepPersonsWord = "only when the person says yes"

// nextStepNeedsThePerson says why the step is the person's to give, or "":
// it passes a consent flag, or the default guard refuses the line, read with
// each placeholder filled by a plain word.
func nextStepNeedsThePerson(reg guard.Registry, template string) string {
	for _, tok := range exampleTokens(template) {
		for _, flag := range consentFlags {
			if tok == flag || strings.HasPrefix(tok, flag+"=") {
				return "passes " + flag
			}
		}
	}
	line := nextPlaceholder.ReplaceAllString(template, "placeholder")
	if d, err := reg.Check(line); err == nil && d.Verdict == guard.VerdictBlock {
		return "is refused by the guard (" + d.EntryID + ")"
	}
	return ""
}

// nextPlaceholder is a template's placeholder, such as <iss-N>.
var nextPlaceholder = regexp.MustCompile(`<[^<>]+>`)

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
			// The key is written when the run returns nil; a post-run hook
			// could still fail the command after it.
			if sub.PostRun != nil || sub.PostRunE != nil || sub.PersistentPostRun != nil || sub.PersistentPostRunE != nil {
				t.Errorf("`%s` declares a post-run hook, which could fail a run whose result already carries next_steps", sub.CommandPath())
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
	out, err := runFixtureErr(t, entry, hidden, args, run)
	if err != nil {
		t.Fatalf("the fixture verb failed: %v", err)
	}
	return out
}

// runFixtureErr is runFixture for a fixture verb that may fail: its stdout
// and the error its run returned.
func runFixtureErr(t *testing.T, entry surface.Next, hidden bool, args []string, run func(cmd *cobra.Command, asJSON bool) error) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := runFixtureTo(t, &out, entry, hidden, args, run)
	return out.String(), err
}

// runFixtureTo is runFixtureErr with the fixture's stdout given.
func runFixtureTo(t *testing.T, out io.Writer, entry surface.Next, hidden bool, args []string, run func(cmd *cobra.Command, asJSON bool) error) error {
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
	root.SetOut(out)
	root.SetErr(io.Discard)
	root.SetArgs(append([]string{"fixture-next"}, args...))
	return root.Execute()
}

// fixtureWriter is a visible fixture command's stdout under --json as the
// root's pre-run hook wraps it, with a check of its own, for the documents
// the binary's check would rightly record.
func fixtureWriter(out io.Writer, entry surface.Next, check *nextStepsChecks) *nextStepsWriter {
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
	w.finish(true)
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
	check.document("abcd fixture-next", []byte("{\n  \"next_steps\": 1\n}\n"))
	if len(got) != 1 || !strings.Contains(got[0], "already carries a top-level next_steps") {
		t.Fatalf("a result carrying the key was not recorded: %q", got)
	}
	got = nil
	check.document("abcd fixture-next", []byte("{\n  \"inner\": {\"next_steps\": 1}\n}\n"))
	check.document("abcd decide", []byte("[]\n"))
	if len(got) != 0 {
		t.Fatalf("the check recorded a nested key or an exempted bare array: %q", got)
	}
	check.document("abcd fixture-next", []byte("[]\n"))
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
	w.finish(true)
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

// TestAFailedRunWritesItsDocumentWithNoNextSteps: a run that returns an
// error, with a code and a message or with the code alone, writes the
// document it rendered exactly as render always wrote it, so no document a
// non-zero exit or an error object follows carries the key, whichever helper
// the verb wrote it through. A streaming command's summary is held to the
// same end.
func TestAFailedRunWritesItsDocumentWithNoNextSteps(t *testing.T) {
	v := newFixtureResult("")
	for _, fail := range []error{&exitError{Code: 2, Msg: "refused"}, &exitError{Code: 1}, fmt.Errorf("plain failure")} {
		for _, entry := range []surface.Next{{Steps: fixtureSteps}, {Steps: fixtureSteps, Streaming: true}} {
			got, err := runFixtureErr(t, entry, false, []string{"--json"}, func(cmd *cobra.Command, asJSON bool) error {
				if err := renderFinal(cmd.OutOrStdout(), asJSON, v, func(io.Writer) {}); err != nil {
					return err
				}
				return fail
			})
			if err == nil {
				t.Fatalf("the fixture's failure %v was lost", fail)
			}
			if got != plainJSON(t, v) {
				t.Errorf("after %v (streaming %v) the document is not as render wrote it:\n%s", fail, entry.Streaming, got)
			}
		}
	}
}

// TestOnlyACommandsLastDocumentCarriesNextSteps: a command that writes two
// documents has its first written as it was, and the key joins the last.
func TestOnlyACommandsLastDocumentCarriesNextSteps(t *testing.T) {
	first, last := newFixtureResult("first"), newFixtureResult("last")
	got := runFixture(t, surface.Next{Steps: fixtureSteps}, false, []string{"--json"}, func(cmd *cobra.Command, asJSON bool) error {
		if err := render(cmd.OutOrStdout(), asJSON, first, func(io.Writer) {}); err != nil {
			return err
		}
		return render(cmd.OutOrStdout(), asJSON, last, func(io.Writer) {})
	})
	want := plainJSON(t, first) + strings.TrimSuffix(plainJSON(t, last), "\n}\n") + ",\n  \"next_steps\": " + stepsJSON(t, fixtureSteps) + "\n}\n"
	if got != want {
		t.Errorf("the documents are not the first as it was and the last with next_steps:\ngot\n%s\nwant\n%s", got, want)
	}
}

// TestATerminalTestLeavesTheDocumentAsItIs: the terminal tests read the
// stream beneath the wrapper and release nothing, so a --json document is the
// same whether they run before the render or after it, and only a caller that
// takes the stream itself (a child's stdout) writes the held document first.
func TestATerminalTestLeavesTheDocumentAsItIs(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := fixtureWriter(f, surface.Next{Steps: fixtureSteps}, nil)
	if streamOf(w) != io.Writer(f) || plainStdout(w) != io.Writer(f) {
		t.Fatal("the wrapper does not give the stream beneath it")
	}
	v := newFixtureResult("")
	want := strings.TrimSuffix(plainJSON(t, v), "\n}\n") + ",\n  \"next_steps\": " + stepsJSON(t, fixtureSteps) + "\n}\n"
	for _, after := range []bool{false, true} {
		got := runFixture(t, surface.Next{Steps: fixtureSteps}, false, []string{"--json"}, func(cmd *cobra.Command, asJSON bool) error {
			look := func() {
				_ = bannerTTY(cmd.OutOrStdout())
				_ = boardWidth(cmd.OutOrStdout())
				_ = boardRung(cmd.OutOrStdout(), false)
			}
			if !after {
				look()
			}
			if err := render(cmd.OutOrStdout(), asJSON, v, func(io.Writer) {}); err != nil {
				return err
			}
			if after {
				look()
			}
			return nil
		})
		if got != want {
			t.Errorf("with the terminal tests after the render %v, the document changed:\n%s", after, got)
		}
	}

	var out bytes.Buffer
	w = fixtureWriter(&out, surface.Next{Steps: fixtureSteps}, nil)
	if err := render(w, true, v, func(io.Writer) {}); err != nil {
		t.Fatal(err)
	}
	if _ = streamOf(w); out.Len() != 0 {
		t.Fatalf("a terminal test released the held document: %q", out.String())
	}
	if _ = plainStdout(w); out.String() != plainJSON(t, v) {
		t.Fatalf("taking the stream did not write the held document first, as it was: %q", out.String())
	}
}

// TestADocumentWrittenAroundTheSpliceIsHeard: the test binary's check hears
// what a run that succeeded wrote to its stdout outside writeDocument, so a
// --json document a verb encodes straight to the stream is recorded; text,
// and anything a failed run wrote, is not.
func TestADocumentWrittenAroundTheSpliceIsHeard(t *testing.T) {
	for _, c := range []struct {
		raw  string
		ok   bool
		want int
	}{
		{"{\n  \"findings\": []\n}\n", true, 1},
		{"[1, 2]\n", true, 1},
		{"{\"findings\": []}\n", false, 0},
		{"a line of text\n", true, 0},
	} {
		var seen []string
		w := fixtureWriter(io.Discard, surface.Next{Steps: fixtureSteps}, checkNextStepsForTest(func(line string) { seen = append(seen, line) }))
		fmt.Fprint(w, c.raw)
		w.finish(c.ok)
		if len(seen) != c.want {
			t.Errorf("%q written around the splice by a run that succeeded %v was recorded %q", c.raw, c.ok, seen)
		}
	}
}

// TestAKeyAFailedExitFollowsIsHeard: run's half of the check records a key
// that a non-zero exit follows, and passes one an exit 0 follows.
func TestAKeyAFailedExitFollowsIsHeard(t *testing.T) {
	var seen []string
	w := fixtureWriter(io.Discard, surface.Next{Steps: fixtureSteps}, checkNextStepsForTest(func(line string) { seen = append(seen, line) }))
	if err := render(w, true, newFixtureResult(""), func(io.Writer) {}); err != nil {
		t.Fatal(err)
	}
	w.finish(true)
	cmd := &cobra.Command{Use: "fixture-next"}
	cmd.SetOut(w)
	checkNextStepsExit(cmd, 0)
	if len(seen) != 0 {
		t.Fatalf("a key an exit 0 follows was recorded: %q", seen)
	}
	checkNextStepsExit(cmd, 2)
	if len(seen) != 1 || !strings.Contains(seen[0], "exited 2") {
		t.Fatalf("a key an exit 2 follows was not recorded: %q", seen)
	}
}

// nextStepsViolations collects, for TestMain, everything the next steps'
// checks recorded while the package's tests ran.
var nextStepsViolations struct {
	sync.Mutex
	lines []string
}

// checkNextStepsForTest is nextStepsCheck under the package's tests:
//
//   - a document already carrying a top-level next_steps is recorded, and so
//     is a document that is not an object from a command the exemption list
//     does not name;
//   - a JSON document a run that succeeded wrote around writeDocument is
//     recorded, since it carries no next steps;
//   - a key that a non-zero exit (and so an error object, under --json)
//     follows is recorded.
func checkNextStepsForTest(record func(string)) *nextStepsChecks {
	return &nextStepsChecks{
		document: func(path string, doc []byte) {
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
		},
		around: func(path string, raw []byte) {
			dec := json.NewDecoder(bytes.NewReader(raw))
			for {
				var v json.RawMessage
				if dec.Decode(&v) != nil {
					return
				}
				if b := bytes.TrimSpace(v); len(b) > 0 && (b[0] == '{' || b[0] == '[') {
					record(fmt.Sprintf("%s: a --json document was written straight to stdout, around writeDocument, "+
						"so it carries no next_steps; write it through render or writeDocument", path))
					return
				}
			}
		},
		exit: func(path string, spliced bool, code int) {
			if spliced && code != 0 {
				record(fmt.Sprintf("%s: exited %d after writing next_steps, which only a success carries", path, code))
			}
		},
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

// failingStdout is a stdout that refuses every write, as a full disk does.
type failingStdout struct{}

func (failingStdout) Write([]byte) (int, error) { return 0, errors.New("no space left on device") }

// TestADocumentThatCannotBeWrittenFailsTheRun: the held document is written
// once the run has returned, and a write that fails there fails the run, as
// it did when the document was written inside it; a run that already failed
// keeps its own error.
func TestADocumentThatCannotBeWrittenFailsTheRun(t *testing.T) {
	v := newFixtureResult("")
	emit := func(cmd *cobra.Command, asJSON bool) error {
		return render(cmd.OutOrStdout(), asJSON, v, func(io.Writer) {})
	}
	if err := runFixtureTo(t, failingStdout{}, surface.Next{Steps: fixtureSteps}, false, []string{"--json"}, emit); err == nil {
		t.Fatal("a result that could not be written left the run succeeding")
	}
	own := &exitError{Code: 2, Msg: "refused"}
	err := runFixtureTo(t, failingStdout{}, surface.Next{Steps: fixtureSteps}, false, []string{"--json"}, func(cmd *cobra.Command, asJSON bool) error {
		_ = emit(cmd, asJSON)
		return own
	})
	if err != own {
		t.Fatalf("a failed run's own error was replaced: %v", err)
	}
}
