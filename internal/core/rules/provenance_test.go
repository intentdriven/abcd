package rules

import (
	"strings"
	"testing"
)

// GHSA-22f8-qf5r-gjgq: a repo override replaces a bundled domain's words, and
// nothing downstream could tell whose words they were. Merge is the one place
// the origin is still known, so it records it, and every accessor carries it.
func TestMergeRecordsRepoOrigin(t *testing.T) {
	over := RuleSet{SchemaVersion: 1, Domains: map[string]Domain{
		"PII":     {Rules: []string{"x"}},                        // rules replaced
		"ROADMAP": {State: StateDormant},                         // state-only: conservative, still repo-chosen
		"CUSTOM":  {Recall: []string{"k"}, Rules: []string{"y"}}, // new key
	}}
	rs := Merge(Defaults(), over)
	for name, want := range map[string]string{
		"PII": SourceRepo, "ROADMAP": SourceRepo, "CUSTOM": SourceRepo, "COMMITTING": SourceBundled,
	} {
		d, ok := rs.Lookup(name)
		if !ok {
			t.Fatalf("Lookup(%q) missing", name)
		}
		if d.Source != want {
			t.Errorf("Lookup(%q).Source = %q, want %q", name, d.Source, want)
		}
	}
	for _, d := range rs.Active() {
		if d.Source == "" {
			t.Errorf("Active() left %s without a source", d.Name)
		}
		if d.Name == "PII" && d.Source != SourceRepo {
			t.Errorf("Active() PII source = %q, want %q", d.Source, SourceRepo)
		}
	}
	matched := rs.Match("redact the api key in this token")
	if len(matched) == 0 || matched[0].Name != "PII" || matched[0].Source != SourceRepo {
		t.Fatalf("Match did not carry the repo origin: %+v", matched)
	}
	// A clone keeps the origins; the defaults alone are bundled everywhere.
	if d, _ := cloneRuleSet(rs).Lookup("PII"); d.Source != SourceRepo {
		t.Errorf("cloneRuleSet dropped the origin: %q", d.Source)
	}
	for _, d := range Defaults().Active() {
		if d.Source != SourceBundled {
			t.Errorf("Defaults() %s source = %q, want %q", d.Name, d.Source, SourceBundled)
		}
	}
}

// The marker is part of the rendered block — in the agent's context, in
// `abcd rules`, and therefore inside the dedup unit Signature hashes.
func TestRenderMarksRepoOverride(t *testing.T) {
	repo := ResolvedDomain{Name: "PII", Source: SourceRepo, Domain: Domain{Rules: []string{"x"}}}
	bundled := ResolvedDomain{Name: "PII", Source: SourceBundled, Domain: Domain{Rules: []string{"x"}}}
	unset := ResolvedDomain{Name: "PII", Domain: Domain{Rules: []string{"x"}}}
	if got, want := renderDomain(repo), "## PII (repo override)\n- x\n"; got != want {
		t.Fatalf("repo override unmarked:\n got %q\nwant %q", got, want)
	}
	if got, want := renderDomain(bundled), "## PII\n- x\n"; got != want {
		t.Fatalf("bundled domain marked:\n got %q\nwant %q", got, want)
	}
	if got, want := renderDomain(unset), "## PII\n- x\n"; got != want {
		t.Fatalf("an unlabelled domain must render as bundled:\n got %q\nwant %q", got, want)
	}
	if Signature(repo) == Signature(bundled) {
		t.Fatal("the marker must sit inside the dedup unit (signatures equal)")
	}
	if out := Render([]ResolvedDomain{repo}); !strings.Contains(out, "\n## PII (repo override)\n") {
		t.Fatalf("Render lost the marker:\n%s", out)
	}
}

// The hook's out-of-band diagnostic prints names; Injected []string cannot say
// whose words they were, so the result carries the source per name and the
// label the diagnostic prints.
func TestInjectLabelsRepoOverrides(t *testing.T) {
	rs := Merge(Defaults(), RuleSet{SchemaVersion: 1, Domains: map[string]Domain{
		"PII": {Rules: []string{"x"}},
	}})
	res := Inject(rs, "redact the api key in this token", SessionState{}, 0)
	if res.Sources["PII"] != SourceRepo {
		t.Fatalf("InjectResult.Sources[PII] = %q, want %q (%+v)", res.Sources["PII"], SourceRepo, res)
	}
	if got := res.Labels(); len(got) != 1 || got[0] != "PII (repo override)" {
		t.Fatalf("Labels() = %v, want [PII (repo override)]", got)
	}
	plain := Inject(Defaults(), "commit and push", SessionState{}, 0)
	if got := plain.Labels(); len(got) != 1 || got[0] != "COMMITTING" {
		t.Fatalf("Labels() for a bundled domain = %v, want [COMMITTING]", got)
	}
}
