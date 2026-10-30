package ahoy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdrepo"
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

// TestTheCheckoutsOwnBuildIsInNoPluginRoot: the build scripts/abcd.sh makes sits in
// the checkout's local tier, under a directory with the plugin layout that holds
// no plugin binary. It is placed in no root, as go run's build never was, so a
// plain terminal resolves the plugin the way go run did instead of reading the
// checkout as one with its binary missing (iss-2610241240133009).
func TestTheCheckoutsOwnBuildIsInNoPluginRoot(t *testing.T) {
	checkout := t.TempDir()
	if err := os.MkdirAll(filepath.Join(checkout, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(checkout, filepath.FromSlash(abcdrepo.CheckoutBuildRel))
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, ok := PluginRootContaining(exe); ok {
		t.Errorf("the checkout's own build was placed in plugin root %q", got)
	}
	t.Setenv("ABCD_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("HOME", t.TempDir())
	prev := osExecutable
	osExecutable = func() (string, error) { return exe, nil }
	t.Cleanup(func() { osExecutable = prev })
	if got, ok := resolvePluginRoot(); ok && resolvePath(got) == resolvePath(checkout) {
		t.Errorf("a plain terminal resolved the checkout as the plugin root: %q", got)
	}
}
