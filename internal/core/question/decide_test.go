package question

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// The shared decision (spc-2610301917245442, step 1): Decide judges a call in
// the hook's order, and Rows is the split the rows rule computes.

// rowsFloor is Default with the rows limit at its floor, 24.
var rowsFloor = Bounded(Default, Bounds[0].Floor, Default.LabelWords, Default.MeaningSentences)

// rowsTab is the 25-row case of itd-2610230756259215: eight rows of header and
// frame, seven of question text and ten of options. It runs past the rows
// limit only at the floor, 24, since itd-2610301914350541 raised the default
// to 32, so the tests that need it over the limit judge it at rowsFloor.
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
// rows finding at the floor the check makes carries exactly those figures, because both
// read Rows.
func TestRowsSplitIsTheChecks(t *testing.T) {
	got := tabRows(rowsTab(), rowsFloor)
	want := RowSplit{HeaderAndFrame: 8, QuestionText: 7, Options: 10, Total: 25}
	if got != want {
		t.Fatalf("Rows = %+v, want %+v", got, want)
	}
	fs := CheckLimits(one(rowsTab()), rowsFloor, Addressee{})
	if len(fs) != 1 || fs[0].Rule != RuleRows {
		t.Fatalf("the 25-row case must draw the rows finding alone; got %v", rulesOf(fs))
	}
	if fs[0].Value != "25 rows: header and frame 8, question text 7, options 10" {
		t.Errorf("rows finding value = %q, not the split Rows computes", fs[0].Value)
	}
	if estimateRows(rowsTab(), rowsFloor) != got.Total {
		t.Errorf("estimateRows and Rows disagree")
	}
}

