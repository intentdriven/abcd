package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
)

// everyLesson is every lesson the registry renders, in the words the guard
// would use, the three with no entry of their own included.
func everyLesson(reg guard.Registry) []string {
	return append(reg.Lessons(), reg.ScriptLesson(), reg.ShellCLesson(), reg.DeleteLesson())
}

// TestShellDomainSendsOnlyWhatTheGuardCannotCorrectFirst: with the guard on,
// the agent gets one line about the guard and the commands it only warns
// about, since a warned command still runs; what it refuses is corrected at
// the refusal and stays readable through abcd rules SHELL.
func TestShellDomainSendsOnlyWhatTheGuardCannotCorrectFirst(t *testing.T) {
	reg := guard.Defaults()
	d, ok := shellDomain(reg, reg)
	if !ok {
		t.Fatal("no domain from the bundled registry")
	}
	if d.Rules[0] != guard.SteeringLine {
		t.Fatalf("first rule = %q, want the steering line", d.Rules[0])
	}
	for _, r := range d.Rules[1:] {
		if !strings.HasPrefix(r, "Warned by the guard") {
			t.Errorf("a steering rule is not a warned command: %q", r)
		}
	}
	if len(d.Checked) == 0 {
		t.Fatal("no rule is carried by the guard")
	}
}

// TestADeleteTheGuardCanRefuseIsNotSentAsWarned: a delete entry sits in the warn
// tier, but in a repository the guard refuses it unless it proves the target
// safe, so the refusal answers for it and only a command that is sure to run
// is sent ahead of time.
func TestADeleteTheGuardCanRefuseIsNotSentAsWarned(t *testing.T) {
	reg := guard.Defaults()
	d, ok := shellDomain(reg, reg)
	if !ok {
		t.Fatal("no domain from the bundled registry")
	}
	for _, id := range []string{"git-push-delete", "git-push-delete-refspec", "git-branch-force-delete", "git-branch-delete-force", "git-worktree-remove"} {
		mark := "(" + id + ")"
		if slices.ContainsFunc(d.Rules, func(r string) bool { return strings.Contains(r, mark) }) {
			t.Errorf("%s is sent ahead of time", id)
		}
		if !slices.ContainsFunc(d.Checked, func(r string) bool { return strings.Contains(r, mark) }) {
			t.Errorf("%s is not in the guard's list", id)
		}
	}
	for _, id := range []string{"rm-rf-working-directory", "git-reset-hard", "git-clean"} {
		mark := "(" + id + ")"
		if !slices.ContainsFunc(d.Rules, func(r string) bool { return strings.Contains(r, mark) }) {
			t.Errorf("%s, sure to run, is not sent ahead of time", id)
		}
	}
}

// TestEveryLessonIsInExactlyOneList: a hazard can never fall out of both
// lists, whichever way the guard is set.
func TestEveryLessonIsInExactlyOneList(t *testing.T) {
	on := guard.Defaults()
	off := guard.Defaults()
	off.Disabled = true
	for name, reg := range map[string]guard.Registry{"guard on": on, "guard off": off} {
		d, ok := shellDomain(reg, on)
		if !ok {
			t.Fatalf("%s: no domain", name)
		}
		for _, l := range everyLesson(reg) {
			n := 0
			for _, list := range [][]string{d.Rules, d.Checked} {
				for _, r := range list {
					if strings.Contains(r, l) || strings.Contains(l, r) {
						n++
					}
				}
			}
			if n != 1 {
				t.Errorf("%s: lesson in %d lists: %q", name, n, l)
			}
		}
	}
}

