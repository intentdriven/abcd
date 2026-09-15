package memory

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)

// oneTopicDistiller emits a single page (omitting source — the ingest path
// injects the computed single-source block that cites the ingested hash).
func oneTopicDistiller(typ, domain, slug, body string) Distiller {
	return func(_ string, _ map[string]any) ([]map[string]any, error) {
		return []map[string]any{{
			"type": typ, "domain": domain, "slug": slug, "body": body,
		}}, nil
	}
}

func writeSource(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	return p
}

// ---------------------------------------------------------------------------
// YAML round-trip
// ---------------------------------------------------------------------------

func TestYAMLRoundTripSourceBlocks(t *testing.T) {
	cases := []map[string]any{
		{
			"source": map[string]any{
				"class":       "external_pdf",
				"citation":    map[string]any{"type": "knowledge", "title": "T", "year": 2026},
				"licence":     "MIT",
				"source_hash": strings.Repeat("a", 64),
				"ingested_at": "2026-07-06",
			},
			"topic_hash":  strings.Repeat("b", 64),
			"contradicts": []any{"topic_auth_other"},
		},
		{
			"source": map[string]any{
				"classes":        []any{"external_pdf", "external_transcript"},
				"weighting_note": "pdf outweighs transcript",
				"sources": []any{
					map[string]any{"class": "external_pdf", "citation": map[string]any{"type": "knowledge"}, "licence": "MIT", "source_hash": strings.Repeat("c", 64), "ingested_at": "2026-07-06"},
					map[string]any{"class": "external_transcript", "citation": map[string]any{"type": "knowledge"}, "licence": "MIT", "source_hash": strings.Repeat("d", 64), "ingested_at": "2026-07-06"},
				},
			},
			"topic_hash": strings.Repeat("e", 64),
		},
	}
	for i, in := range cases {
		region, err := dumpFrontmatter(in)
		if err != nil {
			t.Fatalf("case %d dump: %v", i, err)
		}
		out, err := parseFrontmatter("---\n" + region + "---\n")
		if err != nil {
			t.Fatalf("case %d parse: %v\n%s", i, err, region)
		}
		// The source block must survive validation after the round trip.
		if err := validateSourceBlock(out["source"]); err != nil {
			t.Fatalf("case %d round-trip source invalid: %v\n%s", i, err, region)
		}
	}
}

// TestValidateSourceBlockClassesOrderIndependent proves source.classes is
// validated as a SET: the same classes declared in a different order from their
// first appearance in sources[] must pass, since the schema (and the error
// message) specify set semantics, not list order.
func TestValidateSourceBlockClassesOrderIndependent(t *testing.T) {
	block := map[string]any{
		// Declared order is the REVERSE of the sources[] appearance order.
		"classes":        []any{"external_transcript", "external_pdf"},
		"weighting_note": "pdf outweighs transcript",
		"sources": []any{
			map[string]any{"class": "external_pdf", "citation": map[string]any{"type": "knowledge"}, "licence": "MIT", "source_hash": strings.Repeat("c", 64), "ingested_at": "2026-07-06"},
			map[string]any{"class": "external_transcript", "citation": map[string]any{"type": "knowledge"}, "licence": "MIT", "source_hash": strings.Repeat("d", 64), "ingested_at": "2026-07-06"},
		},
	}
	if err := validateSourceBlock(block); err != nil {
		t.Fatalf("reordered-but-equal class set was rejected: %v", err)
	}
	// A genuinely different set is still rejected.
	block["classes"] = []any{"external_pdf", "external_pdf"}
	if err := validateSourceBlock(block); err == nil {
		t.Fatal("a class multiset that differs from the derived set was accepted")
	}
}

// TestYAMLFlowMapKeyRoundTrip (B06) proves distiller-controlled citation keys
// carrying YAML metacharacters survive the dump->parse round trip: the dumper
// quotes the key and parseFlowMap unquotes it, so a key with a colon, comma, or
// brace neither silently mutates the map nor breaks the read (the previous verbatim
// emission produced "missing ':' in flow-map pair" / "unexpected closing bracket").
func TestYAMLFlowMapKeyRoundTrip(t *testing.T) {
	for _, key := range []string{"injected: pwned", "a, b", "x}", "with\nnewline", "quote\"d"} {
		in := map[string]any{
			"source": map[string]any{
				"class":       "external_pdf",
				"citation":    map[string]any{key: "v", "title": "ok"},
				"licence":     "MIT",
				"source_hash": strings.Repeat("a", 64),
				"ingested_at": "2026-07-06",
			},
			"topic_hash": strings.Repeat("b", 64),
		}
		region, err := dumpFrontmatter(in)
		if err != nil {
			t.Fatalf("key %q dump: %v", key, err)
		}
		out, err := parseFrontmatter("---\n" + region + "---\n")
		if err != nil {
			t.Fatalf("key %q parse (round-trip broken): %v\n%s", key, err, region)
		}
		cit := out["source"].(map[string]any)["citation"].(map[string]any)
		if got, ok := cit[key]; !ok || got != "v" {
			t.Fatalf("key %q not preserved: got %+v\n%s", key, cit, region)
		}
	}
}

