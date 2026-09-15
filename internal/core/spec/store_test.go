package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
)

const (
	specsOpen   = ".abcd/development/specs/open"
	specsClosed = ".abcd/development/specs/closed"
	intentsBase = ".abcd/development/intents"
)

func TestCreateRoundTrip(t *testing.T) {
	root := t.TempDir()
	sp, err := Create(root, "itd-9", "my-feature", "")
	if err != nil {
		t.Fatal(err)
	}
	if !nativeSpecIDRe.MatchString(sp.ID) || sp.Intent != "itd-9" || sp.Status != StatusOpen {
		t.Fatalf("Create returned %+v", sp)
	}
	if sp.Path != filepath.Join(specsOpen, sp.ID+"-my-feature.md") {
		t.Fatalf("Create path = %q, want the minted id and the slug under open/", sp.Path)
	}

	data, err := os.ReadFile(filepath.Join(root, sp.Path))
	if err != nil {
		t.Fatalf("expected spec file on disk: %v", err)
	}
	if !strings.Contains(string(data), "intent: itd-9") {
		t.Fatalf("spec file missing intent link:\n%s", data)
	}

	// Round-trips through Load.
	store, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := store.Lookup(sp.ID); !ok || s.Intent != "itd-9" {
		t.Fatalf("Load/Lookup after Create = %+v, %v", s, ok)
	}
	if s, ok := store.ByIntent("itd-9"); !ok || s.ID != sp.ID {
		t.Fatalf("Load/ByIntent after Create = %+v, %v", s, ok)
	}
}

func TestCreateRejectsBadIntent(t *testing.T) {
	root := t.TempDir()
	if _, err := Create(root, "itd-../../etc", "slug", ""); err == nil {
		t.Fatal("Create with traversal intent id must fail")
	}
	if _, err := Create(root, "spc-1", "slug", ""); err == nil {
		t.Fatal("Create with non-itd intent id must fail")
	}
}

func TestCreateRejectsBadSlug(t *testing.T) {
	root := t.TempDir()
	if _, err := Create(root, "itd-9", "../../etc", ""); err == nil {
		t.Fatal("Create with traversal slug must fail")
	}
	if _, err := Create(root, "itd-9", "Bad Slug", ""); err == nil {
		t.Fatal("Create with non-kebab slug must fail")
	}
}

func TestLoadMissingDirIsEmpty(t *testing.T) {
	root := t.TempDir()
	store, err := Load(root)
	if err != nil {
		t.Fatalf("Load on missing specs dir must be soft: %v", err)
	}
	if len(store.Specs) != 0 {
		t.Fatalf("expected empty store, got %+v", store.Specs)
	}
}

func TestLoadMalformedIsHardError(t *testing.T) {
	root := t.TempDir()
	// No frontmatter at all -> no id -> hard error.
	writeFile(t, root, specsOpen+"/spc-1-broken.md", "# just a title, no frontmatter\n")
	if _, err := Load(root); err == nil {
		t.Fatal("Load must hard-error on a malformed spec file")
	}
}

func TestLoadRejectsTraversalID(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, specsOpen+"/spc-1-evil.md",
		"---\nid: spc-../../etc\nslug: evil\nintent: itd-9\n---\n# evil\n")
	if _, err := Load(root); err == nil {
		t.Fatal("Load must reject a path-traversal id in frontmatter")
	}
}

