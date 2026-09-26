package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// intent_sota is the discipline rung of sota-per-intent: a promoted intent
// declares the state of the art it is built against. The rule reads planned/
// only — the bucket an intent enters by being promoted — so a draft still on the
// bench and a shipped record that predates the principle are out of its reach,
// and a planned intent with no declaration, or a heading with nothing under it,
// is flagged at the configured severity (warn in the shipped config: the
// warn-first rung of the ratchet).
func TestIntentSOTA(t *testing.T) {
	const base = "rec/intents"
	const fm = "---\nid: itd-7\nkind: standalone\nspec_id: null\n---\n"
	cases := []struct {
		name     string
		rel      string
		body     string
		wantLine int // 0: no finding
		wantMsg  string
	}{
		{
			name: "planned intent declaring SOTA is clean",
			rel:  base + "/planned/itd-7-thing.md",
			body: fm + "# T\n\n## SOTA\n\nNothing importable. **Path 2.**\n\n## Open Questions\n",
		},
		{
			name:     "planned intent without a SOTA section is flagged",
			rel:      base + "/planned/itd-7-thing.md",
			body:     fm + "# T\n\n## Press Release\n\n> x\n",
			wantLine: 1,
			wantMsg:  "declares no `## SOTA` section",
		},
		{
			name:     "an empty SOTA section is flagged on its heading",
			rel:      base + "/planned/itd-7-thing.md",
			body:     fm + "# T\n\n## SOTA\n\n\n## Open Questions\n\n- q\n",
			wantLine: 8,
			wantMsg:  "`## SOTA` section is empty",
		},
		{
			name:     "a heading inside a fence is not a declaration",
			rel:      base + "/planned/itd-7-thing.md",
			body:     fm + "# T\n\n```markdown\n## SOTA\n\nfenced\n```\n",
			wantLine: 1,
			wantMsg:  "declares no `## SOTA` section",
		},
		{
			name:     "a deeper heading under SOTA is not a body on its own",
			rel:      base + "/planned/itd-7-thing.md",
			body:     fm + "# T\n\n## SOTA\n\n### Alternatives\n\n## Open Questions\n",
			wantLine: 8,
			wantMsg:  "`## SOTA` section is empty",
		},
		{
			name: "a draft is not yet held to the declaration",
			rel:  base + "/drafts/itd-7-thing.md",
			body: fm + "# T\n",
		},
		{
			name: "a shipped record is not held to it",
			rel:  base + "/shipped/itd-7-thing.md",
			body: fm + "# T\n",
		},
	}
	rule := RuleConfig{Enabled: true, Severity: severityWarn, IntentsDir: "intents"}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, c.rel, c.body)
			cfg := Config{Roots: []string{"rec"}, Rules: map[string]RuleConfig{ruleIntentSOTA: rule}}
			fs, err := Lint(cfg, root)
			if err != nil {
				t.Fatal(err)
			}
			n := countRule(fs, ruleIntentSOTA)
			if c.wantLine == 0 {
				if n != 0 {
					t.Fatalf("want no %s finding, got %+v", ruleIntentSOTA, fs)
				}
				return
			}
			if n != 1 {
				t.Fatalf("want exactly one %s finding, got %d: %+v", ruleIntentSOTA, n, fs)
			}
			f := fs[0]
			if f.File != filepath.FromSlash(c.rel) || f.Line != c.wantLine {
				t.Errorf("finding at %s:%d, want %s:%d", f.File, f.Line, c.rel, c.wantLine)
			}
			if f.Severity != severityWarn {
				t.Errorf("severity = %q, want the configured %q", f.Severity, severityWarn)
			}
			if !strings.Contains(f.Message, c.wantMsg) || !strings.Contains(f.Message, "sota-per-intent") {
				t.Errorf("message = %q, want it to contain %q and name the principle", f.Message, c.wantMsg)
			}
		})
	}
}

// The shipped config arms intent_sota at warn: the warn-first rung of the
// ratchet, so the forty-odd planned intents that predate the rule are named on
// every run without blocking one. Promoting it to blocker is a later, separate
// act once the planned bucket is back-filled.
func TestIntentSOTAArmedInRealConfig(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "..", ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	rc, ok := cfg.Rules[ruleIntentSOTA]
	if !ok || !rc.Enabled {
		t.Fatalf("record-lint.json must enable %s", ruleIntentSOTA)
	}
	if rc.Severity != severityWarn {
		t.Errorf("%s severity = %q, want %q (warn-first)", ruleIntentSOTA, rc.Severity, severityWarn)
	}
	if rc.IntentsDir == "" {
		t.Errorf("%s must declare intents_dir, like intent_lifecycle", ruleIntentSOTA)
	}
}
