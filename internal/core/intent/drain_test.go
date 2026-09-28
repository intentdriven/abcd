package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// drain_test.go pins the drain queue behind `abcd intent audit --owed`
// (itd-53, spc-2609211930059886 piece 1): the owed set from the one reader,
// ordered oldest shipped first, capped by --max, with how many remain.

// shippedOnFrom is a ShippedOn over a fixed map keyed by intent file name.
func shippedOnFrom(days map[string]string) ShippedOn {
	return func(rel string) string { return days[filepath.Base(rel)] }
}

func queueIDs(q ReviewQueue) []string {
	ids := make([]string, 0, len(q.Queue))
	for _, e := range q.Queue {
		ids = append(ids, e.IntentID)
	}
	return ids
}

func TestOwedQueueOrdersOldestShippedFirst(t *testing.T) {
	root := t.TempDir()
	seedReviewStates(t, root)
	days := map[string]string{
		"itd-11-owed.md":     "2026-03-01",
		"itd-12-ingested.md": "2025-01-01", // ingested: never queued, however old
		"itd-13-dead.md":     "2025-01-02", // dead-lettered: never queued
		"itd-14-bare.md":     "2026-01-15",
		// itd-15 has no date: shipped in the working tree, not yet committed.
	}
	q, err := OwedQueue(root, 0, shippedOnFrom(days))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(queueIDs(q), ","), "itd-14,itd-11,itd-15"; got != want {
		t.Fatalf("queue = %s, want %s (oldest first, undated last)", got, want)
	}
	if q.Owed != 3 || q.Remaining != 0 || q.Max != 0 {
		t.Fatalf("owed=%d remaining=%d max=%d, want 3/0/0", q.Owed, q.Remaining, q.Max)
	}
	if q.Queue[0].Shipped != "2026-01-15" || q.Queue[2].Shipped != "" {
		t.Fatalf("shipped days not carried: %+v", q.Queue)
	}
	// Each entry is the reader's own entry: the receipt and the re-emit ride along.
	if q.Queue[1].ReceiptID != owedRcp || q.Queue[1].ReEmit != "abcd intent audit itd-11" {
		t.Fatalf("queued entry lost the reader's fields: %+v", q.Queue[1])
	}
}

// TestOwedQueueTiesFallToMintOrder: two intents shipped the same day, or with no
// day at all, fall to the order their ids were minted in — numerically, so
// itd-9 precedes itd-10, and every ordinal precedes every timestamp id.
func TestOwedQueueTiesFallToMintOrder(t *testing.T) {
	root := t.TempDir()
	for _, id := range []string{"itd-2609150819445595", "itd-10", "itd-9", "itd-100"} {
		writeFile(t, root, shippedDir+"/"+id+"-x.md", shippedWithNotes(id, "x", "_Empty._"))
	}
	q, err := OwedQueue(root, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(queueIDs(q), ","), "itd-9,itd-10,itd-100,itd-2609150819445595"; got != want {
		t.Fatalf("undated queue = %s, want %s", got, want)
	}

	same := map[string]string{}
	for _, id := range []string{"itd-2609150819445595", "itd-10", "itd-9", "itd-100"} {
		same[id+"-x.md"] = "2026-05-05"
	}
	q, err = OwedQueue(root, 0, shippedOnFrom(same))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(queueIDs(q), ","), "itd-9,itd-10,itd-100,itd-2609150819445595"; got != want {
		t.Fatalf("same-day queue = %s, want %s", got, want)
	}
}

func TestOwedQueueCapNamesTheRemainder(t *testing.T) {
	root := t.TempDir()
	seedReviewStates(t, root)
	days := map[string]string{"itd-11-owed.md": "2026-03-01", "itd-14-bare.md": "2026-01-15", "itd-15-dup.md": "2026-04-01"}
	q, err := OwedQueue(root, 2, shippedOnFrom(days))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(queueIDs(q), ","), "itd-14,itd-11"; got != want {
		t.Fatalf("capped queue = %s, want %s", got, want)
	}
	if q.Owed != 3 || q.Remaining != 1 || q.Max != 2 {
		t.Fatalf("owed=%d remaining=%d max=%d, want 3/1/2", q.Owed, q.Remaining, q.Max)
	}

	// A cap above the owed count lists them all and leaves none.
	q, err = OwedQueue(root, 10, shippedOnFrom(days))
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Queue) != 3 || q.Remaining != 0 {
		t.Fatalf("cap 10: queue %d remaining %d, want 3/0", len(q.Queue), q.Remaining)
	}
}

