package claim

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The cap on one person's live reservations is the person's own machine
// setting (iss-2610170717515674): reservation.max_live in
// ~/.abcd.noindex/config.json, a whole number from 1 to 100, ten when absent.
// No repository file is read for it, so a checkout cannot raise a person's
// ceiling.

// reserveN has p reserve iss-1 to iss-n, each with its own file.
func reserveN(t *testing.T, p *participant, n int) {
	t.Helper()
	for i := 1; i <= n; i++ {
		if _, err := p.reserve(fmt.Sprintf("iss-%d", i), fmt.Sprintf("f%d.go", i)); err != nil {
			t.Fatalf("reservation %d: %v", i, err)
		}
	}
}

// TestARaisedCapAdmitsPastTen: reservation.max_live 12 in the person's own
// config.json admits an eleventh and a twelfth, and the thirteenth is refused
// naming twelve and the setting.
func TestARaisedCapAdmitsPastTen(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", machineConfig(t, `{"reservation": {"max_live": 12}}`))
	reserveN(t, p, 12)
	_, err := p.reserve("iss-13", "f13.go")
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "you hold 12 live reservations, the most one person may hold") ||
		!strings.Contains(err.Error(), "reservation.max_live") {
		t.Fatalf("the thirteenth reservation under a cap of 12 = %v; want the cap refusal naming twelve and the setting", err)
	}
	if tip := w.remoteTip("iss-13"); tip != "" {
		t.Fatalf("a reservation over the cap reached origin: %s", tip)
	}
}

// TestALoweredCapRefusesPastIt: a cap lowered below what the person holds
// refuses the next one, saying they are past it.
func TestALoweredCapRefusesPastIt(t *testing.T) {
	w := newWorld(t)
	home := machineConfig(t, `{"reservation": {"max_live": 5}}`)
	p := w.clone("1234+octo@users.noreply.github.com", home)
	reserveN(t, p, 5)
	if _, err := p.reserve("iss-6", "f6.go"); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "you hold 5 live reservations, the most one person may hold") {
		t.Fatalf("the sixth under a cap of 5 = %v; want the cap refusal", err)
	}
	writeConfig(t, home, `{"reservation": {"max_live": 3}}`)
	if _, err := p.reserve("iss-6", "f6.go"); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "you hold 5 live reservations, past the 3 one person may hold") {
		t.Fatalf("the sixth under a cap lowered to 3 = %v; want the refusal saying past the 3", err)
	}
}

// TestAnUnusableCapIsRefused: a reservation.max_live that is not a whole
// number from 1 to 100, or a misspelt key beside it, refuses a new
// reservation naming the file and the key, never falling back to ten, and
// nothing is pushed.
func TestAnUnusableCapIsRefused(t *testing.T) {
	for _, c := range []struct{ body, want string }{
		{`{"reservation": {"max_live": 0}}`, "want a whole number from 1 to 100"},
		{`{"reservation": {"max_live": 101}}`, "want a whole number from 1 to 100"},
		{`{"reservation": {"max_live": -3}}`, "want a whole number from 1 to 100"},
		{`{"reservation": {"max_live": "12"}}`, "reservation.max_live"},
		{`{"reservation": {"max_live": 1.5}}`, "reservation.max_live"},
		{`{"reservation": {"max_live": null}}`, "reservation.max_live"},
		{`{"reservation": {"max_lives": 12}}`, "unknown key reservation.max_lives"},
	} {
		w := newWorld(t)
		p := w.clone("1234+octo@users.noreply.github.com", machineConfig(t, c.body))
		_, err := p.reserve("iss-1", "f1.go")
		if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), "config.json (machine layer)") ||
			!strings.HasSuffix(err.Error(), nothingPushed) {
			t.Errorf("%s: %v; want a refusal naming the machine file and %q", c.body, err, c.want)
		}
		if tip := w.remoteTip("iss-1"); tip != "" {
			t.Errorf("%s: a reservation reached origin past an unusable cap: %s", c.body, tip)
		}
	}
}

// TestAnUnusableCapNeverStopsARelease: the cap is read only when an act adds
// a reservation, so a bad setting never keeps a person from releasing or
// renewing what they hold.
func TestAnUnusableCapNeverStopsARelease(t *testing.T) {
	w := newWorld(t)
	home := t.TempDir()
	p := w.clone("1234+octo@users.noreply.github.com", home)
	reserveN(t, p, 2)
	writeConfig(t, home, `{"reservation": {"max_live": 0}}`)
	if res, err := p.reserve("iss-1", "f1.go"); err != nil || res.Act != ActRenewed {
		t.Fatalf("a renewal under an unusable cap = %+v, %v; want it renewed", res, err)
	}
	if _, err := p.do(Request{Record: "iss-2", Release: true}); err != nil {
		t.Fatalf("a release under an unusable cap: %v", err)
	}
}

