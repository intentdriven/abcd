package lab

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func prob(p float64) *float64 { return &p }

// TestRecordPairWritesUnderTheNewestLabHomeAndNothingInTheRepo is the
// decision adapter's first criterion, the lab's half (itd-2609221009495079):
// a shadow pair is appended to pairs.jsonl in the lab home, in the
// machine-scoped store, and nothing is written into the repository.
func TestRecordPairWritesUnderTheNewestLabHomeAndNothingInTheRepo(t *testing.T) {
	r, home := fixture(t)
	notAbcd(t)
	m := mint(t, r)
	before := r.Git("status", "--porcelain", "--ignored")

	p := Pair{Type: "duplicate-match", Kind: "choose", StateSHA256: strings.Repeat("a", 64), StateTokens: 12,
		Host: "iss-1", Adapter: "iss-1", Probability: prob(0.92), Target: "openrouter/typesafe/jev-1.13"}
	id, where, err := RecordPair(r.Root(), p)
	if err != nil {
		t.Fatalf("RecordPair: %v", err)
	}
	if id != m.ID || !strings.HasSuffix(where, m.ID+"/pairs.jsonl") || strings.Contains(where, home) {
		t.Fatalf("recorded in %s at %s, want lab %s in tilde form", id, where, m.ID)
	}
	if _, _, err := RecordPair(r.Root(), Pair{Type: "duplicate-match", Kind: "choose", StateSHA256: strings.Repeat("b", 64),
		StateTokens: 40000, Host: "none", Target: "openrouter/typesafe/jev-1.13", Skipped: SkippedWindow}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(read(t, filepath.Join(labDir(home, m), "pairs.jsonl"))), "\n")
	if len(lines) != 2 {
		t.Fatalf("pairs.jsonl holds %d lines, want 2", len(lines))
	}
	var got Pair
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil || got.Host != "iss-1" || got.Probability == nil ||
		*got.Probability != 0.92 || got.At == "" {
		t.Fatalf("first pair = %+v, %v", got, err)
	}
	if fi, err := os.Stat(filepath.Join(labDir(home, m), "pairs.jsonl")); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("pairs.jsonl mode = %v, %v; want 0600", fi, err)
	}
	if after := r.Git("status", "--porcelain", "--ignored"); after != before {
		t.Fatalf("the repository changed:\n%s\nwas\n%s", after, before)
	}
}

func TestRecordPairWithNoLabRefuses(t *testing.T) {
	r, _ := fixture(t)
	_, _, err := RecordPair(r.Root(), Pair{Type: "duplicate-match", Kind: "choose", StateSHA256: strings.Repeat("a", 64), Host: "x"})
	if !errors.Is(err, ErrNoLab) {
		t.Fatalf("err = %v, want ErrNoLab", err)
	}
}

func TestRecordPairRefusesAPairWithoutItsType(t *testing.T) {
	r, _ := fixture(t)
	notAbcd(t)
	mint(t, r)
	if _, _, err := RecordPair(r.Root(), Pair{Kind: "choose", StateSHA256: strings.Repeat("a", 64), Host: "x"}); !errors.Is(err, ErrRefused) {
		t.Fatalf("err = %v, want a refusal", err)
	}
}

