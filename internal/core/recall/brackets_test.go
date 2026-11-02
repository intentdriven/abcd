package recall

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/rules"
)

// inBracket is one recall in a bracket, with room for everything.
func inBracket(items []Item, prompt string, ledger map[string]string, b rules.Bracket) Result {
	return Recall(Request{Items: items, Prompt: prompt, Ledger: ledger, Budget: rules.InjectionBudgetBytes, Bracket: b})
}

// oneLine is the line an item is sent as when it is not sent in full.
func oneLine(it Item) string { return renderLine(it) }

// TestRecallShrinksFullThenOneLineToTheEndOfTheSession (N8): over one
// session whose fill rises, notes and pages come back in full while it is
// fresh, the top-ranked in full and the rest as one line each when it is
// moderate, and as one line each once it is depleted — and stay at one line
// to the end of the session.
func TestRecallShrinksFullThenOneLineToTheEndOfTheSession(t *testing.T) {
	big := strings.TrimSpace(strings.Repeat("charlie's long account of the matter ", 300)) // over 8 KiB
	items := []Item{
		item("mem-2610111200000001", []string{"alpha"}, "Alpha lesson", "Alpha body."),
		item("mem-2610111200000002", []string{"bravo", "charlie"}, "Bravo lesson", "Bravo body."),
		item("mem-2610111200000003", []string{"charlie"}, "Charlie lesson", big),
		item("mem-2610111200000004", []string{"delta"}, "Delta lesson", "Delta body."),
		item("mem-2610111200000005", []string{"echo"}, "Echo lesson", "Echo body."),
	}
	var ledger map[string]string
	turn := func(prompt string, b rules.Bracket) Result {
		t.Helper()
		res := inBracket(items, prompt, ledger, b)
		ledger = res.Ledger
		return res
	}

	fresh := turn("alpha", rules.BracketFresh)
	if !strings.Contains(fresh.Text, "Alpha body.") || fresh.Form("mem-2610111200000001") != FormFull {
		t.Fatalf("fresh: the note did not come back in full:\n%s", fresh.Text)
	}

	moderate := turn("bravo charlie", rules.BracketModerate)
	if !slices.Equal(moderate.Recalled, []string{"mem-2610111200000002", "mem-2610111200000003"}) {
		t.Fatalf("moderate: recalled %v", moderate.Recalled)
	}
	if !strings.Contains(moderate.Text, "Bravo body.") || moderate.Form("mem-2610111200000002") != FormFull {
		t.Fatalf("moderate: the top-ranked note did not come back in full:\n%s", moderate.Text)
	}
	if strings.Contains(moderate.Text, "long account") || !strings.Contains(moderate.Text, oneLine(items[2])) ||
		moderate.Form("mem-2610111200000003") != FormLine {
		t.Fatalf("moderate: the note past 8 KiB did not come back as one line:\n%.400s", moderate.Text)
	}

	for _, p := range []string{"delta", "echo"} {
		depleted := turn(p, rules.BracketDepleted)
		if len(depleted.Recalled) != 1 || depleted.Form(depleted.Recalled[0]) != FormLine || strings.Contains(depleted.Text, " body.") {
			t.Fatalf("depleted, %q: not one line:\n%s", p, depleted.Text)
		}
	}
}

// TestNoBracketSendsNothing: in every row of the table, a matched note comes
// back in some form — the facilitator's "never send nothing".
func TestNoBracketSendsNothing(t *testing.T) {
	items := []Item{item("mem-2610111200000001", []string{"deploy"}, "Deploys need the flag", "Pass the dry-run flag first.")}
	for _, row := range rules.BracketTable() {
		res := inBracket(items, "deploy it", nil, row.Bracket)
		if res.Text == "" || !slices.Equal(res.Recalled, []string{"mem-2610111200000001"}) {
			t.Errorf("%v sent nothing for a matched note: %+v", row.Bracket, res)
		}
		if !strings.Contains(res.Text, "Deploys need the flag") || !strings.Contains(res.Text, "mem-2610111200000001") {
			t.Errorf("%v: the note's title and id are not in the block:\n%s", row.Bracket, res.Text)
		}
	}
}