// TestTheRepositoryCannotRaiseTheCap: a checkout's own .abcd/config.json
// setting reservation.max_live is never read for it.
func TestTheRepositoryCannotRaiseTheCap(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if err := os.WriteFile(filepath.Join(p.dir, ".abcd", "config.json"), []byte(`{"reservation": {"max_live": 50}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	p.git("add", ".abcd/config.json")
	p.git("commit", "-q", "-m", "config")
	reserveN(t, p, 10)
	if _, err := p.reserve("iss-11", "f11.go"); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "you hold 10 live reservations") {
		t.Fatalf("the eleventh with the repository setting 50 = %v; want the cap of ten", err)
	}
}

// TestAnUnreadableMachineFileKeepsTheDefault: a machine config.json that
// cannot be read at all (writable by others, not valid JSON) never raises the
// cap, and never stops a reservation below ten either; at the cap the refusal
// says why the file was read past.
func TestAnUnreadableMachineFileKeepsTheDefault(t *testing.T) {
	for _, c := range []struct {
		name, body string
		mode       os.FileMode
		cause      string
	}{
		{"group-writable", `{"reservation": {"max_live": 50}}`, 0o660, "writable by group or other"},
		{"invalid", `{"reservation": {"max_live": 50}`, 0o600, "not valid JSON"},
	} {
		w := newWorld(t)
		home := machineConfig(t, c.body)
		if err := os.Chmod(filepath.Join(abcdhome.Path(home), "config.json"), c.mode); err != nil {
			t.Fatal(err)
		}
		p := w.clone("1234+octo@users.noreply.github.com", home)
		reserveN(t, p, 10)
		_, err := p.reserve("iss-11", "f11.go")
		if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "you hold 10 live reservations, the most one person may hold") ||
			!strings.Contains(err.Error(), c.cause) || strings.Contains(err.Error(), home) {
			t.Errorf("%s: the eleventh = %v; want the cap of ten naming why the file was read past, without the home path", c.name, err)
		}
	}
}

// TestTheHolderReadsBesideTheCap: reservation.max_live beside
// reservation.github_username is no unknown key to the holder's reader.
func TestTheHolderReadsBesideTheCap(t *testing.T) {
	r := newCheckout(t, "pat@corp.com")
	r.Git("config", "user.name", "Pat")
	fake := &forgeFake{login: "octo-person", id: 4242}
	h, err := ResolveHolder(r.Root(), Options{Home: machineConfig(t, `{"reservation": {"github_username": "octo", "max_live": 20}}`), ForgeUser: fake.user})
	if err != nil || h.Login != "octo" || h.Source != HolderFromConfig {
		t.Fatalf("holder = %+v, %v; want octo from the machine configuration", h, err)
	}
}

// TestAHomeInsideAWorkingTreeCannotRaiseTheCap: a HOME that lies inside a
// git working tree, which a repository's committed settings could point a
// session at, is not read as the machine's layer for the cap, as
// layered.RootsFor refuses it for every other machine key.
func TestAHomeInsideAWorkingTreeCannotRaiseTheCap(t *testing.T) {
	w := newWorld(t)
	tree := gittest.NewRepo(t)
	home := filepath.Join(tree.Root(), "home")
	writeConfig(t, home, `{"reservation": {"max_live": 50}}`)
	p := w.clone("1234+octo@users.noreply.github.com", home)
	reserveN(t, p, 10)
	_, err := p.reserve("iss-11", "f11.go")
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "you hold 10 live reservations, the most one person may hold") ||
		!strings.Contains(err.Error(), "HOME lies inside a git working tree") {
		t.Fatalf("the eleventh with a HOME inside a working tree setting 50 = %v; want the cap of ten saying why", err)
	}
}

// TestAtTheCeilingTheRefusalOffersNoRaise: at 100 the setting cannot go
// higher, so the refusal does not offer raising it.
func TestAtTheCeilingTheRefusalOffersNoRaise(t *testing.T) {
	err := capRefusal([]string{"iss-1"}, MaxLiveCeiling, "")
	if strings.Contains(err.Error(), "raise") {
		t.Fatalf("the refusal at the ceiling = %v; want no offer to raise it", err)
	}
	if err := capRefusal([]string{"iss-1"}, 1, ""); !strings.Contains(err.Error(), "raise reservation.max_live") {
		t.Fatalf("the refusal below the ceiling = %v; want the offer to raise it", err)
	}
}

// writeConfig replaces ~/.abcd.noindex/config.json under home.
func writeConfig(t *testing.T, home, body string) {
	t.Helper()
	dir := abcdhome.Path(home)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestLiveHeldByLeavesOutAReleaseHold: the count a drain compares with the
// cap is the cap's own, so a live release hold, which the cap does not count,
// is left out of it.
func TestLiveHeldByLeavesOutAReleaseHold(t *testing.T) {
	snap := Snapshot{Reservations: []Reservation{
		{Record: "iss-1", Holder: "octo", State: StateLive},
		{Record: "rel-v0.15.0", Holder: "octo", State: StateLive},
		{Record: "iss-2", Holder: "Octo", State: StateLive},
		{Record: "iss-3", Holder: "quinn", State: StateLive},
	}}
	if got := snap.LiveHeldBy("octo", "iss-9"); len(got) != 2 || got[0] != "iss-1" || got[1] != "iss-2" {
		t.Fatalf("LiveHeldBy = %v; want iss-1 and iss-2, the release hold left out", got)
	}
}
