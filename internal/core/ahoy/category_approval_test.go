package ahoy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
)

// heldToEveryCheck holds q to the structural check and to the limits, and
// fails on any finding of either.
func heldToEveryCheck(t *testing.T, q question.Question) {
	t.Helper()
	a := question.Ask{Questions: []question.Question{q}}
	for _, f := range question.Check(a) {
		t.Errorf("%s: structural: %s", q.ID, f)
	}
	for _, f := range question.CheckLimits(a.Fields(), question.Default, question.Addressee{}) {
		t.Errorf("%s: %s", q.ID, f)
	}
}

// listedItems is every item of q's material lists, in order.
func listedItems(q question.Question) []string {
	var out []string
	for _, b := range q.Material {
		if b.Kind == question.KindList {
			out = append(out, b.Items...)
		}
	}
	return out
}

// TestACategoryApprovalListsWhatItWrites is iss-2610071528375981: the
// approval of a kind of change shows one line per change it would make, asks
// in plain words rather than by the category's internal name, and its yes
// says how many changes it writes. Asked through resolveApproval, as every
// door receives it, and built as setup builds it.
func TestACategoryApprovalListsWhatItWrites(t *testing.T) {
	gaps := []Gap{
		{ID: "skeleton.config_missing", Category: SafeAutocreate, Title: ".abcd/config.json missing", Resolvable: true},
		{ID: "rules.missing", Category: SafeAutocreate, Title: ".abcd/rules.json missing", Resolvable: true},
		{ID: "plugin.root_missing", Category: PluginOwned, Title: "abcd's plugin files were not found", Resolvable: false},
	}
	p := &recordingPrompter{confirm: true}
	resolveApproval(gaps, InstallOptions{}, nil, p)
	if len(p.asked) != 1 {
		t.Fatalf("asked %d approvals, want the one for the skeleton: %q", len(p.asked), p.asked)
	}
	q := SetupConfirmQuestion(3, p.asked[0])
	if q.ID != approvePrefix+string(SafeAutocreate) {
		t.Fatalf("the approval's id is %q, want %s", q.ID, approvePrefix+string(SafeAutocreate))
	}
	items := listedItems(q)
	if strings.Join(items, "|") != ".abcd/config.json missing|.abcd/rules.json missing" {
		t.Fatalf("the approval lists %q, want one line per change", items)
	}
	if strings.Contains(q.Ask, string(SafeAutocreate)) {
		t.Errorf("the approval asks by the internal name: %q", q.Ask)
	}
	if !strings.Contains(q.Options[0].Meaning, "2 changes") {
		t.Errorf("yes means %q; it names no count of the changes it writes", q.Options[0].Meaning)
	}
	heldToEveryCheck(t, q)
}

// TestEveryCategoryApprovalPassesTheChecks builds the approval of each kind
// of change over one change and over more changes than one question's rows
// hold, and holds each to the structural check and the limits. Every change
// is listed and the yes counts them; the longer list breaks the rows limit
// alone, which the guard shows with a note rather than refusing.
func TestEveryCategoryApprovalPassesTheChecks(t *testing.T) {
	cats := append(append([]GapCategory(nil), categoryPromptOrder...), GapCategory("alpha"))
	for _, c := range cats {
		for _, n := range []int{1, 12} {
			lines := make([]string, n)
			for i := range lines {
				lines[i] = fmt.Sprintf(".abcd/work/file-%02d.md missing", i+1)
			}
			q := SetupConfirmQuestion(1, categoryApprovalText(c, lines))
			if q.ID != approvePrefix+string(c) {
				t.Errorf("%s over %d: id %q", c, n, q.ID)
			}
			if items := listedItems(q); strings.Join(items, "|") != strings.Join(lines, "|") {
				t.Errorf("%s over %d: lists %q", c, n, items)
			}
			if writes := categoryApprovalWords[c].writes; (writes || c == "alpha") && n > 1 &&
				!strings.Contains(q.Options[0].Meaning, fmt.Sprintf("%d changes", n)) {
				t.Errorf("%s over %d: yes means %q", c, n, q.Options[0].Meaning)
			}
			if n == 1 {
				heldToEveryCheck(t, q)
				continue
			}
			a := question.Ask{Questions: []question.Question{q}}
			for _, f := range question.Check(a) {
				t.Errorf("%s over %d: structural: %s", c, n, f)
			}
			for _, f := range question.CheckLimits(a.Fields(), question.Default, question.Addressee{}) {
				if f.Rule != question.RuleRows {
					t.Errorf("%s over %d: %s", c, n, f)
				}
			}
		}
	}
}

