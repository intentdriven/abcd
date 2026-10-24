package ahoy

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/vintage"
)

// sourceCheckout is a git checkout whose go.mod declares module.
func sourceCheckout(t *testing.T, module string) string {
	t.Helper()
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+module+"\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestGoRunOfThisCheckoutIsCurrent is iss-2610240824047504: in a source
// checkout of abcd, `go run ./cmd/abcd ahoy install` is the one sanctioned way
// to run abcd, and go run never stamps a revision, so the install used to
// refuse it as an unknown vintage on every run. A go run build of the module
// the checkout being installed declares is built from that tree as it stands,
// so it is current; any other unstamped binary still refuses.
func TestGoRunOfThisCheckoutIsCurrent(t *testing.T) {
	const mod = "github.com/intentdriven/abcd"
	unknown := vintage.Current{Known: false}
	restore := swapGoRunModule(func() (string, bool) { return mod, true })
	defer restore()

	repo := sourceCheckout(t, mod)
	if reason := staleBinaryRefusal(unknown, repo); reason != "" {
		t.Errorf("a go run build of the checkout being installed was refused: %s", reason)
	}
	sub := filepath.Join(repo, "internal")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if reason := staleBinaryRefusal(unknown, sub); reason != "" {
		t.Errorf("a go run build from a folder inside the checkout was refused: %s", reason)
	}

	other := sourceCheckout(t, "example.com/someone/else")
	if reason := staleBinaryRefusal(unknown, other); reason == "" {
		t.Error("a go run build of abcd installing into another module's checkout was accepted; it is not that tree's build")
	}

	swapGoRunModule(func() (string, bool) { return "", false })
	if reason := staleBinaryRefusal(unknown, repo); reason == "" {
		t.Error("an unstamped binary that is not a go run build was accepted")
	}
}
