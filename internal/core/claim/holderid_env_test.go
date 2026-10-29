package claim

import (
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// A hosted or ephemeral session's home is not kept, so its holder id comes
// from the environment (HolderIDEnv) before the home's file, and a fresh mint
// says so once. These tests prove the read order, the refusal of a malformed
// variable, the notice, and that a mint never writes into a folder the read
// would refuse.

// captureMintNotices registers a mint notice that records each line, for the
// rest of the test.
func captureMintNotices(t *testing.T) func() []string {
	t.Helper()
	var mu sync.Mutex
	var lines []string
	RegisterMintNotice(func(l string) {
		mu.Lock()
		defer mu.Unlock()
		lines = append(lines, l)
	})
	t.Cleanup(func() { RegisterMintNotice(nil) })
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), lines...)
	}
}

// TestTheEnvironmentHolderIDWinsOverTheFile: set and valid, the variable is
// the holder id an act writes and a render reads, over a kept file, which it
// never rewrites; in a home with no file it mints nothing. Unset or empty,
// the kept file is read as before.
func TestTheEnvironmentHolderIDWinsOverTheFile(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	kept := "h_" + strings.Repeat("a", 32)
	keepHolderID(t, p.home, kept)
	env := "h_" + strings.Repeat("b", 32)

	t.Setenv(HolderIDEnv, env)
	res, err := p.reserve("iss-5", "a.go")
	if err != nil || res.Holder != env {
		t.Fatalf("reserved as %q, %v; want the variable's %q", res.Holder, err, env)
	}
	if m := holderIDLine.FindStringSubmatch(w.remoteMessage(w.remoteTip("iss-5"))); m == nil || m[1] != env {
		t.Fatalf("the pushed holder is %v; want %s", m, env)
	}
	if s := p.self(); s.ID != env || !s.Owns(env) || s.Owns(kept) {
		t.Fatalf("the reader is %+v; want the variable's id alone", s)
	}
	if got := p.keptHolderID(); got != kept {
		t.Fatalf("the kept file now holds %q; want it left at %q", got, kept)
	}

	fresh := w.clone("5678+maya@users.noreply.github.com", t.TempDir())
	if _, err := fresh.reserve("iss-6", "b.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(abcdhome.Path(fresh.home, "holder-id")); !os.IsNotExist(err) {
		t.Fatalf("an act under the variable wrote the home's holder-id: %v", err)
	}

	t.Setenv(HolderIDEnv, "")
	w.set(t0.Add(3 * time.Hour))
	res, err = p.reserve("iss-7", "c.go")
	if err != nil || res.Holder != kept {
		t.Fatalf("with the variable empty, reserved as %q, %v; want the kept %q", res.Holder, err, kept)
	}
}

// TestAMalformedEnvironmentHolderIDRefusesAndMintsNothing: a variable set to
// anything but one holder id refuses the act, naming the variable and never
// its value, pushes nothing and mints nothing; a kept file does not stand in,
// and a render reads no id.
func TestAMalformedEnvironmentHolderIDRefusesAndMintsNothing(t *testing.T) {
	notices := captureMintNotices(t)
	for _, bad := range []string{"octo", "h_" + strings.Repeat("A", 32), "h_" + strings.Repeat("a", 31), " h_" + strings.Repeat("a", 32)} {
		t.Run(bad, func(t *testing.T) {
			t.Setenv(HolderIDEnv, bad)
			w := newWorld(t)
			p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
			_, err := p.reserve("iss-5", "a.go")
			if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), HolderIDEnv+" is set but is not a holder id") || strings.Contains(err.Error(), bad) {
				t.Fatalf("err = %v; want a refusal naming %s and not its value", err, HolderIDEnv)
			}
			if w.remoteTip("iss-5") != "" {
				t.Fatal("a refused act pushed a reservation")
			}
			if _, err := os.Lstat(abcdhome.Path(p.home, "holder-id")); !os.IsNotExist(err) {
				t.Fatalf("a refused act minted an id: %v", err)
			}
			if _, err := p.announce("merging", "#1012"); !errors.Is(err, ErrRefused) {
				t.Fatalf("an announcement under the malformed variable = %v; want it refused", err)
			}

			kept := "h_" + strings.Repeat("c", 32)
			keepHolderID(t, p.home, kept)
			if _, err := p.reserve("iss-5", "a.go"); !errors.Is(err, ErrRefused) {
				t.Fatalf("with a kept file, err = %v; want it refused all the same", err)
			}
			if s := p.self(); s.ID != "" || s.Owns(kept) {
				t.Fatalf("the reader under a malformed variable is %+v; want no id", s)
			}
		})
	}
	if got := notices(); len(got) != 0 {
		t.Fatalf("notices %q; want none", got)
	}
}

