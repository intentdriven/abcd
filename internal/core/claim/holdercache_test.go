package claim

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The holder the forge names is asked for once per process for each checkout
// and forge credential (iss-2610161015433495): a drain move makes several
// reservation acts, and a person whose git email is not a GitHub no-reply
// address had each of them call the user endpoint again. These tests run the
// real call against a gh stub on PATH, so the cache under test is the one a
// reservation act reaches.

// forgeStub puts a gh first on PATH that records each call and answers the
// user endpoint with answer, or fails while the returned fail file exists.
// calls counts the calls so far.
type forgeStub struct {
	t    *testing.T
	log  string
	fail string
}

func stubForge(t *testing.T, answer string) *forgeStub {
	t.Helper()
	return stubForgeDuring(t, answer, "")
}

// stubForgeDuring is stubForge whose gh also runs the shell in during before
// it answers, as a gh racing another process's `gh auth switch` would.
func stubForgeDuring(t *testing.T, answer, during string) *forgeStub {
	t.Helper()
	dir := t.TempDir()
	s := &forgeStub{t: t, log: filepath.Join(dir, "calls"), fail: filepath.Join(dir, "fail")}
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '" + s.log + "'\n" +
		"if [ -e '" + s.fail + "' ]; then echo 'HTTP 401' >&2; exit 1; fi\n" +
		during + "\n" +
		"printf '%s' '" + answer + "'\n"
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	resetHolderCache(t)
	return s
}

func (s *forgeStub) calls() int {
	s.t.Helper()
	raw, err := os.ReadFile(s.log)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		s.t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line != "api --hostname github.com user" {
			s.t.Fatalf("gh was asked %q; want only the user endpoint", line)
		}
	}
	return strings.Count(string(raw), "\n")
}

func (s *forgeStub) failing(on bool) {
	s.t.Helper()
	if !on {
		if err := os.Remove(s.fail); err != nil {
			s.t.Fatal(err)
		}
		return
	}
	if err := os.WriteFile(s.fail, nil, 0o644); err != nil {
		s.t.Fatal(err)
	}
}

// resetHolderCache empties the process's holder cache for the test and again
// when it ends, so no test reads another's answer.
func resetHolderCache(t *testing.T) {
	t.Helper()
	holderCache.reset()
	t.Cleanup(holderCache.reset)
}

func TestThreeReservationActsAskTheForgeForTheHolderOnce(t *testing.T) {
	w := newWorld(t)
	p := w.clone("pat@example.com", "")
	gh := stubForge(t, `{"login":"octo-person","id":4242}`)
	for _, c := range []struct {
		req  Request
		want Act
	}{
		{Request{Record: "iss-5", Footprint: []string{"a.go"}}, ActReserved},
		{Request{Record: "iss-5", Renew: true}, ActRenewed},
		{Request{Record: "iss-5", Release: true}, ActReleased},
	} {
		res, err := p.do(c.req)
		if err != nil || res.Act != c.want || res.Holder != "octo-person" {
			t.Fatalf("%s: %+v, %v; want %s as octo-person", c.want, res, err, c.want)
		}
	}
	if n := gh.calls(); n != 1 {
		t.Fatalf("three reservation acts asked the forge for the holder %d times; want once", n)
	}
}

func TestTheHolderCacheKeepsOnlyADefiniteAnswerPerCheckoutAndCredential(t *testing.T) {
	r := newCheckout(t, "pat@example.com")
	o := Options{Home: t.TempDir()}
	gh := stubForge(t, `{"login":"octo-person","id":4242}`)
	want := Holder{Login: "octo-person", ID: 4242, NoReply: "4242+octo-person@users.noreply.github.com", Source: HolderFromToken}
	ask := func(root string, wantCalls int) {
		t.Helper()
		h, err := ResolveHolder(root, o)
		if err != nil || h != want {
			t.Fatalf("holder = %+v, %v; want %+v", h, err, want)
		}
		if n := gh.calls(); n != wantCalls {
			t.Fatalf("the forge was asked %d times; want %d", n, wantCalls)
		}
	}

	// A failure is never kept: each act asks again.
	gh.failing(true)
	for i := 1; i <= 2; i++ {
		if _, err := ResolveHolder(r.Root(), o); !errors.Is(err, ErrRefused) || gh.calls() != i {
			t.Fatalf("a failing forge: %v after %d calls; want a refusal after %d", err, gh.calls(), i)
		}
	}
	gh.failing(false)
	ask(r.Root(), 3)
	ask(r.Root(), 3)

	// Another checkout is asked for on its own.
	other := newCheckout(t, "pat@example.com")
	ask(other.Root(), 4)
	ask(other.Root(), 4)

	// Another forge credential in the environment is asked for on its own.
	t.Setenv("GH_TOKEN", "another-credential")
	ask(r.Root(), 5)
	ask(r.Root(), 5)
}

