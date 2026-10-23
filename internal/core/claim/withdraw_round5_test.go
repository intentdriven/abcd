package claim

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/core/tools"
)

// The release hold, round 5 (iss-2610222014550692, dcn-2610230833367984): a
// change of a withdrawal's state counts only when its withdrawer's own push
// carried it, and a tombstone ends the hold only when its holder pushed it.

// TestAStateAnotherLoginChangedOnALiveReservationIsReArmedByTheRelease: a
// renewal-shaped commit another login pushes over the live reservation,
// recording octo's withdrawn #11 as kept, changes nothing: octo's --release
// re-arms #11 with the rest, names the change once, and the tombstone records
// it re-armed.
func TestAStateAnotherLoginChangedOnALiveReservationIsReArmedByTheRelease(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	for i := range ws {
		if ws[i].Number == 11 {
			ws[i].State = WithdrawnKept
		}
	}
	m.Withdrawn = ws
	m.ClaimedAt, m.ExpiresAt = w.clock(), w.clock().Add(time.Hour)
	planted := w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m.text(), w.clock())
	f.pusherOf[planted] = "mal"
	f.reset()
	w.set(t0.Add(2 * time.Hour))
	res, err := octo.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatal(err)
	}
	if pr := f.pr(11); pr.method != "SQUASH" {
		t.Fatalf("#11 = %+v; want it re-armed: a state its withdrawer's push did not carry changes nothing", pr)
	}
	if _, tws := w.tipWithdrawals("rel-v0.16.0"); stateOf(tws, 11) != WithdrawnRearmed {
		t.Fatalf("tip records #11 %q; want re-armed", stateOf(tws, 11))
	}
	if len(res.Settles) != 1 || !slices.Equal(numbersOf(res.Settles[0].Restated), []int{11}) ||
		!strings.Contains(strings.Join(res.Settles[0].Notes(), "\n"), "#11 on rel-v0.16.0, withdrawn by octo, recorded settled by a commit octo did not push") {
		t.Fatalf("settles = %+v; want #11 named as changed by a commit octo did not push", res.Settles)
	}
	if !slices.Contains(f.posted(), "#11 abcd release hold: the merge of #11 is re-armed; the hold for release v0.16.0 has ended.") {
		t.Fatalf("comments = %q; want #11's re-arm commented on", f.posted())
	}
	// The settle's own tombstone carried the change, pushed by octo: it
	// stands, and once the record has said so it is not asked again.
	f.reset()
	if res, err := octo.reserve("iss-8", "o.go"); err != nil || len(res.Settles) != 0 || len(f.mutations()) != 0 {
		t.Fatalf("next act = %+v, %v, mutations %v; want nothing more to settle", res.Settles, err, f.mutations())
	}
	f.mu.Lock()
	asks := f.pushAsks
	f.mu.Unlock()
	if _, err := octo.reserve("iss-9", "p.go"); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.pushAsks != asks {
		t.Fatalf("the record was asked %d more times; want a change it said octo pushed kept for the tip", f.pushAsks-asks)
	}
}

// TestATombstoneAnotherLoginPushedRecordingEveryWithdrawalSettledChangesNothing:
// a tombstone another login pushes recording every one of octo's
// withdrawals re-armed leaves none still to settle on the tip; octo's own
// --release still reads it, and re-arms all three.
func TestATombstoneAnotherLoginPushedRecordingEveryWithdrawalSettledChangesNothing(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	for i := range ws {
		ws[i].State = WithdrawnRearmed
	}
	tm := message{Record: m.Record, Holder: "octo", Copy: m.Copy, ClaimedAt: w.clock(), ExpiresAt: w.clock(), ReleasedAt: w.clock(), Withdrawn: ws, AssistedBy: m.AssistedBy}
	planted := w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), tm.text(), w.clock())
	f.pusherOf[planted] = "mal"
	w.set(t0.Add(time.Minute))
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if f.pr(11).method != "SQUASH" || !f.pr(12).queued || f.pr(13).method != "MERGE" {
		t.Fatalf("forge = %+v %+v %+v; want all three re-armed", f.pr(11), f.pr(12), f.pr(13))
	}
}

