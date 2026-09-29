package memory

import (
	"strings"
	"testing"
)

// TestValidateSingleSourceRequiresExternalProvenance closes the enforcement
// asymmetry between the two source shapes at the write/file-back trust boundary.
// validateSourceBlock's multi-source branch requires class/citation/licence/
// source_hash/ingested_at of every entry; its single-source branch checked only
// the class enum, so an external_* page supplied through `ask --file-back
// --page-json` with no source_hash passed validation. Downstream,
// externalSourceHashes returned empty and checkQuotation short-circuited to an
// MQ003 info reading "no external source" — false on the page's own frontmatter
// — with the MQ001/MQ002 quotation-budget blockers never running, and the page
// never back-linked into .sources_index.json.
//
// 07-memory.md §3 marks citation and licence required for external_*, and
// source_hash required for external_* and dredge_synthesis; the requirement is
// class-conditional, so hashless session_memory / work_notes pages stay valid.
func TestValidateSingleSourceRequiresExternalProvenance(t *testing.T) {
	hash := strings.Repeat("a", 64)
	citation := map[string]any{"type": "knowledge", "title": "T", "year": 2026}
	full := func() map[string]any {
		return map[string]any{
			"class":       "external_pdf",
			"citation":    map[string]any{"type": "knowledge", "title": "T", "year": 2026},
			"licence":     "CC-BY-4.0",
			"source_hash": hash,
			"ingested_at": "2026-08-19",
		}
	}

	refused := []struct {
		name  string
		block map[string]any
	}{
		{"external_without_source_hash", map[string]any{
			"class": "external_pdf", "citation": citation, "licence": "CC-BY-4.0", "ingested_at": "2026-08-19",
		}},
		{"external_without_licence", map[string]any{
			"class": "external_pdf", "citation": citation, "source_hash": hash, "ingested_at": "2026-08-19",
		}},
		{"external_without_citation", map[string]any{
			"class": "external_pdf", "licence": "CC-BY-4.0", "source_hash": hash, "ingested_at": "2026-08-19",
		}},
		{"external_with_blank_licence", map[string]any{
			"class": "external_pdf", "citation": citation, "licence": "  ", "source_hash": hash,
		}},
		{"external_with_non_digest_source_hash", map[string]any{
			"class": "external_pdf", "citation": citation, "licence": "CC-BY-4.0", "source_hash": "abcd",
		}},
		{"dredge_synthesis_without_source_hash", map[string]any{
			"class": "dredge_synthesis", "citation": citation, "licence": "CC-BY-4.0",
		}},
	}
	for _, tc := range refused {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateSourceBlock(tc.block); err == nil {
				t.Fatalf("validateSourceBlock accepted a single-source %v page missing required provenance: %#v", tc.block["class"], tc.block)
			}
		})
	}

	accepted := []struct {
		name  string
		block map[string]any
	}{
		{"external_complete", full()},
		{"session_memory_bare", map[string]any{"class": "session_memory"}},
		{"work_notes_bare", map[string]any{"class": "work_notes"}},
		{"issue_ledger_bare", map[string]any{"class": "issue_ledger"}},
	}
	for _, tc := range accepted {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateSourceBlock(tc.block); err != nil {
				t.Fatalf("validateSourceBlock rejected a legitimate block: %v (%#v)", err, tc.block)
			}
		})
	}

	// The same refusal must hold at the distiller / file-back boundary, which is
	// the reachable path: `ask --file-back --page-json` skips fileBackSource
	// (which demands all five fields) whenever the supplied page carries its own
	// source block, handing it straight to ValidateDistilledPage.
	if _, err := ValidateDistilledPage(t.TempDir(), map[string]any{
		"type": "topic", "domain": "auth", "slug": "x", "body": "# Subject line",
		"source": map[string]any{
			"class": "external_pdf", "citation": citation, "licence": "CC-BY-4.0", "ingested_at": "2026-08-19",
		},
	}); err == nil {
		t.Fatal("ValidateDistilledPage accepted an external_pdf page with no source_hash")
	}
	if _, err := ValidateDistilledPage(t.TempDir(), map[string]any{
		"type": "topic", "domain": "auth", "slug": "x", "body": "# Subject line",
		"source": full(),
	}); err != nil {
		t.Fatalf("ValidateDistilledPage rejected a fully provenanced external page: %v", err)
	}
}
