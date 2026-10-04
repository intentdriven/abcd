package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestFirstRunCreatesOnlyTheNoindexHome is D1 of itd-2610030720038073: on an
// account with no abcd home, a first install followed by a first run that
// writes a run log leaves ~/.abcd.noindex holding that log, and no ~/.abcd.
// Every home writer reaches the folder through the resolver, which the D3
// boundary test holds, so no writer can create the old name.
func TestFirstRunCreatesOnlyTheNoindexHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	pluginRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(pluginRoot, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "hooks", "hooks.json"), []byte(validHooksJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "abcd"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ABCD_PLUGIN_ROOT", pluginRoot)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("CLAUDE_PLUGIN_DATA", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("ABCD_BIN_TARGET", filepath.Join(t.TempDir(), "bin", "abcd"))
	var kept []string
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if fi, err := os.Lstat(filepath.Join(dir, "abcd")); err == nil && !fi.IsDir() {
			continue
		}
		kept = append(kept, dir)
	}
	t.Setenv("PATH", strings.Join(kept, string(os.PathListSeparator)))

	repo := gittest.NewRepo(t)
	repo.Write("README.md", "fixture\n")
	repo.Commit("init")
	sha := repo.Git("rev-list", "--max-parents=0", "HEAD")
	t.Chdir(repo.Root())

	for _, args := range [][]string{
		{"ahoy", "install", "--yes", "--adopt", "--visibility", "private", "--docs-target", "agents_md",
			"--oracle-backend", "host-delegated", "--scan-deep", "false"},
		{"implement", "join", "--session", "alpha", "--role", "first"},
		{"implement", "log", "lane_open", "--session", "alpha", "--field", "lane=one", "--field", "record=itd-1"},
	} {
		var out, errOut bytes.Buffer
		if code := run(args, strings.NewReader(""), &out, &errOut); code != 0 {
			t.Fatalf("abcd %s exited %d\nstdout: %s\nstderr: %s", strings.Join(args, " "), code, out.String(), errOut.String())
		}
	}

	logs, err := filepath.Glob(abcdhome.Path(home, "runs", sha, "*.jsonl"))
	if err != nil || len(logs) == 0 {
		t.Fatalf("no run log under %s (%v)", abcdhome.Path(home, "runs", sha), err)
	}
	if filepath.Base(abcdhome.Path(home)) != ".abcd.noindex" {
		t.Fatalf("the home is %q, want .abcd.noindex", filepath.Base(abcdhome.Path(home)))
	}
	if _, err := os.Lstat(filepath.Join(home, ".abcd")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a first run created ~/.abcd beside the new home: %v", err)
	}
	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".abcd") && e.Name() != ".abcd.noindex" {
			t.Errorf("a first run created %q in the home", e.Name())
		}
	}
}
