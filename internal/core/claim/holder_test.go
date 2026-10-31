package claim

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/core/tools"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The one holder resolver (iss-2610152046379890): a person's git identity
// names the holder through its noreply address or the machine key, and the
// session's own forge token is asked only when neither does, or when git's
// identity is not a person's at all. A machine identity is never shown back.

// forgeFake is a ForgeUser seam that answers login and id, or err, and counts
// its calls.
type forgeFake struct {
	login string
	id    int64
	err   error
	calls int
}

func (f *forgeFake) user(string) (string, int64, error) {
	f.calls++
	return f.login, f.id, f.err
}

// unsetCheckout is a checkout whose git has no user.name and no user.email,
// with no global or system file to lend it one.
func unsetCheckout(t *testing.T) *gittest.Repo {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	r := gittest.NewRepo(t)
	if err := os.MkdirAll(filepath.Join(r.Root(), ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	return r
}

// A machine identity the attribution gate's list refuses in the author role.
// It is a forge app's, deliberately: its noreply-shaped address names no
// GitHub username a person could hold.
const (
	botName = "renovate[bot]"
	botMail = "29139614+renovate[bot]@users.noreply.github.com"
)

func TestHolderFromNoReplyGitEmail(t *testing.T) {
	fake := &forgeFake{err: errors.New("must not be asked")}
	for _, c := range []struct {
		email string
		want  Holder
	}{
		{"4242+octo@users.noreply.github.com", Holder{Login: "octo", ID: 4242, NoReply: "4242+octo@users.noreply.github.com", Source: HolderFromGit}},
		{"octo@users.noreply.github.com", Holder{Login: "octo", NoReply: "octo@users.noreply.github.com", Source: HolderFromGit}},
	} {
		r := newCheckout(t, c.email)
		h, err := ResolveHolder(r.Root(), Options{Home: t.TempDir(), ForgeUser: fake.user})
		if err != nil || h != c.want {
			t.Fatalf("%s: holder = %+v, %v; want %+v", c.email, h, err, c.want)
		}
	}
	if fake.calls != 0 {
		t.Fatalf("a noreply user.email asked the forge %d times", fake.calls)
	}
}

func TestHolderFromMachineKey(t *testing.T) {
	r := newCheckout(t, "pat@example.com")
	home := machineConfig(t, `{"reservation": {"github_username": "octo"}}`)
	fake := &forgeFake{err: errors.New("must not be asked")}
	h, err := ResolveHolder(r.Root(), Options{Home: home, ForgeUser: fake.user})
	if want := (Holder{Login: "octo", Source: HolderFromConfig}); err != nil || h != want {
		t.Fatalf("holder = %+v, %v; want %+v", h, err, want)
	}
	if fake.calls != 0 {
		t.Fatalf("the machine key asked the forge %d times", fake.calls)
	}
}

func TestHolderFromTokenWhenGitIsPersonWithoutLogin(t *testing.T) {
	r := newCheckout(t, "pat@example.com")
	fake := &forgeFake{login: "octo-person", id: 4242}
	o := Options{Now: clockAt(t0), Home: t.TempDir(), ForgeUser: fake.user}
	h, err := ResolveHolder(r.Root(), o)
	want := Holder{Login: "octo-person", ID: 4242, NoReply: "4242+octo-person@users.noreply.github.com", Source: HolderFromToken}
	if err != nil || h != want {
		t.Fatalf("holder = %+v, %v; want %+v", h, err, want)
	}
	if fake.calls != 1 {
		t.Fatalf("the forge was asked %d times; want once", fake.calls)
	}
	// A person's own git identity stays the author of the reservation.
	p, err := prepare(r.Root(), Request{Record: "iss-5", Footprint: []string{"a.go"}}, o)
	if err != nil || p.self.Login != "octo-person" || !validHolderID(p.holder) || p.who != (identity{Name: "Pat Example", Email: "pat@example.com"}) {
		t.Fatalf("prepare = %+v, %v; want the holder id, octo-person resolved, authored by Pat Example", p, err)
	}
}

func TestHolderFromTokenWhenGitIsVendor(t *testing.T) {
	r := newCheckout(t, botMail)
	r.Git("config", "user.name", botName)
	home := machineConfig(t, `{"reservation": {"github_username": "octo"}}`)
	fake := &forgeFake{login: "octo-person", id: 4242}
	o := Options{Now: clockAt(t0), Home: home, ForgeUser: fake.user}
	h, err := ResolveHolder(r.Root(), o)
	want := Holder{Login: "octo-person", ID: 4242, NoReply: "4242+octo-person@users.noreply.github.com", Source: HolderFromToken}
	if err != nil || h != want {
		t.Fatalf("holder = %+v, %v; want %+v (the machine key is skipped)", h, err, want)
	}
	p, err := prepare(r.Root(), Request{Record: "iss-5", Footprint: []string{"a.go"}}, o)
	if err != nil || p.self.Login != "octo-person" || !validHolderID(p.holder) ||
		p.who != (identity{Name: "octo-person", Email: "4242+octo-person@users.noreply.github.com"}) {
		t.Fatalf("prepare = %+v, %v; want the holder id, and the token's account resolved and the author", p, err)
	}
}

func TestHolderFromTokenWhenGitUnset(t *testing.T) {
	r := unsetCheckout(t)
	fake := &forgeFake{login: "octo-person", id: 4242}
	o := Options{Now: clockAt(t0), Home: t.TempDir(), ForgeUser: fake.user}
	h, err := ResolveHolder(r.Root(), o)
	want := Holder{Login: "octo-person", ID: 4242, NoReply: "4242+octo-person@users.noreply.github.com", Source: HolderFromToken}
	if err != nil || h != want || fake.calls != 1 {
		t.Fatalf("holder = %+v, %v after %d calls; want %+v after one", h, err, fake.calls, want)
	}
	p, err := prepare(r.Root(), Request{Record: "iss-5", Footprint: []string{"a.go"}}, o)
	if err != nil || p.who != (identity{Name: "octo-person", Email: "4242+octo-person@users.noreply.github.com"}) {
		t.Fatalf("prepare = %+v, %v; want the token's account as author", p, err)
	}
}

func TestHolderTokenFailureRefusesWithoutEchoingIdentity(t *testing.T) {
	const prefix = "could not work out whose GitHub account this session uses: "
	for _, who := range []struct {
		name, email string
	}{{"Pat Example", "pat@example.com"}, {botName, botMail}, {"", ""}} {
		var r *gittest.Repo
		if who.email == "" {
			r = unsetCheckout(t)
		} else {
			r = newCheckout(t, who.email)
			r.Git("config", "user.name", who.name)
		}
		leaky := fmt.Errorf("gh: HTTP 401 for %s <%s>", who.name, who.email)
		for _, fake := range []*forgeFake{
			{err: leaky},
			{login: "not a login!", id: 7},
			{login: "octo", id: 0},
		} {
			o := Options{Now: clockAt(t0), Home: machineConfig(t, `{"reservation": {"github_username": "octo"}}`), ForgeUser: fake.user}
			if who.name == "Pat Example" {
				o.Home = t.TempDir() // a person with a machine key never reaches the token
			}
			_, herr := ResolveHolder(r.Root(), o)
			_, perr := prepare(r.Root(), Request{Record: "iss-5", Footprint: []string{"a.go"}}, o)
			for _, err := range []error{herr, perr} {
				if !errors.Is(err, ErrRefused) || !strings.HasPrefix(err.Error(), prefix) || !strings.HasSuffix(err.Error(), "; nothing was sent or pushed") {
					t.Fatalf("%q with %+v: %v; want the token refusal", who.email, fake, err)
				}
				for _, s := range []string{who.name, who.email, "renovate", "Pat", "example.com", "HTTP 401"} {
					if s != "" && strings.Contains(err.Error(), s) {
						t.Fatalf("the refusal echoes %q: %v", s, err)
					}
				}
			}
		}
	}
}

func TestRenderHolderNeverCallsForge(t *testing.T) {
	fake := &forgeFake{login: "octo-person", id: 4242}
	vendor := newCheckout(t, botMail)
	vendor.Git("config", "user.name", botName)
	person := newCheckout(t, "pat@example.com")
	for _, r := range []*gittest.Repo{vendor, person, unsetCheckout(t)} {
		if got := readerHolder(r.Root(), Options{Home: t.TempDir(), ForgeUser: fake.user}); got != "" {
			t.Fatalf("readerHolder = %q; want nothing without the forge", got)
		}
	}
	if fake.calls != 0 {
		t.Fatalf("a render asked the forge %d times; a render makes no network call", fake.calls)
	}
}

// TestHolderRefusesMachineAccountToken: a token whose account is a machine's
// (a release bot, an automation account) names no person, so it is refused in
// a fixed class that never echoes the login.
func TestHolderRefusesMachineAccountToken(t *testing.T) {
	const want = "could not work out whose GitHub account this session uses: the token belongs to a machine account, not a person's; nothing was sent or pushed"
	for _, login := range []string{"release-bot", "acme-automation"} {
		for _, r := range []*gittest.Repo{newCheckout(t, "pat@example.com"), unsetCheckout(t)} {
			fake := &forgeFake{login: login, id: 77}
			o := Options{Now: clockAt(t0), Home: t.TempDir(), ForgeUser: fake.user}
			_, herr := ResolveHolder(r.Root(), o)
			_, perr := prepare(r.Root(), Request{Record: "iss-5", Footprint: []string{"a.go"}}, o)
			for _, err := range []error{herr, perr} {
				if !errors.Is(err, ErrRefused) || err.Error() != want {
					t.Fatalf("token %s: %v; want %q", login, err, want)
				}
			}
		}
	}
}

// TestRepoConfigCannotBlockTokenFallback: a person with no noreply address
// and no machine key reaches the token whatever the repository commits to its
// own configuration, and whether or not a home resolves: only the machine
// layer is read for the key, so repository input can never block the person.
func TestRepoConfigCannotBlockTokenFallback(t *testing.T) {
	want := Holder{Login: "octo-person", ID: 4242, NoReply: "4242+octo-person@users.noreply.github.com", Source: HolderFromToken}
	for _, body := range []string{`{"reservation":{"x":1}}`, `{"reservation":`, `{"reservation":{"github_username":"mallory"}}`} {
		r := newCheckout(t, "pat@corp.com")
		r.Git("config", "user.name", "Pat")
		r.Write(".abcd/config.json", body)
		r.Commit("config")
		fake := &forgeFake{login: "octo-person", id: 4242}
		h, err := ResolveHolder(r.Root(), Options{Home: t.TempDir(), ForgeUser: fake.user})
		if err != nil || h != want || fake.calls != 1 {
			t.Fatalf("committed %s: holder = %+v, %v after %d calls; want %+v from the token", body, h, err, fake.calls, want)
		}
	}
	// An unresolved home holds no machine key either: the token answers.
	r := newCheckout(t, "pat@corp.com")
	t.Setenv("HOME", "")
	fake := &forgeFake{login: "octo-person", id: 4242}
	if h, err := ResolveHolder(r.Root(), Options{ForgeUser: fake.user}); err != nil || h != want {
		t.Fatalf("with no home: holder = %+v, %v; want %+v from the token", h, err, want)
	}
	// The person's own machine layer, malformed for this key, still refuses,
	// naming the file.
	home := machineConfig(t, `{"reservation": {"github_username": 5}}`)
	fake = &forgeFake{login: "octo-person", id: 4242}
	_, err := ResolveHolder(r.Root(), Options{Home: home, ForgeUser: fake.user})
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "config.json (machine layer): reservation.github_username") || fake.calls != 0 {
		t.Fatalf("a malformed machine key: %v after %d calls; want the refusal naming the file", err, fake.calls)
	}
}

// TestOverlongNoReplyFallsThrough: a noreply-shaped user.email whose local
// part is too long to be a GitHub username names no holder, so the next
// source is asked instead of the act being refused.
func TestOverlongNoReplyFallsThrough(t *testing.T) {
	long := "4242+" + strings.Repeat("a", 40) + "@users.noreply.github.com"
	r := newCheckout(t, long)
	fake := &forgeFake{err: errors.New("must not be asked")}
	h, err := ResolveHolder(r.Root(), Options{Home: machineConfig(t, `{"reservation": {"github_username": "octo"}}`), ForgeUser: fake.user})
	if want := (Holder{Login: "octo", Source: HolderFromConfig}); err != nil || h != want {
		t.Fatalf("with the machine key: holder = %+v, %v; want %+v", h, err, want)
	}
	fake = &forgeFake{login: "octo-person", id: 4242}
	h, err = ResolveHolder(r.Root(), Options{Home: t.TempDir(), ForgeUser: fake.user})
	if err != nil || h.Source != HolderFromToken || h.Login != "octo-person" {
		t.Fatalf("without it: holder = %+v, %v; want the token's", h, err)
	}
}

// TestNoForgeCallWhenRefusedLocally: a checkout the act would refuse on its
// own facts (no origin, no local tier) is refused before GitHub is asked.
func TestNoForgeCallWhenRefusedLocally(t *testing.T) {
	noOrigin := newCheckout(t, botMail)
	noOrigin.Git("config", "user.name", botName)
	noTier := gittest.NewRepo(t)
	noTier.Git("config", "user.name", botName)
	noTier.Git("config", "user.email", botMail)
	bare := filepath.Join(t.TempDir(), "origin.git")
	gittest.InitBare(t, bare)
	noTier.Git("remote", "add", "origin", bare)
	for _, c := range []struct {
		r    *gittest.Repo
		want string
	}{
		{noOrigin, "this checkout has no remote"},
		{noTier, "is not a directory in this checkout"},
	} {
		fake := &forgeFake{login: "octo-person", id: 4242}
		_, err := Reserve(c.r.Root(), Request{Record: "iss-5", Footprint: []string{"a.go"}}, Options{Now: clockAt(t0), Home: t.TempDir(), ForgeUser: fake.user})
		if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), c.want) || fake.calls != 0 {
			t.Fatalf("Reserve = %v after %d forge calls; want %q before any", err, fake.calls, c.want)
		}
	}
}

