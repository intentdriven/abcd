package autonomy

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/occupancy"
)

// owedread_session_test.go — whose work is whose when several sessions share
// a working copy or the worktrees beside it (iss-2611020719483054,
// iss-2610311406481214, iss-2611070830056851). The boundary these tests hold:
// a session never loses sight of its own owed work, and an item whose holder
// the evidence cannot tell stays owed.

const (
	sessMe   = "sess-me"
	sessPeer = "sess-peer"
	sessWT   = "sess-wt"
)

// liveAs is a live-session reader answering sessions, counting its reads.
func liveAs(reads *int, sessions ...occupancy.Session) func() ([]occupancy.Session, error) {
	return func() ([]occupancy.Session, error) {
		*reads++
		return sessions, nil
	}
}

// writeDriver lays the run's control record, naming session as its live
// driver ("" for a driver with no session, a plain terminal's).
func (f *fx) writeDriver(runID, session string) {
	f.t.Helper()
	dir := filepath.Join(f.root, ".abcd", ".work.local", "run", runID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		f.t.Fatal(err)
	}
	raw := `{"schema_version":1,"driver":{"session":"` + session + `","pid":4242,"start_unix":1,"at":"2026-11-01T00:00:00Z"}}`
	if err := os.WriteFile(filepath.Join(dir, "control.json"), []byte(raw+"\n"), 0o600); err != nil {
		f.t.Fatal(err)
	}
}

// recordCreated records name as created in this copy by session ("" for a
// record kept before sessions were).
func (f *fx) recordCreated(name, session string) {
	f.t.Helper()
	st, err := openStore(f.home, f.root)
	if err != nil {
		f.t.Fatal(err)
	}
	key, err := claim.CopyKey(f.root)
	if err != nil {
		f.t.Fatal(err)
	}
	if err := st.RecordBranches(key, []BranchChange{{Kind: BranchCreated, Name: name, Session: session}}, f.now); err != nil {
		f.t.Fatal(err)
	}
}

// worktreeWith adds a worktree on a new branch name holding one commit of
// this checkout's, not pushed, and returns its path, symlinks resolved.
func (f *fx) worktreeWith(name string) string {
	f.t.Helper()
	wt := filepath.Join(f.t.TempDir(), strings.ReplaceAll(name, "/", "-"))
	f.git("worktree", "add", "-q", "-b", name, wt)
	if err := os.WriteFile(filepath.Join(wt, "w.txt"), []byte(name+"\n"), 0o644); err != nil {
		f.t.Fatal(err)
	}
	f.git("-C", wt, "add", "w.txt")
	f.git("-C", wt, "commit", "-q", "-m", "work on "+name)
	real, err := filepath.EvalSymlinks(wt)
	if err != nil {
		f.t.Fatal(err)
	}
	return real
}

// laneBranch is a branch holding one commit of this checkout's, not pushed,
// laid as lane-1 of run id for key.
func (f *fx) laneBranch(runID, key, branch string) {
	f.t.Helper()
	f.git("branch", branch)
	f.commitOn(branch, "lane work on "+branch, "")
	f.writeRun(runID, key, false, map[string]any{"id": "lane-1", "branch": branch, "stage": "validate"})
}

// dueReservation is a live reservation under this copy's key past its
// renewal: claimed three hours ago, lapsing in one.
func (f *fx) dueReservation(id string) {
	f.reserve(id, claim.CopyThis, f.now.Add(-3*time.Hour), f.now.Add(time.Hour))
}

