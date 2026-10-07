package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
)

// repoRoot is the checkout this package sits in.
func repoRoot() string { return filepath.Join("..", "..") }

func readPage(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(), filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// committedBlock is the text between the asking-rules markers of page.
func committedBlock(t *testing.T, page string) string {
	t.Helper()
	start := strings.Index(page, beginMarker)
	if start < 0 {
		t.Fatalf("the page carries no %s marker", beginMarker)
	}
	rest := page[start+len(beginMarker):]
	end := strings.Index(rest, endMarker)
	if end < 0 {
		t.Fatalf("the page opens %s and never closes it with %s", beginMarker, endMarker)
	}
	return rest[:end]
}

// TestIntentPageAskingBlockIsGenerated holds the asking-rules block of
// commands/intent.md to the rendering from question.Default
// (spc-2610030944505997 criterion A7): a limit changed in Limits, or a rule
// reworded in asking.go, without running `make asking-sync` fails here, under
// preflight, rather than leaving the page stating a limit the check does not
// enforce.
func TestIntentPageAskingBlockIsGenerated(t *testing.T) {
	page := readPage(t, pagePath)
	if got, want := committedBlock(t, page), render(question.Default); got != want {
		t.Fatalf("commands/intent.md's asking-rules block differs from the rendering from question.Default; run `make asking-sync`.\n--- committed\n%s\n--- rendered\n%s", got, want)
	}
	synced, err := sync([]byte(page), question.Default)
	if err != nil {
		t.Fatal(err)
	}
	if string(synced) != page {
		t.Fatal("sync would rewrite the committed page; run `make asking-sync`")
	}
	for _, gone := range []string{"**What each register is assumed to know.**", "declared in abcd's own repository alone"} {
		if strings.Contains(page, gone) {
			t.Errorf("commands/intent.md still carries %q; the generated block and the glossary page replace it", gone)
		}
	}
}

// sixWordQuestion is a well-built question whose first label is six words:
// refused at the default five, admitted when the limit is six.
func sixWordQuestion() question.Fields {
	return question.Fields{Tabs: []question.Tab{{
		Header: "Product Q2",
		Text: "A person who answers a question can change the answer later.\n\n" +
			"Now: not applicable\nChange later: not applicable\n\nDoes it stand?",
		Options: []question.Choice{
			{Label: "Keep it as it is written", Description: "The criterion stands as written."},
			{Label: "Change it", Description: "The criterion is reworded before planning."},
			{Label: "Decide later", Description: "The criterion stays open until the next walk."},
		},
	}}}
}

// TestOneEditMovesEveryStatementOfALimit is criterion A7 end to end: one edit
// to a copy of question.Default moves the check, the GRILL rule text, and the
// intent page's block together, because all three read the one value.
func TestOneEditMovesEveryStatementOfALimit(t *testing.T) {
	who := question.Addressee{Person: question.ProductThinker}
	before := question.Default
	after := question.Default
	after.LabelWords = 6

	refused := false
	for _, f := range question.CheckLimits(sixWordQuestion(), before, who) {
		if f.Rule == question.RuleLabelWords {
			refused = true
		}
	}
	if !refused {
		t.Fatal("a six-word label is not refused at the default limit of five; the fixture no longer tests the edit")
	}
	if got := question.CheckLimits(sixWordQuestion(), after, who); len(got) != 0 {
		t.Fatalf("a six-word label is still refused after the limit is raised to six: %v", got)
	}

	rulesBefore := strings.Join(question.AskingRules(before), "\n")
	rulesAfter := strings.Join(question.AskingRules(after), "\n")
	if !strings.Contains(rulesBefore, "at most five words") || !strings.Contains(rulesAfter, "at most six words") ||
		strings.Contains(rulesAfter, "at most five words") {
		t.Error("the GRILL rule text does not move from \"five words\" to \"six words\" with the limit")
	}

	pageBefore, pageAfter := render(before), render(after)
	if !strings.Contains(pageBefore, "at most five words") || !strings.Contains(pageAfter, "at most six words") ||
		strings.Contains(pageAfter, "at most five words") {
		t.Error("the intent page's block does not move from \"five words\" to \"six words\" with the limit")
	}

	page := readPage(t, pagePath)
	synced, err := sync([]byte(page), after)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(committedBlock(t, string(synced)), "at most six words") {
		t.Error("syncing the page from the edited limits does not write \"six words\" into its block")
	}
}

// TestRenderStatesEveryAskingRule holds the block to every rule the GRILL
// domain carries, each on its own list line, and to the knowledge-floor
// pointer, so the page never drops or paraphrases one.
func TestRenderStatesEveryAskingRule(t *testing.T) {
	block := render(question.Default)
	for _, r := range question.AskingRules(question.Default) {
		if !strings.Contains(block, "\n- "+r+"\n") {
			t.Errorf("the block does not carry the rule as its own list item: %s", r)
		}
	}
	if !strings.Contains(block, "../"+question.KnowledgeFloorPage) {
		t.Errorf("the block does not link the knowledge floor at %s", question.KnowledgeFloorPage)
	}
}

// TestSyncRefusesAPageWithoutOneBlock: a page with no block, an unclosed
// block, or two blocks is a fault, never a guess about where to write.
func TestSyncRefusesAPageWithoutOneBlock(t *testing.T) {
	for name, page := range map[string]string{
		"no block": "# Page\n\nText.\n",
		"unclosed": "# Page\n\n" + beginMarker + "\nold\n",
		"two":      beginMarker + "\n" + endMarker + "\n" + beginMarker + "\n" + endMarker + "\n",
	} {
		if _, err := sync([]byte(page), question.Default); err == nil {
			t.Errorf("%s: sync accepted the page", name)
		}
	}
}

// TestSyncWritesTheBlockAndKeepsTheRest: the text around the block is kept
// byte for byte, and a second sync changes nothing.
func TestSyncWritesTheBlockAndKeepsTheRest(t *testing.T) {
	page := "# Page\n\nBefore.\n\n" + beginMarker + "\nstale\n" + endMarker + "\n\nAfter.\n"
	out, err := sync([]byte(page), question.Default)
	if err != nil {
		t.Fatal(err)
	}
	want := "# Page\n\nBefore.\n\n" + beginMarker + render(question.Default) + endMarker + "\n\nAfter.\n"
	if string(out) != want {
		t.Fatalf("sync wrote\n%s\nwant\n%s", out, want)
	}
	again, err := sync(out, question.Default)
	if err != nil || string(again) != string(out) {
		t.Fatalf("a second sync changed the page (err %v)", err)
	}
}

// bindingPages are the interviews outside the planning page whose questions
// follow the same order (spc-2610030944505997 criterion S4): the
// retrospective, the unpacking interview, and the setup's routing
// confirmation.
var bindingPages = []string{"commands/reflect.md", "commands/embark.md", "commands/ahoy.md"}

// TestInterviewPagesBindTheOrder holds each interview page to one sentence
// binding its questions to the thing-first order and pointing at the
// generated block, and shows the check holding a retrospective question to
// that order: the earlier answer quoted first is admitted, the bare question
// refused (criterion S4).
func TestInterviewPagesBindTheOrder(t *testing.T) {
	for _, rel := range bindingPages {
		page := strings.Join(strings.Fields(readPage(t, rel)), " ")
		for _, want := range []string{"`commands/intent.md`", "`" + strings.TrimSuffix(strings.TrimPrefix(beginMarker, "<!-- "), " -->") + "`", "the question last"} {
			if !strings.Contains(page, want) {
				t.Errorf("%s does not carry %q: it must bind its questions to the thing-first order and point at the generated block", rel, want)
			}
		}
		if strings.Contains(page, "GRILL rules") {
			t.Errorf("%s still names \"the GRILL rules\"; point at the generated block instead", rel)
		}
	}

	who := question.Addressee{Person: question.ProductThinker}
	quoted := question.Fields{Tabs: []question.Tab{{
		Header: "Product Q1/4",
		Text: "Your answer for what went well:\n\n" +
			"The release reached everyone on the day it was planned for, with no rollback.\n\n" +
			"Now: not applicable\nChange later: not applicable\n\nIs this answer complete?",
		Options: []question.Choice{
			{Label: "Yes, it is complete", Description: "The answer is written as it stands."},
			{Label: "Add to it", Description: "You add what is missing before it is written."},
			{Label: "Decide later", Description: "The section stays open and is asked again."},
		},
	}}}
	if got := question.CheckLimits(quoted, question.Default, who); len(got) != 0 {
		t.Errorf("the retrospective question with the earlier answer quoted first is refused: %v", got)
	}
	bare := quoted
	bare.Tabs = []question.Tab{quoted.Tabs[0]}
	bare.Tabs[0].Text = "Is this answer complete?"
	thingFirst := false
	for _, f := range question.CheckLimits(bare, question.Default, who) {
		if f.Rule == question.RuleThingFirst {
			thingFirst = true
		}
	}
	if !thingFirst {
		t.Error("the retrospective question asked without the earlier answer quoted first is not refused under the thing-first rule")
	}
}

// TestCheckWritesNothingAndSyncWrites: -check reports a stale block and leaves
// the page as it was; without it the block is written, and a second run finds
// nothing to do.
func TestCheckWritesNothingAndSyncWrites(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, filepath.FromSlash(pagePath))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	stale := "# Page\n\n" + beginMarker + "\nstale\n" + endMarker + "\n"
	if err := os.WriteFile(p, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	writeCurrentDrafter(t, root)
	if err := run(root, true); !errors.Is(err, errOutOfDate) {
		t.Fatalf("-check over a stale block returned %v, want errOutOfDate", err)
	}
	if got, _ := os.ReadFile(p); string(got) != stale {
		t.Fatal("-check wrote the page")
	}
	if err := run(root, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(p); !strings.Contains(string(got), render(question.Default)) {
		t.Fatal("the sync did not write the rendered block")
	}
	if err := run(root, true); err != nil {
		t.Fatalf("-check after the sync returned %v", err)
	}
}

// writeCurrentDrafter lays the drafter agent's page under root with its block
// already rendered, so a test about the intent page is not tripped by the
// other target.
func writeCurrentDrafter(t *testing.T, root string) {
	t.Helper()
	d := filepath.Join(root, filepath.FromSlash(drafterPath))
	if err := os.MkdirAll(filepath.Dir(d), 0o755); err != nil {
		t.Fatal(err)
	}
	page := "# Drafter\n\n" + drafterBeginMarker + renderDrafter(question.Default) + endMarker + "\n"
	if err := os.WriteFile(d, []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
}

// drafterBlock is the text between the drafter page's markers.
func drafterBlock(t *testing.T, page string) string {
	t.Helper()
	start := strings.Index(page, drafterBeginMarker)
	if start < 0 {
		t.Fatalf("the drafter page carries no %s marker", drafterBeginMarker)
	}
	rest := page[start+len(drafterBeginMarker):]
	end := strings.Index(rest, endMarker)
	if end < 0 {
		t.Fatalf("the drafter page opens %s and never closes it with %s", drafterBeginMarker, endMarker)
	}
	return rest[:end]
}

// TestDrafterBlockIsGenerated holds the question-drafter agent's generated
// block to the rendering from question.Default, as the intent page's block
// is held: the agent drafts to the rules and the row count the check
// enforces, read from the one value, never from a copy in the page.
func TestDrafterBlockIsGenerated(t *testing.T) {
	page := readPage(t, drafterPath)
	if got, want := drafterBlock(t, page), renderDrafter(question.Default); got != want {
		t.Fatalf("%s's generated block differs from the rendering from question.Default; run `make asking-sync`.\n--- committed\n%s\n--- rendered\n%s", drafterPath, got, want)
	}
}

// TestDrafterBlockStatesTheRulesAndTheRowCount: the drafter's block carries
// every asking rule as its own list item and the row count the check makes,
// each figure filled from the Limits it is handed, so an edit to a limit moves
// the agent's arithmetic with the check's.
func TestDrafterBlockStatesTheRulesAndTheRowCount(t *testing.T) {
	block := renderDrafter(question.Default)
	for _, r := range question.AskingRules(question.Default) {
		if !strings.Contains(block, "\n- "+r+"\n") {
			t.Errorf("the drafter block does not carry the rule as its own list item: %s", r)
		}
	}
	l := question.Default
	for _, want := range []string{
		fmt.Sprintf("%d rows", l.HostChromeRows),
		fmt.Sprintf("wrapped at %d columns", l.HostTextColumns),
		fmt.Sprintf("wrapped at %d columns", l.HostOptionColumns),
		fmt.Sprintf("over %d rows", l.Rows),
		fmt.Sprintf("%d columns wide", l.Columns),
		"a blank line is one row",
		"hard-wraps",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("the drafter block does not state %q:\n%s", want, block)
		}
	}
	edited := question.Default
	edited.HostChromeRows, edited.HostTextColumns, edited.HostOptionColumns, edited.Rows = 9, 70, 68, 30
	moved := renderDrafter(edited)
	for _, want := range []string{"9 rows", "wrapped at 70 columns", "wrapped at 68 columns", "over 30 rows"} {
		if !strings.Contains(moved, want) {
			t.Errorf("the drafter block does not move to %q with the limits", want)
		}
	}
}

// TestCheckCoversTheDrafterBlock: -check reports a stale drafter block and
// writes nothing; the sync then writes it, and the intent page beside it is
// kept as it was.
func TestCheckCoversTheDrafterBlock(t *testing.T) {
	root := t.TempDir()
	intent := filepath.Join(root, filepath.FromSlash(pagePath))
	if err := os.MkdirAll(filepath.Dir(intent), 0o755); err != nil {
		t.Fatal(err)
	}
	current := "# Page\n\n" + beginMarker + render(question.Default) + endMarker + "\n"
	if err := os.WriteFile(intent, []byte(current), 0o644); err != nil {
		t.Fatal(err)
	}
	d := filepath.Join(root, filepath.FromSlash(drafterPath))
	if err := os.MkdirAll(filepath.Dir(d), 0o755); err != nil {
		t.Fatal(err)
	}
	stale := "# Drafter\n\n" + drafterBeginMarker + "\nstale\n" + endMarker + "\n"
	if err := os.WriteFile(d, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	err := run(root, true)
	if !errors.Is(err, errOutOfDate) || !strings.Contains(err.Error(), drafterPath) {
		t.Fatalf("-check over a stale drafter block returned %v, want errOutOfDate naming %s", err, drafterPath)
	}
	if got, _ := os.ReadFile(d); string(got) != stale {
		t.Fatal("-check wrote the drafter page")
	}
	if err := run(root, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(d); !strings.Contains(string(got), renderDrafter(question.Default)) {
		t.Fatal("the sync did not write the drafter block")
	}
	if got, _ := os.ReadFile(intent); string(got) != current {
		t.Fatal("the sync changed an intent page that was already current")
	}
	if err := run(root, true); err != nil {
		t.Fatalf("-check after the sync returned %v", err)
	}
}
