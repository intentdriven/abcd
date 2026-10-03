package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/core/question"
)

// The question check in the guard hook (spc-2610030944505997, step 2): the
// hook decodes the question tool's input, decides whether the question is
// abcd's, holds abcd's questions to the field limits wherever it runs, and keeps
// the mode gate for abcd's questions in a checkout abcd manages.

// hostOption and hostQuestion are the host's question-tool input as a test
// writes it: the host's own JSON key names.
type hostOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
	Preview     string `json:"preview,omitempty"`
}

type hostQuestion struct {
	Header      string       `json:"header"`
	Question    string       `json:"question"`
	MultiSelect bool         `json:"multiSelect"`
	Options     []hostOption `json:"options"`
}

// wellBuilt is A2's question at the hook: a "Product Q2" chip, the thing
// quoted first, the Now: and Change later: lines, the question last, three
// options of short labels and two-sentence meanings, "Decide later" last. It
// names no record and no command, so the product thinker may be asked it.
func wellBuilt() hostQuestion {
	return hostQuestion{
		Header: "Product Q2",
		Question: strings.Join([]string{
			"Every question names who it is for in a short label above it: for example, the label reads Product Q2 on the second question put to you.",
			"",
			"Now: not applicable",
			"Change later: not applicable",
			"",
			"Should the label name the role as well as the number?",
		}, "\n"),
		Options: []hostOption{
			{Label: "Role and number", Description: "It says whom it is for and where you are. It costs a few characters."},
			{Label: "Number only", Description: "The label is shorter. You read the question to learn whom it is for."},
			{Label: "Decide later", Description: "Nothing changes now. The question comes back at the next interview."},
		},
	}
}

// askToolPayload is the host's PreToolUse payload for a question-tool call
// whose tool_input is input, verbatim.
func askToolPayload(t *testing.T, cwd string, input any) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"session_id":      "s1",
		"cwd":             cwd,
		"hook_event_name": "PreToolUse",
		"tool_name":       questionTools[0],
		"tool_input":      input,
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func askPayload(t *testing.T, cwd string, qs ...hostQuestion) string {
	t.Helper()
	return askToolPayload(t, cwd, map[string]any{"questions": qs})
}

// unmanagedRepo is a git checkout abcd does not manage: no marker block, no
// local tier, so no mode store.
func unmanagedRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	gitInitAt(t, repo)
	repo = realPath(t, repo)
	t.Chdir(repo)
	return repo
}

func setMode(t *testing.T, root string, st mode.State) {
	t.Helper()
	if err := mode.SetAt(root, st); err != nil {
		t.Fatal(err)
	}
}

func markedOpen(t *testing.T, root string) bool {
	t.Helper()
	_, err := os.Lstat(questionMarker(root))
	if err == nil {
		return true
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return false
}

// findingLineRe reads one finding line of a refusal: where, the part, the rule.
var findingLineRe = regexp.MustCompile(`^(tab [0-9]+|the call), (.+?) \(([a-z-]+)\): `)

// moreLineRe reads the line that closes a refusal naming fewer parts than it
// counts.
var moreLineRe = regexp.MustCompile(`^\.\.\. and ([0-9]+) more part\(s\)`)

type namedPart struct {
	Tab  int    `json:"tab"`
	Part string `json:"part"`
	Rule string `json:"rule"`
}

// refusal splits a limits refusal into its head line and the parts its finding
// lines name, failing the test when the head line is not the spec's.
func refusal(t *testing.T, stderr string) (head string, parts []namedPart, lines []string) {
	t.Helper()
	all := strings.Split(strings.TrimRight(stderr, "\n"), "\n")
	head = all[0]
	if !strings.HasPrefix(head, "Blocked by the abcd guard (question tool): ") ||
		!strings.HasSuffix(head, " part(s) of this question break abcd's asking rules; fix each and ask again.") {
		t.Fatalf("the head line is not the spec's: %q", head)
	}
	more := 0
	for _, l := range all[1:] {
		if m := moreLineRe.FindStringSubmatch(l); m != nil {
			more, _ = strconv.Atoi(m[1])
			continue
		}
		m := findingLineRe.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		tab := 0
		if strings.HasPrefix(m[1], "tab ") {
			tab, _ = strconv.Atoi(strings.TrimPrefix(m[1], "tab "))
		}
		parts = append(parts, namedPart{Tab: tab, Part: m[2], Rule: m[3]})
		lines = append(lines, l)
	}
	n := strings.TrimSuffix(strings.TrimPrefix(head, "Blocked by the abcd guard (question tool): "), " part(s) of this question break abcd's asking rules; fix each and ask again.")
	if want := strconv.Itoa(len(parts) + more); n != want {
		t.Errorf("the head line counts %s part(s), but %d finding line(s) and %d more follow:\n%s", n, len(parts), more, stderr)
	}
	return head, parts, lines
}

func rulesNamed(parts []namedPart) []string {
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = p.Rule
	}
	return out
}

