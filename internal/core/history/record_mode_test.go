package history

import (
	"os"
	"strings"
	"testing"
)

// The transcript store is private to the account (storeDirPerm), so every file
// it writes is owner-only too: a redacted record is still a verbatim account of
// the caller's sessions (iss-2609012029343438). The assertions compare the exact
// permission bits, which the store sets with fchmod on the open descriptor, so
// they hold whatever the process umask is.

func assertOwnerOnly(t *testing.T, path string) {
	t.Helper()
	fi, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("%s is mode %#o, want 0o600: the store is private and its records are owner-only", path, perm)
	}
}

// TestCaptureWritesTheRecordOwnerOnly pins the mode a captured record lands at.
func TestCaptureWritesTheRecordOwnerOnly(t *testing.T) {
	repoRoot, _ := setupStore(t)
	res, err := Capture(repoRoot, testRootSHA, []byte("assistant: hi\n"), CaptureMeta{SessionID: "sess-mode", Kind: "native"})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if !res.Wrote {
		t.Fatal("expected Wrote=true")
	}
	assertOwnerOnly(t, res.Record.Path)
}

// TestMigrateNarrowsTheRecordItRewrites is the one path by which a record an
// earlier binary wrote group- and world-readable is narrowed: a migration that
// rewrites it in place writes it owner-only, rather than carrying the wider
// mode forward.
func TestMigrateNarrowsTheRecordItRewrites(t *testing.T) {
	repoRoot, home := setupStore(t)
	const full = "5a9221e2-fa77-4be5-84d3-779199c449d7"
	path := planted(t, home, "20260101T000000.000000000Z-5a9221e2--agent-acf07c33.md",
		compositeRecord("5a9221e2--agent-acf07c33", full))
	// Planted at the mode the earlier binary wrote, set explicitly so the
	// starting point does not depend on the umask os.WriteFile applied.
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Migrate(testRootSHA, MigrateOptions{RepoRoot: repoRoot, Apply: true})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if len(res.Migrated) != 1 || !res.Migrated[0].Wrote {
		t.Fatalf("want the record rewritten, got %+v", res)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(onDisk), "session_id: "+full) {
		t.Fatalf("the record was not rewritten:\n%s", onDisk)
	}
	assertOwnerOnly(t, path)
}
