package memory

import (
	"os"
	"path/filepath"
	"testing"
)

// writer_backlink_test.go — the registry's page back-links are IDENTIFIERS the
// store resolves, not acquired prose, and the leaf walk must never rewrite one.
//
// GHSA-x46m-mw9h-5jwj routed the whole registry through redactLeaves. A page
// filename lands in `consumers.<consumer>.pages[]`, and a perfectly ordinary
// slug can match a warn-severity network pattern — `migrating-off-the-nas`
// carries `off-the-nas`, which net_device_hostname matches on the hyphen
// boundary. The registry entry was rewritten while the file on disk kept its
// real name, so the next write's pruneOrphans found no live back-link for it
// and DELETED it, reporting success. Data loss, silently.
//
// The list walk paired a target element with the baseline element at the same
// INDEX, so a list that gains a front element re-judged every shifted element —
// contradicting the walk's own no-lockout contract that a leaf the store
// already holds is never re-judged.

// nasSlug is an ordinary English slug that a device-hostname pattern matches.
const nasSlug = "migrating-off-the-nas"

const nasPage = "topic_home_" + nasSlug + ".md"

func nasDistiller(_ string, sourceBlock map[string]any) ([]map[string]any, error) {
	return []map[string]any{{
		"type": "topic", "domain": "home", "slug": nasSlug,
		"body": "# Storage\nThe array moved to the cloud.\n", "source": sourceBlock,
	}}, nil
}

func registryBackLinks(t *testing.T, repo, contentHash string) []string {
	t.Helper()
	registry, err := LoadRegistry(SourcesIndexPath(repo))
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	entry, _ := registry[contentHash].(map[string]any)
	consumers, _ := entry["consumers"].(map[string]any)
	memc, _ := consumers["memory"].(map[string]any)
	return anyToStrings(memc["pages"])
}

func TestWriteNeverRewritesRegistryBackLinks(t *testing.T) {
	t.Run("a device-shaped slug keeps its registry back-link", func(t *testing.T) {
		repo := t.TempDir()
		src := writeSource(t, repo, "storage.md", "The array moved to the cloud.\n")
		res, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: nasDistiller, Now: fixedNow})
		if err != nil {
			t.Fatalf("ingest: %v", err)
		}
		if len(res.Pages) != 1 || res.Pages[0] != nasPage {
			t.Fatalf("the ingest wrote %v, want [%s]", res.Pages, nasPage)
		}
		if _, serr := os.Stat(filepath.Join(Dir(repo), nasPage)); serr != nil {
			t.Fatalf("the page was not written under its own name: %v", serr)
		}
		links := registryBackLinks(t, repo, res.ContentHash)
		if len(links) != 1 || links[0] != nasPage {
			t.Fatalf("the registry back-link %v does not name the written page %q — pruneOrphans will delete it", links, nasPage)
		}
	})

	t.Run("an unrelated second ingest prunes nothing", func(t *testing.T) {
		repo := t.TempDir()
		first := writeSource(t, repo, "storage.md", "The array moved to the cloud.\n")
		if _, err := Ingest(IngestRequest{RepoRoot: repo, Source: first, Distiller: nasDistiller, Now: fixedNow}); err != nil {
			t.Fatalf("first ingest: %v", err)
		}
		second := writeSource(t, repo, "tokens.md", "Rotate tokens every 24 hours.\n")
		res, err := Ingest(IngestRequest{RepoRoot: repo, Source: second, Distiller: fixedBodyDistiller(nil), Now: fixedNow})
		if err != nil {
			t.Fatalf("second ingest: %v", err)
		}
		if res.WriteReport != nil && len(res.WriteReport.Pruned) > 0 {
			t.Errorf("an unrelated ingest pruned %v", res.WriteReport.Pruned)
		}
		if _, serr := os.Stat(filepath.Join(Dir(repo), nasPage)); serr != nil {
			t.Errorf("the first ingest's page was deleted by an unrelated ingest: %v", serr)
		}
	})

	t.Run("a stored leaf in a reordered list is not re-judged", func(t *testing.T) {
		repo := t.TempDir()
		_, homePath := x46mSpans(t)
		r, err := newStoreRedactor(repo)
		if err != nil {
			t.Fatalf("redactor: %v", err)
		}
		stored := "notes at " + homePath
		baseline := map[string]any{"recall": []any{stored, "keep me"}}
		target := map[string]any{"recall": []any{"a fresh entry", stored, "keep me"}}
		if err := r.redactLeaves(baseline, target, "page.md"); err != nil {
			t.Fatalf("walk: %v", err)
		}
		got := anyToStrings(target["recall"].([]any))
		if len(got) != 3 || got[1] != stored {
			t.Errorf("a leaf the store already held was re-judged after the list shifted: %q -> %q", stored, got)
		}
	})
}