func TestOwedQueueRefusesANegativeCap(t *testing.T) {
	root := t.TempDir()
	seedReviewStates(t, root)
	if _, err := OwedQueue(root, -1, nil); err == nil || !strings.Contains(err.Error(), "--max") {
		t.Fatalf("a negative cap must be refused naming --max, got %v", err)
	}
}

// TestOwedQueueEmptyIsNotAnError: nothing owed is a queue of none, not a fault.
func TestOwedQueueEmptyIsNotAnError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, shippedDir+"/itd-12-ingested.md", shippedWithNotes("itd-12", "ingested",
		"<!-- abcd-review: INGESTED receipt="+ingestedRcp+" -->\nFidelity review — receipt "+ingestedRcp+"."))
	q, err := OwedQueue(root, 5, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Queue) != 0 || q.Owed != 0 || q.Remaining != 0 || q.Queue == nil {
		t.Fatalf("empty queue = %+v (want a non-nil empty queue)", q)
	}
}

// TestOwedQueueNeverWrites: ordering and capping read the record; they never
// re-emit, park or stamp anything.
func TestOwedQueueNeverWrites(t *testing.T) {
	root := t.TempDir()
	seedReviewStates(t, root)
	before := snapshotTree(t, root)
	if _, err := OwedQueue(root, 1, nil); err != nil {
		t.Fatal(err)
	}
	if after := snapshotTree(t, root); after != before {
		t.Fatal("OwedQueue wrote to the tree")
	}
}

