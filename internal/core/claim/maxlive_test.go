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

// The cap on one person's live reservations (iss-2610170717515674) is
// reservation.max_live, a whole number from 1 to 100: the repository's, read
// at origin's default branch, over the bundled ten, which the person's own
// ~/.abcd.noindex/config.json may lower and never raise (itd-2611251809580662
// decision 3; maxlive_repo_test.go holds the repository's cases).

// reserveN has p reserve iss-1 to iss-n, each with its own file.
func reserveN(t *testing.T, p *participant, n int) {
	t.Helper()
	for i := 1; i <= n; i++ {
		if _, err := p.reserve(fmt.Sprintf("iss-%d", i), fmt.Sprintf("f%d.go", i)); err != nil {
			t.Fatalf("reservation %d: %v", i, err)
		}
	}
}

// TestARaisedCapAdmitsPastTen: reservation.max_live 12 in the repository's
// file on origin's default branch admits an eleventh and a twelfth, and the
// thirteenth is refused naming twelve and the setting.
func TestARaisedCapAdmitsPastTen(t *testing.T) {
	w := newWorld(t)
	w.ceilingOnMain(`{"reservation": {"max_live": 12}}`)
	p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
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

// TestALocalCommitCannotRaiseTheCap: a checkout's own .abcd/config.json
// setting reservation.max_live, committed on its local main but not on
// origin's default branch, is never read for it.
func TestALocalCommitCannotRaiseTheCap(t *testing.T) {
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

// TestAnUnreadableMachineFileRefusesAnAddingAct: a machine config.json that
// is there but cannot be read as the person's own (writable by others, not
// valid JSON, not a regular file) refuses an act that adds a reservation,
// naming the file and why, never reading past it to the repository's figure,
// and never quoting the home path; a renewal and a release read no ceiling
// and go ahead.
func TestAnUnreadableMachineFileRefusesAnAddingAct(t *testing.T) {
	for _, c := range []struct {
		name  string
		lay   func(t *testing.T, file string)
		cause string
	}{
		{"group-writable", func(t *testing.T, file string) {
			if err := os.Chmod(file, 0o660); err != nil {
				t.Fatal(err)
			}
		}, "writable by group or other"},
		{"invalid", func(t *testing.T, file string) {
			if err := os.WriteFile(file, []byte(`{"reservation": {"max_live": 2}`), 0o600); err != nil {
				t.Fatal(err)
			}
		}, "not valid JSON"},
		{"a directory", func(t *testing.T, file string) {
			if err := os.Remove(file); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(file, 0o700); err != nil {
				t.Fatal(err)
			}
		}, "not a regular file"},
	} {
		w := newWorld(t)
		home := machineConfig(t, `{"reservation": {"max_live": 2}}`)
		p := w.clone("1234+octo@users.noreply.github.com", home)
		reserveN(t, p, 1)
		c.lay(t, filepath.Join(abcdhome.Path(home), "config.json"))
		_, err := p.reserve("iss-2", "f2.go")
		if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "config.json cannot be read for "+MaxLiveKey) ||
			!strings.Contains(err.Error(), c.cause) || strings.Contains(err.Error(), home) || !strings.HasSuffix(err.Error(), nothingPushed) {
			t.Errorf("%s: a new reservation = %v; want it refused naming the file and why, without the home path", c.name, err)
		}
		if w.remoteTip("iss-2") != "" {
			t.Errorf("%s: a reservation reached origin past an unreadable machine file", c.name)
		}
		if res, err := p.reserve("iss-1", "f1.go"); err != nil || res.Act != ActRenewed {
			t.Errorf("%s: a renewal = %+v, %v; want it renewed", c.name, res, err)
		}
		if _, err := p.do(Request{Record: "iss-1", Release: true}); err != nil {
			t.Errorf("%s: a release: %v", c.name, err)
		}
	}
}

// TestNoMachineFileReadsAsNone: a machine directory with no config.json, and
// one this account cannot search, read as no machine file: the act judges by
// the repository's figure and refuses nothing for the machine's file.
func TestNoMachineFileReadsAsNone(t *testing.T) {
	w := newWorld(t)
	w.ceilingOnMain(`{"reservation": {"max_live": 4}}`)
	p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	home := machineConfig(t, `{"reservation": {"max_live": 2}}`)
	if err := os.Remove(filepath.Join(abcdhome.Path(home), "config.json")); err != nil {
		t.Fatal(err)
	}
	if lim, _, err := actMaxLive(p.dir, originDefault(p.dir), home); err != nil || lim.N != 4 || lim.Layer != LimitRepository {
		t.Fatalf("with no machine file: %+v, %v; want the repository's 4", lim, err)
	}
	writeConfig(t, home, `{"reservation": `)
	if err := os.Chmod(abcdhome.Path(home), 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(abcdhome.Path(home), 0o700) })
	if lim, _, err := actMaxLive(p.dir, originDefault(p.dir), home); err != nil || lim.N != 4 || lim.Layer != LimitRepository {
		t.Fatalf("with a machine directory that cannot be searched: %+v, %v; want the repository's 4", lim, err)
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
// layered.RootsFor refuses it for every other machine key. Since the holder
// id (iss-2610290809526033) is neither read nor kept under such a HOME, an
// act there is refused before any cap is read, so no reservation is made at
// all, and the cap the tree sets raises nothing; the ceiling a render reads
// under it is still ten, saying why.
func TestAHomeInsideAWorkingTreeCannotRaiseTheCap(t *testing.T) {
	w := newWorld(t)
	tree := gittest.NewRepo(t)
	home := filepath.Join(tree.Root(), "home")
	writeConfig(t, home, `{"reservation": {"max_live": 50}}`)
	p := w.clone("1234+octo@users.noreply.github.com", home)
	_, err := p.reserve("iss-1", "f1.go")
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "HOME lies inside the git working tree") || w.remoteTip("iss-1") != "" {
		t.Fatalf("a reservation under a HOME inside a working tree setting 50 = %v; want it refused, nothing pushed", err)
	}
	if n, note, err := maxLive("", "", home); err != nil || n.N != MaxLive || !strings.Contains(note, "HOME lies inside a git working tree") {
		t.Fatalf("the cap under such a HOME = %d, %q, %v; want ten, saying why", n.N, note, err)
	}
}

// TestTheRefusalNamesWhereTheCapIsSetAndOffersNoRaise: whichever layer set
// the cap, the refusal never tells a person to raise the machine's key: it
// names the repository's file, changed through a reviewed change, and the
// machine's file as the one that may only lower it.
func TestTheRefusalNamesWhereTheCapIsSetAndOffersNoRaise(t *testing.T) {
	for _, c := range []struct {
		lim  liveLimit
		want []string
	}{
		{liveLimit{N: 4, Layer: LimitRepository, Origin: ".abcd/config.json at origin/main"},
			[]string{"the ceiling of 4 is the repository's reservation.max_live in .abcd/config.json at origin/main", "reviewed change", "may only lower it"}},
		{bundledLimit(), []string{"the ceiling of 10 is abcd's default", "reservation.max_live in .abcd/config.json", "reviewed change", "may only lower it"}},
		{liveLimit{N: 3, Layer: LimitMachine, Origin: abcdhome.Display("config.json")},
			[]string{"the ceiling of 3 is this machine's reservation.max_live", "may only lower the repository's", ".abcd/config.json"}},
	} {
		err := capRefusal([]string{"iss-1"}, nil, c.lim, "")
		if strings.Contains(err.Error(), "raise") {
			t.Errorf("%s: %v; want no offer to raise the cap", c.lim.Layer, err)
		}
		for _, want := range c.want {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s: %v; want %q", c.lim.Layer, err, want)
			}
		}
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
	if got := snap.LiveHeldBy(Self{Login: "octo"}, "iss-9"); len(got) != 2 || got[0] != "iss-1" || got[1] != "iss-2" {
		t.Fatalf("LiveHeldBy = %v; want iss-1 and iss-2, the release hold left out", got)
	}
}
