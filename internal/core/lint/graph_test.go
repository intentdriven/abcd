package lint

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// graphFixture writes a miniature four-store record and returns the config that
// points at it, so the graph export is exercised without the repo's own corpus.
func graphFixture(t *testing.T) (string, Config) {
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

	write("rec/adrs/0001-first.md", `---
id: adr-1
slug: first
status: accepted
date: 2026-01-02
supersedes: null
superseded_by: null
related_intents: [itd-1]
related_adrs: []
---

# ADR-1: The first decision
`)
	write("rec/adrs/0002-second.md", `---
id: adr-2
slug: second
status: accepted
date: 2026-01-03
supersedes:
- adr-9
superseded_by: null
related_adrs: [adr-1]
---

# ADR-2: The second decision
`)
	write("rec/intents/shipped/itd-1-a-shipped-intent.md", `---
id: itd-1
slug: a-shipped-intent
spec_id: spc-1
kind: standalone
severity: minor
impact: additive
---

# A shipped intent
`)
	write("rec/specs/open/spc-1-a-spec.md", `---
id: spc-1
slug: a-spec
intent: itd-1
---
# A spec
`)
	write("work/issues/open/iss-1-an-issue.md", `---
schema_version: 1
id: "iss-1"
slug: "an-issue"
severity: "minor"
category: "tech-debt"
---

an issue with no heading at all
`)

	cfg := Config{Rules: map[string]RuleConfig{
		ruleRecordSchema: {
			Enabled:  true,
			Severity: severityBlocker,
			RecordStores: map[string]string{
				"adr": "rec/adrs",
				"itd": "rec/intents",
				"spc": "rec/specs",
				"iss": "work/issues",
			},
		},
	}}
	return root, cfg
}

func TestLoadRecordGraphNodes(t *testing.T) {
	root, cfg := graphFixture(t)
	g, err := LoadRecordGraph(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Nodes) != 5 {
		t.Fatalf("nodes: got %d, want 5: %+v", len(g.Nodes), g.Nodes)
	}
	byID := map[string]RecordNode{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}
	adr1, ok := byID["adr-1"]
	if !ok {
		t.Fatal("adr-1 missing")
	}
	if adr1.Type != "adr" || adr1.Lifecycle != "" {
		t.Errorf("adr-1 type/lifecycle: %q/%q", adr1.Type, adr1.Lifecycle)
	}
	if adr1.Title != "ADR-1: The first decision" {
		t.Errorf("adr-1 title: %q", adr1.Title)
	}
	if adr1.Date != "2026-01-02" || adr1.Status != "accepted" {
		t.Errorf("adr-1 date/status: %q/%q", adr1.Date, adr1.Status)
	}
	if adr1.Path != "rec/adrs/0001-first.md" {
		t.Errorf("adr-1 path: %q", adr1.Path)
	}
	itd1 := byID["itd-1"]
	if itd1.Type != "intent" || itd1.Lifecycle != "shipped" || itd1.Kind != "standalone" || itd1.Severity != "minor" {
		t.Errorf("itd-1: %+v", itd1)
	}
	iss1 := byID["iss-1"]
	if iss1.Type != "issue" || iss1.Lifecycle != "open" {
		t.Errorf("iss-1: %+v", iss1)
	}
	// An issue carries no H1; its title is the first body line, as the ledger's
	// own promote path derives it.
	if iss1.Title != "an issue with no heading at all" {
		t.Errorf("iss-1 title: %q", iss1.Title)
	}
	spc1 := byID["spc-1"]
	if spc1.Type != "spec" || spc1.Lifecycle != "open" {
		t.Errorf("spc-1: %+v", spc1)
	}
	// Nodes are sorted by id so the export is deterministic.
	for i := 1; i < len(g.Nodes); i++ {
		if g.Nodes[i-1].ID >= g.Nodes[i].ID {
			t.Fatalf("nodes not sorted by id: %q then %q", g.Nodes[i-1].ID, g.Nodes[i].ID)
		}
	}
}

