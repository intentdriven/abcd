package claim

import (
	"strings"
	"testing"
	"time"
)

// TestARenewalFromAnyWorktreeKeepsTheHoldersCopy (iss-2611220942285216): a
// renewal asked for from any worktree of the checkout (Request.AnyWorktree)
// renews the person's own reservation that a sibling worktree holds, keeping
// its footprint and the holding worktree's copy, so the reservation stays that
// worktree's and its own renewals go on from there. It still refuses another
// person's reservation and the person's copy on another computer, and a plain
// renewal from the sibling is still refused as another copy's.
func TestARenewalFromAnyWorktreeKeepsTheHoldersCopy(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	sib := p.worktree("lane-b")
	// Another clone of the same person: a copy elsewhere, not a worktree.
	far := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")

	if _, err := p.reserve("iss-5", "a.go", "b.go"); err != nil {
		t.Fatal(err)
	}
	before := w.remoteTip("iss-5")
	copyLine := ""
	for _, l := range strings.Split(w.remoteMessage(before), "\n") {
		if strings.HasPrefix(l, keyCopy+": ") {
			copyLine = l
		}
	}
	if copyLine == "" {
		t.Fatal("the reservation names no copy")
	}

	w.set(t0.Add(time.Hour))
	notHeld(t, func() error { _, err := sib.do(Request{Record: "iss-5", Renew: true}); return err }(), NotHeldAnotherCopy)

	res, err := sib.do(Request{Record: "iss-5", Renew: true, AnyWorktree: true})
	if err != nil || res.Act != ActRenewed {
		t.Fatalf("the sibling worktree renews the person's own reservation: %+v %v", res, err)
	}
	if strings.Join(res.Footprint, ",") != "a.go,b.go" || !res.ExpiresAt.Equal(t0.Add(3*time.Hour)) {
		t.Fatalf("the renewal keeps the footprint and counts the lease from now: %+v", res)
	}
	if res.Copy != CopyAnother || !strings.Contains(res.HeldFrom, "lane") && !strings.Contains(res.HeldFrom, "copy") {
		t.Fatalf("the result names the worktree that holds it: %+v", res)
	}
	tip := w.remoteTip("iss-5")
	if tip == before || w.parentOf(tip) != before {
		t.Fatal("the renewal is not the tip's child")
	}
	if !strings.Contains(w.remoteMessage(tip), copyLine+"\n") {
		t.Fatalf("the renewal moved the reservation off the holding worktree's copy:\n%s", w.remoteMessage(tip))
	}
	if got := pushedHere(p.dir, "iss-5"); got != tip {
		t.Fatalf("the holding worktree's note of its last push = %q; want the renewal %q", got, tip)
	}

	// The holding worktree goes on renewing it as its own.
	w.set(t0.Add(2 * time.Hour))
	if res, err := p.do(Request{Record: "iss-5", Renew: true, Live: true}); err != nil || res.Copy != CopyThis {
		t.Fatalf("the holding worktree renews it after: %+v %v", res, err)
	}
	tip = w.remoteTip("iss-5")

	// The holding worktree itself renews with AnyWorktree as a plain renewal.
	if res, err := p.do(Request{Record: "iss-5", Renew: true, AnyWorktree: true}); err != nil || res.Copy != CopyThis {
		t.Fatalf("AnyWorktree from the holding worktree is a plain renewal: %+v %v", res, err)
	}
	tip = w.remoteTip("iss-5")

	notHeld(t, func() error {
		_, err := far.do(Request{Record: "iss-5", Renew: true, AnyWorktree: true})
		return err
	}(), NotHeldAnotherCopy)
	notHeld(t, func() error {
		_, err := q.do(Request{Record: "iss-5", Renew: true, AnyWorktree: true})
		return err
	}(), NotHeldAnotherPerson)
	if w.remoteTip("iss-5") != tip {
		t.Fatal("a refused renewal moved the branch")
	}

	if _, err := sib.do(Request{Record: "iss-5", AnyWorktree: true, Footprint: []string{"c.go"}}); err == nil {
		t.Fatal("AnyWorktree without a renewal is accepted")
	}
}