// TestOwnWorkStaysOwedBesidePeer is the boundary: with a live peer session
// sharing this copy, this session's own work still blocks its stop: the
// branch it recorded creating, the lane of the run it drives, that run's
// reservation due for renewal; and so does every item whose holder cannot be
// told: a reservation no run names, a branch recorded before sessions were
// kept, and the lane of a run whose driving session has ended.
func TestOwnWorkStaysOwedBesidePeer(t *testing.T) {
	f := newFx(t)
	f.worktreeWith("feat/mine")
	f.recordCreated("feat/mine", sessMe)
	f.worktreeWith("feat/old-record")
	f.recordCreated("feat/old-record", "")
	f.laneBranch("run-0000000000000001", "itd-1", "build/mine-lane")
	f.writeDriver("run-0000000000000001", sessMe)
	f.dueReservation("itd-1")
	f.dueReservation("iss-2") // reserved by hand: no run names it
	f.laneBranch("run-0000000000000002", "itd-3", "build/ended-lane")
	f.writeDriver("run-0000000000000002", "sess-ended")
	f.reserveFresh("itd-3")
	f.laneBranch("run-0000000000000003", "itd-4", "build/terminal-lane")
	f.writeDriver("run-0000000000000003", "")
	f.reserveFresh("itd-4")

	reads := 0
	o := f.opts()
	o.Record = f.awayRecord(false)
	o.SessionID = sessMe
	o.LiveSessions = liveAs(&reads,
		occupancy.Session{ID: sessMe, Checkout: f.root},
		occupancy.Session{ID: sessPeer, Checkout: f.root})
	owed := f.readWith(o)
	for _, want := range []struct {
		kind OwedKind
		key  string
	}{
		{OwedUnpushed, "feat/mine"},
		{OwedUnpushed, "build/mine-lane"},
		{OwedRenewal, "itd-1"},
		{OwedRenewal, "iss-2"},
		{OwedUnpushed, "feat/old-record"},
		{OwedUnpushed, "build/ended-lane"},
		{OwedUnpushed, "build/terminal-lane"},
	} {
		if !has(owed, want.kind, want.key) {
			t.Errorf("%s %s must stay owed to this session: %s", want.kind, want.key, describe(owed))
		}
	}
	v := Judge(o.Record, owed, StopInput{SessionID: sessMe}, SessionState{}, f.now)
	if !v.Block {
		t.Fatalf("this session's own owed work must block its stop: row %d", v.Row)
	}
}

// TestPeerSessionWorkNamedNotOwed is the detector (iss-2611020719483054,
// iss-2610311406481214, iss-2611070830056851): in a copy two live sessions
// share, the lane of the run the peer drives, that run's reservation due for
// renewal and the branch the peer recorded creating are the peer's, and a
// branch checked out in the worktree a third live session started in is that
// session's: none is owed to this session, each holder is named once, and the
// shared copy is named as a breach of one session per checkout. The holder is
// told: read as the peer, the same items are owed; read from the third
// session's worktree, its branch is.
func TestPeerSessionWorkNamedNotOwed(t *testing.T) {
	f := newFx(t)
	f.laneBranch("run-0000000000000009", "itd-9", "build/peer-lane")
	f.writeDriver("run-0000000000000009", sessPeer)
	f.dueReservation("itd-9")
	f.worktreeWith("feat/peer-made")
	f.recordCreated("feat/peer-made", sessPeer)
	wt := f.worktreeWith("feat/in-their-worktree")
	f.recordCreated("feat/in-their-worktree", "")
	live := []occupancy.Session{
		{ID: sessMe, Checkout: f.root},
		{ID: sessPeer, Checkout: f.root},
		{ID: sessWT, Checkout: wt},
	}
	read := func(session, root string) (Owed, int) {
		reads := 0
		o := f.opts()
		o.Root = root
		o.Record = f.awayRecord(false)
		o.SessionID = session
		o.LiveSessions = liveAs(&reads, live...)
		return f.readWith(o), reads
	}

	mine, reads := read(sessMe, f.root)
	for _, it := range mine.Items {
		switch it.Branch + it.Record {
		case "build/peer-lane", "itd-9", "feat/peer-made", "feat/in-their-worktree":
			t.Errorf("another live session's item is owed to this one: %s %s\n%s", it.Kind, it.Branch+it.Record, describe(mine))
		}
	}
	if reads != 1 {
		t.Errorf("the live sessions were read %d times; once a check", reads)
	}
	notes := strings.Join(mine.Notes, "\n")
	for _, want := range []string{sessPeer, "build/peer-lane", "feat/peer-made", "itd-9", sessWT, "feat/in-their-worktree"} {
		if !strings.Contains(notes, want) {
			t.Errorf("the notes do not name %q:\n%s", want, notes)
		}
	}
	if n := strings.Count(notes, "session "+sessPeer+" "); n != 1 {
		t.Errorf("the peer is named %d times; once, with every item it holds:\n%s", n, notes)
	}
	if n := strings.Count(notes, "one session per checkout"); n != 1 {
		t.Errorf("the copy two live sessions share is named as a breach %d times; once:\n%s", n, notes)
	}
	v := Judge(f.awayRecord(false), mine, StopInput{SessionID: sessMe}, SessionState{}, f.now)
	if v.Row == 2 || v.Row == 4 || v.Row == 5 {
		t.Fatalf("this session was held for another session's work: row %d: %s", v.Row, v.Reason)
	}

	theirs, _ := read(sessPeer, f.root)
	for _, want := range []struct {
		kind OwedKind
		key  string
	}{{OwedUnpushed, "build/peer-lane"}, {OwedRenewal, "itd-9"}, {OwedUnpushed, "feat/peer-made"}} {
		if !has(theirs, want.kind, want.key) {
			t.Errorf("the holder is not told of %s %s: %s", want.kind, want.key, describe(theirs))
		}
	}
	if has(theirs, OwedUnpushed, "feat/in-their-worktree") {
		t.Errorf("the third session's worktree branch is owed to the peer: %s", describe(theirs))
	}
	inWT, _ := read(sessWT, wt)
	if !has(inWT, OwedUnpushed, "feat/in-their-worktree") {
		t.Errorf("the session working in the worktree is not told of its branch: %s", describe(inWT))
	}
}