// TestAStateAnotherLoginsTombstoneChangedIsReArmedWhenTheReleaseEnds: a
// tombstone another login pushes over the live reservation, recording octo's
// withdrawn #11 re-armed, ends nothing: octo's next act acts on nothing and
// names it once; octo's own --release then ends the release and re-arms all
// three, #11 among them, named as changed by a commit octo did not push.
func TestAStateAnotherLoginsTombstoneChangedIsReArmedWhenTheReleaseEnds(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	for i := range ws {
		if ws[i].Number == 11 {
			ws[i].State = WithdrawnRearmed
		}
	}
	tm := message{Record: m.Record, Holder: "octo", Copy: m.Copy, ClaimedAt: w.clock(), ExpiresAt: w.clock(), ReleasedAt: w.clock(), Withdrawn: ws, AssistedBy: m.AssistedBy}
	planted := w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), tm.text(), w.clock())
	f.pusherOf[planted] = "mal"
	f.reset()
	w.set(t0.Add(time.Minute))
	s := settleAfter(t, octo)
	if len(f.mutations()) != 0 || len(f.posted()) != 3 || s.NotEnded != "octo" || s.Commit != "" {
		t.Fatalf("settle = %+v, mutations %v, comments %q; want nothing acted on and the tombstone named", s, f.mutations(), f.posted())
	}
	if !strings.Contains(strings.Join(s.Notes(), "\n"), "rel-v0.16.0's tip is a tombstone octo did not push") || !slices.Equal(numbersOf(s.Restated), []int{11}) {
		t.Fatalf("notes = %q; want the tombstone named as ending nothing, and #11 as changed by a commit octo did not push", s.Notes())
	}
	if res, err := octo.reserve("iss-9", "p.go"); err != nil || len(res.Settles) != 0 {
		t.Fatalf("next act = %+v, %v; want the tombstone named once for its tip", res.Settles, err)
	}
	res, err := octo.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatal(err)
	}
	if f.pr(11).method != "SQUASH" || !f.pr(12).queued || f.pr(13).method != "MERGE" {
		t.Fatalf("forge = %+v %+v %+v; want all three re-armed by octo's --release", f.pr(11), f.pr(12), f.pr(13))
	}
	if len(res.Settles) != 1 || len(res.Settles[0].Settled) != 3 || stateOf(res.Settles[0].Settled, 11) != WithdrawnRearmed {
		t.Fatalf("settles = %+v; want all three settled, #11 re-armed", res.Settles)
	}
}

// TestAnotherLoginsTombstoneMidCutEndsNothing: a tombstone another login
// pushes over the live reservation, every state as it was, ends nothing:
// nothing is re-armed, nothing commented on, the tombstone is named once, and
// the tag still ends the release.
func TestAnotherLoginsTombstoneMidCutEndsNothing(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	tm := message{Record: m.Record, Holder: "octo", Copy: m.Copy, ClaimedAt: w.clock(), ExpiresAt: w.clock(), ReleasedAt: w.clock(), Withdrawn: ws, AssistedBy: m.AssistedBy}
	planted := w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), tm.text(), w.clock())
	f.pusherOf[planted] = "mal"
	f.reset()
	w.set(t0.Add(time.Minute))
	before := len(f.posted())
	s := settleAfter(t, octo)
	if len(f.mutations()) != 0 || len(f.posted()) != before || s.NotEnded != "octo" || len(s.Settled) != 0 {
		t.Fatalf("settle = %+v, mutations %v; want nothing re-armed or commented on", s, f.mutations())
	}
	// The same tombstone pushed by octo would end it: the record decides.
	snap, err := View(octo.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	for _, rw := range snap.ReleaseWithdrawals(octo.dir, "octo", w.clock()) {
		if len(rw.Due) != 0 {
			t.Fatalf("peers = %q; want nothing listed as due on a tombstone octo did not push", rw.Lines())
		}
	}
	w.tagOrigin("v0.16.0")
	if _, err := octo.reserve("iss-9", "p.go"); err != nil {
		t.Fatal(err)
	}
	if got := f.mutations(); len(got) != 3 {
		t.Fatalf("mutations after the tag = %v; want all three re-armed", got)
	}
}

