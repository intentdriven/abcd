package claim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// worktree adds a worktree of p's clone named name, with its own local tier
// and p's home: a second session of the same person on the same machine.
func (p *participant) worktree(name string) *participant {
	p.w.t.Helper()
	return p.worktreeAt(filepath.Join(p.w.t.TempDir(), name))
}

// worktreeAt is worktree at the path dir.
func (p *participant) worktreeAt(dir string) *participant {
	p.w.t.Helper()
	p.git("worktree", "add", "-q", "--detach", dir)
	if err := os.MkdirAll(filepath.Join(dir, ".abcd", ".work.local"), 0o755); err != nil {
		p.w.t.Fatal(err)
	}
	return &participant{w: p.w, dir: dir, home: p.home}
}

// TestTheViewNamesTheSiblingWorktreeAReservationWasMadeFrom
// (iss-2610231004140527): a reservation made from another worktree of this
// clone reads as that worktree, by its name and never its key; one made from
// another clone, though by the same person, reads as made elsewhere.
// NameSiblings is handed the reader's holder id and their login as the
// resolver would yield it, in another case.
func TestTheViewNamesTheSiblingWorktreeAReservationWasMadeFrom(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	sib := p.worktree("lane-b")
	far := w.clone("1234+octo@users.noreply.github.com", "")
	for _, c := range []struct {
		who *participant
		id  string
	}{{p, "iss-1"}, {sib, "iss-2"}, {far, "iss-3"}} {
		if _, err := c.who.reserve(c.id, c.id+".go"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err := View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	// The view alone lists no worktree: only NameSiblings does.
	for _, r := range snap.Reservations {
		if r.Sibling != "" {
			t.Fatalf("the plain view named %s's sibling %q", r.Record, r.Sibling)
		}
	}
	snap.NameSiblings(p.dir, Self{ID: p.keptHolderID(), Login: "OCTO"})
	got := map[string]Reservation{}
	for _, r := range snap.Reservations {
		got[r.Record] = r
	}
	if r := got["iss-1"]; r.Copy != CopyThis || r.Sibling != "" {
		t.Errorf("iss-1 = copy %q sibling %q; want this copy", r.Copy, r.Sibling)
	}
	if r := got["iss-2"]; r.Copy != "" || r.Sibling != "lane-b" {
		t.Errorf("iss-2 = copy %q sibling %q; want the sibling worktree lane-b", r.Copy, r.Sibling)
	}
	if r := got["iss-3"]; r.Copy != "" || r.Sibling != "" {
		t.Errorf("iss-3 = copy %q sibling %q; want neither: it was made elsewhere", r.Copy, r.Sibling)
	}
	key := OwnCopyKey(sib.dir)
	for _, r := range snap.Reservations {
		if key == "" || strings.Contains(r.Sibling, key) {
			t.Fatalf("the sibling's key %q is shown, or it has none", key)
		}
	}
}

// TestTheCeilingCountsAPersonsHoldsByCopyAndWarnsNearTheLimit
// (iss-2610231004140527): a person's live holds are counted by copy (this
// one, each sibling worktree here, and elsewhere) against this machine's
// reservation.max_live, with a warning from four fifths of it that says
// every account whose holds are the person's counts too: here another
// computer's, whose holder id this home declares.
func TestTheCeilingCountsAPersonsHoldsByCopyAndWarnsNearTheLimit(t *testing.T) {
	w := newWorld(t)
	home := t.TempDir()
	writeConfig(t, home, `{"reservation": {"max_live": 5}}`)
	p := w.clone("1234+octo@users.noreply.github.com", home)
	sib := p.worktree("lane-b")
	far := w.clone("1234+octo@users.noreply.github.com", "")
	other := w.clone("99+quinn@users.noreply.github.com", "")
	for _, c := range []struct {
		who *participant
		id  string
	}{{p, "iss-1"}, {p, "iss-2"}, {sib, "iss-3"}, {far, "iss-4"}, {other, "iss-9"}} {
		if _, err := c.who.reserve(c.id, c.id+".go"); err != nil {
			t.Fatal(err)
		}
	}
	declareHolderIDs(t, home, far.keptHolderID()+"\n")
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err := View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	c, ok := CeilingFor(p.dir, &snap, p.opts())
	if !ok {
		t.Fatal("no ceiling for the reader")
	}
	if c.Holder != p.keptHolderID() || c.Live != 4 || c.Limit != 5 || c.ThisCopy != 2 || c.Elsewhere != 1 ||
		len(c.Siblings) != 1 || c.Siblings[0].Name != "lane-b" || c.Siblings[0].Live != 1 {
		t.Fatalf("ceiling = %+v; want the holder id 4 of 5: this copy 2, lane-b 1, elsewhere 1", c)
	}
	if !c.Warn {
		t.Fatal("4 of 5 is not warned of")
	}
	msg := c.Warning()
	for _, want := range []string{"4 of", "5", "account whose holds are yours", "not readable here"} {
		if !strings.Contains(msg, want) {
			t.Errorf("warning %q lacks %q", msg, want)
		}
	}

	// Below four fifths there is no warning, and the counts still show.
	if _, err := p.do(Request{Record: "iss-2", Release: true}); err != nil {
		t.Fatal(err)
	}
	snap, err = View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if c, _ := CeilingFor(p.dir, &snap, p.opts()); c.Live != 3 || c.Warn || c.Warning() != "" {
		t.Fatalf("ceiling at 3 of 5 = %+v, warning %q; want no warning", c, c.Warning())
	}
}

// TestTheCapRefusalSaysWhichCopyHoldsEachReservation (iss-2610231004140527):
// at the cap, the refusal names each held record with the copy it was made
// from, so the person can tell this machine's holds from the rest; the one
// made elsewhere is another computer's, whose holder id this home declares.
func TestTheCapRefusalSaysWhichCopyHoldsEachReservation(t *testing.T) {
	w := newWorld(t)
	home := t.TempDir()
	writeConfig(t, home, `{"reservation": {"max_live": 3}}`)
	p := w.clone("1234+octo@users.noreply.github.com", home)
	sib := p.worktree("lane-b")
	far := w.clone("1234+octo@users.noreply.github.com", "")
	for _, c := range []struct {
		who *participant
		id  string
	}{{p, "iss-1"}, {sib, "iss-2"}, {far, "iss-3"}} {
		if _, err := c.who.reserve(c.id, c.id+".go"); err != nil {
			t.Fatal(err)
		}
	}
	declareHolderIDs(t, home, far.keptHolderID()+"\n")
	_, err := p.reserve("iss-4", "iss-4.go")
	if err == nil {
		t.Fatal("a fourth reservation at a cap of 3 was made")
	}
	for _, want := range []string{`this copy: iss-1; "lane-b", on this machine: iss-2; elsewhere: iss-3`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal %q lacks %q", err, want)
		}
	}
}

// TestNameSiblingsNamesOnlyTheHoldersOwnAndTellsClashingNamesApart
// (iss-2610231004140527 review): a tip of another holder carrying a sibling's
// key is not named, since the key is no secret from whoever pushed it; two
// worktrees with one directory name read as parent/name; two holding one key
// (a copied worktree) read as both; and worktrees that cannot be listed are
// said to be, every tip then reading as made elsewhere.
func TestNameSiblingsNamesOnlyTheHoldersOwnAndTellsClashingNamesApart(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	a := p.worktreeAt(filepath.Join(t.TempDir(), "a", "lane-b"))
	b := p.worktreeAt(filepath.Join(t.TempDir(), "b", "lane-b"))
	c := p.worktreeAt(filepath.Join(t.TempDir(), "lane-c"))
	d := p.worktreeAt(filepath.Join(t.TempDir(), "lane-d"))
	keys := map[string]string{}
	for name, q := range map[string]*participant{"a": a, "b": b, "c": c} {
		k, err := CopyKey(q.dir)
		if err != nil {
			t.Fatal(err)
		}
		keys[name] = k
	}
	// d is a copy of c: the same kept key.
	if err := os.MkdirAll(filepath.Join(d.dir, ".abcd", ".work.local", "reservation"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d.dir, ".abcd", ".work.local", "reservation", "copy-key"), []byte(keys["c"]+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	snap := Snapshot{Reservations: []Reservation{
		{Record: "iss-1", Holder: "octo", State: StateLive, key: keys["a"]},
		{Record: "iss-2", Holder: "octo", State: StateLive, key: keys["b"]},
		{Record: "iss-3", Holder: "octo", State: StateLive, key: keys["c"]},
		{Record: "iss-4", Holder: "mallory", State: StateLive, key: keys["a"]},
	}}
	snap.NameSiblings(p.dir, Self{Login: "octo"})
	want := map[string]string{"iss-1": "a/lane-b", "iss-2": "b/lane-b", "iss-4": ""}
	for _, r := range snap.Reservations {
		if r.Record == "iss-3" {
			if r.Sibling != "lane-c or lane-d" && r.Sibling != "lane-d or lane-c" {
				t.Errorf("iss-3 sibling = %q; want both names of the copied worktree", r.Sibling)
			}
			continue
		}
		if r.Sibling != want[r.Record] {
			t.Errorf("%s sibling = %q; want %q", r.Record, r.Sibling, want[r.Record])
		}
	}
	if snap.SiblingsUnread != "" {
		t.Errorf("SiblingsUnread = %q on a listing that worked", snap.SiblingsUnread)
	}

	gone := Snapshot{Reservations: []Reservation{{Record: "iss-1", Holder: "octo", State: StateLive, key: keys["a"]}}}
	gone.NameSiblings(t.TempDir(), Self{Login: "octo"})
	if gone.Reservations[0].Sibling != "" || !strings.Contains(gone.SiblingsUnread, "could not be listed") {
		t.Fatalf("outside a repository: sibling %q, unread %q; want none and the reason", gone.Reservations[0].Sibling, gone.SiblingsUnread)
	}
}

// TestARefusedCapStillNamesTheSiblings (iss-2610231004140527 review round 2):
// a reservation.max_live an act would refuse gives no ceiling, but the
// reader's holds from a sibling worktree are still named.
func TestARefusedCapStillNamesTheSiblings(t *testing.T) {
	w := newWorld(t)
	home := t.TempDir()
	p := w.clone("1234+octo@users.noreply.github.com", home)
	sib := p.worktree("lane-b")
	if _, err := sib.reserve("iss-1", "a.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	// The setting goes bad after the hold is made, which it would refuse.
	writeConfig(t, home, `{"reservation": {"max_live": 150}}`)
	snap, err := View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := CeilingFor(p.dir, &snap, p.opts()); ok {
		t.Fatal("a refused max_live gave a ceiling")
	}
	if len(snap.Reservations) != 1 || snap.Reservations[0].Sibling != "lane-b" {
		t.Fatalf("reservations = %+v; want iss-1 named from lane-b", snap.Reservations)
	}
}

// TestASiblingIsNamedFromOneReadingOfTheWorktrees (iss-2610272352153282): a
// hold's path names a copied worktree as both its folders, as abcd peers'
// listing names it, not whichever one a walk of the worktrees found first.
func TestASiblingIsNamedFromOneReadingOfTheWorktrees(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	c := p.worktree("lane-c")
	d := p.worktree("lane-d")
	key, err := CopyKey(c.dir)
	if err != nil {
		t.Fatal(err)
	}
	// d is a copy of c: the same kept key.
	if err := os.MkdirAll(filepath.Join(d.dir, ".abcd", ".work.local", "reservation"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d.dir, ".abcd", ".work.local", "reservation", "copy-key"), []byte(key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := siblingCopy(p.dir, key)
	if !strings.Contains(got, "lane-c") || !strings.Contains(got, "lane-d") || !strings.Contains(got, " or ") {
		t.Fatalf("siblingCopy = %q; want both paths of the copied worktree", got)
	}
}

// TestSiblingsSharingAFolderAndItsParentAreToldApart (iss-2610272352168723):
// two worktrees whose folders and their parents share names read by the
// shortest path tail that tells them apart, and the ceiling counts each.
func TestSiblingsSharingAFolderAndItsParentAreToldApart(t *testing.T) {
	w := newWorld(t)
	home := t.TempDir()
	writeConfig(t, home, `{"reservation": {"max_live": 10}}`)
	p := w.clone("1234+octo@users.noreply.github.com", home)
	base := t.TempDir()
	x := p.worktreeAt(filepath.Join(base, "x", "a", "lane-b"))
	y := p.worktreeAt(filepath.Join(base, "y", "a", "lane-b"))
	if _, err := x.reserve("iss-1", "a.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := y.reserve("iss-2", "b.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err := View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	c, ok := CeilingFor(p.dir, &snap, p.opts())
	if !ok {
		t.Fatal("no ceiling for the reader")
	}
	want := []SiblingLive{{Name: "x/a/lane-b", Live: 1}, {Name: "y/a/lane-b", Live: 1}}
	if len(c.Siblings) != len(want) || c.Siblings[0] != want[0] || c.Siblings[1] != want[1] {
		t.Fatalf("siblings = %+v; want %+v", c.Siblings, want)
	}
}

// TestTheCapRefusalNamesOnlyTheHoldersOwnCopiesAndWhyNoSiblingIs
// (iss-2610272352179242): the refusal's places come from the holder's own
// live tips, never another holder's tip on the same record, and say why the
// worktrees could not be listed when they could not.
func TestTheCapRefusalNamesOnlyTheHoldersOwnCopiesAndWhyNoSiblingIs(t *testing.T) {
	key := strings.Repeat("ab", copyKeyLen/2)
	snap := Snapshot{Reservations: []Reservation{
		{Record: "iss-1", Holder: "octo", State: StateLive, key: key},
		{Record: "iss-1", Holder: "mallory", State: StateLive, Copy: CopyThis},
		{Record: "iss-1", Holder: "octo", State: StateExpired, Copy: CopyThis},
	}}
	err := capRefusalIn(&snap, t.TempDir(), Self{Login: "octo"}, []string{"iss-1"}, 1, "")
	for _, want := range []string{"elsewhere: iss-1", "could not be listed"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal %q lacks %q", err, want)
		}
	}
}

// TestASiblingsNameNeverShowsTheHomeFolders (iss-2610272352168723 review):
// tails are cut from home-redacted paths, so telling two siblings apart never
// reaches up into the home directory's own folders, the login name among them.
func TestASiblingsNameNeverShowsTheHomeFolders(t *testing.T) {
	home := filepath.Join(t.TempDir(), "octo")
	t.Setenv("HOME", home)
	paths := []string{filepath.Join(home, "a", "lane-b"), filepath.Join(home, "w", "a", "lane-b")}
	if got := siblingTail(paths[0], paths); got != "~/a/lane-b" {
		t.Errorf("siblingTail(%s) = %q; want ~/a/lane-b", paths[0], got)
	}
	if got := siblingTail(paths[1], paths); got != "w/a/lane-b" {
		t.Errorf("siblingTail(%s) = %q; want w/a/lane-b", paths[1], got)
	}
}

// TestASiblingsNameAndPathAreTerminalSafe (iss-2610272352153282 review): the
// one reading of the worktrees masks control characters in a sibling's name
// and path, so neither reaches a terminal raw from any surface.
func TestASiblingsNameAndPathAreTerminalSafe(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	sib := p.worktree("lane\x1b[31mb")
	key, err := CopyKey(sib.dir)
	if err != nil {
		t.Fatal(err)
	}
	sibs, unread := siblingCopies(p.dir, ownCopyKey(p.dir))
	at, ok := sibs[key]
	if unread != "" || !ok {
		t.Fatalf("siblings = %+v, unread %q; want the worktree listed", sibs, unread)
	}
	if strings.ContainsRune(at.Name, 0x1b) || strings.ContainsRune(at.Path, 0x1b) {
		t.Fatalf("sibling = %+v; want no escape character", at)
	}
}