func TestLoadRecordGraphEdgesAndDangling(t *testing.T) {
	root, cfg := graphFixture(t)
	g, err := LoadRecordGraph(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[RecordEdge]bool{
		{From: "adr-1", To: "itd-1", Field: "related_intents"}: false,
		{From: "adr-2", To: "adr-1", Field: "related_adrs"}:    false,
		{From: "itd-1", To: "spc-1", Field: "spec_id"}:         false,
		{From: "spc-1", To: "itd-1", Field: "intent"}:          false,
	}
	for _, e := range g.Edges {
		if _, ok := want[e]; !ok {
			t.Errorf("unexpected edge %+v", e)
			continue
		}
		want[e] = true
	}
	for e, seen := range want {
		if !seen {
			t.Errorf("missing edge %+v", e)
		}
	}
	// The block-sequence `supersedes: - adr-9` names a record no file carries.
	if len(g.Dangling) != 1 {
		t.Fatalf("dangling: %+v", g.Dangling)
	}
	if g.Dangling[0] != (RecordEdge{From: "adr-2", To: "adr-9", Field: "supersedes"}) {
		t.Errorf("dangling: %+v", g.Dangling[0])
	}
	// adr-9 is above the ADR store's high-water mark (adr-2), so the record
	// cannot have pruned it and it is NOT accounted for as retired.
	if len(g.Retired) != 0 {
		t.Errorf("retired: %+v", g.Retired)
	}
}

// TestLoadRecordGraphRetiredIsBounded pins the retirement read: a pruned id below
// the store's high-water mark is accounted for; one above it is not.
func TestLoadRecordGraphRetiredIsBounded(t *testing.T) {
	root, cfg := graphFixture(t)
	abs := filepath.Join(root, "rec", "adrs", "0003-third.md")
	body := `---
id: adr-3
slug: third
status: accepted
date: 2026-01-04
supersedes: [adr-2]
superseded_by: null
related_adrs: []
---

# ADR-3: The third decision
`
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// adr-2 still has a file, so it is present rather than retired; retire it by
	// removing the file the way the ADR lifecycle prunes one.
	if err := os.Remove(filepath.Join(root, "rec", "adrs", "0002-second.md")); err != nil {
		t.Fatal(err)
	}
	g, err := LoadRecordGraph(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Retired) != 1 || g.Retired[0] != "adr-2" {
		t.Fatalf("retired: %+v", g.Retired)
	}
}

// TestLoadRecordGraphOverThisRepo pins the export against the real corpus: the
// graph is the record-lint scan's own view, so every id it names must resolve.
func TestLoadRecordGraphOverThisRepo(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	cfg, err := LoadConfig(filepath.Join(repoRoot, ".abcd", "record-lint.json"))
	if err != nil {
		t.Fatal(err)
	}
	g, err := LoadRecordGraph(cfg, repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Nodes) < 100 {
		t.Fatalf("nodes: got %d, want the repo's corpus", len(g.Nodes))
	}
	ids := map[string]bool{}
	for _, n := range g.Nodes {
		ids[n.ID] = true
	}
	for _, e := range g.Edges {
		if !ids[e.From] || !ids[e.To] {
			t.Errorf("resolved edge names an absent record: %+v", e)
		}
	}
	for _, e := range g.Dangling {
		if ids[e.To] {
			t.Errorf("dangling edge names a present record: %+v", e)
		}
	}
}

// TestHandleLessOrdersOrdinalsBeforeStamps pins the index sort the ADR store's
// two id vintages need (the 2026-09-01 ruling): ascending, the hand-numbered
// ordinals come first and the minted timestamps follow, because the comparison
// is on the number and a 16-digit stamp is larger than every ordinal ever
// issued. Every derived index — the record export's edges, the site's decision
// lists — orders through this one comparator.
func TestHandleLessOrdersOrdinalsBeforeStamps(t *testing.T) {
	got := []string{"adr-2609012206053814", "adr-58", "adr-1", "adr-2609012206050042", "adr-12"}
	sort.SliceStable(got, func(i, j int) bool { return HandleLess(got[i], got[j]) })
	want := []string{"adr-1", "adr-12", "adr-58", "adr-2609012206050042", "adr-2609012206053814"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("HandleLess order:\n got %v\nwant %v", got, want)
	}
}
