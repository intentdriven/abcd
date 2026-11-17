package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/layered"
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

// TestGuardQuestionExitsOneOnARowsOverrunAlone: the rows-rule case, 34 rows
// against the default limit of 32 (the 25-row case runs over only at the
// floor, 24, since itd-2610301914350541 raised the default), is admitted by
// the hook with a note, and the verb still exits 1, because a draft is
// never ready while a tab runs past the limit.
func TestGuardQuestionExitsOneOnARowsOverrunAlone(t *testing.T) {
	unmanagedRepo(t)
	res, _, code := runQuestionVerb(t, "rule-rows")
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if res.Hook != question.OutcomeAdmitWithRowsNote {
		t.Errorf("hook = %q, want admit-with-rows-note", res.Hook)
	}
	if len(res.Findings) != 1 || res.Findings[0].Rule != string(question.RuleRows) || res.Findings[0].Refuses {
		t.Errorf("findings = %+v, want the one rows finding, not refusing", res.Findings)
	}
	want := question.RowSplit{HeaderAndFrame: 8, QuestionText: 20, Options: 6, Total: 34}
	if len(res.Rows) != 1 || res.Rows[0].RowSplit != want || res.RowsLimit != question.Default.Rows {
		t.Errorf("rows = %+v (limit %d), want tab 1 at %+v against %d", res.Rows, res.RowsLimit, want, question.Default.Rows)
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
		"rule-rows":          question.OutcomeAdmitWithRowsNote,
		// The hook marks an unlabelled question open under a named mode;
		// the verb, which never asks, marks nothing.
		"clean-unchipped": question.OutcomeAdmitUnchecked,
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
// draft is one the hook refuses, so the verb says refuse-away and offers no
// ready input; no rule is applied, so there is no finding and it exits 0
// (decision 8), leaving the routing to the hook field.
func TestGuardQuestionReportsRefuseAwayWhileAway(t *testing.T) {
	awayCheckout(t)
	setAwayHere(t)
	for _, name := range []string{"clean-product-chip", "clean-unchipped", "four-tabs-21-faults"} {
		res, _, code := runQuestionVerb(t, name)
		if res.Hook != question.OutcomeRefuseAway || code != 0 || len(res.Findings) != 0 || len(res.Ready) != 0 || !res.DependsOnState {
			t.Errorf("%s while away: hook %q, exit %d, ready %d bytes, depends_on_state %v", name, res.Hook, code, len(res.Ready), res.DependsOnState)
		}
	}
}

// TestGuardQuestionMarksAPartialResult: a mode store the verb cannot read
// leaves a number-form draft checked by every rule but the register, which
// follows the mode, so the verb says unchecked, marks the result partial and
// names the store and that rule; an older-form chipped draft is
// judged on its chip all the same, refused on a rule it breaks, partial and
// naming the store with no rule left unapplied; an away record it cannot read
// marks the result partial too, naming every rule, since it decides whether
// any is read.
func TestGuardQuestionMarksAPartialResult(t *testing.T) {
	t.Run("mode", func(t *testing.T) {
		root := managedCheckout(t)
		if err := os.WriteFile(modeFile(root), []byte("sideways\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		res, _, code := runQuestionVerb(t, "clean-number")
		if res.Hook != question.OutcomeUnchecked || !res.Abcds || code != 2 || !res.Partial || len(res.Unread) != 1 || res.Unread[0].Store != string(question.StoreMode) || !slices.Equal(res.Unread[0].Rules, []string{string(question.RuleRegister)}) {
			t.Errorf("hook %q, abcds %v, exit %d, partial %v, unread %+v; want unchecked, abcd's, exit 2, the mode store with the register rule", res.Hook, res.Abcds, code, res.Partial, res.Unread)
		}
		if strings.Contains(res.Unread[0].Why, root) {
			t.Errorf("the unread reason leaks the checkout's path: %q", res.Unread[0].Why)
		}
	})
	t.Run("mode, chipped", func(t *testing.T) {
		root := managedCheckout(t)
		if err := os.WriteFile(modeFile(root), []byte("sideways\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		res, stderr, code := runQuestionVerb(t, "rule-never-recommended")
		if res.Hook != question.OutcomeRefuse || !res.Abcds || code != 2 || len(res.Ready) != 0 {
			t.Errorf("hook %q, abcds %v, exit %d, ready %d bytes; want refuse, abcd's, exit 2 and no ready input (stderr %q)", res.Hook, res.Abcds, code, len(res.Ready), stderr)
		}
		if !slices.ContainsFunc(res.Findings, func(f questionFindingJSON) bool { return f.Rule == string(question.RuleNeverRecommended) && f.Refuses }) {
			t.Errorf("findings %+v; want the refusing never-recommended finding", res.Findings)
		}
		if !res.Partial || len(res.Unread) != 1 || res.Unread[0].Store != string(question.StoreMode) || len(res.Unread[0].Rules) != 0 {
			t.Errorf("partial %v, unread %+v; want the mode store with no rule left unapplied", res.Partial, res.Unread)
		}
		res, _, code = runQuestionVerb(t, "clean-product-chip")
		if res.Hook != question.OutcomeUnchecked || !res.Abcds || code != 2 || len(res.Findings) != 0 || res.AddresseeChanges != "" {
			t.Errorf("clean: hook %q, abcds %v, exit %d, findings %+v, addressee_changes %q; want unchecked, abcd's, exit 2, none, and no mode change", res.Hook, res.Abcds, code, res.Findings, res.AddresseeChanges)
		}
		text, _, _ := runGuard("", "guard", "question", corpusPath(t, "clean-product-chip"))
		if !strings.Contains(text, "this question was checked and has no finding, but the mode store could not be read") {
			t.Errorf("the text does not say the clean chipped draft was checked; stdout:\n%s", text)
		}
		// A chipped --ask draft with no id has a structural finding alone,
		// and the outcome line says so rather than that it has none
		// (iss-2611011943547454).
		noID := strings.Replace(structuredDraft("x"), `"id":"labels"`, `"id":""`, 1)
		text, _, code = runGuard(noID, "guard", "question", "--ask")
		if code != 2 || !strings.Contains(text, "this question was checked and its only finding is structural, which the hook never sees, but the mode store could not be read") || strings.Contains(text, "has no finding") {
			t.Errorf("exit %d; the outcome line does not agree with the structural finding; stdout:\n%s", code, text)
		}
		if !strings.Contains(text, "findings: 1, 0 of them refusing") {
			t.Errorf("the findings line is gone; stdout:\n%s", text)
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

// TestGuardQuestionSaysAnUnlabelledDraftIsNotAbcds: a draft with no header in
// abcd's grammar is another tool's question to the hook whatever the mode,
// the product thinker's included (decision 36): the verb says so in its hook
// field and its text, checks nothing, and exits 0, since no finding was made
// (decision 8); the text names the missing header so it is caught before the
// ask.
func TestGuardQuestionSaysAnUnlabelledDraftIsNotAbcds(t *testing.T) {
	for _, st := range []mode.State{mode.Managed, mode.ProductThinker} {
		t.Run(string(st), func(t *testing.T) {
			root := managedCheckout(t)
			setMode(t, root, st)
			res, _, code := runQuestionVerb(t, "clean-unchipped")
			if res.Hook != question.OutcomeAdmitUnchecked || res.Abcds || len(res.Findings) != 0 || code != 0 {
				t.Errorf("hook %q, abcds %v, %d findings, exit %d", res.Hook, res.Abcds, len(res.Findings), code)
			}
			stdout, _, _ := runGuard("", "guard", "question", corpusPath(t, "clean-unchipped"))
			if !strings.Contains(stdout, "not abcd's question") || !strings.Contains(stdout, `"Q1"`) {
				t.Errorf("the text render does not say the draft is not abcd's and name the number header:\n%s", stdout)
			}
		})
	}
}

// TestGuardQuestionListsAllTwentyOneFindings: the 2026-10-23 case, four
// issues as four tabs asked while the mode named the product thinker, rebuilt
// with four number headers so it is abcd's under decision 36, is listed
// whole: all 21 findings, every one refusing.
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
	stdout, _, code := runGuard("", "guard", "question", corpusPath(t, "rule-rows"))
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	for _, want := range []string{
		"hook: admit-with-rows-note:",
		"tab 1: 34 rows: header and frame 8, question text 20, options 6",
		"findings: 1, 0 of them refusing",
		"tab 1, whole tab (rows):",
		"ready:",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the text render lacks %q:\n%s", want, stdout)
		}
	}
}

// The verb under the question limits a repository sets (spc-2610301950037469,
// step 2): it judges at the limits the hook holds abcd's question to, reports
// them and where each came from, and exits 2 while a limit is refused or a
// file of them could not be read.

// runQuestionVerbOnDraft runs the verb with --json on q, written as the host
// question tool's input, and decodes the result.
func runQuestionVerbOnDraft(t *testing.T, q hostQuestion) (questionCheckResult, string, int) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"questions": []hostQuestion{q}})
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "draft.json")
	if err := os.WriteFile(p, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runGuard("", "guard", "question", p, "--json")
	var res questionCheckResult
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Fatalf("the verb's output is not its JSON (exit %d): %v\nstdout %q\nstderr %q", code, err, stdout, stderr)
	}
	return res, stderr, code
}

// TestGuardQuestionChecksWithTheRepositorysLimits: under a repository's
// question.rows of 40, a 30-row tab has no rows finding and a 41-row one has
// one naming 40; the result reports the figures in force and the file that
// set each, and rows_limit equals limits.rows. With no file, the figures are
// the defaults, each bundled.
func TestGuardQuestionChecksWithTheRepositorysLimits(t *testing.T) {
	repo := unmanagedRepo(t)
	res, _, code := runQuestionVerbOnDraft(t, wellBuilt())
	want := questionLimitsJSON{Rows: question.Default.Rows, LabelWords: question.Default.LabelWords, DescriptionSentences: question.Default.MeaningSentences}
	if code != 0 || res.Limits == nil || *res.Limits != want || res.LimitsSource == nil ||
		*res.LimitsSource != (questionLimitsSourceJSON{Rows: "bundled", LabelWords: "bundled", DescriptionSentences: "bundled"}) {
		t.Fatalf("an unconfigured repository: exit %d, limits %+v from %+v; want the defaults, bundled", code, res.Limits, res.LimitsSource)
	}

	writeRepoQuestionConfig(t, repo, `{"question":{"rows":40,"label_words":6}}`)
	res, _, code = runQuestionVerbOnDraft(t, questionOfRows(t, 30))
	if code != 0 || len(res.Findings) != 0 || res.Hook != question.OutcomeAdmit {
		t.Errorf("a 30-row tab under rows 40: exit %d, hook %q, findings %+v; want it admitted with none", code, res.Hook, res.Findings)
	}
	if res.Limits == nil || *res.Limits != (questionLimitsJSON{Rows: 40, LabelWords: 6, DescriptionSentences: question.Default.MeaningSentences}) {
		t.Errorf("limits = %+v, want rows 40, label words 6 and the default sentences", res.Limits)
	}
	if res.LimitsSource == nil || *res.LimitsSource != (questionLimitsSourceJSON{Rows: ".abcd/config.json", LabelWords: ".abcd/config.json", DescriptionSentences: "bundled"}) {
		t.Errorf("limits_source = %+v, want the repository's file for the two it set", res.LimitsSource)
	}
	if res.RowsLimit != 40 {
		t.Errorf("rows_limit = %d, want limits.rows, 40", res.RowsLimit)
	}
	res, _, code = runQuestionVerbOnDraft(t, questionOfRows(t, 41))
	if code != 1 || len(res.Findings) != 1 || res.Findings[0].Rule != string(question.RuleRows) || !strings.Contains(res.Findings[0].Limit, "40 rows") {
		t.Errorf("a 41-row tab under rows 40: exit %d, findings %+v; want the one rows finding naming 40", code, res.Findings)
	}

	writeMachineQuestionConfig(t, `{"question":{"rows":30}}`)
	res, _, _ = runQuestionVerbOnDraft(t, wellBuilt())
	if res.Limits == nil || res.Limits.Rows != 30 || res.LimitsSource.Rows != abcdhome.Display(layered.Config.MachineRel) || res.LimitsSource.LabelWords != ".abcd/config.json" {
		t.Errorf("a person's tighter rows: limits %+v from %+v; want rows 30 from the machine file", res.Limits, res.LimitsSource)
	}
}

// TestGuardQuestionExitsTwoOnARefusedValue: while a question limit is refused
// the verb exits 2 on any draft, clean and not abcd's included, carrying
// refused (the key, the value, the bounds and the file, and for a personal
// value that loosens the limit in force and its file) and no ready input and
// no limits, since none is in force. The hook refuses abcd's draft on the
// configuration, with no finding.
func TestGuardQuestionExitsTwoOnARefusedValue(t *testing.T) {
	t.Run("outside the bounds", func(t *testing.T) {
		repo := unmanagedRepo(t)
		writeRepoQuestionConfig(t, repo, `{"question":{"rows":100}}`)
		for _, name := range []string{"clean-product-chip", "clean-unchipped", "rule-label-words"} {
			res, stderr, code := runQuestionVerb(t, name)
			if code != 2 || res.Refused == nil || len(res.Ready) != 0 || res.Limits != nil || res.LimitsSource != nil {
				t.Fatalf("%s: exit %d, refused %+v, ready %d bytes, limits %+v; want exit 2, refused, no ready, no limits", name, code, res.Refused, len(res.Ready), res.Limits)
			}
			r := *res.Refused
			if r.Key != question.RowsKey || r.Value != "100" || r.Bounds != "24 to 60" || r.File != ".abcd/config.json" || r.Limit != 0 || r.LimitFile != "" {
				t.Errorf("%s: refused = %+v", name, r)
			}
			if !strings.Contains(stderr, "a question limit is refused") {
				t.Errorf("%s: stderr does not say a limit is refused: %q", name, stderr)
			}
		}
		res, _, _ := runQuestionVerb(t, "clean-product-chip")
		if res.Hook != question.OutcomeRefuse || len(res.Findings) != 0 {
			t.Errorf("abcd's draft: hook %q, findings %+v; want refuse with no finding", res.Hook, res.Findings)
		}
		res, _, _ = runQuestionVerb(t, "clean-unchipped")
		if res.Hook != question.OutcomeAdmitUnchecked {
			t.Errorf("another tool's draft: hook %q; want admit-unchecked, as the hook does", res.Hook)
		}
	})
	t.Run("a personal value that loosens", func(t *testing.T) {
		repo := unmanagedRepo(t)
		writeRepoQuestionConfig(t, repo, `{"question":{"rows":30}}`)
		writeMachineQuestionConfig(t, `{"question":{"rows":40}}`)
		res, _, code := runQuestionVerb(t, "clean-product-chip")
		if code != 2 || res.Refused == nil {
			t.Fatalf("exit %d, refused %+v", code, res.Refused)
		}
		r := *res.Refused
		if r.Value != "40" || r.File != abcdhome.Display(layered.Config.MachineRel) || r.Limit != 30 || r.LimitFile != ".abcd/config.json" {
			t.Errorf("refused = %+v; want 40 in the machine file, looser than 30 from the repository's", r)
		}
	})
	t.Run("an unclaimed key", func(t *testing.T) {
		repo := unmanagedRepo(t)
		writeRepoQuestionConfig(t, repo, `{"question":{"options":3}}`)
		res, _, code := runQuestionVerb(t, "clean-product-chip")
		if code != 2 || res.Refused == nil || !strings.Contains(res.Refused.Why, "question.options") || !strings.Contains(res.Refused.Why, ".abcd/config.json") {
			t.Errorf("exit %d, refused %+v; want the unclaimed key and the file named", code, res.Refused)
		}
	})
}

// TestGuardQuestionExitsTwoNamingAnUnreadConfig: a .abcd/config.json the
// resolver could not read, a symbolic link or a file it cannot parse, makes
// the result partial with an unread entry naming the file, and the verb exits
// 2. The draft is checked at the defaults, as the hook checks it: a rule the
// draft breaks is still found, and the limits reported are the defaults.
func TestGuardQuestionExitsTwoNamingAnUnreadConfig(t *testing.T) {
	for name, lay := range map[string]func(t *testing.T, repo string){
		"a symbolic link": func(t *testing.T, repo string) {
			target := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(target, []byte(`{"question":{"rows":40}}`), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, filepath.Join(repo, filepath.FromSlash(layered.Config.RepoRel))); err != nil {
				t.Fatal(err)
			}
		},
		"a key given twice": func(t *testing.T, repo string) {
			writeRepoQuestionConfig(t, repo, `{"question":{"rows":40},"question":{"rows":40}}`)
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo := unmanagedRepo(t)
			lay(t, repo)
			res, stderr, code := runQuestionVerb(t, "clean-product-chip")
			if code != 2 || !res.Partial || len(res.Unread) != 1 || res.Unread[0].Store != string(question.StoreLimits) || res.Unread[0].File != ".abcd/config.json" {
				t.Fatalf("exit %d, partial %v, unread %+v; want exit 2 and the limits' file named", code, res.Partial, res.Unread)
			}
			if res.Hook != question.OutcomeUnchecked || res.Limits == nil || res.Limits.Rows != question.Default.Rows || res.LimitsSource.Rows != "bundled" {
				t.Errorf("hook %q, limits %+v from %+v; want the defaults, as the hook judges at them", res.Hook, res.Limits, res.LimitsSource)
			}
			if !strings.Contains(stderr, "the question limits in .abcd/config.json") {
				t.Errorf("stderr does not name the unread file: %q", stderr)
			}
			res, _, code = runQuestionVerb(t, "rule-label-words")
			if code != 2 || res.Hook != question.OutcomeRefuse ||
				!slices.ContainsFunc(res.Findings, func(f questionFindingJSON) bool { return f.Rule == string(question.RuleLabelWords) }) {
				t.Errorf("a draft breaking a rule: exit %d, hook %q, findings %+v; want it refused on the rule at the defaults", code, res.Hook, res.Findings)
			}
		})
	}
	t.Run("home unset", func(t *testing.T) {
		unmanagedRepo(t)
		t.Setenv("HOME", "")
		res, _, code := runQuestionVerb(t, "rule-label-words")
		if code != 2 || !res.Partial || res.Hook != question.OutcomeRefuse || len(res.Findings) == 0 {
			t.Errorf("exit %d, partial %v, hook %q, %d findings; want the draft checked at the defaults and the result partial", code, res.Partial, res.Hook, len(res.Findings))
		}
	})
}

// TestGuardQuestionTextStatesTheLimitsInForce: the text render a person reads
// by hand states each figure in force after the outcome, naming the file
// after each figure a file set and nothing after a default, or the refusal.
func TestGuardQuestionTextStatesTheLimitsInForce(t *testing.T) {
	repo := unmanagedRepo(t)
	writeRepoQuestionConfig(t, repo, `{"question":{"rows":40}}`)
	stdout, _, _ := runGuard("", "guard", "question", corpusPath(t, "clean-product-chip"))
	lines := strings.Split(stdout, "\n")
	want := fmt.Sprintf("limits: 40 rows (.abcd/config.json), %d label words, %d description sentences", question.Default.LabelWords, question.Default.MeaningSentences)
	if len(lines) < 2 || !strings.HasPrefix(lines[0], "hook: ") || lines[1] != want {
		t.Errorf("the line after the outcome must be %q:\n%s", want, stdout)
	}
	writeRepoQuestionConfig(t, repo, `{"question":{"rows":100}}`)
	stdout, _, code := runGuard("", "guard", "question", corpusPath(t, "clean-product-chip"))
	if code != 2 || !strings.Contains(stdout, "limits: refused: .abcd/config.json (repo layer): question.rows is 100: outside 24 to 60") || strings.Contains(stdout, "ready:") {
		t.Errorf("a refused value: exit %d; the render must state the refusal and no ready input:\n%s", code, stdout)
	}
}

// collidingDrafts are host-form drafts whose objects carry a key twice, or a
// key equal to one the check reads only under case folding: encoding/json
// would judge one value while a host reading exact keys shows another. Each
// is built from a well-built question's JSON with one key added.
func collidingDrafts(t *testing.T) map[string]string {
	t.Helper()
	q := wellBuilt()
	q.Header = "Q1"
	one, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	body := strings.TrimSuffix(string(one), "}")
	opts, err := json.Marshal(q.Options[1:])
	if err != nil {
		t.Fatal(err)
	}
	optionDup := `{"label":"Keep it","label":"Keep it (Recommended)","description":"It says whom it is for. It costs a few characters."}`
	return map[string]string{
		"question under another case":  `{"questions":[` + body + `,"Question":"Ship it?"}]}`,
		"header twice":                 `{"questions":[` + body + `,"header":"Q2"}]}`,
		"an option's label twice":      `{"questions":[` + body[:strings.Index(body, `"options":`)] + `"options":[` + optionDup + `,` + strings.TrimPrefix(string(opts), "[") + `}]}`,
		"questions under another case": `{"questions":[` + string(one) + `],"Questions":[` + string(one) + `]}`,
	}
}

// TestGuardQuestionRefusesCollidingKeys: the verb exits 2 on a host-form
// draft whose objects carry a key twice or a read key under another case,
// naming the collision, since the check would judge one value and the host
// could show another. An unknown key that collides with nothing is kept.
func TestGuardQuestionRefusesCollidingKeys(t *testing.T) {
	unmanagedRepo(t)
	for name, draft := range collidingDrafts(t) {
		t.Run(name, func(t *testing.T) {
			if !json.Valid([]byte(draft)) {
				t.Fatalf("the fixture is not JSON: %s", draft)
			}
			stdout, stderr, code := runGuard(draft, "guard", "question", "-")
			if code != 2 || stdout != "" {
				t.Fatalf("exit %d, stdout %q, stderr %q; want exit 2 and nothing on stdout", code, stdout, stderr)
			}
			if !strings.Contains(stderr, "only in case") {
				t.Errorf("stderr does not name the collision: %q", stderr)
			}
		})
	}
	q := wellBuilt()
	q.Header = "Q1"
	one, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	kept := `{"questions":[` + strings.TrimSuffix(string(one), "}") + `,"x-note":"kept","X-Other":1}]}`
	if _, stderr, code := runGuard(kept, "guard", "question", "-"); code != 0 {
		t.Errorf("a draft with unknown keys that collide with nothing: exit %d, stderr %q; want 0", code, stderr)
	}
}

// TestHookDeniesCollidingKeys: the hook refuses the same drafts the verb
// exits 2 on, naming the colliding key and saying to drop it, after the away
// check: a collision is known for certain, so no question is shown that the
// check judged on one value while the host could show another. Under a mode
// naming somebody the refusal is marked open, so the next message resets the
// mode like any refusal's.
func TestHookDeniesCollidingKeys(t *testing.T) {
	keys := map[string]string{
		"question under another case":  `"Question"`,
		"header twice":                 `"header"`,
		"an option's label twice":      `"label"`,
		"questions under another case": `"Questions"`,
	}
	for name, draft := range collidingDrafts(t) {
		t.Run(name, func(t *testing.T) {
			root := managedCheckout(t)
			setMode(t, root, mode.ProductThinker)
			cwd, err := json.Marshal(root)
			if err != nil {
				t.Fatal(err)
			}
			payload := `{"session_id":"s1","hook_event_name":"PreToolUse","cwd":` + string(cwd) +
				`,"tool_name":"` + questionTools[0] + `","tool_input":` + draft + `}`
			stdout, stderr, code := runGuard(payload, "guard", "hook")
			reason := mustDeny(t, stdout, stderr, code)
			if !strings.Contains(reason, "the key "+keys[name]) || !strings.Contains(reason, "Drop ") {
				t.Errorf("the deny must name the key %s and say to drop it; reason = %q", keys[name], reason)
			}
			mustBeMarkedUnder(t, root, mode.ProductThinker)
			if got, _ := mode.ReadAt(root); got != mode.ProductThinker {
				t.Errorf("mode = %q, want product-thinker left as it was", got)
			}
		})
	}
}

// TestGuardQuestionReportsWhetherTheBadgeShows: the verb's JSON carries
// badge, the State value the hook decides with, so the writer can tell a tab
// for nobody under a badge that shows (the caller did not set the mode) from
// one where no mode is read at all: true in a managed checkout with the local
// tier, false once the tier is gone, as in a fresh clone, and false in a
// checkout abcd does not manage. The text render says the same.
func TestGuardQuestionReportsWhetherTheBadgeShows(t *testing.T) {
	q := wellBuilt()
	q.Header = "Q1"
	raw, err := json.Marshal(map[string]any{"questions": []hostQuestion{q}})
	if err != nil {
		t.Fatal(err)
	}
	badge := func(t *testing.T) any {
		t.Helper()
		stdout, stderr, code := runGuard(string(raw), "guard", "question", "-", "--json")
		var res map[string]any
		if err := json.Unmarshal([]byte(stdout), &res); err != nil {
			t.Fatalf("the verb's output is not its JSON (exit %d): %v\nstderr %q", code, err, stderr)
		}
		return res["badge"]
	}
	text := func(t *testing.T) string {
		t.Helper()
		stdout, _, _ := runGuard(string(raw), "guard", "question", "-")
		return stdout
	}
	root := managedCheckout(t)
	if got := badge(t); got != true {
		t.Errorf("badge in a managed checkout with the tier = %v, want true", got)
	}
	if !strings.Contains(text(t), "badge: shown") {
		t.Errorf("the text render does not say the badge shows: %q", text(t))
	}
	if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(mode.TierRelPath))); err != nil {
		t.Fatal(err)
	}
	if got := badge(t); got != false {
		t.Errorf("badge in a managed checkout with no tier = %v, want false", got)
	}
	if !strings.Contains(text(t), "badge: not shown") {
		t.Errorf("the text render does not say the badge is not shown: %q", text(t))
	}
	unmanagedRepo(t)
	if got := badge(t); got != false {
		t.Errorf("badge in a checkout abcd does not manage = %v, want false", got)
	}
}

// TestCheckAndGrillStateOneLimits (spc-2610301950037469, the Mechanism claim):
// from one fixture checkout, the verb's limits and limits_source and the
// figures the prompt hook's GRILL injection states agree. Under a
// repository's rows 40, label words 6 and sentences 3, the injected layout
// rule states forty rows, six words and three sentences and names the
// repository's file, rows_limit equals limits.rows, and every source is that
// file. A machine file setting rows 30 makes both say thirty, the rows'
// source the machine file and the other two the repository's.
func TestCheckAndGrillStateOneLimits(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	repo := unmanagedRepo(t)
	writeRepoQuestionConfig(t, repo, `{"question":{"rows":40,"label_words":6,"description_sentences":3}}`)
	const file = ".abcd/config.json"
	machine := abcdhome.Display(layered.Config.MachineRel)

	inject := func(session string) string {
		t.Helper()
		out, errlog := runHook(t, hookInputJSON(t, session, repo, "which option should we choose"), "hook", "prompt-router")
		if !strings.Contains(out, "## GRILL\n") {
			t.Fatalf("an asking prompt did not inject GRILL:\n%s\nstderr:\n%s", out, errlog)
		}
		return out
	}
	statesInWords := func(text string, l questionLimitsJSON) {
		t.Helper()
		for _, want := range []string{
			"fits " + numberWordForTest(l.Rows) + " rows",
			"each label is at most " + numberWordForTest(l.LabelWords) + " words",
			"each description at most " + numberWordForTest(l.DescriptionSentences) + " sentences",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("the injected GRILL does not state the verb's limits as %q:\n%s", want, text)
			}
		}
	}

	res, _, code := runQuestionVerbOnDraft(t, wellBuilt())
	want := questionLimitsJSON{Rows: 40, LabelWords: 6, DescriptionSentences: 3}
	if code != 0 || res.Limits == nil || *res.Limits != want || res.RowsLimit != res.Limits.Rows {
		t.Fatalf("the verb: exit %d, limits %+v, rows_limit %d; want %+v and rows_limit equal to limits.rows", code, res.Limits, res.RowsLimit, want)
	}
	if res.LimitsSource == nil || *res.LimitsSource != (questionLimitsSourceJSON{Rows: file, LabelWords: file, DescriptionSentences: file}) {
		t.Errorf("limits_source = %+v; want %s for every key", res.LimitsSource, file)
	}
	text := inject("one-limits-repo")
	statesInWords(text, *res.Limits)
	if !strings.Contains(text, "In this repository, `"+file+"` set the rows, the label words, and the description sentences.") {
		t.Errorf("the injected GRILL does not name %s as having set the three figures:\n%s", file, text)
	}

	writeMachineQuestionConfig(t, `{"question":{"rows":30}}`)
	res, _, _ = runQuestionVerbOnDraft(t, wellBuilt())
	want.Rows = 30
	if res.Limits == nil || *res.Limits != want || res.RowsLimit != 30 {
		t.Fatalf("the verb under a machine rows 30: limits %+v, rows_limit %d; want %+v", res.Limits, res.RowsLimit, want)
	}
	if res.LimitsSource == nil || *res.LimitsSource != (questionLimitsSourceJSON{Rows: machine, LabelWords: file, DescriptionSentences: file}) {
		t.Errorf("limits_source = %+v; want the rows from %s and the rest from %s", res.LimitsSource, machine, file)
	}
	text = inject("one-limits-machine")
	statesInWords(text, *res.Limits)
	if !strings.Contains(text, "`"+file+"` set the label words and the description sentences, and the person's own `"+machine+"` tightened the rows.") {
		t.Errorf("the injected GRILL does not name the files key by key:\n%s", text)
	}
}

// numberWordForTest writes n as the asking rules write a limit, for the
// figures this file's fixtures set.
func numberWordForTest(n int) string {
	words := map[int]string{3: "three", 6: "six", 30: "thirty", 40: "forty"}
	w, ok := words[n]
	if !ok {
		panic(fmt.Sprintf("numberWordForTest: no word for %d", n))
	}
	return w
}
