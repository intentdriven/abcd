package claim

import (
	"slices"
	"strings"
	"testing"
)

// No push carries a login (iss-2610290809526033): a withdrawal of the
// person's own recorded before the holder id names their login, and every
// push of theirs that carries it names their kept id instead; one of another
// holder's still to settle is written under the marker for a holder from
// before the holder id (holderid_marker_test.go).

// newCommitsSince is every commit on origin that before does not list, as
// each message reads.
func newCommitsSince(t *testing.T, w *world, before []string) []string {
	t.Helper()
	var out []string
	for _, c := range strings.Fields(runGit(t, w.bare, "rev-list", "--all")) {
		if !slices.Contains(before, c) {
			out = append(out, runGit(t, w.bare, "log", "-1", "--format=%B", c))
		}
	}
	return out
}

// allCommits is every commit origin holds.
func allCommits(t *testing.T, w *world) []string {
	t.Helper()
	return strings.Fields(runGit(t, w.bare, "rev-list", "--all"))
}

// TestAPersonsOwnPreIDWithdrawalIsPushedUnderTheirKeptID: a release whose
// only withdrawal is octo's own, recorded before the holder id under octo's
// login, is settled by octo's next pushing act, a release reservation as the
// first act or a plain reservation: the tombstone names #11 by the kept id,
// no commit the act makes names the login, #11 is re-armed, nothing is
// refused, and a new cut's reservation is the hold every other person's
// merge waits on.
func TestAPersonsOwnPreIDWithdrawalIsPushedUnderTheirKeptID(t *testing.T) {
	for _, c := range []struct {
		name string
		act  Request
	}{
		{"a release reservation as the first act", Request{Record: "rel-v0.17.0"}},
		{"a plain reservation", Request{Record: "iss-8", Footprint: []string{"o.go"}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			w, f, octo := loginNamedEndedRelease(t)
			before := allCommits(t, w)
			res, err := octo.do(c.act)
			if err != nil {
				t.Fatalf("octo's first act: %v", err)
			}
			id := octo.keptHolderID()
			i := slices.IndexFunc(res.Settles, func(s Settle) bool { return s.Subject == "rel-v0.16.0" })
			if i < 0 || res.Settles[i].Failed != "" || !slices.Equal(numbers(res.Settles[i].Settled), []string{"#11"}) {
				t.Fatalf("settles = %+v; want #11 of rel-v0.16.0 settled, nothing refused", res.Settles)
			}
			if got := f.mutations(); len(got) == 0 || got[0] != "enablePullRequestAutoMerge #11" {
				t.Fatalf("mutations = %v; want #11 re-armed first", got)
			}
			old, ows := w.tipWithdrawals("rel-v0.16.0")
			if !old.released() || len(ows) != 1 || ows[0].By != id || ows[0].State != WithdrawnRearmed {
				t.Fatalf("rel-v0.16.0's tip = %+v; want a tombstone naming #11 re-armed by the kept id", old)
			}
			for _, m := range newCommitsSince(t, w, before) {
				if strings.Contains(strings.ToLower(m), "octo") {
					t.Fatalf("a commit the act pushed names the login:\n%s", m)
				}
			}
			if c.act.Record != "rel-v0.17.0" {
				return
			}
			cut, cws := w.tipWithdrawals("rel-v0.17.0")
			if cut.released() || cut.Holder != id || !slices.Equal(numbers(cws), []string{"#11", "#12", "#13"}) {
				t.Fatalf("rel-v0.17.0's tip = %+v; want the new cut, live under the id, holding #11-#13", cut)
			}
			if h, err := MergeHolds(octo.dir, w.clock(), octo.opts()); err != nil || h.Held() {
				t.Fatalf("octo's own merge hold = %+v, %v; want the cutting copy held by nothing", h, err)
			}
			maya := w.clone("5678+maya@users.noreply.github.com", "")
			if h, err := FetchMergeHold(maya.dir, w.clock(), maya.opts()); err != nil || len(h.Holds) != 1 || h.Holds[0].Subject != "rel-v0.17.0" {
				t.Fatalf("maya's merge hold = %+v, %v; want rel-v0.17.0 holding it", h, err)
			}
		})
	}
}
