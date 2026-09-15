package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writer_leaf_key_test.go — the write-time leaf walk judged VALUES only.
//
// redactLeaves sanitised every string leaf a write introduced but never looked
// at the keys it walked, on the claim that keys are schema-fixed. They are not:
// validateSourceBlock rejects no unknown key in the source: block and
// checkBlockKey admits any identifier-shaped key — a `ghp_` token among them —
// so a host distiller's page JSON could carry a credential as a YAML KEY into a
// committed page, with only the MR001 read-side lint to notice afterwards.
//
// The token is the same FAKE fixture the value-side test uses: `ghp_` and forty
// literal 'A's.

func TestWriteRefusesASecretShapedKey(t *testing.T) {
	t.Run("frontmatter key", func(t *testing.T) {
		repo := t.TempDir()
		token, _ := x46mSpans(t)
		src := writeSource(t, repo, "notes.md", "Rotate tokens every 24 hours.\n")
		distiller := fixedBodyDistiller(func(block map[string]any) {
			block[token] = "an innocent-looking value"
		})
		_, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: distiller, Now: fixedNow})
		if err == nil {
			t.Fatalf("a token-shaped frontmatter KEY was accepted into the store")
		}
		if strings.Contains(err.Error(), token) {
			t.Errorf("the refusal echoes the secret it refused: %v", err)
		}
		if _, statErr := os.Stat(filepath.Join(Dir(repo), "topic_auth_tokens.md")); !os.IsNotExist(statErr) {
			t.Errorf("the page was written despite the refusal (%v)", statErr)
		}
	})

	t.Run("a clean key is still accepted", func(t *testing.T) {
		repo := t.TempDir()
		src := writeSource(t, repo, "notes.md", "Rotate tokens every 24 hours.\n")
		distiller := fixedBodyDistiller(func(block map[string]any) {
			block["reviewer_note"] = "checked by hand"
		})
		if _, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: distiller, Now: fixedNow}); err != nil {
			t.Fatalf("an ordinary extra key must still be written: %v", err)
		}
		raw, err := os.ReadFile(filepath.Join(Dir(repo), "topic_auth_tokens.md"))
		if err != nil {
			t.Fatalf("read page: %v", err)
		}
		if !strings.Contains(string(raw), "reviewer_note") {
			t.Errorf("the clean key was dropped:\n%s", raw)
		}
	})
}
