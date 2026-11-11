package machineload

import (
	"context"
	"errors"
	"runtime"
	"testing"
)

// TestReadContextSpentStartsNothing: a context already ended reads nothing,
// the load averages included, and answers the context's own error, so a
// caller on a deadline (the away mode's stop check) can tell a reading its
// deadline ended from a machine that could not be read.
func TestReadContextSpentStartsNothing(t *testing.T) {
	spent, cancel := context.WithCancel(context.Background())
	cancel()
	snap, err := ReadContext(spent)
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		if !errors.Is(err, ErrUnsupported) {
			t.Fatalf("ReadContext on %s = %v, want ErrUnsupported", runtime.GOOS, err)
		}
		return
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ReadContext on a spent context = %v, want the context's error", err)
	}
	if snap.HasLoad || snap.HasProcs {
		t.Fatalf("ReadContext on a spent context read the machine: %+v", snap)
	}
}

// TestReadContextReadsTheMachine: a live context reads what Read reads.
func TestReadContextReadsTheMachine(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("no reader on %s", runtime.GOOS)
	}
	var snap Snapshot
	var err error
	for attempt := 0; attempt < readAttempts; attempt++ {
		if snap, err = ReadContext(context.Background()); err == nil {
			break
		}
		t.Logf("reading the machine, attempt %d of %d, failed: %v", attempt+1, readAttempts, err)
	}
	if err != nil {
		t.Fatal(err)
	}
	if !snap.HasLoad || !snap.HasProcs || len(snap.Procs) == 0 {
		t.Fatalf("ReadContext read no load or no processes: %+v", snap)
	}
}
