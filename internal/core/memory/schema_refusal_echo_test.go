package memory

import (
	"strings"
	"testing"
)

// memoryLeak is a payload value no refusal may carry back: a marker and a third
// party's absolute home path.
const memoryLeak = "zzleak-7f3a /Users/zzotherperson/notes" // abcd-lint:allow — a planted home path the refusal must not echo

// TestPageSchemaRefusalsDoNotEchoThePayload — iss-2609290300464268. A
// DistilledPage arrives host-produced (memory ingest --pages-json, memory ask
// --page-json), and the schema boundary quoted a refused source class, an
// ingested_at, the declared classes, the undeclared keys, and the type, domain
// and slug with %v or %q, never redacted, so a token or a home path in any of
// them reached the terminal and the transcript. A value is described now, a
// declared class is quoted only when it is in the closed enum, and an undeclared
// key is named through the canonical redactor.
func TestPageSchemaRefusalsDoNotEchoThePayload(t *testing.T) {
	page := func(mut func(p map[string]any)) map[string]any {
		p := map[string]any{
			"type": "topic", "domain": "auth", "slug": "x", "body": "# Subject line",
			"source": map[string]any{"class": "session_memory"},
		}
		mut(p)
		return p
	}
	entry := func(class string) map[string]any {
		return map[string]any{
			"class": class, "citation": map[string]any{"title": "t"}, "licence": "unknown",
			"source_hash": strings.Repeat("a", 64), "ingested_at": "2026-09-01",
		}
	}
	cases := []struct {
		name, names string
		data        map[string]any
	}{
		{"source class", "source class", page(func(p map[string]any) {
			p["source"] = map[string]any{"class": memoryLeak}
		})},
		{"ingested_at", "ingested_at", page(func(p map[string]any) {
			p["source"] = map[string]any{"class": "session_memory", "ingested_at": memoryLeak}
		})},
		{"declared classes", "source.classes", page(func(p map[string]any) {
			p["source"] = map[string]any{
				"classes": []any{"session_memory", memoryLeak},
				"sources": []any{entry("session_memory")},
			}
		})},
		{"undeclared key", "reviewer_notes", page(func(p map[string]any) {
			p["reviewer_notes "+"/Users/zzotherperson/notes"] = 1 // abcd-lint:allow — a planted home path in a KEY
		})},
		{"type", "DistilledPage.type", page(func(p map[string]any) { p["type"] = memoryLeak })},
		{"domain", "DistilledPage.domain", page(func(p map[string]any) { p["domain"] = memoryLeak })},
		{"slug", "DistilledPage.slug", page(func(p map[string]any) { p["slug"] = memoryLeak })},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateDistilledPage(t.TempDir(), tc.data)
			if err == nil {
				t.Fatal("a page carrying the leak was accepted")
			}
			for _, part := range []string{"zzleak-7f3a", "zzotherperson"} {
				if strings.Contains(err.Error(), part) {
					t.Errorf("the refusal echoes the payload (%q): %v", part, err)
				}
			}
			if !strings.Contains(err.Error(), tc.names) {
				t.Errorf("the refusal no longer names %s: %v", tc.names, err)
			}
		})
	}
}

// TestUncitedPageRefusalDoesNotEchoTheFilename — found by the sweep for
// iss-2609290300464268. A distilled page that does not cite the ingested source
// was refused naming page.Filename(), which carries the host-chosen slug; slugRe
// admits a token's characters, so a token-shaped slug was echoed whole. The page
// is named by its position in the distiller's output instead.
func TestUncitedPageRefusalDoesNotEchoTheFilename(t *testing.T) {
	repo := t.TempDir()
	token, _ := x46mSpans(t)
	src := writeSource(t, repo, "notes.md", "Rotate tokens every 24 hours.\n")
	distiller := func(_ string, _ map[string]any) ([]map[string]any, error) {
		return []map[string]any{{
			"type": "topic", "domain": "auth", "slug": token, "body": "# Token rotation\nRotate.\n",
			"source": map[string]any{"class": "session_memory"},
		}}, nil
	}
	_, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: distiller, Now: fixedNow})
	if err == nil {
		t.Fatal("a page that does not cite the ingested source was accepted")
	}
	if strings.Contains(err.Error(), token) {
		t.Errorf("the refusal echoes the page's token-shaped slug: %v", err)
	}
	if !strings.Contains(err.Error(), "does not cite the ingested source hash") || !strings.Contains(err.Error(), "page 1") {
		t.Errorf("the refusal no longer locates the page: %v", err)
	}
}
