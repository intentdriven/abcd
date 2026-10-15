package cli

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/intentdriven/abcd/internal/core/vintage"
)

// The record and board renders read the reservations from disk alone
// (spc-2610031753359849 step 11, itd-2609091034175565): `abcd <record-id>`
// names the record's reservation and the view's age, and the board names
// the reservation another copy holds on an intent, with no fetch and no push.

// reservedRecords seeds the world's origin with what a managed checkout
// holds: the marker block, an open issue and a draft intent.
func reservedRecords(t *testing.T, w *reserveWorld) {
	t.Helper()
	seed := filepath.Join(w.home, "seed")
	writeRel(t, seed, "CLAUDE.md", "# Project\n\n<!-- BEGIN ABCD -->\nx\n<!-- END ABCD -->\n")
	writeRel(t, seed, ".abcd/work/issues/open/iss-5-a-reserved-finding.md", peerIssue("iss-5", "a-reserved-finding", "A reserved finding"))
	writeRel(t, seed, ".abcd/work/issues/open/iss-6-a-later-finding.md", peerIssue("iss-6", "a-later-finding", "A later finding"))
	writeRel(t, seed, ".abcd/development/intents/drafts/itd-7-x.md", peerIntent("itd-7", "A reserved draft"))
	writeRel(t, seed, ".abcd/development/intents/planned/itd-8-ready.md", readyReservedIntent)
	writeRel(t, seed, ".abcd/development/specs/open/spc-8-ready.md", readyReservedSpec)
	gitCmd(t, seed, "add", "-A")
	gitCommit(t, seed, "commit", "-q", "-m", "records")
	gitCmd(t, seed, "push", "-q", "origin", "main")
}

// readyReservedIntent is a planned intent every other check `abcd build` runs
// passes, and readyReservedSpec its spec, so only a reservation can refuse it.
const (
	readyReservedIntent = "---\nid: itd-8\nslug: ready\nspec_id: spc-8\nkind: standalone\n---\n# ready\n\n" +
		"## Mechanism\n\nWe expect it to work because it is small; shown wrong if it is not.\n\n" +
		"## Scope Conditions\n\nNone stated.\n\n## Acceptance Criteria\n\n- Given x, when y, then z.\n\n" +
		"## Open Questions\n\n_None open._\n\n" +
		"## Grounds\n\n- pursued: we expect a reservation to hold it; shown wrong if a build starts it\n"
	readyReservedSpec = "---\nid: spc-8\nslug: ready\nintent: itd-8\n---\n# ready\n\n## Summary\n\nA written design record.\n\n" +
		"## Footprint\n\n- packages: d.go\n- tests: none\n"
)

// recordJSON is the reservation half of `abcd <record-id> --json`.
type recordJSON struct {
	Reservation *struct {
		State      string `json:"state"`
		Holder     string `json:"holder"`
		Copy       string `json:"copy"`
		AgeSeconds *int64 `json:"age_seconds"`
		Text       string `json:"text"`
	} `json:"reservation"`
}

// boardStatusJSON is the reservation half of the board's --json status block.
type boardStatusJSON struct {
	Status *struct {
		Later []struct {
			ID       string `json:"id"`
			Reserved string `json:"reserved"`
		} `json:"later"`
	} `json:"status"`
}