// TestAMismatchedNodeIDIsNotCommentedOn: a re-arm that settles as mismatch,
// the forge answering the recorded number with another node id, posts no
// comment at all, least of all on the node the recorded id now names.
func TestAMismatchedNodeIDIsNotCommentedOn(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	before := len(f.posted())
	f.mu.Lock()
	old := f.prs[11].id
	f.prs[11].id = "PR_elsewhere"
	f.prs[99] = &fakePR{id: old, head: headA}
	f.mu.Unlock()
	res, err := p.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Settles) != 1 || stateOf(res.Settles[0].Settled, 11) != WithdrawnMismatch {
		t.Fatalf("settles = %+v; want #11 settled as mismatch", res.Settles)
	}
	for _, c := range f.posted()[before:] {
		if strings.HasPrefix(c, "#99 ") || strings.Contains(c, "#11 ") {
			t.Fatalf("comments = %q; want none for the mismatched #11", f.posted()[before:])
		}
	}
	for _, x := range res.Settles[0].Failures {
		if x.Number == 11 {
			t.Fatalf("failures = %+v; want nothing reported for a comment never meant", res.Settles[0].Failures)
		}
	}
}

// TestAPartialWithdrawalIsNotCommentedOnAsWithdrawn: auto-merge off but the
// dequeue refused leaves the pull request queued; no comment says it is
// withdrawn, and the cutter's failure line names it.
func TestAPartialWithdrawalIsNotCommentedOnAsWithdrawn(t *testing.T) {
	w := newWorld(t)
	f := newFakeForge(t, map[int]*fakePR{14: {id: "PR_fourteen", head: headA, method: "SQUASH", queued: true}})
	releaseForge.comments = f
	f.refuse["dequeuePullRequest #14"] = "GraphQL: something"
	p := w.clone("1234+octo@users.noreply.github.com", "")
	res, err := p.do(Request{Record: "rel-v0.16.0"})
	if err != nil {
		t.Fatal(err)
	}
	if !f.pr(14).queued || f.pr(14).method != "" {
		t.Fatalf("#14 = %+v; want a partial withdrawal: auto-merge off, still queued", f.pr(14))
	}
	if got := f.posted(); len(got) != 0 {
		t.Fatalf("comments = %q; want none for a partial withdrawal", got)
	}
	if len(res.ForgeFailures) != 1 || res.ForgeFailures[0].Act != ForgeWithdraw || !strings.Contains(res.ForgeFailures[0].Line(), "taking it out of the merge queue failed") {
		t.Fatalf("failures = %+v; want the partial withdrawal named to the cutter", res.ForgeFailures)
	}
}

