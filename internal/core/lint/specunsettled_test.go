package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// spec_unsettled is iss-2611101318042398's record-lint half: an open spec that
// says in its own text that it is not ready — a sentence opening "Not READY",
// or a bold "Open:" label on any line but the one an open question's item
// opens on — is refused, one finding per statement on its line. It reads through spec.Unsettled, the
// reader the readiness gate's spec_body row asks, so the two never disagree. A
// question asked under the section is the place a spec asks one, and passes; a
// closed spec is never built again and is out of reach.
func TestSpecUnsettled(t *testing.T) {
	const base = "rec/specs"
	const fm = "---\nid: spc-7\nslug: thing\nintent: itd-7\n---\n# T\n\n## Summary\n\n"
	cases := []struct {
		name  string
		rel   string
		body  string
		lines []int
		want  string
	}{
		{"a not-READY sentence in the summary is refused", base + "/open/spc-7-thing.md",
			fm + "_Filed by a remainder close. Not READY: the first step needs a ruling._\n", []int{10}, "Not READY"},
		{"a not-READY sentence opening a line is refused", base + "/open/spc-7-thing.md",
			fm + "Written.\nNot READY: its design is unsettled below.\n", []int{11}, "Not READY"},
		{"an open question outside Open Questions is refused", base + "/open/spc-7-thing.md",
			fm + "- The cache needs a ruling. **Open:** one cache file, or a rollup?\n", []int{10}, "Open:"},
		{"a bold Open label after the bold is refused", base + "/open/spc-7-thing.md",
			fm + "**Open question**: which cache?\n", []int{10}, "Open:"},
		{"an open question under Open Questions passes", base + "/open/spc-7-thing.md",
			fm + "Written.\n\n## Open Questions\n\n- **Open:** one cache file, or a rollup?\n", nil, ""},
		{"prose that only mentions readiness passes", base + "/open/spc-7-thing.md",
			fm + "A record that was READY before is NOT READY after it. `Not READY` in code says nothing.\n" +
				"**Open pull requests a crash left drifting** are listed; **Open, and not gating** too.\n" +
				"```\nNot READY: inside a fence.\n**Open:** inside a fence.\n```\n", nil, ""},
		{"a one-line comment passes", base + "/open/spc-7-thing.md",
			fm + "<!-- Done. Not READY: parked -->\n<!-- - **Open:** which cache? -->\n", nil, ""},
		{"text after a comment's closer is refused", base + "/open/spc-7-thing.md",
			fm + "<!--\nparked\n--> Status: Not READY\n", []int{12}, "Not READY"},
		{"an unclosed comment is refused on its opener", base + "/open/spc-7-thing.md",
			fm + "<!-- parked\nNot READY: hidden\n", []int{10}, "never closed"},
		{"an unclosed fence is refused on its opener", base + "/open/spc-7-thing.md",
			fm + "```\n## Open Questions\n- hidden?\n", []int{10}, "never closed"},
		{"a heading form is refused", base + "/open/spc-7-thing.md",
			fm + "## Not READY\n", []int{10}, "Not READY"},
		{"a quoted phrase is told to go in a code span", base + "/open/spc-7-thing.md",
			fm + "- NOT READY rows are listed last.\n", []int{10}, "code span"},
		{"a closed spec is out of reach", base + "/closed/spc-7-thing.md",
			fm + "Not READY: once.\n", nil, ""},
	}
	rule := RuleConfig{Enabled: true, Severity: severityBlocker, SpecsDir: "specs", IntentsDir: "intents"}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, c.rel, c.body)
			cfg := Config{Roots: []string{"rec"}, Rules: map[string]RuleConfig{ruleSpecUnsettled: rule}}
			fs, err := Lint(cfg, root)
			if err != nil {
				t.Fatal(err)
			}
			var got []Finding
			for _, f := range fs {
				if f.RuleID == ruleSpecUnsettled {
					got = append(got, f)
				}
			}
			if len(got) != len(c.lines) {
				t.Fatalf("want %d %s finding(s), got %+v", len(c.lines), ruleSpecUnsettled, fs)
			}
			for i, f := range got {
				if f.File != filepath.FromSlash(c.rel) || f.Line != c.lines[i] || f.Severity != severityBlocker {
					t.Errorf("finding %+v, want %s line %d at %s", f, c.rel, c.lines[i], severityBlocker)
				}
				for _, w := range []string{c.want, "## Open Questions"} {
					if !strings.Contains(f.Message, w) {
						t.Errorf("message = %q, want it to contain %q", f.Message, w)
					}
				}
			}
		})
	}
}

// The shipped config arms spec_unsettled as a blocker: the build reads the
// same statements as not READY, so a spec making one is refused where it is
// written, not only where it is built.
func TestSpecUnsettledArmedAsBlockerInRealConfig(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "..", ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	rc, ok := cfg.Rules[ruleSpecUnsettled]
	if !ok || !rc.Enabled || rc.Severity != severityBlocker || rc.SpecsDir == "" {
		t.Fatalf("record-lint.json must arm %s as a blocker with specs_dir: %+v", ruleSpecUnsettled, rc)
	}
}
