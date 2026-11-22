package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReserveRenewFromAnyWorktree (iss-2611220942285216): `abcd reserve
// --renew <id>` renews the person's own reservation with the footprint it
// already records, from the worktree that took it or any other worktree of
// the checkout, and refuses a reservation the account does not hold (exit
// 3), a footprint (exit 2) and a release subject (exit 2).
func TestReserveRenewFromAnyWorktree(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	q := w.clone(t, "q", quinnMail)
	sib := filepath.Join(w.home, "p-lane")
	gitCmd(t, p, "worktree", "add", "-q", "--detach", sib)
	if err := os.MkdirAll(filepath.Join(sib, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	cloneHomes[sib] = cloneHomes[p]

	if code, out, errs := reserveRun(t, p, "reserve", "iss-5", "--footprint", "a.go", "--footprint", "b.go"); code != 0 {
		t.Fatalf("the reservation: exit %d\n%s%s", code, out, errs)
	}
	before := w.remoteTip(t, "iss-5")

	code, out, errs := reserveRun(t, sib, "reserve", "--renew", "iss-5")
	if code != 0 || !strings.Contains(out, "renewed iss-5") || !strings.Contains(out, "your worktree") ||
		!strings.Contains(out, `footprint: "a.go", "b.go"`) {
		t.Fatalf("the renewal from the sibling worktree: exit %d\n%s%s", code, out, errs)
	}
	if w.remoteTip(t, "iss-5") == before {
		t.Fatal("the renewal pushed nothing")
	}

	code, raw, errs := reserveRun(t, p, "reserve", "--renew", "iss-5", "--json")
	if code != 0 {
		t.Fatalf("the renewal from the holding worktree: exit %d\n%s%s", code, raw, errs)
	}
	var got struct {
		Act       string   `json:"act"`
		Copy      string   `json:"copy"`
		Footprint []string `json:"footprint"`
		HeldFrom  string   `json:"held_from"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	if got.Act != "renewed" || got.Copy != "this" || strings.Join(got.Footprint, ",") != "a.go,b.go" || got.HeldFrom != "" {
		t.Fatalf("payload %+v; want a renewal from this copy keeping the footprint", got)
	}

	if code, _, errs := reserveRun(t, q, "reserve", "--renew", "iss-5"); code != 3 || !strings.Contains(errs, "not by you") {
		t.Fatalf("another person's renewal: exit %d, stderr %q; want exit 3", code, errs)
	}
	if code, _, errs := reserveRun(t, sib, "reserve", "--renew", "iss-5", "--footprint", "c.go"); code != 2 || !strings.Contains(errs, "--renew takes no --footprint") {
		t.Fatalf("a renewal naming a footprint: exit %d, stderr %q; want exit 2", code, errs)
	}
	if code, _, errs := reserveRun(t, sib, "reserve", "--renew", "rel-v1.2.3"); code != 2 || !strings.Contains(errs, "release") {
		t.Fatalf("a renewal of a release subject: exit %d, stderr %q; want exit 2", code, errs)
	}
	if code, _, errs := reserveRun(t, sib, "reserve", "--renew", "iss-6"); code != 3 || !strings.Contains(errs, "no reservation") {
		t.Fatalf("a renewal of an absent reservation: exit %d, stderr %q; want exit 3", code, errs)
	}
}