// TestYAMLBlockKeyRejected (B06) proves a nested block-level map key that cannot be
// represented as a bare identifier is rejected (fail closed) rather than emitted
// verbatim and silently corrupted on read-back.
func TestYAMLBlockKeyRejected(t *testing.T) {
	in := map[string]any{
		"source": map[string]any{
			"bad: key": "v",
		},
	}
	if _, err := dumpFrontmatter(in); err == nil {
		t.Fatal("a block-level key with a metacharacter must be rejected, got no error")
	}
}

// ---------------------------------------------------------------------------
// Ingest -> Ask -> re-ingest
// ---------------------------------------------------------------------------

func TestIngestAskLintFlow(t *testing.T) {
	repo := t.TempDir()
	src := writeSource(t, repo, "article.txt", "Token rotation policy: rotate tokens every 24 hours.")

	// --- ingest ---
	res, err := Ingest(IngestRequest{
		RepoRoot:  repo,
		Source:    src,
		Distiller: oneTopicDistiller("topic", "auth", "tokens", "# Token rotation\nRotate tokens every 24 hours."),
		Now:       fixedNow,
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if res.Status != "ingested" {
		t.Fatalf("status = %q, want ingested", res.Status)
	}
	if len(res.Pages) != 1 || res.Pages[0] != "topic_auth_tokens.md" {
		t.Fatalf("pages = %v, want [topic_auth_tokens.md]", res.Pages)
	}
	pagePath := filepath.Join(Dir(repo), "topic_auth_tokens.md")
	if _, err := os.Stat(pagePath); err != nil {
		t.Fatalf("page not written: %v", err)
	}
	if res.SourceTokenCount == 0 {
		t.Fatalf("source token count not persisted")
	}
	// Registry records the page under consumers.memory.pages for the hash.
	reg, err := LoadRegistry(SourcesIndexPath(repo))
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	entry := reg[res.ContentHash].(map[string]any)
	pages := anyToStrings(entry["consumers"].(map[string]any)["memory"].(map[string]any)["pages"])
	if len(pages) != 1 || pages[0] != "topic_auth_tokens.md" {
		t.Fatalf("registry pages = %v", pages)
	}

	// --- ask: deterministic recall returns the relevant fact ---
	ask, err := Ask(AskRequest{RepoRoot: repo, Question: "how does token rotation work?", Now: fixedNow})
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if len(ask.Matches) != 1 || ask.Matches[0].Filename != "topic_auth_tokens.md" {
		t.Fatalf("ask matches = %+v", ask.Matches)
	}
	if ask.Matches[0].Score < 1 {
		t.Fatalf("ask score = %d, want >=1", ask.Matches[0].Score)
	}
	if !strings.Contains(ask.Answer, res.ContentHash) {
		t.Fatalf("answer omits the source hash:\n%s", ask.Answer)
	}
	if !strings.Contains(ask.Answer, "external_pdf") && !strings.Contains(ask.Answer, "external_article") {
		t.Fatalf("answer omits the source class:\n%s", ask.Answer)
	}

	// A class filter that excludes the only page returns nothing.
	empty, err := Ask(AskRequest{RepoRoot: repo, Question: "class:external_pdf tokens", Now: fixedNow})
	if err != nil {
		t.Fatalf("ask filtered: %v", err)
	}
	if len(empty.Matches) != 0 {
		t.Fatalf("class:external_pdf should exclude an external_article page, got %d", len(empty.Matches))
	}

	// --- re-ingest the identical source: registry-only, no distillation ---
	res2, err := Ingest(IngestRequest{
		RepoRoot: repo,
		Source:   src,
		Distiller: func(_ string, _ map[string]any) ([]map[string]any, error) {
			t.Fatalf("distiller must not run on a valid registry hit")
			return nil, nil
		},
		Now: fixedNow,
	})
	if err != nil {
		t.Fatalf("re-ingest: %v", err)
	}
	if res2.Status != "registry_only" {
		t.Fatalf("re-ingest status = %q, want registry_only", res2.Status)
	}

	// --- lint: a clean store has no blockers (exit 0) ---
	lr, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	if lr.Summary.Blockers != 0 || lr.ExitCode != 0 {
		t.Fatalf("clean lint: blockers=%d exit=%d findings=%+v", lr.Summary.Blockers, lr.ExitCode, lr.Findings)
	}
	if _, err := os.Stat(filepath.Join(lr.ReportDir, "report.json")); err != nil {
		t.Fatalf("lint report.json not written: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Lint rejects a malformed memory file
// ---------------------------------------------------------------------------

func TestLintRejectsMalformed(t *testing.T) {
	cases := []struct {
		name     string
		page     string
		wantCode string
	}{
		{
			name: "external_missing_licence",
			// external_pdf single-source page with no licence -> ML001 blocker.
			page: "---\nsource:\n  class: external_pdf\n  source_hash: " + strings.Repeat("a", 64) +
				"\ntopic_hash: " + strings.Repeat("b", 64) + "\n---\n\n# A quoted claim\n",
			wantCode: "ML001",
		},
		{
			name: "multi_source_entry_missing_licence",
			// The multi-source half of the ML001 licence check: an external
			// sources[] entry with no `licence` key is the violation, even though a
			// sibling entry of the same class declares one.
			page: "---\nsource:\n  classes: [external_pdf]\n  sources:\n" +
				"    -\n      class: external_pdf\n      citation: { type: knowledge }\n      licence: MIT\n      source_hash: " + strings.Repeat("a", 64) + "\n      ingested_at: 2026-07-06\n" +
				"    -\n      class: external_pdf\n      citation: { type: knowledge }\n      source_hash: " + strings.Repeat("b", 64) + "\n      ingested_at: 2026-07-06\n" +
				"topic_hash: " + strings.Repeat("c", 64) + "\n---\n\n# A fused claim\n",
			wantCode: "ML001",
		},
		{
			name: "mixed_classes_no_weighting_note",
			page: "---\nsource:\n  classes: [external_pdf, external_transcript]\n  sources:\n" +
				"    -\n      class: external_pdf\n      citation: { type: knowledge }\n      licence: MIT\n      source_hash: " + strings.Repeat("c", 64) + "\n      ingested_at: 2026-07-06\n" +
				"    -\n      class: external_transcript\n      citation: { type: knowledge }\n      licence: MIT\n      source_hash: " + strings.Repeat("d", 64) + "\n      ingested_at: 2026-07-06\n" +
				"topic_hash: " + strings.Repeat("e", 64) + "\n---\n\n# A fused claim\n",
			wantCode: "MS002",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			mem := Dir(repo)
			if err := os.MkdirAll(mem, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(mem, "topic_auth_bad.md"), []byte(tc.page), 0o644); err != nil {
				t.Fatal(err)
			}
			lr, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
			if err != nil {
				t.Fatalf("lint: %v", err)
			}
			if lr.ExitCode != 1 || lr.Summary.Blockers < 1 {
				t.Fatalf("expected a blocker (exit 1); got exit=%d summary=%+v findings=%+v", lr.ExitCode, lr.Summary, lr.Findings)
			}
			found := false
			for _, f := range lr.Findings {
				if f.Code == tc.wantCode && f.Severity == "blocker" {
					found = true
				}
			}
			if !found {
				t.Fatalf("want a %s blocker; findings=%+v", tc.wantCode, lr.Findings)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Lint: the MQ001 quotation budget (iss-34)
// ---------------------------------------------------------------------------

// TestLintQuotationBudgetMQ001 covers both MQ001 refusals — the contiguous
// quoted-span ceiling and the per-page share of a source — plus the under-budget
// control that proves the finding is the budget's doing. MQ001 is curator-
// advisory: it warns, never blocks (exit stays 0), and lint mutates no page.
func TestLintQuotationBudgetMQ001(t *testing.T) {
	sourceHash := strings.Repeat("a", 64)
	topicHash := strings.Repeat("b", 64)
	quote := "one two three four five six seven eight nine ten eleven twelve" // 12 words
	page := "---\nsource:\n  class: external_pdf\n  citation: { type: knowledge }\n  licence: MIT\n" +
		"  source_hash: " + sourceHash + "\n  ingested_at: 2026-07-06\n" +
		"topic_hash: " + topicHash + "\n---\n\n# Rotation practice\n\n> " + quote + "\n"

	cases := []struct {
		name        string
		budget      string
		tokenCount  int
		wantMQ001   int
		wantMessage string
	}{
		{
			name:        "contiguous_span_over_ceiling",
			budget:      `{"max_contiguous_quote_words": 5, "per_page_pct": 0.5}`,
			tokenCount:  1000, // 12/1000 = 1.2%, well under the per-page share
			wantMQ001:   1,
			wantMessage: "exceeds the 5-word ceiling",
		},
		{
			name:        "per_page_share_over_budget",
			budget:      `{"max_contiguous_quote_words": 500, "per_page_pct": 0.01}`,
			tokenCount:  100, // 12/100 = 12%, over the 1% per-page budget
			wantMQ001:   1,
			wantMessage: "over the 1% per-page budget",
		},
		{
			name:       "under_budget_is_clean",
			budget:     `{"max_contiguous_quote_words": 500, "per_page_pct": 0.5}`,
			tokenCount: 1000,
			wantMQ001:  0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			mem := Dir(repo)
			if err := os.MkdirAll(mem, 0o755); err != nil {
				t.Fatal(err)
			}
			pagePath := filepath.Join(mem, "topic_auth_rotation.md")
			if err := os.WriteFile(pagePath, []byte(page), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(mem, "config.json"),
				[]byte(`{"quotation_budget": `+tc.budget+`}`), 0o644); err != nil {
				t.Fatal(err)
			}
			registry := fmt.Sprintf(`{%q: {"source_token_count": %d}}`, sourceHash, tc.tokenCount)
			if err := os.WriteFile(SourcesIndexPath(repo), []byte(registry), 0o644); err != nil {
				t.Fatal(err)
			}

			lr, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
			if err != nil {
				t.Fatalf("lint: %v", err)
			}
			var mq001 []Finding
			for _, f := range lr.Findings {
				if f.Code == "MQ001" {
					mq001 = append(mq001, f)
				}
			}
			if len(mq001) != tc.wantMQ001 {
				t.Fatalf("got %d MQ001 finding(s), want %d; findings=%+v", len(mq001), tc.wantMQ001, lr.Findings)
			}
			for _, f := range mq001 {
				if f.Severity != "warn" {
					t.Errorf("MQ001 severity = %q, want warn (curator-advisory)", f.Severity)
				}
				if f.File != pagePath {
					t.Errorf("MQ001 file = %q, want the offending page %q", f.File, pagePath)
				}
				if f.Line <= 0 {
					t.Errorf("MQ001 carries no line: %+v", f)
				}
				if !strings.Contains(f.Message, tc.wantMessage) {
					t.Errorf("MQ001 message = %q, want it to name %q", f.Message, tc.wantMessage)
				}
			}
			// The budget is advisory: a warning never blocks the exit contract.
			if lr.Summary.Blockers != 0 || lr.ExitCode != 0 {
				t.Errorf("MQ001 must not block: blockers=%d exit=%d", lr.Summary.Blockers, lr.ExitCode)
			}
			// No side effects: lint mutates no memory-store page.
			after, err := os.ReadFile(pagePath)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != page {
				t.Errorf("lint mutated the linted page:\n%s", after)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Ask --file-back: the one mutating branch of ask (iss-34)
// ---------------------------------------------------------------------------

// storeSnapshot reads every file in the memory store (bar the advisory lock,
// whose presence is not store state) so a refusal can be proved side-effect-free.
func storeSnapshot(t *testing.T, repo string) map[string]string {
	t.Helper()
	snap := map[string]string{}
	err := filepath.WalkDir(Dir(repo), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Base(path) == ".lock" {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		rel, relErr := filepath.Rel(Dir(repo), path)
		if relErr != nil {
			return nil
		}
		snap[filepath.ToSlash(rel)] = string(raw)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot store: %v", err)
	}
	return snap
}

func assertStoreUnchanged(t *testing.T, before, after map[string]string) {
	t.Helper()
	for name, content := range after {
		prev, ok := before[name]
		switch {
		case !ok:
			t.Errorf("a refused file-back wrote a new store file: %s", name)
		case prev != content:
			t.Errorf("a refused file-back mutated store file %s", name)
		}
	}
	for name := range before {
		if _, ok := after[name]; !ok {
			t.Errorf("a refused file-back removed store file %s", name)
		}
	}
}

// seedAskStore ingests one source so the store holds a fully-attributed page,
// returning its content hash.
func seedAskStore(t *testing.T, repo string) string {
	t.Helper()
	src := writeSource(t, repo, "article.txt", "Token rotation policy: rotate tokens every 24 hours.")
	res, err := Ingest(IngestRequest{
		RepoRoot:  repo,
		Source:    src,
		Distiller: oneTopicDistiller("topic", "auth", "tokens", "# Token rotation\nRotate tokens every 24 hours."),
		Now:       fixedNow,
	})
	if err != nil {
		t.Fatalf("seed ingest: %v", err)
	}
	return res.ContentHash
}

// TestAskFileBackWritesCitedPage is the accept path of `ask --file-back`: a page
// filed back without its own source block inherits the provenance of the cited
// matches, is written through the same seams as ingest, and is back-linked in the
// registry under the cited source hash.
func TestAskFileBackWritesCitedPage(t *testing.T) {
	repo := t.TempDir()
	hash := seedAskStore(t, repo)

	ask, err := Ask(AskRequest{
		RepoRoot: repo,
		Question: "how does token rotation work?",
		FileBackPage: map[string]any{
			"type": "topic", "domain": "auth", "slug": "rotation-summary",
			"body": "# Rotation summary\nTokens are rotated on a daily cadence.",
		},
		DecideFileBack: func(DistilledPage) bool { return true },
		Now:            fixedNow,
	})
	if err != nil {
		t.Fatalf("ask --file-back: %v", err)
	}
	if ask.FileBack == nil {
		t.Fatal("file-back result missing")
	}
	if ask.FileBack.Status != "written" {
		t.Fatalf("file-back status = %q, want written", ask.FileBack.Status)
	}
	const filed = "topic_auth_rotation-summary.md"
	if len(ask.FileBack.Pages) != 1 || ask.FileBack.Pages[0] != filed {
		t.Fatalf("file-back pages = %v, want [%s]", ask.FileBack.Pages, filed)
	}
	raw, err := os.ReadFile(filepath.Join(Dir(repo), filed))
	if err != nil {
		t.Fatalf("filed-back page not written: %v", err)
	}
	// The derived source block cites the ingested source, not an invented one.
	fm, err := parseFrontmatter(string(raw))
	if err != nil {
		t.Fatalf("filed-back page frontmatter: %v\n%s", err, raw)
	}
	src, ok := fm["source"].(map[string]any)
	if !ok {
		t.Fatalf("filed-back page carries no source block:\n%s", raw)
	}
	if err := validateSourceBlock(src); err != nil {
		t.Fatalf("derived source block is invalid: %v\n%s", err, raw)
	}
	if got := SourceHashes(src); len(got) != 1 || got[0] != hash {
		t.Fatalf("derived source hashes = %v, want [%s]", got, hash)
	}
	// The registry back-links the new page under the cited hash.
	reg, err := LoadRegistry(SourcesIndexPath(repo))
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := reg[hash].(map[string]any)
	if !ok {
		t.Fatalf("registry entry for the cited hash missing: %+v", reg)
	}
	pages := anyToStrings(entry["consumers"].(map[string]any)["memory"].(map[string]any)["pages"])
	if !contains(pages, filed) {
		t.Errorf("registry pages = %v, want the filed-back page %s back-linked", pages, filed)
	}
}

// TestAskFileBackRefusesWithoutWriting is the refusal corpus of the file-back
// write path: each refusing condition must be raised BEFORE any write, with the
// pinned error shape, and must leave the memory store byte-identical.
func TestAskFileBackRefusesWithoutWriting(t *testing.T) {
	fbPage := func() map[string]any {
		return map[string]any{
			"type": "topic", "domain": "auth", "slug": "rotation-summary",
			"body": "# Rotation summary\nTokens are rotated on a daily cadence.",
		}
	}

	t.Run("no_matches", func(t *testing.T) {
		repo := t.TempDir()
		seedAskStore(t, repo)
		before := storeSnapshot(t, repo)

		res, err := Ask(AskRequest{
			RepoRoot: repo, Question: "photosynthesis chlorophyll",
			FileBackPage: fbPage(), Now: fixedNow,
		})
		if err == nil {
			t.Fatalf("a file-back with no cited matches must be refused, got %+v", res)
		}
		var ae *AskError
		if !errors.As(err, &ae) {
			t.Fatalf("error = %T (%v), want *AskError", err, err)
		}
		if !strings.Contains(err.Error(), "no matching memory pages") {
			t.Errorf("refusal message = %q, want the unattributable-page refusal", err)
		}
		assertStoreUnchanged(t, before, storeSnapshot(t, repo))
	})

	t.Run("declined_by_decision", func(t *testing.T) {
		repo := t.TempDir()
		seedAskStore(t, repo)
		before := storeSnapshot(t, repo)

		ask, err := Ask(AskRequest{
			RepoRoot: repo, Question: "how does token rotation work?",
			FileBackPage:   fbPage(),
			DecideFileBack: func(DistilledPage) bool { return false },
			Now:            fixedNow,
		})
		if err != nil {
			t.Fatalf("a declined file-back is not an error: %v", err)
		}
		if ask.FileBack == nil || ask.FileBack.Status != "declined" {
			t.Fatalf("file-back = %+v, want status declined", ask.FileBack)
		}
		if len(ask.FileBack.Pages) != 0 || ask.FileBack.WriteReport != nil {
			t.Errorf("a declined file-back must report no pages and no write: %+v", ask.FileBack)
		}
		assertStoreUnchanged(t, before, storeSnapshot(t, repo))
	})

	t.Run("cited_pages_without_provenance", func(t *testing.T) {
		repo := t.TempDir()
		mem := Dir(repo)
		if err := os.MkdirAll(mem, 0o755); err != nil {
			t.Fatal(err)
		}
		// A page whose source block carries a class but none of the per-source
		// provenance a filed-back page would have to inherit.
		page := "---\nsource:\n  class: session_memory\ntopic_hash: " + strings.Repeat("c", 64) +
			"\n---\n\n# Token rotation\n"
		if err := os.WriteFile(filepath.Join(mem, "topic_auth_tokens.md"), []byte(page), 0o644); err != nil {
			t.Fatal(err)
		}
		before := storeSnapshot(t, repo)

		res, err := Ask(AskRequest{
			RepoRoot: repo, Question: "token rotation",
			FileBackPage: fbPage(), Now: fixedNow,
		})
		if err == nil {
			t.Fatalf("a file-back citing pages without complete provenance must be refused, got %+v", res)
		}
		var ae *AskError
		if !errors.As(err, &ae) {
			t.Fatalf("error = %T (%v), want *AskError", err, err)
		}
		if !strings.Contains(err.Error(), "file-back needs provenance") {
			t.Errorf("refusal message = %q, want the missing-provenance refusal", err)
		}
		assertStoreUnchanged(t, before, storeSnapshot(t, repo))
	})
}

// ---------------------------------------------------------------------------
// Single-writer lock
// ---------------------------------------------------------------------------

func TestStoreLockFailsClosed(t *testing.T) {
	repo := t.TempDir()
	err := WithStoreLock(repo, func() error {
		// A nested acquisition while the lock is held must fail closed.
		_, err := WritePages(repo, nil, nil, fixedNow)
		if _, ok := err.(*StoreLockHeldError); !ok {
			t.Fatalf("nested WritePages error = %v, want *StoreLockHeldError", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("outer WithStoreLock: %v", err)
	}
}

// TestWritePagesRegistryMergeNoLostUpdate is the Finding-C regression: the
// registry mutation must be recomputed against the registry read fresh under the
// store lock, not a pre-lock snapshot. It seeds hash A, then applies a second
// merge (as a concurrent ingest whose basis predates A's write would) that adds
// hash B. Both entries must survive — a wholesale write from a stale basis would
// silently drop A (and orphan its pages).
func TestWritePagesRegistryMergeNoLostUpdate(t *testing.T) {
	repo := t.TempDir()
	hashA := strings.Repeat("a", 64)
	hashB := strings.Repeat("b", 64)
	event := func(h string) IngestEvent {
		return IngestEvent{
			ContentHash: h, Consumer: "memory", SourceClass: "external_article",
			Citation: map[string]any{}, Origin: "https://example.test/" + h[:8],
			Licence: "MIT", IngestedAt: "2026-07-06",
		}
	}

	if _, err := WritePages(repo, nil, func(cur map[string]any) (map[string]any, error) {
		return MergeIngest(cur, event(hashA))
	}, fixedNow); err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	// The second merge closure is applied to the registry loaded fresh under the
	// lock (which now holds A), so A must not be lost when B is added.
	if _, err := WritePages(repo, nil, func(cur map[string]any) (map[string]any, error) {
		return MergeIngest(cur, event(hashB))
	}, fixedNow); err != nil {
		t.Fatalf("second ingest: %v", err)
	}

	reg, err := LoadRegistry(SourcesIndexPath(repo))
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range []string{hashA, hashB} {
		if _, ok := reg[h]; !ok {
			t.Errorf("registry entry %s… lost (concurrent-ingest lost update)", h[:8])
		}
	}
}

// TestIngestRefusesSSRFTargets is the Finding-E regression: memory URL ingest
// must refuse to fetch link-local, loopback, private, and internal/metadata
// targets (cloud metadata at 169.254.169.254 must be unreachable). Every case
// resolves offline (IP literals, localhost, and the *.internal name-based guard).
// Every fixture is https so the SSRF guard stays the thing under test: since
// GHSA-35fj-9w6f-7h62 a plaintext http:// source is refused on its scheme
// alone, and an http fixture would pass this test without the host guard ever
// running.
func TestIngestRefusesSSRFTargets(t *testing.T) {
	repo := t.TempDir()
	distiller := func(_ string, _ map[string]any) ([]map[string]any, error) {
		return nil, nil // never reached — the fetch must be refused first
	}
	cases := []string{
		"https://169.254.169.254/latest/meta-data/", // cloud metadata (link-local)
		"https://127.0.0.1/x",                       // loopback
		"https://localhost/x",                       // loopback by name
		"https://[::1]/x",                           // IPv6 loopback
		"https://10.0.0.1/x",                        // private  abcd-audit:allow
		"https://192.168.1.1/x",                     // private  abcd-audit:allow
		"https://172.16.5.5/x",                      // private  abcd-audit:allow
		"https://metadata.google.internal/x",        // metadata name
		"https://svc.internal/x",                    // .internal name
		"https://[64:ff9b::a9fe:a9fe]/x",            // NAT64 embedding 169.254.169.254 (metadata)
		"https://[64:ff9b::7f00:1]/x",               // NAT64 embedding 127.0.0.1 (loopback)
		"https://[2002:a9fe:a9fe::]/x",              // 6to4 embedding the metadata endpoint  abcd-audit:allow
	}
	for _, target := range cases {
		t.Run(target, func(t *testing.T) {
			_, err := Ingest(IngestRequest{RepoRoot: repo, Source: target, Distiller: distiller})
			if err == nil {
				t.Fatalf("expected refusal for SSRF target %s", target)
			}
			// The guard must REFUSE before dialling — distinct from a mere
			// "connection refused" transport error (which would still reach the
			// target). "refusing to" / "cannot resolve host" are only produced by
			// the SSRF guard, never by a failed connection.
			msg := err.Error()
			if !strings.Contains(msg, "refusing to") && !strings.Contains(msg, "cannot resolve host") {
				t.Errorf("want an SSRF-guard refusal, got a different error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Schema boundary
// ---------------------------------------------------------------------------

func TestValidateDistilledPageRejectsSuppliedTopicHash(t *testing.T) {
	_, err := ValidateDistilledPage(map[string]any{
		"type": "topic", "domain": "auth", "slug": "x", "body": "# b",
		"source":     map[string]any{"class": "session_memory"},
		"topic_hash": strings.Repeat("a", 64),
	})
	if err == nil {
		t.Fatal("supplied topic_hash must be rejected")
	}
}

func TestValidateDistilledPageComputesTopicHash(t *testing.T) {
	p, err := ValidateDistilledPage(map[string]any{
		"type": "topic", "domain": "auth", "slug": "x", "body": "# Subject line",
		"source": map[string]any{"class": "session_memory"},
	})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !hex64Re.MatchString(p.TopicHash) {
		t.Fatalf("topic hash = %q", p.TopicHash)
	}
	if p.Filename() != "topic_auth_x.md" {
		t.Fatalf("filename = %q", p.Filename())
	}
}

// TestReadFetchedResponseRejectsNon2xx (iss-30 C12) proves a non-2xx HTTP
// response (a 404/500 error page) is an ingest error, not stored as source
// content.
func TestReadFetchedResponseRejectsNon2xx(t *testing.T) {
	for _, code := range []int{404, 500, 301, 403} {
		resp := &http.Response{
			StatusCode: code,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader("<html>error page body</html>")),
		}
		if _, err := readFetchedResponse("http://x/y", resp); err == nil {
			t.Fatalf("HTTP %d must be an ingest error, not accepted as content", code)
		}
	}
	// A 200 is read normally.
	ok := &http.Response{
		StatusCode: 200,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("real content")),
	}
	fs, err := readFetchedResponse("http://x/y", ok)
	if err != nil || string(fs.Body) != "real content" {
		t.Fatalf("200 response = %q, %v", fs.Body, err)
	}
}

// TestMaterialFromLocalSizeCap (iss-30 C13) proves a local file larger than the
// fetch cap is rejected before it is read whole into memory.
func TestMaterialFromLocalSizeCap(t *testing.T) {
	big := filepath.Join(t.TempDir(), "big.txt")
	f, err := os.Create(big)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxFetchBytes + 1); err != nil { // sparse; fast
		t.Fatal(err)
	}
	f.Close()
	// Assert the SIZE cap is what rejects it — a sparse (all-zero) file would also
	// trip the downstream NUL-byte check, so a bare "err != nil" would pass even
	// with the cap removed (false pin). The "exceeds" message is produced only by
	// the size guard, so this fails on revert.
	_, err = materialFromLocal("", big, nil)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("an over-cap local file must be rejected by the size cap, got: %v", err)
	}
}

// TestMaterialFromLocalTildeUser (iss-30) proves a "~/…" path expands to the home
// dir but a "~user" path is left literal (not mangled into home+"user").
func TestMaterialFromLocalTildeUser(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, "real.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := materialFromLocal("", "~/real.txt", nil); err != nil {
		t.Fatalf("~/real.txt must expand to the home dir and be found: %v", err)
	}
	// Create home/baduser/real.txt — the file a MANGLED "~baduser" would resolve
	// to (home + "baduser/real.txt"). Left literal, "~baduser/real.txt" must not
	// find it.
	if err := os.MkdirAll(filepath.Join(home, "baduser"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "baduser", "real.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := materialFromLocal("", "~baduser/real.txt", nil); err == nil {
		t.Fatal("~baduser must be left literal, not expanded to home+baduser (which would wrongly succeed)")
	}
}

// ---------------------------------------------------------------------------
// iss-30: ingest-boundary — partial-failure reporting + CRLF parser parity
// ---------------------------------------------------------------------------

// TestIngestKeepOriginalFailureStillReportsIngest proves the partial-failure
// instance: when --keep-original's storeOriginal fails AFTER WritePages has
// durably written the pages and registry, Ingest must report the successful
// ingest (not a bare total-failure error) and record the keep-original failure.
// The failure is forced by pre-creating the sources dir as a regular file.
func TestIngestKeepOriginalFailureStillReportsIngest(t *testing.T) {
	repo := t.TempDir()
	src := writeSource(t, repo, "article.txt", "Token rotation policy: rotate tokens every 24 hours.")

	// Make .abcd/memory/sources a regular file so storeOriginal fails.
	if err := os.MkdirAll(Dir(repo), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(Dir(repo), "sources"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Ingest(IngestRequest{
		RepoRoot:     repo,
		Source:       src,
		KeepOriginal: true,
		Distiller:    oneTopicDistiller("topic", "auth", "tokens", "# Token rotation\nRotate tokens every 24 hours."),
		Now:          fixedNow,
	})
	if err != nil {
		t.Fatalf("ingest must not report total failure when only keep-original failed: %v", err)
	}
	if res.Status != "ingested" {
		t.Fatalf("status = %q, want ingested", res.Status)
	}
	if len(res.Pages) != 1 || res.Pages[0] != "topic_auth_tokens.md" {
		t.Fatalf("pages = %v, want [topic_auth_tokens.md]", res.Pages)
	}
	// The page reached disk — the mutation the old code hid behind a total error.
	if _, err := os.Stat(filepath.Join(Dir(repo), "topic_auth_tokens.md")); err != nil {
		t.Fatalf("page not durably written: %v", err)
	}
	if res.KeepOriginalError == "" {
		t.Fatalf("keep-original failure must be recorded in the result")
	}
	if strings.Contains(res.KeepOriginalError, repo) {
		t.Fatalf("keep-original error leaked the absolute repo path:\n%s", res.KeepOriginalError)
	}
	if res.KeptOriginal != "" {
		t.Fatalf("KeptOriginal = %q, want empty when the copy failed", res.KeptOriginal)
	}
}

// TestIngestKeepOriginalWritesSourceCanonically is the success-path regression
// test for --keep-original: the stored original lands with the exact source bytes
// and a 0644 mode, and its repo-relative path is reported. (The reroute itself is
// pinned by the canonical-primitive detector; this locks the happy path.)
func TestIngestKeepOriginalWritesSourceCanonically(t *testing.T) {
	repo := t.TempDir()
	content := "Rotate tokens every 24 hours."
	src := writeSource(t, repo, "article.txt", content)

	res, err := Ingest(IngestRequest{
		RepoRoot:     repo,
		Source:       src,
		KeepOriginal: true,
		Distiller:    oneTopicDistiller("topic", "auth", "tokens", "# Token rotation\nRotate tokens every 24 hours."),
		Now:          fixedNow,
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if res.KeepOriginalError != "" {
		t.Fatalf("unexpected keep-original error: %s", res.KeepOriginalError)
	}
	if res.KeptOriginal == "" {
		t.Fatalf("KeptOriginal path must be reported on success")
	}
	stored := filepath.Join(repo, res.KeptOriginal)
	fi, err := os.Stat(stored)
	if err != nil {
		t.Fatalf("stored original not written: %v", err)
	}
	if fi.Mode().Perm() != 0o644 {
		t.Fatalf("stored original mode = %v, want 0644 (fsutil.WriteFileAtomic chmod)", fi.Mode().Perm())
	}
	got, err := os.ReadFile(stored)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Fatalf("stored original bytes = %q, want %q", got, content)
	}
}

// TestSplitFileFrontmatterCRLFParity proves the parser-parity instance: a
// CRLF-terminated document must split identically to its LF twin. Before the
// fix splitFileFrontmatter's exact-match closing delimiter ("---" != "---\r")
// rejected CRLF, while parseFrontmatter accepted it — degrading hashes.
func TestSplitFileFrontmatterCRLFParity(t *testing.T) {
	lf := "---\ntitle: Rotation\nyear: 2026\n---\nbody line one\nbody line two\n"
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")

	lfRegion, lfBody, err := splitFileFrontmatter(lf)
	if err != nil {
		t.Fatalf("LF split: %v", err)
	}
	crlfRegion, crlfBody, err := splitFileFrontmatter(crlf)
	if err != nil {
		t.Fatalf("CRLF split must not error (parser parity): %v", err)
	}
	if crlfRegion != lfRegion {
		t.Fatalf("region parity broken:\n LF=%q\nCRLF=%q", lfRegion, crlfRegion)
	}
	if crlfBody != lfBody {
		t.Fatalf("body parity broken:\n LF=%q\nCRLF=%q", lfBody, crlfBody)
	}
}

// TestKeepOriginalErrorMessageNoPathLeak locks the no-absolute-path invariant
// for every filesystem error shape storeOriginal can return: *os.PathError
// (single path) and *os.LinkError (rename, two paths). Both must be reduced to
// their bare cause against the repo-relative store location.
func TestKeepOriginalErrorMessageNoPathLeak(t *testing.T) {
	abs := "/Users/someone/secret/repo/.abcd/memory/sources" // abcd-audit:allow
	cases := []error{
		&os.PathError{Op: "open", Path: abs + "/deadbeef.pdf.tmp", Err: os.ErrPermission},
		&os.LinkError{Op: "rename", Old: abs + "/deadbeef.pdf.1.memtmp", New: abs + "/deadbeef.pdf", Err: os.ErrExist},
	}
	for _, in := range cases {
		msg := keepOriginalErrorMessage(in)
		if strings.Contains(msg, "/Users/someone") { // abcd-audit:allow
			t.Fatalf("leaked absolute path for %T: %q", in, msg)
		}
		if !strings.Contains(msg, sourcesRelPath) {
			t.Fatalf("message for %T omits the repo-relative store path: %q", in, msg)
		}
	}
}