// TestDecideFollowsTheHooksOrder: away refuses before any rule; an unread mode
// store fails open for a question without a chip and leaves a chipped one
// checked; a question without a chip while the mode names nobody is not
// abcd's; otherwise the limits decide, the rows limit alone admitting with a
// note.
func TestDecideFollowsTheHooksOrder(t *testing.T) {
	broken := wellBuiltTab()
	broken.Options[0].Label = "Role and number (Recommended)"
	unchipped := wellBuiltTab()
	unchipped.Header = "Labels"
	unchippedBroken := broken
	unchippedBroken.Header = "Labels"
	errStore := errors.New("unreadable")
	refusedRows := &LimitsRefusal{Key: RowsKey, Value: "100", Bounds: "24 to 60", File: ".abcd/config.json", Reason: "refused"}
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
		{"mode unread, chipped, clean", wellBuiltTab(), State{Repo: true, Badge: true, ModeErr: errStore}, OutcomeUnchecked, true, false},
		{"mode unread, chipped, broken", broken, State{Repo: true, Badge: true, ModeErr: errStore}, OutcomeRefuse, true, true},
		{"mode unread, chipped, rows alone", rowsTab(), State{Repo: true, Badge: true, ModeErr: errStore}, OutcomeAdmitWithRowsNote, true, true},
		{"mode unread, limits unread, chipped, broken", broken, State{Repo: true, Badge: true, ModeErr: errStore, LimitsErr: errStore}, OutcomeRefuse, true, true},
		{"mode unread, unchipped", unchipped, State{Repo: true, Badge: true, ModeErr: errStore}, OutcomeUnchecked, true, false},
		{"mode unread, unchipped, broken", unchippedBroken, State{Repo: true, Badge: true, ModeErr: errStore}, OutcomeUnchecked, true, false},
		{"root unread, unchipped", unchipped, State{RootErr: errStore}, OutcomeUnchecked, true, false},
		{"root unread, chipped", wellBuiltTab(), State{RootErr: errStore}, OutcomeAdmit, false, false},
		{"away unread, clean", wellBuiltTab(), State{Repo: true, AwayErr: errStore}, OutcomeUnchecked, true, false},
		{"away unread, broken", broken, State{Repo: true, AwayErr: errStore}, OutcomeRefuse, true, true},
		{"away unread, rows alone", rowsTab(), State{Repo: true, AwayErr: errStore}, OutcomeAdmitWithRowsNote, true, true},
		{"limit refused, clean chipped", wellBuiltTab(), State{Repo: true, LimitsRefused: refusedRows}, OutcomeRefuse, false, false},
		{"limit refused, unchipped", unchipped, State{Repo: true, Badge: true, LimitsRefused: refusedRows}, OutcomeAdmitUnchecked, false, false},
		{"limit refused, mode unread, chipped", wellBuiltTab(), State{Repo: true, Badge: true, ModeErr: errStore, LimitsRefused: refusedRows}, OutcomeRefuse, true, false},
		{"limits unread, clean", wellBuiltTab(), State{Repo: true, LimitsErr: errStore}, OutcomeUnchecked, true, false},
		{"limits unread, broken", broken, State{Repo: true, LimitsErr: errStore}, OutcomeRefuse, true, true},
		{"limits unread, rows alone", rowsTab(), State{Repo: true, LimitsErr: errStore}, OutcomeAdmitWithRowsNote, true, true},
		{"limits unread, unchipped", unchipped, State{Repo: true, Badge: true, LimitsErr: errStore}, OutcomeAdmitUnchecked, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := Decide(one(tc.tab), tc.s, rowsFloor)
			if v.Outcome != tc.want {
				t.Errorf("outcome = %q, want %q", v.Outcome, tc.want)
			}
			if v.Partial != tc.partial || v.Partial != (len(v.Unread) > 0) {
				t.Errorf("partial = %v with %d unread, want %v", v.Partial, len(v.Unread), tc.partial)
			}
			if (len(v.Findings) > 0) != tc.findings {
				t.Errorf("findings = %v, want any: %v", rulesOf(v.Findings), tc.findings)
			}
			if len(v.Rows) != 1 || v.Rows[0] != tabRows(tc.tab, rowsFloor) {
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

// TestDecideChecksAChippedQuestionWithTheModeUnread: a chipped question
// needs no mode for its rules, since its chip names whom it is for, so with
// the mode store unread it is checked at the limits passed: a refusing
// finding refuses it, a tab over the rows limit alone is noted, and a clean
// one is admitted on the loud status. None of them sets a mode, since the
// mode could not be read.
func TestDecideChecksAChippedQuestionWithTheModeUnread(t *testing.T) {
	s := State{Repo: true, Badge: true, ModeErr: errors.New("unreadable")}
	broken := wellBuiltTab()
	broken.Options[0].Label = "Role and number (Recommended)"
	v := Decide(one(broken), s, Default)
	if v.Outcome != OutcomeRefuse || !v.Abcds || !slices.Contains(rulesOf(v.Findings), RuleNeverRecommended) {
		t.Errorf("broken: outcome %q, abcds %v, findings %v; want refuse on never-recommended", v.Outcome, v.Abcds, rulesOf(v.Findings))
	}
	v = Decide(one(rowsTab()), s, rowsFloor)
	if v.Outcome != OutcomeAdmitWithRowsNote || len(v.Findings) != 1 || v.Findings[0].Rule != RuleRows {
		t.Errorf("rows alone at the floor: outcome %q, findings %v; want the rows note", v.Outcome, rulesOf(v.Findings))
	}
	if v = Decide(one(rowsTab()), s, Default); v.Outcome != OutcomeUnchecked || len(v.Findings) != 0 {
		t.Errorf("the 25-row tab at the default rows: outcome %q, findings %v; want unchecked with none", v.Outcome, rulesOf(v.Findings))
	}
	for _, tab := range []Tab{broken, rowsTab(), wellBuiltTab()} {
		v := Decide(one(tab), s, rowsFloor)
		if v.ModeChanges || v.ModeAfter != Unnamed || !v.Partial {
			t.Errorf("%q: mode after %v (changes %v), partial %v; want no mode set and a partial verdict", tab.Header, v.ModeAfter, v.ModeChanges, v.Partial)
		}
	}
}

// TestDecideCarriesTheLimitsOutcome (spc-2610301950037469, step 2): a refused
// limit refuses abcd's call with no finding and the refusal on the verdict,
// sets no mode, and leaves another tool's call alone; an unread layer of the
// limits names its file and the three rules the keys decide, and the call is
// still checked at the limits passed.
func TestDecideCarriesTheLimitsOutcome(t *testing.T) {
	r := &LimitsRefusal{Key: RowsKey, Value: "100", Bounds: "24 to 60", File: ".abcd/config.json", Reason: "refused"}
	v := Decide(one(wellBuiltTab()), State{Repo: true, Badge: true, Mode: Facilitator, LimitsRefused: r}, Default)
	if v.Outcome != OutcomeRefuse || v.Refused != r || len(v.Findings) != 0 || v.ModeChanges || v.ModeAfter != Facilitator {
		t.Errorf("refused: outcome %q, refused %+v, findings %v, mode after %v (changes %v)", v.Outcome, v.Refused, rulesOf(v.Findings), v.ModeAfter, v.ModeChanges)
	}
	broken := wellBuiltTab()
	broken.Options[0].Label = "One two three four five six"
	v = Decide(one(broken), State{Repo: true, LimitsErr: errors.New("unparseable"), LimitsFile: ".abcd/config.json"}, Default)
	if len(v.Unread) != 1 || v.Unread[0].Store != StoreLimits || v.Unread[0].File != ".abcd/config.json" || !slices.Equal(v.Unread[0].Rules, limitsRules()) {
		t.Errorf("unread = %+v, want the limits store naming the file and the three rules", v.Unread)
	}
	if !slices.Contains(rulesOf(v.Findings), RuleLabelWords) || v.Outcome != OutcomeRefuse {
		t.Errorf("an unread layer must leave the call checked at the limits passed: outcome %q, findings %v", v.Outcome, rulesOf(v.Findings))
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
