package question

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// The shared decision (spc-2610301917245442, step 1): Decide judges a call in
// the hook's order, and Rows is the split the rows rule computes.

// rowsTab is the 25-row case of itd-2610230756259215: eight rows of header and
// frame, seven of question text and ten of options.
func rowsTab() Tab {
	return Tab{
		Header: "Product Q1",
		Text: strings.Join([]string{
			"The status line names whom the session waits on.",
			"It shows above every question.",
			"",
			"Now: the label names the role.",
			"Change later: edit the interview page.",
			"",
			"Should the label name the role as well as the number?",
		}, "\n"),
		Options: []Choice{
			{Label: "Role and number", Description: "It says whom the question is for and where you are in the interview. It costs a few characters."},
			{Label: "Number only", Description: "The label is shorter and the status line says whom it is for, so nothing is lost."},
			{Label: "Role only", Description: "The label says whom. You lose your place."},
			{Label: "Decide later", Description: "Nothing changes now."},
		},
	}
}

// TestRowsSplitIsTheChecks: the 25-row case reads 8, 7, 10 and 25, and the
// rows finding the check makes carries exactly those figures, because both
// read Rows.
func TestRowsSplitIsTheChecks(t *testing.T) {
	got := tabRows(rowsTab(), Default)
	want := RowSplit{HeaderAndFrame: 8, QuestionText: 7, Options: 10, Total: 25}
	if got != want {
		t.Fatalf("Rows = %+v, want %+v", got, want)
	}
	fs := CheckLimits(one(rowsTab()), Default, Addressee{})
	if len(fs) != 1 || fs[0].Rule != RuleRows {
		t.Fatalf("the 25-row case must draw the rows finding alone; got %v", rulesOf(fs))
	}
	if fs[0].Value != "25 rows: header and frame 8, question text 7, options 10" {
		t.Errorf("rows finding value = %q, not the split Rows computes", fs[0].Value)
	}
	if estimateRows(rowsTab(), Default) != got.Total {
		t.Errorf("estimateRows and Rows disagree")
	}
}

// TestDecideFollowsTheHooksOrder: away refuses before any rule; an unread mode
// store fails open; a question without a chip while the mode names nobody is
// not abcd's; otherwise the limits decide, the rows limit alone admitting
// with a note.
func TestDecideFollowsTheHooksOrder(t *testing.T) {
	broken := wellBuiltTab()
	broken.Options[0].Label = "Role and number (Recommended)"
	unchipped := wellBuiltTab()
	unchipped.Header = "Labels"
	errStore := errors.New("unreadable")
	for _, tc := range []struct {
		name     string
		tab      Tab
		s        State
		want     Outcome
		partial  bool
		findings bool
	}{
		{"away refuses a broken question first", broken, State{Repo: true, Away: true}, OutcomeRefuseAway, false, false},
		{"away refuses another tool's question", unchipped, State{Repo: true, Away: true}, OutcomeRefuseAway, false, false},
		{"clean chipped", wellBuiltTab(), State{Repo: true}, OutcomeAdmit, false, false},
		{"broken chipped", broken, State{}, OutcomeRefuse, false, true},
		{"rows alone", rowsTab(), State{}, OutcomeAdmitWithRowsNote, false, true},
		{"unchipped, mode names nobody", unchipped, State{Repo: true, Badge: true}, OutcomeAdmitUnchecked, false, false},
		{"unchipped, mode names the product thinker", unchipped, State{Repo: true, Badge: true, Mode: ProductThinker}, OutcomeRefuse, false, true},
		{"mode unread, chipped", wellBuiltTab(), State{Repo: true, Badge: true, ModeErr: errStore}, OutcomeUnchecked, true, false},
		{"mode unread, unchipped", unchipped, State{Repo: true, Badge: true, ModeErr: errStore}, OutcomeUnchecked, true, false},
		{"root unread, unchipped", unchipped, State{RootErr: errStore}, OutcomeUnchecked, true, false},
		{"root unread, chipped", wellBuiltTab(), State{RootErr: errStore}, OutcomeAdmit, false, false},
		{"away unread, clean", wellBuiltTab(), State{Repo: true, AwayErr: errStore}, OutcomeUnchecked, true, false},
		{"away unread, broken", broken, State{Repo: true, AwayErr: errStore}, OutcomeRefuse, true, true},
		{"away unread, rows alone", rowsTab(), State{Repo: true, AwayErr: errStore}, OutcomeAdmitWithRowsNote, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := Decide(one(tc.tab), tc.s, Default)
			if v.Outcome != tc.want {
				t.Errorf("outcome = %q, want %q", v.Outcome, tc.want)
			}
			if v.Partial != tc.partial || v.Partial != (len(v.Unread) > 0) {
				t.Errorf("partial = %v with %d unread, want %v", v.Partial, len(v.Unread), tc.partial)
			}
			if (len(v.Findings) > 0) != tc.findings {
				t.Errorf("findings = %v, want any: %v", rulesOf(v.Findings), tc.findings)
			}
			if len(v.Rows) != 1 || v.Rows[0] != tabRows(tc.tab, Default) {
				t.Errorf("rows = %+v, want the tab's split", v.Rows)
			}
			if v.DependsOnState != tc.s.Repo {
				t.Errorf("depends_on_state = %v, want %v", v.DependsOnState, tc.s.Repo)
			}
		})
	}
}

