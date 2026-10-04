package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestRegisterRepoConcurrentDoesNotLoseUpdate is the iss-101 lost-update repro.
// Two registerRepo sequences for distinct repos interleave so the second's
// mutation section runs while the first is paused mid-critical-section. Pre-fix
// (unlocked load-modify-write, no re-load) each writes its own stale in-memory
// index and last-rename-wins drops one registration; post-fix each mutation runs
// under the history lock and re-loads inside it, so both registrations survive.
func TestRegisterRepoConcurrentDoesNotLoseUpdate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, err := bootstrapHistory(); err != nil {
		t.Fatal(err)
	}

	// One goroutine grabs the single token, becoming the paused "victim": it signals
	// it has loaded inside the lock and waits. The other proceeds to a full
	// registration; then the victim is released.
	tokens := make(chan struct{}, 1)
	tokens <- struct{}{}
	loaded := make(chan struct{})
	release := make(chan struct{})
	afterHistoryReloadHook = func() {
		select {
		case <-tokens:
			close(loaded)
			<-release
		default:
		}
	}
	defer func() { afterHistoryReloadHook = nil }()

	mk := func(sha, name string) *applyCtx {
		return &applyCtx{
			cwd:      filepath.Join(home, name),
			det:      DetectionResult{RepoIdentity: RepoIdentity{Name: name, RootSHA: sha}},
			prompter: RefusingPrompter{},
		}
	}

	done := make(chan struct{}, 2)
	go func() { mk("sha-aaaaaaaa", "repo-a").registerRepo("sha-aaaaaaaa"); done <- struct{}{} }()
	go func() { mk("sha-bbbbbbbb", "repo-b").registerRepo("sha-bbbbbbbb"); done <- struct{}{} }()

	<-loaded       // victim loaded inside the lock and is paused
	close(release) // let it resume; the other has been blocked on the lock
	<-done
	<-done

	idx, err := loadHistoryIndex()
	if err != nil || idx == nil {
		t.Fatalf("load index: %v", err)
	}
	if !indexHasRoot(idx, "sha-aaaaaaaa") || !indexHasRoot(idx, "sha-bbbbbbbb") {
		t.Fatalf("lost update: both registrations must survive, got %d repos: %+v", len(idx.Repos), idx.Repos)
	}
}

// TestBootstrapHistoryExclusiveCreate is the iss-101 bootstrap check-then-act
// repro. The first bootstrapper pauses just before it publishes and lets a second
// full bootstrap complete in that window. Post-fix the seed is published by an
// atomic hard link (write temp, sync, os.Link), so exactly one link wins and the
// other gets EEXIST and observes the existing file (no write); pre-fix
// (check-then-act + last-rename-wins) both report a create.
func TestBootstrapHistoryExclusiveCreate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	first := true
	var secondWrote bool
	var secondErr error
	beforeHistoryIndexCreateHook = func() {
		if !first {
			return
		}
		first = false
		beforeHistoryIndexCreateHook = nil // disarm so the nested bootstrap runs clean
		secondWrote, secondErr = bootstrapHistory()
	}
	defer func() { beforeHistoryIndexCreateHook = nil }()

	firstWrote, err := bootstrapHistory()
	if err != nil {
		t.Fatalf("first bootstrap: %v", err)
	}
	if secondErr != nil {
		t.Fatalf("second bootstrap: %v", secondErr)
	}
	if firstWrote == secondWrote {
		t.Fatalf("exactly one bootstrap must create index.json; got first=%v second=%v", firstWrote, secondWrote)
	}

	// The surviving index.json must be valid, not an empty-clobbered file.
	idx, lerr := loadHistoryIndex()
	if lerr != nil || idx == nil {
		t.Fatalf("index.json unreadable after bootstrap race: %v", lerr)
	}
}

// TestBootstrapHistoryNoEmptyReadWindow guards the atomic-seed property (the
// reviewer FIX-FIRST for iss-101): a bootstrap must never publish a 0-byte
// index.json that a concurrent loadHistoryIndex could parse-fail on and drop its
// own registration. A read taken at the publish point sees either no file yet or
// a complete one — never a malformed empty file.
func TestBootstrapHistoryNoEmptyReadWindow(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	var readErr error
	beforeHistoryIndexCreateHook = func() {
		beforeHistoryIndexCreateHook = nil
		// A reader racing the publish must not observe a malformed index.json.
		_, readErr = loadHistoryIndex()
	}
	defer func() { beforeHistoryIndexCreateHook = nil }()

	wrote, err := bootstrapHistory()
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if !wrote {
		t.Fatal("expected the bootstrap to seed the index")
	}
	if readErr != nil {
		t.Fatalf("concurrent read in the bootstrap publish window saw a malformed index: %v", readErr)
	}
	idx, lerr := loadHistoryIndex()
	if lerr != nil || idx == nil {
		t.Fatalf("final index invalid after bootstrap: %v", lerr)
	}
}

