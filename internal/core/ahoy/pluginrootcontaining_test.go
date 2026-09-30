package ahoy

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPluginRootContaining: the executable-ancestor rung on its own names the
// root a binary sits in, whether beside hooks/ or below it, through a symlinked
// route too, and names nothing for a binary inside no plugin root
// (iss-2609020113012227).
func TestPluginRootContaining(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "0e22abfd6739")
	if err := os.MkdirAll(filepath.Join(root, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	loose := filepath.Join(base, "loose")
	if err := os.MkdirAll(loose, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	want := resolvePath(root)

	for _, exe := range []string{
		filepath.Join(root, "abcd"),
		filepath.Join(root, "bin", "abcd-darwin-arm64"),
		filepath.Join(link, "abcd"),
	} {
		got, ok := PluginRootContaining(exe)
		if !ok || resolvePath(got) != want {
			t.Errorf("PluginRootContaining(%s) = %q, %v; want %q", exe, got, ok, want)
		}
	}
	if got, ok := PluginRootContaining(filepath.Join(loose, "abcd")); ok {
		t.Errorf("a binary inside no plugin root named %q", got)
	}
}