// TestReservationRendersTouchNoNetwork is C49: after Q's checkout fetched the
// reservations, the record render and the board show P's reservations and
// the view's age from disk alone, and `abcd build <itd-N>` and `abcd build
// next` refuse P's reserved intent at their peers check from disk alone. A reservation P makes after that fetch is
// not seen, the fetch stamp is not moved, the origin can be gone, and no HTTP
// request is made.
func TestReservationRendersTouchNoNetwork(t *testing.T) {
	w := newReserveWorld(t)
	reservedRecords(t, w)
	p := w.clone(t, "p", octoMail)
	q := w.clone(t, "q", quinnMail)
	for _, args := range [][]string{
		{"reserve", "iss-5", "--footprint", "a.go"},
		{"reserve", "itd-7", "--footprint", "b.go"},
		{"reserve", "itd-8", "--footprint", "d.go"},
	} {
		if code, out, errOut := reserveRun(t, p, args...); code != 0 {
			t.Fatalf("%v: exit %d\n%s%s", args, code, out, errOut)
		}
	}
	if code, out, errOut := reserveRun(t, q, "peers", "--json"); code != 0 {
		t.Fatalf("Q's fetch: exit %d\n%s%s", code, out, errOut)
	}
	if code, out, errOut := reserveRun(t, p, "reserve", "iss-6", "--footprint", "c.go"); code != 0 {
		t.Fatalf("P's later reservation: exit %d\n%s%s", code, out, errOut)
	}
	stampPath := filepath.Join(q, ".abcd", ".work.local", "reservation", "fetched-at")
	stamped, err := os.ReadFile(stampPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(w.origin, w.origin+".away"); err != nil {
		t.Fatal(err)
	}

	var hits int32
	origTransport := http.DefaultTransport
	http.DefaultTransport = countingTransport{&hits}
	origFetcher := newReleaseFetcher
	newReleaseFetcher = func() vintage.ReleaseFetcher { return countingFetcher{&hits} }
	t.Cleanup(func() {
		http.DefaultTransport = origTransport
		newReleaseFetcher = origFetcher
	})

	code, raw, errOut := reserveRun(t, q, "iss-5", "--json")
	if code != 0 {
		t.Fatalf("abcd iss-5: exit %d\n%s%s", code, raw, errOut)
	}
	var rec recordJSON
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		t.Fatalf("abcd iss-5 --json: %v\n%s", err, raw)
	}
	if r := rec.Reservation; r == nil || r.State != "live" || r.Holder != "octo" || r.Copy != "" || r.AgeSeconds == nil ||
		!strings.HasPrefix(r.Text, "live, reserved by octo until ") || !strings.Contains(r.Text, ", as of the last fetch ") {
		t.Fatalf("iss-5's reservation = %+v; want P's live reservation with the view's age", rec.Reservation)
	}
	code, text, errOut := reserveRun(t, q, "iss-5")
	if code != 0 || !strings.Contains(text, "\n  reservation: live, reserved by octo until ") {
		t.Fatalf("abcd iss-5: exit %d, want the reservation line\n%s%s", code, text, errOut)
	}
	code, raw, errOut = reserveRun(t, q, "iss-6", "--json")
	if code != 0 {
		t.Fatalf("abcd iss-6: exit %d\n%s%s", code, raw, errOut)
	}
	rec = recordJSON{}
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		t.Fatal(err)
	}
	if r := rec.Reservation; r == nil || r.State != "none" {
		t.Fatalf("iss-6, reserved after Q's fetch, reads %+v; a render that fetched would see it", rec.Reservation)
	}

	code, raw, errOut = reserveRun(t, q, "--json")
	if code != 0 {
		t.Fatalf("the board: exit %d\n%s%s", code, raw, errOut)
	}
	var board boardStatusJSON
	if err := json.Unmarshal([]byte(raw), &board); err != nil {
		t.Fatalf("the board --json: %v\n%s", err, raw)
	}
	var reserved string
	if board.Status != nil {
		for _, r := range board.Status.Later {
			if r.ID == "itd-7" {
				reserved = r.Reserved
			}
		}
	}
	if !strings.HasPrefix(reserved, "reserved by octo until ") || !strings.Contains(reserved, ", as of the last fetch ") {
		t.Fatalf("the board's itd-7 reads reserved %q; want P's reservation with the view's age\n%s", reserved, raw)
	}
	// P's own board names nothing: its reservation is this copy's.
	code, raw, _ = reserveRun(t, p, "--json")
	if code != 0 || strings.Contains(raw, `"reserved"`) {
		t.Fatalf("the holder's own board names its reservation as another's: exit %d\n%s", code, raw)
	}

	// The build's peers check reads the same view: P's reserved intent is
	// refused, and `build next` excludes it, with no fetch and no push.
	code, raw, errOut = reserveRun(t, q, "build", "itd-8", "--json")
	if code != 3 || !strings.Contains(raw, `"check": "peers"`) || !strings.Contains(raw, "reserved by octo until ") ||
		!strings.Contains(raw, "as of the last fetch") {
		t.Fatalf("abcd build itd-8: exit %d, want the peers check naming P's reservation\n%s%s", code, raw, errOut)
	}
	code, raw, errOut = reserveRun(t, q, "build", "next", "--json")
	if code != 2 || !strings.Contains(raw, `"id": "itd-8"`) || !strings.Contains(raw, "reserved by octo until ") {
		t.Fatalf("abcd build next: exit %d, want itd-8 excluded naming P's reservation\n%s%s", code, raw, errOut)
	}

	if now, err := os.ReadFile(stampPath); err != nil || string(now) != string(stamped) {
		t.Errorf("a render moved the fetch stamp: %q, was %q (%v)", now, stamped, err)
	}
	if n := atomic.LoadInt32(&hits); n != 0 {
		t.Fatalf("the renders made %d network request(s)", n)
	}
}
