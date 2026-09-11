package history

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// virginHome points HOME at an empty temp dir — no ~/.abcd, no history root, no
// per-repo dir — which is the state iss-95 describes: a machine where
// `abcd ahoy install` has never run.
func virginHome(t *testing.T) (repoRoot, home string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	return t.TempDir(), home
}

// TestCaptureOnAMachineThatNeverInstalled is iss-95's detector: the store's
// user-level default must exist BY CONSTRUCTION, so a machine where
// `abcd ahoy install` never ran still captures. Before the store created
// itself, Capture returned "not a real directory (absent or symlink); run
// `abcd ahoy install`", the SessionEnd hook logged it, exited 0, and nothing
// was ever stored.
func TestCaptureOnAMachineThatNeverInstalled(t *testing.T) {
	repoRoot, home := virginHome(t)

	res, err := Capture(repoRoot, testRootSHA, "sess-neverinstalled", []byte("assistant: hi\n"), "native")
	if err != nil {
		t.Fatalf("Capture on an uninstalled machine must bootstrap the store, got: %v", err)
	}
	if !res.Wrote {
		t.Fatal("expected the transcript to be stored")
	}
	want := filepath.Join(home, ".abcd", "transcripts", testRootSHA, "records")
	if got := filepath.Dir(res.Record.Path); got != want {
		t.Errorf("record stored at %q, want it under the user-level default %q", got, want)
	}
	if _, err := os.Stat(res.Record.Path); err != nil {
		t.Errorf("stored record is not on disk: %v", err)
	}
}

// TestStageOnAMachineThatNeverInstalled covers the same gap on the hook's real
// path: SessionEnd stages rather than captures, so staging is the call that
// actually silently stored nothing.
func TestStageOnAMachineThatNeverInstalled(t *testing.T) {
	repoRoot, home := virginHome(t)

	res, err := Stage(repoRoot, testRootSHA, "sess-staged", []byte("assistant: hi\n"))
	if err != nil {
		t.Fatalf("Stage on an uninstalled machine must bootstrap the store, got: %v", err)
	}
	if !res.Wrote {
		t.Fatal("expected the transcript to be staged")
	}
	want := filepath.Join(home, ".abcd", "transcripts", testRootSHA, "staging")
	if got := filepath.Dir(res.Staged.Path); got != want {
		t.Errorf("staged at %q, want %q", got, want)
	}
}

// TestTwoReposStayDistinguishableInOneStore is the keying detector. One
// user-level store now holds every repo's transcripts, so the root-commit SHA
// has to keep them apart: each repo's list must return its own transcript and
// only its own.
func TestTwoReposStayDistinguishableInOneStore(t *testing.T) {
	repoA, _ := virginHome(t)
	repoB := t.TempDir()
	shaA := testRootSHA
	shaB := strings.Repeat("c", 40)

	if _, err := Capture(repoA, shaA, "sess-a", []byte("assistant: alpha\n"), "native"); err != nil {
		t.Fatalf("capture into repo A: %v", err)
	}
	if _, err := Capture(repoB, shaB, "sess-b", []byte("assistant: beta\n"), "native"); err != nil {
		t.Fatalf("capture into repo B: %v", err)
	}

	for _, c := range []struct{ name, repo, sha, session string }{
		{"A", repoA, shaA, "sess-a"},
		{"B", repoB, shaB, "sess-b"},
	} {
		recs, err := List(c.repo, c.sha)
		if err != nil {
			t.Fatalf("List for repo %s: %v", c.name, err)
		}
		if len(recs) != 1 {
			t.Fatalf("repo %s sees %d record(s) in the shared store, want exactly its own 1", c.name, len(recs))
		}
		if recs[0].SessionID != c.session {
			t.Errorf("repo %s sees session %q, want %q — the store is not keyed per repo", c.name, recs[0].SessionID, c.session)
		}
		if _, _, err := Read(c.repo, c.sha, c.session); err != nil {
			t.Errorf("Read for repo %s: %v", c.name, err)
		}
	}
}

