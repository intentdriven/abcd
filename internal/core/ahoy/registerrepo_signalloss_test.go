package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// confirmingPrompter accepts every re-founding confirmation.
type confirmingPrompter struct{}

func (confirmingPrompter) Confirm(string) bool                            { return true }
func (confirmingPrompter) Prompt(_ string, _ []string, def string) string { return def }

// TestRegisterRepoLockContentionSurfacesNote is the iss-128 edge (1): a
// history-lock contention timeout must not silently skip registration. The lock is
// held elsewhere so registerRepo's acquisition times out with ErrLockContention;
// the skip has to surface as a change-note instead of being discarded.
func TestRegisterRepoLockContentionSurfacesNote(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, err := bootstrapHistory(); err != nil {
		t.Fatal(err)
	}
	root, err := historyRoot()
	if err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(root, historyLockFilename)

	old := historyLockTimeout
	historyLockTimeout = 50 * time.Millisecond
	defer func() { historyLockTimeout = old }()

	held := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_ = fsutil.WithFileLock(lockPath, 5*time.Second, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held

	a := &applyCtx{
		cwd:      filepath.Join(home, "repo-x"),
		det:      DetectionResult{RepoIdentity: RepoIdentity{Name: "repo-x", RootSHA: "sha-xxxxxxxx"}},
		prompter: RefusingPrompter{},
	}
	a.registerRepo("sha-xxxxxxxx")
	close(release)

	idx, _ := loadHistoryIndex()
	if indexEntry(idx, "sha-xxxxxxxx") != nil {
		t.Fatal("precondition: the held lock should have blocked the registration")
	}
	joined := strings.ToLower(strings.Join(a.changes, "\n"))
	if !strings.Contains(joined, "regist") || !(strings.Contains(joined, "lock") || strings.Contains(joined, "skip")) {
		t.Fatalf("a lock-contention skip must surface a change-note; a.changes=%v", a.changes)
	}
}

// TestRegisterRepoRefreshAppliesPendingLineage is the iss-128 edge (2): under a
// concurrent double-install of the same NEW repo, the session that answered the
// re-founding prompt (linkLineage=true) can lose the register race and arrive on
// the refresh branch — which never consulted linkLineage, silently dropping the
// human-approved lineage link. Post-fix the refresh branch re-checks and applies
// the pending link.
func TestRegisterRepoRefreshAppliesPendingLineage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, err := bootstrapHistory(); err != nil {
		t.Fatal(err)
	}

	// Seed the re-founding candidate: same name, different root_commit.
	seed := &applyCtx{
		cwd:      filepath.Join(home, "old"),
		det:      DetectionResult{RepoIdentity: RepoIdentity{Name: "myrepo", RootSHA: "sha-old"}},
		prompter: RefusingPrompter{},
	}
	seed.registerRepo("sha-old")

	// Simulate a concurrent install winning the register race: on the first reload
	// INSIDE our session's lock, inject an entry for sha-new (no lineage link), so
	// our fn re-loads and takes the refresh branch.
	injected := false
	afterHistoryReloadHook = func() {
		if injected {
			return
		}
		injected = true
		idx, err := loadHistoryIndex()
		if err != nil || idx == nil {
			t.Fatalf("hook load: %v", err)
		}
		idx.Repos = append(idx.Repos, historyRepo{
			RootCommit: "sha-new", Name: "myrepo", Github: "", Path: filepath.Join(home, "peer"), Status: "active",
		})
		if err := writeHistoryIndex(idx); err != nil {
			t.Fatalf("hook write: %v", err)
		}
	}
	defer func() { afterHistoryReloadHook = nil }()

	nu := &applyCtx{
		cwd:      filepath.Join(home, "new"),
		det:      DetectionResult{RepoIdentity: RepoIdentity{Name: "myrepo", RootSHA: "sha-new"}},
		prompter: confirmingPrompter{},
	}
	nu.registerRepo("sha-new")

	idx, err := loadHistoryIndex()
	if err != nil || idx == nil {
		t.Fatalf("load index: %v", err)
	}
	e := indexEntry(idx, "sha-new")
	if e == nil {
		t.Fatal("sha-new must be registered")
	}
	if e.Supersedes != "sha-old" {
		t.Fatalf("the human-approved lineage link was dropped on the refresh branch: Supersedes=%q, want sha-old", e.Supersedes)
	}
	cand := indexEntry(idx, "sha-old")
	if cand == nil || cand.SupersededBy != "sha-new" || cand.Status != "superseded" {
		t.Fatalf("the candidate must be marked superseded by sha-new: %+v", cand)
	}
}

// TestRegisterRepoLockNoteCarriesNoAbsolutePath pins the lock-failure note to
// the receipt scrub. A contention timeout names no path, but a lock path the
// primitive refuses — here the hostile-clone shape, .index.lock planted as a
// symlink — is reported with the store's absolute path under the home
// directory, and the note that carries it must render through the same seam
// every written path does, so the receipt and --json output name neither the
// home directory nor the repo.
func TestRegisterRepoLockNoteCarriesNoAbsolutePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, err := bootstrapHistory(); err != nil {
		t.Fatal(err)
	}
	root, err := historyRoot()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "elsewhere"), filepath.Join(root, historyLockFilename)); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	repo := filepath.Join(home, "repo-x")
	a := &applyCtx{
		cwd:      repo,
		det:      DetectionResult{RepoIdentity: RepoIdentity{Name: "repo-x", RootSHA: "sha-xxxxxxxx"}},
		prompter: RefusingPrompter{},
	}
	a.registerRepo("sha-xxxxxxxx")

	var note string
	for _, c := range a.changes {
		if strings.Contains(c, "history registration") {
			note = c
		}
	}
	if note == "" {
		t.Fatalf("the skipped registration was not reported; a.changes=%v", a.changes)
	}
	if strings.Contains(note, home) {
		t.Errorf("the note names the home directory: %q", note)
	}
	if strings.Contains(note, repo) {
		t.Errorf("the note names the repo's absolute path: %q", note)
	}
}