// TestRoutingQuestionOf20261003IsRefused is criterion A1: the routing question
// of 2026-10-03, rebuilt as a fixture, is refused with exit 2 and exactly the
// parts its findings list names, with the mode at product-thinker.
func TestRoutingQuestionOf20261003IsRefused(t *testing.T) {
	input, err := os.ReadFile(filepath.Join("testdata", "questions", "routing-2026-10-03.json"))
	if err != nil {
		t.Fatal(err)
	}
	rawWant, err := os.ReadFile(filepath.Join("testdata", "questions", "routing-2026-10-03.findings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var want struct {
		Findings []namedPart `json:"findings"`
	}
	if err := json.Unmarshal(rawWant, &want); err != nil {
		t.Fatal(err)
	}
	root := managedCheckout(t)
	setMode(t, root, mode.ProductThinker)

	stdout, stderr, code := runGuard(askToolPayload(t, root, json.RawMessage(input)), "guard", "hook")
	if code != 2 {
		t.Fatalf("the routing question must be refused with exit 2; got %d (stderr %q)", code, stderr)
	}
	if stdout != "" {
		t.Errorf("the refusal belongs on stderr alone, and the input is never rewritten; stdout = %q", stdout)
	}
	_, got, _ := refusal(t, stderr)
	if !slices.Equal(got, want.Findings) {
		t.Errorf("the refusal named\n%v\nwant exactly\n%v\nstderr:\n%s", got, want.Findings, stderr)
	}
	if markedOpen(t, root) {
		t.Error("a refused question was marked open")
	}
}

// TestWellBuiltQuestionIsAdmitted is criterion A2 at the hook: the well-built
// question runs with exit 0 and nothing on either stream, and is marked open
// so the next human message resets the mode. Where abcd does not manage the
// repository, the same question runs the same way and nothing is written.
func TestWellBuiltQuestionIsAdmitted(t *testing.T) {
	t.Run("managed", func(t *testing.T) {
		root := managedCheckout(t)
		setMode(t, root, mode.ProductThinker)
		stdout, stderr, code := runGuard(askPayload(t, root, wellBuilt()), "guard", "hook")
		if code != 0 || stdout != "" || stderr != "" {
			t.Fatalf("a well-built question must be admitted silently; code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
		if !markedOpen(t, root) {
			t.Error("the admitted question was not marked open")
		}
	})
	t.Run("unmanaged", func(t *testing.T) {
		repo := unmanagedRepo(t)
		stdout, stderr, code := runGuard(askPayload(t, repo, wellBuilt()), "guard", "hook")
		if code != 0 || stdout != "" || stderr != "" {
			t.Fatalf("a well-built question must be admitted silently; code=%d stdout=%q stderr=%q", code, stdout, stderr)
		}
		if _, err := os.Lstat(filepath.Join(repo, ".abcd")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("the check created .abcd/ in an unmanaged repository: %v", err)
		}
	})
}

// TestRecommendedStarredOrLongHeaderIsRefused is criterion A3 at the hook:
// each case is refused with exit 2 and a line naming the rule, the value and
// the limit. A header outside the chip grammar makes the question abcd's only
// through a mode naming somebody, so the mode here names the product thinker.
func TestRecommendedStarredOrLongHeaderIsRefused(t *testing.T) {
	for _, tc := range []struct {
		name, rule, value, limit string
		edit                     func(*hostQuestion)
	}{
		{"recommended", "never-recommended", "Keep it (Recommended)", "no (Recommended) label", func(q *hostQuestion) { q.Options[0].Label = "Keep it (Recommended)" }},
		{"starred", "never-recommended", "★ Keep it", "no star mark", func(q *hostQuestion) { q.Options[0].Label = "★ Keep it" }},
		{"long header", "header", "Product thinker Q2", "12 columns", func(q *hostQuestion) { q.Header = "Product thinker Q2" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := managedCheckout(t)
			setMode(t, root, mode.ProductThinker)
			q := wellBuilt()
			tc.edit(&q)
			stdout, stderr, code := runGuard(askPayload(t, root, q), "guard", "hook")
			if code != 2 || stdout != "" {
				t.Fatalf("want exit 2 and an empty stdout; code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			_, parts, lines := refusal(t, stderr)
			hit := false
			for i, p := range parts {
				if p.Rule == tc.rule && strings.Contains(lines[i], strconv.Quote(tc.value)) && strings.Contains(lines[i], tc.limit) {
					hit = true
				}
			}
			if !hit {
				t.Errorf("no line names the rule %q, the value %q and the limit %q:\n%s", tc.rule, tc.value, tc.limit, stderr)
			}
		})
	}
}

// TestProductThinkerQuestionNamesNoRecordOrCommand is criterion R4 at the
// hook. With the mode at product-thinker a record number, a command named by
// one of the binary's verbs, and a label in backticks are each refused; the
// same questions under the facilitator mode are admitted. With no mode store,
// the "Product" chip stands in for the mode and refuses the same way.
func TestProductThinkerQuestionNamesNoRecordOrCommand(t *testing.T) {
	cases := []struct {
		name string
		edit func(*hostQuestion)
	}{
		{"record number", func(q *hostQuestion) {
			q.Options[0].Description = "It closes iss-2609202058058301. It costs a few characters."
		}},
		{"verb from the command tree", func(q *hostQuestion) {
			q.Question = "Set it with abcd mode before you ask, for example.\n\n" + q.Question
		}},
		{"label in backticks", func(q *hostQuestion) { q.Options[1].Label = "`Number only`" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := wellBuilt()
			tc.edit(&q)

			root := managedCheckout(t)
			setMode(t, root, mode.ProductThinker)
			_, stderr, code := runGuard(askPayload(t, root, q), "guard", "hook")
			if code != 2 {
				t.Fatalf("product thinker: want exit 2; got %d (stderr %q)", code, stderr)
			}
			if _, parts, _ := refusal(t, stderr); !slices.Equal(rulesNamed(parts), []string{"register"}) {
				t.Errorf("product thinker: want one register finding; got %v:\n%s", rulesNamed(parts), stderr)
			}

			setMode(t, root, mode.Facilitator)
			if _, stderr, code := runGuard(askPayload(t, root, q), "guard", "hook"); code != 0 || stderr != "" {
				t.Errorf("facilitator: the mechanism and the ids are theirs; code=%d stderr=%q", code, stderr)
			}

			repo := unmanagedRepo(t)
			_, stderr, code = runGuard(askPayload(t, repo, q), "guard", "hook")
			if code != 2 {
				t.Fatalf("no mode store, Product chip: want exit 2; got %d (stderr %q)", code, stderr)
			}
			if _, parts, _ := refusal(t, stderr); !slices.Equal(rulesNamed(parts), []string{"register"}) {
				t.Errorf("no mode store, Product chip: want one register finding; got %v:\n%s", rulesNamed(parts), stderr)
			}
		})
	}
}

// TestRecommendedOrStarredOptionIsRefused is criterion R6 at the hook: the
// refusal names the label, and its remedy names the host's own instruction and
// abcd's rule reversing it, so the agent does not loop.
func TestRecommendedOrStarredOptionIsRefused(t *testing.T) {
	root := managedCheckout(t)
	setMode(t, root, mode.Facilitator)
	q := wellBuilt()
	q.Options[1].Label = "Number only *"
	_, stderr, code := runGuard(askPayload(t, root, q), "guard", "hook")
	if code != 2 {
		t.Fatalf("want exit 2; got %d (stderr %q)", code, stderr)
	}
	_, parts, lines := refusal(t, stderr)
	if len(parts) != 1 || parts[0].Rule != "never-recommended" || parts[0].Part != "option 2 label" {
		t.Fatalf("want one never-recommended finding on option 2's label; got %v:\n%s", parts, stderr)
	}
	if !strings.Contains(lines[0], strconv.Quote("Number only *")) {
		t.Errorf("the refusal does not name the label:\n%s", stderr)
	}
	if !strings.Contains(lines[0], question.RecommendedRemedy) {
		t.Errorf("the remedy must name the host's instruction and abcd's rule reversing it:\n%s", stderr)
	}
}

// TestForeignQuestionIsNotRefused is the mode gate's scope (itd-201 decision
// 10): a question with no abcd chip, asked while the mode reads managed, is
// another tool's question. It runs with exit 0, unchecked and unmarked, in a
// managed checkout and elsewhere alike.
func TestForeignQuestionIsNotRefused(t *testing.T) {
	foreign := hostQuestion{
		Header:   "Pick a theme",
		Question: "Which theme?",
		Options:  []hostOption{{Label: "Dark (Recommended)"}, {Label: "Light"}},
	}
	root := managedCheckout(t)
	stdout, stderr, code := runGuard(askPayload(t, root, foreign), "guard", "hook")
	if code != 0 || stdout != "" || stderr != "" {
		t.Fatalf("another tool's question must run unchecked; code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if markedOpen(t, root) {
		t.Error("another tool's question was marked open")
	}
	repo := unmanagedRepo(t)
	if stdout, stderr, code := runGuard(askPayload(t, repo, foreign), "guard", "hook"); code != 0 || stdout != "" || stderr != "" {
		t.Fatalf("another tool's question must run unchecked where unmanaged; code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

// TestAbcdChipWhileManagedIsRefused is the other half of the mode gate's
// scope: an abcd chip while the mode reads managed is refused with the
// existing one-line refusal and its `abcd mode` remedy, and nothing is marked
// open. With a field finding as well, both are named: the finding lines first,
// then the mode's line.
func TestAbcdChipWhileManagedIsRefused(t *testing.T) {
	root := managedCheckout(t)
	q := wellBuilt()
	q.Header = "Product Q1"
	stdout, stderr, code := runGuard(askPayload(t, root, q), "guard", "hook")
	if code != 2 || stdout != "" {
		t.Fatalf("an abcd chip while managed must exit 2; code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if strings.TrimRight(stderr, "\n") != questionRefusal {
		t.Errorf("want the existing one-line refusal alone; stderr = %q", stderr)
	}
	if markedOpen(t, root) {
		t.Error("a refused question was marked open")
	}

	q.Options[0].Label = "Keep it (Recommended)"
	_, stderr, code = runGuard(askPayload(t, root, q), "guard", "hook")
	if code != 2 {
		t.Fatalf("want exit 2; got %d (stderr %q)", code, stderr)
	}
	if _, parts, _ := refusal(t, stderr); !slices.Equal(rulesNamed(parts), []string{"never-recommended"}) {
		t.Errorf("want the one field finding; got %v:\n%s", rulesNamed(parts), stderr)
	}
	all := strings.Split(strings.TrimRight(stderr, "\n"), "\n")
	if all[len(all)-1] != questionRefusal {
		t.Errorf("the mode's refusal must close the lines; stderr:\n%s", stderr)
	}
}

// TestUnreadableQuestionsFailOpenLoud: a questions field the check cannot read
// is not a decision. The question runs, and the hook says so on exit 1, the
// guard's fail-open-loud contract, wherever it runs.
func TestUnreadableQuestionsFailOpenLoud(t *testing.T) {
	for name, input := range map[string]any{
		"absent":            map[string]any{},
		"null":              map[string]any{"questions": nil},
		"a string":          map[string]any{"questions": "sideways"},
		"options a number":  map[string]any{"questions": []any{map[string]any{"header": "Product Q1", "question": "Ship?", "options": 3}}},
		"a label an object": map[string]any{"questions": []any{map[string]any{"header": "Product Q1", "question": "Ship?", "options": []any{map[string]any{"label": map[string]any{}}}}}},
	} {
		t.Run(name, func(t *testing.T) {
			for where, dir := range map[string]func(*testing.T) string{"managed": managedCheckout, "unmanaged": unmanagedRepo} {
				root := dir(t)
				stdout, stderr, code := runGuard(askToolPayload(t, root, input), "guard", "hook")
				if code != 1 || stdout != "" {
					t.Fatalf("%s: an unreadable questions field must fail open loud (exit 1); code=%d stdout=%q stderr=%q", where, code, stdout, stderr)
				}
				for _, want := range []string{"NOT CHECKED", "UNGATED"} {
					if !strings.Contains(stderr, want) {
						t.Errorf("%s: the fail-open must say %q; stderr = %q", where, want, stderr)
					}
				}
			}
		})
	}
}

// TestQuestionRefusalEchoesSanitisedValues (invariant 13): a value the
// refusal quotes back carries no control character, escape sequence,
// bidirectional control or zero-width rune, whichever field it came from.
func TestQuestionRefusalEchoesSanitisedValues(t *testing.T) {
	repo := unmanagedRepo(t)
	q := wellBuilt()
	q.Options[1].Description = "One. \x1b[2JTwo\u202e. Three."
	q.Options[0].Label = "Keep\x07 it \u200b(Recommended)\x1b]8;;http://example.com\x07"
	_, stderr, code := runGuard(askPayload(t, repo, q), "guard", "hook")
	if code != 2 {
		t.Fatalf("want exit 2; got %d (stderr %q)", code, stderr)
	}
	for _, r := range stderr {
		if (r < 0x20 && r != '\n') || r == 0x7f || (r >= 0x80 && r <= 0x9f) || r == '\u202e' || r == '\u200b' {
			t.Fatalf("the refusal echoes an unsafe rune %U:\n%q", r, stderr)
		}
	}
	if _, parts, _ := refusal(t, stderr); !slices.Equal(rulesNamed(parts), []string{"meaning-sentences", "never-recommended"}) {
		t.Fatalf("want the description and the label each named; got %v:\n%s", rulesNamed(parts), stderr)
	}
}

// TestLongChipHeaderIsRefusedWithoutAMode: a header in the chip grammar makes
// the question abcd's with no mode at all, so where abcd does not manage the
// repository a chip over the width limit is still refused on its width alone.
func TestLongChipHeaderIsRefusedWithoutAMode(t *testing.T) {
	repo := unmanagedRepo(t)
	q := wellBuilt()
	q.Header = "Product Q12/40"
	_, stderr, code := runGuard(askPayload(t, repo, q), "guard", "hook")
	if code != 2 {
		t.Fatalf("want exit 2; got %d (stderr %q)", code, stderr)
	}
	_, parts, lines := refusal(t, stderr)
	if !slices.Equal(rulesNamed(parts), []string{"header"}) || !strings.Contains(lines[0], "12 columns") {
		t.Errorf("want one header finding naming the 12-column limit; got %v:\n%s", rulesNamed(parts), stderr)
	}
}

// TestQuestionRefusalIsBoundedOnAFlood: a payload of 40,000 tabs, or one tab of
// 30,000 options, is refused in a few lines, not one line per broken field:
// the head line keeps the true count, a bounded number of parts are named, and
// one line says how many more there are (review-askGuard-security finding 1).
func TestQuestionRefusalIsBoundedOnAFlood(t *testing.T) {
	repo := unmanagedRepo(t)
	tabs := make([]map[string]any, 40000)
	for i := range tabs {
		tabs[i] = map[string]any{"header": "Product Q1"}
	}
	opts := make([]map[string]any, 30000)
	for i := range opts {
		opts[i] = map[string]any{"label": "*", "description": ""}
	}
	q := wellBuilt()
	flood := map[string]any{"header": q.Header, "question": strings.Repeat("- a \u2014\n", 1000) + q.Question, "options": q.Options}
	for name, input := range map[string]any{
		"40,000 tabs":       map[string]any{"questions": tabs},
		"30,000 options":    map[string]any{"questions": []any{map[string]any{"header": "Product Q1", "question": q.Question, "options": opts}}},
		"1,000 dashed list": map[string]any{"questions": []any{flood}},
	} {
		t.Run(name, func(t *testing.T) {
			payload := askToolPayload(t, repo, input)
			if len(payload) > maxHookStdinBytes {
				t.Fatalf("the payload is %d bytes, over the hook's cap", len(payload))
			}
			stdout, stderr, code := runGuard(payload, "guard", "hook")
			if code != 2 || stdout != "" {
				t.Fatalf("want exit 2 and no stdout; code=%d stdout=%d bytes", code, len(stdout))
			}
			if len(stderr) > 4096 {
				t.Fatalf("the refusal is %d bytes; it must stay under 4 KiB:\n%.600s", len(stderr), stderr)
			}
			// refusal checks the head line's count against the parts named
			// and the closing line's count of the rest.
			if _, parts, _ := refusal(t, stderr); len(parts) > maxRefusalParts {
				t.Errorf("the refusal names %d parts, over the %d it may name:\n%s", len(parts), maxRefusalParts, stderr)
			}
		})
	}
}
