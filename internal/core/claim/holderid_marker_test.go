package claim

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// An entry another holder recorded before the holder id names that holder's
// login, which nothing abcd pushes carries (iss-2610290809526033). An act
// that would write one still to settle writes it under the one login-free
// marker (preIDMarker) instead: the entry stays on the release, it is no
// reader's own, no copy ever re-arms it, and the act that ends the release
// settles it as left to be re-armed by hand, naming its number and no login.
// A release holds one entry for each pull request.

// mayaPreID and octoPreID are withdrawals maya and octo recorded before the
// holder id, under their logins.
var (
	octoPreID = Withdrawal{Number: 11, ID: "PR_eleven", Form: FormAutoSquash, Head: headA, By: "octo", State: WithdrawnPending}
	mayaPreID = Withdrawal{Number: 12, ID: "PR_twelve", Form: FormQueue, Head: headB, By: "maya", State: WithdrawnPending}
)

// quietForge is a forge with #11, #12 and #13 open and armed for nothing,
// as a cut's withdrawals leave them, and its comment writer set.
func quietForge(t *testing.T) *fakeForge {
	t.Helper()
	f := newFakeForge(t, map[int]*fakePR{
		11: {id: "PR_eleven", head: headA},
		12: {id: "PR_twelve", head: headB},
		13: {id: "PR_thirteen", head: headC},
	})
	releaseForge.comments = f
	return f
}

// octosTwoLoginCut pushes rel-v0.16.0 as octo reserved it before the holder
// id, from copy, recording octo's own #11 and #12 under maya's login, in one
// commit the forge's record says octo pushed.
func octosTwoLoginCut(w *world, copy string) string {
	m := message{Record: "rel-v0.16.0", Holder: "octo", Copy: copy, ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: []Withdrawal{octoPreID, mayaPreID}, AssistedBy: "abcd:0.0.0"}
	return w.pushTipAt("rel-v0.16.0", m.text(), w.clock())
}

// actedOn is each call that re-armed or withdrew pull request n, or
// commented on it.
func actedOn(f *fakeForge, n int) []string {
	var out []string
	suffix, prefix := " #"+strconv.Itoa(n), "#"+strconv.Itoa(n)+" "
	for _, m := range f.mutations() {
		if strings.HasSuffix(m, suffix) {
			out = append(out, m)
		}
	}
	for _, c := range f.posted() {
		if strings.HasPrefix(c, prefix) {
			out = append(out, c)
		}
	}
	return out
}

// settleOf is res's settle of id, failing the test when there is none.
func settleOf(t *testing.T, res Result, id string) Settle {
	t.Helper()
	i := slices.IndexFunc(res.Settles, func(s Settle) bool { return s.Subject == id })
	if i < 0 {
		t.Fatalf("settles = %+v; want one of %s", res.Settles, id)
	}
	return res.Settles[i]
}

// wantByHand fails unless s names exactly ns as left to be re-armed by hand,
// on one line naming each number and no login.
func wantByHand(t *testing.T, s Settle, ns ...string) {
	t.Helper()
	if !slices.Equal(numbers(s.ByHand), ns) {
		t.Fatalf("settle of %s = %+v; want %v left to be re-armed by hand", s.Subject, s, ns)
	}
	for _, w := range s.ByHand {
		if w.By != preIDMarker || w.State != WithdrawnManual {
			t.Fatalf("by hand = %+v; want each under the marker, settled %s", s.ByHand, WithdrawnManual)
		}
	}
	var line string
	for _, l := range s.Notes() {
		if strings.Contains(l, "by hand") {
			line = l
		}
	}
	for _, n := range ns {
		if !strings.Contains(line, n) {
			t.Fatalf("notes = %q; want one line naming %v to be re-armed by hand", s.Notes(), ns)
		}
	}
	noLogin(t, "the by-hand line", line)
}

// noLogin fails when text names maya's or octo's login.
func noLogin(t *testing.T, what, text string) {
	t.Helper()
	for _, login := range []string{"maya", "octo", "quinn"} {
		if strings.Contains(strings.ToLower(text), login) {
			t.Fatalf("%s names the login %s:\n%s", what, login, text)
		}
	}
}

