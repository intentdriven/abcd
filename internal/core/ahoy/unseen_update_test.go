package ahoy

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

// seedUnseenCache writes the cache record a bootstrap swap leaves behind.
func seedUnseenCache(t *testing.T, meta string) (data string) {
	t.Helper()
	data = t.TempDir()
	cache := filepath.Join(data, "cache")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "binary-meta"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_PLUGIN_DATA", data)
	return data
}

const unseenMeta = "release_tag=v0.12.0\nrelease_sha=unknown\nbinary_sha256=" +
	"0000000000000000000000000000000000000000000000000000000000000000\n" +
	"fetched_at=2026-09-30T00:00:00Z\nprevious_tag=v0.11.1\ntransition_unseen=yes\n"

// TestTakeUnseenUpdateShowsOnce is the ruling CJ1b's single exception: a swap
// made where no one saw its output is shown by the next session start, once,
// and the marker that makes it once is the only thing the session check writes.
func TestTakeUnseenUpdateShowsOnce(t *testing.T) {
	data := seedUnseenCache(t, unseenMeta)
	cwd := t.TempDir()

	from, to, ok := TakeUnseenUpdate("", cwd)
	if !ok || from != "v0.11.1" || to != "v0.12.0" {
		t.Fatalf("the first session after an unseen swap must show it; got %q -> %q (%v)", from, to, ok)
	}
	if _, err := os.Stat(filepath.Join(data, "cache", "update-shown-v0.12.0")); err != nil {
		t.Errorf("the claim for the release shown must be cache/update-shown-v0.12.0: %v", err)
	}
	if _, _, ok := TakeUnseenUpdate("", cwd); ok {
		t.Errorf("the second session must not show the same update again")
	}
}

// TestTakeUnseenUpdateLeavesASeenSwapAlone: a swap whose own output was
// relayed carries no transition_unseen flag, so the session check neither
// shows it nor writes anything.
func TestTakeUnseenUpdateLeavesASeenSwapAlone(t *testing.T) {
	data := seedUnseenCache(t, "release_tag=v0.12.0\nprevious_tag=v0.11.1\n")
	if _, _, ok := TakeUnseenUpdate("", t.TempDir()); ok {
		t.Errorf("a seen swap must not be shown again at session start")
	}
	if _, err := os.Stat(filepath.Join(data, "cache", "update-shown-v0.12.0")); !os.IsNotExist(err) {
		t.Errorf("the session check must write nothing for a seen swap: %v", err)
	}
}

// TestTakeUnseenUpdateRefusesAHazardousDataDir: the data dir comes from the
// environment, so it passes the same shape check as every other reader of it
// (dataDirHazard) before anything is read from it or written into it.
func TestTakeUnseenUpdateRefusesAHazardousDataDir(t *testing.T) {
	data := seedUnseenCache(t, unseenMeta)
	if err := os.Chmod(filepath.Join(data, "cache"), 0o777); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := TakeUnseenUpdate("", t.TempDir()); ok {
		t.Errorf("a world-writable data dir must not be believed")
	}
	if _, err := os.Stat(filepath.Join(data, "cache", "update-shown-v0.12.0")); !os.IsNotExist(err) {
		t.Errorf("nothing may be written into a hazardous data dir: %v", err)
	}
}

// TestTakeUnseenUpdateRefusesAMisshapenTag: the tags come off a record on
// disk that an HTTP redirect filled, so only a plain release-tag shape is shown
// or written; a control byte (or anything else) shows nothing and writes
// nothing.
func TestTakeUnseenUpdateRefusesAMisshapenTag(t *testing.T) {
	for _, meta := range []string{
		"release_tag=v0.12.0\x1b[2J\nprevious_tag=v0.11.1\ntransition_unseen=yes\n",
		"release_tag=v0.12.0\nprevious_tag=v0.11.1 run this\ntransition_unseen=yes\n",
	} {
		data := seedUnseenCache(t, meta)
		if from, to, ok := TakeUnseenUpdate("", t.TempDir()); ok {
			t.Errorf("a misshapen tag must not be shown; got %q -> %q", from, to)
		}
		if _, err := os.Stat(filepath.Join(data, "cache", "update-shown-v0.12.0")); !os.IsNotExist(err) {
			t.Errorf("nothing may be written for a misshapen tag: %v", err)
		}
	}
}

// TestTakeUnseenUpdateShowsOnceUnderConcurrentSessions: an autonomous run
// starts several sessions in one plugin root within the same second, and each
// runs the session check. The claim is one exclusive create per release, so
// exactly one of them shows the line, however they interleave.
func TestTakeUnseenUpdateShowsOnceUnderConcurrentSessions(t *testing.T) {
	seedUnseenCache(t, unseenMeta)
	cwd := t.TempDir()
	const sessions = 16
	var shown atomic.Int32
	var ready, done sync.WaitGroup
	start := make(chan struct{})
	for range sessions {
		ready.Add(1)
		done.Add(1)
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			if _, _, ok := TakeUnseenUpdate("", cwd); ok {
				shown.Add(1)
			}
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()
	if n := shown.Load(); n != 1 {
		t.Errorf("%d concurrent session starts must show the unseen update exactly once; shown %d times", sessions, n)
	}
}

// TestTakeUnseenUpdateFailsClosedOnAnExistingClaim: whatever already sits at
// the claim path (a claim, or a link to someone else's file) means the release
// was claimed, so nothing is shown and nothing is written through it.
func TestTakeUnseenUpdateFailsClosedOnAnExistingClaim(t *testing.T) {
	data := seedUnseenCache(t, unseenMeta)
	victim := filepath.Join(t.TempDir(), "victim")
	if err := os.Symlink(victim, filepath.Join(data, "cache", "update-shown-v0.12.0")); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := TakeUnseenUpdate("", t.TempDir()); ok {
		t.Errorf("an existing claim path must show nothing")
	}
	if _, err := os.Lstat(victim); !os.IsNotExist(err) {
		t.Errorf("the claim must never be created through a link: %v", err)
	}
}

// TestTakeUnseenUpdateShowsAPerRootSwapOnce: with no plugin data directory the
// bootstrap runs in its degraded per-root mode and records the swap in the
// root's own .binary-meta, so the session check reads that record and claims
// the release beside it, as .update-shown-<tag>, with the same single write.
func TestTakeUnseenUpdateShowsAPerRootSwapOnce(t *testing.T) {
	t.Setenv("CLAUDE_PLUGIN_DATA", "")
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".binary-meta"), []byte(unseenMeta), 0o644); err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	from, to, ok := TakeUnseenUpdate(root, cwd)
	if !ok || from != "v0.11.1" || to != "v0.12.0" {
		t.Fatalf("the first session after an unseen per-root swap must show it; got %q -> %q (%v)", from, to, ok)
	}
	if _, err := os.Stat(filepath.Join(root, ".update-shown-v0.12.0")); err != nil {
		t.Errorf("the per-root claim must be .update-shown-v0.12.0 in the plugin root: %v", err)
	}
	if _, _, ok := TakeUnseenUpdate(root, cwd); ok {
		t.Errorf("the second session must not show the same per-root update again")
	}
}
