package rules

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/core/question"
)

// widgetSet is a rule set with one custom domain of three rules and one of a
// single rule, both recalled by "widget".
func widgetSet() RuleSet {
	return RuleSet{SchemaVersion: 1, Domains: map[string]Domain{
		"WIDGETS":  {Recall: []string{"widget"}, Rules: []string{"Count widgets twice.", "Never paint a widget blue.", "Ship widgets in pairs."}},
		"SPROCKET": {Recall: []string{"widget"}, Rules: []string{"Sprockets turn with widgets."}},
	}}
}

// TestBracketForTheFill: the table reads the fill as the share of the
// window still free — 70% or more free is fresh, 40% to 70% moderate, below
// 40% depleted — and no fill, or a share that is not one, is unknown.
func TestBracketForTheFill(t *testing.T) {
	for _, tc := range []struct {
		used float64
		want Bracket
	}{
		{0, BracketFresh}, {30, BracketFresh}, {30.01, BracketModerate}, {45, BracketModerate}, {60, BracketModerate},
		{60.01, BracketDepleted}, {99, BracketDepleted}, {100, BracketDepleted}, {-1, BracketUnknown}, {101, BracketUnknown},
	} {
		if got := BracketForUsed(tc.used); got != tc.want {
			t.Errorf("BracketForUsed(%v) = %v, want %v", tc.used, got, tc.want)
		}
	}
	if got := BracketFor(Fill{Source: FillSourceUnknown, Reason: "no source"}); got != BracketUnknown {
		t.Errorf("an unknown fill is in %v, want unknown", got)
	}
	used := 50.0
	if got := BracketFor(Fill{Source: FillSourceStatusLine, UsedPercentage: &used}); got != BracketModerate {
		t.Errorf("a 50%% fill is in %v, want moderate", got)
	}
	var zero Bracket
	if zero != BracketUnknown {
		t.Errorf("the zero bracket is %v, want unknown", zero)
	}
	for _, r := range BracketTable() {
		if r.Bracket != BracketFresh && r.Bracket != BracketUnknown && !r.ShrinkRules {
			t.Errorf("%v keeps rules in full; a known fill below fresh shrinks them", r.Bracket)
		}
		if r.MemoryFullBytes == 0 && r.MemoryLineBytes == 0 {
			t.Errorf("%v sends notes and pages in no form: no bracket sends nothing", r.Bracket)
		}
	}
}

// TestRulesShrinkToHeadingAndFirstRule: under a known moderate or depleted
// fill each matched domain is sent as its heading and its first rule, never
// reordered; fresh and unknown send every rule.
func TestRulesShrinkToHeadingAndFirstRule(t *testing.T) {
	rs := widgetSet()
	for _, b := range []Bracket{BracketModerate, BracketDepleted} {
		res := Inject(rs, "count the widget", SessionState{}, 0, b)
		if !strings.Contains(res.Text, "## WIDGETS\n- Count widgets twice.\n") {
			t.Fatalf("%v: WIDGETS is not its heading then its first rule:\n%s", b, res.Text)
		}
		for _, later := range []string{"Never paint a widget blue.", "Ship widgets in pairs."} {
			if strings.Contains(res.Text, later) {
				t.Errorf("%v: a shrunk domain carries a later rule %q:\n%s", b, later, res.Text)
			}
		}
		if !slices.Equal(res.Injected, []string{"SPROCKET", "WIDGETS"}) || !slices.Equal(res.Shrunk, []string{"WIDGETS"}) {
			t.Errorf("%v: injected %v, shrunk %v; want both injected, WIDGETS shrunk", b, res.Injected, res.Shrunk)
		}
		// A domain of one rule is already its heading and first rule.
		if !strings.Contains(res.Text, "## SPROCKET\n- Sprockets turn with widgets.\n") {
			t.Errorf("%v: SPROCKET did not come back whole:\n%s", b, res.Text)
		}
	}
	for _, b := range []Bracket{BracketFresh, BracketUnknown} {
		res := Inject(rs, "count the widget", SessionState{}, 0, b)
		if want := Render(rs.Match("count the widget")); res.Text != want {
			t.Errorf("%v changed the rules' form:\n got %q\nwant %q", b, res.Text, want)
		}
		if len(res.Shrunk) != 0 {
			t.Errorf("%v shrank %v", b, res.Shrunk)
		}
	}
}

