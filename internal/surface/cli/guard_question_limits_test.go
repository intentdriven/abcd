package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/gittest"
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
		if strings.HasPrefix(l, rowsNotRefusingPrefix) {
			// The rows findings that follow are named but refuse nothing
			// (iss-2610100626327722); rowsAfterRefusal reads them.
			break
		}
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

// rowsNotRefusingPrefix opens the line a refusal for another fault writes
// before the rows findings it names but does not refuse on.
const rowsNotRefusingPrefix = "The rows limit does not refuse on its own"

// rowsAfterRefusal returns the parts a refusal names after its rows line: the
// rows findings that are not a cause of the refusal. It returns nil when the
// refusal has no rows line.
func rowsAfterRefusal(reason string) (line string, parts []namedPart) {
	all := strings.Split(strings.TrimRight(reason, "\n"), "\n")
	for i, l := range all {
		if !strings.HasPrefix(l, rowsNotRefusingPrefix) {
			continue
		}
		line = l
		for _, r := range all[i+1:] {
			if m := findingLineRe.FindStringSubmatch(r); m != nil {
				tab := 0
				if strings.HasPrefix(m[1], "tab ") {
					tab, _ = strconv.Atoi(strings.TrimPrefix(m[1], "tab "))
				}
				parts = append(parts, namedPart{Tab: tab, Part: m[2], Rule: m[3]})
			}
		}
		break
	}
	return line, parts
}

func rulesNamed(parts []namedPart) []string {
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = p.Rule
	}
	return out
}

// TestRoutingQuestionOf20261003IsRefused is criterion A1: the routing question
// of 2026-10-03, rebuilt as a fixture, is refused with the host's deny and
// exactly the parts its findings list names, with the mode at product-thinker.
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
		More     int         `json:"more"`
	}
	if err := json.Unmarshal(rawWant, &want); err != nil {
		t.Fatal(err)
	}
	root := managedCheckout(t)
	setMode(t, root, mode.ProductThinker)

	stdout, stderr, code := runGuard(askToolPayload(t, root, json.RawMessage(input)), "guard", "hook")
	reason := mustDeny(t, stdout, stderr, code)
	head, got, _ := refusal(t, reason)
	if !slices.Equal(got, want.Findings) {
		t.Errorf("the refusal named\n%v\nwant exactly\n%v\nreason:\n%s", got, want.Findings, reason)
	}
	if n := strconv.Itoa(len(want.Findings) + want.More); !strings.Contains(head, ": "+n+" part(s)") {
		t.Errorf("the head line must count %s part(s), the named and the %d more:\n%s", n, want.More, head)
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
// each case is refused with the host's deny and a line naming the rule, the value and
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
			reason := mustDeny(t, stdout, stderr, code)
			_, parts, lines := refusal(t, reason)
			hit := false
			for i, p := range parts {
				if p.Rule == tc.rule && strings.Contains(lines[i], strconv.Quote(tc.value)) && strings.Contains(lines[i], tc.limit) {
					hit = true
				}
			}
			if !hit {
				t.Errorf("no line names the rule %q, the value %q and the limit %q:\n%s", tc.rule, tc.value, tc.limit, reason)
			}
		})
	}
}