// TestPeerWorkReturnsWhenThePeerEnds: once the peer's session has ended, what
// it held is owed again here; what the evidence named as nobody's never left.
func TestPeerWorkReturnsWhenThePeerEnds(t *testing.T) {
	f := newFx(t)
	f.laneBranch("run-0000000000000009", "itd-9", "build/peer-lane")
	f.writeDriver("run-0000000000000009", sessPeer)
	f.worktreeWith("feat/peer-made")
	f.recordCreated("feat/peer-made", sessPeer)
	f.reserveFresh("itd-9")
	reads := 0
	o := f.opts()
	o.SessionID = sessMe
	o.LiveSessions = liveAs(&reads, occupancy.Session{ID: sessMe, Checkout: f.root})
	owed := f.readWith(o)
	if !has(owed, OwedUnpushed, "build/peer-lane") || !has(owed, OwedUnpushed, "feat/peer-made") {
		t.Fatalf("an ended session's work returns to the sessions left: %s", describe(owed))
	}
}

// TestUnreadableLiveSessionsLeaveEverythingOwed: when the live sessions
// cannot be read, nothing is left out as another's, and a note says why.
func TestUnreadableLiveSessionsLeaveEverythingOwed(t *testing.T) {
	f := newFx(t)
	f.laneBranch("run-0000000000000009", "itd-9", "build/peer-lane")
	f.writeDriver("run-0000000000000009", sessPeer)
	f.reserveFresh("itd-9")
	o := f.opts()
	o.SessionID = sessMe
	o.LiveSessions = func() ([]occupancy.Session, error) { return nil, errors.New("no process table") }
	owed := f.readWith(o)
	if !has(owed, OwedUnpushed, "build/peer-lane") {
		t.Fatalf("with no live sessions read, every item stays owed: %s", describe(owed))
	}
	if !strings.Contains(strings.Join(owed.Notes, "\n"), "live sessions could not be read") {
		t.Errorf("the notes do not say the live sessions were not read: %s", describe(owed))
	}
}

// TestNoSessionIDScopesNothing: a check with no session id it can key on
// leaves every item owed as this copy's, and never reads the live sessions.
func TestNoSessionIDScopesNothing(t *testing.T) {
	f := newFx(t)
	f.laneBranch("run-0000000000000009", "itd-9", "build/peer-lane")
	f.writeDriver("run-0000000000000009", sessPeer)
	f.reserveFresh("itd-9")
	for _, id := range []string{"", "not a session id"} {
		reads := 0
		o := f.opts()
		o.SessionID = id
		o.LiveSessions = liveAs(&reads, occupancy.Session{ID: sessPeer, Checkout: f.root})
		owed := f.readWith(o)
		if !has(owed, OwedUnpushed, "build/peer-lane") {
			t.Errorf("session id %q: the lane must stay owed: %s", id, describe(owed))
		}
		if reads != 0 {
			t.Errorf("session id %q: the live sessions were read with no session to tell apart", id)
		}
	}
}

// recordAs runs the guard's created-branch recorder for one change, as
// session would.
func (f *fx) recordAs(session string, c BranchChange) {
	f.t.Helper()
	if err := RecordCreated(f.root, f.root, []BranchChange{c}, Options{Home: f.home, SessionID: session}); err != nil {
		f.t.Fatal(err)
	}
}

