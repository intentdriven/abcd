package lint

import (
	"path/filepath"
	"strings"
	"testing"
)

// spec_steps_serial is dcn-2610230857411093's record-lint half: an open spec of
// two or more steps where no step declares `needs:` runs fully serial, each step
// waiting for every earlier one (ruling DR6b), and is named at the configured
// severity. It reads the steps through spec.SerialByDefault, the judgement the
// readiness gate's steps row makes, so the two never disagree. A closed spec is
// never built again and is out of its reach.
func TestSpecStepsSerial(t *testing.T) {
	const base = "rec/specs"
	const fm = "---\nid: spc-7\nslug: thing\nintent: itd-7\n---\n# T\n\n## Summary\n\nWritten.\n"
	cases := []struct {
		name string
		rel  string
		body string
		want bool
	}{
		{"two steps, no needs, is flagged", base + "/open/spc-7-thing.md",
			fm + "\n## Steps\n\n1. The parser\n   - packages: internal/core/spec\n2. The loop\n", true},
		{"one step is clean", base + "/open/spc-7-thing.md",
			fm + "\n## Steps\n\n1. The parser\n", false},
		{"no steps section is clean", base + "/open/spc-7-thing.md", fm, false},
		{"a step declaring none is clean", base + "/open/spc-7-thing.md",
			fm + "\n## Steps\n\n1. A\n2. B\n   - needs: none\n", false},
		{"a step declaring an earlier step is clean", base + "/open/spc-7-thing.md",
			fm + "\n## Steps\n\n1. A\n2. B\n3. C\n   - needs: 1\n", false},
		{"only step 1 declaring none is still flagged", base + "/open/spc-7-thing.md",
			fm + "\n## Steps\n\n1. A\n   - needs: none\n2. B\n3. C\n", true},
		{"one step left to build is clean", base + "/open/spc-7-thing.md",
			fm + "\n## Steps\n\n1. A\n   - landed: #1\n2. B\n", false},
		{"a closed spec is out of reach", base + "/closed/spc-7-thing.md",
			fm + "\n## Steps\n\n1. A\n2. B\n", false},
		// A section the parser refuses is the readiness gate's steps row to name,
		// with its shape; this rule judges needs, not shape, and says nothing.
		{"an unreadable steps section is left to the readiness gate", base + "/open/spc-7-thing.md",
			fm + "\n## Steps\n\nFirst the parser, then the loop.\n", false},
	}
	rule := RuleConfig{Enabled: true, Severity: severityWarn, SpecsDir: "specs", IntentsDir: "intents"}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, c.rel, c.body)
			cfg := Config{Roots: []string{"rec"}, Rules: map[string]RuleConfig{ruleSpecStepsSerial: rule}}
			fs, err := Lint(cfg, root)
			if err != nil {
				t.Fatal(err)
			}
			n := countRule(fs, ruleSpecStepsSerial)
			if !c.want {
				if n != 0 {
					t.Fatalf("want no %s finding, got %+v", ruleSpecStepsSerial, fs)
				}
				return
			}
			if n != 1 {
				t.Fatalf("want exactly one %s finding, got %d: %+v", ruleSpecStepsSerial, n, fs)
			}
			f := fs[0]
			if f.File != filepath.FromSlash(c.rel) {
				t.Errorf("finding on %s, want %s", f.File, c.rel)
			}
			if f.Severity != severityWarn {
				t.Errorf("severity = %q, want the configured %q", f.Severity, severityWarn)
			}
			for _, w := range []string{"fully serial", "every earlier step", "- needs: none"} {
				if !strings.Contains(f.Message, w) {
					t.Errorf("message = %q, want it to contain %q", f.Message, w)
				}
			}
		})
	}
}

// The shipped config arms spec_steps_serial at warn, never blocker: the ruling
// is a warning, never a refusal, and the open specs that predate it are named on
// every run without blocking one.
func TestSpecStepsSerialArmedAtWarnInRealConfig(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join("..", "..", "..", ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	rc, ok := cfg.Rules[ruleSpecStepsSerial]
	if !ok || !rc.Enabled {
		t.Fatalf("record-lint.json must enable %s", ruleSpecStepsSerial)
	}
	if rc.Severity != severityWarn {
		t.Errorf("%s severity = %q, want %q (a warning, never a refusal)", ruleSpecStepsSerial, rc.Severity, severityWarn)
	}
	if rc.SpecsDir == "" {
		t.Errorf("%s must declare specs_dir, like spec_lifecycle", ruleSpecStepsSerial)
	}
}