// TestProductThinkerQuestionNamesNoRecordOrCommand is criterion R4 at the
// hook. With the mode at product-thinker a record number, a command named by
// one of the binary's verbs, and a label in backticks are each refused; the
// same questions chipped for the facilitator are admitted. With no mode store,
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
			stdout, stderr, code := runGuard(askPayload(t, root, q), "guard", "hook")
			reason := mustDeny(t, stdout, stderr, code)
			if _, parts, _ := refusal(t, reason); !slices.Equal(rulesNamed(parts), []string{"register"}) {
				t.Errorf("product thinker: want one register finding; got %v:\n%s", rulesNamed(parts), reason)
			}

			// The chip names the addressee (iss-2610100626211810): the same
			// question chipped for the facilitator is theirs to read.
			setMode(t, root, mode.Facilitator)
			tech := q
			tech.Header = "Tech Q2"
			if stdout, stderr, code := runGuard(askPayload(t, root, tech), "guard", "hook"); code != 0 || stdout != "" || stderr != "" {
				t.Errorf("facilitator: the mechanism and the ids are theirs; code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}

			repo := unmanagedRepo(t)
			stdout, stderr, code = runGuard(askPayload(t, repo, q), "guard", "hook")
			reason = mustDeny(t, stdout, stderr, code)
			if _, parts, _ := refusal(t, reason); !slices.Equal(rulesNamed(parts), []string{"register"}) {
				t.Errorf("no mode store, Product chip: want one register finding; got %v:\n%s", rulesNamed(parts), reason)
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
	stdout, stderr, code := runGuard(askPayload(t, root, q), "guard", "hook")
	reason := mustDeny(t, stdout, stderr, code)
	_, parts, lines := refusal(t, reason)
	if len(parts) != 1 || parts[0].Rule != "never-recommended" || parts[0].Part != "option 2 label" {
		t.Fatalf("want one never-recommended finding on option 2's label; got %v:\n%s", parts, reason)
	}
	if !strings.Contains(lines[0], strconv.Quote("Number only *")) {
		t.Errorf("the refusal does not name the label:\n%s", reason)
	}
	if !strings.Contains(lines[0], question.RecommendedRemedy) {
		t.Errorf("the remedy must name the host's instruction and abcd's rule reversing it:\n%s", reason)
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

// TestAbcdChipWhileManagedIsNeverRefusedOnTheMode is the other half of the
// mode gate's scope (iss-2610100626211810): an abcd chip names whom the
// question is for, so while the mode reads managed it is admitted and sets the
// mode from the chip. With a field finding it is refused on that finding
// alone: no line about the mode, the mode left managed, nothing marked open.
func TestAbcdChipWhileManagedIsNeverRefusedOnTheMode(t *testing.T) {
	root := managedCheckout(t)
	q := wellBuilt()
	q.Header = "Product Q1"
	q.Options[0].Label = "Keep it (Recommended)"
	stdout, stderr, code := runGuard(askPayload(t, root, q), "guard", "hook")
	reason := mustDeny(t, stdout, stderr, code)
	if _, parts, _ := refusal(t, reason); !slices.Equal(rulesNamed(parts), []string{"never-recommended"}) {
		t.Errorf("want the one field finding; got %v:\n%s", rulesNamed(parts), reason)
	}
	for _, not := range []string{"mode reads managed", "abcd mode"} {
		if strings.Contains(reason, not) {
			t.Errorf("a chipped question is never refused on the mode, yet the refusal says %q:\n%s", not, reason)
		}
	}
	if got, _ := mode.ReadAt(root); got != mode.Managed {
		t.Errorf("a refused question set the mode to %q", got)
	}
	if markedOpen(t, root) {
		t.Error("a refused question was marked open")
	}

	q.Options[0].Label = "Keep it"
	if stdout, stderr, code := runGuard(askPayload(t, root, q), "guard", "hook"); code != 0 || stdout != "" || stderr != "" {
		t.Fatalf("the fixed question must be admitted silently; code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if got, _ := mode.ReadAt(root); got != mode.ProductThinker {
		t.Errorf("mode = %q, want the chip's product-thinker", got)
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
	stdout, stderr, code := runGuard(askPayload(t, repo, q), "guard", "hook")
	reason := mustDeny(t, stdout, stderr, code)
	for _, r := range reason {
		if (r < 0x20 && r != '\n') || r == 0x7f || (r >= 0x80 && r <= 0x9f) || r == '\u202e' || r == '\u200b' {
			t.Fatalf("the refusal echoes an unsafe rune %U:\n%q", r, reason)
		}
	}
	if _, parts, _ := refusal(t, reason); !slices.Equal(rulesNamed(parts), []string{"meaning-sentences", "never-recommended"}) {
		t.Fatalf("want the description and the label each named; got %v:\n%s", rulesNamed(parts), reason)
	}
}

// TestLongChipHeaderIsRefusedWithoutAMode: a header in the chip grammar makes
// the question abcd's with no mode at all, so where abcd does not manage the
// repository a chip over the width limit is still refused on its width alone.
func TestLongChipHeaderIsRefusedWithoutAMode(t *testing.T) {
	repo := unmanagedRepo(t)
	q := wellBuilt()
	q.Header = "Product Q12/40"
	stdout, stderr, code := runGuard(askPayload(t, repo, q), "guard", "hook")
	reason := mustDeny(t, stdout, stderr, code)
	_, parts, lines := refusal(t, reason)
	if !slices.Equal(rulesNamed(parts), []string{"header"}) || !strings.Contains(lines[0], "12 columns") {
		t.Errorf("want one header finding naming the 12-column limit; got %v:\n%s", rulesNamed(parts), reason)
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
			reason := mustDeny(t, stdout, stderr, code)
			if len(reason) > 4096 {
				t.Fatalf("the refusal is %d bytes; it must stay under 4 KiB:\n%.600s", len(reason), reason)
			}
			// refusal checks the head line's count against the parts named
			// and the closing line's count of the rest.
			if _, parts, _ := refusal(t, reason); len(parts) > maxRefusalParts {
				t.Errorf("the refusal names %d parts, over the %d it may name:\n%s", len(parts), maxRefusalParts, reason)
			}
		})
	}
}

// TestModeMadeQuestionAbcdsSaysSo: while the mode names somebody every question
// is abcd's, so another tool's question asked then is held to abcd's rules. Its
// refusal says why in one line, naming the mode's person, so an agent that did
// not write the question does not loop on it; a question carrying abcd's chip
// is abcd's on the chip, and its refusal carries no such line
// (review-askGuard-security finding 4).
func TestModeMadeQuestionAbcdsSaysSo(t *testing.T) {
	foreign := hostQuestion{Header: "Ship", Question: "Ship it?", Options: []hostOption{{Label: "Yes"}, {Label: "No"}}}
	for st, who := range map[mode.State]string{mode.ProductThinker: "the product thinker", mode.Facilitator: "the technical facilitator"} {
		root := managedCheckout(t)
		setMode(t, root, st)
		stdout, stderr, code := runGuard(askPayload(t, root, foreign), "guard", "hook")
		reason := mustDeny(t, stdout, stderr, code)
		want := "treated as abcd's because the mode names " + who + "; another tool's question asked now is held to abcd's rules"
		if !strings.Contains(reason, want) {
			t.Errorf("%s: the refusal must say why the question is abcd's (%q):\n%s", st, want, reason)
		}
		refusal(t, reason)

		q := wellBuilt()
		q.Options[0].Label = "Keep it (Recommended)"
		if stdout, _, _ := runGuard(askPayload(t, root, q), "guard", "hook"); strings.Contains(stdout, "treated as abcd's") {
			t.Errorf("%s: a question carrying abcd's chip is abcd's on the chip, not the mode:\n%s", st, stdout)
		}
	}
}

// TestModeMadeQuestionSurvivesAGitHoldingItsPipe: a question without abcd's
// chip is abcd's only through the mode store, and the gate finds the store
// through git's toplevel. The hook runs in-process here, so nothing is left
// unflushed: TestModeMadeQuestionAbcdsSaysSo's empty stdout on a loaded macOS
// leg was the gate deciding to write nothing. git's toplevel answer missed the
// runner's WaitDelay, the root lookup failed, the gate read the checkout as one
// with no mode store and admitted the question silently, and the test read the
// empty stdout as "not one hook output object: EOF" (iss-2610090642394550; the
// lookup's own fixes are iss-2610100846469473 and iss-2610272041031418).
// gittest.SlowPipeGit holds git's output open without load, on every toplevel
// call, and the refusal still arrives.
func TestModeMadeQuestionSurvivesAGitHoldingItsPipe(t *testing.T) {
	foreign := hostQuestion{Header: "Ship", Question: "Ship it?", Options: []hostOption{{Label: "Yes"}, {Label: "No"}}}
	root := managedCheckout(t)
	setMode(t, root, mode.ProductThinker)
	calls := gittest.SlowPipeGit(t, "--show-toplevel", -1)

	stdout, stderr, code := runGuard(askPayload(t, root, foreign), "guard", "hook")
	if calls() == 0 {
		t.Fatal("the gate never asked git for the toplevel, so the held pipe was never in its way")
	}
	reason := mustDeny(t, stdout, stderr, code)
	if want := "treated as abcd's because the mode names the product thinker"; !strings.Contains(reason, want) {
		t.Errorf("the refusal must say why the question is abcd's (%q):\n%s", want, reason)
	}
}

// TestUnreadableQuestionPayloadFailsOpenAsAQuestion: a question-tool payload
// whose outer JSON the hook cannot decode (a tool_input that is a string, or a
// sibling nested past the decoder's depth) lets the question run on exit 1
// with the question's wording, not the shell's, and echoes no Go type from the
// decoder. Where the tool name cannot be read at all, the wording names no
// tool (review-askGuard-security finding 5).
func TestUnreadableQuestionPayloadFailsOpenAsAQuestion(t *testing.T) {
	repo := unmanagedRepo(t)
	cwd, err := json.Marshal(repo)
	if err != nil {
		t.Fatal(err)
	}
	deep := strings.Repeat("[", 20000) + strings.Repeat("]", 20000)
	tool := `"tool_name":"` + questionTools[0] + `"`
	for name, tc := range map[string]struct {
		payload string
		want    string
	}{
		"tool_input a string":      {`{"cwd":` + string(cwd) + `,` + tool + `,"tool_input":"x"}`, "The question runs UNGATED"},
		"a sibling nested deeply":  {`{"cwd":` + string(cwd) + `,` + tool + `,"tool_input":{"questions":[],"x":` + deep + `}}`, "The question runs UNGATED"},
		"the tool after the fault": {`{"cwd":` + string(cwd) + `,"tool_input":{"x":` + deep + `},` + tool + `}`, "The call runs UNGUARDED"},
	} {
		t.Run(name, func(t *testing.T) {
			stdout, stderr, code := runGuard(tc.payload, "guard", "hook")
			if code != 1 || stdout != "" {
				t.Fatalf("want exit 1 and no stdout; code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			if !strings.Contains(stderr, "NOT CHECKED") || !strings.Contains(stderr, tc.want) {
				t.Errorf("the fail-open must say NOT CHECKED and %q; stderr = %q", tc.want, stderr)
			}
			for _, banned := range []string{"This command runs UNGUARDED", "jsontext", "json:", "struct", "Go value"} {
				if strings.Contains(stderr, banned) {
					t.Errorf("the fail-open must not carry %q; stderr = %q", banned, stderr)
				}
			}
		})
	}
}

// TestHookDenyNamesTheRefusedKeyAndNoRule (itd-2610301914350541 decision 7):
// a question value read and refused refuses every abcd question, however well
// built, with a deny naming the key, the value, the bounds (or, for a personal
// value that loosens, the limit in force and the file that set it), and the
// file, and no asking rule, so the agent fixes the configuration rather than
// redrafting. Nothing falls through to the defaults, and nothing is marked
// open.
func TestHookDenyNamesTheRefusedKeyAndNoRule(t *testing.T) {
	cases := []struct {
		name    string
		repo    string
		machine string
		want    []string
	}{
		{name: "outside the bounds", repo: `{"question":{"rows":100}}`,
			want: []string{".abcd/config.json (repo layer)", "question.rows is 100", "outside 24 to 60"}},
		{name: "not a whole number", repo: `{"question":{"label_words":5.5}}`,
			want: []string{".abcd/config.json (repo layer)", "question.label_words is 5.5", "4 to 8"}},
		{name: "a key the namespace does not claim", repo: `{"question":{"options":3}}`,
			want: []string{".abcd/config.json", "question.options"}},
		{name: "a personal value that loosens", machine: `{"question":{"rows":40}}`,
			want: []string{abcdhome.Display(layered.Config.MachineRel) + " (machine layer)", "question.rows is 40", "looser than 32 (bundled)"}},
		{name: "a personal value looser than the repository's", repo: `{"question":{"rows":30}}`, machine: `{"question":{"rows":40}}`,
			want: []string{abcdhome.Display(layered.Config.MachineRel) + " (machine layer)", "question.rows is 40", "looser than 30 (.abcd/config.json)"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := managedCheckout(t)
			if c.repo != "" {
				writeRepoQuestionConfig(t, root, c.repo)
			}
			if c.machine != "" {
				writeMachineQuestionConfig(t, c.machine)
			}
			stdout, stderr, code := runGuard(askPayload(t, root, wellBuilt()), "guard", "hook")
			reason := mustDeny(t, stdout, stderr, code)
			lines := strings.Split(reason, "\n")
			if !strings.HasPrefix(lines[0], "Blocked by the abcd guard (question tool): a question limit in this repository's configuration is refused") {
				t.Errorf("the head line must say a question limit is refused; reason:\n%s", reason)
			}
			for _, w := range c.want {
				if !strings.Contains(reason, w) {
					t.Errorf("the deny must name %q; reason:\n%s", w, reason)
				}
			}
			for _, b := range question.Bounds {
				if !strings.Contains(reason, b.Key+" takes "+b.String()) {
					t.Errorf("the deny must state %s's bounds; reason:\n%s", b.Key, reason)
				}
			}
			for _, ln := range lines {
				if findingLineRe.MatchString(ln) {
					t.Errorf("the deny must name no asking rule; it carries the finding line %q", ln)
				}
			}
			if strings.Contains(reason, "break abcd's asking rules") {
				t.Errorf("the deny must not read as an asking-rules refusal; reason:\n%s", reason)
			}
			if markedOpen(t, root) {
				t.Error("a question refused on its configuration was marked open")
			}
		})
	}
}

// TestUnlabelledQuestionRunsUncheckedUnderARefusedValue: a refused value
// refuses only abcd's question. Another tool's question, one with no header in
// abcd's grammar asked while the mode names nobody, runs unchecked and
// unmarked whatever the configuration says, in a managed checkout and
// elsewhere.
func TestUnlabelledQuestionRunsUncheckedUnderARefusedValue(t *testing.T) {
	foreign := hostQuestion{
		Header:   "Pick a theme",
		Question: "Which theme?",
		Options:  []hostOption{{Label: "Dark (Recommended)"}, {Label: "Light"}},
	}
	root := managedCheckout(t)
	writeRepoQuestionConfig(t, root, `{"question":{"rows":100}}`)
	stdout, stderr, code := runGuard(askPayload(t, root, foreign), "guard", "hook")
	mustRunSilently(t, stdout, stderr, code)
	if markedOpen(t, root) {
		t.Error("another tool's question was marked open")
	}
	repo := unmanagedRepo(t)
	writeRepoQuestionConfig(t, repo, `{"question":{"rows":100}}`)
	stdout, stderr, code = runGuard(askPayload(t, repo, foreign), "guard", "hook")
	mustRunSilently(t, stdout, stderr, code)
}

// TestHookChecksAtTheDefaultsOnASymlinkedConfig (itd-2610301914350541 decision 8,
// approach 3): a .abcd/config.json the resolver will not read, here a
// symbolic link, is an unread layer. abcd's question is still checked, at
// abcd's defaults and not at the 40 rows the linked file sets: a well-built
// question runs at exit 1 with the NOT CHECKED line naming the file, one that
// breaks a rule is refused on it, and a 36-row tab is noted against the
// default 32. Another tool's question runs with nothing said.
func TestHookChecksAtTheDefaultsOnASymlinkedConfig(t *testing.T) {
	repo := unmanagedRepo(t)
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
	mustCheckAtTheDefaults(t, repo, ".abcd/config.json")
}

// TestHookChecksAtTheDefaultsOnAnUnparseableConfig (approach 3): a
// .abcd/config.json the resolver reads but cannot parse (not JSON, a key
// given twice, or over the resolver's size cap) is an unread layer too, and
// never turns the check off: abcd's question is held to the defaults, and
// the NOT CHECKED line names the file.
func TestHookChecksAtTheDefaultsOnAnUnparseableConfig(t *testing.T) {
	for name, body := range map[string]string{
		"not JSON":       `{"question":{"rows":40}`,
		"a key twice":    `{"question":{"rows":40},"question":{"rows":40}}`,
		"over the cap":   `{"question":{"rows":40}}` + strings.Repeat(" ", layered.MaxFileBytes),
		"trailing bytes": `{"question":{"rows":40}} {}`,
	} {
		t.Run(name, func(t *testing.T) {
			repo := unmanagedRepo(t)
			writeRepoQuestionConfig(t, repo, body)
			mustCheckAtTheDefaults(t, repo, ".abcd/config.json")
		})
	}
}

// TestHookChecksAtTheDefaultsWithHomeUnset (approach 3): with HOME unset the
// machine layer cannot be read, which is an unread layer: abcd's question is
// checked at the defaults, loudly, and never let through unchecked.
func TestHookChecksAtTheDefaultsWithHomeUnset(t *testing.T) {
	repo := unmanagedRepo(t)
	writeRepoQuestionConfig(t, repo, `{"question":{"rows":40}}`)
	t.Setenv("HOME", "")
	mustCheckAtTheDefaults(t, repo, "the home directory is unresolved")
}

// mustCheckAtTheDefaults asserts the gate's answer for abcd's question in
// repo while a layer of the question limits is unread: checked at abcd's
// defaults, with the NOT CHECKED line naming what was not read (want) on
// stderr for an admission, at the head of the reason for a deny, and at the
// head of a rows note. Another tool's question runs with nothing said.
func mustCheckAtTheDefaults(t *testing.T, repo, want string) {
	t.Helper()
	const head = "abcd guard: NOT CHECKED at this repository's question limits"
	stdout, stderr, code := runGuard(askPayload(t, repo, wellBuilt()), "guard", "hook")
	if code != 1 || stdout != "" {
		t.Fatalf("a well-built question under an unread config: want exit 1 and nothing on stdout; code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.HasPrefix(stderr, head) || !strings.Contains(stderr, want) || !strings.Contains(stderr, "checked at abcd's defaults") {
		t.Errorf("the loud line must name %q and say the question is checked at the defaults; stderr = %q", want, stderr)
	}
	if strings.Contains(stderr, "UNGATED") {
		t.Errorf("the question is checked at the defaults, so it must not be said to run ungated; stderr = %q", stderr)
	}

	broken := wellBuilt()
	broken.Options[0].Label = "Role and number (Recommended)"
	stdout, stderr, code = runGuard(askPayload(t, repo, broken), "guard", "hook")
	reason := mustDeny(t, stdout, stderr, code)
	if !strings.HasPrefix(reason, head) || !strings.Contains(reason, "break abcd's asking rules") || !strings.Contains(reason, "(never-recommended)") {
		t.Errorf("a question that breaks a rule must be refused on it, the line first; reason:\n%s", reason)
	}

	if question.Default.Rows >= 36 {
		t.Fatalf("precondition: the default rows limit (%d) must be below 36", question.Default.Rows)
	}
	stdout, stderr, code = runGuard(askPayload(t, repo, questionOfRows(t, 36)), "guard", "hook")
	note := mustAdmitWithNote(t, stdout, stderr, code)
	if !strings.HasPrefix(note, head) || !strings.Contains(note, fmt.Sprintf("limit: %d rows at 80 columns", question.Default.Rows)) {
		t.Errorf("a 36-row tab must be noted against the default rows, the line first; note = %q", note)
	}

	foreign := hostQuestion{Header: "Pick a theme", Question: "Which theme?", Options: []hostOption{{Label: "Dark"}, {Label: "Light"}}}
	stdout, stderr, code = runGuard(askPayload(t, repo, foreign), "guard", "hook")
	mustRunSilently(t, stdout, stderr, code)
}

// TestHookNamesAnUnreadMachineConfig (itd-2610301914350541 decision 8): a
// ~/.abcd.noindex/config.json the account cannot vouch for, here writable by
// others, is unread whatever keys it holds, even none under question: abcd's
// question is checked at the defaults and the gate names that file. A symlinked ~/.abcd.noindex
// holding no config.json reads as absent: the limits are the defaults and
// nothing is said.
func TestHookNamesAnUnreadMachineConfig(t *testing.T) {
	t.Run("writable by others", func(t *testing.T) {
		repo := unmanagedRepo(t)
		p := writeMachineQuestionConfig(t, `{}`)
		if err := os.Chmod(p, 0o666); err != nil {
			t.Fatal(err)
		}
		mustCheckAtTheDefaults(t, repo, abcdhome.Display(layered.Config.MachineRel))
	})
	t.Run("a symlinked home with no config.json", func(t *testing.T) {
		repo := unmanagedRepo(t)
		if err := os.Symlink(t.TempDir(), abcdhome.Path(os.Getenv("HOME"))); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, code := runGuard(askPayload(t, repo, wellBuilt()), "guard", "hook")
		mustRunSilently(t, stdout, stderr, code)
	})
	t.Run("a symlinked home holding config.json", func(t *testing.T) {
		repo := unmanagedRepo(t)
		dotfiles := t.TempDir()
		if err := os.WriteFile(filepath.Join(dotfiles, filepath.FromSlash(layered.Config.MachineRel)), []byte(`{}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(dotfiles, abcdhome.Path(os.Getenv("HOME"))); err != nil {
			t.Fatal(err)
		}
		mustCheckAtTheDefaults(t, repo, abcdhome.Display())
	})
}

// Fixtures for the hook's read of the question limits a repository sets
// (itd-2610301914350541, spc-2610301950037469 step 2): the repository's
// .abcd/config.json and the person's own ~/.abcd.noindex/config.json, and a
// question of an exact height.

// writeRepoQuestionConfig writes body as the repository's .abcd/config.json.
func writeRepoQuestionConfig(t *testing.T, repo, body string) string {
	t.Helper()
	p := filepath.Join(repo, filepath.FromSlash(layered.Config.RepoRel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// writeMachineQuestionConfig writes body as the person's own
// ~/.abcd.noindex/config.json under the sandboxed HOME, owner-only, as the
// home-declaration guard reads it.
func writeMachineQuestionConfig(t *testing.T, body string) string {
	t.Helper()
	dir := abcdhome.Path(os.Getenv("HOME"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, filepath.FromSlash(layered.Config.MachineRel))
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// rowsOfRe reads the estimate a rows finding names.
var rowsOfRe = regexp.MustCompile(`^([0-9]+) rows:`)

// rowsOf is the rows the check estimates q takes, read from the rows finding
// it names at a limit of one row, so the test measures with the check's own
// estimate and never a copy of it.
func rowsOf(t *testing.T, q hostQuestion) int {
	t.Helper()
	tab := question.Tab{Header: q.Header, Text: q.Question}
	for _, o := range q.Options {
		tab.Options = append(tab.Options, question.Choice{Label: o.Label, Description: o.Description})
	}
	one := question.Bounded(question.Default, 1, question.Default.Configurable()[1], question.Default.Configurable()[2])
	for _, f := range question.CheckLimits(question.Fields{Tabs: []question.Tab{tab}}, one, question.Addressee{Person: question.Unnamed}) {
		if f.Rule == question.RuleRows {
			m := rowsOfRe.FindStringSubmatch(f.Value)
			if m == nil {
				t.Fatalf("the rows finding's value is not in its form: %q", f.Value)
			}
			n, _ := strconv.Atoi(m[1])
			return n
		}
	}
	t.Fatal("no rows finding at a limit of one row")
	return 0
}

// questionOfRows is wellBuilt grown, one short line of material at a time, to
// exactly rows rows, every other rule holding.
func questionOfRows(t *testing.T, rows int) hostQuestion {
	t.Helper()
	q := wellBuilt()
	base := q.Question
	var more []string
	for rowsOf(t, q) < rows {
		more = append(more, fmt.Sprintf("Line %d of the material is short.", len(more)+1))
		q.Question = strings.Join(more, "\n") + "\n\n" + base
	}
	if got := rowsOf(t, q); got != rows {
		t.Fatalf("precondition: the fixture is %d rows, want %d", got, rows)
	}
	return q
}

// mustRunSilently asserts the hook admitted the call with nothing to say:
// exit 0 and nothing on either stream.
func mustRunSilently(t *testing.T, stdout, stderr string, code int) {
	t.Helper()
	if code != 0 || stdout != "" || stderr != "" {
		t.Fatalf("want the question admitted with nothing said; code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}
