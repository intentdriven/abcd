package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/core/question"
)

// `abcd guard question` (spc-2610301917245442, step 1): the read-only check
// of a draft question, judged by the decision the hook makes.

// questionCorpusDir is the question corpus, resolved when the package loads,
// before any test moves the process into a fixture checkout.
var questionCorpusDir = func() string {
	p, err := filepath.Abs(filepath.Join("testdata", "questions", "corpus"))
	if err != nil {
		panic(err)
	}
	return p
}()

// corpusPath is a draft of the question corpus.
func corpusPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(questionCorpusDir, name+".json")
}

func corpusDraft(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(corpusPath(t, name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// runQuestionVerb runs the verb on a corpus draft from the process's working
// directory, with --json, and decodes the result.
func runQuestionVerb(t *testing.T, name string, extra ...string) (questionCheckResult, string, int) {
	t.Helper()
	args := append([]string{"guard", "question", corpusPath(t, name), "--json"}, extra...)
	stdout, stderr, code := runGuard("", args...)
	var res questionCheckResult
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Fatalf("the verb's output is not its JSON (exit %d): %v\nstdout %q\nstderr %q", code, err, stdout, stderr)
	}
	if res.Type != questionCheckType {
		t.Fatalf("_type = %q, want %q", res.Type, questionCheckType)
	}
	return res, stderr, code
}

// TestGuardQuestionExitsZeroOnACleanDraft: a draft of abcd's with no finding,
// read from a file, from - and from stdin with no argument, exits 0 and says
// the hook would admit it.
func TestGuardQuestionExitsZeroOnACleanDraft(t *testing.T) {
	unmanagedRepo(t)
	res, stderr, code := runQuestionVerb(t, "clean-product-chip")
	if code != 0 || res.Hook != question.OutcomeAdmit || len(res.Findings) != 0 || res.Partial {
		t.Fatalf("a clean draft: exit %d, hook %q, %d findings, partial %v (stderr %q)", code, res.Hook, len(res.Findings), res.Partial, stderr)
	}
	draft := string(corpusDraft(t, "clean-product-chip"))
	for _, args := range [][]string{{"guard", "question", "-"}, {"guard", "question"}} {
		if _, stderr, code := runGuard(draft, args...); code != 0 {
			t.Errorf("%v on stdin: exit %d (stderr %q)", args, code, stderr)
		}
	}
}

// TestGuardQuestionExitsOneOnARowsOverrunAlone: the 25-row case is admitted
// by the hook with a note, and the verb still exits 1, because a draft is
// never ready while a tab runs past the limit.
func TestGuardQuestionExitsOneOnARowsOverrunAlone(t *testing.T) {
	unmanagedRepo(t)
	res, _, code := runQuestionVerb(t, "rows-25")
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if res.Hook != question.OutcomeAdmitWithRowsNote {
		t.Errorf("hook = %q, want admit-with-rows-note", res.Hook)
	}
	if len(res.Findings) != 1 || res.Findings[0].Rule != string(question.RuleRows) || res.Findings[0].Refuses {
		t.Errorf("findings = %+v, want the one rows finding, not refusing", res.Findings)
	}
	want := question.RowSplit{HeaderAndFrame: 8, QuestionText: 7, Options: 10, Total: 25}
	if len(res.Rows) != 1 || res.Rows[0].RowSplit != want || res.RowsLimit != 24 {
		t.Errorf("rows = %+v (limit %d), want tab 1 at %+v against 24", res.Rows, res.RowsLimit, want)
	}
	if len(res.Ready) == 0 {
		t.Error("an admitted draft carries no ready input")
	}
}

// TestGuardQuestionExitsTwoNamingTheShape: a draft that is not JSON, one byte
// over the hook's bound, or of neither shape exits 2 naming the shape it
// expected, and so does a structured draft carrying a key the type does not
// have.
func TestGuardQuestionExitsTwoNamingTheShape(t *testing.T) {
	unmanagedRepo(t)
	over := `{"questions": []}` + strings.Repeat(" ", maxHookStdinBytes+1-len(`{"questions": []}`))
	for _, tc := range []struct {
		name, stdin, want string
		ask               bool
	}{
		{"text", "Should we ship it?\n", `{"questions": [...]}`, false},
		{"over the bound", over, "bound", false},
		{"questions not a list", `{"questions": 1}`, `{"questions": [...]}`, false},
		{"no questions", `{}`, `{"questions": [...]}`, false},
		{"a list", `[]`, `{"questions": [...]}`, false},
		{"ask with an unknown key", `{"questions": [], "extra": 1}`, "structured question type", true},
		{"ask not JSON", `nope`, "structured question type", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"guard", "question", "-"}
			if tc.ask {
				args = append(args, "--ask")
			}
			stdout, stderr, code := runGuard(tc.stdin, args...)
			if code != 2 || stdout != "" {
				t.Fatalf("exit %d, stdout %q; want exit 2 and nothing on stdout", code, stdout)
			}
			if !strings.Contains(stderr, tc.want) {
				t.Errorf("stderr does not name %q: %q", tc.want, stderr)
			}
		})
	}
	if len(over) != maxHookStdinBytes+1 {
		t.Fatalf("the over-bound draft is %d bytes", len(over))
	}
}

