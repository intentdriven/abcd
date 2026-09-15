package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writer_leaf_redact_test.go — GHSA-x46m-mw9h-5jwj (superset of
// iss-2608291941064448). After GHSA-j5f5 the store redacted page BODIES; every
// other string a write introduced — host-supplied frontmatter (recall,
// contradicts, citation scalars, sources[].licence, weighting_note), the
// core-copied licence from an SPDX line or a License: header, the
// redirect-controlled origin/title, and the registry leaves MergeIngest,
// backlinkOtherHashes and ask --file-back place — landed unscanned in the
// committed .abcd/memory/ tree and came back raw through ingest --json and
// ask --json. Every string leaf a write INTRODUCES is now judged by the store
// redactor with the body's discipline (redact, sweep the caller's home, refuse
// only a blocking residual); leaves the registry already held are never
// re-judged, so a legacy store cannot lock a re-ingest out.
//
// Both spans are FAKE (a ghp_ token shape of literal 'A's; a home path under a
// set-for-the-test $HOME). Nothing here is a live credential.

const x46mHome = "/Users/testperson"

func x46mSpans(t *testing.T) (token, homePath string) {
	t.Helper()
	t.Setenv("HOME", x46mHome)
	return "ghp_" + strings.Repeat("A", 40), x46mHome + "/private/keys.txt"
}

func mustBeClean(t *testing.T, label, path string, spans ...string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s (%s): %v", label, path, err)
	}
	for _, s := range spans {
		if strings.Contains(string(raw), s) {
			t.Errorf("GHSA-x46m: %s persists the raw span %q unredacted:\n%s", label, s[:8], raw)
		}
	}
	return string(raw)
}

func storeMustBeClean(t *testing.T, repo, page string, spans ...string) string {
	t.Helper()
	mem := Dir(repo)
	text := mustBeClean(t, "page "+page, filepath.Join(mem, page), spans...)
	mustBeClean(t, "sources registry", SourcesIndexPath(repo), spans...)
	mustBeClean(t, "contradictions.md", filepath.Join(mem, "contradictions.md"), spans...)
	mustBeClean(t, "index.md", filepath.Join(mem, "index.md"), spans...)
	return text
}

func fixedBodyDistiller(mutate func(src map[string]any)) Distiller {
	return func(_ string, sourceBlock map[string]any) ([]map[string]any, error) {
		src := deepCopyMap(sourceBlock)
		if mutate != nil {
			mutate(src)
		}
		return []map[string]any{{
			"type": "topic", "domain": "auth", "slug": "tokens",
			"body": "# Token rotation\nRotate tokens every 24 hours.\n", "source": src,
		}}, nil
	}
}

