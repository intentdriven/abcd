package lab

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// passedLab mints a lab of a repository that is not abcd's own and passes its
// preflight, so a procedure may open it.
func passedLab(t *testing.T) (Minted, string, string) {
	t.Helper()
	r, home := fixture(t)
	notAbcd(t)
	m := mint(t, r)
	if res, err := Preflight(r.Root(), m.ID); err != nil || !res.Passed {
		t.Fatalf("Preflight = %+v, %v; want a pass", res, err)
	}
	return m, home, r.Root()
}

// A procedure is a lab's own work, so it obeys the lab's gates: a halted lab
// records nothing more, and a lab whose preflight never passed is not yet a lab
// whose isolation anyone has proved.
func TestAProcedureRefusesAHaltedLab(t *testing.T) {
	r, _ := fixture(t)
	m := mint(t, r)
	// Never preflighted: refused, and the refusal says what to run.
	if _, err := OpenProcedure(r.Root(), m.ID); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "preflight") {
		t.Fatalf("OpenProcedure on a lab never preflighted = %v; want a refusal naming the preflight", err)
	}
	// Halted by a failed preflight (no work binary in abcd's own lab).
	if _, err := Preflight(r.Root(), m.ID); !errors.Is(err, ErrHalted) {
		t.Fatalf("Preflight = %v; want a halt", err)
	}
	if _, err := OpenProcedure(r.Root(), m.ID); !errors.Is(err, ErrHalted) {
		t.Fatalf("OpenProcedure on a halted lab = %v; want ErrHalted", err)
	}
	// An id that is not a lab is refused before anything is read.
	if _, err := OpenProcedure(r.Root(), "lab-../../x"); !errors.Is(err, ErrRefused) {
		t.Fatalf("OpenProcedure on a malformed id = %v; want ErrRefused", err)
	}
}