// TestDepletedBracketInjectsIndexLinesOnly (P2): below 40% free, three
// matched pages come back as their index lines — each page's name and
// summary — and none of their bodies.
func TestDepletedBracketInjectsIndexLinesOnly(t *testing.T) {
	home, repo, workspace, _ := pagesFixture(t)
	writePage(t, workspace, "fact_eng_auth-flow.md", "# Auth uses short-lived tokens\n\nFlow body words.\n", "auth")
	writePage(t, workspace, "fact_eng_auth-keys.md", "# Auth keys live in the keychain\n\nKeys body words.\n", "auth")
	writePage(t, workspace, "fact_eng_auth-audit.md", "# Auth is audited weekly\n\nAudit body words.\n", "auth", "login")
	items, refused := ReadItems(home, repo)
	if len(refused) != 0 {
		t.Fatalf("refused = %+v", refused)
	}
	res := inBracket(items, "the auth login is broken", nil, rules.BracketDepleted)
	if len(res.Recalled) != 3 {
		t.Fatalf("recalled %v, want the three pages", res.Recalled)
	}
	for _, want := range []string{"page:fact_eng_auth-flow.md", "Auth uses short-lived tokens", "page:fact_eng_auth-keys.md",
		"Auth keys live in the keychain", "page:fact_eng_auth-audit.md", "Auth is audited weekly"} {
		if !strings.Contains(res.Text, want) {
			t.Errorf("the block does not carry %q:\n%s", want, res.Text)
		}
	}
	if strings.Contains(res.Text, "body words") || strings.Contains(res.Text, "~~~") {
		t.Errorf("a page's body came back below 40%% free:\n%s", res.Text)
	}
	if lines := strings.Count(res.Text, "\n"); lines != 4 {
		t.Errorf("the block is %d lines, want the heading and one per page:\n%s", lines, res.Text)
	}
}

// TestTopRankedLineAlwaysComesBack: in the depleted bracket the top-ranked
// candidate comes back even when its line alone is past the 2 KiB the lines
// get; the next does not, and is tried again on the next turn.
func TestTopRankedLineAlwaysComesBack(t *testing.T) {
	long := strings.TrimSpace(strings.Repeat("kerberos deploy title ", 120)) // a line past 2 KiB
	items := []Item{
		item("mem-2610111200000001", []string{"deploy"}, "A deploy lesson", "Body."),
		item("mem-2610111200000002", []string{"deploy", "kerberos"}, long, "Body."),
	}
	res := inBracket(items, "deploy with kerberos", nil, rules.BracketDepleted)
	if !slices.Equal(res.Recalled, []string{"mem-2610111200000002"}) || !slices.Equal(res.Omitted, []string{"mem-2610111200000001"}) {
		t.Fatalf("recalled %v omitted %v; want the top-ranked line alone", res.Recalled, res.Omitted)
	}
	if _, ok := res.Ledger["mem-2610111200000001"]; ok {
		t.Fatal("a line that did not fit was recorded, so it would never come back")
	}
	next := inBracket(items, "deploy with kerberos", res.Ledger, rules.BracketDepleted)
	if !slices.Equal(next.Recalled, []string{"mem-2610111200000001"}) {
		t.Fatalf("the next turn recalled %v; want the line that did not fit", next.Recalled)
	}
}

// TestRankingChoosesWhichLinesFit: when only some lines fit the depleted
// bracket's 2 KiB, the ranked order chooses them, not the id order.
func TestRankingChoosesWhichLinesFit(t *testing.T) {
	var items []Item
	for i := 0; i < 40; i++ {
		// The later the id, the more "kerberos" recall words, so the
		// ranked order is the reverse of the id order.
		recall := []string{"deploy"}
		if i >= 20 {
			recall = append(recall, "kerberos")
		}
		items = append(items, item(fmt.Sprintf("mem-26101112000000%02d", i), recall,
			fmt.Sprintf("Lesson %02d about deploying a service with its own words", i), "Body."))
	}
	res := inBracket(items, "deploy with kerberos", nil, rules.BracketDepleted)
	row := rules.BracketDepleted.Row()
	if len(res.Recalled) == 0 || len(res.Recalled) >= len(items) {
		t.Fatalf("recalled %d of %d; the 2 KiB should hold some and not all", len(res.Recalled), len(items))
	}
	ranked := rankedKeys(rankOf(items, "deploy with kerberos"))
	if !slices.Equal(res.Recalled, ranked[:len(res.Recalled)]) {
		t.Fatalf("recalled %v, want the ranked prefix %v", res.Recalled, ranked[:len(res.Recalled)])
	}
	size := len(res.Text) - len(headerLine)
	if size > row.MemoryLineBytes {
		t.Fatalf("the lines are %d bytes, past the bracket's %d", size, row.MemoryLineBytes)
	}
	byKey := map[string]Item{}
	for _, it := range items {
		byKey[it.Key] = it
	}
	if next := len(oneLine(byKey[ranked[len(res.Recalled)]])); size+next <= row.MemoryLineBytes {
		t.Fatalf("the next ranked line (%d bytes) would have fit in %d", next, row.MemoryLineBytes-size)
	}
}

