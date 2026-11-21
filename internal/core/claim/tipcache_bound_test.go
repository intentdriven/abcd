package claim

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestATipCacheOfEveryCommitAReadingWalksReadsBackWhole: what a reading
// placed on as many commits as it reads down a release branch, each pushed by
// a login of the longest length, reads back whole, its marks with it.
func TestATipCacheOfEveryCommitAReadingWalksReadsBackWhole(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, tierRelDir), 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	k := readTipCache(root, "rel-v0.16.0", headA)
	for i := range maxChainRead {
		k.place(fmt.Sprintf("%040x", i+1), pushedByLogin, longestLogin)
	}
	k.markDone("quinn", "")
	k.markFailed("octo", now)
	k.markEndFailed(now)
	k.save()
	again := readTipCache(root, "rel-v0.16.0", headA)
	if len(again.kinds) != maxChainRead {
		t.Fatalf("read back %d placements, want %d", len(again.kinds), maxChainRead)
	}
	if !again.isDone("quinn", "") || again.failed["octo"].IsZero() || again.endFailed.IsZero() {
		t.Fatalf("the marks were not read back: done %v, failed %v, end-failed %v", again.done, again.failed, again.endFailed)
	}
}

// TestATipCacheIsNeverWrittenPastWhatItsReadingTakes: a cache holding more
// than its reading takes is written within that bound, its marks kept and
// placements dropped, so it still reads back.
func TestATipCacheIsNeverWrittenPastWhatItsReadingTakes(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, tierRelDir), 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	k := readTipCache(root, "rel-v0.16.0", headA)
	for i := range 4 * maxChainRead {
		k.place(fmt.Sprintf("%040x", i+1), pushedByLogin, longestLogin)
	}
	k.markDone("quinn", "")
	k.markFailed("octo", now)
	k.markEndFailed(now)
	k.save()
	info, err := os.Stat(filepath.Join(root, tipCacheDirRel, "rel-v0.16.0"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > maxTipCacheBytes {
		t.Fatalf("the cache was written at %d bytes, past the %d its reading takes", info.Size(), maxTipCacheBytes)
	}
	again := readTipCache(root, "rel-v0.16.0", headA)
	if len(again.kinds) < maxChainRead {
		t.Fatalf("read back %d placements, want at least %d", len(again.kinds), maxChainRead)
	}
	if !again.isDone("quinn", "") || again.failed["octo"].IsZero() || again.endFailed.IsZero() {
		t.Fatalf("the marks were not read back: done %v, failed %v, end-failed %v", again.done, again.failed, again.endFailed)
	}
}