func TestAProcedureRefusesALabHaltedAfterItOpened(t *testing.T) {
	m, home, root := passedLab(t)
	p, err := OpenProcedure(root, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	write(t, filepath.Join(labDir(home, m), "state", "halt-sweep.json"), `{"gate":"sweep","finding":"F-1"}`+"\n")
	if err := p.WriteState("mx-261023012949-ab12/run.json", []byte("{}\n")); !errors.Is(err, ErrHalted) {
		t.Fatalf("WriteState on a lab halted since it opened = %v; want ErrHalted", err)
	}
}

func TestAProcedureWritesOnlyUnderStateMatrixAndNeverOverwrites(t *testing.T) {
	m, home, root := passedLab(t)
	p, err := OpenProcedure(root, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if p.Entry().ID != m.ID || p.Entry().Pin != m.Pin {
		t.Fatalf("Entry = %+v, want the minted lab's", p.Entry())
	}

	rel := "mx-261023012949-ab12/trials/t1/c1/1.json"
	if err := p.WriteState(rel, []byte("one\n")); err != nil {
		t.Fatalf("WriteState: %v", err)
	}
	onDisk := filepath.Join(labDir(home, m), "state", "matrix", filepath.FromSlash(rel))
	if got := read(t, onDisk); got != "one\n" {
		t.Fatalf("%s holds %q", onDisk, got)
	}
	if fi, err := os.Stat(onDisk); err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("mode of %s: %v %v, want 0600", onDisk, fi, err)
	}
	got, err := p.ReadState(rel)
	if err != nil || !bytes.Equal(got, []byte("one\n")) {
		t.Fatalf("ReadState = %q, %v", got, err)
	}
	if err := p.WriteState(rel, []byte("two\n")); !errors.Is(err, ErrRefused) {
		t.Fatalf("a second write of %s = %v; want a refusal, never an overwrite", rel, err)
	}
	if got := read(t, onDisk); got != "one\n" {
		t.Fatalf("the refused rewrite changed the file: %q", got)
	}

	for _, bad := range []string{"", ".", "..", "../preflight.json", "/etc/passwd", "a/../../x", "a//b", "a/./b", `a\b`, "a b", "x/"} {
		if err := p.WriteState(bad, []byte("x")); !errors.Is(err, ErrRefused) {
			t.Errorf("WriteState(%q) = %v; want ErrRefused", bad, err)
		}
		if _, err := p.ReadState(bad); !errors.Is(err, ErrRefused) {
			t.Errorf("ReadState(%q) = %v; want ErrRefused", bad, err)
		}
	}
	if _, err := os.Stat(filepath.Join(labDir(home, m), "state", "x")); !os.IsNotExist(err) {
		t.Errorf("a refused path was written outside state/matrix: %v", err)
	}
}

// The matrix pins the binary that drives a run with the very hash the
// preflight pins a lab's work binary with: one hashing path, not two.
func TestBinaryIdentityIsThePreflightsHash(t *testing.T) {
	r, home := fixture(t)
	m := mint(t, r)
	stubVintage(t, m.Pin, true)
	write(t, filepath.Join(labDir(home, m), "bin", "abcd"), "work binary v1")
	if res, err := Preflight(r.Root(), m.ID); err != nil || !res.Passed {
		t.Fatalf("Preflight = %+v, %v", res, err)
	}
	p, err := OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	sha, rev, ok := p.WorkBinary()
	if !ok || rev != m.Pin {
		t.Fatalf("WorkBinary = %q %q %v; want the pinned binary at %s", sha, rev, ok, m.Pin)
	}
	f, err := os.Open(filepath.Join(labDir(home, m), "bin", "abcd"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cur, hash, err := BinaryIdentity(f)
	if err != nil || hash != sha || cur.Revision != m.Pin || !cur.Known {
		t.Fatalf("BinaryIdentity = %+v %q %v; want the preflight's pin %q", cur, hash, err, sha)
	}
}

func TestWorkBinaryIsAbsentOutsideAbcd(t *testing.T) {
	m, _, root := passedLab(t)
	p, err := OpenProcedure(root, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if sha, rev, ok := p.WorkBinary(); ok || sha != "" || rev != "" {
		t.Fatalf("WorkBinary = %q %q %v; want none where no work binary applies", sha, rev, ok)
	}
}

func TestAProcedureRecordNotWrittenReadsAsNotExist(t *testing.T) {
	m, _, root := passedLab(t)
	p, err := OpenProcedure(root, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if _, err := p.ReadState("mx-261023012949-ab12/run.json"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReadState of an unwritten record = %v; want os.ErrNotExist", err)
	}
}

// A procedure's records that only make sense together are written together:
// each once, in order, and a write that fails part-way removes what it wrote,
// so a failed write leaves nothing and can be made again.
func TestAProcedureWritesASetOfRecordsWholeOrNotAtAll(t *testing.T) {
	m, home, root := passedLab(t)
	p, err := OpenProcedure(root, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	dir := filepath.Join(labDir(home, m), "state", "matrix", "mx-261023012949-ab12")

	if err := p.WriteState("mx-261023012949-ab12/b.json", []byte("taken\n")); err != nil {
		t.Fatal(err)
	}
	err = p.WriteStates(StateFile{Rel: "mx-261023012949-ab12/a.json", Data: []byte("a\n")},
		StateFile{Rel: "mx-261023012949-ab12/b.json", Data: []byte("b\n")})
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("WriteStates over a written record = %v; want a refusal", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.json")); !os.IsNotExist(err) {
		t.Fatalf("a set written part-way left a.json behind: %v", err)
	}
	if got := read(t, filepath.Join(dir, "b.json")); got != "taken\n" {
		t.Fatalf("the refused set changed the record already written: %q", got)
	}
	// A bad path anywhere in the set refuses the set before anything is
	// written.
	err = p.WriteStates(StateFile{Rel: "mx-261023012949-ab12/c.json", Data: []byte("c\n")}, StateFile{Rel: "../x", Data: []byte("x")})
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("WriteStates with a bad path = %v; want ErrRefused", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "c.json")); !os.IsNotExist(err) {
		t.Fatalf("a set with a bad path wrote c.json: %v", err)
	}
	if err := p.WriteStates(StateFile{Rel: "mx-261023012949-ab12/c.json", Data: []byte("c\n")},
		StateFile{Rel: "mx-261023012949-ab12/d.json", Data: []byte("d\n")}); err != nil {
		t.Fatalf("WriteStates: %v", err)
	}
	if read(t, filepath.Join(dir, "c.json")) != "c\n" || read(t, filepath.Join(dir, "d.json")) != "d\n" {
		t.Fatal("WriteStates did not write the set")
	}
}
