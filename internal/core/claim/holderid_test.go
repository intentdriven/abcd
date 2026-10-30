package claim

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The holder is named by an opaque id (iss-2610290809526033): nothing abcd
// pushes carries the person's forge login in its text or its ref names. These
// tests prove the id's shape, its mint and its refusals, what a reservation
// and an announcement push, and that the person's own holds are still told
// apart: by the ids their home declares, and by the login of an old hold.

var holderIDLine = regexp.MustCompile(`(?m)^Abcd-Holder: (h_[0-9a-f]{32})$`)

// keptHolderID is the holder id kept in p's home, failing the test when
// none is.
func (p *participant) keptHolderID() string {
	p.w.t.Helper()
	data, err := os.ReadFile(abcdhome.Path(p.home, "holder-id"))
	if err != nil {
		p.w.t.Fatalf("the holder id is not kept: %v", err)
	}
	id := strings.TrimSpace(string(data))
	if !validHolderID(id) {
		p.w.t.Fatalf("the kept holder id %q is not h_ and 32 lower-case hex characters", id)
	}
	return id
}

// self is whose holds are p's own, read as a render reads them: never
// minting an id.
func (p *participant) self() Self { return ReaderSelf(p.dir, p.opts()) }

// holderIDOf is the holder id kept in the home email's clones share.
func (w *world) holderIDOf(email string) string {
	w.t.Helper()
	return (&participant{w: w, home: w.homes[strings.ToLower(email)]}).keptHolderID()
}

