package lint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/record/findingkey"
)

// reciprocalFixture writes a corpus whose cross-document links exercise the
// three A1 rules (spc-2610021456200324 step 2): itd-1 carries one link of each
// kind that is reciprocated, one that is not, and one whose target the corpus
// does not hold.
func reciprocalFixture(t *testing.T) (string, Config) {
	t.Helper()
	root := t.TempDir()
	intent := func(bucket, id, extra, body string) {
		writeFile(t, root, "rec/intents/"+bucket+"/"+id+"-x.md",
			"---\nid: "+id+"\nslug: x\nkind: standalone\n"+extra+"---\n\n# "+id+"\n\n"+body+"\n")
	}
	spec := func(id, body string) {
		writeFile(t, root, "rec/specs/open/"+id+"-x.md", "---\nid: "+id+"\nslug: x\n---\n# "+id+"\n\n"+body+"\n")
	}
	issue := func(bucket, id, extra string) {
		writeFile(t, root, "work/issues/"+bucket+"/"+id+"-x.md", "---\nid: "+id+"\n"+extra+"---\nAn issue.\n")
	}

	intent("planned", "itd-1",
		"builds_on: [itd-2, itd-3, itd-9]\n"+
			"related_issues:\n  - iss-5\n  - iss-6\n  - iss-99\n"+
			"routed_from: [\"spc-7:A1\", \"spc-8:B2\", \"spc-9:C3\", \"spc-404:D4\"]\n",
		"The follower.")
	// itd-2 names its follower back; itd-3 names only itd-10 and xitd-1, which
	// are not itd-1 as a whole token.
	intent("shipped", "itd-2", "", "Built on by itd-1.")
	intent("shipped", "itd-3", "", "See itd-10 and xitd-1.")
	intent("shipped", "itd-10", "", "Unrelated.")
	// spc-7 routed item A1 and never names itd-1; spc-8 routed B2 and names it;
	// spc-9 carries no C3, so it is not shown to be the routing record.
	spec("spc-7", "Item A1: the follow-up.")
	spec("spc-8", "Item B2 went to itd-1.")
	spec("spc-9", "Nothing routed here.")
	// iss-5 lists itd-1 back; iss-6 (resolved) does not.
	issue("open", "iss-5", "related_intents: [itd-1]\n")
	issue("resolved", "iss-6", "related_intents: [itd-2]\n")

	cfg := Config{Rules: map[string]RuleConfig{
		ruleRecordSchema:           {Severity: severityBlocker, RecordStores: schemaStores()},
		"builds_on_reciprocal":     {Enabled: true, Severity: severityWarn},
		"routed_from_acknowledged": {Enabled: true, Severity: severityWarn},
		"related_issues_backlink":  {Enabled: true, Severity: severityWarn},
	}}
	return root, cfg
}

// keyed reports whether f carries the A1 key of rule over the two files, at
// the end of its message.
func keyed(f Finding, rule, a, b string) bool {
	k := findingkey.Derive(rule, findingkey.End{Path: a}, findingkey.End{Path: b})
	return strings.HasSuffix(f.Message, "(key "+k+")")
}

// TestReciprocalRulesNameBothFilesAndTheKey pins the three A1 rules: each
// fires once on its one-way link, on the file that declares the link, naming
// the other file and ending with the finding key; each is silent on the
// reciprocated link and on a target the corpus does not hold; and a
// routed_from whose source does not carry the item label is skipped.
func TestReciprocalRulesNameBothFilesAndTheKey(t *testing.T) {
	root, cfg := reciprocalFixture(t)
	src := "rec/intents/planned/itd-1-x.md"
	cases := []struct {
		rule, other, want string
	}{
		{"builds_on_reciprocal", "rec/intents/shipped/itd-3-x.md", "itd-3"},
		{"routed_from_acknowledged", "rec/specs/open/spc-7-x.md", "spc-7:A1"},
		{"related_issues_backlink", "work/issues/resolved/iss-6-x.md", "iss-6"},
	}
	for _, c := range cases {
		got := edgeFindings(t, root, cfg, c.rule)
		if len(got) != 1 {
			t.Errorf("%s: got %d findings, want 1: %+v", c.rule, len(got), got)
			continue
		}
		f := got[0]
		if f.File != src || f.Severity != severityWarn || f.Line < 1 {
			t.Errorf("%s: finding on %s:%d at %s, want %s at warn on the field's line", c.rule, f.File, f.Line, f.Severity, src)
		}
		if !strings.Contains(f.Message, c.other) || !strings.Contains(f.Message, c.want) {
			t.Errorf("%s: message does not name %s and %s: %s", c.rule, c.other, c.want, f.Message)
		}
		if !keyed(f, c.rule, src, c.other) {
			t.Errorf("%s: message does not end with the key over %s and %s: %s", c.rule, src, c.other, f.Message)
		}
	}
}

