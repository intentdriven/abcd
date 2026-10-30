package claim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// A declared holder id may be paired with the GitHub login of the account it
// is the person's on (iss-2610290809526033): `h_<32 hex> <login>`. The login
// is read on this machine alone and is what a commit naming the id is held
// to; an id declared without one names no account.

const (
	pairedA = "h_0123456789abcdef0123456789abcdef"
	pairedB = "h_fedcba9876543210fedcba9876543210"
)

// TestADeclaredHolderIDMayBePairedWithItsAccountsLogin: a line holds one id,
// alone or followed by one login; an id with a login is paired with it, one
// without is unpaired, and both are the person's own.
func TestADeclaredHolderIDMayBePairedWithItsAccountsLogin(t *testing.T) {
	home := t.TempDir()
	declareHolderIDs(t, home, "# the other account\n"+pairedA+"  Maya  # the studio\n"+pairedB+"\n"+pairedA+" maya\n")
	s, err := ownSelf(home, "octo", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Declared) != 2 || s.Declared[0] != pairedA || s.Declared[1] != pairedB {
		t.Fatalf("declared = %v; want %s and %s", s.Declared, pairedA, pairedB)
	}
	if l, ok := s.pairedLogin(pairedA); !ok || l != "Maya" || s.unpaired(pairedA) {
		t.Errorf("%s pairs with %q (%v); want Maya", pairedA, l, ok)
	}
	if _, ok := s.pairedLogin(pairedB); ok || !s.unpaired(pairedB) {
		t.Errorf("%s reads as paired; want it unpaired", pairedB)
	}
	for _, h := range []string{pairedA, pairedB, s.ID, "octo"} {
		if !s.Owns(h) {
			t.Errorf("%q is not read as the person's own", h)
		}
	}
	if _, ok := s.pairedLogin(s.ID); ok || s.unpaired(s.ID) || !s.isOwn(s.ID) || s.isOwn(pairedA) {
		t.Errorf("the kept id reads as a declared one")
	}
	if r := ReaderSelf(t.TempDir(), Options{Home: home}); r.paired[pairedA] != "Maya" || len(r.Declared) != 2 {
		t.Errorf("a reader's self = %+v; want the same pairing", r)
	}
}

// TestAMalformedHolderIDsLineRefusesNamingOnlyItsNumber: a line with more
// than one login, a login that is not one, or a second pairing of an id
// refuses, naming the file and the line's number, never what it holds.
func TestAMalformedHolderIDsLineRefusesNamingOnlyItsNumber(t *testing.T) {
	for name, c := range map[string]struct{ body, secret string }{
		"two logins":       {pairedA + " maya quinnsecret\n", "quinnsecret"},
		"not a login":      {pairedA + " not_a_login!\n", "not_a_login"},
		"a login first":    {"pattersecret " + pairedA + "\n", "pattersecret"},
		"two pairings":     {pairedA + " maya\n" + pairedA + " quinnsecret\n", "quinnsecret"},
		"paired, then not": {pairedA + " mayasecret\n" + pairedA + "\n", "mayasecret"},
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			declareHolderIDs(t, home, c.body)
			_, err := ownSelf(home, "octo", true)
			if err == nil || !strings.Contains(err.Error(), "~/.abcd.noindex/holder-ids line ") {
				t.Fatalf("err = %v; want a refusal naming the file and the line", err)
			}
			if strings.Contains(err.Error(), c.secret) || strings.Contains(err.Error(), "maya") {
				t.Fatalf("the refusal %q echoes the line", err)
			}
		})
	}
}

// TestAHolderIDsFileNoOneElseMayTrustActsOnNoDeclaredID: a declaration
// another account owns or can write, one behind a symbolic link, one that is
// not a regular file and one past the size cap are each refused naming the
// file, so an act refuses before anything is pushed; a render reads no
// declared id from it, and still reads the kept id and the login.
func TestAHolderIDsFileNoOneElseMayTrustActsOnNoDeclaredID(t *testing.T) {
	for _, c := range []struct {
		name  string
		setup func(t *testing.T, home, path string)
	}{
		{"group writable", func(t *testing.T, _, path string) {
			if err := os.Chmod(path, 0o620); err != nil {
				t.Fatal(err)
			}
		}},
		{"world writable", func(t *testing.T, _, path string) {
			if err := os.Chmod(path, 0o606); err != nil {
				t.Fatal(err)
			}
		}},
		{"another owner", func(t *testing.T, _, _ string) {
			t.Cleanup(fsutil.SwapOwnerUIDForTest(func(p string) (uint32, error) {
				if filepath.Base(p) == "holder-ids" {
					return uint32(os.Getuid()) + 1, nil
				}
				return uint32(os.Getuid()), nil
			}))
		}},
		{"a symbolic link", func(t *testing.T, home, path string) {
			real := filepath.Join(t.TempDir(), "elsewhere")
			if err := os.WriteFile(real, []byte(pairedA+" maya\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(real, path); err != nil {
				t.Fatal(err)
			}
		}},
		{"not a regular file", func(t *testing.T, _, path string) {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{"past the size cap", func(t *testing.T, _, path string) {
			body := pairedA + " maya\n" + strings.Repeat("# padding\n", maxHolderIDsBytes/10+1)
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t)
			p := w.clone("1234+octo@users.noreply.github.com", "")
			if _, err := p.reserve("iss-1", "a.go"); err != nil {
				t.Fatal(err)
			}
			id := p.keptHolderID()
			path := declareHolderIDs(t, p.home, pairedA+" maya\n")
			c.setup(t, p.home, path)
			if _, _, err := declaredHolderIDs(p.home); err == nil || !strings.Contains(err.Error(), "~/.abcd.noindex/holder-ids") {
				t.Fatalf("err = %v; want a refusal naming the file", err)
			}
			tip := w.remoteTip("iss-2")
			if _, err := p.reserve("iss-2", "b.go"); err == nil || !strings.Contains(err.Error(), "~/.abcd.noindex/holder-ids") {
				t.Fatalf("an act over the refused declaration: %v; want it refused naming the file", err)
			}
			if w.remoteTip("iss-2") != tip {
				t.Fatal("an act over the refused declaration pushed")
			}
			r := p.self()
			if len(r.Declared) != 0 || len(r.paired) != 0 || r.Owns(pairedA) || r.ID != id || r.Login == "" || !r.Owns("octo") {
				t.Fatalf("a render's self = %+v; want the kept id and the login, and no declared id", r)
			}
		})
	}
}
