package lint

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/spec"
)

// intent_layout is iss-2611121000338677: a layout that makes an intent's text
// read more than one way, or an opener nothing closes, withholds readiness at
// the build's open_questions row (spec.LayoutStatements); record-lint names it
// where it is written, on its line, through the same reader. It reads the
// buckets that row judges, drafts and planned, and never a shipped,
// superseded or discipline record, which is never built.
func TestIntentLayout(t *testing.T) {
	const base = "rec/intents"
	const fm = "---\nid: itd-7\nkind: standalone\nspec_id: null\n---\n# T\n\n## Summary\n\n"
	// Line 10 opens the list item; the fence under it, indented four columns,
	// opens on line 12 and is read as text from there.
	const nested = fm + "- an item\n\n    ```\n    code\n    ```\n"
	cases := []struct {
		name  string
		rel   string
		body  string
		lines []int
		want  string
	}{
		{"a planned intent with a four-column fence under a list item is reported on its line",
			base + "/planned/itd-7-thing.md", nested, []int{12}, "is not read one way"},
		{"a planned intent with an unclosed fence is reported on its opener",
			base + "/planned/itd-7-thing.md", fm + "```\n## Open Questions\n- hidden?\n", []int{10}, "nothing below this line is read"},
		{"a planned intent with an unclosed comment is reported on its opener",
			base + "/planned/itd-7-thing.md", fm + "<!-- parked\nhidden\n", []int{10}, "nothing below this line is read"},
		{"a draft with the same layout is reported, since planning it reads the same row",
			base + "/drafts/itd-7-thing.md", nested, []int{12}, "is not read one way"},
		{"a planned intent read one way is clean",
			base + "/planned/itd-7-thing.md", fm + "- an item\n\n```\ncode\n```\n\n## Open Questions\n\n- resolved: settled.\n", nil, ""},
		{"a planned intent's own Not READY sentence is not a layout statement",
			base + "/planned/itd-7-thing.md", fm + "Not READY: a ruling is owed.\n", nil, ""},
		{"a shipped intent is never built and is out of reach",
			base + "/shipped/itd-7-thing.md", nested, nil, ""},
		{"a superseded intent is out of reach",
			base + "/superseded/itd-7-thing.md", nested, nil, ""},
		{"a discipline record is never built and is out of reach",
			base + "/disciplines/itd-7-thing.md", nested, nil, ""},
	}
	rule := RuleConfig{Enabled: true, Severity: severityWarn, IntentsDir: "intents"}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, c.rel, c.body)
			cfg := Config{Roots: []string{"rec"}, Rules: map[string]RuleConfig{ruleIntentLayout: rule}}
			fs, err := Lint(cfg, root)
			if err != nil {
				t.Fatal(err)
			}
			var got []Finding
			for _, f := range fs {
				if f.RuleID == ruleIntentLayout {
					got = append(got, f)
				}
			}
			if len(got) != len(c.lines) {
				t.Fatalf("want %d %s finding(s), got %+v", len(c.lines), ruleIntentLayout, fs)
			}
			for i, f := range got {
				if f.File != filepath.FromSlash(c.rel) || f.Line != c.lines[i] || f.Severity != severityWarn {
					t.Errorf("finding %+v, want %s line %d at %s", f, c.rel, c.lines[i], severityWarn)
				}
				for _, w := range []string{c.want, "open_questions", "intent ready", spec.LayoutRemedy} {
					if !strings.Contains(f.Message, w) {
						t.Errorf("message = %q, want it to contain %q", f.Message, w)
					}
				}
			}
		})
	}
}

// The rule names the lines the build's open_questions row names: each text,
// fed to spec.LayoutStatements and to record-lint, gives the same lines in the
// same order. Every fixture holds a statement, and they sit on different lines,
// so an offset or a dropped statement cannot pass.
func TestIntentLayoutNamesTheReadersLines(t *testing.T) {
	const rel = "rec/intents/planned/itd-7-thing.md"
	const fm = "---\nid: itd-7\nkind: standalone\nspec_id: null\n---\n# T\n\n## Summary\n\n"
	texts := []string{
		fm + "- an item\n\n    ```\n    code\n    ```\n\n## Open Questions\n\n- resolved: settled.\n",
		fm + "Some prose.\n\n```\n## Open Questions\n- hidden?\n",
		fm + "Some prose.\n\nMore prose.\n\n<!-- parked\nhidden\n",
	}
	rule := RuleConfig{Enabled: true, Severity: severityWarn, IntentsDir: "intents"}
	for i, text := range texts {
		var want []int
		for _, st := range spec.LayoutStatements(text) {
			want = append(want, st.Line)
		}
		if len(want) == 0 {
			t.Fatalf("fixture %d must hold a layout statement for the reader", i)
		}
		root := t.TempDir()
		writeFile(t, root, rel, text)
		fs, err := Lint(Config{Roots: []string{"rec"}, Rules: map[string]RuleConfig{ruleIntentLayout: rule}}, root)
		if err != nil {
			t.Fatal(err)
		}
		var got []int
		for _, f := range fs {
			if f.RuleID == ruleIntentLayout {
				got = append(got, f.Line)
			}
		}
		if len(got) != len(want) {
			t.Fatalf("fixture %d: %s lines = %v, reader's lines = %v", i, ruleIntentLayout, got, want)
		}
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("fixture %d: %s lines = %v, reader's lines = %v", i, ruleIntentLayout, got, want)
			}
		}
	}
}

// The shipped config arms intent_layout at warn: the build's open_questions
// row already refuses the intent, so the rule names it where it is written
// without blocking a run of record-lint, which exits 0 on warnings alone.
func TestIntentLayoutArmedAtWarnInRealConfig(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "..", ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	rc, ok := cfg.Rules[ruleIntentLayout]
	if !ok || !rc.Enabled || rc.Severity != severityWarn || rc.IntentsDir == "" {
		t.Fatalf("record-lint.json must arm %s at warn with intents_dir: %+v", ruleIntentLayout, rc)
	}
}