// TestGuardOffKeepsEveryShellRuleSteeringAndClaimsNoGuard: where the guard
// refuses nothing, nothing corrects the agent later, so every rule stays
// steering, under a first line saying the guard is off, and no line says a guard
// is running.
func TestGuardOffKeepsEveryShellRuleSteeringAndClaimsNoGuard(t *testing.T) {
	off := guard.Defaults()
	off.Disabled = true
	d, ok := shellDomain(off, guard.Defaults())
	if !ok {
		t.Fatal("no domain")
	}
	if len(d.Checked) != 0 {
		t.Fatalf("%d rules carried by a guard that is off", len(d.Checked))
	}
	if slices.Contains(d.Rules, guard.SteeringLine) {
		t.Fatal("a line claims a guard is running")
	}
	if d.Rules[0] != guard.OffLine {
		t.Fatalf("first rule = %q, want the line saying the guard is off", d.Rules[0])
	}
	if len(d.Rules) != len(everyLesson(off))+1 {
		t.Fatalf("%d rules, want %d and the line", len(d.Rules), len(everyLesson(off)))
	}
	for _, l := range []string{off.ScriptLesson(), off.ShellCLesson(), off.DeleteLesson()} {
		for _, claim := range []string{"refuse", "judge", "guard allow", "the guard ", "read before"} {
			if strings.Contains(strings.ToLower(l), claim) {
				t.Errorf("with the guard off a rule says %q: %q", claim, l)
			}
		}
	}
}

// TestAShortenedDomainCountsWhatTheGuardCarries: the line under a shortened SHELL
// must not say the command renders nine rules when it lists every hazard too.
func TestAShortenedDomainCountsWhatTheGuardCarries(t *testing.T) {
	reg := guard.Defaults()
	d, ok := shellDomain(reg, reg)
	if !ok {
		t.Fatal("no domain from the bundled registry")
	}
	block, shortened := renderShrunk(ResolvedDomain{Name: "SHELL", Source: "bundled", Domain: d})
	if !shortened {
		t.Fatalf("SHELL did not shorten:\n%s", block)
	}
	want := fmt.Sprintf("renders them and the %d the guard carries", len(d.Checked))
	if !strings.Contains(block, want) {
		t.Fatalf("the shortened block does not count the guard's %d rules:\n%s", len(d.Checked), block)
	}
}

// TestAShellPromptGetsNoTextForWhatIsRefused walks the whole path, the hook
// and the listing: the injected block has no refused text and the listing has
// it all.
func TestAShellPromptGetsNoTextForWhatIsRefused(t *testing.T) {
	rs := Defaults()
	res := Inject(rs, "*SHELL clean up", SessionState{}, 0, BracketUnknown)
	if !strings.Contains(res.Text, guard.SteeringLine) {
		t.Fatalf("no steering line:\n%s", res.Text)
	}
	if strings.Contains(res.Text, "Refused by the guard") {
		t.Fatalf("refused text was sent ahead of time:\n%s", res.Text)
	}
	d, ok := rs.Lookup(ShellDomain)
	if !ok {
		t.Fatal("no SHELL")
	}
	listed := RenderListing([]ResolvedDomain{d})
	for _, l := range everyLesson(guard.Defaults()) {
		if !strings.Contains(listed, l) {
			t.Errorf("the listing leaves out %q", l)
		}
	}
}

// TestChangingARefusedLessonDoesNotResendTheDomain: the signature hashes what
// is sent, so a word changed in a list that is not sent costs no re-injection.
func TestChangingARefusedLessonDoesNotResendTheDomain(t *testing.T) {
	d := ResolvedDomain{Name: "X", Domain: Domain{Rules: []string{"a"}, Checked: []string{"b"}}}
	e := ResolvedDomain{Name: "X", Domain: Domain{Rules: []string{"a"}, Checked: []string{"c"}}}
	if Signature(d) != Signature(e) {
		t.Fatal("the signature reads the list that is not sent")
	}
}