// noLoginPushedSince fails when a commit origin gained since before, a ref
// name, or a comment the forge was sent names a login.
func noLoginPushedSince(t *testing.T, w *world, f *fakeForge, before []string) {
	t.Helper()
	noLogin(t, "a commit pushed", strings.Join(newCommitsSince(t, w, before), "\n"))
	noLogin(t, "a ref name", runGit(t, w.bare, "for-each-ref", "--format=%(refname)"))
	noLogin(t, "a comment", strings.Join(f.posted(), "\n"))
}

// tipEntries is the withdrawals origin's id tip records, failing the test
// when it does not read or names a pull request twice.
func tipEntries(t *testing.T, w *world, id string) []Withdrawal {
	t.Helper()
	msg := w.remoteMessage(w.remoteTip(id))
	m, why := parseMessage(id, msg)
	if why != "" {
		t.Fatalf("origin's %s tip does not read (%s):\n%s", id, why, msg)
	}
	return m.Withdrawn
}

// TestAnotherHoldersPreIDEntryNeverLocksTheRelease: rel-v0.16.0, reserved
// before the holder id by octo, records octo's #11 and #12 under maya's
// login. From either holder's side, in either order, the release settles,
// renews and releases: octo's #11 is re-armed under octo's kept id, #12 is
// written under the marker, never re-armed, and settled as left to be
// re-armed by hand when the release ends, and no act pushes or posts a login.
func TestAnotherHoldersPreIDEntryNeverLocksTheRelease(t *testing.T) {
	tagged := func(t *testing.T) (*world, *fakeForge, *participant, *participant, []string) {
		w := newWorld(t)
		f := quietForge(t)
		octosTwoLoginCut(w, "9f2c41d07be35a6c80e1f4b2a7d39c55")
		w.tagOrigin("v0.16.0")
		octo := w.clone("1234+octo@users.noreply.github.com", "")
		maya := w.clone("5678+maya@users.noreply.github.com", "")
		keepHolderID(t, octo.home, octoKept)
		keepHolderID(t, maya.home, mayaB)
		return w, f, octo, maya, allCommits(t, w)
	}
	octoSettles := func(t *testing.T, w *world, f *fakeForge, octo *participant) {
		t.Helper()
		res, err := octo.reserve("iss-8", "o.go")
		if err != nil {
			t.Fatalf("octo's act: %v", err)
		}
		s := settleOf(t, res, "rel-v0.16.0")
		if s.Failed != "" || !slices.Equal(numbers(s.Settled), []string{"#11"}) {
			t.Fatalf("settle = %+v; want #11 re-armed and nothing refused", s)
		}
		wantByHand(t, s, "#12")
		ws := tipEntries(t, w, "rel-v0.16.0")
		if len(ws) != 2 || stateOf(ws, 11) != WithdrawnRearmed || stateOf(ws, 12) != WithdrawnManual {
			t.Fatalf("tip records %+v; want #11 re-armed and #12 left by hand", ws)
		}
		for _, e := range ws {
			if want := map[int]string{11: octoKept, 12: preIDMarker}[e.Number]; e.By != want {
				t.Fatalf("tip records %+v; want #11 under octo's kept id and #12 under the marker", ws)
			}
		}
		if got := actedOn(f, 12); len(got) != 0 {
			t.Fatalf("#12 was acted on: %v; want it never re-armed", got)
		}
	}
	t.Run("octo settles the tagged release", func(t *testing.T) {
		w, f, octo, _, before := tagged(t)
		octoSettles(t, w, f, octo)
		noLoginPushedSince(t, w, f, before)
	})
	t.Run("maya acts first, then octo settles", func(t *testing.T) {
		w, f, octo, maya, before := tagged(t)
		tip := w.remoteTip("rel-v0.16.0")
		res, err := maya.reserve("iss-9", "m.go")
		if err != nil {
			t.Fatalf("maya's act: %v", err)
		}
		for _, s := range res.Settles {
			if s.Failed != "" {
				t.Fatalf("maya's settle = %+v; want nothing refused", s)
			}
		}
		if w.remoteTip("rel-v0.16.0") != tip || len(actedOn(f, 12)) != 0 {
			t.Fatalf("maya's act moved the release or touched #12 (%v); want neither: octo's push recorded #12, not hers", actedOn(f, 12))
		}
		octoSettles(t, w, f, octo)
		if _, err := maya.reserve("iss-10", "n.go"); err != nil {
			t.Fatalf("maya's act after octo's settle: %v", err)
		}
		if got := actedOn(f, 12); len(got) != 0 {
			t.Fatalf("#12 was acted on: %v; want it never re-armed", got)
		}
		noLoginPushedSince(t, w, f, before)
	})
	t.Run("octo renews and releases the live cut", func(t *testing.T) {
		w := newWorld(t)
		f := quietForge(t)
		octo := w.clone("1234+octo@users.noreply.github.com", "")
		keepHolderID(t, octo.home, octoKept)
		if _, err := octo.reserve("iss-1", "a.go"); err != nil {
			t.Fatal(err)
		}
		octosTwoLoginCut(w, ownCopyKey(octo.dir))
		before := allCommits(t, w)
		res, err := octo.do(Request{Record: "rel-v0.16.0", Renew: true})
		if err != nil || res.Act != ActRenewed {
			t.Fatalf("octo's renewal = %+v, %v; want it renewed", res, err)
		}
		ws := tipEntries(t, w, "rel-v0.16.0")
		if len(ws) != 2 || stateOf(ws, 12) != WithdrawnPending {
			t.Fatalf("renewal records %+v; want #11 and #12 still withdrawn", ws)
		}
		if e, _ := withdrawalOf(ws, 12); e.By != preIDMarker {
			t.Fatalf("renewal records %+v; want #12 under the marker", ws)
		}
		res, err = octo.do(Request{Record: "rel-v0.16.0", Release: true})
		if err != nil {
			t.Fatalf("octo's release: %v", err)
		}
		s := settleOf(t, res, "rel-v0.16.0")
		if s.Failed != "" || !slices.Equal(numbers(s.Settled), []string{"#11"}) {
			t.Fatalf("release's settle = %+v; want #11 re-armed", s)
		}
		wantByHand(t, s, "#12")
		if ws := tipEntries(t, w, "rel-v0.16.0"); stateOf(ws, 12) != WithdrawnManual {
			t.Fatalf("release records %+v; want #12 left by hand", ws)
		}
		if got := actedOn(f, 12); len(got) != 0 {
			t.Fatalf("#12 was acted on: %v; want it never re-armed", got)
		}
		maya := w.clone("5678+maya@users.noreply.github.com", "")
		if _, err := maya.do(Request{Record: "rel-v0.17.0"}); err != nil {
			t.Fatalf("a new cut after the release: %v; want it reserved", err)
		}
		noLoginPushedSince(t, w, f, before)
	})
	t.Run("maya takes over the lapsed cut and releases it", func(t *testing.T) {
		w := newWorld(t)
		f := quietForge(t)
		octosTwoLoginCut(w, "9f2c41d07be35a6c80e1f4b2a7d39c55")
		before := allCommits(t, w)
		w.set(w.clock().Add(2 * time.Hour))
		maya := w.clone("5678+maya@users.noreply.github.com", "")
		keepHolderID(t, maya.home, mayaB)
		if _, err := maya.do(Request{Record: "rel-v0.16.0"}); err != nil {
			t.Fatalf("maya's take-over: %v; want the lapsed cut taken over", err)
		}
		if e, ok := withdrawalOf(tipEntries(t, w, "rel-v0.16.0"), 11); !ok || e.By != preIDMarker || !e.Pending() {
			t.Fatalf("take-over records %+v; want octo's #11 under the marker, still withdrawn", tipEntries(t, w, "rel-v0.16.0"))
		}
		res, err := maya.do(Request{Record: "rel-v0.16.0", Release: true})
		if err != nil {
			t.Fatalf("maya's release: %v", err)
		}
		wantByHand(t, settleOf(t, res, "rel-v0.16.0"), "#11")
		if got := actedOn(f, 11); len(got) != 0 {
			t.Fatalf("#11 was acted on: %v; want it never re-armed", got)
		}
		// octo finds his own #11 through his own push, not the tip, and
		// re-arms it: his entry wins over the marker one, which is dropped.
		octo := w.clone("1234+octo@users.noreply.github.com", "")
		keepHolderID(t, octo.home, octoKept)
		res, err = octo.reserve("iss-8", "o.go")
		if err != nil {
			t.Fatalf("octo's act after the release: %v", err)
		}
		if s := settleOf(t, res, "rel-v0.16.0"); len(s.Settled) != 1 || s.Settled[0].By != octoKept || s.Settled[0].State != WithdrawnRearmed {
			t.Fatalf("octo's settle = %+v; want his own #11 re-armed under his kept id", s)
		}
		if got := actedOn(f, 11); len(got) != 2 || got[0] != "enablePullRequestAutoMerge #11" {
			t.Fatalf("#11 was acted on: %v; want it re-armed once, by octo's own act, and commented on", got)
		}
		if ws := tipEntries(t, w, "rel-v0.16.0"); len(ws) != 1 || ws[0].By != octoKept {
			t.Fatalf("tip records %+v; want #11 once, under octo's kept id", ws)
		}
		noLoginPushedSince(t, w, f, before)
	})
}