// bothLive is a live-session reader with this session and the peer both
// started in this copy.
func (f *fx) bothLive(reads *int) func() ([]occupancy.Session, error) {
	return liveAs(reads,
		occupancy.Session{ID: sessMe, Checkout: f.root},
		occupancy.Session{ID: sessPeer, Checkout: f.root})
}

// TestRecreatedBranchIsItsNewCreators: a name the peer created and deleted,
// which this session then creates again, is this session's: its commits not
// pushed block this session's stop, and the peer is not named as holding it.
func TestRecreatedBranchIsItsNewCreators(t *testing.T) {
	f := newFx(t)
	f.recordAs(sessPeer, BranchChange{Kind: BranchCreated, Name: "fix/x"})
	f.git("branch", "fix/x")
	f.commitOn("fix/x", "the peer's work", "")
	f.git("branch", "-D", "fix/x")
	f.recordAs(sessMe, BranchChange{Kind: BranchCreated, Name: "fix/x"})
	f.git("branch", "fix/x")
	f.commitOn("fix/x", "this session's work", "")

	reads := 0
	o := f.opts()
	o.Record = f.awayRecord(false)
	o.SessionID = sessMe
	o.LiveSessions = f.bothLive(&reads)
	owed := f.readWith(o)
	if !has(owed, OwedUnpushed, "fix/x") {
		t.Fatalf("this session's recreated branch must stay owed to it: %s", describe(owed))
	}
	if strings.Contains(strings.Join(owed.Notes, "\n"), "fix/x (it created)") {
		t.Errorf("the peer that made the name first is named as its creator: %s", describe(owed))
	}
}

// TestStackedOnPeerBranchStaysStacked: this session's branch cut from a
// branch a live peer holds is still judged stacked on it: the peer's branch
// is not owed here, but it serves as the base, so the clearing rule owes the
// preflight a receipt-minting repository needs for this session's head.
func TestStackedOnPeerBranchStaysStacked(t *testing.T) {
	f := newFx(t)
	f.git("branch", "feat/peer-base")
	f.commitOn("feat/peer-base", "the peer's base", "")
	f.push("feat/peer-base")
	f.recordCreated("feat/peer-base", sessPeer)
	f.git("branch", "feat/my-top", "feat/peer-base")
	f.commitOn("feat/my-top", "this session's top", "")
	f.push("feat/my-top")
	f.recordCreated("feat/my-top", sessMe)
	f.mintReceipts()

	reads := 0
	o := f.opts()
	o.Record = f.awayRecord(false)
	o.SessionID = sessMe
	o.LiveSessions = f.bothLive(&reads)
	owed := f.readWith(o)
	it, ok := find(owed, OwedStacked, "feat/my-top")
	if !ok || !strings.Contains(it.Reason, "feat/peer-base") || !strings.Contains(it.Reason, "preflight") {
		t.Fatalf("this session's branch stacked on the peer's is owed by the clearing rule: %s", describe(owed))
	}
	for _, i := range owed.Items {
		if i.Branch == "feat/peer-base" {
			t.Errorf("the peer's base is owed to this session: %s", describe(owed))
		}
	}
}

// TestSharedCopyBreachNamedWithNoClaim (iss-2610311406481214): in a copy two
// live sessions share, a branch reached only through the shared HEAD reflog,
// which no session's record names, stays owed, and the shared copy is still
// named as a breach of one session per checkout, once.
func TestSharedCopyBreachNamedWithNoClaim(t *testing.T) {
	f := newFx(t)
	f.git("branch", "feat/pr-123")
	f.commitOn("feat/pr-123", "work on an existing branch", "")
	f.own("feat/pr-123")

	reads := 0
	o := f.opts()
	o.Record = f.awayRecord(false)
	o.SessionID = sessMe
	o.LiveSessions = f.bothLive(&reads)
	owed := f.readWith(o)
	if !has(owed, OwedUnpushed, "feat/pr-123") {
		t.Fatalf("a branch no session's record names stays owed: %s", describe(owed))
	}
	notes := strings.Join(owed.Notes, "\n")
	if n := strings.Count(notes, "one session per checkout"); n != 1 {
		t.Errorf("the shared copy is named as a breach %d times; once:\n%s", n, notes)
	}
	if reads != 1 {
		t.Errorf("the live sessions were read %d times; once a check", reads)
	}
}
