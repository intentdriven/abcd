package decide

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// nativeADRIDRe is the shape of a native adr id: the family tag, a 12-digit UTC
// second stamp and a 4-digit suffix (adr-45; mechanics per spc-33; the ADR
// family adopted by the 2026-09-01 ruling).
var nativeADRIDRe = regexp.MustCompile(`^adr-[0-9]{16}$`)

// writeFile lays one file into a fixture repository.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// seedOrdinals writes the two highest hand-numbered ADRs a real checkout holds,
// so a mint that counted a maximum would be visible as 0059.
func seedOrdinals(t *testing.T, root string) {
	t.Helper()
	writeFile(t, root, ADRsRelDir+"/0057-grounds-accumulate.md",
		"---\nid: adr-57\nslug: grounds-accumulate\nstatus: accepted\n---\n# ADR-57: Grounds accumulate\n")
	writeFile(t, root, ADRsRelDir+"/0058-a-reading-is-commissioned.md",
		"---\nid: adr-58\nslug: a-reading-is-commissioned\nstatus: accepted\n---\n# ADR-58: A reading is commissioned\n")
}

// setMinter swaps the package mint seam for the test's lifetime.
func setMinter(t *testing.T, m recordid.Minter) {
	t.Helper()
	prev := minter
	minter = m
	t.Cleanup(func() { minter = prev })
}

// TestCreateMintsPastTheOrdinalsWithoutCounting is the 2026-09-01 ruling at the
// ADR store: a checkout already holding 0058 mints a timestamp-shaped id, never
// adr-59. The mint reads no maximum — not the store's, not the citations'.
func TestCreateMintsPastTheOrdinalsWithoutCounting(t *testing.T) {
	root := t.TempDir()
	seedOrdinals(t, root)

	d, err := Create(root, "Decision records mint through the timestamp seam")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if d.ID == "adr-59" || d.ID == "adr-0059" {
		t.Fatalf("minted %s: the mint counted a maximum instead of stamping the clock", d.ID)
	}
	if !nativeADRIDRe.MatchString(d.ID) {
		t.Fatalf("minted id %q is not native-shaped (adr-<yymmddHHMMSS><rrrr>)", d.ID)
	}
	stamp := strings.TrimPrefix(d.ID, "adr-")
	if want := stamp + "-" + d.Slug + ".md"; filepath.Base(d.Path) != want {
		t.Fatalf("filename %q, want %q — the filename is ordered by the stamp", filepath.Base(d.Path), want)
	}
	if _, err := os.Stat(filepath.Join(root, d.Path)); err != nil {
		t.Fatalf("minted record is not on disk: %v", err)
	}
	// The ordinals keep their ids and their filenames; nothing is renumbered.
	for _, rel := range []string{
		ADRsRelDir + "/0057-grounds-accumulate.md",
		ADRsRelDir + "/0058-a-reading-is-commissioned.md",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Errorf("%s was disturbed by the mint: %v", rel, err)
		}
	}
}

// TestCreateInTwoCheckoutsNeverCollides is the collision the ruling names: two
// branches minted 0055 and 0056 on the same day for different decisions. Two
// current checkouts of one tree, minting in the same second, must not converge
// — the entropy alone separates them, so the draws are scripted and the
// assertion is exact.
func TestCreateInTwoCheckoutsNeverCollides(t *testing.T) {
	instant := time.Date(2026, 9, 1, 22, 6, 5, 0, time.UTC)
	setMinter(t, recordid.Minter{
		Now:     func() time.Time { return instant },
		Entropy: bytes.NewReader([]byte{0x00, 0x2A, 0x11, 0x11}),
	})

	checkoutA, checkoutB := t.TempDir(), t.TempDir()
	seedOrdinals(t, checkoutA)
	seedOrdinals(t, checkoutB)

	a, err := Create(checkoutA, "The construal stands in the record")
	if err != nil {
		t.Fatalf("checkout A Create: %v", err)
	}
	b, err := Create(checkoutB, "An exclusion control asserts only what it can prove")
	if err != nil {
		t.Fatalf("checkout B Create: %v", err)
	}
	if a.ID == b.ID {
		t.Fatalf("two checkouts minted one id in the same second: %s", a.ID)
	}
	if a.ID != "adr-2609012206050042" || b.ID != "adr-2609012206054369" {
		t.Fatalf("ids = %s, %s: want the pinned instant with the scripted suffixes 0042 and 4369", a.ID, b.ID)
	}
}

// TestCreateRedrawsOnAPresentID is the presence-check redraw loop (spc-33
// ruling 2): a candidate already naming a record in this checkout is redrawn,
// never bumped — a bump would re-derive the next id from the store's occupancy,
// a miniature maximum-plus-one.
func TestCreateRedrawsOnAPresentID(t *testing.T) {
	instant := time.Date(2026, 9, 1, 22, 6, 5, 0, time.UTC)
	setMinter(t, recordid.Minter{
		Now: func() time.Time { return instant },
		// The first draw is 0042, which the fixture already holds; the second is
		// 4369 and is free.
		Entropy: bytes.NewReader([]byte{0x00, 0x2A, 0x11, 0x11}),
	})
	root := t.TempDir()
	writeFile(t, root, ADRsRelDir+"/2609012206050042-already-here.md",
		"---\nid: adr-2609012206050042\nslug: already-here\nstatus: accepted\n---\n# ADR\n")

	d, err := Create(root, "A second decision in the same second")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if d.ID != "adr-2609012206054369" {
		t.Fatalf("id = %s, want adr-2609012206054369 (the taken draw redrawn, never bumped)", d.ID)
	}
}

// TestCreateWritesTheSkeleton pins the record shape the store's existing files
// carry: the nine frontmatter keys in their order, the ADR-<id> H1, and the four
// body sections the README specifies.
func TestCreateWritesTheSkeleton(t *testing.T) {
	root := t.TempDir()
	d, err := Create(root, "Decision records mint through the timestamp seam")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, d.Path))
	if err != nil {
		t.Fatalf("record unreadable: %v", err)
	}
	body := string(data)
	lines := strings.Split(body, "\n")

	fields := frontmatter.Fields(lines)
	for key, want := range map[string]string{
		"id":              d.ID,
		"slug":            d.Slug,
		"status":          "proposed",
		"date":            d.Date,
		"supersedes":      "null",
		"superseded_by":   "null",
		"related_intents": "[]",
		"related_rfcs":    "[]",
		"related_adrs":    "[]",
	} {
		if got := fields[key].Value; got != want {
			t.Errorf("frontmatter %s = %q, want %q", key, got, want)
		}
	}
	if want := "# ADR-" + strings.TrimPrefix(d.ID, "adr-") + ": " + d.Title; !strings.Contains(body, want) {
		t.Errorf("body missing the H1 %q", want)
	}
	for _, heading := range []string{"## Context", "## Decision", "## Alternatives Considered", "## Consequences"} {
		if !strings.Contains(body, "\n"+heading+"\n") {
			t.Errorf("body missing the %q section", heading)
		}
	}
}

// TestCreateRefusesUnusableTitles: the title becomes a filename, so anything
// that cannot yield a kebab-case slug is refused and nothing is written.
func TestCreateRefusesUnusableTitles(t *testing.T) {
	for _, title := range []string{"", "   ", "///", "…"} {
		root := t.TempDir()
		if d, err := Create(root, title); err == nil {
			t.Fatalf("Create(%q) = %+v, want a refusal", title, d)
		}
		if entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(ADRsRelDir))); err == nil && len(entries) > 0 {
			t.Fatalf("Create(%q) wrote %d file(s) on a refusal", title, len(entries))
		}
	}
}
