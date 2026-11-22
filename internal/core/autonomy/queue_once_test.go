package autonomy

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestQueueStepOnceQueuesAStepOnceUnderTheLock: QueueStepOnce keeps a step
// for the person's return only when the record does not already hold it,
// judged and written under the store's one lock, so racing callers queue it
// exactly once and exactly one of them reports queuing it. A step that
// differs only in spacing is the same step. With no away record it refuses.
func TestQueueStepOnceQueuesAStepOnceUnderTheLock(t *testing.T) {
	root, home := managedCopy(t)
	now := time.Now().Truncate(time.Second)
	o := fixedOptions(home, now)
	if _, _, err := QueueStepOnce(root, "publish the advisory", o); !errors.Is(err, ErrNotAway) {
		t.Fatalf("with no away record: %v, want ErrNotAway", err)
	}
	if _, err := SetAway(root, "", false, o); err != nil {
		t.Fatal(err)
	}
	const n = 8
	var wg sync.WaitGroup
	var queued atomic.Int32
	errs := make(chan error, n)
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _, err := QueueStepOnce(root, "publish  the advisory", o)
			if ok {
				queued.Add(1)
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := queued.Load(); got != 1 {
		t.Fatalf("%d racing callers reported queuing the step; want exactly one", got)
	}
	st, err := openStore(home, root)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := st.Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Queue) != 1 || rec.Queue[0].Step != "publish the advisory" {
		t.Fatalf("queue %+v; want the one step", rec.Queue)
	}
	ok, rec, err := QueueStepOnce(root, "publish the advisory", o)
	if err != nil || ok || rec == nil || len(rec.Queue) != 1 {
		t.Fatalf("a step already queued: queued %v, record %+v, err %v", ok, rec, err)
	}
}
