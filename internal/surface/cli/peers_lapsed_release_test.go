package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// TestPeersNamesALapsedReleaseAsHoldingNoMerge (iss-2610241104098581): a
// release whose cut stopped renewing two days ago, with no tag on origin, is
// not listed as a hold; peers names it as a lapsed cut holding no merge, with
// how to resume it, in lapsed_releases and in the text form.
func TestPeersNamesALapsedReleaseAsHoldingNoMerge(t *testing.T) {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	at := time.Now().UTC().Truncate(time.Second).Add(-48 * time.Hour)
	t.Chdir(p)
	if _, err := claim.Reserve(p, claim.Request{Record: "rel-v0.16.0"}, claim.Options{Now: func() time.Time { return at }}); err != nil {
		t.Fatal(err)
	}
	r := w.clone(t, "r", quinnMail)
	code, raw, stderr := reserveRun(t, r, "peers", "--json")
	if code != 0 {
		t.Fatalf("peers --json: exit %d\n%s%s", code, raw, stderr)
	}
	var got struct {
		ReleaseHolds   []json.RawMessage `json:"release_holds"`
		LapsedReleases []struct {
			Release   string `json:"release"`
			Subject   string `json:"subject"`
			Holder    string `json:"holder"`
			RenewedAt string `json:"renewed_at"`
			LapsedAt  string `json:"lapsed_at"`
			Text      string `json:"text"`
		} `json:"lapsed_releases"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("peers --json is not one JSON document: %v\n%s", err, raw)
	}
	if len(got.ReleaseHolds) != 0 || len(got.LapsedReleases) != 1 {
		t.Fatalf("release_holds = %s, lapsed_releases = %+v; want no hold and one lapsed cut", got.ReleaseHolds, got.LapsedReleases)
	}
	l := got.LapsedReleases[0]
	if l.Release != "v0.16.0" || l.Subject != "rel-v0.16.0" || l.Holder != "octo" || l.RenewedAt != at.Format(time.RFC3339) ||
		!strings.Contains(l.Text, "no merge is held") {
		t.Fatalf("lapsed row = %+v", l)
	}
	code, text, stderr := reserveRun(t, r, "peers")
	if code != 0 {
		t.Fatalf("peers: exit %d\n%s%s", code, text, stderr)
	}
	if !strings.Contains(text, "\n  "+l.Text+"\n") || strings.Contains(text, "merges are held") {
		t.Fatalf("the listing names the lapsed cut and no hold:\n%s", text)
	}
}
