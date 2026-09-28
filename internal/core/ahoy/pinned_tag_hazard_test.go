package ahoy

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadPinnedTagRefusesAHazardousDataDir is iss-2609020630242279: the
// release tag the vintage line reports, and staleBinaryRefusal reads, came from
// CLAUDE_PLUGIN_DATA's cache record without the shape check every other reader
// of that directory applies. A world-writable, relative or in-repository data
// directory is one the harness never produces, so its record is not believed.
func TestReadPinnedTagRefusesAHazardousDataDir(t *testing.T) {
	_, pluginRoot := setupHermetic(t)
	repo := t.TempDir()
	plant := func(t *testing.T, dir string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(dir, "cache"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "cache", "binary-meta"), []byte("release_tag=v9.9.9\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("control", func(t *testing.T) {
		dir := t.TempDir()
		plant(t, dir)
		t.Setenv("CLAUDE_PLUGIN_DATA", dir)
		if got := readPinnedTag(pluginRoot, repo); got != "v9.9.9" {
			t.Fatalf("a well-shaped data dir's tag = %q, want v9.9.9", got)
		}
	})
	t.Run("world-writable", func(t *testing.T) {
		dir := t.TempDir()
		plant(t, dir)
		if err := os.Chmod(dir, 0o777); err != nil {
			t.Fatal(err)
		}
		t.Setenv("CLAUDE_PLUGIN_DATA", dir)
		if got := readPinnedTag(pluginRoot, repo); got != "" {
			t.Fatalf("a world-writable data dir supplied the tag %q", got)
		}
	})
	t.Run("inside-the-repo", func(t *testing.T) {
		dir := filepath.Join(repo, "data")
		plant(t, dir)
		t.Setenv("CLAUDE_PLUGIN_DATA", dir)
		if got := readPinnedTag(pluginRoot, repo); got != "" {
			t.Fatalf("an in-repository data dir supplied the tag %q", got)
		}
	})
	t.Run("relative", func(t *testing.T) {
		t.Chdir(repo)
		plant(t, filepath.Join(repo, "rel"))
		t.Setenv("CLAUDE_PLUGIN_DATA", "rel")
		if got := readPinnedTag(pluginRoot, t.TempDir()); got != "" {
			t.Fatalf("a relative data dir supplied the tag %q", got)
		}
	})
}