// TestDecideNamesTheUnreadStoreAndItsRules: an unread away record names every
// rule, since it decides whether any is read; an unread mode store names every
// rule for a question it cannot tell is abcd's, and none for a chipped one,
// whose rules need no mode.
func TestDecideNamesTheUnreadStoreAndItsRules(t *testing.T) {
	errStore := errors.New("unreadable")
	unchipped := wellBuiltTab()
	unchipped.Header = "Labels"
	v := Decide(one(wellBuiltTab()), State{Repo: true, AwayErr: errStore}, Default)
	if len(v.Unread) != 1 || v.Unread[0].Store != StoreAway || !slices.Equal(v.Unread[0].Rules, allRules()) {
		t.Errorf("away unread = %+v, want the away store with every rule", v.Unread)
	}
	v = Decide(one(unchipped), State{Repo: true, Badge: true, ModeErr: errStore}, Default)
	if len(v.Unread) != 1 || v.Unread[0].Store != StoreMode || !slices.Equal(v.Unread[0].Rules, allRules()) {
		t.Errorf("mode unread, unchipped = %+v, want the mode store with every rule", v.Unread)
	}
	v = Decide(one(wellBuiltTab()), State{Repo: true, Badge: true, ModeErr: errStore}, Default)
	if len(v.Unread) != 1 || v.Unread[0].Store != StoreMode || len(v.Unread[0].Rules) != 0 {
		t.Errorf("mode unread, chipped = %+v, want the mode store with no rule", v.Unread)
	}
}

// TestDecideAddresseesAndTheModeAfter: each tab's addressee is its chip's
// person, or the mode's without a chip, and an admitted chipped call that
// names someone else than the mode changes it (decision 15).
func TestDecideAddresseesAndTheModeAfter(t *testing.T) {
	tech := wellBuiltTab()
	tech.Header = "Tech Q1"
	v := Decide(Fields{Tabs: []Tab{tech, wellBuiltTab()}}, State{Repo: true, Badge: true, Mode: Facilitator}, Default)
	if !slices.Equal(v.Addressees, []Person{Facilitator, ProductThinker}) {
		t.Errorf("addressees = %v, want facilitator then product thinker", v.Addressees)
	}
	if v.ModeAfter != ProductThinker || !v.ModeChanges {
		t.Errorf("mode after = %v (changes %v), want the product thinker, changed", v.ModeAfter, v.ModeChanges)
	}
	v = Decide(one(tech), State{Repo: true, Badge: true, Mode: Facilitator}, Default)
	if v.ModeAfter != Facilitator || v.ModeChanges {
		t.Errorf("mode after = %v (changes %v), want the facilitator, unchanged", v.ModeAfter, v.ModeChanges)
	}
	v = Decide(one(tech), State{}, Default)
	if v.ModeChanges {
		t.Error("a mode change was reported where no badge shows")
	}
}