// TestARefusedWithdrawalIsNotCommentedOn: a withdrawal the forge refuses
// (kept) posts no comment saying it is withdrawn; a live renewal that then
// withdraws it posts one, since that is a real disarm.
func TestARefusedWithdrawalIsNotCommentedOn(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	f.refuse["disablePullRequestAutoMerge #11"] = "GraphQL: octo does not have permission to disable auto-merge"
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	for _, c := range f.posted() {
		if strings.HasPrefix(c, "#11 ") {
			t.Fatalf("a comment says #11 is withdrawn, but the forge refused the withdrawal: %q", c)
		}
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnKept {
		t.Fatalf("#11 recorded %q; want kept", stateOf(ws, 11))
	}
	f.mu.Lock()
	delete(f.refuse, "disablePullRequestAutoMerge #11")
	f.mu.Unlock()
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	want := "#11 abcd release hold: the merge of #11 is withdrawn by octo while release v0.16.0 is cut, and is re-armed after it."
	if got := f.posted(); len(got) != 3 || got[2] != want {
		t.Fatalf("comments = %q; want the retried withdrawal commented on once", got)
	}
}

// TestAnAdmitOfAnotherPersonsWithdrawalIsRefused: octo takes over mallory's
// lapsed release and asks to admit #11, which mallory withdrew: refused
// before anything is pushed or armed, naming whose it is and the remedy.
func TestAnAdmitOfAnotherPersonsWithdrawalIsRefused(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	m := w.clone("5678+mallory@users.noreply.github.com", "")
	if _, err := m.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	w.set(w.clock().Add(25 * time.Hour))
	o := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := o.do(Request{Record: "rel-v0.16.0", TakeOver: true}); err != nil {
		t.Fatalf("take-over: %v", err)
	}
	f.reset()
	comments, tip := len(f.posted()), w.remoteTip("rel-v0.16.0")
	_, err := o.do(Request{Record: "rel-v0.16.0", Admit: 11})
	if err == nil || !strings.Contains(err.Error(), "#11 was withdrawn by mallory, not by you, and only its withdrawer admits it") ||
		!strings.Contains(err.Error(), "mallory admits it, or takes the release over themselves (abcd reserve rel-v0.16.0 --take-over)") {
		t.Fatalf("admit = %v; want it refused, naming mallory and the remedy", err)
	}
	if len(f.mutations()) != 0 || len(f.posted()) != comments || w.remoteTip("rel-v0.16.0") != tip || f.pr(11).method != "" {
		t.Fatalf("mutations %v, comments %q, tip moved %v; want the forge and origin untouched", f.mutations(), f.posted(), w.remoteTip("rel-v0.16.0") != tip)
	}
}

// TestFirstRecordComparesEveryFieldAForgeCallReads: a commit carrying an
// entry with its head, node id or withdrawer changed is that entry's own
// first record, never the commit beneath that recorded the original.
func TestFirstRecordComparesEveryFieldAForgeCallReads(t *testing.T) {
	orig := Withdrawal{Number: 11, ID: "PR_eleven", Form: FormAutoSquash, Head: headA, By: "octo", State: WithdrawnPending}
	link := func(c string, w Withdrawal) chainLink {
		return chainLink{commit: c, readable: true, m: message{Record: "rel-v0.16.0", Holder: "octo", Withdrawn: []Withdrawal{w}}}
	}
	for name, change := range map[string]func(*Withdrawal){
		"head": func(w *Withdrawal) { w.Head = headB },
		"id":   func(w *Withdrawal) { w.ID = "PR_other" },
		"by":   func(w *Withdrawal) { w.By = "Octo2" },
	} {
		changed := orig
		change(&changed)
		links := []chainLink{link("c-new", changed), link("c-old", orig)}
		links[0].m.Holder = changed.By
		if first, ok := firstRecord(links, true, changed); !ok || first != "c-new" {
			t.Errorf("%s changed: first record = %q, %v; want the commit that changed it", name, first, ok)
		}
	}
}

// commentStub is a comment writer answering as told.
type commentStub struct {
	posted bool
	read   string
	err    error
	calls  int
}

func (c *commentStub) Comment(_, _, body string) (bool, string, error) {
	c.calls++
	read := c.read
	if read == "=" {
		read = body
	}
	return c.posted, read, c.err
}

// TestACommentIsPostedOnlyAsSentAndJudged: a body the outbound policy cannot
// judge is not posted; one posted but not read back, or read back otherwise
// by so much as a trailing newline, is reported; one read back exactly is
// not.
func TestACommentIsPostedOnlyAsSentAndJudged(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	wd := Withdrawal{Number: 11, ID: "PR_eleven", State: WithdrawnRearmed}
	body := commentBody("rel-v0.16.0", wd, "")
	post := func(c *commentStub, root string) *ForgeFailure {
		return mergeForge{comments: c}.postComment(root, wd, body)
	}
	if x := post(&commentStub{posted: true, read: "="}, p.dir); x != nil {
		t.Fatalf("a comment read back as sent = %+v; want no failure", x)
	}
	c := &commentStub{posted: true, read: body + "\n"}
	if x := post(c, p.dir); x == nil || !strings.Contains(x.Reason, "was posted, but the forge reads it back as") {
		t.Fatalf("a comment read back with a newline more = %+v; want it reported", x)
	}
	c = &commentStub{posted: true, err: errors.New("HTTP 502")}
	if x := post(c, p.dir); x == nil || x.Reason != "the comment saying it is re-armed was posted, but could not be read back from the forge to check it (HTTP 502); read it on the pull request" {
		t.Fatalf("a comment not read back = %+v; want it reported as posted and unread", x)
	}
	// A repository whose outbound policy does not read: nothing is judged,
	// so nothing is posted.
	cfg := filepath.Join(p.dir, ".abcd", "config")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "pii.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	c = &commentStub{posted: true, read: "="}
	if x := post(c, p.dir); x == nil || !strings.Contains(x.Reason, "was not posted: it could not be judged against the outbound policy") || c.calls != 0 {
		t.Fatalf("a body the policy cannot judge = %+v after %d posts; want it not posted", x, c.calls)
	}
}

// TestTheCommentWriterReadsBackOnlyACommentItsAnswerNames: an addComment
// answer naming no valid comment id is reported, and nothing is read by it.
func TestTheCommentWriterReadsBackOnlyACommentItsAnswerNames(t *testing.T) {
	for _, id := range []string{"", "IC with spaces", strings.Repeat("x", 101)} {
		var docs int
		c := ghCommenter{graphql: func(_, doc string, _ map[string]string, _ map[string]int) ([]byte, error) {
			docs++
			if strings.HasPrefix(doc, "mutation") {
				b, _ := json.Marshal(map[string]any{"data": map[string]any{"addComment": map[string]any{"commentEdge": map[string]any{"node": map[string]any{"id": id}}}}})
				return b, nil
			}
			return []byte(`{"data":{"node":{"body":"a body"}}}`), nil
		}}
		posted, read, err := c.Comment("", "PR_eleven", "a body")
		if !posted || read != "" || err == nil || docs != 1 {
			t.Errorf("comment id %q: Comment = %v, %q, %v after %d calls; want posted, nothing read back", id, posted, read, err, docs)
		}
	}
}

// TestForgeRefusedCountsNoTemporaryFailure: gh missing or out of time, and
// GitHub's own words for a failure on its side, are never a refusal, even
// where the text carries GitHub's GraphQL prefix.
func TestForgeRefusedCountsNoTemporaryFailure(t *testing.T) {
	for _, err := range []error{
		fmt.Errorf("GraphQL: busy: %w", github.ErrTimeout),
		&tools.MissingError{Cause: errors.New("GraphQL: exec: gh: not found")},
		answerError("API rate limit exceeded for user ID 1234."),
		errors.New("exit status 1 (GraphQL: Something went wrong while executing your query. Please try again.)"),
		answerError("You have exceeded a secondary rate limit"),
	} {
		if forgeRefused(err) {
			t.Errorf("forgeRefused(%q) = true; want a temporary failure not counted", err)
		}
	}
	if !forgeRefused(answerError("Pull request is in clean status")) {
		t.Error("a refusal the forge's answer carries is not counted")
	}
}

// TestStillBeingCutIsOnlyARelease: a reservation of any other subject is
// never a release being cut, live or lapsed with entries.
func TestStillBeingCutIsOnlyARelease(t *testing.T) {
	ws := []Withdrawal{{Number: 11, State: WithdrawnPending}}
	for _, st := range []string{StateLive, StateExpired} {
		if stillBeingCut(Reservation{Record: "iss-8", State: st, Withdrawn: ws}) {
			t.Errorf("iss-8 %s reads as a release being cut", st)
		}
		if !stillBeingCut(Reservation{Record: "rel-v0.16.0", State: st, Withdrawn: ws}) {
			t.Errorf("rel-v0.16.0 %s holding a withdrawal does not read as being cut", st)
		}
	}
}

// TestLocalCountsAndMarkersAreKeptOnlyForARelease: the refusal counts and
// the shown marker are written only under a release's id, and a count line
// that is not a number, a head and a count is passed over.
func TestLocalCountsAndMarkersAreKeptOnlyForARelease(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	keepRefusals(p.dir, "iss-8", map[string]int{"11 " + headA: 2})
	s := Settle{Unrecorded: []Withdrawal{{Number: 11}}}
	s.shownOnce(p.dir, "iss-8", headA)()
	for _, d := range []string{refusalsDir, shownDirRel} {
		if _, err := os.Stat(filepath.Join(p.dir, d, "iss-8")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s/iss-8 = %v; want nothing kept for a subject that is not a release", d, err)
		}
	}
	keepRefusals(p.dir, "rel-v0.16.0", map[string]int{"11 " + headA: 2})
	f := filepath.Join(p.dir, refusalsDir, "rel-v0.16.0")
	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f, append(data, []byte("abc "+headB+" 3\n12 nothead 4\n12 "+headB+" 1\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readRefusals(p.dir, "rel-v0.16.0"); fmt.Sprint(got) != fmt.Sprint(map[string]int{"11 " + headA: 2, "12 " + headB: 1}) {
		t.Fatalf("counts = %v; want only the lines that read", got)
	}
}

// TestTheRecordOfPushesIsReadOnlyAsTheForgeAnswersIt: no lookup reads as
// no record, a failed one names why, and a push this checkout cannot place
// leaves the record settling nothing, even when every push is the person's.
func TestTheRecordOfPushesIsReadOnlyAsTheForgeAnswersIt(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	tip := w.remoteTip("rel-v0.16.0")
	if _, err := Fetch(octo.dir, octo.opts()); err != nil {
		t.Fatal(err)
	}
	tree, err := emptyTree(octo.dir)
	if err != nil {
		t.Fatal(err)
	}
	links, _, err := releaseChain(octo.dir, "rel-v0.16.0", tip, tree)
	if err != nil {
		t.Fatal(err)
	}
	saved := releaseForge.pushers
	releaseForge.pushers = nil
	if r := readPushRecord(octo.dir, "rel-v0.16.0", links, tip); !strings.Contains(r.why, "no record of who pushed") {
		t.Errorf("no lookup: why = %q", r.why)
	}
	releaseForge.pushers = saved
	f.pushersErr = errors.New("HTTP 403")
	if r := readPushRecord(octo.dir, "rel-v0.16.0", links, tip); !strings.Contains(r.why, "could not be read (HTTP 403)") || r.verdict(tip, "octo") != pushNotVerified {
		t.Errorf("a failed lookup: why = %q", r.why)
	}
	f.pushersErr = nil
	unseen := strings.Repeat("ab", 20)
	f.edit = func(ps []Push) []Push {
		return slices.Concat([]Push{{Before: unseen, After: ps[0].After, Pusher: "octo"}, {Before: ps[0].Before, After: unseen, Pusher: "octo"}}, ps[1:])
	}
	if r := readPushRecord(octo.dir, "rel-v0.16.0", links, tip); !strings.Contains(r.why, "is not held here to place") || r.verdict(tip, "octo") != pushNotVerified {
		t.Errorf("a push not held here: why = %q, verdict %v; want it settling nothing", r.why, r.verdict(tip, "octo"))
	}
}

// TestAFailedReadingOfTheBranchListsNothingDue: peers' reading of the
// reader's withdrawals names every one not recorded when the checkout or the
// branch's history does not read, never due, and reads only those still
// withdrawn.
func TestAFailedReadingOfTheBranchListsNothingDue(t *testing.T) {
	w, _, octo := releasedWithNothingArmed(t)
	ws := []Withdrawal{{Number: 11, ID: "PR_eleven", Form: FormAutoSquash, Head: headA, By: "octo", State: WithdrawnPending}}
	r := Reservation{Record: "rel-v0.16.0", Commit: strings.Repeat("e", 40)}
	for name, root := range map[string]string{"no checkout": filepath.Join(t.TempDir(), "none"), "no such commit": octo.dir} {
		v, _ := vouchedForReader(root, r, "octo", ws)
		if len(v.Due) != 0 || !slices.Equal(numbersOf(v.Unrecorded), []int{11}) {
			t.Errorf("%s: %+v; want #11 named not recorded, never due", name, v)
		}
	}
	r.Commit = w.remoteTip("rel-v0.16.0")
	if _, err := Fetch(octo.dir, octo.opts()); err != nil {
		t.Fatal(err)
	}
	kept := ws[0]
	kept.State = WithdrawnKept
	if v, _ := vouchedForReader(octo.dir, r, "octo", []Withdrawal{kept}); len(v.Due)+len(v.Unrecorded)+len(v.AppPushed)+len(v.Unverified) != 0 {
		t.Errorf("a kept withdrawal read as %+v; want it passed over", v)
	}
}

// mixedTombstone pushes over octo's rel-v0.16.0 a record of #11 octo pushed,
// one adding #12 that mal pushed, and octo's tombstone over both: #11 is
// due, #12 unrecorded.
func (w *world) mixedTombstone(f *fakeForge) {
	w.t.Helper()
	id := "rel-v0.16.0"
	w11 := Withdrawal{Number: 11, ID: "PR_eleven", Form: FormAutoSquash, Head: headA, By: "octo", State: WithdrawnPending}
	w12 := Withdrawal{Number: 12, ID: "PR_twelve", Form: FormQueue, Head: headB, By: "octo", State: WithdrawnPending}
	rec := message{Record: id, Holder: "octo", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: t0, ExpiresAt: t0.Add(time.Hour), Withdrawn: []Withdrawal{w11}}
	c1 := w.pushChild(id, w.remoteTip(id), rec.text(), t0)
	rec.Withdrawn = []Withdrawal{w11, w12}
	c2 := w.pushChild(id, c1, rec.text(), t0)
	f.pusherOf[c2] = "mal"
	rec.ReleasedAt, rec.ExpiresAt = t0, t0
	w.pushChild(id, c2, rec.text(), t0)
}

// TestASettleThatActsOnNothingNamesWhatItLeavesOncePerTip: a settle stopped
// by gh acting as another account, or by a re-arm the forge did not answer,
// names the unrecorded withdrawal once for the tip, not on every act.
func TestASettleThatActsOnNothingNamesWhatItLeavesOncePerTip(t *testing.T) {
	for _, stop := range []string{"login", "unanswered"} {
		w, f, octo := releasedWithNothingArmed(t)
		w.mixedTombstone(f)
		switch stop {
		case "login":
			f.logins[octo.dir] = "quinn"
		default:
			f.refuse["enablePullRequestAutoMerge #11"] = "connection reset"
		}
		first := settleAfter(t, octo)
		if !slices.Equal(numbersOf(first.Unrecorded), []int{12}) || f.pr(11).method != "" {
			t.Fatalf("%s: first settle = %+v; want #12 named unrecorded, #11 not re-armed", stop, first)
		}
		if again := settleAfter(t, octo); len(again.Unrecorded) != 0 {
			t.Fatalf("%s: second settle = %+v; want #12 not named again for the same tip", stop, again)
		}
	}
}

// TestALiveRenewalWithdrawsAgainWhileTheRecordLagsItsOwnPush: the record of
// pushes not yet naming the renewal the act just pushed still settles the
// withdrawal beneath it: the tip it read counts as the branch's tip.
func TestALiveRenewalWithdrawsAgainWhileTheRecordLagsItsOwnPush(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.prs[11].method = "SQUASH" // re-armed by hand
	f.lag = 1
	f.mu.Unlock()
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if got := f.pr(11); got.method != "" {
		t.Fatalf("#11 = %+v; want the live renewal to withdraw it again", got)
	}
}

// TestALiveRenewalThatCannotReadItsBranchWithdrawsNothingAgain: a branch
// whose history does not read names why, and nothing is withdrawn again.
func TestALiveRenewalThatCannotReadItsBranchWithdrawsNothingAgain(t *testing.T) {
	w := newWorld(t)
	armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	p := octo.prepared(t)
	tree, err := emptyTree(octo.dir)
	if err != nil {
		t.Fatal(err)
	}
	_, ws := w.tipWithdrawals("rel-v0.16.0")
	due, s := p.vouchAgain(tree, strings.Repeat("e", 40), strings.Repeat("d", 40), ws)
	if len(due) != 0 || !strings.Contains(s.Failed, "could not read the history of rel-v0.16.0's branch") {
		t.Fatalf("due %v, settle %+v; want nothing withdrawn again and why named", due, s)
	}
}

// TestALiveRenewalNamesWhatItDoesNotWithdrawAgainOnce: an armed withdrawal
// another login recorded is named on the first live renewal that finds it
// armed, and not on the next.
func TestALiveRenewalNamesWhatItDoesNotWithdrawAgainOnce(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	m.Withdrawn = append(ws, Withdrawal{Number: 14, ID: "PR_fourteen", Form: FormAutoMerge, Head: headA, By: "octo", State: WithdrawnPending})
	m.ClaimedAt, m.ExpiresAt = w.clock(), w.clock().Add(time.Hour)
	c := w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m.text(), w.clock())
	f.pusherOf[c] = "mal"
	f.mu.Lock()
	f.prs[14].method = "MERGE"
	f.mu.Unlock()
	named := 0
	for range 2 {
		res, err := octo.do(Request{Record: "rel-v0.16.0"})
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range res.Settles {
			if slices.Contains(numbersOf(s.Unrecorded), 14) {
				named++
			}
		}
	}
	if named != 1 || f.pr(14).method != "MERGE" {
		t.Fatalf("#14 named %d times, forge %+v; want it named once and left armed", named, f.pr(14))
	}
}

// TestTheLaunchReadingNamesAKeptWithdrawal: a pull request the forge refused
// to withdraw is named kept on the launch check's reading, so the check
// points it at a renewal rather than at --admit.
func TestTheLaunchReadingNamesAKeptWithdrawal(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	f.refuse["disablePullRequestAutoMerge #11"] = "GraphQL: octo does not have permission to disable auto-merge"
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	a, err := ArmedMergesForCut(p.dir, p.opts())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(numbersOf(a.Kept), []int{11}) {
		t.Fatalf("kept = %+v; want #11", a.Kept)
	}
}

// TestAReservationOverAnotherLoginsTombstoneCarriesWhatItChanged: octo
// reserves the release again over a tombstone another login pushed, which
// recorded octo's withdrawn #11 re-armed; the new reservation carries #11
// withdrawn, and octo's --release re-arms it.
func TestAReservationOverAnotherLoginsTombstoneCarriesWhatItChanged(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	for i := range ws {
		if ws[i].Number == 11 {
			ws[i].State = WithdrawnRearmed
		}
	}
	tm := message{Record: m.Record, Holder: "octo", Copy: m.Copy, ClaimedAt: w.clock(), ExpiresAt: w.clock(), ReleasedAt: w.clock(), Withdrawn: ws, AssistedBy: m.AssistedBy}
	planted := w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), tm.text(), w.clock())
	f.pusherOf[planted] = "mal"
	w.set(t0.Add(time.Minute))
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, tws := w.tipWithdrawals("rel-v0.16.0"); stateOf(tws, 11) != WithdrawnPending {
		t.Fatalf("the new reservation records #11 %q; want it carried withdrawn", stateOf(tws, 11))
	}
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if f.pr(11).method != "SQUASH" {
		t.Fatalf("#11 = %+v; want it re-armed by the release", f.pr(11))
	}
}