// TestReciprocalRulesLeaveUnresolvedTargetsToRecordSchema pins the split: a
// builds_on naming a record the corpus does not hold is record_schema's
// finding, and builds_on_reciprocal says nothing about it; a related_issues or
// routed_from target that does not resolve is likewise skipped.
func TestReciprocalRulesLeaveUnresolvedTargetsToRecordSchema(t *testing.T) {
	root, cfg := reciprocalFixture(t)
	rs := cfg.Rules[ruleRecordSchema]
	rs.Enabled = true
	cfg.Rules[ruleRecordSchema] = rs
	all, err := Lint(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	src := "rec/intents/planned/itd-1-x.md"
	if !findingWith(all, src, ruleRecordSchema, "builds_on names 'itd-9'") {
		t.Errorf("record_schema did not report the unresolved builds_on itd-9: %+v", all)
	}
	for _, f := range all {
		switch f.RuleID {
		case "builds_on_reciprocal", "routed_from_acknowledged", "related_issues_backlink":
			for _, gone := range []string{"itd-9", "iss-99", "spc-404", "spc-9"} {
				if strings.Contains(f.Message, gone) {
					t.Errorf("%s judged %s, which it must skip: %s", f.RuleID, gone, f.Message)
				}
			}
		case ruleRecordSchema:
			for _, field := range []string{"related_issues", "routed_from"} {
				if strings.HasPrefix(f.Message, field+" names") {
					t.Errorf("record_schema resolved %s, which it must leave alone: %s", field, f.Message)
				}
			}
		}
	}
}

// TestReciprocalRulesSilentWhenReciprocated pins the quiet case: a corpus whose
// every link is named back raises none of the three rules.
func TestReciprocalRulesSilentWhenReciprocated(t *testing.T) {
	root, cfg := reciprocalFixture(t)
	writeFile(t, root, "rec/intents/shipped/itd-3-x.md", "---\nid: itd-3\nslug: x\nkind: standalone\n---\n\n# itd-3\n\nFollowed by itd-1.\n")
	writeFile(t, root, "rec/specs/open/spc-7-x.md", "---\nid: spc-7\nslug: x\n---\n# spc-7\n\nItem A1 went to itd-1.\n")
	writeFile(t, root, "work/issues/resolved/iss-6-x.md", "---\nid: iss-6\nrelated_intents:\n  - itd-2\n  - itd-1\n---\nAn issue.\n")
	for _, rule := range []string{"builds_on_reciprocal", "routed_from_acknowledged", "related_issues_backlink"} {
		if got := edgeFindings(t, root, cfg, rule); len(got) != 0 {
			t.Errorf("%s fired on a reciprocated corpus: %+v", rule, got)
		}
	}
}

// TestReciprocalFindingCarriesItsKeyInJSON pins the key as a field too: the
// pre-commit gate reads a finding's key from --json output rather than parsing
// the message, and a finding of any other rule carries none.
func TestReciprocalFindingCarriesItsKeyInJSON(t *testing.T) {
	root, cfg := reciprocalFixture(t)
	got := edgeFindings(t, root, cfg, "related_issues_backlink")
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1: %+v", len(got), got)
	}
	want := findingkey.Derive("related_issues_backlink",
		findingkey.End{Path: "rec/intents/planned/itd-1-x.md"}, findingkey.End{Path: "work/issues/resolved/iss-6-x.md"})
	if got[0].Key != want {
		t.Errorf("Key = %q, want %q", got[0].Key, want)
	}
	b, err := json.Marshal(got[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"Key":"`+want+`"`) {
		t.Errorf("JSON does not carry the key: %s", b)
	}
	b, err = json.Marshal(Finding{File: "a.md", Line: 1, RuleID: "links_resolve", Severity: severityWarn, Message: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"Key"`) {
		t.Errorf("a finding with no key serialises one: %s", b)
	}
}

// TestBuildsOnExemptEdgesSilenceOnlyTheListedEdge pins the exemption list's
// reading: a listed edge is silent, an unlisted one still fires, and an entry
// that is not `<source> -> <target>` refuses the run rather than exempting
// nothing.
func TestBuildsOnExemptEdgesSilenceOnlyTheListedEdge(t *testing.T) {
	root, cfg := reciprocalFixture(t)
	writeFile(t, root, "rec/intents/shipped/itd-11-x.md", "---\nid: itd-11\nslug: x\nkind: standalone\nbuilds_on: [itd-10]\n---\n\n# itd-11\n")
	rc := cfg.Rules["builds_on_reciprocal"]
	rc.ExemptEdges = []string{"itd-1 -> itd-3"}
	cfg.Rules["builds_on_reciprocal"] = rc
	got := edgeFindings(t, root, cfg, "builds_on_reciprocal")
	if len(got) != 1 || got[0].File != "rec/intents/shipped/itd-11-x.md" {
		t.Errorf("want the one unlisted edge itd-11 -> itd-10, got %+v", got)
	}

	rc.ExemptEdges = []string{"itd-1 builds_on itd-3"}
	cfg.Rules["builds_on_reciprocal"] = rc
	if _, err := Lint(cfg, root); err == nil || !strings.Contains(err.Error(), "itd-1 builds_on itd-3") {
		t.Errorf("a malformed exempt_edges entry must refuse the run naming it, got %v", err)
	}
}

// TestA1RulesAreKnownAndArmedAtWarn pins the three A1 rules to the config
// loader and to this repository's record-lint configuration: a config naming
// them loads, and the committed one arms each at warn, so a tree that passed
// record-lint before they landed still passes.
func TestA1RulesAreKnownAndArmedAtWarn(t *testing.T) {
	rules := []string{"builds_on_reciprocal", "routed_from_acknowledged", "related_issues_backlink"}
	dir := t.TempDir()
	var body []string
	for _, r := range rules {
		body = append(body, `"`+r+`": {"enabled": true, "severity": "warn"}`)
	}
	writeFile(t, dir, "record-lint.json", `{"rules": {`+strings.Join(body, ", ")+`}}`)
	if _, err := LoadConfig(filepath.Join(dir, "record-lint.json")); err != nil {
		t.Fatalf("a config naming the A1 rules does not load: %v", err)
	}
	cfg, err := LoadConfig(filepath.Join("..", "..", "..", ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rules {
		rc, ok := cfg.Rules[r]
		if !ok || !rc.Enabled || rc.Severity != severityWarn {
			t.Errorf(".abcd/record-lint.json must arm %s at warn; got present=%v enabled=%v severity=%q", r, ok, rc.Enabled, rc.Severity)
		}
	}
}

// TestBuildsOnExemptionsNeverGrow pins builds_on_reciprocal's exemption list
// in the committed configuration: every entry is one it landed with
// (testdata/builds-on-exempt-at-landing.txt, the 155 one-way edges the tree
// then held), so the list may shrink but never gains one; it is sorted (by source, then target, in handle order) without
// duplicates, and every entry still names a builds_on edge in this tree, so an
// entry whose edge was dropped is dropped from the list too.
func TestBuildsOnExemptionsNeverGrow(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	cfg, err := LoadConfig(filepath.Join(root, ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	list := cfg.Rules["builds_on_reciprocal"].ExemptEdges
	landing, err := os.ReadFile(filepath.Join("testdata", "builds-on-exempt-at-landing.txt"))
	if err != nil {
		t.Fatal(err)
	}
	// The landing file is itself pinned, by its entry count and its digest,
	// so the list cannot grow by editing the testdata beside it: growth means
	// editing these two constants in this test, in a diff a reviewer reads.
	const landingEntries = 155
	const landingSHA256 = "d708b3cd7c70f5e66b62aaf839692e9927d2325005299b1f7712d80dd1fd3056"
	if sum := sha256.Sum256(landing); hex.EncodeToString(sum[:]) != landingSHA256 {
		t.Errorf("testdata/builds-on-exempt-at-landing.txt has sha256 %x, want %s: the landing set is frozen; "+
			"a new one-way builds_on is answered in its target's file, not exempted", sum, landingSHA256)
	}
	entries := strings.Split(strings.TrimSpace(string(landing)), "\n")
	if len(entries) != landingEntries {
		t.Errorf("testdata/builds-on-exempt-at-landing.txt holds %d entries, want exactly %d", len(entries), landingEntries)
	}
	landed := map[string]bool{}
	for _, e := range entries {
		landed[e] = true
	}
	for _, e := range list {
		if !landed[e] {
			t.Errorf("exempt_edges entry %q is not one the list landed with: the list never grows; "+
				"a new one-way builds_on is answered in its target's file, not exempted", e)
		}
	}
	split := func(e string) (string, string) {
		from, to, _ := strings.Cut(e, " -> ")
		return from, to
	}
	for i := 1; i < len(list); i++ {
		af, at := split(list[i-1])
		bf, bt := split(list[i])
		if !(HandleLess(af, bf) || (af == bf && HandleLess(at, bt))) {
			t.Errorf("exempt_edges is not sorted without duplicates at %q, %q", list[i-1], list[i])
		}
	}
	g, err := LoadRecordGraph(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	edges := map[string]bool{}
	for _, e := range g.Edges {
		if e.Field == "builds_on" {
			edges[e.From+" -> "+e.To] = true
		}
	}
	for _, e := range list {
		if !edges[e] {
			t.Errorf("exempt_edges entry %q names no builds_on edge in this tree: drop it", e)
		}
	}
}