// mayasPreIDCutTakenOver is rel-v0.16.0 as maya reserved it before the holder
// id, pushed by maya, recording #12 under her login, then, lapsed, taken over
// by octo, whose take-over writes #12 under the marker. Maya's home keeps
// mayaB, octo's octoKept.
func mayasPreIDCutTakenOver(t *testing.T) (*world, *fakeForge, *participant, *participant, []string) {
	t.Helper()
	w := newWorld(t)
	f := quietForge(t)
	m := message{Record: "rel-v0.16.0", Holder: "maya", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: []Withdrawal{mayaPreID}, AssistedBy: "abcd:0.0.0"}
	w.pushTipAt("rel-v0.16.0", m.text(), w.clock())
	before := allCommits(t, w)
	w.set(w.clock().Add(2 * time.Hour))
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	keepHolderID(t, octo.home, octoKept)
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatalf("octo's take-over: %v", err)
	}
	if e, _ := withdrawalOf(tipEntries(t, w, "rel-v0.16.0"), 12); e.By != preIDMarker {
		t.Fatalf("take-over records %+v; want #12 under the marker", tipEntries(t, w, "rel-v0.16.0"))
	}
	maya := w.clone("5678+maya@users.noreply.github.com", "")
	keepHolderID(t, maya.home, mayaB)
	return w, f, octo, maya, before
}