// TestNoRuleFileCanDeclareChecked: the second list is generated, so a file
// cannot move a rule out of what is sent by naming it.
func TestNoRuleFileCanDeclareChecked(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"schema_version":1,"domains":{"MINE":{"recall":["zebra"],"rules":["a"],"checked":["b"]}}}`
	if err := os.WriteFile(filepath.Join(root, ".abcd", "rules.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	rs, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if d := rs.Domains["MINE"]; len(d.Checked) != 0 {
		t.Fatalf("a file declared checked rules: %q", d.Checked)
	}
}

// TestListingJSONSeparatesTheListsAndGivesSizes: the review of the rule set
// can be run again from this alone.
func TestListingJSONSeparatesTheListsAndGivesSizes(t *testing.T) {
	d, ok := Defaults().Lookup(ShellDomain)
	if !ok {
		t.Fatal("no SHELL")
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Rules        []string `json:"rules"`
		Checked      []string `json:"checked"`
		RulesBytes   *int     `json:"rules_bytes"`
		CheckedBytes *int     `json:"checked_bytes"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Rules) == 0 || len(got.Checked) == 0 {
		t.Fatalf("lists: %d steering, %d checked", len(got.Rules), len(got.Checked))
	}
	if got.RulesBytes == nil || got.CheckedBytes == nil || *got.RulesBytes == 0 || *got.CheckedBytes == 0 {
		t.Fatalf("sizes: %v, %v", got.RulesBytes, got.CheckedBytes)
	}
}

// sizedSet is a rule set of the named domains, each holding n rules of size
// bytes, all recalled on "zebra".
func sizedSet(n, size int, names ...string) RuleSet {
	rs := RuleSet{SchemaVersion: 1, Domains: map[string]Domain{}}
	for _, name := range names {
		var list []string
		for i := range n {
			list = append(list, fmt.Sprintf("%s rule %d %s", name, i, strings.Repeat("w", size)))
		}
		rs.Domains[name] = Domain{Recall: []string{"zebra"}, Rules: list}
	}
	return rs
}

// TestInjectWithinGivesEveryDomainItsFirstRuleAndACommand: overflow never
// drops a domain nor cuts one mid-rule, each is sent as its first rule and the
// command, and the session records exactly that.
func TestInjectWithinGivesEveryDomainItsFirstRuleAndACommand(t *testing.T) {
	names := []string{"AAA", "BBB", "CCC", "DDD"}
	rs := sizedSet(10, 300, names...)
	const budget = 2500
	res := InjectWithin(rs, "zebra", SessionState{}, 0, BracketUnknown, budget)
	if len(res.Text) > budget {
		t.Fatalf("%d bytes over the budget of %d", len(res.Text), budget)
	}
	for _, n := range names {
		if !strings.Contains(res.Text, "## "+n) || !strings.Contains(res.Text, n+" rule 0 ") {
			t.Errorf("%s has no first rule:\n%s", n, res.Text)
		}
		if !strings.Contains(res.Text, "abcd rules "+n) {
			t.Errorf("%s names no command:\n%s", n, res.Text)
		}
		if res.State.Ledger[n] == "" {
			t.Errorf("%s not recorded", n)
		}
	}
	if !slices.Equal(res.Shrunk, names) {
		t.Fatalf("shrunk = %v", res.Shrunk)
	}
	again := InjectWithin(rs, "zebra", res.State, 0, BracketUnknown, budget)
	if again.Text != "" {
		t.Fatalf("a shortened domain was sent again:\n%s", again.Text)
	}
}

// TestInjectWithinKeepsRulesFullWhenTheyFit: the bracket ruling stands, the
// overflow answer changes nothing in a set that fits.
func TestInjectWithinKeepsRulesFullWhenTheyFit(t *testing.T) {
	rs := sizedSet(3, 50, "AAA", "BBB")
	res := InjectWithin(rs, "zebra", SessionState{}, 0, BracketUnknown, 4000)
	if len(res.Shrunk) != 0 {
		t.Fatalf("shrunk %v with room to spare", res.Shrunk)
	}
	if !strings.Contains(res.Text, "AAA rule 2") || !strings.Contains(res.Text, "BBB rule 2") {
		t.Fatalf("not in full:\n%s", res.Text)
	}
}

