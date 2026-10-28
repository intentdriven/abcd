package ahoy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdrepo"
	"github.com/intentdriven/abcd/internal/core/vintage"
	"github.com/intentdriven/abcd/internal/gittest"
)

// sourceCheckout is a git checkout whose go.mod declares module.
func sourceCheckout(t *testing.T, module string) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "-C", dir, "init", "-q")
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
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
	defer swapCheckoutBuild(func() (string, string, bool) { return "", "", false })()

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

// localTierBuild puts a file where scripts/abcd builds checkout's abcd and
// returns its path.
func localTierBuild(t *testing.T, checkout string) string {
	t.Helper()
	p := filepath.Join(checkout, filepath.FromSlash(abcdrepo.CheckoutBuildRel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestTheScriptsBuildOfThisCheckoutIsCurrent is iss-2610241240133009: the
// build scripts/abcd has just made in the local tier of the checkout being
// installed is that tree as it stands, so it proceeds as a go run build does.
// The build of another checkout of the same module, run by that checkout's
// script path while standing here, is that tree's build, not this one's, and
// still refuses, as does any build the script did not just make.
func TestTheScriptsBuildOfThisCheckoutIsCurrent(t *testing.T) {
	const mod = "github.com/intentdriven/abcd"
	unknown := vintage.Current{Known: false}
	defer swapGoRunModule(func() (string, bool) { return "", false })()

	repo := sourceCheckout(t, mod)
	own := localTierBuild(t, repo)
	other := localTierBuild(t, sourceCheckout(t, mod))
	sub := filepath.Join(repo, "internal")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	defer swapCheckoutBuild(func() (string, string, bool) { return mod, own, true })()
	for _, cwd := range []string{repo, sub} {
		if reason := staleBinaryRefusal(unknown, cwd); reason != "" {
			t.Errorf("the script's build of the checkout being installed was refused from %s: %s", cwd, reason)
		}
	}

	swapCheckoutBuild(func() (string, string, bool) { return mod, other, true })
	if reason := staleBinaryRefusal(unknown, repo); reason == "" {
		t.Error("another checkout's local-tier build of the same module was accepted; it is that tree's build, not this one's")
	}

	swapCheckoutBuild(func() (string, string, bool) { return "example.com/someone/else", own, true })
	if reason := staleBinaryRefusal(unknown, repo); reason == "" {
		t.Error("a local-tier build of another module was accepted")
	}

	swapCheckoutBuild(func() (string, string, bool) { return "", "", false })
	if reason := staleBinaryRefusal(unknown, repo); reason == "" {
		t.Error("an unstamped binary the script did not just build was accepted")
	}
}

// TestAnUnknownVintageInASourceCheckoutIsSentToTheScript: in a source checkout
// of abcd the remedy for an unstamped binary is the checkout's scripts/abcd,
// because `make build` in a linked worktree makes another unstamped binary.
// Elsewhere the remedy stays the rebuild.
func TestAnUnknownVintageInASourceCheckoutIsSentToTheScript(t *testing.T) {
	const mod = "github.com/intentdriven/abcd"
	unknown := vintage.Current{Known: false}
	defer swapGoRunModule(func() (string, bool) { return "", false })()
	defer swapCheckoutBuild(func() (string, string, bool) { return "", "", false })()

	src := sourceCheckout(t, mod)
	mainGo := filepath.Join(src, "cmd", "abcd", "main.go")
	if err := os.MkdirAll(filepath.Dir(mainGo), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainGo, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if reason := staleBinaryRefusal(unknown, src); !strings.Contains(reason, "`scripts/abcd ahoy install`") || strings.Contains(reason, "make build") {
		t.Errorf("in a source checkout the refusal should send the person to scripts/abcd, not make build: %s", reason)
	}
	if reason := staleBinaryRefusal(unknown, sourceCheckout(t, "example.com/someone/else")); !strings.Contains(reason, "make build") {
		t.Errorf("outside a source checkout the refusal should name the rebuild: %s", reason)
	}
}