// TestTheOriginalHolderReArmsTheirOwnEntryOverTheMarker: maya's pre-id cut
// is taken over by octo, which writes her #12 under the marker. Once the
// release ends, by octo's release or by the tag, maya's next act finds #12
// through her own push, not the tip, re-arms it, and pushes a tombstone that
// lists #12 once, under her kept id, and reads; the marker entry is never
// re-armed, and nothing pushed or posted names a login.
func TestTheOriginalHolderReArmsTheirOwnEntryOverTheMarker(t *testing.T) {
	for _, c := range []struct {
		name string
		end  func(t *testing.T, w *world, octo *participant)
	}{
		{"after the taker's release", func(t *testing.T, w *world, octo *participant) {
			res, err := octo.do(Request{Record: "rel-v0.16.0", Release: true})
			if err != nil {
				t.Fatalf("octo's release: %v", err)
			}
			wantByHand(t, settleOf(t, res, "rel-v0.16.0"), "#12")
		}},
		{"with the tag on origin before the taker releases", func(t *testing.T, w *world, octo *participant) {
			w.tagOrigin("v0.16.0")
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			w, f, octo, maya, before := mayasPreIDCutTakenOver(t)
			c.end(t, w, octo)
			if got := actedOn(f, 12); len(got) != 0 {
				t.Fatalf("#12 was acted on before maya's act: %v; want the marker entry never re-armed", got)
			}
			res, err := maya.reserve("iss-9", "m.go")
			if err != nil {
				t.Fatalf("maya's act: %v", err)
			}
			s := settleOf(t, res, "rel-v0.16.0")
			if s.Failed != "" || len(s.Settled) != 1 || s.Settled[0].Number != 12 || s.Settled[0].By != mayaB || s.Settled[0].State != WithdrawnRearmed {
				t.Fatalf("maya's settle = %+v; want #12 re-armed under her kept id", s)
			}
			if got := f.mutations(); !slices.Equal(got, []string{"enqueuePullRequest #12"}) {
				t.Fatalf("mutations = %v; want #12 queued again, once", got)
			}
			msg := w.remoteMessage(w.remoteTip("rel-v0.16.0"))
			if n := strings.Count(msg, keyWithdrawn+": 12 "); n != 1 {
				t.Fatalf("maya's tombstone lists #12 %d times; want once:\n%s", n, msg)
			}
			ws := tipEntries(t, w, "rel-v0.16.0")
			if e, _ := withdrawalOf(ws, 12); e.By != mayaB || e.State != WithdrawnRearmed {
				t.Fatalf("tip records %+v; want #12 re-armed under maya's kept id", ws)
			}
			noLoginPushedSince(t, w, f, before)
		})
	}
}