// TestInjectWithinWidensTheSmallestFirst: what room is left goes to as many
// whole domains as it can hold.
func TestInjectWithinWidensTheSmallestFirst(t *testing.T) {
	rs := sizedSet(10, 300, "BIG")
	small := sizedSet(3, 40, "SMALL")
	rs.Domains["SMALL"] = small.Domains["SMALL"]
	res := InjectWithin(rs, "zebra", SessionState{}, 0, BracketUnknown, 1200)
	if slices.Contains(res.Shrunk, "SMALL") || !slices.Contains(res.Shrunk, "BIG") {
		t.Fatalf("shrunk = %v, want BIG alone", res.Shrunk)
	}
}

// TestAHugeFirstRuleDoesNotHoldBackTheOthers: a domain whose first rule cannot
// fit goes as its heading and the command, and the rest still come back.
func TestAHugeFirstRuleDoesNotHoldBackTheOthers(t *testing.T) {
	rs := sizedSet(1, 5000, "AHUGE")
	rs.Domains["BSMALL"] = sizedSet(2, 40, "BSMALL").Domains["BSMALL"]
	res := InjectWithin(rs, "zebra", SessionState{}, 0, BracketUnknown, 1500)
	if len(res.Text) > 1500 {
		t.Fatalf("%d bytes", len(res.Text))
	}
	if !strings.Contains(res.Text, "BSMALL rule 1") {
		t.Fatalf("the small domain was held back:\n%s", res.Text)
	}
	if !strings.Contains(res.Text, "## AHUGE") || !strings.Contains(res.Text, "abcd rules AHUGE") {
		t.Fatalf("the huge domain has no pointer:\n%s", res.Text)
	}
	if !slices.Contains(res.Pointed, "AHUGE") {
		t.Fatalf("pointed = %v", res.Pointed)
	}
}

// TestAPointerIsNotRecordedAsSent: a domain that went as its pointer, which
// says there was no room this turn, is sent with its rules on the next turn
// that has room, rather than counted as sent until the backstop.
func TestAPointerIsNotRecordedAsSent(t *testing.T) {
	rs := sizedSet(1, 5000, "AHUGE")
	first := InjectWithin(rs, "zebra", SessionState{}, 0, BracketUnknown, 1500)
	if !slices.Contains(first.Pointed, "AHUGE") {
		t.Fatalf("pointed = %v, want AHUGE", first.Pointed)
	}
	next := InjectWithin(rs, "zebra", first.State, 0, BracketUnknown, RulesBudget(0))
	if !strings.Contains(next.Text, "AHUGE rule 0") {
		t.Fatalf("the pointed domain was not sent once there was room:\n%s", next.Text)
	}
	again := InjectWithin(rs, "zebra", next.State, 0, BracketUnknown, RulesBudget(0))
	if again.Text != "" {
		t.Fatalf("a domain sent with its rules was sent again:\n%s", again.Text)
	}
}

// TestALongFirstRuleInFrontDoesNotCrowdOutTheRest: a first rule that fits but
// would leave no room behind it waits for what is left, so every domain behind it
// still gets its command.
func TestALongFirstRuleInFrontDoesNotCrowdOutTheRest(t *testing.T) {
	rs := sizedSet(2, 1200, "AAA")
	for _, n := range []string{"BBB", "CCC", "DDD", "EEE", "FFF"} {
		rs.Domains[n] = sizedSet(2, 40, n).Domains[n]
	}
	res := InjectWithin(rs, "zebra", SessionState{}, 0, BracketUnknown, 1300)
	for _, n := range []string{"AAA", "BBB", "CCC", "DDD", "EEE", "FFF"} {
		if !strings.Contains(res.Text, "## "+n) {
			t.Fatalf("%s was crowded out:\n%s", n, res.Text)
		}
	}
}