func TestWritePagesRedactsEveryIntroducedLeaf(t *testing.T) {
	t.Run("recall", func(t *testing.T) {
		repo := t.TempDir()
		token, homePath := x46mSpans(t)
		src := writeSource(t, repo, "notes.md", "Rotate tokens every 24 hours.\n")
		distiller := func(_ string, sourceBlock map[string]any) ([]map[string]any, error) {
			return []map[string]any{{
				"type": "topic", "domain": "auth", "slug": "tokens", "body": "# Tokens\nRotate daily.\n",
				"source": sourceBlock, "recall": []any{"rotate " + token, "notes at " + homePath},
			}}, nil
		}
		if _, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: distiller, Now: fixedNow}); err != nil {
			t.Fatalf("ingest: %v", err)
		}
		page := storeMustBeClean(t, repo, "topic_auth_tokens.md", token, homePath)
		if !strings.Contains(page, "recall:") {
			t.Errorf("the recall list was dropped rather than redacted:\n%s", page)
		}
	})

	t.Run("contradicts", func(t *testing.T) {
		repo := t.TempDir()
		token, _ := x46mSpans(t)
		src := writeSource(t, repo, "notes.md", "Rotate tokens every 24 hours.\n")
		distiller := func(_ string, sourceBlock map[string]any) ([]map[string]any, error) {
			return []map[string]any{{
				"type": "topic", "domain": "auth", "slug": "tokens", "body": "# Tokens\nRotate daily.\n",
				"source": sourceBlock, "contradicts": []any{token + ".md"},
			}}, nil
		}
		if _, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: distiller, Now: fixedNow}); err != nil {
			t.Fatalf("ingest: %v", err)
		}
		storeMustBeClean(t, repo, "topic_auth_tokens.md", token)
	})

	t.Run("host citation.title", func(t *testing.T) {
		repo := t.TempDir()
		token, homePath := x46mSpans(t)
		src := writeSource(t, repo, "notes.md", "Rotate tokens every 24 hours.\n")
		distiller := fixedBodyDistiller(func(src map[string]any) {
			src["citation"].(map[string]any)["title"] = "keys " + token + " kept at " + homePath
		})
		if _, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: distiller, Now: fixedNow}); err != nil {
			t.Fatalf("ingest: %v", err)
		}
		storeMustBeClean(t, repo, "topic_auth_tokens.md", token, homePath)
	})

	t.Run("sources[].licence and weighting_note", func(t *testing.T) {
		repo := t.TempDir()
		token, homePath := x46mSpans(t)
		src := writeSource(t, repo, "notes.md", "Rotate tokens every 24 hours.\n")
		distiller := func(_ string, sourceBlock map[string]any) ([]map[string]any, error) {
			multi := map[string]any{
				"classes":        []any{"external_article", "session_memory"},
				"weighting_note": "weighted by " + token + " from " + homePath,
				"sources": []any{
					map[string]any{
						"class": "external_article", "citation": map[string]any{"title": "a"},
						"licence": "MIT " + token, "source_hash": sourceBlock["source_hash"], "ingested_at": "2026-07-06",
					},
					map[string]any{
						"class": "session_memory", "citation": map[string]any{"title": "b"},
						"licence": "unknown", "source_hash": strings.Repeat("b", 64), "ingested_at": "2026-07-06",
					},
				},
			}
			return []map[string]any{{
				"type": "topic", "domain": "auth", "slug": "tokens", "body": "# Tokens\nRotate daily.\n", "source": multi,
			}}, nil
		}
		if _, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: distiller, Now: fixedNow}); err != nil {
			t.Fatalf("ingest: %v", err)
		}
		storeMustBeClean(t, repo, "topic_auth_tokens.md", token, homePath)
	})

	t.Run("spdx licence", func(t *testing.T) {
		repo := t.TempDir()
		token, _ := x46mSpans(t)
		src := writeSource(t, repo, "notes.md", "SPDX-License-Identifier: "+token+"\nRotate tokens every 24 hours.\n")
		res, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: fixedBodyDistiller(nil), Now: fixedNow})
		if err != nil {
			t.Fatalf("ingest: %v", err)
		}
		if strings.Contains(res.Licence, token) {
			t.Errorf("GHSA-x46m: IngestResult.Licence (the --json field) carries the raw SPDX token: %q", res.Licence)
		}
		storeMustBeClean(t, repo, "topic_auth_tokens.md", token)
	})

	t.Run("header licence and query origin", func(t *testing.T) {
		repo := t.TempDir()
		token, _ := x46mSpans(t)
		fetcher := func(string) (FetchedSource, error) {
			return FetchedSource{
				FinalURL: "https://example.com/doc?access_token=" + token,
				Headers:  map[string]string{"Content-Type": "text/plain", "License": "MIT " + token},
				Body:     []byte("Rotate tokens every 24 hours.\n"),
			}, nil
		}
		res, err := Ingest(IngestRequest{RepoRoot: repo, Source: "https://example.com/doc", Distiller: fixedBodyDistiller(nil), Fetcher: fetcher, Now: fixedNow})
		if err != nil {
			t.Fatalf("ingest: %v", err)
		}
		if strings.Contains(res.Licence, token) {
			t.Errorf("GHSA-x46m: IngestResult.Licence carries the raw License: header token: %q", res.Licence)
		}
		for _, key := range []string{"origin", "title"} {
			v, _ := res.Citation[key].(string)
			if strings.Contains(v, token) {
				t.Errorf("GHSA-x46m: IngestResult.Citation[%s] carries the raw query token: %q", key, v)
			}
			if !strings.Contains(v, "example.com/doc") {
				t.Errorf("redaction must keep the legitimate address in citation %s: %q", key, v)
			}
		}
		storeMustBeClean(t, repo, "topic_auth_tokens.md", token)
	})

	t.Run("file-back explicit source", func(t *testing.T) {
		repo := t.TempDir()
		token, homePath := x46mSpans(t)
		hash := seedAskStore(t, repo)
		ask, err := Ask(AskRequest{
			RepoRoot: repo, Question: "how does token rotation work?",
			FileBackPage: map[string]any{
				"type": "topic", "domain": "auth", "slug": "rotation-summary",
				"body": "# Rotation summary\nTokens are rotated daily.\n",
				"source": map[string]any{
					"class": "external_article", "citation": map[string]any{"title": "keys " + token, "origin": homePath},
					"licence": "unknown", "source_hash": hash, "ingested_at": "2026-07-06",
				},
			},
			DecideFileBack: func(DistilledPage) bool { return true }, Now: fixedNow,
		})
		if err != nil {
			t.Fatalf("ask --file-back: %v", err)
		}
		if ask.FileBack == nil || ask.FileBack.Status != "written" {
			t.Fatalf("file-back did not write: %+v", ask.FileBack)
		}
		storeMustBeClean(t, repo, "topic_auth_rotation-summary.md", token, homePath)
	})

	t.Run("backlink onto a registered hash with no memory consumer", func(t *testing.T) {
		repo := t.TempDir()
		token, _ := x46mSpans(t)
		other := strings.Repeat("a", 64)
		if err := os.MkdirAll(Dir(repo), 0o755); err != nil {
			t.Fatal(err)
		}
		planted := map[string]any{other: map[string]any{"origin": "", "licence": "", "consumers": map[string]any{}}}
		if err := os.WriteFile(SourcesIndexPath(repo), []byte(SerializeRegistry(planted)), 0o644); err != nil {
			t.Fatal(err)
		}
		src := writeSource(t, repo, "notes.md", "Rotate tokens every 24 hours.\n")
		distiller := func(_ string, sourceBlock map[string]any) ([]map[string]any, error) {
			multi := map[string]any{
				"classes": []any{"external_article"},
				"sources": []any{
					map[string]any{
						"class": "external_article", "citation": map[string]any{"title": "ours"},
						"licence": "unknown", "source_hash": sourceBlock["source_hash"], "ingested_at": "2026-07-06",
					},
					map[string]any{
						"class": "external_article", "citation": map[string]any{"title": "theirs " + token},
						"licence": "unknown", "source_hash": other, "ingested_at": "2026-07-06",
					},
				},
			}
			return []map[string]any{{
				"type": "topic", "domain": "auth", "slug": "tokens", "body": "# Tokens\nRotate daily.\n", "source": multi,
			}}, nil
		}
		if _, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: distiller, Now: fixedNow}); err != nil {
			t.Fatalf("ingest: %v", err)
		}
		storeMustBeClean(t, repo, "topic_auth_tokens.md", token)
		reg, err := LoadRegistry(SourcesIndexPath(repo))
		if err != nil {
			t.Fatal(err)
		}
		entry, _ := reg[other].(map[string]any)
		consumers, _ := entry["consumers"].(map[string]any)
		if _, ok := consumers["memory"]; !ok {
			t.Fatalf("the backlink consumer was not created on %s…: %v", other[:8], entry)
		}
	})

	t.Run("fill-if-empty origin on the registry fast path", func(t *testing.T) {
		repo := t.TempDir()
		token, _ := x46mSpans(t)
		body := []byte("Rotate tokens every 24 hours.\n")
		clean := func(string) (FetchedSource, error) {
			return FetchedSource{FinalURL: "https://example.com/doc", Headers: map[string]string{"Content-Type": "text/plain"}, Body: body}, nil
		}
		res, err := Ingest(IngestRequest{RepoRoot: repo, Source: "https://example.com/doc", Distiller: fixedBodyDistiller(nil), Fetcher: clean, Now: fixedNow})
		if err != nil {
			t.Fatalf("first ingest: %v", err)
		}
		reg, err := LoadRegistry(SourcesIndexPath(repo))
		if err != nil {
			t.Fatal(err)
		}
		reg[res.ContentHash].(map[string]any)["origin"] = ""
		if err := os.WriteFile(SourcesIndexPath(repo), []byte(SerializeRegistry(reg)), 0o644); err != nil {
			t.Fatal(err)
		}
		dirty := func(string) (FetchedSource, error) {
			return FetchedSource{FinalURL: "https://example.com/doc?access_token=" + token, Headers: map[string]string{"Content-Type": "text/plain"}, Body: body}, nil
		}
		again, err := Ingest(IngestRequest{RepoRoot: repo, Source: "https://example.com/doc", Distiller: fixedBodyDistiller(nil), Fetcher: dirty, Now: fixedNow})
		if err != nil {
			t.Fatalf("re-ingest: %v", err)
		}
		if again.Status != "registry_only" {
			t.Fatalf("status = %q, want registry_only (the fast path)", again.Status)
		}
		raw := mustBeClean(t, "sources registry", SourcesIndexPath(repo), token)
		if !strings.Contains(raw, "example.com/doc") {
			t.Errorf("the emptied origin was not refilled: %s", raw)
		}
	})
}