// TestNoPushedTextNamesTheLoginOfAPreIDEntry: across maya's pre-id cut taken
// over by octo, octo's renewal and release, and maya's settle, no commit
// message or trailer, no ref name and no comment body names either login.
func TestNoPushedTextNamesTheLoginOfAPreIDEntry(t *testing.T) {
	w, f, octo, maya, before := mayasPreIDCutTakenOver(t)
	for _, req := range []Request{{Record: "rel-v0.16.0", Renew: true}, {Record: "rel-v0.16.0", Release: true}} {
		if _, err := octo.do(req); err != nil {
			t.Fatalf("octo's %+v: %v", req, err)
		}
	}
	if _, err := maya.reserve("iss-9", "m.go"); err != nil {
		t.Fatalf("maya's act: %v", err)
	}
	if len(f.posted()) == 0 {
		t.Fatal("no comment was posted to check")
	}
	noLoginPushedSince(t, w, f, before)
}

// TestAMarkerEntryIsReArmedByNoReaderAndListedByHand: a tagged release
// records quinn's #11 under quinn's kept id and #12 under the marker, in one
// commit quinn pushed. No reader's act re-arms #12, octo's, maya's and
// quinn's own alike; until the release's settle, a listing names #12 to be
// re-armed by hand; quinn's settle re-arms #11 and settles #12 as left by
// hand, and a listing names it so after.
func TestAMarkerEntryIsReArmedByNoReaderAndListedByHand(t *testing.T) {
	w := newWorld(t)
	f := quietForge(t)
	marker := Withdrawal{Number: 12, ID: "PR_twelve", Form: FormQueue, Head: headB, By: preIDMarker, State: WithdrawnPending}
	own := Withdrawal{Number: 11, ID: "PR_eleven", Form: FormAutoSquash, Head: headA, By: quinnID, State: WithdrawnPending}
	m := message{Record: "rel-v0.16.0", Holder: quinnID, Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: []Withdrawal{own, marker}, AssistedBy: "abcd:0.0.0"}
	f.pusherOf[w.pushTipAt("rel-v0.16.0", m.text(), w.clock())] = "quinn"
	w.tagOrigin("v0.16.0")
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	maya := w.clone("5678+maya@users.noreply.github.com", "")
	quinn := w.clone("99+quinn@users.noreply.github.com", "")
	keepHolderID(t, quinn.home, quinnID)
	listedByHand := func(p *participant) {
		t.Helper()
		if _, err := Fetch(p.dir, p.opts()); err != nil {
			t.Fatal(err)
		}
		snap, err := View(p.dir, w.clock())
		if err != nil {
			t.Fatal(err)
		}
		for _, rw := range snap.ReleaseWithdrawals(p.dir, p.self(), w.clock()) {
			if rw.Subject != "rel-v0.16.0" {
				continue
			}
			lines := strings.Join(rw.Lines(), "\n")
			if !slices.Equal(numbers(rw.ByHand), []string{"#12"}) || slices.ContainsFunc(append(rw.Due, rw.Stranded...), func(w Withdrawal) bool { return w.Number == 12 }) ||
				!strings.Contains(lines, "#12") || !strings.Contains(lines, "by hand") {
				t.Fatalf("listing = %+v\n%s\nwant #12 named to be re-armed by hand, never due or stranded", rw, lines)
			}
			noLogin(t, "the listing", lines)
			return
		}
		t.Fatal("the listing names nothing of rel-v0.16.0")
	}
	for i, p := range []*participant{octo, maya} {
		if _, err := p.reserve("iss-"+strconv.Itoa(20+i), "x"+strconv.Itoa(i)+".go"); err != nil {
			t.Fatalf("an act: %v", err)
		}
		listedByHand(p)
	}
	res, err := quinn.reserve("iss-7", "q.go")
	if err != nil {
		t.Fatalf("quinn's act: %v", err)
	}
	s := settleOf(t, res, "rel-v0.16.0")
	if !slices.Equal(numbers(s.Settled), []string{"#11"}) {
		t.Fatalf("quinn's settle = %+v; want #11 re-armed", s)
	}
	wantByHand(t, s, "#12")
	for i, p := range []*participant{octo, maya, quinn} {
		if _, err := p.reserve("iss-"+strconv.Itoa(30+i), "y"+strconv.Itoa(i)+".go"); err != nil {
			t.Fatalf("an act after the settle: %v", err)
		}
		listedByHand(p)
	}
	if got := actedOn(f, 12); len(got) != 0 {
		t.Fatalf("#12 was acted on: %v; want the marker entry re-armed by no reader", got)
	}
}

