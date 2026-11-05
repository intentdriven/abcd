//go:build unix

package loop

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestAReservationRenewsDuringACall: an agent's call outlasting half the lease
// renews what the lane holds while it runs, before its receipt is handed back.
func TestAReservationRenewsDuringACall(t *testing.T) {
	was := terminalRenewTick
	terminalRenewTick = 20 * time.Millisecond
	t.Cleanup(func() { terminalRenewTick = was })
	w := newTerminalWorld(t, "shape", "slow")
	var at atomic.Int64
	at.Store(time.Now().UTC().UnixNano())
	o := w.options(t)
	o.Options.Now = func() time.Time { return time.Unix(0, at.Load()).UTC() }
	o.Drain.Max = 1
	for i := range w.stages {
		if w.stages[i].Name == StageImplement {
			run := w.stages[i].Run
			w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
				at.Add(int64(90 * time.Minute))
				return run(c, l)
			}
		}
	}
	during := make(chan bool, 1)
	go func() {
		for range 1000 {
			st, _, err := readDrain(w.repo.Root())
			if err == nil && len(st.Lanes) > 0 {
				if run, err := ReadState(w.repo.Root(), st.Lanes[0].RunID); err == nil {
					for _, e := range run.Record {
						if e.Stage == StageReservationRenewed {
							during <- len(run.Lanes[0].Receipts) == 0
							return
						}
					}
					if len(run.Lanes[0].Receipts) > 0 {
						during <- false
						return
					}
				}
			}
			time.Sleep(5 * time.Millisecond)
		}
		during <- false
	}()
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(w.lines, "\n"))
	}
	if !<-during {
		t.Fatal("the reservation was not renewed while the agent was in its call")
	}
}

// TestAReservationRenewsDuringAPreflight: a preflight queued or running past half
// the lease renews what the lane holds too.
func TestAReservationRenewsDuringAPreflight(t *testing.T) {
	was := terminalRenewTick
	terminalRenewTick = 20 * time.Millisecond
	t.Cleanup(func() { terminalRenewTick = was })
	w := newTerminalWorld(t)
	w.needsPreflight = true
	var at atomic.Int64
	at.Store(time.Now().UTC().UnixNano())
	o := w.options(t)
	o.Options.Now = func() time.Time { return time.Unix(0, at.Load()).UTC() }
	o.Drain.Max = 1
	renewed := false
	inner := o.Preflight
	o.Preflight = func(ctx context.Context, wt string) error {
		at.Add(int64(90 * time.Minute))
		st, _, err := readDrain(w.repo.Root())
		if err != nil || len(st.Lanes) == 0 {
			t.Errorf("no drain: %v", err)
			return inner(ctx, wt)
		}
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline) && !renewed; time.Sleep(20 * time.Millisecond) {
			run, err := ReadState(w.repo.Root(), st.Lanes[0].RunID)
			if err != nil {
				continue
			}
			for _, e := range run.Record {
				renewed = renewed || e.Stage == StageReservationRenewed
			}
		}
		return inner(ctx, wt)
	}
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(w.lines, "\n"))
	}
	if !renewed {
		t.Fatal("the reservation was not renewed while the preflight ran")
	}
}
