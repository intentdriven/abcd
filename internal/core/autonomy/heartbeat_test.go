package autonomy

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// TestHeartbeatChangesNameWhatMovedSinceTheLastBeat (iss-2611220942554289):
// a fact is a change when it is new or its state moved; a reservation no
// longer held and a failed read now made are changes too, while a pull
// request reported merged or closed is not reported again when the next
// beat no longer reads it. Nothing moved is no change at all.
func TestHeartbeatChangesNameWhatMovedSinceTheLastBeat(t *testing.T) {
	prev := []BeatItem{
		{Kind: BeatReservation, Key: "iss-1", State: "live", Line: "iss-1: held"},
		{Kind: BeatReservation, Key: "iss-2", State: "live", Line: "iss-2: held"},
		{Kind: BeatPullRequest, Key: "#5", State: "queued:AWAITING_CHECKS:2", Line: "#5: queued"},
		{Kind: BeatPullRequest, Key: "#6", State: "merged", Line: "#6: merged", Terminal: true},
		{Kind: BeatDisk, Key: "disk", State: "ok", Line: "disk: ok"},
		{Kind: BeatForge, Key: "reservations", State: "unfetched", Line: "reservations: not fetched"},
	}
	if got := HeartbeatChanges(prev, prev[:]); len(got) != 0 {
		t.Fatalf("an unchanged beat reports %+v", got)
	}
	now := []BeatItem{
		{Kind: BeatReservation, Key: "iss-1", State: "live", Line: "iss-1: held"},
		{Kind: BeatPullRequest, Key: "#5", State: "removed:2026-11-22T07:00:00Z", Line: "#5: removed"},
		{Kind: BeatPullRequest, Key: "#7", State: "open", Line: "#7: open"},
		{Kind: BeatDisk, Key: "disk", State: "ok", Line: "disk: ok"},
	}
	got := HeartbeatChanges(prev, now)
	want := []BeatChange{
		{BeatItem: now[1], Was: "queued:AWAITING_CHECKS:2"},
		{BeatItem: now[2]},
		{BeatItem: BeatItem{Kind: BeatReservation, Key: "iss-2", State: "live", Line: "iss-2: no longer held by you from this checkout"}, Gone: true},
		{BeatItem: BeatItem{Kind: BeatForge, Key: "reservations", State: "unfetched", Line: "reservations: read again"}, Gone: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("changes =\n%+v\nwant\n%+v", got, want)
	}
	if tr := TrackedPRs(prev); !reflect.DeepEqual(tr, []int{5}) {
		t.Fatalf("tracked = %v; want the one not yet terminal", tr)
	}
}

// TestHeartbeatStateRoundTripsInTheLocalTier: the facts are kept in the
// checkout's local tier and read back; a checkout with none keeps nothing
// and says so, and no heartbeat yet reads as none.
func TestHeartbeatStateRoundTripsInTheLocalTier(t *testing.T) {
	root := t.TempDir()
	if items, at, err := ReadHeartbeat(root); err != nil || items != nil || !at.IsZero() {
		t.Fatalf("no heartbeat yet: %v %v %v", items, at, err)
	}
	at := time.Date(2026, 11, 22, 9, 30, 0, 0, time.UTC)
	items := []BeatItem{{Kind: BeatDisk, Key: "disk", State: "ok", Line: "disk: ok"}}
	if err := WriteHeartbeat(root, at, items); !errors.Is(err, ErrHeartbeatNoTier) {
		t.Fatalf("a checkout with no local tier: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteHeartbeat(root, at, items); err != nil {
		t.Fatal(err)
	}
	got, gotAt, err := ReadHeartbeat(root)
	if err != nil || !reflect.DeepEqual(got, items) || !gotAt.Equal(at) {
		t.Fatalf("read back %+v at %v (%v)", got, gotAt, err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(HeartbeatRel)), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadHeartbeat(root); err == nil {
		t.Fatal("a state file that does not parse reads")
	}
}
