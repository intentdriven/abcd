package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// The memory store's literal home backstop sweeps the home in the spelling the
// kernel reports for a linked home, whose target is not named after the user,
// so the pattern detector has no username to find there either
// (iss-2609290055092630).
func TestStoreRedactorSweepsTheResolvedSpellingOfALinkedHome(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "Volumes", "Data", "u1042")
	for _, d := range []string{target, filepath.Join(base, "Users")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	home := filepath.Join(base, "Users", "maya")
	if err := os.Symlink(target, home); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv(abcdhome.StageHomeEnv, "")
	r, err := newStoreRedactor(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := r.redactText("Scratch lives at "+resolved+"/scratch on this box.\n", "page")
	if err != nil {
		t.Fatalf("redactText: %v", err)
	}
	if strings.Contains(got, resolved) || !strings.Contains(got, "~/scratch") {
		t.Errorf("the resolved spelling of the home was not swept to ~:\n%s", got)
	}
}
