package ahoy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// placeholderToken is a clearly fake credential, distinctive enough that no
// other fixture bytes can pass for it. Never a real one.
const placeholderToken = "ghp_EXAMPLE0000000000000000000000000000"

// TestScrubRemoteUserinfo pins the one derivation rule for the origin URL
// (GHSA-qc3w-8pv5-crc3): a credential in the userinfo never survives, while an
// SSH login name — which is a route, not a secret — does.
func TestScrubRemoteUserinfo(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"https://ci-bot:" + placeholderToken + "@github.com/owner/repo.git", "https://github.com/owner/repo.git"},
		{"https://" + placeholderToken + "@github.com/owner/repo.git", "https://github.com/owner/repo.git"},
		{"http://user:secret@example.com/o/r", "http://example.com/o/r"},
		{"https://github.com/owner/repo.git", "https://github.com/owner/repo.git"},
		{"ssh://git@github.com/owner/repo.git", "ssh://git@github.com/owner/repo.git"},
		{"ssh://git:secret@github.com/owner/repo.git", "ssh://github.com/owner/repo.git"},
		{"git@github.com:owner/repo.git", "git@github.com:owner/repo.git"},
		{"user:secret@example.com:owner/repo.git", "example.com:owner/repo.git"},
		// A password may itself carry a slash; the host:path colon after the
		// userinfo is what makes this scp-like, so it is still a credential.
		{"user:pa/ss@example.com:owner/repo.git", "example.com:owner/repo.git"},
		// No userinfo at all: the colon is the host:path separator and the @ is
		// inside the PATH. Splitting on the @ here would rewrite a repository's
		// address into "po.git" and lose the host.
		{"example.com:owner/re@po.git", "example.com:owner/re@po.git"},
		{"git@example.com:owner/re@po.git", "git@example.com:owner/re@po.git"},
		{"/srv/git/repo.git", "/srv/git/repo.git"},
		{"", ""},
	} {
		if got := scrubRemoteUserinfo(tc.in); got != tc.want {
			t.Errorf("scrubRemoteUserinfo(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestDeriveIdentityStripsRemoteUserinfo: the identity every sink reads is
// derived once, so the scrub at that site is the whole fix.
func TestDeriveIdentityStripsRemoteUserinfo(t *testing.T) {
	setupHermetic(t)
	r := gittest.NewRepo(t)
	r.Git("remote", "add", "origin", "https://ci-bot:"+placeholderToken+"@github.com/owner/repo.git")
	id := deriveIdentity(r.Root())
	if id.Github != "https://github.com/owner/repo.git" {
		t.Fatalf("Github = %q, want the origin URL without its userinfo", id.Github)
	}
}

// TestInstallPersistsNoRemoteUserinfo drives a credentialed origin through a
// real install into an isolated home and reads back the two files the registry
// writes: neither index.json nor the per-repo meta.json may carry the credential.
func TestInstallPersistsNoRemoteUserinfo(t *testing.T) {
	home, _ := setupHermetic(t)
	r := gittest.NewRepo(t)
	r.Commit("root")
	r.Git("remote", "add", "origin", "https://ci-bot:"+placeholderToken+"@github.com/owner/repo.git")

	res, err := Install(r.Root(), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status == "refused" {
		t.Fatalf("install refused: %v", res.Notes)
	}
	sha := deriveIdentity(r.Root()).RootSHA
	if sha == "" {
		t.Fatal("fixture has no root commit")
	}
	for _, rel := range []string{
		filepath.Join(".abcd", "history", "index.json"),
		filepath.Join(".abcd", "history", sha, "meta.json"),
	} {
		data, err := os.ReadFile(filepath.Join(home, rel))
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		if strings.Contains(string(data), placeholderToken) {
			t.Fatalf("%s persisted the origin credential:\n%s", rel, data)
		}
		if !strings.Contains(string(data), "https://github.com/owner/repo.git") {
			t.Fatalf("%s does not record the scrubbed origin URL:\n%s", rel, data)
		}
	}
}

// TestInstallScrubsALegacyCredentialFromTheHistoryStore is the other half of
// GHSA-qc3w-8pv5-crc3, and the claim its first fix made but did not keep: a
// credential that entered the store BEFORE the derivation-site scrub existed is
// already at rest, and nothing was going back for it. meta.json is written once
// and never revisited (`!fileExists(metaPath)`), and index.json is only rewritten
// when there is registration work to do — so the legacy entry survived every
// later install.
//
// The fixture plants both shapes the store can hold: a credential in ANOTHER
// repo's index entry, which no registration of THIS repo would touch, and one in
// this repo's own meta.json. The next install must leave neither behind.
func TestInstallScrubsALegacyCredentialFromTheHistoryStore(t *testing.T) {
	home, _ := setupHermetic(t)
	r := gittest.NewRepo(t)
	r.Commit("root")
	r.Git("remote", "add", "origin", "https://github.com/owner/repo.git")

	if res, err := Install(r.Root(), installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	} else if res.Status != "clean" {
		t.Fatalf("precondition: first install status = %q (remaining %v), want clean", res.Status, res.Remaining)
	}
	sha := deriveIdentity(r.Root()).RootSHA
	if sha == "" {
		t.Fatal("fixture has no root commit")
	}
	credentialed := "https://ci-bot:" + placeholderToken + "@github.com/owner/legacy.git"

	// A sibling repo's entry, credentialed. Registering THIS repo never reads it.
	indexPath := filepath.Join(home, ".abcd", "history", "index.json")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("reading the freshly bootstrapped index: %v", err)
	}
	var idx historyIndex
	if err := json.Unmarshal(raw, &idx); err != nil {
		t.Fatal(err)
	}
	idx.Repos = append(idx.Repos, historyRepo{
		RootCommit: "0000000000000000000000000000000000000001",
		Name:       "legacy", Github: credentialed,
		Path: filepath.Join(home, "legacy"), Status: "active",
	})
	if err := writeJSON(indexPath, idx); err != nil {
		t.Fatal(err)
	}

	// This repo's own meta.json, credentialed — the write-once file.
	metaPath := filepath.Join(home, ".abcd", "history", sha, "meta.json")
	meta := map[string]any{
		"root_commit": sha, "name": "repo", "github": credentialed,
		"corpus": map[string]any{"transcripts": "transcripts/"},
	}
	if err := writeJSON(metaPath, meta); err != nil {
		t.Fatal(err)
	}

	// Detection must see it: without a gap the install short-circuits as
	// already_up_to_date and never reaches the step that would heal anything.
	det, err := Detect(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if !gapIDSet(det.Gaps)["history.credential_at_rest"] {
		t.Fatalf("no history.credential_at_rest gap for a store holding a credential; gaps: %v", gapIDs(det.Gaps))
	}

	if _, err := Install(r.Root(), installOpts(), RefusingPrompter{}); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{indexPath, metaPath} {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if strings.Contains(string(data), placeholderToken) {
			t.Fatalf("%s still holds the legacy credential:\n%s", p, data)
		}
		if !strings.Contains(string(data), "github.com/owner/") {
			t.Fatalf("%s lost the remote URL instead of scrubbing it:\n%s", p, data)
		}
	}
	// And the heal is complete: a re-detect raises the gap no more.
	det2, err := Detect(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	if gapIDSet(det2.Gaps)["history.credential_at_rest"] {
		t.Fatalf("history.credential_at_rest survived the install that was supposed to close it")
	}
}

// TestLoadHistoryIndexScrubsAsItReads: the index is loaded by every writer of
// it, so scrubbing on the way in is what makes any rewrite drop a credential —
// including one in an entry the writer had no other reason to touch. It is also
// what keeps a renderer from printing a credential it merely read.
func TestLoadHistoryIndexScrubsAsItReads(t *testing.T) {
	home, _ := setupHermetic(t)
	root := filepath.Join(home, ".abcd", "history")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	idx := historyIndex{Schema: 1, Description: historyIndexDescription, Repos: []historyRepo{{
		RootCommit: "0000000000000000000000000000000000000001",
		Name:       "legacy",
		Github:     "https://ci-bot:" + placeholderToken + "@github.com/owner/legacy.git",
		Status:     "active",
	}}}
	if err := writeJSON(filepath.Join(root, "index.json"), idx); err != nil {
		t.Fatal(err)
	}
	got, err := loadHistoryIndex()
	if err != nil || got == nil {
		t.Fatalf("loadHistoryIndex: %v", err)
	}
	if want := "https://github.com/owner/legacy.git"; got.Repos[0].Github != want {
		t.Fatalf("Github = %q, want %q", got.Repos[0].Github, want)
	}
}