// TestReingestNeverReJudgesCachedRegistryLeaves is the CONTROL for the diff
// walk: a leaf the registry already holds is not the write's to judge. A dirty
// cached citation must neither refuse the re-ingest (a legacy store would
// otherwise be locked out of the one verb that repairs it) nor be rewritten
// behind the operator's back — the lint (GHSA-xj89) is what reports it.
func TestReingestNeverReJudgesCachedRegistryLeaves(t *testing.T) {
	repo := t.TempDir()
	token, _ := x46mSpans(t)
	src := writeSource(t, repo, "notes.md", "Rotate tokens every 24 hours.\n")
	res, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: fixedBodyDistiller(nil), Now: fixedNow})
	if err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	reg, err := LoadRegistry(SourcesIndexPath(repo))
	if err != nil {
		t.Fatal(err)
	}
	dirtyTitle := "legacy " + token
	reg[res.ContentHash].(map[string]any)["consumers"].(map[string]any)["memory"].(map[string]any)["citation"].(map[string]any)["title"] = dirtyTitle
	if err := os.WriteFile(SourcesIndexPath(repo), []byte(SerializeRegistry(reg)), 0o644); err != nil {
		t.Fatal(err)
	}
	again, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: fixedBodyDistiller(nil), Now: fixedNow})
	if err != nil {
		t.Fatalf("a dirty cached citation locked the re-ingest out: %v", err)
	}
	if again.Status != "registry_only" {
		t.Fatalf("status = %q, want registry_only", again.Status)
	}
	after, err := LoadRegistry(SourcesIndexPath(repo))
	if err != nil {
		t.Fatal(err)
	}
	got := after[res.ContentHash].(map[string]any)["consumers"].(map[string]any)["memory"].(map[string]any)["citation"].(map[string]any)["title"]
	if got != dirtyTitle {
		t.Errorf("a cached registry leaf was re-judged on re-ingest: got %v", got)
	}
}

