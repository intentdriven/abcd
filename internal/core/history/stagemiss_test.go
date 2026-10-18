package history

import (
	"strings"
	"sync"
	"testing"
)

// TestStageMissCountsEveryConcurrentMiss: a burst of completions that all fail
// to stage writes the marker at once, and the marker's lock keeps every one
// counted (iss-2610051434156079). Without it, two read-modify-writes at once
// would each count from the same prior and one miss would vanish from the
// count.
func TestStageMissCountsEveryConcurrentMiss(t *testing.T) {
	repoRoot, _ := setupStore(t)
	if _, ok, err := StageMiss(repoRoot, testRootSHA); err != nil || ok {
		t.Fatalf("a fresh store reports a miss: ok=%v err=%v", ok, err)
	}
	const n = 12
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = NoteStageMiss(repoRoot, testRootSHA, "agent-"+string(rune('a'+i)), "lock held")
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("miss %d: %v", i, err)
		}
	}
	note, ok, err := StageMiss(repoRoot, testRootSHA)
	if err != nil || !ok {
		t.Fatalf("marker not readable: ok=%v err=%v", ok, err)
	}
	if note.Count != n {
		t.Errorf("count = %d, want %d: a concurrent miss went uncounted", note.Count, n)
	}
	if note.FirstSeen.IsZero() || note.LastSeen.Before(note.FirstSeen) {
		t.Errorf("first/last seen = %v / %v", note.FirstSeen, note.LastSeen)
	}
}

// TestStageMissKeepsTheReasonOneBoundedLine: the reason is kept on one line
// and bounded, whatever the failure said.
func TestStageMissKeepsTheReasonOneBoundedLine(t *testing.T) {
	repoRoot, _ := setupStore(t)
	long := "first line\nsecond\r" + strings.Repeat("x", 1000)
	if err := NoteStageMiss(repoRoot, testRootSHA, "a1", long); err != nil {
		t.Fatal(err)
	}
	note, _, err := StageMiss(repoRoot, testRootSHA)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(note.LastReason, "\r\n") || len(note.LastReason) > stageMissReasonMax {
		t.Errorf("reason not one bounded line: %q (%d bytes)", note.LastReason, len(note.LastReason))
	}
}
