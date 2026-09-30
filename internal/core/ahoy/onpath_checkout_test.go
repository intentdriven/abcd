package ahoy

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeTool writes an executable named tool into dir and returns dir.
func fakeTool(t *testing.T, dir, tool string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, tool), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestAProgramInsideTheCheckoutIsNotInstalled pins that presence agrees with
// what abcd would run: the installer refuses a program PATH resolves inside
// the checkout as repository content, so ahoy must not count it as installed
// either, whether PATH names the in-checkout directory itself or a directory
// outside that is a symlink into it. A program outside the checkout counts.
func TestAProgramInsideTheCheckoutIsNotInstalled(t *testing.T) {
	cases := map[string]func(repo string) string{
		"a PATH entry inside the checkout": func(repo string) string {
			return fakeTool(t, filepath.Join(repo, "tools", "bin"), "gitleaks")
		},
		"a PATH entry outside that links into the checkout": func(repo string) string {
			inside := fakeTool(t, filepath.Join(repo, "tools", "bin"), "gitleaks")
			link := filepath.Join(t.TempDir(), "bin")
			if err := os.Symlink(inside, link); err != nil {
				t.Fatal(err)
			}
			return link
		},
		"a program outside that links into the checkout": func(repo string) string {
			inside := fakeTool(t, filepath.Join(repo, "tools", "bin"), "gitleaks")
			out := t.TempDir()
			if err := os.Symlink(filepath.Join(inside, "gitleaks"), filepath.Join(out, "gitleaks")); err != nil {
				t.Fatal(err)
			}
			return out
		},
	}
	for name, pathEntry := range cases {
		t.Run(name, func(t *testing.T) {
			repo := t.TempDir()
			t.Setenv("PATH", pathEntry(repo))
			if onPath(repo, "gitleaks") {
				t.Error("onPath counts a gitleaks inside the checkout as installed")
			}
			depGap(t, detectDependencies(repo), "deps.gitleaks_missing")
		})
	}

	repo := t.TempDir()
	t.Setenv("PATH", fakeTool(t, t.TempDir(), "gitleaks"))
	if !onPath(repo, "gitleaks") {
		t.Fatal("onPath does not count a gitleaks outside the checkout")
	}
	for _, g := range detectDependencies(repo) {
		if g.ID == "deps.gitleaks_missing" {
			t.Fatalf("a gitleaks outside the checkout is reported missing: %+v", g)
		}
	}
}