// keepHolderID keeps id as home's holder id, as a mint would have, for a test
// whose bytes depend on it.
func keepHolderID(t *testing.T, home, id string) {
	t.Helper()
	dir := abcdhome.Path(home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "holder-id"), []byte(id+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// declareHolderIDs writes home's declaration of the holder ids that are also
// the person's.
func declareHolderIDs(t *testing.T, home, body string) string {
	t.Helper()
	dir := abcdhome.Path(home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "holder-ids")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestANewReservationNamesItsHolderByTheHolderIDAlone: a reservation, its
// renewal and its tombstone each carry Abcd-Holder: h_<32 hex>, the id kept in
// the person's home, and the login the resolver named appears nowhere in the
// message.
func TestANewReservationNamesItsHolderByTheHolderIDAlone(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+Octo@users.noreply.github.com", "")
	res, err := p.reserve("iss-5", "a.go")
	if err != nil {
		t.Fatal(err)
	}
	id := p.keptHolderID()
	if res.Holder != id {
		t.Fatalf("the result names %q as the holder; want the holder id %q", res.Holder, id)
	}
	w.set(t0.Add(time.Hour))
	if _, err := p.do(Request{Record: "iss-5", Renew: true}); err != nil {
		t.Fatal(err)
	}
	renewal := w.remoteTip("iss-5")
	w.set(t0.Add(2 * time.Hour))
	if _, err := p.do(Request{Record: "iss-5", Release: true}); err != nil {
		t.Fatal(err)
	}
	for _, sha := range []string{w.parentOf(renewal), renewal, w.remoteTip("iss-5")} {
		msg := w.remoteMessage(sha)
		m := holderIDLine.FindStringSubmatch(msg)
		if m == nil || m[1] != id {
			t.Fatalf("message carries no Abcd-Holder: %s line:\n%s", id, msg)
		}
		if strings.Contains(strings.ToLower(msg), "octo") {
			t.Fatalf("the message carries the login:\n%s", msg)
		}
	}
}

// TestAnAnnouncementNamesItsSenderAndBranchByTheHolderID: an announcement's
// Abcd-Sender is the holder id and it goes to abcd-status/<holder id>; no
// login-named status branch is pushed, and the message carries no login.
func TestAnAnnouncementNamesItsSenderAndBranchByTheHolderID(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+Octo@users.noreply.github.com", "")
	res, err := p.announce("merging", "#1012")
	if err != nil {
		t.Fatal(err)
	}
	id := p.keptHolderID()
	if res.Sender != id || res.Branch != "abcd-status/"+id {
		t.Fatalf("sent as %q on %q; want %q on abcd-status/%s", res.Sender, res.Branch, id, id)
	}
	tip := w.statusTip(id)
	if tip == "" || tip != res.Commit {
		t.Fatalf("abcd-status/%s on origin is %q; want %s", id, tip, res.Commit)
	}
	msg := w.remoteMessage(tip)
	if !strings.Contains(msg, "\nAbcd-Sender: "+id+"\n") || strings.Contains(strings.ToLower(msg), "octo") {
		t.Fatalf("message:\n%s\nwant Abcd-Sender: %s and no login", msg, id)
	}
	if refs := runGit(t, w.bare, "for-each-ref", "--format=%(refname)", StatusRefPrefix); refs != StatusRefPrefix+id {
		t.Fatalf("origin's status branches are %q; want only abcd-status/%s", refs, id)
	}
}

// TestTheHolderIDIsMintedOnceAndKept: the first need mints h_ and 32
// lower-case hex characters into the home, owner-only, and every later need
// reads the same id back; two homes mint two ids.
func TestTheHolderIDIsMintedOnceAndKept(t *testing.T) {
	home := t.TempDir()
	id, err := holderID(home)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^h_[0-9a-f]{32}$`).MatchString(id) {
		t.Fatalf("minted %q; want h_ and 32 lower-case hex characters", id)
	}
	fi, err := os.Stat(abcdhome.Path(home, "holder-id"))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("the kept id: %v, %v; want a file at mode 0600", fi, err)
	}
	if again, err := holderID(home); err != nil || again != id {
		t.Fatalf("read back %q, %v; want %q", again, err, id)
	}
	if other, err := holderID(t.TempDir()); err != nil || other == id {
		t.Fatalf("another home's id %q, %v; want one of its own", other, err)
	}
}

// TestARacingMintKeepsTheFirstID: mints racing on one home all come back
// with one id, and a mint that finds the file already made reads it rather
// than replacing it; only the mint that wrote the file tells it.
func TestARacingMintKeepsTheFirstID(t *testing.T) {
	home := t.TempDir()
	first, err := mintHolderID(home)
	if err != nil {
		t.Fatal(err)
	}
	if second, err := mintHolderID(home); err != nil || second != first {
		t.Fatalf("a second mint gave %q, %v; want the first's %q", second, err, first)
	}

	home = t.TempDir()
	notices := captureMintNotices(t)
	const n = 8
	ids := make([]string, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids[i], errs[i] = holderID(home)
		}()
	}
	wg.Wait()
	for i := range n {
		if errs[i] != nil || ids[i] != ids[0] || !validHolderID(ids[i]) {
			t.Fatalf("racing mints gave %v, %v; want one id", ids, errs)
		}
	}
	if got := notices(); len(got) != 1 || got[0] != MintNoticeLine(ids[0]) {
		t.Fatalf("racing mints told %q; want the one mint that wrote the file told once", got)
	}
}

// TestAHolderIDFileNoOneElseMayTrustIsRefused: a kept id another account
// could have written, one owned by another account and one that is not an
// id are each refused, naming the file, and never minted again.
func TestAHolderIDFileNoOneElseMayTrustIsRefused(t *testing.T) {
	for _, c := range []struct {
		name  string
		setup func(t *testing.T, path string)
		want  string
	}{
		{"group writable", func(t *testing.T, path string) {
			if err := os.Chmod(path, 0o620); err != nil {
				t.Fatal(err)
			}
		}, "another account can write it"},
		{"world writable", func(t *testing.T, path string) {
			if err := os.Chmod(path, 0o606); err != nil {
				t.Fatal(err)
			}
		}, "another account can write it"},
		{"foreign owner", func(t *testing.T, path string) {
			t.Cleanup(fsutil.SwapOwnerUIDForTest(func(string) (uint32, error) { return uint32(os.Getuid()) + 1, nil }))
		}, "another account owns it"},
		{"malformed", func(t *testing.T, path string) {
			if err := os.WriteFile(path, []byte("octo\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}, "does not hold a holder id"},
	} {
		t.Run(c.name, func(t *testing.T) {
			home := t.TempDir()
			if _, err := holderID(home); err != nil {
				t.Fatal(err)
			}
			path := abcdhome.Path(home, "holder-id")
			c.setup(t, path)
			before, _ := os.ReadFile(path)
			_, err := holderID(home)
			if err == nil || !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), "~/.abcd.noindex/holder-id") {
				t.Fatalf("err = %v; want a refusal naming the file and %q", err, c.want)
			}
			if after, _ := os.ReadFile(path); string(after) != string(before) {
				t.Fatalf("the refused file was rewritten: %q to %q", before, after)
			}
			if _, err := ownSelf(home, "octo", false); err == nil {
				t.Fatal("a read of the person's own ids took the refused file")
			}
		})
	}
}

// TestTheDeclaredHolderIDsAreReadAsTheirFileSays: the declaration lists one
// id per line, with # comments, under the same guards as the kept id; an
// absent one declares nothing, and a line that is not an id refuses.
func TestTheDeclaredHolderIDsAreReadAsTheirFileSays(t *testing.T) {
	home := t.TempDir()
	const a, b = "h_0123456789abcdef0123456789abcdef", "h_fedcba9876543210fedcba9876543210"
	s, err := ownSelf(home, "octo", true)
	if err != nil || len(s.Declared) != 0 || !validHolderID(s.ID) {
		t.Fatalf("with no declaration: %+v, %v; want the minted id alone", s, err)
	}
	declareHolderIDs(t, home, "# my other machines\n"+a+"\n\n  "+b+"  # the laptop\n")
	s, err = ownSelf(home, "octo", false)
	if err != nil || len(s.Declared) != 2 || s.Declared[0] != a || s.Declared[1] != b {
		t.Fatalf("declared: %+v, %v; want %s and %s", s, err, a, b)
	}
	for _, h := range []string{s.ID, a, b, "OCTO"} {
		if !s.Owns(h) {
			t.Errorf("%q is not read as the person's own", h)
		}
	}
	for _, h := range []string{"h_00000000000000000000000000000000", "quinn", strings.ToUpper(a), ""} {
		if s.Owns(h) {
			t.Errorf("%q is read as the person's own", h)
		}
	}
	p := declareHolderIDs(t, home, a+"\nocto\n")
	if _, err := ownSelf(home, "octo", false); err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("a declaration naming a login: %v; want a refusal naming line 2", err)
	}
	if err := os.WriteFile(p, []byte(a+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0o622); err != nil {
		t.Fatal(err)
	}
	if _, err := ownSelf(home, "octo", false); err == nil || !strings.Contains(err.Error(), "~/.abcd.noindex/holder-ids") {
		t.Fatalf("a declaration another account can write: %v; want a refusal naming it", err)
	}
}

// TestAnOldLoginReservationIsStillTheHoldersOwn: a reservation pushed before
// the holder id, naming the person's login, from this copy, is renewed as
// theirs, and the renewal names the holder id; another person's login is
// still another person's.
func TestAnOldLoginReservationIsStillTheHoldersOwn(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	key, err := copyKey(p.dir)
	if err != nil {
		t.Fatal(err)
	}
	old := "abcd reservation: iss-5\n\nAbcd-Reservation: iss-5\nAbcd-Holder: Octo\nAbcd-Copy: " + key + "\n" +
		"Abcd-Footprint: a.go\nAbcd-Claimed-At: " + stamp(t0) + "\nAbcd-Expires-At: " + stamp(t0.Add(lease.Default)) + "\n"
	tip := w.pushTipAt("iss-5", old, t0)
	w.set(t0.Add(time.Hour))
	res, err := p.do(Request{Record: "iss-5", Renew: true})
	if err != nil || res.Act != ActRenewed {
		t.Fatalf("renewing one's own old-format reservation: %+v, %v; want it renewed", res, err)
	}
	next := w.remoteTip("iss-5")
	if w.parentOf(next) != tip {
		t.Fatal("the renewal is not the old reservation's child")
	}
	msg := w.remoteMessage(next)
	if m := holderIDLine.FindStringSubmatch(msg); m == nil || m[1] != p.keptHolderID() || strings.Contains(strings.ToLower(msg), "octo") {
		t.Fatalf("the renewal:\n%s\nwant the holder id and no login", msg)
	}

	q := w.clone("99+quinn@users.noreply.github.com", "")
	w.pushTipAt("iss-6", strings.ReplaceAll(old, "iss-5", "iss-6"), t0.Add(time.Hour))
	if _, err := q.reserve("iss-6", "b.go"); err == nil || !strings.Contains(err.Error(), "reserved by Octo") {
		t.Fatalf("another person over an old login reservation: %v; want a back-off naming its holder", err)
	}
}

// TestADeclaredHolderIDIsTheHoldersOwn: a live reservation under an id this
// home declares is the person's own from another copy: an overlapping
// reservation by hand is made with the warning, and the record itself is
// refused as the person's other copy, not another person's. Undeclared, the
// same reservation is another person's.
func TestADeclaredHolderIDIsTheHoldersOwn(t *testing.T) {
	w := newWorld(t)
	laptop := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := laptop.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	theirs := laptop.keptHolderID()
	studio := w.clone("1234+octo@users.noreply.github.com", t.TempDir())

	if _, err := studio.do(Request{Record: "iss-6", Footprint: []string{"a.go"}, WarnOwnOverlap: true}); err == nil ||
		!strings.Contains(err.Error(), "reserved by "+theirs) {
		t.Fatalf("undeclared, another home's id is another person's: %v", err)
	}

	declareHolderIDs(t, studio.home, theirs+"\n")
	res, err := studio.do(Request{Record: "iss-6", Footprint: []string{"a.go"}, WarnOwnOverlap: true})
	if err != nil || len(res.OwnOverlap) != 1 || res.OwnOverlap[0].Record != "iss-5" {
		t.Fatalf("declared: %+v, %v; want the reservation made with the other-copy warning", res, err)
	}
	if _, err := studio.reserve("iss-5", "a.go"); err == nil || !strings.Contains(err.Error(), "you are already building iss-5 on another computer") {
		t.Fatalf("declared, the record itself: %v; want the other-copy back-off", err)
	}
}

// TestOldLoginStatusBranchesAreStillRead: an announcement on an old
// login-named status branch is read beside one on a holder-id branch, each
// under its own branch.
func TestOldLoginStatusBranchesAreStillRead(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("5+maya@users.noreply.github.com", "")
	w.pushStatus("maya", announcementMsg("asking", "#7", "maya", t0, time.Hour), "")
	sent := q.announceAs("merging", "#1012")
	p.status(nil)
	got := map[string]string{}
	for _, a := range p.read().Announcements {
		got[a.Branch] = a.Sender
	}
	id := strings.TrimPrefix(sent.Branch, "abcd-status/")
	if len(got) != 2 || got["maya"] != "maya" || !validHolderID(id) || got[id] != id {
		t.Fatalf("read %v; want maya's old branch and the holder id's branch %q", got, id)
	}
	line := ""
	for _, a := range p.read().Announcements {
		if a.Branch == id {
			line = a.Line(w.clock())
		}
	}
	if !strings.HasPrefix(line, id+" (claimed): merging #1012") {
		t.Fatalf("the holder id's line = %q; want it named as it is, no @", line)
	}
}

// TestAReaderThatCannotReadTheHolderBacksOffUntilTheHoldEnds pins the
// rollout's forward direction: a copy on a release older than the holder id
// meets a new hold. That reader is not run here; what it does is pinned in two
// halves instead. Every minted id fails validUsername, the holder rule the
// older reader applies (unchanged here), so it reads the hold as unreadable;
// and this reader, meeting a holder it cannot read in its own turn, holds the
// record until the hold ends: another copy backs off until the default lease
// from the commit's date and the take-over margin have passed, and the
// person's own renewal from the copy the hold names is refused as unreadable.
// Neither panics, and origin's tip is left as it stands.
func TestAReaderThatCannotReadTheHolderBacksOffUntilTheHoldEnds(t *testing.T) {
	for i := 0; i < 64; i++ {
		id, err := mintHolderID(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if validUsername(id) {
			t.Fatalf("the minted id %q passes the older reader's holder rule", id)
		}
	}

	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	key, err := copyKey(p.dir)
	if err != nil {
		t.Fatal(err)
	}
	// h_ and 31 hex characters: a holder this reader cannot read, as the older
	// reader cannot read h_ and 32.
	unread := "abcd reservation: iss-5\n\nAbcd-Reservation: iss-5\nAbcd-Holder: h_" + strings.Repeat("a", 31) + "\nAbcd-Copy: " + key + "\n" +
		"Abcd-Footprint: a.go\nAbcd-Claimed-At: " + stamp(t0) + "\nAbcd-Expires-At: " + stamp(t0.Add(lease.Default)) + "\n"
	tip := w.pushTipAt("iss-5", unread, t0)

	w.set(t0.Add(lease.Default + TakeoverMargin - time.Second))
	_, err = q.reserve("iss-5", "b.go")
	want := "the reservation on iss-5 is unreadable (" + reasonHolder + "); it is held until " + stamp(t0.Add(lease.Default)) +
		" and can be taken over after " + stamp(t0.Add(lease.Default+TakeoverMargin))
	if !errors.Is(err, ErrBackOff) || err.Error() != want {
		t.Fatalf("another copy within the hold = %v; want %q", err, want)
	}
	_, err = p.do(Request{Record: "iss-5", Renew: true})
	var nh *NotHeld
	if !errors.Is(err, ErrBackOff) || !errors.As(err, &nh) || nh.Why != NotHeldUnreadable || nh.Unreadable != reasonHolder {
		t.Fatalf("the person's own renewal of a hold it cannot read = %v; want it refused as unreadable", err)
	}
	if got := w.remoteTip("iss-5"); got != tip {
		t.Fatalf("a hold the reader cannot read was overwritten: %s, was %s", got, tip)
	}
}

// TestAHomeInsideAWorkingTreeKeepsNoHolderID: a HOME inside a git working tree
// that is not the account's own, as a checkout that sets HOME into its tree
// would make it, is refused as the machine layer is: an act refuses before
// anything is pushed and mints nothing there, and a reader reads no id from
// it, so a holder id the tree carries never makes another's holds the
// person's.
func TestAHomeInsideAWorkingTreeKeepsNoHolderID(t *testing.T) {
	tree := t.TempDir()
	if err := os.Mkdir(filepath.Join(tree, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(tree, "home")
	seeded := "h_" + strings.Repeat("c", 32)
	keepHolderID(t, home, seeded)
	declareHolderIDs(t, home, "h_"+strings.Repeat("d", 32)+"\n")

	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", home)
	_, err := p.reserve("iss-5", "a.go")
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "HOME lies inside the git working tree") || w.remoteTip("iss-5") != "" {
		t.Fatalf("an act under a HOME inside a working tree = %v; want it refused, nothing pushed", err)
	}
	if s := ReaderSelf(p.dir, p.opts()); s.ID != "" || len(s.Declared) != 0 || s.Owns(seeded) || s.Login != "" || s.Owns("octo") || s.Known() {
		t.Fatalf("the reader under such a HOME = %+v; want no holder id read and no login, so no hold reads as theirs", s)
	}

	bare := filepath.Join(tree, "fresh")
	if _, err := OwnSelf("octo", Options{Home: bare}); err == nil {
		t.Fatal("a holder id was minted under a HOME inside a working tree")
	}
	if _, err := os.Lstat(abcdhome.Path(bare, "holder-id")); !os.IsNotExist(err) {
		t.Fatalf("the refused mint wrote %s: %v", abcdhome.Path(bare, "holder-id"), err)
	}
}

// TestAnIDNamedStatusBranchWhoseSenderIsAnotherIDIsDoubtful: a branch named
// by one holder id whose announcement names another id as its sender is
// shown as doubtful, as a login-named branch naming another login is: the
// name of the branch is its owner, whatever its commits claim.
func TestAnIDNamedStatusBranchWhoseSenderIsAnotherIDIsDoubtful(t *testing.T) {
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	owner, other := "h_"+strings.Repeat("1", 32), "h_"+strings.Repeat("2", 32)
	w.pushStatus(owner, announcementMsg("merged", "#7", other, t0, 30*time.Minute), "")
	w.pushStatus(other, announcementMsg("asking", "#8", other, t0, 30*time.Minute), "")

	pat.status(nil)
	v := pat.read()
	byBranch := map[string]Announcement{}
	for _, a := range v.Announcements {
		byBranch[a.Branch] = a
	}
	if a, ok := byBranch[owner]; !ok || a.Doubtful != DoubtSender || a.Sender != other {
		t.Fatalf("abcd-status/%s reads %+v; want its announcement doubtful: %s", owner, a, DoubtSender)
	}
	if a, ok := byBranch[other]; !ok || a.Doubtful != "" {
		t.Fatalf("abcd-status/%s reads %+v; want its own announcement not doubtful", other, a)
	}
	if got := pat.notes(); !strings.Contains(got, other+" on abcd-status/"+owner+" (doubtful: "+DoubtSender+")") {
		t.Fatalf("the block:\n%s\nwant the id-named branch's other sender shown as doubtful", got)
	}
}

// TestOnePersonsHandBacksUnderTheirLoginAndTheirIDListAsTwoHolders: the
// listing of the last day's hand-backs knows no one's ids but by what the
// tombstones name, so one person's hand-back made before the holder id (named
// by their login) and one made after (named by their id) list as two
// holders. It is the stated cost of the change, and it lapses with the
// window: no tombstone named by a login is made any more, so a day after the
// update the person lists once.
func TestOnePersonsHandBacksUnderTheirLoginAndTheirIDListAsTwoHolders(t *testing.T) {
	now := t0.Add(time.Hour)
	id := "h_" + strings.Repeat("3", 32)
	handBack := func(record, holder string, at time.Time) Reservation {
		return Reservation{Record: record, Holder: holder, State: StateReleased, ReleasedBy: ReleasedByTombstone,
			HandedBack: HandBackKinds[0], ReleasedAt: at}
	}
	s := Snapshot{Reservations: []Reservation{
		handBack("iss-5", "octo", now.Add(-2*time.Hour)),
		handBack("iss-6", id, now.Add(-time.Hour)),
	}}
	got := s.RecentHandBacks(now)
	if len(got) != 2 || got[0].Holder != id || got[1].Holder != "octo" {
		t.Fatalf("hand-backs = %+v; want the id's and the login's listed apart", got)
	}
	later := now.Add(HandBackWindow - 90*time.Minute)
	if got := s.RecentHandBacks(later); len(got) != 1 || got[0].Holder != id {
		t.Fatalf("a day on, hand-backs = %+v; want the id's alone", got)
	}
}