// TestModerateBracketKeepsTheTopInFullWithin8KiB: in the moderate bracket
// the top-ranked candidates come back in full while their blocks fit 8 KiB;
// the first that does not, and every one ranked below it, comes back as one
// line, so nothing comes back in full below an item sent as a line.
func TestModerateBracketKeepsTheTopInFullWithin8KiB(t *testing.T) {
	mid := strings.TrimSpace(strings.Repeat("words ", 600)) // about 3.6 KiB
	items := []Item{
		item("mem-2610111200000001", []string{"deploy", "kerberos", "service"}, "First", mid),
		item("mem-2610111200000002", []string{"deploy", "kerberos"}, "Second", mid),
		item("mem-2610111200000003", []string{"deploy", "service"}, "Third", mid),
		item("mem-2610111200000004", []string{"deploy"}, "Fourth", "Short."),
	}
	res := inBracket(items, "deploy the kerberos service", nil, rules.BracketModerate)
	want := map[string]string{"mem-2610111200000001": FormFull, "mem-2610111200000002": FormFull,
		"mem-2610111200000003": FormLine, "mem-2610111200000004": FormLine}
	for k, f := range want {
		if res.Form(k) != f {
			t.Errorf("%s came back %q, want %q", k, res.Form(k), f)
		}
	}
}

// TestUnknownFillShortensNotesAndPages: with no source for the fill (the
// zero bracket), notes and pages come back as one line each, carrying their
// id and title and never their body.
func TestUnknownFillShortensNotesAndPages(t *testing.T) {
	items := []Item{
		item("mem-2610111200000001", []string{"deploy"}, "Deploys need the flag", "Pass the dry-run flag first."),
		{Key: "page:fact_eng_deploy.md", Kind: KindPage, Layer: LayerMachine, Title: "Deploys go out on Tuesdays", Body: "The page body.", Recall: []string{"deploy"}},
	}
	res := Recall(Request{Items: items, Prompt: "deploy it", Budget: rules.InjectionBudgetBytes})
	if len(res.Recalled) != 2 {
		t.Fatalf("recalled %v", res.Recalled)
	}
	for _, it := range items {
		if !strings.Contains(res.Text, oneLine(it)) || res.Form(it.Key) != FormLine {
			t.Errorf("%s is not one line:\n%s", it.Key, res.Text)
		}
		if strings.Contains(res.Text, it.Body) {
			t.Errorf("%s's body came back under an unknown fill:\n%s", it.Key, res.Text)
		}
	}
}

// TestABracketChangeNeverResendsAnItem: the recalled ledger keys on an
// item's content, not on its form, so an item sent as one line is not sent
// again in full when the fill is read lower, nor the other way round.
func TestABracketChangeNeverResendsAnItem(t *testing.T) {
	items := []Item{item("mem-2610111200000001", []string{"deploy"}, "Deploys need the flag", "Pass the dry-run flag first.")}
	line := inBracket(items, "deploy it", nil, rules.BracketDepleted)
	if again := inBracket(items, "deploy it", line.Ledger, rules.BracketFresh); again.Text != "" || len(again.Recalled) != 0 {
		t.Fatalf("a line was sent again in full:\n%s", again.Text)
	}
	full := inBracket(items, "deploy it", nil, rules.BracketFresh)
	if again := inBracket(items, "deploy it", full.Ledger, rules.BracketUnknown); again.Text != "" {
		t.Fatalf("a full item was sent again as a line:\n%s", again.Text)
	}
}