// TestFileBackDoesNotPropagateStoredResidue is the file-back clone half of
// GHSA-xj89-cc2c-wgwr: fileBackSource deep-copies citation and licence from
// every matched page onto the new page, so a store already carrying a secret
// in a citation propagated it on every ask --file-back. The clone now lands
// through the frontmatter walk, redacted.
func TestFileBackDoesNotPropagateStoredResidue(t *testing.T) {
	repo := t.TempDir()
	token, _ := x46mSpans(t)
	seedAskStore(t, repo)
	seeded := filepath.Join(Dir(repo), "topic_auth_tokens.md")
	raw, err := os.ReadFile(seeded)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "article.txt") {
		t.Fatalf("fixture drift: the seeded citation no longer names article.txt:\n%s", raw)
	}
	if err := os.WriteFile(seeded, []byte(strings.ReplaceAll(string(raw), "article.txt", "keys "+token)), 0o644); err != nil {
		t.Fatal(err)
	}
	ask, err := Ask(AskRequest{
		RepoRoot: repo, Question: "how does token rotation work?",
		FileBackPage: map[string]any{
			"type": "topic", "domain": "auth", "slug": "rotation-summary",
			"body": "# Rotation summary\nTokens are rotated daily.\n",
		},
		DecideFileBack: func(DistilledPage) bool { return true }, Now: fixedNow,
	})
	if err != nil {
		t.Fatalf("ask --file-back: %v", err)
	}
	if ask.FileBack == nil || ask.FileBack.Status != "written" {
		t.Fatalf("file-back did not write: %+v", ask.FileBack)
	}
	page := mustBeClean(t, "filed page", filepath.Join(Dir(repo), "topic_auth_rotation-summary.md"), token)
	if !strings.Contains(page, "citation") {
		t.Errorf("the cloned citation was dropped rather than redacted:\n%s", page)
	}
}