// TestAMadeUpLoginOnALapsedReleaseGetsTheMarker: quinn's lapsed pre-id cut
// records #12 under a login no account pushed. octo's take-over writes it
// under the marker, the take-over names it to be re-armed by hand, octo's
// release settles it so, and nothing arms it.
func TestAMadeUpLoginOnALapsedReleaseGetsTheMarker(t *testing.T) {
	w := newWorld(t)
	f := quietForge(t)
	madeUp := mayaPreID
	madeUp.By = "rosa-nobody"
	m := message{Record: "rel-v0.16.0", Holder: "quinn", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: []Withdrawal{madeUp}, AssistedBy: "abcd:0.0.0"}
	w.pushTipAt("rel-v0.16.0", m.text(), w.clock())
	before := allCommits(t, w)
	w.set(w.clock().Add(2 * time.Hour))
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	keepHolderID(t, octo.home, octoKept)
	res, err := octo.do(Request{Record: "rel-v0.16.0"})
	if err != nil {
		t.Fatalf("octo's take-over: %v", err)
	}
	if e, _ := withdrawalOf(tipEntries(t, w, "rel-v0.16.0"), 12); e.By != preIDMarker || !e.Pending() {
		t.Fatalf("take-over records %+v; want #12 under the marker, still withdrawn", tipEntries(t, w, "rel-v0.16.0"))
	}
	note := StrandedNote("rel-v0.16.0", res.Stranded, octoKept, true)
	if !strings.Contains(note, "#12") || !strings.Contains(note, "by hand") || strings.Contains(note, "rosa") {
		t.Fatalf("take-over's stranded note = %q; want #12 named to be re-armed by hand, and no login", note)
	}
	res, err = octo.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatalf("octo's release: %v", err)
	}
	wantByHand(t, settleOf(t, res, "rel-v0.16.0"), "#12")
	if got := actedOn(f, 12); len(got) != 0 {
		t.Fatalf("#12 was acted on: %v; want it never armed", got)
	}
	if strings.Contains(strings.Join(newCommitsSince(t, w, before), "\n"), "rosa") {
		t.Fatal("a commit pushed names the made-up login")
	}
}