// TestADomainWithNoRoomIsTriedAgain: nothing is recorded as sent unless it
// went with a rule, and the output stays within the budget even then.
func TestADomainWithNoRoomIsTriedAgain(t *testing.T) {
	var names []string
	for i := range 40 {
		names = append(names, fmt.Sprintf("D%02d", i))
	}
	rs := sizedSet(2, 200, names...)
	const budget = 1500
	res := InjectWithin(rs, "zebra", SessionState{}, 0, BracketUnknown, budget)
	if len(res.Text) > budget {
		t.Fatalf("%d bytes over %d", len(res.Text), budget)
	}
	if len(res.Injected) == 0 || len(res.Injected) == len(names) {
		t.Fatalf("injected %d of %d", len(res.Injected), len(names))
	}
	if !strings.Contains(res.Text, injectionTruncatedMarker) {
		t.Fatalf("nothing says a domain was left out:\n%s", res.Text)
	}
	if strings.Contains(res.Text, "hostile") {
		t.Fatalf("routine overflow reads as an attack:\n%s", res.Text)
	}
	for _, n := range names {
		sent := slices.Contains(res.Injected, n) && !slices.Contains(res.Pointed, n)
		if recorded := res.State.Ledger[n] != ""; recorded != sent {
			t.Errorf("%s: sent %v, recorded %v", n, sent, recorded)
		}
	}
}

// TestATightBudgetStillSendsWhatFits: the notice's reserve never costs the
// only domain that fits.
func TestATightBudgetStillSendsWhatFits(t *testing.T) {
	rs := sizedSet(2, 200, "AAA", "BBB", "CCC", "DDD", "EEE", "FFF", "GGG", "HHH")
	const budget = 300
	res := InjectWithin(rs, "zebra", SessionState{}, 0, BracketUnknown, budget)
	if len(res.Text) > budget {
		t.Fatalf("%d bytes over %d", len(res.Text), budget)
	}
	if len(res.Injected) == 0 {
		t.Fatal("nothing was sent though a pointer fits")
	}
}

// TestNoBudgetSendsNothingAndRecordsNothing: what the status and away blocks
// took can leave no room at all.
func TestNoBudgetSendsNothingAndRecordsNothing(t *testing.T) {
	rs := sizedSet(2, 200, "AAA")
	res := InjectWithin(rs, "zebra", SessionState{}, 0, BracketUnknown, 0)
	if res.Text != "" || len(res.Injected) != 0 || res.State.Ledger["AAA"] != "" {
		t.Fatalf("text %q, injected %v, ledger %v", res.Text, res.Injected, res.State.Ledger)
	}
}

// TestTheBudgetSitsUnderTheHostCap holds the one figure: the whole turn and
// the rules within it.
func TestTheBudgetSitsUnderTheHostCap(t *testing.T) {
	if InjectionBudgetBytes > HostOutputCap {
		t.Fatalf("the turn budget %d is over the host cap %d", InjectionBudgetBytes, HostOutputCap)
	}
	if got, want := RulesBudget(100), InjectionBudgetBytes-100-MemoryFloorBytes; got != want {
		t.Fatalf("RulesBudget(100) = %d, want %d", got, want)
	}
	if got := RulesBudget(InjectionBudgetBytes * 2); got != 0 {
		t.Fatalf("RulesBudget past the budget = %d, want 0", got)
	}
}

// TestEveryBundledDomainTogetherFits: the worst a prompt can ask for from the
// bundled set, every domain named at once, in the row that keeps rules whole:
// each carries at least its first rule, none only a pointer.
func TestEveryBundledDomainTogetherFits(t *testing.T) {
	rs := Defaults()
	var prompt strings.Builder
	var names []string
	for name := range rs.Domains {
		names = append(names, name)
		prompt.WriteString(" *" + name)
	}
	res := Inject(rs, prompt.String(), SessionState{}, 0, BracketUnknown)
	if len(res.Text) > RulesBudget(0) {
		t.Fatalf("%d bytes, over %d", len(res.Text), RulesBudget(0))
	}
	for _, n := range names {
		if !strings.Contains(res.Text, "## "+n) {
			t.Errorf("%s was not sent:\n%s", n, res.Text)
		}
	}
	if len(res.Pointed) != 0 {
		t.Errorf("sent without a rule: %v", res.Pointed)
	}
}
