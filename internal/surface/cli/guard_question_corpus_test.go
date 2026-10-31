package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The verb and the hook decide alike (spc-2610301917245442, step 1): on every
// draft of testdata/questions/corpus/, in every checkout state the gate reads,
// `abcd guard question` and `abcd guard hook` of the same build give the same
// answer, because both call questionState and question.Decide.

// questionCorpusState is one checkout state a draft is judged in. Each makes
// a fresh fixture and leaves the process standing in it.
type questionCorpusState struct {
	name  string
	make  func(t *testing.T) string
	badge bool // the hook marks an admitted question of abcd's open here
}

var questionCorpusStates = []questionCorpusState{
	{"outside every repository", func(t *testing.T) string {
		t.Setenv("HOME", t.TempDir())
		dir := realPath(t, t.TempDir())
		t.Chdir(dir)
		return dir
	}, false},
	{"unmanaged", unmanagedRepo, false},
	{"managed", managedCheckout, true},
	{"managed, product thinker", func(t *testing.T) string {
		root := managedCheckout(t)
		setMode(t, root, mode.ProductThinker)
		return root
	}, true},
	{"managed, facilitator", func(t *testing.T) string {
		root := managedCheckout(t)
		setMode(t, root, mode.Facilitator)
		return root
	}, true},
	{"away", func(t *testing.T) string {
		root := awayCheckout(t)
		setAwayHere(t)
		return root
	}, true},
	{"away record unreadable", func(t *testing.T) string {
		root := awayCheckout(t)
		setAwayHere(t)
		if err := os.Chmod(awayStore(t, root).RecordPath(), 0o666); err != nil {
			t.Fatal(err)
		}
		return root
	}, true},
	{"mode store unreadable", func(t *testing.T) string {
		root := managedCheckout(t)
		if err := os.WriteFile(modeFile(root), []byte("sideways\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return root
	}, true},
	{"root unfindable", func(t *testing.T) string {
		root := managedCheckout(t)
		setMode(t, root, mode.ProductThinker)
		fakeGit(t, `case "$*" in *rev-parse*) echo 'fatal: refused' >&2; exit 128;; esac
exec "$REAL_GIT" "$@"`)
		return root
	}, false},
}

// denyHeadRe reads the count a limits refusal's head line gives.
var denyHeadRe = regexp.MustCompile(`^Blocked by the abcd guard \(question tool\): ([0-9]+) part\(s\)`)

// hookOutcome reads what the hook did with a call as one of Decide's
// outcomes. A silent admission is admit or admit-unchecked; it is told apart
// only where the badge shows, by whether the question was marked open.
func hookOutcome(t *testing.T, stdout, stderr string, code int, root string, badge bool) (outs []question.Outcome, reason, note string) {
	t.Helper()
	switch {
	case isDeny(stdout):
		reason = mustDeny(t, stdout, stderr, code)
		if strings.Contains(reason, "the person is away") {
			return []question.Outcome{question.OutcomeRefuseAway}, reason, ""
		}
		return []question.Outcome{question.OutcomeRefuse}, reason, ""
	case code == 0 && stdout != "":
		var out struct {
			HookSpecificOutput struct {
				AdditionalContext string `json:"additionalContext"`
			} `json:"hookSpecificOutput"`
		}
		if err := json.Unmarshal([]byte(stdout), &out); err != nil {
			t.Fatalf("the hook's stdout is neither a deny nor a note: %q", stdout)
		}
		return []question.Outcome{question.OutcomeAdmitWithRowsNote}, "", out.HookSpecificOutput.AdditionalContext
	case code == 0:
		if badge {
			if markedOpen(t, root) {
				return []question.Outcome{question.OutcomeAdmit}, "", ""
			}
			return []question.Outcome{question.OutcomeAdmitUnchecked}, "", ""
		}
		return []question.Outcome{question.OutcomeAdmit, question.OutcomeAdmitUnchecked}, "", ""
	case code == 1:
		return []question.Outcome{question.OutcomeUnchecked}, "", ""
	}
	t.Fatalf("the hook answered with exit %d (stdout %q, stderr %q)", code, stdout, stderr)
	return nil, "", ""
}

func findingLine(f questionFindingJSON) string {
	return termsafe.Sanitize(question.Finding{Tab: f.Tab, Part: f.Part, Rule: question.Rule(f.Rule), Value: f.Value, Limit: f.Limit, Remedy: f.Remedy}.String())
}

// TestGuardQuestionAndHookDecideAlike: for every draft of the corpus in every
// state, the verb's outcome is the hook's, and the findings the hook names in
// its deny or its note are the verb's.
func TestGuardQuestionAndHookDecideAlike(t *testing.T) {
	ents, err := os.ReadDir(questionCorpusDir)
	if err != nil {
		t.Fatal(err)
	}
	var drafts []string
	for _, e := range ents {
		if name, ok := strings.CutSuffix(e.Name(), ".json"); ok {
			drafts = append(drafts, name)
		}
	}
	if len(drafts) < 20 {
		t.Fatalf("the corpus holds %d drafts; want the 21-fault case, the 25-row case, one per rule and the clean drafts", len(drafts))
	}
	for _, rule := range question.Rules() {
		if !slices.Contains(drafts, "rule-"+string(rule)) {
			t.Errorf("the corpus has no draft for the %s rule", rule)
		}
	}
	for _, st := range questionCorpusStates {
		for _, name := range drafts {
			t.Run(st.name+"/"+name, func(t *testing.T) {
				root := st.make(t)
				res, _, _ := runQuestionVerb(t, name)
				var in struct {
					Questions json.RawMessage `json:"questions"`
				}
				if err := json.Unmarshal(corpusDraft(t, name), &in); err != nil {
					t.Fatal(err)
				}
				payload := askToolPayload(t, root, map[string]json.RawMessage{"questions": in.Questions})
				stdout, stderr, code := runGuard(payload, "guard", "hook")
				outs, reason, note := hookOutcome(t, stdout, stderr, code, root, st.badge)
				if !slices.Contains(outs, res.Hook) {
					t.Fatalf("the verb says %q, the hook did %v (exit %d, stdout %q, stderr %q)", res.Hook, outs, code, stdout, stderr)
				}
				var refusing, rows []questionFindingJSON
				for _, f := range res.Findings {
					if f.Refuses {
						refusing = append(refusing, f)
					} else {
						rows = append(rows, f)
					}
				}
				switch res.Hook {
				case question.OutcomeRefuse:
					m := denyHeadRe.FindStringSubmatch(reason[strings.Index(reason, "Blocked by"):])
					if m == nil {
						t.Fatalf("the deny has no head line:\n%s", reason)
					}
					if n, _ := strconv.Atoi(m[1]); n != len(refusing) {
						t.Errorf("the deny counts %d parts, the verb %d", n, len(refusing))
					}
					for _, f := range refusing[:min(len(refusing), maxRefusalParts)] {
						if !strings.Contains(reason, findingLine(f)) {
							t.Errorf("the deny lacks the verb's finding %q:\n%s", findingLine(f), reason)
						}
					}
					if more := len(refusing) - maxRefusalParts; more > 0 && !strings.Contains(reason, fmt.Sprintf("... and %d more part(s)", more)) {
						t.Errorf("the deny does not count the %d findings it leaves out:\n%s", more, reason)
					}
				case question.OutcomeAdmitWithRowsNote:
					if len(refusing) != 0 || len(rows) == 0 {
						t.Errorf("a rows note with findings %+v", res.Findings)
					}
					for _, f := range rows {
						if !strings.Contains(note, fmt.Sprintf("tab %d is %s", f.Tab, f.Value)) {
							t.Errorf("the note lacks tab %d's rows %q:\n%s", f.Tab, f.Value, note)
						}
					}
				case question.OutcomeRefuseAway, question.OutcomeAdmitUnchecked:
					if len(res.Findings) != 0 {
						t.Errorf("%s with findings %+v", res.Hook, res.Findings)
					}
				case question.OutcomeAdmit:
					if len(res.Findings) != 0 {
						t.Errorf("admit with findings %+v", res.Findings)
					}
				}
				if rule, ok := strings.CutPrefix(name, "rule-"); ok && st.name == "unmanaged" &&
					!slices.ContainsFunc(res.Findings, func(f questionFindingJSON) bool { return f.Rule == rule }) {
					t.Errorf("the %s draft draws no %s finding: %+v", name, rule, res.Findings)
				}
				if res.Partial != (len(res.Unread) > 0) {
					t.Errorf("partial %v with unread %+v", res.Partial, res.Unread)
				}
			})
		}
	}
}
