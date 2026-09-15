package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBackfillLegacyAsksTheParserNotTheBytePrefix pins backfillLegacy — the only
// mutating step of the WritePages sweep — to the same "does this page have
// frontmatter?" answer the package's parsers give. The gate used to be the raw
// byte test strings.HasPrefix(text, "---"), which has no preamble tolerance,
// while parseFrontmatter/splitFileFrontmatter deliberately skip a leading
// HTML-comment or a delimiter carrying leading whitespace. On a page they
// disagree about, the else branch wrapped the ENTIRE original text — real
// frontmatter included — as the body of a synthetic block, demoting the page's
// declared `source:` provenance into prose and replacing it with a fabricated
// `session_memory` class. That silently exits the page from the ML001 licence
// regime and severs its quotation-coverage attribution.
func TestBackfillLegacyAsksTheParserNotTheBytePrefix(t *testing.T) {
	hash := strings.Repeat("9", 64)

	commentLed := "<!-- Adapted from interview-2026-03.pdf (CC-BY-4.0). See ACKNOWLEDGEMENTS. -->\n" +
		"---\ntitle: Q3 revenue analysis\nsource:\n  class: external_transcript\n  source_hash: " + hash +
		"\n---\nThe interview subject stated that revenue fell.\n"
	indentedOpen := " ---\nsource:\n  class: external_pdf\n  source_hash: " + hash + "\n---\nBody.\n"
	commentLedUnterminated := "<!-- provenance note -->\n---\nsource:\n  class: external_pdf\n" +
		"the block was never closed\n"

	cases := []struct {
		name string
		file string
		page string
	}{
		{"comment_led_frontmatter", "note_finance_q3-revenue.md", commentLed},
		{"indented_opening_delimiter", "note_finance_q3-margin.md", indentedOpen},
		{"comment_led_unterminated", "note_finance_q3-costs.md", commentLedUnterminated},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mem := t.TempDir()
			path := filepath.Join(mem, tc.file)
			if err := os.WriteFile(path, []byte(tc.page), 0o644); err != nil {
				t.Fatal(err)
			}
			backfilled, err := backfillLegacy(mem)
			if err != nil {
				t.Fatalf("backfillLegacy: %v", err)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != tc.page {
				t.Fatalf("backfillLegacy rewrote a page the parsers read as frontmatter-bearing.\n--- before ---\n%s\n--- after ---\n%s", tc.page, after)
			}
			if contains(backfilled, tc.file) {
				t.Fatalf("page reported as backfilled: %v", backfilled)
			}
		})
	}
}

// TestBackfillLegacyLeavesAPreambleItCannotPreserve covers the guard the cases
// above never reach: they all short-circuit on the fm["source"] check, so the
// preamble guard that follows it is load-bearing but untested. A page that has
// a preamble AND no source: key is the one that reaches it — and backfilling it
// would mean rebuilding the file from (region, body), which cannot carry the
// preamble across. The comment here carries the page's attribution, so the
// backfill leaves the page alone rather than dropping it: the page simply stays
// unclassified, which the lint crawl already tolerates.
func TestBackfillLegacyLeavesAPreambleItCannotPreserve(t *testing.T) {
	cases := []struct {
		name string
		file string
		page string
	}{
		{
			name: "comment_led_frontmatter_without_source",
			file: "note_finance_q3-attrib.md",
			page: "<!-- Adapted from interview.pdf (CC-BY-4.0). -->\n---\ntitle: T\n---\nProse.\n",
		},
		{
			name: "indented_open_without_source",
			file: "note_finance_q3-indent.md",
			page: " ---\ntitle: T\n---\nProse.\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mem := t.TempDir()
			path := filepath.Join(mem, tc.file)
			if err := os.WriteFile(path, []byte(tc.page), 0o644); err != nil {
				t.Fatal(err)
			}
			backfilled, err := backfillLegacy(mem)
			if err != nil {
				t.Fatalf("backfillLegacy: %v", err)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != tc.page {
				t.Fatalf("backfillLegacy rewrote a page whose preamble the rebuild cannot carry across.\n--- before ---\n%s\n--- after ---\n%s", tc.page, after)
			}
			if contains(backfilled, tc.file) {
				t.Fatalf("page reported as backfilled: %v", backfilled)
			}
		})
	}
}

// TestBackfillLegacyStillBackfillsGenuineLegacyPages is the control: the pages
// backfillLegacy exists for — a pre-itd-36 flat page with no frontmatter, and a
// page whose frontmatter carries no source block — must still be backfilled.
func TestBackfillLegacyStillBackfillsGenuineLegacyPages(t *testing.T) {
	mem := t.TempDir()
	flat := filepath.Join(mem, "note_finance_flat.md")
	if err := os.WriteFile(flat, []byte("# A flat legacy page\n\nProse.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sourceless := filepath.Join(mem, "note_finance_sourceless.md")
	if err := os.WriteFile(sourceless, []byte("---\ntitle: T\n---\nProse.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	backfilled, err := backfillLegacy(mem)
	if err != nil {
		t.Fatalf("backfillLegacy: %v", err)
	}
	if len(backfilled) != 2 {
		t.Fatalf("backfilled = %v, want both legacy pages", backfilled)
	}
	for _, path := range []string{flat, sourceless} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		fm, err := parseFrontmatter(string(raw))
		if err != nil {
			t.Fatalf("%s: %v\n%s", path, err, raw)
		}
		src, ok := fm["source"].(map[string]any)
		if !ok || src["class"] != backfillSourceClass {
			t.Fatalf("%s: source = %#v, want class %s", filepath.Base(path), fm["source"], backfillSourceClass)
		}
		if !strings.Contains(string(raw), "Prose.") {
			t.Fatalf("%s lost its body:\n%s", filepath.Base(path), raw)
		}
	}
	// The sourceless page keeps its existing frontmatter keys rather than having
	// them demoted into the body.
	raw, _ := os.ReadFile(sourceless)
	fm, _ := parseFrontmatter(string(raw))
	if fm["title"] != "T" {
		t.Fatalf("existing frontmatter key lost: %#v", fm)
	}
}

// TestPageReadersToleratePreamble covers the three non-mutating siblings that
// carried the same byte-prefix gate: a comment-led page's citations vanished
// from `ask` (pageSourceBlock), its raw YAML was rendered into an answer body
// (pageBody), and it showed as (unclassified) in the generated index
// (pageInfoFrom).
func TestPageReadersToleratePreamble(t *testing.T) {
	hash := strings.Repeat("9", 64)
	page := "<!-- Adapted from interview-2026-03.pdf (CC-BY-4.0). -->\n" +
		"---\nsource:\n  class: external_transcript\n  source_hash: " + hash +
		"\n---\nThe interview subject stated that revenue fell.\n"

	src := pageSourceBlock(page)
	if src["class"] != "external_transcript" {
		t.Errorf("pageSourceBlock = %#v, want the declared external_transcript block", src)
	}
	if body := parsePage(page).body; strings.Contains(body, "source:") {
		t.Errorf("parsePage body carries raw frontmatter: %q", body)
	}
	info := pageInfoFrom("note_finance_q3-revenue.md", page)
	if len(info.Classes) != 1 || info.Classes[0] != "external_transcript" {
		t.Errorf("pageInfoFrom classes = %v, want [external_transcript]", info.Classes)
	}
	if strings.Contains(info.Summary, "<!--") {
		t.Errorf("pageInfoFrom summarised the preamble comment: %q", info.Summary)
	}
}