func TestCloseMovesOpenToClosed(t *testing.T) {
	root := t.TempDir()
	minted, err := Create(root, "itd-9", "my-feature", "")
	if err != nil {
		t.Fatal(err)
	}

	sp, err := Close(root, minted.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sp.Status != StatusClosed || sp.Intent != "itd-9" {
		t.Fatalf("Close returned %+v", sp)
	}

	name := minted.ID + "-my-feature.md"
	if _, err := os.Stat(filepath.Join(root, specsOpen, name)); !os.IsNotExist(err) {
		t.Fatal("open file should be gone after Close")
	}
	if _, err := os.Stat(filepath.Join(root, specsClosed, name)); err != nil {
		t.Fatalf("closed file should exist after Close: %v", err)
	}

	// The store now reports it closed.
	store, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := store.Lookup(minted.ID); !ok || s.Status != StatusClosed {
		t.Fatalf("after Close, Lookup = %+v, %v", s, ok)
	}
}

// TestCloseRefusesWhenClosedTargetExists proves Close fails closed rather than
// clobbering a same-name spec already sitting in closed/.
func TestCloseRefusesWhenClosedTargetExists(t *testing.T) {
	root := t.TempDir()
	minted, err := Create(root, "itd-9", "my-feature", "")
	if err != nil {
		t.Fatal(err)
	}
	// A same-name spec already occupies closed/.
	name := minted.ID + "-my-feature.md"
	writeFile(t, root, specsClosed+"/"+name,
		"---\nid: "+minted.ID+"\nslug: my-feature\nintent: itd-9\n---\n# pre-existing\n")

	if _, err := Close(root, minted.ID); err == nil {
		t.Fatal("Close must refuse to overwrite an existing closed target")
	}
	// The open file is untouched (still there), the closed one not clobbered.
	if _, err := os.Stat(filepath.Join(root, specsOpen, name)); err != nil {
		t.Fatalf("open file must remain after refusal: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(root, specsClosed, name))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "pre-existing") {
		t.Fatalf("closed target was clobbered:\n%s", body)
	}
}

func TestCloseMissingFails(t *testing.T) {
	root := t.TempDir()
	if _, err := Close(root, "spc-99"); err == nil {
		t.Fatal("Close on a missing spec must fail")
	}
}

func TestCloseAlreadyClosedFails(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, specsClosed+"/spc-1-done.md",
		"---\nid: spc-1\nslug: done\nintent: itd-9\n---\n# done\n")
	if _, err := Close(root, "spc-1"); err == nil {
		t.Fatal("Close on an already-closed spec must fail")
	}
}

// TestLoadRejectsFifoSpecFile (iss-68 P7) proves a FIFO at a spec path is rejected
// promptly, not hung on. The read opens with O_NOFOLLOW|O_NONBLOCK and validates
// the fd, so a FIFO returns a not-regular error instead of blocking os.ReadFile.
func TestLoadRejectsFifoSpecFile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, specsOpen), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(root, specsOpen, "spc-1-x.md"), 0o644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := Load(root)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("a FIFO spec file must be refused, not read")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Load hung on a FIFO spec file (open must not block)")
	}
}

// Concurrent Create runs in the same worktree must mint distinct ids, and the
// lock is what makes that hold when two of them draw the same candidate: the
// presence check and the write serialize, so the second run sees the first's
// file and redraws instead of writing a duplicate under a different slug.
// Live entropy makes that draw rare, so the run below asserts the invariant
// rather than provoking the clash; the scripted-entropy test in mint_test.go
// forces it.
func TestCreateConcurrentMintsDistinctIDs(t *testing.T) {
	root := t.TempDir()

	const n = 8
	start := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	ids := make(map[string]int)
	errs := make([]error, 0)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start // release all goroutines together to maximise the collision window
			sp, err := Create(root, "itd-1", fmt.Sprintf("slug-%d", i), "")
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			ids[sp.ID]++
		}(i)
	}
	close(start)
	wg.Wait()

	for _, err := range errs {
		t.Fatalf("Create failed: %v", err)
	}
	if len(ids) != n {
		t.Fatalf("concurrent Create minted %d distinct ids, want %d (duplicates: %v)", len(ids), n, ids)
	}
	for id, count := range ids {
		if count != 1 {
			t.Errorf("id %s minted %d times", id, count)
		}
	}
}

// TestSpecCreateStampsProvenance proves the spec store's mint carries the same
// disclosure pair as every other write path. A spec is minted by a verb a person
// invoked, so its arrival path is researcher-authored — the value is derived from
// which command ran, never asked for.
func TestSpecCreateStampsProvenance(t *testing.T) {
	root := t.TempDir()
	sp, err := Create(root, "itd-9", "my-feature", "scribe-transcribed")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, sp.Path))
	if err != nil {
		t.Fatal(err)
	}
	fields := frontmatter.Fields(strings.Split(string(data), "\n"))
	if got := fields["origin"].Value; got != "researcher-authored" {
		t.Errorf("origin = %q, want researcher-authored", got)
	}
	if got := fields["production_mode"].Value; got != "scribe-transcribed" {
		t.Errorf("production_mode = %q, want scribe-transcribed", got)
	}

	// An unset mode takes the default rather than writing no line: a record
	// written through a command carries BOTH keys.
	sp2, err := Create(root, "itd-10", "second-feature", "")
	if err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join(root, sp2.Path))
	if err != nil {
		t.Fatal(err)
	}
	fields = frontmatter.Fields(strings.Split(string(data), "\n"))
	if got := fields["production_mode"].Value; got != "hand-written" {
		t.Errorf("defaulted production_mode = %q, want hand-written", got)
	}

	// An out-of-vocabulary mode is refused before the id is minted.
	if _, err := Create(root, "itd-11", "third-feature", "typed"); err == nil {
		t.Error("an out-of-vocabulary production mode must be refused")
	}
}