// ghAccountStore points gh's configuration at a directory of the test's own
// and returns the hosts.yml gh keeps its accounts in there.
func ghAccountStore(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", dir)
	return filepath.Join(dir, "hosts.yml")
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// With no token in the environment, gh's own account store decides the
// account: `gh auth switch`, `login` and `logout` rewrite hosts.yml and change
// no variable, so the cache must see the file.
func TestRewritingGhsAccountStoreAsksTheForgeAgain(t *testing.T) {
	r := newCheckout(t, "pat@example.com")
	o := Options{Home: t.TempDir()}
	hosts := ghAccountStore(t)
	writeFile(t, hosts, "github.com:\n    user: octo-person\n")
	gh := stubForge(t, `{"login":"octo-person","id":4242}`)
	ask := func(wantCalls int, why string) {
		t.Helper()
		if _, err := ResolveHolder(r.Root(), o); err != nil {
			t.Fatalf("%s: %v", why, err)
		}
		if n := gh.calls(); n != wantCalls {
			t.Fatalf("%s: the forge was asked %d times; want %d", why, n, wantCalls)
		}
	}
	ask(1, "the first act")
	ask(1, "an act with the store unchanged")
	writeFile(t, hosts, "github.com:\n    user: octo-other\n")
	ask(2, "an act after gh auth switch rewrote hosts.yml")
	ask(2, "an act with the switched store unchanged")
	if err := os.Remove(hosts); err != nil {
		t.Fatal(err)
	}
	ask(3, "an act after hosts.yml was removed")
	ask(3, "an act with hosts.yml still missing")
}

// ghHostsFile finds hosts.yml the way gh finds its configuration directory,
// and a relative directory resolves against the checkout gh runs in.
func TestGhsAccountStoreIsFoundTheWayGhFindsIt(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "work", "checkout")
	home := t.TempDir()
	for _, c := range []struct {
		name, ghConfig, xdg, appData string
		want                         string
	}{
		{"GH_CONFIG_DIR first", "/gh-config", "/xdg", "/appdata", "/gh-config/hosts.yml"},
		{"then XDG_CONFIG_HOME", "", "/xdg", "/appdata", "/xdg/gh/hosts.yml"},
		{"a relative GH_CONFIG_DIR is the checkout's", "rel", "", "", filepath.Join(root, "rel", "hosts.yml")},
		{"a relative XDG_CONFIG_HOME is the checkout's", "", "rel", "", filepath.Join(root, "rel", "gh", "hosts.yml")},
		{"else the home's", "", "", "", filepath.Join(home, ".config", "gh", "hosts.yml")},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("GH_CONFIG_DIR", c.ghConfig)
			t.Setenv("XDG_CONFIG_HOME", c.xdg)
			t.Setenv("AppData", c.appData)
			t.Setenv("HOME", home)
			if got := ghHostsFile(root); got != filepath.FromSlash(c.want) {
				t.Fatalf("ghHostsFile = %q; want %q", got, filepath.FromSlash(c.want))
			}
		})
	}
	t.Run("AppData on Windows only", func(t *testing.T) {
		t.Setenv("GH_CONFIG_DIR", "")
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("AppData", filepath.Join(home, "appdata"))
		t.Setenv("HOME", home)
		want := filepath.Join(home, ".config", "gh", "hosts.yml")
		if runtime.GOOS == "windows" {
			want = filepath.Join(home, "appdata", "GitHub CLI", "hosts.yml")
		}
		if got := ghHostsFile(root); got != want {
			t.Fatalf("ghHostsFile = %q; want %q", got, want)
		}
	})
}
