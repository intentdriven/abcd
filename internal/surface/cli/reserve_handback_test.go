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
	if _, err := claim.Reserve(p, claim.Request{Record: "iss-5", Release: true, HandBack: claim.HandBackUserVisible}, claim.Options{}); err != nil {
		t.Fatalf("P's hand-back: %v", err)
	}

	q := w.clone(t, "q", quinnMail)
	code, text, stderr := reserveRun(t, q, "reserve", "iss-5", "--footprint", "b.go")
	if code != 0 || !strings.Contains(text, "reserved iss-5 for quinn") ||
		!strings.Contains(text, "\n  note: iss-5 was handed back by octo as user-visible at ") ||
		!strings.Contains(text, "this reservation is a person's own, so it was made\n") {
		t.Fatalf("Q's reserve over the hand-back: exit %d, stdout %q, stderr %q; want it made, with the note", code, text, stderr)
	}
	if code, _, stderr := reserveRun(t, q, "reserve", "iss-5", "--release"); code != 0 {
		t.Fatalf("Q's release: exit %d, %s", code, stderr)
	}
	t.Chdir(p)
	if _, err := claim.Reserve(p, claim.Request{Record: "iss-5", Footprint: []string{"a.go"}}, claim.Options{}); err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(p, claim.Request{Record: "iss-5", Release: true, HandBack: claim.HandBackStuck}, claim.Options{}); err != nil {
		t.Fatal(err)
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
	if err := json.Unmarshal([]byte(raw), &got); err != nil || got.HandedBack == nil || got.HandedBack.Kind != "stuck" ||
		got.HandedBack.Holder != "octo" || got.HandedBack.At == "" {
		t.Fatalf("the --json payload's hand-back = %+v (%v); want octo's stuck hand-back\n%s", got.HandedBack, err, raw)
	}
}
