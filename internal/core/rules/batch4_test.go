package rules

import (
	"strings"
	"testing"
)

// TestValidateRejectsEmptyRuleBody (iss-2608261550497978) pins that an empty or
// whitespace-only rule string is refused at validation. Left unchecked it passes
// Validate (which only inspects domain names and states) and renders as a bare
// contentless "- " bullet in the injected block.
func TestValidateRejectsEmptyRuleBody(t *testing.T) {
	for _, body := range []string{"", "   ", "\t", "\n", " \t\n "} {
		rs := RuleSet{SchemaVersion: 1, Domains: map[string]Domain{
			"X": {State: StateActive, Rules: []string{"a real rule", body}},
		}}
		if err := Validate(rs); err == nil {
			t.Fatalf("empty/whitespace-only rule %q must be refused by Validate", body)
		}
	}
	// A non-empty rule set still validates.
	ok := RuleSet{SchemaVersion: 1, Domains: map[string]Domain{
		"X": {State: StateActive, Rules: []string{"a real rule"}},
	}}
	if err := Validate(ok); err != nil {
		t.Fatalf("valid rule set rejected: %v", err)
	}
}

// TestRenderNeverEmitsBareBullet (iss-2608261550497978) confirms renderDomain
// emits no contentless "- " bullet even if handed a whitespace-only body
// directly (the defensive second line behind Validate's loud refusal).
func TestRenderNeverEmitsBareBullet(t *testing.T) {
	d := ResolvedDomain{Name: "X", Domain: Domain{Rules: []string{"keep me", "   ", "\t\t"}}}
	out := renderDomain(d)
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimRight(line, " \t") == "-" {
			t.Fatalf("renderDomain emitted a contentless bullet:\n%q", out)
		}
	}
	if !strings.Contains(out, "- keep me") {
		t.Fatalf("renderDomain dropped a real rule:\n%q", out)
	}
}

// TestLoadRefusesDuplicateDomainKey (iss-2608261550498779) pins that duplicate
// domain keys in .abcd/rules.json are refused loudly rather than silently
// resolving last-wins (the JSON decoder keeps the final block).
func TestLoadRefusesDuplicateDomainKey(t *testing.T) {
	dir := t.TempDir()
	writeRepoRules(t, dir, `{"schema_version":1,"domains":{"ROADMAP":{"state":"dormant"},"ROADMAP":{"state":"active"}}}`)
	if _, err := Load(dir); err == nil {
		t.Fatal("duplicate domain key must fail closed, not resolve last-wins")
	}
}

// TestLoadRefusesDuplicateKeyAnyLevel (iss-2608261550498779) proves the scan is
// token-level, catching a repeated key inside a domain object too, not only at
// the domains map.
func TestLoadRefusesDuplicateKeyAnyLevel(t *testing.T) {
	dir := t.TempDir()
	writeRepoRules(t, dir, `{"schema_version":1,"domains":{"CUSTOM":{"state":"active","state":"dormant","recall":["x"],"rules":["r"]}}}`)
	if _, err := Load(dir); err == nil {
		t.Fatal("duplicate nested key must fail closed")
	}
}

// TestLoadRefusesACaseTwinOfAKey (iss-2608261550498779, iss-2609252251311346):
// encoding/json binds "Disabled" to the disabled field case-insensitively and
// keeps the last, so a second spelling low in the file would flip the kill switch
// a reader saw set false. A domains map naming PII and pii is refused the same
// way: one domain, two spellings, is illegible.
func TestLoadRefusesACaseTwinOfAKey(t *testing.T) {
	for _, body := range []string{
		`{"schema_version":1,"disabled":false,"Disabled":true,"domains":{}}`,
		`{"schema_version":1,"domains":{"PII":{"state":"active"},"pii":{"state":"dormant"}}}`,
		`{"schema_version":1,"domains":{"CUSTOM":{"state":"active","State":"dormant","recall":["x"],"rules":["r"]}}}`,
	} {
		dir := t.TempDir()
		writeRepoRules(t, dir, body)
		if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "duplicate key") {
			t.Fatalf("%s: a case twin must fail closed as a duplicate key, got %v", body, err)
		}
	}
}

// TestLoadAcceptsDistinctKeys (iss-2608261550498779) guards against a
// false-positive: a well-formed file with distinct keys still loads.
func TestLoadAcceptsDistinctKeys(t *testing.T) {
	dir := t.TempDir()
	writeRepoRules(t, dir, `{"schema_version":1,"domains":{"ROADMAP":{"state":"dormant"},"ISSUES":{"state":"active"}}}`)
	if _, err := Load(dir); err != nil {
		t.Fatalf("distinct domain keys must load: %v", err)
	}
}

// TestInjectEnforcesBudgetWithLoudNotice (iss-2608261551077971) pins the
// per-repo injection budget: a rendered set that overflows the budget is
// truncated with an unmistakable notice, and the emitted text stays within the
// budget. Without the budget the whole ~oversize set lands in context every
// refresh.
func TestInjectEnforcesBudgetWithLoudNotice(t *testing.T) {
	// Eight domains, each rendering to ~12 KiB, all matching one keyword —
	// ~96 KiB total, comfortably over the 64 KiB budget.
	big := strings.Repeat("x", 12*1024)
	domains := map[string]Domain{}
	for _, n := range []string{"AA", "BB", "CC", "DD", "EE", "FF", "GG", "HH"} {
		domains[n] = Domain{State: StateActive, Recall: []string{"flood"}, Rules: []string{big}}
	}
	rs := RuleSet{SchemaVersion: 1, Domains: domains}
	res := Inject(rs, "a flood of rules", SessionState{}, 0)
	if len(res.Text) > injectionBudgetBytes {
		t.Fatalf("injected text %d bytes exceeds budget %d", len(res.Text), injectionBudgetBytes)
	}
	if !strings.Contains(res.Text, injectionTruncatedMarker) {
		t.Fatalf("over-budget injection lacks the loud truncation notice:\n%.200s", res.Text)
	}
}

// TestInjectNormalSetUntouched (iss-2608261551077971) proves a legitimate
// multi-domain match is byte-identical to the plain render — the budget only
// bites on a flood, never on ordinary rule sets.
func TestInjectNormalSetUntouched(t *testing.T) {
	rs := Defaults()
	res := Inject(rs, "commit and push and open the pull request", SessionState{}, 0)
	if strings.Contains(res.Text, injectionTruncatedMarker) {
		t.Fatalf("normal set wrongly truncated:\n%s", res.Text)
	}
	if res.Text != Render(rs.Match("commit and push and open the pull request")) {
		t.Fatal("normal-path injection is not byte-identical to the plain render")
	}
}
