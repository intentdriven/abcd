package record

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The record render's reservation line (spc-2610031753359849 step 11,
// itd-2609091034175565): `abcd <record-id>` names the record's reservation as
// this checkout last fetched it, from disk alone, with the view's age.

// reservationRepo is a git checkout with a remote named origin, its local tier
// in place, and one open issue; the clock is fixed at at.
func reservationRepo(t *testing.T, at time.Time) (*gittest.Repo, string) {
	t.Helper()
	r := gittest.NewRepo(t)
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	r.Git("remote", "add", "origin", bare)
	if err := os.MkdirAll(filepath.Join(r.Root(), ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := capture.Capture(capture.CaptureRequest{
		RepoRoot: r.Root(), Text: "a reserved issue", Severity: capture.SeverityMinor,
		Category: "observation", Source: "user-observation", FoundDuring: "t", Slug: "reserved", Remedy: "a remedy",
	})
	if err != nil {
		t.Fatal(err)
	}
	prev := clock
	clock = func() time.Time { return at }
	t.Cleanup(func() { clock = prev })
	return r, res.ID
}

// TestANeverFetchedViewSaysSoRatherThanFree is C50: with no fetch stamped
// here, a record the view holds nothing for reads as unknown, never as free;
// a reservation a plain git fetch left is shown with its age unknown; a
// stamped view holding nothing reads as none, with its age.
func TestANeverFetchedViewSaysSoRatherThanFree(t *testing.T) {
	at := time.Date(2026, 10, 3, 14, 30, 0, 0, time.UTC)
	repo, id := reservationRepo(t, at)

	d, err := Describe(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	if d.Reservation == nil || d.Reservation.State != "unknown" ||
		d.Reservation.Text != "unknown, nothing has been fetched here; `abcd peers` fetches" {
		t.Fatalf("a never-fetched view renders %+v; want unknown, never none", d.Reservation)
	}

	// A plain git fetch left the reservation; nothing here stamped it.
	msg := "abcd reservation: " + id + "\n\nAbcd-Reservation: " + id + "\nAbcd-Holder: octo\n" +
		"Abcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\nAbcd-Footprint: a.go\n" +
		"Abcd-Claimed-At: 2026-10-03T14:00:00Z\nAbcd-Expires-At: 2026-10-03T16:00:00Z\n"
	tree := repo.Git("hash-object", "-w", "-t", "tree", "/dev/null")
	sha := repo.Git("commit-tree", tree, "-m", msg)
	repo.Git("update-ref", "refs/remotes/origin/abcd-reserved/"+id, sha)
	if d, err = Describe(repo.Root(), id); err != nil {
		t.Fatal(err)
	}
	if want := "live, reserved by octo until 2026-10-03T16:00:00Z, as of a fetch of unknown age"; d.Reservation == nil ||
		d.Reservation.Text != want || d.Reservation.AgeSeconds != nil {
		t.Fatalf("a plain fetch's reservation renders %+v; want %q with no age", d.Reservation, want)
	}

	// A view abcd stamped twelve minutes ago, holding nothing for the record.
	repo.Git("update-ref", "-d", "refs/remotes/origin/abcd-reserved/"+id)
	stampDir := filepath.Join(repo.Root(), ".abcd", ".work.local", "reservation")
	if err := os.MkdirAll(stampDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stampDir, "fetched-at"), []byte("2026-10-03T14:18:00Z\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if d, err = Describe(repo.Root(), id); err != nil {
		t.Fatal(err)
	}
	if d.Reservation == nil || d.Reservation.Text != "none, as of the last fetch 12 minutes ago" {
		t.Fatalf("a stamped view with nothing renders %+v; want none with its age", d.Reservation)
	}

	// A checkout with no remote named origin carries no line.
	repo.Git("remote", "remove", "origin")
	if d, err = Describe(repo.Root(), id); err != nil || d.Reservation != nil {
		t.Fatalf("a checkout with no origin renders %+v, %v; want no reservation line", d.Reservation, err)
	}
}
