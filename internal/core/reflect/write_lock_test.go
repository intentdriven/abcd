package reflect

import (
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/intent"
)

// The write holds the intent store's lock, the lock a lifeboat embark writes
// under, so a retrospective and an embark are serialised: a write arriving
// while an embark holds the lock waits for it rather than landing in the
// window between the embark's rejudge and its write.
func TestWriteWaitsForTheIntentStoresLock(t *testing.T) {
	r := releaseRepo(t)
	out := abs(r, outputRel("v0.2.0"))
	var early bool
	var landed chan error
	err := intent.WithMintLock(r.Root(), func() error {
		done := make(chan error, 1)
		go func() {
			_, err := Write(r.Root(), WriteRequest{Tag: "v0.2.0", Answers: fullAnswers(), ProceedDespiteUnshipped: true, Now: fixedNow})
			done <- err
		}()
		select {
		case err := <-done:
			early = true
			done <- err
		case <-time.After(500 * time.Millisecond):
		}
		if exists(t, out) {
			early = true
		}
		landed = done
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := <-landed; err != nil {
		t.Fatalf("Write after the lock was released: %v", err)
	}
	if early {
		t.Error("a retrospective landed while the intent store's lock was held: the write does not take it")
	}
	if !exists(t, out) {
		t.Error("the write never landed once the lock was released")
	}
}