// structuredDraft is abcd's structured question type as a draft carries it.
func structuredDraft(now string) string {
	q := map[string]any{
		"id":   "labels",
		"chip": "Product Q1",
		"material": []map[string]any{
			{"kind": "paragraph", "text": "Every question names who it is for in a short label above it."},
		},
		"ask": "Should the label name the role as well as the number?",
		"options": []map[string]any{
			{"value": "both", "label": "Role and number", "meaning": "It says whom it is for and where you are. It costs a few characters."},
			{"value": "number", "label": "Number only", "meaning": "The label is shorter. You read the question to learn whom it is for."},
		},
		"later":        map[string]any{"value": "later", "label": "Decide later", "meaning": "Nothing changes now."},
		"change_later": "edit the interview page",
	}
	if now != "" {
		q["now"] = now
	}
	b, _ := json.Marshal(map[string]any{"questions": []any{q}})
	return string(b)
}

// TestGuardQuestionAskComposesTheDraft: with --ask the verb judges the
// structured draft as composed into the host's input, and its ready input is
// that composition: the material, the Now: and Change later: lines, then the
// ask, with the decide-later option last. Without its Now: line the draft is
// named for it, never given an invented one.
func TestGuardQuestionAskComposesTheDraft(t *testing.T) {
	unmanagedRepo(t)
	stdout, stderr, code := runGuard(structuredDraft("the label names the role"), "guard", "question", "--ask", "--json")
	var res questionCheckResult
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Fatalf("exit %d: %v (stderr %q)", code, err, stderr)
	}
	if code != 0 || res.Form != questionFormAsk || res.Hook != question.OutcomeAdmit {
		t.Fatalf("a well-built structured draft: exit %d, form %q, hook %q, findings %+v", code, res.Form, res.Hook, res.Findings)
	}
	var ready struct {
		Questions []hostQuestionOutput `json:"questions"`
	}
	if err := json.Unmarshal(res.Ready, &ready); err != nil || len(ready.Questions) != 1 {
		t.Fatalf("ready is not the host's input: %v\n%s", err, res.Ready)
	}
	q := ready.Questions[0]
	text := q.Question
	m, n, c, a := strings.Index(text, "Every question"), strings.Index(text, "Now: "), strings.Index(text, "Change later: "), strings.Index(text, "Should the label")
	if m != 0 || !(m < n && n < c && c < a) || !strings.HasSuffix(text, "number?") {
		t.Errorf("composed text out of order:\n%s", text)
	}
	if q.Header != "Product Q1" || q.Options[len(q.Options)-1].Label != "Decide later" {
		t.Errorf("composed header %q, last option %q", q.Header, q.Options[len(q.Options)-1].Label)
	}

	stdout, _, code = runGuard(structuredDraft(""), "guard", "question", "--ask", "--json")
	res = questionCheckResult{}
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Fatal(err)
	}
	if code != 1 || res.Hook != question.OutcomeRefuse || len(res.Findings) != 1 || res.Findings[0].Rule != string(question.RuleNowAndChangeLater) {
		t.Errorf("a draft missing Now: exit %d, hook %q, findings %+v; want the now-and-change-later finding alone", code, res.Hook, res.Findings)
	}
	if len(res.Ready) != 0 {
		t.Error("a draft the hook refuses carries a ready input")
	}

	// A structural fault is reported under the structure rule and does not
	// move the hook's outcome, which never sees the structure.
	stdout, _, code = runGuard(strings.Replace(structuredDraft("x"), `"id":"labels"`, `"id":""`, 1), "guard", "question", "--ask", "--json")
	res = questionCheckResult{}
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Fatal(err)
	}
	if code != 1 || res.Hook != question.OutcomeAdmit || len(res.Findings) != 1 || res.Findings[0].Rule != string(question.RuleStructure) || res.Findings[0].Refuses {
		t.Errorf("a draft with no id: exit %d, hook %q, findings %+v", code, res.Hook, res.Findings)
	}
}

