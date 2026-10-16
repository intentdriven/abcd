package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `abcd build` reserves, releases and takes over (spc-2610031753359849 step
// 13, itd-2609091034175565): the run's reservation is taken at the start with
// its spec's footprint, `--release` pushes the tombstone and changes nothing
// in the run's state, and `--take-over` passes through to the reservation.

// buildStartJSON is the reservation half of `abcd build --json`.
type buildStartJSON struct {
	RunID       string `json:"run_id"`
	State       string `json:"state"`
	Reservation string `json:"reservation"`
}

// TestBuildReleasePushesATombstone is C36: `abcd build <itd-N> --release`
// pushes the tombstone over this copy's reservation and leaves the run's
// state exactly as it was; a later step meets the release at the land fence.
func TestBuildReleasePushesATombstone(t *testing.T) {
	w := newReserveWorld(t)
	reservedRecords(t, w)
	p := w.clone(t, "p", octoMail)

	code, raw, errOut := reserveRun(t, p, "build", "itd-8", "--json")
	if code != 0 {
		t.Fatalf("abcd build itd-8: exit %d\n%s%s", code, raw, errOut)
	}
	var started buildStartJSON
	if err := json.Unmarshal([]byte(raw), &started); err != nil {
		t.Fatalf("abcd build --json: %v\n%s", err, raw)
	}
	if !strings.HasPrefix(started.Reservation, "reserved itd-8 on origin for this copy until ") ||
		!strings.Contains(started.Reservation, "footprint: d.go") {
		t.Fatalf("the start names the reservation it took: %q", started.Reservation)
	}
	reserved := w.remoteTip(t, "itd-8")
	if reserved == "" {
		t.Fatal("the start pushed no reservation")
	}
	statePath := filepath.Join(p, filepath.FromSlash(started.State))
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}

	code, out, errOut := reserveRun(t, p, "build", "itd-8", "--release")
	if code != 0 || !strings.Contains(out, "released the reservation on itd-8") || !strings.Contains(out, "the run's state is unchanged") {
		t.Fatalf("abcd build itd-8 --release: exit %d\n%s%s", code, out, errOut)
	}
	tomb := w.remoteTip(t, "itd-8")
	if tomb == reserved || !strings.HasPrefix(gitCmd(t, w.origin, "log", "-1", "--format=%s", tomb), "abcd reservation released: itd-8") {
		t.Fatalf("the release pushes a tombstone over %s, got %s", reserved, tomb)
	}
	if parent := gitCmd(t, w.origin, "rev-parse", tomb+"^"); parent != reserved {
		t.Fatalf("the tombstone is the reservation's child: %s", parent)
	}
	if after, err := os.ReadFile(statePath); err != nil || string(after) != string(before) {
		t.Fatalf("--release changed the run's state (%v)", err)
	}

	code, raw, _ = reserveRun(t, p, "build", "itd-8", "--release", "--json")
	if code != 0 || !strings.Contains(raw, `"act": "already_released"`) {
		t.Fatalf("a second release says it was already released: exit %d\n%s", code, raw)
	}
	if code, _, errOut := reserveRun(t, p, "build", "itd-8", "--release", "--pace", "90/0"); code != 2 || !strings.Contains(errOut, "--release takes no --pace") {
		t.Fatalf("--release with --pace is a usage error: exit %d\n%s", code, errOut)
	}
	if tip := w.remoteTip(t, "itd-8"); tip != tomb {
		t.Fatalf("a refused release pushed %s", tip)
	}
}

// TestBuildTakeOverIsPassedThrough: a build over this person's live
// reservation from another computer backs off naming --take-over, and
// `--take-over` moves it to this copy.
func TestBuildTakeOverIsPassedThrough(t *testing.T) {
	w := newReserveWorld(t)
	reservedRecords(t, w)
	laptop := w.clone(t, "laptop", octoMail)
	cloud := w.clone(t, "cloud", octoMail)
	if code, out, errOut := reserveRun(t, laptop, "build", "itd-8"); code != 0 {
		t.Fatalf("the laptop's build: exit %d\n%s%s", code, out, errOut)
	}
	held := w.remoteTip(t, "itd-8")

	code, out, errOut := reserveRun(t, cloud, "build", "itd-8")
	if code != 3 || !strings.Contains(errOut, "pass --take-over to continue here") {
		t.Fatalf("the cloud's build backs off naming --take-over: exit %d\n%s%s", code, out, errOut)
	}
	code, raw, errOut := reserveRun(t, cloud, "build", "itd-8", "--take-over", "--json")
	if code != 0 || !strings.Contains(raw, "took over the reservation on itd-8") {
		t.Fatalf("abcd build itd-8 --take-over: exit %d\n%s%s", code, raw, errOut)
	}
	if parent := gitCmd(t, w.origin, "rev-parse", w.remoteTip(t, "itd-8")+"^"); parent != held {
		t.Fatalf("the take-over is the laptop's reservation's child: %s, want %s", parent, held)
	}
}