// TestNextOwedAuditEmitsTheHead: the step re-emits the request for the oldest
// owed review — the single audit's own emit — and names it, so a host without
// the plugin page can drive the drain by hand. A markerless head has its
// receipt minted by that emit, and the queue reports the receipt it now owes.
func TestNextOwedAuditEmitsTheHead(t *testing.T) {
	root := t.TempDir()
	seedReviewStates(t, root)
	days := map[string]string{"itd-11-owed.md": "2026-03-01", "itd-14-bare.md": "2026-01-15"}
	step, err := NextOwedAudit(root, 0, shippedOnFrom(days), AuditEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if step.Next == nil || step.Next.IntentID != "itd-14" || step.Next.Status != "owed" {
		t.Fatalf("next = %+v, want the oldest owed (itd-14) freshly emitted", step.Next)
	}
	if _, err := os.Stat(filepath.Join(root, step.Next.RequestPath)); err != nil {
		t.Fatalf("the head's request was not written at %s: %v", step.Next.RequestPath, err)
	}
	head := step.Queue[0]
	if head.IntentID != "itd-14" || head.State != ReviewOwed || head.ReceiptID != step.Next.ReceiptID {
		t.Fatalf("queue head does not report the receipt its emit minted: %+v", head)
	}
	body, err := os.ReadFile(filepath.Join(root, shippedDir, "itd-14-bare.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "OWED receipt="+step.Next.ReceiptID) {
		t.Fatalf("the emit parked no OWED marker:\n%s", body)
	}
	// Only the head is emitted: the rest of the queue is untouched.
	if _, err := os.Stat(filepath.Join(root, reviewsRelDir, owedRcp+".request.md")); err == nil {
		t.Fatal("a request was written for an entry behind the head")
	}
}

// TestNextOwedAuditOnNothingOwedWritesNothing: an empty queue has no head, so
// the step names no request and writes nothing.
func TestNextOwedAuditOnNothingOwedWritesNothing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, shippedDir+"/itd-12-ingested.md", shippedWithNotes("itd-12", "ingested",
		"<!-- abcd-review: INGESTED receipt="+ingestedRcp+" -->\nFidelity review — receipt "+ingestedRcp+"."))
	before := snapshotTree(t, root)
	step, err := NextOwedAudit(root, 0, nil, AuditEmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if step.Next != nil || len(step.Queue) != 0 {
		t.Fatalf("nothing owed, got %+v", step)
	}
	if after := snapshotTree(t, root); after != before {
		t.Fatal("an empty drain step wrote to the tree")
	}
}

// TestOwedQueueTellsUnknownFromUncommitted (iss-2609252052381777): with no
// history every day is unknown, which is not the same fact as an intent
// shipped in the working tree and not yet committed, so the two carry
// distinct states.
func TestOwedQueueTellsUnknownFromUncommitted(t *testing.T) {
	root := t.TempDir()
	seedReviewStates(t, root)
	q, err := OwedQueue(root, 0, shippedOnFrom(map[string]string{"itd-14-bare.md": "2026-01-15"}))
	if err != nil {
		t.Fatal(err)
	}
	states := map[string]string{}
	for _, e := range q.Queue {
		states[e.IntentID] = e.ShippedState
	}
	if states["itd-14"] != ShippedDated || states["itd-11"] != ShippedUncommitted {
		t.Fatalf("with a history: states %v, want itd-14 %q and itd-11 %q", states, ShippedDated, ShippedUncommitted)
	}
	q, err = OwedQueue(root, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range q.Queue {
		if e.ShippedState != ShippedUnknown || e.Shipped != "" {
			t.Fatalf("with no history every day is unknown, got %+v", e)
		}
	}
}

// TestNextOwedAuditSkipsAHeadThatCannotBeEmitted (iss-2609252052386874): a head
// whose request cannot be emitted (here a spec_id carrying no number) carries
// its error on its own queue entry, and the step emits the next entry instead,
// so one bad record never blocks the drain. With every listed entry failing,
// the queue still comes back, with no next.
func TestNextOwedAuditSkipsAHeadThatCannotBeEmitted(t *testing.T) {
	root := t.TempDir()
	seedReviewStates(t, root)
	bad := strings.Replace(shippedWithNotes("itd-9", "bad", "_Empty._"), "spec_id: spc-1", "spec_id: none", 1)
	writeFile(t, root, shippedDir+"/itd-9-bad.md", bad)
	days := map[string]string{"itd-9-bad.md": "2025-12-01", "itd-11-owed.md": "2026-03-01", "itd-14-bare.md": "2026-01-15"}

	step, err := NextOwedAudit(root, 0, shippedOnFrom(days), AuditEmitOptions{})
	if err != nil {
		t.Fatalf("a head that cannot be emitted must not fail the step: %v", err)
	}
	if got, want := strings.Join(queueIDs(step.ReviewQueue), ","), "itd-9,itd-14,itd-11,itd-15"; got != want {
		t.Fatalf("queue = %s, want %s", got, want)
	}
	if !strings.Contains(step.Queue[0].EmitError, "spec_id") {
		t.Fatalf("the bad head carries no emit error: %+v", step.Queue[0])
	}
	if step.Next == nil || step.Next.IntentID != "itd-14" || step.Queue[1].EmitError != "" {
		t.Fatalf("next = %+v, want itd-14, the first entry that emits", step.Next)
	}

	step, err = NextOwedAudit(root, 1, shippedOnFrom(days), AuditEmitOptions{})
	if err != nil {
		t.Fatalf("a queue of one bad entry must still come back: %v", err)
	}
	if step.Next != nil || len(step.Queue) != 1 || step.Queue[0].EmitError == "" || step.Remaining != 3 {
		t.Fatalf("want the one bad entry listed with its error and no next, got %+v", step)
	}
}

// TestNextOwedAuditFailedRequestWriteLeavesEveryIntentUntouched
// (iss-2609252127427592): the request is written before the intent file, so an
// environment-shaped failure to write it (here the reviews directory is a plain
// file) parks no OWED stub on any entry the step tries. Every row carries its
// error and keeps the receipt state it read: a markerless row still has no
// receipt, which is now the truth, and an already-parked one stays OWED.
func TestNextOwedAuditFailedRequestWriteLeavesEveryIntentUntouched(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, shippedDir+"/itd-20-owed.md", shippedWithNotes("itd-20", "owed", owedBlock(owedRcp)))
	for _, id := range []string{"itd-21", "itd-22", "itd-23", "itd-24"} {
		writeFile(t, root, shippedDir+"/"+id+"-bare.md", shippedWithNotes(id, "bare", "_Empty._"))
	}
	writeFile(t, root, reviewsRelDir, "not a directory\n")
	before := snapshotTree(t, root)

	step, err := NextOwedAudit(root, 4, nil, AuditEmitOptions{})
	if err != nil {
		t.Fatalf("a failed emit must not fail the step: %v", err)
	}
	if after := snapshotTree(t, root); after != before {
		t.Fatal("a failed request write modified the tree: an OWED stub was parked with no request")
	}
	if step.Next != nil || len(step.Queue) != 4 || step.Remaining != 1 {
		t.Fatalf("want four rows, no next, one remaining; got %+v", step)
	}
	for _, e := range step.Queue {
		if e.EmitError == "" {
			t.Fatalf("%s: the row does not name its error: %+v", e.IntentID, e)
		}
		if e.IntentID == "itd-20" {
			if e.State != ReviewOwed || e.ReceiptID != owedRcp {
				t.Fatalf("an already-parked receipt must stay reported as OWED: %+v", e)
			}
			continue
		}
		if e.State != ReviewNone || e.ReceiptID != "" {
			t.Fatalf("%s: a markerless row whose emit failed must still read no receipt: %+v", e.IntentID, e)
		}
	}
}
