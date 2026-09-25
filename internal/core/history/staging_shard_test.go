package history

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestStageFanOutBurstLosesNothing is the M22 ruling (iss-2609090828371674,
// 2026-09-23): each agent stages behind its own lock. With one shared staging
// lock a contended flock admits roughly ten writers a second (the helper's
// backoff ceiling, not the critical section, sets the rate), so a burst of
// simultaneous sub-agent completions past about fifty exceeds the staging
// timeout and every stage that times out is a transcript written nowhere.
// Distinct agents must not queue behind one another at all.
func TestStageFanOutBurstLosesNothing(t *testing.T) {
	repoRoot, _ := setupStore(t)
	const agents = 64
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, agents)
	for i := 0; i < agents; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			agent := "agent-fan-" + strconv.Itoa(i)
			if _, err := Stage(repoRoot, testRootSHA, subAgentStage("sess-fan", agent), []byte("branch "+agent+"\n")); err != nil {
				errs <- err
			}
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("Stage in a %d-agent burst: %v", agents, err)
	}
	staged, err := ListStaged(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	for _, s := range staged {
		seen[s.AgentID]++
	}
	for i := 0; i < agents; i++ {
		if n := seen["agent-fan-"+strconv.Itoa(i)]; n != 1 {
			t.Errorf("agent-fan-%d: %d staged copies, want 1", i, n)
		}
	}
}

// TestStageLockIsPerAgent pins the shape of the ruling rather than its rate:
// while one agent's staging lock is held, a different agent of the same
// session still stages, and the held agent's own stage is the one that waits.
func TestStageLockIsPerAgent(t *testing.T) {
	repoRoot, home := setupStore(t)
	if _, err := Stage(repoRoot, testRootSHA, subAgentStage("sess-lk", "agent-held"), []byte("first\n")); err != nil {
		t.Fatal(err)
	}
	sdir := stagingDir(home)
	held := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- fsutil.WithFileLock(stagingLockPath(sdir, "agent-held"), 5*time.Second, func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	if _, err := Stage(repoRoot, testRootSHA, subAgentStage("sess-lk", "agent-free"), []byte("other\n")); err != nil {
		t.Fatalf("a different agent's stage waited on a lock it does not share: %v", err)
	}
	blocked := make(chan error, 1)
	go func() {
		_, err := Stage(repoRoot, testRootSHA, subAgentStage("sess-lk", "agent-held"), []byte("second\n"))
		blocked <- err
	}()
	select {
	case err := <-blocked:
		t.Fatalf("the held agent's own stage did not wait for its lock (err=%v)", err)
	case <-time.After(150 * time.Millisecond):
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := <-blocked; err != nil {
		t.Fatalf("the held agent's stage after release: %v", err)
	}
}

// TestDrainRetiresTheLockWithTheStagedFile: a per-agent lock that outlived its
// staged file would leave one empty file per sub-agent ever run. Whoever
// retires the staged file retires its lock.
func TestDrainRetiresTheLockWithTheStagedFile(t *testing.T) {
	repoRoot, home := setupStore(t)
	for _, a := range []string{"agent-r1", "agent-r2"} {
		if _, err := Stage(repoRoot, testRootSHA, subAgentStage("sess-r", a), []byte(`{"type":"user","message":{"content":"hi"}}`+"\n")); err != nil {
			t.Fatal(err)
		}
	}
	locks := filepath.Join(stagingDir(home), stagingLocksDirName)
	if entries, _ := os.ReadDir(locks); len(entries) != 2 {
		t.Fatalf("after two stages the locks dir holds %d entries, want 2", len(entries))
	}
	res, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Failed) != 0 {
		t.Fatalf("drain failures: %+v", res.Failed)
	}
	entries, err := os.ReadDir(locks)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		t.Errorf("lock %s outlived the staged file it guarded", e.Name())
	}
}

// TestStageWaitsOutARetiredLock holds, at the staging layer, the guarantee the
// per-key lock files lean on. A mutator that retires a staged file unlinks its
// lock while holding it (retireStagingLock), so a Stage queued on that lock
// file before the unlink is granted the flock on an orphaned inode once the
// retirer releases. It must notice and wait for whoever holds the fresh lock
// file at the same path, not run its critical section beside them. Only
// fsutil's inode revalidation makes that true; without it this Stage returns
// while the newcomer is still inside.
func TestStageWaitsOutARetiredLock(t *testing.T) {
	repoRoot, home := setupStore(t)
	const agent = "agent-retire"
	meta := subAgentStage("sess-rt", agent)
	if _, err := Stage(repoRoot, testRootSHA, meta, []byte("first\n")); err != nil {
		t.Fatal(err)
	}
	sdir := stagingDir(home)

	holderIn := make(chan struct{})
	retired := make(chan struct{})
	newcomerIn := make(chan struct{})
	var newcomerInside atomic.Bool
	overlapped := make(chan bool, 1)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() { // A: a drain-shaped holder that retires the lock it holds
		defer wg.Done()
		if err := withStagingLock(sdir, agent, func() error {
			close(holderIn)
			time.Sleep(80 * time.Millisecond) // B opens the lock file and polls
			retireStagingLock(sdir, agent)
			close(retired)
			<-newcomerIn
			return nil
		}); err != nil {
			t.Error(err)
		}
	}()
	<-holderIn

	wg.Add(1)
	go func() { // B: a re-fired Stage queued on the lock file before it was retired
		defer wg.Done()
		_, err := Stage(repoRoot, testRootSHA, meta, []byte("second\n"))
		overlapped <- newcomerInside.Load()
		if err != nil {
			t.Error(err)
		}
	}()

	<-retired
	wg.Add(1)
	go func() { // C: takes the fresh lock file at the same path and holds it
		defer wg.Done()
		if err := withStagingLock(sdir, agent, func() error {
			newcomerInside.Store(true)
			close(newcomerIn)
			time.Sleep(300 * time.Millisecond)
			newcomerInside.Store(false)
			return nil
		}); err != nil {
			t.Error(err)
		}
	}()
	wg.Wait()
	if <-overlapped {
		t.Fatal("a Stage queued on a retired lock file ran while a newcomer held the fresh one: two holders of one staging lock")
	}
}
