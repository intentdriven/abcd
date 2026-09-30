package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
)

// edgesFixture writes an intent store whose dependency edges exercise both edge
// rules: planned and draft intents naming superseded records (followed along a
// two-step chain, and to a record that names no successor), a shipped intent
// naming one (out of scope), and a three-record builds_on/blocked_by cycle.
func edgesFixture(t *testing.T) (string, Config) {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	intent := func(bucket, id, extra string) {
		write("rec/intents/"+bucket+"/"+id+"-x.md", "---\nid: "+id+"\nslug: x\nkind: standalone\n"+extra+"---\n\n# "+id+"\n")
	}
	intent("superseded", "itd-2", "superseded_by: itd-3\n")
	intent("superseded", "itd-3", "superseded_by: itd-4\n")
	intent("planned", "itd-4", "")
	intent("drafts", "itd-5", "blocked_by: [itd-2]\n")
	intent("planned", "itd-6", "builds_on:\n  - itd-3\n")
	intent("shipped", "itd-7", "builds_on: [itd-2]\n")
	intent("planned", "itd-8", "builds_on: [itd-4]\n")
	intent("superseded", "itd-9", "superseded_by: null\n")
	intent("drafts", "itd-10", "builds_on: [itd-9]\n")
	intent("planned", "itd-11", "builds_on: [itd-12]\n")
	intent("drafts", "itd-12", "blocked_by: [itd-13]\n")
	intent("drafts", "itd-13", "builds_on: [itd-11, itd-4]\n")

	cfg := Config{Rules: map[string]RuleConfig{
		ruleRecordSchema: {
			Severity:     severityBlocker,
			RecordStores: map[string]string{"itd": "rec/intents"},
		},
		ruleStaleEdge: {Enabled: true, Severity: severityWarn},
		ruleEdgeCycle: {Enabled: true, Severity: severityWarn},
	}}
	return root, cfg
}

// withChainReader registers the intent package's chain reader for one test,
// as the front doors do, and restores what was registered before.
func withChainReader(t *testing.T, fn func(string, []SupersessionLink, string) ([]string, string, string, string, error)) {
	t.Helper()
	prev := supersessionChain
	SetSupersessionChain(fn)
	t.Cleanup(func() { supersessionChain = prev })
}

func edgeFindings(t *testing.T, root string, cfg Config, rule string) []Finding {
	t.Helper()
	all, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	var out []Finding
	for _, f := range all {
		if f.RuleID == rule {
			out = append(out, f)
		}
	}
	return out
}

// TestStaleEdgeNamesTheLiveSuccessor pins stale_edge: a planned or draft intent
// whose builds_on or blocked_by names a superseded record is reported once per
// edge, at the configured severity, naming the record its supersession chain
// ends at; a chain that ends nowhere says so; a shipped intent and an edge to a
// live record are not reported.
func TestStaleEdgeNamesTheLiveSuccessor(t *testing.T) {
	withChainReader(t, intent.SupersessionChainOf)
	root, cfg := edgesFixture(t)
	got := edgeFindings(t, root, cfg, ruleStaleEdge)
	want := map[string][]string{
		"rec/intents/drafts/itd-5-x.md":  {"blocked_by names itd-2", "itd-2 → itd-3 → itd-4", "itd-4 (planned)"},
		"rec/intents/planned/itd-6-x.md": {"builds_on names itd-3", "itd-3 → itd-4", "itd-4 (planned)"},
		"rec/intents/drafts/itd-10-x.md": {"builds_on names itd-9", "names no successor"},
	}
	if len(got) != len(want) {
		t.Fatalf("stale_edge: got %d findings, want %d: %+v", len(got), len(want), got)
	}
	for _, f := range got {
		parts, ok := want[filepath.ToSlash(f.File)]
		if !ok {
			t.Errorf("unexpected stale_edge finding: %+v", f)
			continue
		}
		if f.Severity != severityWarn {
			t.Errorf("%s: severity %q, want warn", f.File, f.Severity)
		}
		if f.Line < 2 {
			t.Errorf("%s: line %d, want the edge field's line", f.File, f.Line)
		}
		for _, p := range parts {
			if !strings.Contains(f.Message, p) {
				t.Errorf("%s: message %q lacks %q", f.File, f.Message, p)
			}
		}
	}
}