// TestALiveRenewalAsksTheRecordOnlyForAMergeArmedAgain: a live renewal lists
// the armed merges first, and asks the forge's record of pushes only when
// one it records withdrawn is armed again.
func TestALiveRenewalAsksTheRecordOnlyForAMergeArmedAgain(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	asks := func() int { f.mu.Lock(); defer f.mu.Unlock(); return f.pushAsks }
	before := asks()
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if got := asks() - before; got != 0 {
		t.Fatalf("a live renewal with nothing armed again asked the record %d times; want none", got)
	}
	f.mu.Lock()
	f.prs[11].method = "SQUASH" // re-armed by hand
	f.mu.Unlock()
	before = asks()
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if got := asks() - before; got != 1 || f.pr(11).method != "" {
		t.Fatalf("asked %d times, #11 %+v; want one ask and #11 withdrawn again", got, f.pr(11))
	}
}

// TestALiveRenewalVouchesOnlyAMergeArmedAgain: a renewal vouches only the
// merges the forge lists armed again, so an entry it does not withdraw again
// is never named, however the record of pushes reads its commit.
func TestALiveRenewalVouchesOnlyAMergeArmedAgain(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	m.Withdrawn = append(ws, Withdrawal{Number: 14, ID: "PR_fourteen", Form: FormAutoMerge, Head: headA, By: "octo", State: WithdrawnPending})
	m.ClaimedAt, m.ExpiresAt = w.clock(), w.clock().Add(time.Hour)
	c := w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m.text(), w.clock())
	f.pusherOf[c] = "mal"
	f.mu.Lock()
	f.prs[11].method = "SQUASH" // re-armed by hand; #14 stays off
	f.mu.Unlock()
	res, err := octo.do(Request{Record: "rel-v0.16.0"})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range res.Settles {
		if slices.Contains(numbersOf(s.Unrecorded), 14) {
			t.Fatalf("settle %+v names #14; want only a merge armed again vouched", s)
		}
	}
	if f.pr(11).method != "" {
		t.Fatalf("#11 = %+v; want it withdrawn again", f.pr(11))
	}
}