// TestForgeFailureClasses: the real call's failure reduces to a class, a
// timeout among them, and the class is what the refusal names.
func TestForgeFailureClasses(t *testing.T) {
	for _, c := range []struct {
		err  error
		want error
	}{
		{fmt.Errorf("%w within 30s", github.ErrTimeout), errForgeTimeout},
		{tools.Missing(errors.New("no gh"), "gh", tools.GitHubSettings), errForgeMissing},
		{errors.New("gh: HTTP 401 for pat@example.com"), errForgeNoToken},
	} {
		if got := forgeClass(c.err); got != c.want {
			t.Fatalf("forgeClass(%v) = %v; want %v", c.err, got, c.want)
		}
	}
	r := newCheckout(t, "pat@example.com")
	_, err := ResolveHolder(r.Root(), Options{Home: t.TempDir(), ForgeUser: func(string) (string, int64, error) { return "", 0, errForgeTimeout }})
	if want := "could not work out whose GitHub account this session uses: " + errForgeTimeout.Error() + "; nothing was sent or pushed"; err == nil || err.Error() != want {
		t.Fatalf("a timeout: %v; want %q", err, want)
	}
}

// TestHolderRefusalsAskForNoManualStep: the product thinker ruled out an
// identity a person must set by hand, so no refusal the resolver makes asks
// for one. Each says what abcd tried and that nothing was sent or pushed.
func TestHolderRefusalsAskForNoManualStep(t *testing.T) {
	manual := regexp.MustCompile(`(?i)\b(set|retry)\b`)
	var errs []error
	person := newCheckout(t, "pat@example.com")
	vendor := newCheckout(t, botMail)
	vendor.Git("config", "user.name", botName)
	for _, r := range []*gittest.Repo{person, vendor, unsetCheckout(t)} {
		who, err := gitIdentity(r.Root())
		if err != nil {
			t.Fatal(err)
		}
		_, err = resolveHolder(r.Root(), who, Options{Home: t.TempDir()}, false)
		errs = append(errs, err)
		for _, fe := range []error{errForgeMissing, errForgeNoToken, errForgeTimeout, errForgeAnswer} {
			fake := &forgeFake{err: fe}
			_, err = resolveHolder(r.Root(), who, Options{Home: t.TempDir(), ForgeUser: fake.user}, true)
			errs = append(errs, err)
		}
		_, err = resolveHolder(r.Root(), who, Options{Home: t.TempDir(), ForgeUser: (&forgeFake{login: "release-bot", id: 9}).user}, true)
		errs = append(errs, err)
	}
	for _, err := range errs {
		if err == nil {
			t.Fatal("a resolution with no source answered; want a refusal")
		}
		if m := manual.FindString(err.Error()); m != "" {
			t.Errorf("the refusal asks for a manual step (%q): %v", m, err)
		}
		if !strings.Contains(err.Error(), "nothing was sent or pushed") {
			t.Errorf("the refusal does not say nothing was sent or pushed: %v", err)
		}
	}
}
