package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// TestReserveOverAHandBackGoesAheadWithANote (spc-2610161044461448 step 3):
// a person's own `abcd reserve` over a hand-back tombstone whose record is
// unchanged on origin's default branch is made, and both renders say which
// hand-back it was made over.
func TestReserveOverAHandBackGoesAheadWithANote(t *testing.T) {
	w := newReserveWorld(t)
	filer := w.clone(t, "filer", octoMail)
	writeRel(t, filer, ".abcd/work/issues/open/iss-5-the-widget.md", "---\nid: iss-5\n---\n# The widget\n")
	gitCmd(t, filer, "add", "-A")
	gitCommit(t, filer, "commit", "-q", "-m", "file iss-5")
	gitCmd(t, filer, "push", "-q", "origin", "HEAD:main")

	p := w.clone(t, "p", octoMail)
	if code, stdout, stderr := reserveRun(t, p, "reserve", "iss-5", "--footprint", "a.go"); code != 0 {
		t.Fatalf("P's reservation: exit %d\n%s%s", code, stdout, stderr)
	}
	t.Chdir(p)
	if _, err := claim.Reserve(p, claim.Request{Record: "iss-5", Release: true, HandBack: claim.HandBackUserVisible}, claim.Options{Home: cloneHomes[p]}); err != nil {
		t.Fatalf("P's hand-back: %v", err)
	}

	q := w.clone(t, "q", quinnMail)
	code, text, stderr := reserveRun(t, q, "reserve", "iss-5", "--footprint", "b.go")
	if code != 0 || !strings.Contains(text, "reserved iss-5 for "+heldBy(t, q)) ||
		!strings.Contains(text, "\n  note: iss-5 was handed back by "+heldBy(t, p)+" as user-visible at ") ||
		!strings.Contains(text, "this reservation is a person's own, so it was made\n") {
		t.Fatalf("Q's reserve over the hand-back: exit %d, stdout %q, stderr %q; want it made, with the note", code, text, stderr)
	}
	// Q's plain release, the record unchanged, keeps the hand-back standing:
	// the tombstone carries it, and the release says so.
	code, text, stderr = reserveRun(t, q, "reserve", "iss-5", "--release")
	if code != 0 || !strings.Contains(text, "released iss-5") ||
		!strings.Contains(text, "\n  note: iss-5 was handed back by "+heldBy(t, p)+" as user-visible at ") ||
		!strings.Contains(text, "so the hand-back still stands") {
		t.Fatalf("Q's release: exit %d, stdout %q, stderr %q; want it released, the hand-back kept", code, text, stderr)
	}
	code, raw, stderr := reserveRun(t, q, "reserve", "iss-5", "--footprint", "b.go", "--json")
	if code != 0 {
		t.Fatalf("Q's --json reserve: exit %d\n%s%s", code, raw, stderr)
	}
	var got struct {
		HandedBack *struct {
			Kind   string `json:"kind"`
			Holder string `json:"holder"`
			At     string `json:"at"`
		} `json:"handed_back"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil || got.HandedBack == nil || got.HandedBack.Kind != "user-visible" ||
		got.HandedBack.Holder != heldBy(t, q) || got.HandedBack.At == "" {
		t.Fatalf("the --json payload's hand-back = %+v (%v); want the user-visible hand-back Q's release kept\n%s", got.HandedBack, err, raw)
	}
	code, raw, stderr = reserveRun(t, q, "reserve", "iss-5", "--release", "--json")
	var kept struct {
		HandBackKept *struct {
			Kind string `json:"kind"`
		} `json:"hand_back_kept"`
	}
	if err := json.Unmarshal([]byte(raw), &kept); code != 0 || err != nil || kept.HandBackKept == nil || kept.HandBackKept.Kind != "user-visible" {
		t.Fatalf("Q's --json release: exit %d, %+v (%v); want the hand-back it kept named\n%s%s", code, kept.HandBackKept, err, raw, stderr)
	}
}