// TestStaleEdgeSaysASettledChainIsSettled pins the settled ends of a
// supersession chain (rulings CF1 and CF2 of 2026-09-30): a chain ending at an
// accepted decision, or at an intent kept as a discipline, has no live record
// to repoint at, so the finding says which record settled it and to drop the
// edge, rather than inviting a repoint.
func TestStaleEdgeSaysASettledChainIsSettled(t *testing.T) {
	withChainReader(t, intent.SupersessionChainOf)
	root, cfg := edgesFixture(t)
	write := func(rel, body string) {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".abcd/development/decisions/adrs/0037-settled.md", "---\nid: adr-37\nslug: settled\nstatus: accepted\n---\n\n# ADR-37\n")
	write("rec/intents/superseded/itd-20-x.md", "---\nid: itd-20\nslug: x\nkind: standalone\nsuperseded_by: adr-37\n---\n\n# itd-20\n")
	write("rec/intents/superseded/itd-21-x.md", "---\nid: itd-21\nslug: x\nkind: standalone\nsuperseded_by: itd-22\n---\n\n# itd-21\n")
	write("rec/intents/disciplines/itd-22-x.md", "---\nid: itd-22\nslug: x\nkind: discipline\n---\n\n# itd-22\n")
	write("rec/intents/drafts/itd-23-x.md", "---\nid: itd-23\nslug: x\nkind: standalone\nbuilds_on: [itd-20]\nblocked_by: [itd-21]\n---\n\n# itd-23\n")
	var seen []string
	for _, f := range edgeFindings(t, root, cfg, ruleStaleEdge) {
		if filepath.ToSlash(f.File) != "rec/intents/drafts/itd-23-x.md" {
			continue
		}
		seen = append(seen, f.Message)
		if strings.Contains(f.Message, "repoint") {
			t.Errorf("a settled chain has no live record to repoint at: %q", f.Message)
		}
	}
	for _, want := range []string{"is settled by adr-37 (accepted): drop the edge", "is settled by itd-22 (disciplines): drop the edge"} {
		found := false
		for _, m := range seen {
			found = found || strings.Contains(m, want)
		}
		if !found {
			t.Errorf("no finding on itd-23 says %q: %q", want, seen)
		}
	}
}

// TestEdgeCycleNamesEveryRecord pins edge_cycle: a cycle through builds_on and
// blocked_by together is one finding that names every record on it, and an
// acyclic edge set yields none.
func TestEdgeCycleNamesEveryRecord(t *testing.T) {
	root, cfg := edgesFixture(t)
	got := edgeFindings(t, root, cfg, ruleEdgeCycle)
	if len(got) != 1 {
		t.Fatalf("edge_cycle: got %d findings, want 1: %+v", len(got), got)
	}
	f := got[0]
	if f.Severity != severityWarn {
		t.Errorf("severity %q, want warn", f.Severity)
	}
	if filepath.ToSlash(f.File) != "rec/intents/planned/itd-11-x.md" {
		t.Errorf("file %q, want the cycle's first record", f.File)
	}
	for _, id := range []string{"itd-11", "itd-12", "itd-13"} {
		if !strings.Contains(f.Message, id) {
			t.Errorf("message %q does not name %s", f.Message, id)
		}
	}
	if strings.Contains(f.Message, "itd-4") {
		t.Errorf("message %q names itd-4, which is not on the cycle", f.Message)
	}
}

// TestEdgeRulesSilentWhenDisabled pins that neither rule reports when the
// configuration does not arm it.
func TestEdgeRulesSilentWhenDisabled(t *testing.T) {
	withChainReader(t, intent.SupersessionChainOf)
	root, cfg := edgesFixture(t)
	cfg.Rules[ruleStaleEdge] = RuleConfig{Severity: severityWarn}
	cfg.Rules[ruleEdgeCycle] = RuleConfig{Severity: severityWarn}
	for _, rule := range []string{ruleStaleEdge, ruleEdgeCycle} {
		if got := edgeFindings(t, root, cfg, rule); len(got) != 0 {
			t.Errorf("%s disabled: got %+v", rule, got)
		}
	}
}

// TestStaleEdgeSaysWhenNoChainReaderIsRegistered pins the seam's failure mode: a
// front door that arms stale_edge without registering the chain reader gets one
// finding naming the omission, never a silent pass.
func TestStaleEdgeSaysWhenNoChainReaderIsRegistered(t *testing.T) {
	withChainReader(t, nil)
	root, cfg := edgesFixture(t)
	got := edgeFindings(t, root, cfg, ruleStaleEdge)
	if len(got) != 1 || !strings.Contains(got[0].Message, "SetSupersessionChain") {
		t.Fatalf("unregistered chain reader: got %+v, want one finding naming lint.SetSupersessionChain", got)
	}
}