// TestAFreshMintIsToldOnce: the act that mints a holder id tells it once,
// naming the id, the file, what becomes of holds made under an earlier id,
// and how to keep one id; the next act, which reads the id kept, tells
// nothing, and neither does an act under the variable.
func TestAFreshMintIsToldOnce(t *testing.T) {
	notices := captureMintNotices(t)
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	id := p.keptHolderID()
	got := notices()
	if len(got) != 1 {
		t.Fatalf("notices %q; want one", got)
	}
	for _, want := range []string{id, "~/.abcd.noindex/holder-id", "read as someone else's until they lapse", HolderIDEnv, "~/.abcd.noindex/holder-ids"} {
		if !strings.Contains(got[0], want) {
			t.Fatalf("notice %q does not name %q", got[0], want)
		}
	}
	if _, err := p.reserve("iss-6", "b.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.announce("merging", "#1012"); err != nil {
		t.Fatal(err)
	}
	t.Setenv(HolderIDEnv, "h_"+strings.Repeat("e", 32))
	q := w.clone("5678+maya@users.noreply.github.com", t.TempDir())
	if _, err := q.reserve("iss-7", "c.go"); err != nil {
		t.Fatal(err)
	}
	if got := notices(); len(got) != 1 {
		t.Fatalf("notices after the kept id was read and under the variable %q; want the first alone", got)
	}
}

// TestAMintIntoAFolderOthersCanWriteRefuses: the mint judges each folder it
// makes or finds as the read judges it, so an ~/.abcd.noindex every account
// can write is refused, naming it and the remedy, before anything is written
// or pushed; and a read of the same home takes no id from it.
func TestAMintIntoAFolderOthersCanWriteRefuses(t *testing.T) {
	notices := captureMintNotices(t)
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	dir := abcdhome.Path(p.home)
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	_, err := p.reserve("iss-5", "a.go")
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "~/.abcd.noindex can be written by every account") ||
		!strings.Contains(err.Error(), "no holder id is minted there") {
		t.Fatalf("err = %v; want the exposed folder refused", err)
	}
	if _, err := os.Lstat(abcdhome.Path(p.home, "holder-id")); !os.IsNotExist(err) {
		t.Fatalf("the refused mint wrote the id: %v", err)
	}
	if w.remoteTip("iss-5") != "" || len(notices()) != 0 {
		t.Fatal("a refused mint pushed a reservation or told a mint")
	}
}

// TestAnEmptyKeptIDIsRetriedNotRemoved: a kept file still empty after the
// wait is most likely another run's mint between its create and its write,
// and that run may already have pushed under its id: the refusal says to
// retry, and does not say to remove the file.
func TestAnEmptyKeptIDIsRetriedNotRemoved(t *testing.T) {
	home := t.TempDir()
	keepHolderID(t, home, "")
	if err := os.WriteFile(abcdhome.Path(home, "holder-id"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := holderID(home)
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "~/.abcd.noindex/holder-id is empty") || !strings.Contains(err.Error(), "retry") ||
		strings.Contains(err.Error(), "remove it") {
		t.Fatalf("err = %v; want a retry, and no removal advised", err)
	}
	if data, _ := os.ReadFile(abcdhome.Path(home, "holder-id")); len(data) != 0 {
		t.Fatalf("the empty file was written: %q", data)
	}
}
