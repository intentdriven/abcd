package ahoy

import (
	"strings"
	"testing"
)

// recordingPrompter answers every confirm yes and keeps the questions in the
// order they were asked. Order is the whole subject here: a caller that answers
// positionally — a human reading down the list, or a piped stream of answers —
// can only match answers to questions if the questions come in a fixed order.
type recordingPrompter struct {
	asked   []string
	confirm bool
	// terminal makes it a TerminalPrompter a person answers at a terminal.
	terminal bool
}

func (p *recordingPrompter) AtTerminal() bool { return p.terminal }

func (p *recordingPrompter) Confirm(q string) bool {
	p.asked = append(p.asked, q)
	return p.confirm
}

func (p *recordingPrompter) Prompt(_ string, _ []string, def string) string { return def }

// allCategoryGaps is one resolvable gap per category, so every category is
// present and every one of them is asked about.
func allCategoryGaps() []Gap {
	return []Gap{
		{ID: "user.a", Category: UserState, Resolvable: true},
		{ID: "config.b", Category: ConfigChange, Resolvable: true},
		{ID: "plugin.c", Category: PluginOwned, Resolvable: true},
		{ID: "deps.d", Category: Dependency, Resolvable: true},
		{ID: "skeleton.e", Category: SafeAutocreate, Resolvable: true},
		{ID: "statusline.f", Category: StatusLine, Resolvable: true},
		{ID: "oracle_routing.g", Category: OracleRouting, Resolvable: true},
		{ID: "drain_rule.h", Category: DrainRule, Resolvable: true},
		{ID: "conventions.i", Category: ConventionsFile, Resolvable: true},
	}
}

// TestResolveApprovalPromptsInCanonicalOrder is the order contract: the same
// gaps must produce the same questions in the same sequence on every run.
//
// Without it "answer y to the first question" means a different category each
// time, and a caller feeding answers positionally approves a category at
// random — a silent, nondeterministic wrong answer, not a visible failure. The
// repetitions are the test: a single run of a randomised order passes by luck
// often enough to be useless.
func TestResolveApprovalPromptsInCanonicalOrder(t *testing.T) {
	want := []string{
		"Apply dependency changes?",
		"Apply safe-autocreate changes?",
		"Apply config-change changes?",
		"Apply status-line changes?",
		"Apply oracle-routing changes?",
		"Apply drain-rule changes?",
		"Apply conventions-file changes?",
		"Apply user-state changes?",
		"Apply plugin-owned changes?",
	}
	for i := 0; i < 64; i++ {
		p := &recordingPrompter{confirm: true, terminal: true}
		resolveApproval(allCategoryGaps(), InstallOptions{}, p)
		if strings.Join(p.asked, "|") != strings.Join(want, "|") {
			t.Fatalf("run %d asked in a different order:\n got %v\nwant %v", i, p.asked, want)
		}
	}
	// Off a terminal the drain-rule and conventions-file questions are not
	// asked (stepDrainRule, stepConventionsFiles), and not counted as
	// declined: the rest keep their order, so a piped stream written before
	// the offers existed still lines up.
	offTerminal := make([]string, 0, len(want))
	for _, q := range want {
		if q != "Apply drain-rule changes?" && q != "Apply conventions-file changes?" {
			offTerminal = append(offTerminal, q)
		}
	}
	p := &recordingPrompter{confirm: false}
	_, declined := resolveApproval(allCategoryGaps(), InstallOptions{}, p)
	if strings.Join(p.asked, "|") != strings.Join(offTerminal, "|") {
		t.Fatalf("off a terminal:\n got %v\nwant %v", p.asked, offTerminal)
	}
	for _, c := range declined {
		if c == string(DrainRule) || c == string(ConventionsFile) {
			t.Fatalf("off a terminal an unasked terminal-only category is reported declined: %v", declined)
		}
	}
}

// TestCategoryPromptOrderCoversEveryCategory keeps the canonical order honest:
// a category missing from it would be asked in the sorted-tail fallback, which
// is still deterministic but no longer the order the apply pass acts in.
func TestCategoryPromptOrderCoversEveryCategory(t *testing.T) {
	all := []GapCategory{SafeAutocreate, ConfigChange, PluginOwned, Dependency, UserState, StatusLine, OracleRouting, DrainRule, ConventionsFile}
	for _, c := range all {
		found := false
		for _, oc := range categoryPromptOrder {
			if oc == c {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("category %q is not in categoryPromptOrder", c)
		}
	}
	if len(categoryPromptOrder) != len(all) {
		t.Errorf("categoryPromptOrder has %d entries, want %d", len(categoryPromptOrder), len(all))
	}
}

// TestResolveApprovalAsksUnknownCategoriesLast proves the fallback is
// deterministic too: a category added to the type but not to the canonical
// order is still asked in a fixed place, sorted, rather than at random.
func TestResolveApprovalAsksUnknownCategoriesLast(t *testing.T) {
	gaps := append(allCategoryGaps(),
		Gap{ID: "zeta.a", Category: GapCategory("zeta"), Resolvable: true},
		Gap{ID: "alpha.a", Category: GapCategory("alpha"), Resolvable: true},
	)
	for i := 0; i < 32; i++ {
		p := &recordingPrompter{confirm: true, terminal: true}
		resolveApproval(gaps, InstallOptions{}, p)
		if len(p.asked) != 11 {
			t.Fatalf("asked %d questions, want 11: %v", len(p.asked), p.asked)
		}
		tail := strings.Join(p.asked[9:], "|")
		if tail != "Apply alpha changes?|Apply zeta changes?" {
			t.Fatalf("run %d: unknown categories not asked last and sorted: %v", i, p.asked)
		}
	}
}