// treeHashes hashes every file under root, the .git directory included.
func treeHashes(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	out := map[string][32]byte{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			out[p+"/"] = [32]byte{}
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out[p] = sha256.Sum256(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestGuardQuestionWritesNothing: in a managed checkout with the mode on the
// product thinker and no question open, the verb on a clean, a refused and a
// rows-only draft leaves the mode file, the open-question marker and every
// other file byte-identical, and creates none.
func TestGuardQuestionWritesNothing(t *testing.T) {
	root := managedCheckout(t)
	setMode(t, root, mode.ProductThinker)
	modeBefore, err := os.ReadFile(modeFile(root))
	if err != nil {
		t.Fatal(err)
	}
	before := treeHashes(t, root)
	for name, want := range map[string]question.Outcome{
		"clean-product-chip": question.OutcomeAdmit,
		"rule-no-bold":       question.OutcomeRefuse,
		"rows-25":            question.OutcomeAdmitWithRowsNote,
	} {
		res, _, _ := runQuestionVerb(t, name)
		if res.Hook != want {
			t.Errorf("%s: hook = %q, want %q", name, res.Hook, want)
		}
		if _, _, code := runGuard("", "guard", "question", corpusPath(t, name)); code == 2 {
			t.Errorf("%s: the text render exits 2", name)
		}
	}
	modeAfter, err := os.ReadFile(modeFile(root))
	if err != nil || !bytes.Equal(modeBefore, modeAfter) {
		t.Errorf("the mode file changed: before %q, after %q (%v)", modeBefore, modeAfter, err)
	}
	if markedOpen(t, root) {
		t.Error("the verb marked a question open")
	}
	after := treeHashes(t, root)
	for p, h := range after {
		if b, ok := before[p]; !ok || b != h {
			t.Errorf("the verb wrote %s", p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			t.Errorf("the verb removed %s", p)
		}
	}
}

// TestGuardQuestionReportsRefuseAwayWhileAway: while the person is away, any
// draft is one the hook refuses, so the verb says refuse-away, exits 1, and
// offers no ready input.
func TestGuardQuestionReportsRefuseAwayWhileAway(t *testing.T) {
	awayCheckout(t)
	setAwayHere(t)
	for _, name := range []string{"clean-product-chip", "clean-unchipped", "four-tabs-21-faults"} {
		res, _, code := runQuestionVerb(t, name)
		if res.Hook != question.OutcomeRefuseAway || code != 1 || len(res.Ready) != 0 || !res.DependsOnState {
			t.Errorf("%s while away: hook %q, exit %d, ready %d bytes, depends_on_state %v", name, res.Hook, code, len(res.Ready), res.DependsOnState)
		}
	}
}

// TestGuardQuestionMarksAPartialResult: a mode store the verb cannot read
// makes the hook fail open, so the verb says unchecked, marks the result
// partial and names the store; an away record it cannot read does the same,
// naming every rule, since it decides whether any is read.
func TestGuardQuestionMarksAPartialResult(t *testing.T) {
	t.Run("mode", func(t *testing.T) {
		root := managedCheckout(t)
		if err := os.WriteFile(modeFile(root), []byte("sideways\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		res, _, _ := runQuestionVerb(t, "clean-unchipped")
		if res.Hook != question.OutcomeUnchecked || !res.Partial || len(res.Unread) != 1 || res.Unread[0].Store != string(question.StoreMode) || len(res.Unread[0].Rules) != len(questionRules) {
			t.Errorf("hook %q, partial %v, unread %+v", res.Hook, res.Partial, res.Unread)
		}
		if strings.Contains(res.Unread[0].Why, root) {
			t.Errorf("the unread reason leaks the checkout's path: %q", res.Unread[0].Why)
		}
	})
	t.Run("away", func(t *testing.T) {
		root := awayCheckout(t)
		setAwayHere(t)
		if err := os.Chmod(awayStore(t, root).RecordPath(), 0o666); err != nil {
			t.Fatal(err)
		}
		res, _, _ := runQuestionVerb(t, "clean-product-chip")
		if res.Hook != question.OutcomeUnchecked || !res.Partial || len(res.Unread) != 1 || res.Unread[0].Store != string(question.StoreAway) {
			t.Errorf("hook %q, partial %v, unread %+v", res.Hook, res.Partial, res.Unread)
		}
	})
}

// TestGuardQuestionPartialNeverExitsZero: a partial result with no finding
// among the rules it applied exits 2, and says why on stderr.
func TestGuardQuestionPartialNeverExitsZero(t *testing.T) {
	root := managedCheckout(t)
	if err := os.WriteFile(modeFile(root), []byte("sideways\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, stderr, code := runQuestionVerb(t, "clean-product-chip")
	if code != 2 || len(res.Findings) != 0 || !res.Partial {
		t.Errorf("exit %d, %d findings, partial %v; want exit 2 on a partial result with no finding", code, len(res.Findings), res.Partial)
	}
	if !strings.Contains(stderr, "partial") || !strings.Contains(stderr, "mode store") {
		t.Errorf("stderr does not say the result is partial: %q", stderr)
	}
}

// TestGuardQuestionSaysAnUnlabelledDraftIsNotAbcds: a draft without abcd's
// chip, asked while the mode names nobody, is another tool's question to the
// hook: the verb says so, checks nothing and exits 1, so a label forgotten
// is caught before the ask.
func TestGuardQuestionSaysAnUnlabelledDraftIsNotAbcds(t *testing.T) {
	managedCheckout(t)
	res, _, code := runQuestionVerb(t, "clean-unchipped")
	if res.Hook != question.OutcomeAdmitUnchecked || res.Abcds || len(res.Findings) != 0 || code != 1 {
		t.Errorf("hook %q, abcds %v, %d findings, exit %d", res.Hook, res.Abcds, len(res.Findings), code)
	}
	stdout, _, _ := runGuard("", "guard", "question", corpusPath(t, "clean-unchipped"))
	if !strings.Contains(stdout, "not abcd's question") {
		t.Errorf("the text render does not say the draft is not abcd's:\n%s", stdout)
	}
}

// TestGuardQuestionListsAllTwentyOneFindings: the 2026-10-23 case, four
// issues as four tabs asked while the mode named the product thinker, is
// listed whole: all 21 findings, every one refusing.
func TestGuardQuestionListsAllTwentyOneFindings(t *testing.T) {
	root := managedCheckout(t)
	setMode(t, root, mode.ProductThinker)
	res, _, code := runQuestionVerb(t, "four-tabs-21-faults")
	if code != 1 || res.Hook != question.OutcomeRefuse {
		t.Fatalf("exit %d, hook %q; want 1 and refuse", code, res.Hook)
	}
	if len(res.Findings) != 21 {
		t.Fatalf("findings = %d, want 21: %+v", len(res.Findings), res.Findings)
	}
	tabs := map[int]bool{}
	for _, f := range res.Findings {
		tabs[f.Tab] = true
		if !f.Refuses {
			t.Errorf("finding %+v does not refuse", f)
		}
	}
	if len(tabs) != 4 {
		t.Errorf("findings span tabs %v, want all four", tabs)
	}
}

// TestGuardQuestionReadyKeepsUnreadKeys: a host-form draft carrying keys the
// check does not read (multiSelect and a key the host may add) comes back as
// ready with every key and value as written: the JSON value is the draft's,
// and the text render prints the draft's bytes.
func TestGuardQuestionReadyKeepsUnreadKeys(t *testing.T) {
	unmanagedRepo(t)
	draft := corpusDraft(t, "clean-unknown-key")
	res, _, code := runQuestionVerb(t, "clean-unknown-key")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var want, got bytes.Buffer
	if err := json.Compact(&want, draft); err != nil {
		t.Fatal(err)
	}
	if err := json.Compact(&got, res.Ready); err != nil {
		t.Fatal(err)
	}
	if want.String() != got.String() {
		t.Errorf("ready is not the draft:\n got %s\nwant %s", got.String(), want.String())
	}
	for _, key := range []string{`"multiSelect"`, `"host_extra"`} {
		if !strings.Contains(got.String(), key) {
			t.Errorf("ready dropped %s", key)
		}
	}
	stdout, _, _ := runGuard("", "guard", "question", corpusPath(t, "clean-unknown-key"))
	if !strings.Contains(stdout, strings.TrimRight(string(draft), "\n")) {
		t.Errorf("the text render does not print the draft's bytes:\n%s", stdout)
	}
}

// TestGuardQuestionJSONCarriesTheVersion: the result names the binary's
// version, the evidence of which hook the verdict speaks for.
func TestGuardQuestionJSONCarriesTheVersion(t *testing.T) {
	unmanagedRepo(t)
	orig := core.Version
	core.Version = "v9.8.7-test"
	t.Cleanup(func() { core.Version = orig })
	res, _, _ := runQuestionVerb(t, "clean-product-chip")
	if res.Version != "v9.8.7-test" {
		t.Errorf("version = %q, want the binary's", res.Version)
	}
}

// TestGuardQuestionTextGroupsRowsAndOutcome: the text render a person reads
// by hand says what the hook would do, the rows of each tab in the check's
// split, and every finding.
func TestGuardQuestionTextGroupsRowsAndOutcome(t *testing.T) {
	unmanagedRepo(t)
	stdout, _, code := runGuard("", "guard", "question", corpusPath(t, "rows-25"))
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	for _, want := range []string{
		"hook: admit-with-rows-note:",
		"tab 1: 25 rows: header and frame 8, question text 7, options 10",
		"findings: 1, 0 of them refusing",
		"tab 1, whole tab (rows):",
		"ready:",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the text render lacks %q:\n%s", want, stdout)
		}
	}
}