// TestReadVerbsWorkAgainstTheRelocatedStore walks the four read/finish verbs
// `abcd history` exposes — list, show, staged, drain — against the new store.
func TestReadVerbsWorkAgainstTheRelocatedStore(t *testing.T) {
	repoRoot, _ := virginHome(t)

	if _, err := Stage(repoRoot, testRootSHA, "sess-pending", []byte("assistant: pending\n")); err != nil {
		t.Fatalf("Stage: %v", err)
	}
	staged, err := ListStaged(repoRoot, testRootSHA)
	if err != nil {
		t.Fatalf("ListStaged: %v", err)
	}
	if len(staged) != 1 {
		t.Fatalf("ListStaged returned %d entries, want 1", len(staged))
	}

	dr, err := Drain(repoRoot, testRootSHA, 0)
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if len(dr.Captured) != 1 || len(dr.Failed) != 0 {
		t.Fatalf("Drain captured=%d failed=%d, want 1/0", len(dr.Captured), len(dr.Failed))
	}
	recs, err := List(repoRoot, testRootSHA)
	if err != nil || len(recs) != 1 {
		t.Fatalf("List after drain: %v (%d records)", err, len(recs))
	}
	if _, body, err := Read(repoRoot, testRootSHA, "sess-pending"); err != nil || !strings.Contains(string(body), "pending") {
		t.Fatalf("Read after drain: %v / %q", err, string(body))
	}
	left, err := ListStaged(repoRoot, testRootSHA)
	if err != nil || len(left) != 0 {
		t.Fatalf("ListStaged after drain: %v (%d left)", err, len(left))
	}
}

// TestPerRepoPullInIsOptInOnly is the opt-in detector. The per-repo location is
// a pull, never the default: without a declaration in the caller's own home the
// transcript lands in the user-level store, and the repo tree is untouched.
func TestPerRepoPullInIsOptInOnly(t *testing.T) {
	repoRoot, home := virginHome(t)

	res, err := Capture(repoRoot, testRootSHA, "sess-default", []byte("assistant: hi\n"), "native")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if !strings.HasPrefix(res.Record.Path, filepath.Join(home, ".abcd", "transcripts")) {
		t.Errorf("undeclared repo stored at %q, want the user-level default", res.Record.Path)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, ".abcd")); err == nil {
		t.Error("an undeclared repo must not have a store written into its working tree")
	}
}

// TestPerRepoPullInIsHonouredWhenDeclared: with the repo root declared in the
// home-scoped ~/.abcd/local-transcript-roots, the same capture lands inside the
// repo's gitignored local tier instead.
func TestPerRepoPullInIsHonouredWhenDeclared(t *testing.T) {
	repoRoot, home := virginHome(t)
	declareLocal(t, home, repoRoot, 0o600)

	res, err := Capture(repoRoot, testRootSHA, "sess-local", []byte("assistant: hi\n"), "native")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	want := filepath.Join(repoRoot, ".abcd", ".work.local", "transcripts", testRootSHA, "records")
	if got := filepath.Dir(res.Record.Path); got != want {
		t.Errorf("declared repo stored at %q, want the pulled-in location %q", got, want)
	}
	recs, err := List(repoRoot, testRootSHA)
	if err != nil || len(recs) != 1 {
		t.Fatalf("List against the pulled-in store: %v (%d records)", err, len(recs))
	}
	if _, err := os.Stat(filepath.Join(home, ".abcd", "transcripts", testRootSHA, "records", filepath.Base(res.Record.Path))); err == nil {
		t.Error("the record must not also be in the user-level store")
	}
}

// TestPullInDeclarationAnyoneCanWriteIsIgnoredLoudly: a declaration that is not
// the caller's own word pulls nothing in, and says so. A silently ignored opt-in
// is indistinguishable from one that was never written.
func TestPullInDeclarationAnyoneCanWriteIsIgnoredLoudly(t *testing.T) {
	repoRoot, home := virginHome(t)
	declareLocal(t, home, repoRoot, 0o666)

	store, err := Resolve(repoRoot, testRootSHA)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if store.Local {
		t.Error("a world-writable declaration must not pull transcripts into the repo")
	}
	if len(store.Notes) == 0 || !strings.Contains(strings.Join(store.Notes, "\n"), "IGNORED") {
		t.Errorf("an ignored declaration must be reported, got notes %q", store.Notes)
	}
	for _, n := range store.Notes {
		if strings.Contains(n, home) {
			t.Errorf("a diagnostic must not carry the caller's home path: %q", n)
		}
	}
}

