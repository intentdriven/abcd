package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// BenchmarkQueryPages measures one `memory ask` question over a synthetic
// 500-page store — the per-question cost every page's frontmatter handling
// contributes, since QueryPages reads the whole store each time.
func BenchmarkQueryPages(b *testing.B) {
	repo := b.TempDir()
	mem := Dir(repo)
	if err := os.MkdirAll(mem, 0o755); err != nil {
		b.Fatal(err)
	}
	hash := strings.Repeat("a", 64)
	body := strings.Repeat("Rotate tokens on a daily cadence and record the rotation.\n", 40)
	for i := 0; i < 500; i++ {
		page := "---\ntopic: rotation\nsource:\n  class: session_memory\n  licence: MIT\n  source_hash: " + hash +
			"\n  ingested_at: 2026-08-19\n---\n\n# Token rotation " +
			fmt.Sprint(i) + "\n" + body
		name := fmt.Sprintf("topic_auth_rotation-%03d.md", i)
		if err := os.WriteFile(filepath.Join(mem, name), []byte(page), 0o644); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches, err := QueryPages(repo, "token rotation", 10)
		if err != nil {
			b.Fatal(err)
		}
		if len(matches) != 10 {
			b.Fatalf("matches = %d, want 10", len(matches))
		}
	}
}