// TestAValueFlagIsListedInTheSettingsApproval: a flag that changes a saved
// setting puts the settings approval, and the approval names the change the
// flag makes, so its list is never empty.
func TestAValueFlagIsListedInTheSettingsApproval(t *testing.T) {
	setupHermetic(t)
	repo := flagOnlyRepo(t)
	lines := overrideChanges(repo, map[string]string{"docs_target": "agents_md"})
	if len(lines) != 1 || !strings.Contains(lines[0], "docs.target") || !strings.Contains(lines[0], "agents_md") {
		t.Fatalf("the flag's change reads %q", lines)
	}
	p := &recordingPrompter{}
	resolveApproval(nil, InstallOptions{}, lines, p)
	if len(p.asked) != 1 {
		t.Fatalf("asked %q", p.asked)
	}
	if items := listedItems(SetupConfirmQuestion(1, p.asked[0])); len(items) != 1 || items[0] != lines[0] {
		t.Fatalf("the settings approval lists %q, want the flag's change", items)
	}
}

// TestAnApprovalWithNoMaterialPointsAtNothingAbove: an approval asked whole,
// with no material (the adoption), passes the structural check, so its yes
// never points at text above.
func TestAnApprovalWithNoMaterialPointsAtNothingAbove(t *testing.T) {
	q := SetupConfirmQuestion(1, "Adopt this unmanaged repo into abcd?")
	if fs := question.Check(question.Ask{Questions: []question.Question{q}}); len(fs) > 0 {
		t.Fatalf("structural findings: %v", fs)
	}
}

// sevenGaps is seven resolvable changes of one kind, more than the host's 24
// rows hold in one question.
func sevenGaps() []Gap {
	var gaps []Gap
	for i := 1; i <= 7; i++ {
		gaps = append(gaps, Gap{ID: fmt.Sprintf("skeleton.%d", i), Category: SafeAutocreate,
			Title: fmt.Sprintf(".abcd/work/file-%d.md missing", i), Resolvable: true})
	}
	return gaps
}

// TestEveryRouteListsEveryChange: the approval lists every change it would
// make on every route, at a terminal (the drawn question, the [y/N] line) and
// off one (the answers file the host relays, the piped stream), never cut to
// fit: the person sees every file and setting before approving, and every
// door records the same question (spc-2610030911534855 B3, B5). A question
// taller than the host's rows is shown with the guard's note, not refused.
func TestEveryRouteListsEveryChange(t *testing.T) {
	for _, terminal := range []bool{true, false} {
		p := &recordingPrompter{terminal: terminal}
		resolveApproval(sevenGaps(), InstallOptions{}, nil, p)
		if len(p.asked) != 1 {
			t.Fatalf("terminal=%v: asked %q", terminal, p.asked)
		}
		items := listedItems(SetupConfirmQuestion(1, p.asked[0]))
		if len(items) != 7 || items[6] != ".abcd/work/file-7.md missing" {
			t.Fatalf("terminal=%v: the approval lists %q, want all seven", terminal, items)
		}
	}
}

// TestTheSettingsApprovalSaysValuesAreAskedFirst: approving the settings
// writes nothing by itself; each value not yet chosen is asked next, and one
// left unanswered saves none of them (stepConfigValues), so its yes goes on
// rather than claiming to write.
func TestTheSettingsApprovalSaysValuesAreAskedFirst(t *testing.T) {
	q := SetupConfirmQuestion(1, categoryApprovalText(ConfigChange, []string{"repo.visibility not set", "docs.target not set"}))
	yes := q.Options[0]
	if strings.Contains(yes.Meaning, "Writes") || yes.Label != nextLabel {
		t.Fatalf("the settings yes is %q: %q; it claims to write what is still to be asked", yes.Label, yes.Meaning)
	}
	heldToEveryCheck(t, q)
}