// TestHarvestReportsAgreementPerTypeWithBands is the second criterion: the
// harvest reports agreement per judgement type with counts and probability
// bands, and the note it renders names the lab home and the count.
func TestHarvestReportsAgreementPerTypeWithBands(t *testing.T) {
	r, home := fixture(t)
	notAbcd(t)
	m := mint(t, r)
	sha := strings.Repeat("c", 64)
	for _, p := range []Pair{
		{Type: "duplicate-match", Host: "a", Adapter: "a", Probability: prob(0.95)},
		{Type: "duplicate-match", Host: "a", Adapter: "b", Probability: prob(0.95)},
		{Type: "duplicate-match", Host: "b", Adapter: "b", Probability: prob(0.6)},
		{Type: "duplicate-match", Host: "b", Skipped: SkippedWindow},
		{Type: "remedy-read", Host: "yes", Adapter: "yes", Probability: prob(0.3)},
	} {
		p.Kind, p.StateSHA256, p.Target = "choose", sha, "openrouter/typesafe/jev-1.13"
		if _, _, err := RecordPair(r.Root(), p); err != nil {
			t.Fatal(err)
		}
	}
	// A line that does not parse is counted, never guessed at.
	f, err := os.OpenFile(filepath.Join(labDir(home, m), "pairs.jsonl"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("{not json\n")
	f.Close()

	res, err := Harvest(r.Root(), m.ID)
	if err != nil {
		t.Fatalf("Harvest: %v", err)
	}
	a := res.Agreement
	if a == nil || a.Pairs != 5 || a.Unreadable != 1 || len(a.Types) != 2 {
		t.Fatalf("agreement = %+v", a)
	}
	dm := a.Types[0]
	if dm.Type != "duplicate-match" || dm.Pairs != 4 || dm.Compared != 3 || dm.Agreed != 2 || dm.Skipped != 1 {
		t.Fatalf("duplicate-match = %+v", dm)
	}
	bands := map[string]Band{}
	for _, b := range dm.Bands {
		bands[b.Band] = b
	}
	if b := bands["0.90-1.00"]; b.Compared != 2 || b.Agreed != 1 {
		t.Fatalf("0.90-1.00 band = %+v (bands %+v)", b, dm.Bands)
	}
	if b := bands["0.50-0.70"]; b.Compared != 1 || b.Agreed != 1 {
		t.Fatalf("0.50-0.70 band = %+v", b)
	}
	if rr := a.Types[1]; rr.Type != "remedy-read" || rr.Compared != 1 || rr.Agreed != 1 {
		t.Fatalf("remedy-read = %+v", rr)
	}
	labHome := StoreDisplay + "/" + m.RootSHA + "/" + m.ID
	if a.LabHome != labHome || !strings.Contains(a.Note, labHome) || !strings.Contains(a.Note, "5 pairs") {
		t.Fatalf("note = %q, home = %q; want the lab home %s and the count", a.Note, a.LabHome, labHome)
	}
	doc := read(t, filepath.Join(labDir(home, m), "harvest", "harvest.md"))
	for _, want := range []string{"## Agreement", "| duplicate-match | 4 | 3 | 2 | 1 | 67% |", "0.90-1.00: 1/2", a.Note} {
		if !strings.Contains(doc, want) {
			t.Errorf("harvest.md lacks %q:\n%s", want, doc)
		}
	}
}

func TestHarvestWithNoPairsHasNoAgreement(t *testing.T) {
	r, home := fixture(t)
	notAbcd(t)
	m := mint(t, r)
	res, err := Harvest(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Agreement != nil {
		t.Fatalf("agreement = %+v, want none without pairs", res.Agreement)
	}
	if strings.Contains(read(t, filepath.Join(labDir(home, m), "harvest", "harvest.md")), "## Agreement") {
		t.Fatal("harvest.md carries an agreement section with no pairs recorded")
	}
}

// TestHarvestReadsAPairsFilePastTheDocumentBound: a busy lab's pairs.jsonl
// grows past the bound one document read holds, and the harvest still reads
// every pair, a line at a time; a line longer than a pair may be is counted as
// one that did not parse, and the harvest goes on past it.
func TestHarvestReadsAPairsFilePastTheDocumentBound(t *testing.T) {
	r, home := fixture(t)
	notAbcd(t)
	m := mint(t, r)
	line, err := json.Marshal(Pair{At: "2026-10-29T06:00:00Z", Type: "eligibility-residual", Kind: "choose",
		StateSHA256: strings.Repeat("d", 64), StateTokens: 300, Host: "no", Adapter: "no", Probability: prob(0.93),
		Target: "openrouter/typesafe/jev-1.13", ModelReported: "typesafe/jev-1.13-0901"})
	if err != nil {
		t.Fatal(err)
	}
	line = append(line, '\n')
	n := maxDocBytes/len(line) + 1000
	var b strings.Builder
	for i := range n {
		if i == n/2 {
			b.WriteString(strings.Repeat("x", maxPairBytes+10) + "\n")
		}
		b.Write(line)
	}
	path := filepath.Join(labDir(home, m), "pairs.jsonl")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Size() <= maxDocBytes {
		t.Fatalf("the fixture is %v bytes, %v; want more than %d", fi, err, maxDocBytes)
	}
	res, err := Harvest(r.Root(), m.ID)
	if err != nil {
		t.Fatalf("Harvest over a pairs file past the bound: %v", err)
	}
	a := res.Agreement
	if a == nil || a.Pairs != n || a.Unreadable != 1 || len(a.Types) != 1 || a.Types[0].Agreed != n {
		t.Fatalf("agreement = %+v, want %d pairs, all agreed, and one unreadable line", a, n)
	}
}

// TestHarvestRefusesALinkedPairsFile: the harvest reads pairs.jsonl only as a
// regular file in the lab home, never through a link to one elsewhere.
func TestHarvestRefusesALinkedPairsFile(t *testing.T) {
	r, home := fixture(t)
	notAbcd(t)
	m := mint(t, r)
	elsewhere := filepath.Join(t.TempDir(), "pairs.jsonl")
	if err := os.WriteFile(elsewhere, []byte(`{"type":"remedy-read","kind":"choose","host":"yes"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(labDir(home, m), "pairs.jsonl")); err != nil {
		t.Fatal(err)
	}
	if _, err := Harvest(r.Root(), m.ID); err == nil || !strings.Contains(err.Error(), "pairs.jsonl") {
		t.Fatalf("Harvest through a linked pairs file = %v, want a refusal naming it", err)
	}
}
