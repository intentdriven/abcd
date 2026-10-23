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
	dir := filepath.Join(p.w.t.TempDir(), name)
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
// another account pushing as the same login counts too.
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
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err := View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	c, ok := CeilingFor(p.dir, snap, p.opts())
	if !ok {
		t.Fatal("no ceiling for the reader")
	}
	if c.Holder != "octo" || c.Live != 4 || c.Limit != 5 || c.ThisCopy != 2 || c.Elsewhere != 1 ||
		len(c.Siblings) != 1 || c.Siblings[0].Name != "lane-b" || c.Siblings[0].Live != 1 {
		t.Fatalf("ceiling = %+v; want octo 4 of 5: this copy 2, lane-b 1, elsewhere 1", c)
	}
	if !c.Warn {
		t.Fatal("4 of 5 is not warned of")
	}
	msg := c.Warning()
	for _, want := range []string{"4 of", "5", "same login", "not readable here"} {
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
	if c, _ := CeilingFor(p.dir, snap, p.opts()); c.Live != 3 || c.Warn || c.Warning() != "" {
		t.Fatalf("ceiling at 3 of 5 = %+v, warning %q; want no warning", c, c.Warning())
	}
}

// TestTheCapRefusalSaysWhichCopyHoldsEachReservation (iss-2610231004140527):
// at the cap, the refusal names each held record with the copy it was made
// from, so the person can tell this machine's holds from the rest.
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
	_, err := p.reserve("iss-4", "iss-4.go")
	if err == nil {
		t.Fatal("a fourth reservation at a cap of 3 was made")
	}
	for _, want := range []string{"this copy: iss-1; lane-b, on this machine: iss-2; elsewhere: iss-3"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal %q lacks %q", err, want)
		}
	}
}
