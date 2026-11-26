package cli

import (
	"cmp"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// refusal_raisers_coordination_test.go — the probes for the codes of reserve,
// announce, peers, status, embargo and banlist (TestRaisedByIsWhatTheVerbsRaise,
// spc-2610301431108175 step 3).

var (
	_ = registerRaiserProbes("coordination", coordinationRaiserProbes)
	_ = registerRaiserProbes("embargo", embargoRaiserProbes)
	_ = registerRaiserProbes("banlist", banlistRaiserProbes)
)

// inClone is a probe's in that runs the call in a reserve world's clone, under
// the home that clone's person runs under, as reserveRun does.
func inClone(dir string) func(t *testing.T) {
	return func(t *testing.T) {
		t.Chdir(dir)
		if h := cmp.Or(cloneHomes[dir], worldHome); h != "" && harnessHome(os.Getenv("HOME")) {
			t.Setenv("HOME", h)
		}
	}
}

// coordinationRaiserProbes builds a reserve world: P holds iss-5, Q is
// another person, U's origin is gone, and a directory outside any checkout.
// It returns the probes of reserve, announce, peers and status.
func coordinationRaiserProbes(t *testing.T) []raiserProbe {
	w := newReserveWorld(t)
	p := w.clone(t, "p", octoMail)
	q := w.clone(t, "q", quinnMail)
	u := w.clone(t, "u", quinnMail)
	gitCmd(t, u, "remote", "set-url", "origin", filepath.Join(w.home, "gone.git"))
	pat := w.clone(t, "pat", "pat@example.com")
	fakeForge(t, "", 0, errors.New("gh: not logged in"))
	if code, out, errOut := reserveRun(t, p, "reserve", "iss-5", "--footprint", "a.go"); code != 0 {
		t.Fatalf("P's reservation: exit %d\n%s%s", code, out, errOut)
	}
	outside := t.TempDir()
	capped := func(t *testing.T) {
		inClone(p)(t)
		cfg := abcdhome.Path(cmp.Or(cloneHomes[p], worldHome), "config.json")
		if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cfg, []byte(`{"reservation": {"max_live": 1}}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	return []raiserProbe{
		probe("peers/outside-checkout", "peers").from(inDir(outside)),
		probe("status/outside-checkout", "status").from(inDir(outside)),
		probe("announce/outside-checkout", "announce", "merging", "iss-5").from(inDir(outside)),
		probe("reserve/outside-checkout", "reserve", "iss-5", "--footprint", "a.go").from(inDir(outside)),

		probe("usage/missing-argument", "reserve").from(inClone(p)),
		probe("usage/extra-argument", "reserve", "iss-5", "iss-6", "--footprint", "a.go").from(inClone(p)),
		probe("usage/flag-conflict", "reserve", "iss-5", "--release", "--footprint", "a.go").from(inClone(p)),
		probe("usage/invalid-flag-value", "reserve", "--sweep", "--older-than", "0").from(inClone(p)),
		probe("usage/missing-argument", "announce", "merging").from(inClone(p)),
		probe("usage/extra-argument", "status", "everyone").from(inClone(p)),

		heldProbe("reserve/already-held", "reserve", "iss-5", "--footprint", "b.go").from(inClone(q)),
		probe("reserve/not-held", "reserve", "iss-5", "--release").from(inClone(q)),
		probe("reserve/invalid-request", "reserve", "spc-5", "--footprint", "a.go").from(inClone(p)),
		probe("reserve/unreachable-origin", "reserve", "iss-6", "--footprint", "c.go").from(inClone(u)),
		probe("reserve/unknown-holder", "reserve", "--holder").from(inClone(pat)),
		probe("announce/invalid-request", "announce", "bogus", "iss-5").from(inClone(p)),
		probe("announce/not-public", "announce", "merging", "iss-77").from(inClone(p)),
		probe("announce/unreachable-origin", "announce", "merging", "iss-5").from(inClone(u)),
		// Last: the cap holds for every later act of P's.
		probe("reserve/exceeded-cap", "reserve", "iss-6", "--footprint", "c.go").from(capped),
	}
}

// embargoRaiserProbes builds the embargo surface's checkout, with no store
// first and then one an intake made, and returns embargo's probes.
func embargoRaiserProbes(t *testing.T) []raiserProbe {
	r := embargoSurfaceRepo(t)
	outside := t.TempDir()
	leak := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(leak, []byte("fixes "+surfaceGHSA+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	seeded := func(t *testing.T) {
		t.Chdir(r.Root())
		seedEmbargo(t)
	}
	return []raiserProbe{
		probe("embargo/missing-store", "embargo", "publish"),
		probe("embargo/missing-store", "embargo", "triage", surfaceGHSA, "vulnerability", "--by", "Alice"),
		probe("embargo/missing-input", "embargo", "check"),
		probe("embargo/not-away", "embargo", "queue"),
		probe("embargo/outside-checkout", "embargo", "intake").from(inDir(outside)),
		probe("embargo/embargoed-text", "embargo", "check", "--file", leak).from(seeded),
		probe("embargo/invalid-request", "embargo", "check", "--range", "not-a-range"),
		probe("embargo/invalid-request", "embargo", "triage", surfaceGHSA, "not-a-vulnerability", "--by", "Alice"),
		probe("embargo/invalid-request", "embargo", "close", surfaceGHSA),
		// Last: no later probe runs without a home.
		probe("embargo/unknown-home", "embargo", "intake").from(func(t *testing.T) {
			t.Chdir(r.Root())
			t.Setenv("HOME", "")
		}),
	}
}

// banlistRaiserProbes builds a directory holding the public config and a keyed
// private store, one holding a legacy store, and one with no public config,
// and returns banlist's probes.
func banlistRaiserProbes(t *testing.T) []raiserProbe {
	repo := blRepo(t, "widget-partner   widgetworks\n")
	legacy := blRepo(t, "")
	store := filepath.Join(legacy, filepath.FromSlash(".abcd/.work.local/private-names.txt"))
	if err := os.MkdirAll(filepath.Dir(store), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store, []byte("# notes\nwidgetworks\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bare := t.TempDir()
	t.Chdir(repo)
	return []raiserProbe{
		probe("usage/missing-required-flag", "banlist", "add", "k", "p"),
		probe("usage/flag-conflict", "banlist", "list", "--private", "--public"),
		probe("banlist/invalid-entry", "banlist", "add", "--private", "bad key", "widgetworks"),
		probe("banlist/invalid-entry", "banlist", "add", "--private", "k3", "-").from(stdinHolding("one\ntwo\n")),
		probe("banlist/duplicate-key", "banlist", "add", "--private", "widget-partner", "other"),
		probe("banlist/unknown-key", "banlist", "remove", "--private", "nosuchkey"),
		probe("banlist/unmanaged-entry", "banlist", "remove", "--public", "harness/gemini"),
		probe("banlist/legacy-store", "banlist", "add", "--private", "k2", "other").from(inDir(legacy)),
		probe("banlist/invalid-store", "banlist", "remove", "--public", "nokey").from(inDir(bare)),
	}
}

// stdinHolding is a probe's in that hands the call text on stdin: Run reads
// os.Stdin, so the probe swaps it for a file holding text until the subtest
// ends.
func stdinHolding(text string) func(t *testing.T) {
	return func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "stdin")
		if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(p)
		if err != nil {
			t.Fatal(err)
		}
		saved := os.Stdin
		os.Stdin = f
		t.Cleanup(func() { os.Stdin = saved; f.Close() })
	}
}