// TestShrunkDomainNamesTheRulesCommand: a shrunk domain is never a heading
// alone or a rule alone — a line under its first rule says how many rules it
// holds and that `abcd rules <DOMAIN>` renders the rest.
func TestShrunkDomainNamesTheRulesCommand(t *testing.T) {
	res := Inject(widgetSet(), "count the widget", SessionState{}, 0, BracketDepleted)
	block := res.Text[strings.Index(res.Text, "## WIDGETS"):]
	lines := strings.Split(strings.TrimRight(block, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("a shrunk domain is %d lines, want heading, first rule and the pointer:\n%s", len(lines), block)
	}
	if !strings.Contains(lines[2], "`abcd rules WIDGETS`") || !strings.Contains(lines[2], "3 rules") {
		t.Fatalf("the pointer line does not name the count and the command: %q", lines[2])
	}
	if strings.HasPrefix(lines[2], "## ") || strings.HasPrefix(lines[2], "#") {
		t.Fatalf("the pointer line reads as a heading: %q", lines[2])
	}
}

// TestGeneratedDomainsShrinkToTheirFirstGeneratedRule: no author orders
// SHELL's and GRILL's rules, so a shrunk SHELL is the hazard registry's first
// entry in id order and a shrunk GRILL the first rule question.AskingRules
// returns.
func TestGeneratedDomainsShrinkToTheirFirstGeneratedRule(t *testing.T) {
	reg := guard.Defaults()
	ids := make([]string, 0, len(reg.Entries))
	for id := range reg.Entries {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	first := reg.Entries[ids[0]]
	first.ID = ids[0]
	shellFirst := sanitizeRuleBody(first.Lesson())
	grillRules := question.AskingRules(question.Default)
	grillFirst := sanitizeRuleBody(grillRules[0])

	res := Inject(Defaults(), "*SHELL *GRILL", SessionState{}, 0, BracketModerate)
	for _, tc := range []struct {
		name, first, second string
		count               int
	}{
		{ShellDomain, shellFirst, Defaults().Domains[ShellDomain].Rules[1], len(Defaults().Domains[ShellDomain].Rules)},
		{GrillDomain, grillFirst, grillRules[1], len(grillRules)},
	} {
		i := strings.Index(res.Text, "## "+tc.name+"\n")
		if i < 0 {
			t.Fatalf("%s did not come back:\n%s", tc.name, res.Text)
		}
		block := res.Text[i:]
		if j := strings.Index(block[3:], "\n## "); j >= 0 {
			block = block[:j+4]
		}
		if !strings.HasPrefix(block, "## "+tc.name+"\n- "+tc.first+"\n") {
			t.Errorf("%s is not shrunk to its first generated rule:\n%s", tc.name, block)
		}
		if strings.Contains(block, sanitizeRuleBody(tc.second)) {
			t.Errorf("%s carries its second rule:\n%s", tc.name, block)
		}
		if !strings.Contains(block, "`abcd rules "+tc.name+"`") || !strings.Contains(block, itoaRules(tc.count)) {
			t.Errorf("%s's pointer does not name its %d rules and the command:\n%s", tc.name, tc.count, block)
		}
	}
}

func itoaRules(n int) string { return strconv.Itoa(n) + " rules" }

// TestUnknownFillKeepsRulesFull: with no source for the fill, rules inject
// exactly as they did before the brackets: the unknown row, the zero
// bracket, shrinks nothing.
func TestUnknownFillKeepsRulesFull(t *testing.T) {
	rs := Defaults()
	prompt := "commit and push, then check the shell for pii"
	want := Render(rs.Match(prompt))
	if want == "" {
		t.Fatal("the prompt matched nothing")
	}
	// The zero bracket is the unknown row: a caller that knows no fill.
	var none Bracket
	if got := Inject(rs, prompt, SessionState{}, 0, none); got.Text != want || got.Bracket != BracketUnknown || len(got.Shrunk) != 0 {
		t.Errorf("an unknown fill changed the rules' form (bracket %v, shrunk %v):\n got %q\nwant %q", got.Bracket, got.Shrunk, got.Text, want)
	}
	// So does a domain whose rules a known fill would shrink.
	if got := Inject(widgetSet(), "count the widget", SessionState{}, 0, none); !strings.Contains(got.Text, "Ship widgets in pairs.") {
		t.Errorf("an unknown fill shrank a domain:\n%s", got.Text)
	}
	if !strings.Contains(BracketUnknown.Row().Describe(), "Rules in full") {
		t.Errorf("the unknown row's words do not say rules come in full: %q", BracketUnknown.Row().Describe())
	}
}

// TestUnknownFillIsSaidOncePerSession: the line saying the fill is unknown is
// said on the first prompt of a session it is unknown on, recorded in the
// session's state, carried through Inject, and said again only after a reset
// removes that state.
func TestUnknownFillIsSaidOncePerSession(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	unknown := Fill{Source: FillSourceUnknown, Reason: "no fill file for this session and no transcript named"}
	used := 20.0
	known := Fill{Source: FillSourceTranscript, UsedPercentage: &used}

	var st SessionState
	if line := UnknownFillNotice(&st, known); line != "" || st.FillUnknownSaid {
		t.Fatalf("a known fill was said to be unknown: %q", line)
	}
	line := UnknownFillNotice(&st, unknown)
	if !strings.Contains(line, "unknown") || !strings.Contains(line, unknown.Reason) || !st.FillUnknownSaid {
		t.Fatalf("the first unknown fill was not said with its reason: %q", line)
	}
	next := Inject(Defaults(), "commit it", st, 0, BracketUnknown).State
	if !next.FillUnknownSaid {
		t.Fatal("Inject forgot that the unknown fill was said")
	}
	// The backstop clears the ledgers, not the saying.
	if bs := Inject(Defaults(), "commit it", SessionState{Count: 14, FillUnknownSaid: true}, 15, BracketUnknown).State; !bs.FillUnknownSaid {
		t.Fatal("the backstop forgot that the unknown fill was said")
	}
	if err := SaveState("said", next); err != nil {
		t.Fatal(err)
	}
	loaded := LoadState("said")
	if again := UnknownFillNotice(&loaded, unknown); again != "" {
		t.Fatalf("the unknown fill was said twice in one session: %q", again)
	}
	if err := ResetState("said"); err != nil {
		t.Fatal(err)
	}
	fresh := LoadState("said")
	if again := UnknownFillNotice(&fresh, unknown); again == "" {
		t.Fatal("after a reset the unknown fill was not said again")
	}
}

// TestABracketChangeNeverResendsADomain: the rules ledger keys on a domain's
// content, not on its form, so a domain sent shrunk is not sent again in
// full when the bracket changes, nor the other way round.
func TestABracketChangeNeverResendsADomain(t *testing.T) {
	rs := widgetSet()
	shrunk := Inject(rs, "count the widget", SessionState{}, 0, BracketDepleted)
	if len(shrunk.Injected) != 2 {
		t.Fatalf("injected %v", shrunk.Injected)
	}
	if again := Inject(rs, "count the widget", shrunk.State, 0, BracketFresh); again.Text != "" || len(again.Injected) != 0 {
		t.Fatalf("a bracket change sent a domain again:\n%s", again.Text)
	}
	full := Inject(rs, "count the widget", SessionState{}, 0, BracketFresh)
	if again := Inject(rs, "count the widget", full.State, 0, BracketDepleted); again.Text != "" {
		t.Fatalf("a bracket change sent a domain again, shrunk:\n%s", again.Text)
	}
}

// TestBracketTableHasOneSource: the thresholds and the memory budgets are
// set in brackets.go alone. No other source file of the rules loader, the
// recall or the CLI's recall front door holds a threshold literal — a free
// or used share of 70, 40, 30 or 60, as a percentage or a fraction — or an
// 8 KiB or 2 KiB figure, so no payload can grow a bracket of its own.
func TestBracketTableHasOneSource(t *testing.T) {
	banned := map[string]bool{"70": true, "40": true, "30": true, "60": true, "0.7": true, "0.4": true, "0.3": true, "0.6": true,
		"8192": true, "2048": true, ".7": true, ".4": true, ".3": true, ".6": true}
	var files []string
	for _, dir := range []string{".", "../recall"} {
		matches, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, matches...)
	}
	files = append(files, "../../surface/cli/rules_prompt.go")
	checked := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || filepath.Base(f) == "brackets.go" {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, f, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		checked++
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.BasicLit:
				if (x.Kind == token.INT || x.Kind == token.FLOAT) && banned[x.Value] {
					t.Errorf("%s: a threshold-shaped literal %s outside the bracket table", fset.Position(x.Pos()), x.Value)
				}
			case *ast.BinaryExpr:
				if x.Op != token.MUL {
					return true
				}
				l, lok := x.X.(*ast.BasicLit)
				r, rok := x.Y.(*ast.BasicLit)
				if lok && rok && r.Value == "1024" && (l.Value == "8" || l.Value == "2") {
					t.Errorf("%s: a bracket budget %s * 1024 outside the bracket table", fset.Position(x.Pos()), l.Value)
				}
			}
			return true
		})
	}
	if checked < 5 {
		t.Fatalf("checked %d files; the globs found too few", checked)
	}
}
