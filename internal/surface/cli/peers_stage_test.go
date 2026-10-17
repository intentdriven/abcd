package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// `abcd peers` shows where each drain lane stands and what each holder handed
// back of late (spc-2610161044461448 step 18; iss-2610161015434746;
// dcn-2610161040394970): a live reservation's lane stage beside it, and each
// holder's hand-back tombstones of the last day, newest first and bounded,
// read from the view as last fetched.

// peersLanesJSON is the stage and hand-back half of the peers --json document.
type peersLanesJSON struct {
	Reservations []struct {
		Record string `json:"record"`
		Holder string `json:"holder"`
		State  string `json:"state"`
		Stage  string `json:"stage"`
	} `json:"reservations"`
	HandBacksByHolder []struct {
		Holder    string `json:"holder"`
		HandBacks []struct {
			Record     string `json:"record"`
			Kind       string `json:"kind"`
			ReleasedAt string `json:"released_at"`
		} `json:"hand_backs"`
		More int `json:"more"`
	} `json:"hand_backs_by_holder"`
}

// TestPeersListsALanesStageAndItsHoldersRecentHandBacks: Q's drain lane
// renewed at the implement stage is listed with it, in plain and in JSON, and
// so are Q's and P's hand-backs of the last day with their kinds, newest
// first; a hand-back older than a day and one dated in the future are not
// listed, and Q's list stops at the bound with a count of the rest.
func TestPeersListsALanesStageAndItsHoldersRecentHandBacks(t *testing.T) {
	w := newReserveWorld(t)
	filer := w.clone(t, "filer", octoMail)
	const recent = 12 // past the bound of claim.HandBacksShown
	var ids []string
	for i := range recent + 3 {
		ids = append(ids, fmt.Sprintf("iss-%d", 100+i))
	}
	for _, id := range ids {
		writeRel(t, filer, ".abcd/work/issues/open/"+id+"-the-widget.md", "---\nid: "+id+"\n---\n# The widget\n")
	}
	gitCmd(t, filer, "add", "-A")
	gitCommit(t, filer, "commit", "-q", "-m", "file the widgets")
	gitCmd(t, filer, "push", "-q", "origin", "HEAD:main")

	now := time.Now().UTC().Truncate(time.Second)
	act := func(dir string, at time.Time, req claim.Request) {
		t.Helper()
		t.Chdir(dir)
		if _, err := claim.Reserve(dir, req, claim.Options{Now: func() time.Time { return at }}); err != nil {
			t.Fatalf("%+v at %s: %v", req, at, err)
		}
	}
	handBack := func(dir, id, kind string, at time.Time) {
		t.Helper()
		act(dir, at, claim.Request{Record: id, Footprint: []string{"a.go"}})
		act(dir, at, claim.Request{Record: id, Release: true, HandBack: kind})
	}

	q := w.clone(t, "q", quinnMail)
	p := w.clone(t, "p", octoMail)
	// Q's twelve hand-backs of the last hour, released out of id order: ids[i]
	// is released ago(i) before now, (i*5)%12 minutes past the hour, so
	// iss-100 is the newest, iss-102 and iss-107 the two oldest, which the
	// bound cuts, and neither id order nor its reverse is the order shown.
	ago := func(i int) time.Duration { return time.Hour + time.Duration(i*5%recent)*time.Minute }
	kindOf := func(i int) string { return claim.HandBackKinds[i%len(claim.HandBackKinds)] }
	for i := range recent {
		handBack(q, ids[i], kindOf(i), now.Add(-ago(i)))
	}
	newest := make([]int, recent) // the indexes of ids, newest first
	for i := range recent {
		newest[i*5%recent] = i
	}
	old, ahead, ps := ids[recent], ids[recent+1], ids[recent+2]
	handBack(q, old, claim.HandBackStuck, now.Add(-30*time.Hour))
	handBack(p, ps, claim.HandBackTrustRule, now.Add(-2*time.Hour))
	// Q's lane on iss-200, renewed at the implement stage; P's plain one.
	act(q, now.Add(-10*time.Minute), claim.Request{Record: "iss-200", Footprint: []string{"b.go"}})
	act(q, now.Add(-5*time.Minute), claim.Request{Record: "iss-200", Renew: true, Stage: claim.LaneStageImplement})
	act(p, now.Add(-5*time.Minute), claim.Request{Record: "iss-201", Footprint: []string{"c.go"}})
	// Q's lane on iss-202, renewed at the validate stage five hours ago, has
	// lapsed: its stage is shown on no stream.
	act(q, now.Add(-5*time.Hour-time.Minute), claim.Request{Record: "iss-202", Footprint: []string{"d.go"}})
	act(q, now.Add(-5*time.Hour), claim.Request{Record: "iss-202", Renew: true, Stage: claim.LaneStageValidate})
	// Last: a hand-back dated three hours ahead of every clock here.
	handBack(q, ahead, claim.HandBackClosed, now.Add(3*time.Hour))

	r := w.clone(t, "r", "5+rae@users.noreply.github.com")
	code, raw, stderr := reserveRun(t, r, "peers", "--json")
	if code != 0 {
		t.Fatalf("peers --json: exit %d\n%s%s", code, raw, stderr)
	}
	var got peersLanesJSON
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("peers --json is not one JSON document: %v\n%s", err, raw)
	}
	stages := map[string]string{}
	for _, res := range got.Reservations {
		stages[res.Record] = res.Stage
		if res.Record == "iss-200" && (res.State != "live" || res.Holder != "quinn") {
			t.Errorf("iss-200 = %+v; want Q's live lane", res)
		}
		if res.Record == "iss-202" && res.State != "expired" {
			t.Errorf("iss-202 = %+v; want Q's lapsed lane", res)
		}
	}
	if stages["iss-200"] != "implement" || stages["iss-201"] != "" || stages["iss-202"] != "" {
		t.Errorf("the stages = %v; want iss-200 at implement, and iss-201 and the lapsed iss-202 at none", stages)
	}

	byHolder := map[string]int{}
	for i, h := range got.HandBacksByHolder {
		byHolder[h.Holder] = i
	}
	if len(got.HandBacksByHolder) != 2 {
		t.Fatalf("hand_backs_by_holder = %+v; want octo's and quinn's", got.HandBacksByHolder)
	}
	qh := got.HandBacksByHolder[byHolder["quinn"]]
	if len(qh.HandBacks) != claim.HandBacksShown || qh.More != recent-claim.HandBacksShown {
		t.Fatalf("quinn's hand-backs = %d shown, %d more; want %d and %d", len(qh.HandBacks), qh.More, claim.HandBacksShown, recent-claim.HandBacksShown)
	}
	for n, hb := range qh.HandBacks {
		i := newest[n]
		want := now.Add(-ago(i)).Format(time.RFC3339)
		if hb.Record != ids[i] || hb.Kind != kindOf(i) || hb.ReleasedAt != want {
			t.Errorf("quinn's hand-back %d = %+v; want %s as %s at %s, newest first", n, hb, ids[i], kindOf(i), want)
		}
		if hb.Record == old || hb.Record == ahead {
			t.Errorf("%s is listed, but it was handed back a day ago or ahead of the clock: %+v", hb.Record, hb)
		}
	}
	ph := got.HandBacksByHolder[byHolder["octo"]]
	if len(ph.HandBacks) != 1 || ph.HandBacks[0].Record != ps || ph.HandBacks[0].Kind != "trust-rule" || ph.More != 0 {
		t.Errorf("octo's hand-backs = %+v; want %s as trust-rule alone", ph, ps)
	}

	code, text, stderr := reserveRun(t, r, "peers")
	if code != 0 {
		t.Fatalf("peers: exit %d\n%s%s", code, text, stderr)
	}
	line := func(prefix string) string {
		for _, l := range strings.Split(text, "\n") {
			if strings.HasPrefix(l, prefix) {
				return l
			}
		}
		return ""
	}
	if l := line("  iss-200  live  quinn"); !strings.Contains(l, "stage implement") {
		t.Errorf("Q's lane is not listed with its stage: %q\n%s", l, text)
	}
	if l := line("  iss-201  live  octo"); l == "" || strings.Contains(l, "stage") {
		t.Errorf("P's plain reservation is listed with a stage: %q\n%s", l, text)
	}
	if l := line("  iss-202  expired  quinn"); l == "" || strings.Contains(l, "stage") {
		t.Errorf("Q's lapsed lane is not listed, or is listed with a stage: %q\n%s", l, text)
	}
	for _, want := range []string{
		"  handed back by quinn in the last day:\n    " + ids[newest[0]] + "  " + kindOf(newest[0]) + "  at " + now.Add(-ago(newest[0])).Format(time.RFC3339) + "\n" +
			"    " + ids[newest[1]] + "  " + kindOf(newest[1]) + "  at " + now.Add(-ago(newest[1])).Format(time.RFC3339) + "\n",
		fmt.Sprintf("    and %d more\n", recent-claim.HandBacksShown),
		"  handed back by octo in the last day:\n    " + ps + "  trust-rule  at ",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the text listing lacks %q:\n%s", want, text)
		}
	}
	for _, id := range []string{old, ahead, ids[newest[recent-1]], ids[newest[recent-2]]} {
		if strings.Contains(text, "    "+id+"  ") {
			t.Errorf("the text listing shows %s's hand-back:\n%s", id, text)
		}
	}
}
