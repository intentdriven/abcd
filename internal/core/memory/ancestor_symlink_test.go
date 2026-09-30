package memory

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMemoryStoreDirSymlinkRefused is the attack-input test for the store
// DIRECTORY symlink (GHSA-72rp-qxm2-r8vq). The write path (validatedMemoryDir)
// already refuses a symlinked `.abcd/memory`; the read/lint/coverage/bare paths
// historically did not — they joined the raw Dir() path and walked/read through
// a committed directory symlink, disclosing an out-of-repo tree and (worse)
// writing `.coverage_index.json` INTO the symlink target.
//
// The store directory itself is committed as a git mode-120000 symlink to a
// tree outside the repo. The read/lint entry points must refuse it, never walk
// it, and never write into the target.
func TestMemoryStoreDirSymlinkRefused(t *testing.T) {
	repoRoot := t.TempDir()
	outside := t.TempDir() // the symlink target — an out-of-repo tree.

	if err := os.MkdirAll(filepath.Join(repoRoot, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Commit `.abcd/memory` as a DIRECTORY symlink pointing outside the repo.
	memLink := filepath.Join(repoRoot, ".abcd", "memory")
	if err := os.Symlink(outside, memLink); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	// A typed memory page under the target that ask/lint would otherwise read.
	const secret = "SECRET_TOKEN_exfiltrated_abc123"
	page := "---\nsource:\n  class: session_memory\n---\nleak of confidential " + secret + " data\n"
	if err := os.WriteFile(filepath.Join(outside, "note_secret_leak.md"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}

	coverageInTarget := filepath.Join(outside, ".coverage_index.json")

	// (a) Lint must refuse the symlinked store and must NOT write the coverage
	// index into the out-of-repo target.
	if _, err := Lint(LintRequest{RepoRoot: repoRoot}); err == nil {
		t.Error("Lint followed a symlinked .abcd/memory store; a committed directory symlink must be refused")
	} else {
		var unsafe *UnsafeStorePathError
		if !errors.As(err, &unsafe) {
			t.Errorf("Lint returned %T (%v); want a typed *UnsafeStorePathError for the symlinked store", err, err)
		}
	}
	if _, err := os.Lstat(coverageInTarget); err == nil {
		t.Errorf("Lint wrote %s INTO the symlink target — a write escaped the repo", coverageInTarget)
	}

	// (b) A read (queryPages / Ask) must not disclose the out-of-repo page.
	matches, err := queryPages(repoRoot, "secret leak", 5)
	if err == nil {
		t.Error("QueryPages followed a symlinked .abcd/memory store; the directory symlink must be refused")
	}
	for _, m := range matches {
		if strings.Contains(m.Body, secret) {
			t.Fatalf("QueryPages disclosed an out-of-repo page through the store symlink: %q", m.Filename)
		}
	}

	res, err := Ask(AskRequest{RepoRoot: repoRoot, Question: "secret leak"})
	if err == nil {
		if strings.Contains(res.Answer, secret) {
			t.Fatalf("Ask disclosed an out-of-repo secret through the store symlink: %q", res.Answer)
		}
		t.Error("Ask followed a symlinked .abcd/memory store; the directory symlink must be refused")
	}

	// (c) Bare status must refuse rather than crawl the symlink target.
	if _, err := Bare(repoRoot); err == nil {
		t.Error("Bare followed a symlinked .abcd/memory store; the directory symlink must be refused")
	}
}