// TestARoundTripBetweenTwoHoldersKeepsOneEntryForAPullRequest: maya cuts
// rel-v0.16.0 and withdraws #12; octo takes the lapsed cut over and, #12
// armed again by hand, withdraws it again; maya takes the lapsed cut back.
// Her own newest push records her #12 and the tip octo's: both still to
// settle, both vouched for by their holder's own push, and hers is her own,
// which another login's entry never displaces, so hers is the one her
// take-over writes, once, and her release re-arms it. Octo still finds his
// own #12 through his own push: his next act keeps it over hers on the tip,
// finds #12 armed, and records it so, once, under his id.
func TestARoundTripBetweenTwoHoldersKeepsOneEntryForAPullRequest(t *testing.T) {
	w := newWorld(t)
	f := quietForge(t)
	maya := w.clone("5678+maya@users.noreply.github.com", "")
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	keepHolderID(t, maya.home, mayaB)
	keepHolderID(t, octo.home, octoKept)
	arm12 := func() {
		f.mu.Lock()
		f.prs[12].queued = true
		f.mu.Unlock()
	}
	arm12()
	if _, err := maya.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	w.set(w.clock().Add(25 * time.Hour))
	arm12()
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatalf("octo's take-over: %v", err)
	}
	if e, _ := withdrawalOf(tipEntries(t, w, "rel-v0.16.0"), 12); e.By != octoKept {
		t.Fatalf("octo's take-over records %+v; want #12 withdrawn again under octo's id", tipEntries(t, w, "rel-v0.16.0"))
	}
	w.set(w.clock().Add(25 * time.Hour))
	if _, err := maya.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatalf("maya's take-over: %v", err)
	}
	ws := tipEntries(t, w, "rel-v0.16.0")
	if len(ws) != 1 || ws[0].Number != 12 || ws[0].By != mayaB || !ws[0].Pending() {
		t.Fatalf("maya's take-over records %+v; want #12 once, her own", ws)
	}
	f.reset()
	res, err := maya.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatalf("maya's release: %v", err)
	}
	if s := settleOf(t, res, "rel-v0.16.0"); !slices.Equal(numbers(s.Settled), []string{"#12"}) {
		t.Fatalf("maya's release = %+v; want #12 re-armed", s)
	}
	if ws := tipEntries(t, w, "rel-v0.16.0"); len(ws) != 1 {
		t.Fatalf("release records %+v; want one entry", ws)
	}
	f.reset()
	res, err = octo.reserve("iss-8", "o.go")
	if err != nil {
		t.Fatalf("octo's act after the release: %v", err)
	}
	if s := settleOf(t, res, "rel-v0.16.0"); len(s.Settled) != 1 || s.Settled[0].By != octoKept || s.Settled[0].State != WithdrawnFoundArmed {
		t.Fatalf("octo's settle = %+v; want his own #12 found armed", s)
	}
	if ws := tipEntries(t, w, "rel-v0.16.0"); len(ws) != 1 || ws[0].By != octoKept || len(f.mutations()) != 0 {
		t.Fatalf("tip records %+v, mutations %v; want #12 once, octo's, and nothing armed twice", ws, f.mutations())
	}
}

// TestALaterUnsettledEntryAnotherLoginPushedNeverDisplacesAVouchedOne:
// maya's cut records #12 under her kept id, pushed by her; mal pushes a
// reservation over it recording #12 under mal's own id, still withdrawn, and
// the tag lands. Both are still to settle and both are vouched for, so the
// earliest recorded, maya's, is the one her settle keeps and re-arms, and
// the tombstone lists #12 once.
func TestALaterUnsettledEntryAnotherLoginPushedNeverDisplacesAVouchedOne(t *testing.T) {
	w := newWorld(t)
	f := quietForge(t)
	mine := Withdrawal{Number: 12, ID: "PR_twelve", Form: FormQueue, Head: headB, By: mayaB, State: WithdrawnPending}
	m := message{Record: "rel-v0.16.0", Holder: mayaB, Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: []Withdrawal{mine}, AssistedBy: "abcd:0.0.0"}
	f.pusherOf[w.pushTipAt("rel-v0.16.0", m.text(), w.clock())] = "maya"
	malID := "h_" + strings.Repeat("7", 32)
	theirs := mine
	theirs.By, theirs.Head = malID, headC
	m.Holder, m.Copy, m.Withdrawn = malID, "0123456789abcdef0123456789abcdef", []Withdrawal{theirs}
	w.pushAs(f, "rel-v0.16.0", "mal", m.text())
	w.tagOrigin("v0.16.0")
	maya := w.clone("5678+maya@users.noreply.github.com", "")
	keepHolderID(t, maya.home, mayaB)
	res, err := maya.reserve("iss-9", "m.go")
	if err != nil {
		t.Fatalf("maya's act: %v", err)
	}
	s := settleOf(t, res, "rel-v0.16.0")
	if len(s.Settled) != 1 || s.Settled[0].By != mayaB || s.Settled[0].Head != headB || s.Settled[0].State != WithdrawnRearmed {
		t.Fatalf("maya's settle = %+v; want her own #12 re-armed at its withdrawn head", s)
	}
	ws := tipEntries(t, w, "rel-v0.16.0")
	if len(ws) != 1 || ws[0].By != mayaB {
		t.Fatalf("tip records %+v; want #12 once, maya's", ws)
	}
}