// TestLegacyStoreIsMigratedNotOrphaned: a corpus written under the old
// ~/.abcd/history/<root-sha>/transcripts/ layout is moved into the store, is
// visible through the read verbs afterwards, and leaves a tombstone at the old
// path so the move is discoverable there too.
func TestLegacyStoreIsMigratedNotOrphaned(t *testing.T) {
	repoRoot, home := virginHome(t)

	legacyRepo := filepath.Join(home, ".abcd", "history", testRootSHA)
	legacyRecords := filepath.Join(legacyRepo, "transcripts")
	legacyStaging := filepath.Join(legacyRepo, "staging")
	for _, d := range []string{legacyRecords, legacyStaging} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	record := "---\nschema: 1\nsession_id: sess-old\nroot_commit: " + testRootSHA +
		"\ncaptured_at: 2026-01-01T00:00:00Z\nsource_kind: native\nsource_sha256: " +
		strings.Repeat("d", 64) + "\nredacted_secrets: 0\nredacted_home_paths: 0\n---\nassistant: from the old store\n"
	name := "20260101T000000.000000000Z-sess-old.md"
	if err := os.WriteFile(filepath.Join(legacyRecords, name), []byte(record), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyStaging, "20260101T000001.000000000Z-sess-pending.raw"), []byte("assistant: staged\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	store, err := Resolve(repoRoot, testRootSHA)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(store.Notes) == 0 || !strings.Contains(strings.Join(store.Notes, "\n"), "moved") {
		t.Errorf("a migration must be reported out loud, got notes %q", store.Notes)
	}

	recs, err := List(repoRoot, testRootSHA)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(recs) != 1 || recs[0].SessionID != "sess-old" {
		t.Fatalf("the legacy record is orphaned: List returned %d record(s)", len(recs))
	}
	if !strings.HasPrefix(recs[0].Path, store.Records) {
		t.Errorf("record still at %q, want it under %q", recs[0].Path, store.Records)
	}
	staged, err := ListStaged(repoRoot, testRootSHA)
	if err != nil || len(staged) != 1 {
		t.Fatalf("the legacy staged transcript is orphaned: %v (%d staged)", err, len(staged))
	}
	if _, err := os.Stat(filepath.Join(legacyRepo, "transcripts.moved")); err != nil {
		t.Errorf("the old location must carry a tombstone naming the new one: %v", err)
	}
	if _, err := os.Stat(filepath.Join(legacyRecords, name)); err == nil {
		t.Error("the migrated record is still at the legacy path — a move, not a copy")
	}
	// Idempotent: a second resolve has nothing to say and moves nothing.
	again, err := Resolve(repoRoot, testRootSHA)
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}
	if len(again.Notes) != 0 {
		t.Errorf("a settled store must resolve quietly, got %q", again.Notes)
	}
}

// TestMigrationRefusesASymlinkedLegacyLeaf: the old path is judged on the same
// bar the store's own levels are, so a symlink planted where the legacy corpus
// sat cannot walk its target's files into the store under names the store then
// treats as its own.
func TestMigrationRefusesASymlinkedLegacyLeaf(t *testing.T) {
	repoRoot, home := virginHome(t)
	elsewhere := t.TempDir()
	if err := os.WriteFile(filepath.Join(elsewhere, "20260101T000000.000000000Z-planted.md"),
		[]byte("---\nschema: 1\nsession_id: planted\nroot_commit: "+testRootSHA+
			"\ncaptured_at: 2026-01-01T00:00:00Z\nsource_kind: native\nsource_sha256: "+
			strings.Repeat("e", 64)+"\nredacted_secrets: 0\nredacted_home_paths: 0\n---\nassistant: not ours\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	legacyRepo := filepath.Join(home, ".abcd", "history", testRootSHA)
	if err := os.MkdirAll(legacyRepo, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(legacyRepo, "transcripts")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	// A real staging dir beside it, so the migration runs at all.
	if err := os.MkdirAll(filepath.Join(legacyRepo, "staging"), 0o700); err != nil {
		t.Fatal(err)
	}

	recs, err := List(repoRoot, testRootSHA)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(recs) != 0 {
		t.Errorf("the migration walked a symlinked legacy path into the store: %+v", recs)
	}
	if entries, _ := os.ReadDir(elsewhere); len(entries) != 1 {
		t.Errorf("the migration moved a file out of the symlink's target; %d left", len(entries))
	}
}

// TestStoreRefusesASymlinkedLevel holds the ownedDirsReal discipline across the
// relocation: the store now CREATES its directories, but it must still never
// create or write THROUGH a planted symlink.
func TestStoreRefusesASymlinkedLevel(t *testing.T) {
	repoRoot, home := virginHome(t)
	elsewhere := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(home, ".abcd", "transcripts")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	_, err := Capture(repoRoot, testRootSHA, "sess-planted", []byte("assistant: hi\n"), "native")
	if err == nil {
		t.Fatal("Capture through a symlinked store level must be refused")
	}
	var spe *StorePathError
	if !errors.As(err, &spe) {
		t.Fatalf("want a StorePathError, got %T: %v", err, err)
	}
	if entries, _ := os.ReadDir(elsewhere); len(entries) != 0 {
		t.Errorf("the store wrote through the symlink: %d entries appeared at the target", len(entries))
	}
}

// declareLocal writes the home-scoped pull-in declaration at the given mode.
func declareLocal(t *testing.T, home, repoRoot string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".abcd", "local-transcript-roots")
	body := "# transcripts for these checkouts stay in the checkout\n" + repoRoot + "\n"
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}
