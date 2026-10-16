package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// The `abcd status` front door and the prompt hook's status block
// (spc-2610150728396006 step 3, itd-2609151838312703): the verb fetches once
// and lists everyone's status in text and --json, and the hook shows each new
// announcement once, from disk alone. The reader itself is proved in
// internal/core/claim; these tests prove the surfaces drive it and say what
// it read.

type statusJSON struct {
	View struct {
		Origin     bool   `json:"origin"`
		FetchedAt  string `json:"fetched_at"`
		AgeSeconds *int64 `json:"age_seconds"`
	} `json:"status_view"`
	Announcements []struct {
		Branch    string `json:"branch"`
		Sender    string `json:"sender"`
		Kind      string `json:"kind"`
		Item      string `json:"item"`
		SentAt    string `json:"sent_at"`
		Until     string `json:"until"`
		Check     string `json:"check"`
		Text      string `json:"text"`
		Commit    string `json:"commit"`
		Doubtful  string `json:"doubtful"`
		Unreadble string `json:"unreadable"`
	} `json:"announcements"`
	Doubtful []struct {
		Branch string `json:"branch"`
		Reason string `json:"reason"`
		Text   string `json:"text"`
	} `json:"doubtful"`
	Unread           []string `json:"unread"`
	SendersUnchecked string   `json:"senders_unchecked"`
	Reservations     []struct {
		Record string `json:"record"`
		Holder string `json:"holder"`
	} `json:"reservations"`
	ReleaseHolds []struct {
		Release string `json:"release"`
		Text    string `json:"text"`
	} `json:"release_holds"`
	FetchError string `json:"fetch_error"`
}

// TestStatusListsEveryoneAndSaysWhatItRead is C16 and C8 at the surface: one
// fetch, then the announcements newest first with sender, kind, item, time and
// how long each stands, the reservations and the release being cut, in text
// and in --json, with the line that an announcement grants nothing.
func TestStatusListsEveryoneAndSaysWhatItRead(t *testing.T) {
	w := newReserveWorld(t)
	octo := w.clone(t, "octo", octoMail)
	quinn := w.clone(t, "quinn", quinnMail)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	if code, _, stderr := reserveRun(t, octo, "announce", "merging", "1012"); code != 0 {
		t.Fatalf("announce: exit %d, %s", code, stderr)
	}
	if code, _, stderr := reserveRun(t, quinn, "reserve", "rel-v0.16.0"); code != 0 {
		t.Fatalf("reserve the release: exit %d, %s", code, stderr)
	}

	code, stdout, stderr := reserveRun(t, pat, "status")
	if code != 0 {
		t.Fatalf("status: exit %d, stderr %q", code, stderr)
	}
	for _, want := range []string{
		"abcd status — announcements as of the fetch 0m ago (",
		"\n  @octo (claimed): merging #1012, for 30 min  sent ",
		"\nreservations — as of the fetch 0m ago",
		"\n  rel-v0.16.0  live  quinn  until ",
		"\n  release v0.16.0 being cut by @quinn: merges are held\n",
		"\n" + claim.GrantsNothing + "\n",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("status:\n%s\nwant it to carry %q", stdout, want)
		}
	}
	if strings.Contains(stdout, w.home) || strings.Contains(stdout, "Abcd-") {
		t.Fatalf("the render carries the home directory or a commit's raw text:\n%s", stdout)
	}

	code, stdout, stderr = reserveRun(t, pat, "status", "--json")
	if code != 0 {
		t.Fatalf("status --json: exit %d, stderr %q", code, stderr)
	}
	var got statusJSON
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("--json is not one JSON document: %v\n%s", err, stdout)
	}
	if !got.View.Origin || got.View.FetchedAt == "" || got.View.AgeSeconds == nil || len(got.Announcements) != 1 {
		t.Fatalf("--json view %+v, announcements %+v", got.View, got.Announcements)
	}
	a := got.Announcements[0]
	if a.Branch != "abcd-status/octo" || a.Sender != "octo" || a.Kind != "merging" || a.Item != "#1012" || a.Check != claim.SenderClaimed ||
		a.SentAt == "" || a.Until == "" || a.Commit == "" || a.Text != "@octo (claimed): merging #1012, for 30 min" {
		t.Fatalf("--json announcement %+v", a)
	}
	if len(got.ReleaseHolds) != 1 || got.ReleaseHolds[0].Text != "release v0.16.0 being cut by @quinn: merges are held" ||
		len(got.Reservations) != 1 || got.Reservations[0].Record != "rel-v0.16.0" {
		t.Fatalf("--json holds %+v, reservations %+v", got.ReleaseHolds, got.Reservations)
	}
	if !strings.Contains(got.SendersUnchecked, "github.com") {
		t.Fatalf("senders_unchecked = %q; want the reason a local origin cannot say who pushed", got.SendersUnchecked)
	}
}

// TestStatusSaysAFailedFetchAndExitsOne is C10 at the surface: the view is
// listed as last fetched, says the status is unread since the last read and
// why, and the verb exits 1; with no shared remote it says so and exits 0.
func TestStatusSaysAFailedFetchAndExitsOne(t *testing.T) {
	w := newReserveWorld(t)
	pat := w.clone(t, "pat", "5+pat@users.noreply.github.com")
	if code, stdout, _ := reserveRun(t, pat, "status"); code != 0 || !strings.Contains(stdout, "\n  none standing\n") {
		t.Fatalf("an empty status after a fetch that succeeded: exit %d\n%s", code, stdout)
	}
	gitCmd(t, pat, "remote", "set-url", "origin", filepath.Join(w.home, "gone.git"))
	code, stdout, stderr := reserveRun(t, pat, "status")
	if code != 1 || !strings.Contains(stdout, "\n  status unread since ") || !strings.Contains(stderr, "abcd status: could not read") {
		t.Fatalf("a failed fetch: exit %d\nstdout %s\nstderr %s", code, stdout, stderr)
	}
	if strings.Contains(stdout, w.home) || strings.Contains(stderr, w.home) {
		t.Fatalf("the failure carries the home directory:\n%s\n%s", stdout, stderr)
	}

	lone := filepath.Join(w.home, "lone")
	gitInitAt(t, lone)
	if err := os.MkdirAll(filepath.Join(lone, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, stdout, stderr := reserveRun(t, lone, "status"); code != 0 || !strings.Contains(stdout, "this checkout has no remote named origin") {
		t.Fatalf("no origin: exit %d\nstdout %s\nstderr %s", code, stdout, stderr)
	}
	if code, _, stderr := reserveRun(t, w.home, "status"); code != 2 || !strings.Contains(stderr, "abcd status:") {
		t.Fatalf("outside a checkout: exit %d, %s", code, stderr)
	}
	if code, _, stderr := reserveRun(t, pat, "status", "extra"); code != 2 || !strings.Contains(stderr, "abcd status") {
		t.Fatalf("an operand: exit %d, %s", code, stderr)
	}
}