// TestTheWriteGuardRefusesAMessageListingAPullRequestTwice: no commit abcd
// writes lists one pull request twice, here under two holder ids, nor names
// a withdrawal's holder by anything but a holder id or the marker; each is
// refused with nothing written, and the line names no holder. One entry
// under the marker is written.
func TestTheWriteGuardRefusesAMessageListingAPullRequestTwice(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	tree, err := emptyTree(p.dir)
	if err != nil {
		t.Fatal(err)
	}
	who := identity{Name: "Octo Example", Email: "1234+octo@users.noreply.github.com"}
	base := message{Record: "rel-v0.16.0", Holder: octoKept, Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), AssistedBy: "abcd:0.0.0"}
	twelve := Withdrawal{Number: 12, ID: "PR_twelve", Form: FormQueue, Head: headB, By: octoKept, State: WithdrawnPending}
	other := twelve
	other.By = mayaB
	login := twelve
	login.By = "maya"
	for _, c := range []struct {
		name string
		ws   []Withdrawal
	}{
		{"#12 twice", []Withdrawal{twelve, other}},
		{"#12 under a login", []Withdrawal{login}},
	} {
		m := base
		m.Withdrawn = c.ws
		if _, err := writeEmptyCommit(p.dir, who, w.clock(), m, tree, ""); err == nil || strings.Contains(err.Error(), "maya") {
			t.Errorf("%s: %v; want it refused, naming no holder", c.name, err)
		}
	}
	marker := twelve
	marker.By = preIDMarker
	m := base
	m.Withdrawn = []Withdrawal{marker}
	if _, err := writeEmptyCommit(p.dir, who, w.clock(), m, tree, ""); err != nil {
		t.Fatalf("one entry for #12 under the marker: %v; want it written", err)
	}
}

// TestTheReadersVouchedEntryWinsOverAnEarlierMarkerEntry: a tagged release
// whose first commit, octo's, records #12 under the marker; maya's own
// reservation over it records her #12 under her kept id; and mal's commit
// over hers carries the marker entry again. The marker entry is recorded
// earlier than maya's, but it is never vouched for, so maya's settle keeps
// her own, re-arms it, and lists #12 once.
func TestTheReadersVouchedEntryWinsOverAnEarlierMarkerEntry(t *testing.T) {
	w := newWorld(t)
	f := quietForge(t)
	marker := Withdrawal{Number: 12, ID: "PR_twelve", Form: FormQueue, Head: headB, By: preIDMarker, State: WithdrawnPending}
	mine := marker
	mine.By = mayaB
	m := message{Record: "rel-v0.16.0", Holder: octoKept, Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: []Withdrawal{marker}, AssistedBy: "abcd:0.0.0"}
	f.pusherOf[w.pushTipAt("rel-v0.16.0", m.text(), w.clock())] = "octo"
	m.Holder, m.Copy, m.Withdrawn = mayaB, "0123456789abcdef0123456789abcdef", []Withdrawal{mine}
	w.pushAs(f, "rel-v0.16.0", "maya", m.text())
	m.Holder, m.Copy, m.Withdrawn = "h_"+strings.Repeat("7", 32), "fedcba9876543210fedcba9876543210", []Withdrawal{marker}
	w.pushAs(f, "rel-v0.16.0", "mal", m.text())
	w.tagOrigin("v0.16.0")
	maya := w.clone("5678+maya@users.noreply.github.com", "")
	keepHolderID(t, maya.home, mayaB)
	res, err := maya.reserve("iss-9", "m.go")
	if err != nil {
		t.Fatalf("maya's act: %v", err)
	}
	if s := settleOf(t, res, "rel-v0.16.0"); len(s.Settled) != 1 || s.Settled[0].By != mayaB || s.Settled[0].State != WithdrawnRearmed {
		t.Fatalf("maya's settle = %+v; want her own #12 re-armed", s)
	}
	if ws := tipEntries(t, w, "rel-v0.16.0"); len(ws) != 1 || ws[0].By != mayaB {
		t.Fatalf("tip records %+v; want #12 once, maya's", ws)
	}
}