// lockProbingPrompter records, on each Confirm, whether the ~/.abcd/history lock
// is held at prompt time — the no-lock-across-prompt guard for iss-101.
type lockProbingPrompter struct {
	t         *testing.T
	lockPath  string
	confirmed bool
	lockHeld  bool
}

func (p *lockProbingPrompter) Confirm(string) bool {
	p.confirmed = true
	fd, err := syscall.Open(p.lockPath, syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW, 0o644)
	if err != nil {
		p.t.Fatalf("probe open lock: %v", err)
	}
	defer syscall.Close(fd)
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
		syscall.Flock(fd, syscall.LOCK_UN)
	} else {
		// Could not acquire: registerRepo is holding the history lock across this
		// interactive prompt — exactly the condition iss-101 forbids.
		p.lockHeld = true
	}
	return true // confirm the lineage link
}

func (p *lockProbingPrompter) Prompt(_ string, _ []string, def string) string { return def }

// TestRegisterRepoDoesNotHoldLockAcrossPrompt proves the re-founding confirmation
// is asked with the history lock NOT held (iss-101 hard constraint). A prior
// entry with a matching name makes the new registration trigger the lineage
// prompt; the prompter probes the lock while answering.
func TestRegisterRepoDoesNotHoldLockAcrossPrompt(t *testing.T) {
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

	// Seed a re-founding candidate: same name, different root_commit.
	seed := &applyCtx{
		cwd:      filepath.Join(home, "old"),
		det:      DetectionResult{RepoIdentity: RepoIdentity{Name: "myrepo", RootSHA: "sha-old"}},
		prompter: RefusingPrompter{},
	}
	seed.registerRepo("sha-old")

	probe := &lockProbingPrompter{t: t, lockPath: lockPath}
	nu := &applyCtx{
		cwd:      filepath.Join(home, "new"),
		det:      DetectionResult{RepoIdentity: RepoIdentity{Name: "myrepo", RootSHA: "sha-new"}},
		prompter: probe,
	}
	nu.registerRepo("sha-new")

	if !probe.confirmed {
		t.Fatal("the re-founding lineage prompt was never triggered; test did not exercise the constraint")
	}
	if probe.lockHeld {
		t.Fatal("history lock was held across the interactive prompt (iss-101 violation)")
	}
	// The lineage link the user confirmed must have landed.
	idx, err := loadHistoryIndex()
	if err != nil || idx == nil {
		t.Fatalf("load index: %v", err)
	}
	e := indexEntry(idx, "sha-new")
	if e == nil || e.Supersedes != "sha-old" {
		t.Fatalf("confirmed lineage link not recorded: %+v", e)
	}
}

// TestRegisterRepoWritesTheIndexItLocked is the history registry's case of
// iss-2609290300313698's pattern: registerRepo takes the lock beside
// ~/.abcd/history/index.json through one walk of ~/.abcd/history, and the
// load and the write under it must go through that same directory. Here the
// directory is swapped for a different real one, carrying an index of its
// own, once the lock is held; the registration lands in the index whose lock
// was taken, and the directory swapped in is left as it was. Loading and
// writing by walking ~/.abcd/history again wrote the swapped-in index, which
// the lock never covered.
func TestRegisterRepoWritesTheIndexItLocked(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, err := bootstrapHistory(); err != nil {
		t.Fatal(err)
	}
	hist := abcdhome.Path(home, "history")
	aside := abcdhome.Path(home, "history-aside")
	fresh := abcdhome.Path(home, "history-fresh")
	if err := os.MkdirAll(fresh, 0o755); err != nil {
		t.Fatal(err)
	}
	freshBody := []byte(`{"repos":[{"root_commit":"sha-swapped","name":"swapped"}]}` + "\n")
	if err := os.WriteFile(filepath.Join(fresh, "index.json"), freshBody, 0o644); err != nil {
		t.Fatal(err)
	}
	swapped := false
	afterHistoryReloadHook = func() {
		if swapped {
			return
		}
		swapped = true
		if err := os.Rename(hist, aside); err != nil {
			t.Errorf("swap: %v", err)
		}
		if err := os.Rename(fresh, hist); err != nil {
			t.Errorf("swap: %v", err)
		}
	}
	defer func() { afterHistoryReloadHook = nil }()

	a := &applyCtx{
		cwd:      filepath.Join(home, "repo-a"),
		det:      DetectionResult{RepoIdentity: RepoIdentity{Name: "repo-a", RootSHA: "sha-aaaaaaaa"}},
		prompter: RefusingPrompter{},
	}
	a.registerRepo("sha-aaaaaaaa")
	if !swapped {
		t.Fatal("registerRepo never took the history lock, so the swap could not be staged")
	}
	held, err := os.ReadFile(filepath.Join(aside, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(held), "sha-aaaaaaaa") {
		t.Errorf("the registration did not land in the index whose lock was taken:\n%s", held)
	}
	if after, err := os.ReadFile(filepath.Join(hist, "index.json")); err != nil || string(after) != string(freshBody) {
		t.Errorf("the directory swapped in was written although its lock was never taken: err %v, now\n%s", err, after)
	}
}
