package claim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// A ~/.abcd.noindex/holder-ids the guarded reader refuses is never read
// quietly (iss-2610290809526033): every render that reads whose holds are
// the person's names the class of the refusal, never the file's contents or
// a login, and says no declared id is read; an act still refuses outright.

// TestARefusedHolderIDsFileIsNamedByItsClassOnEveryRead: each way the file
// is refused gives its class on the reader's Self and on what --holder
// reads, with the kept id still read, and a line naming the class alone;
// the act's read still refuses, and an absent file gives no class.
func TestARefusedHolderIDsFileIsNamedByItsClassOnEveryRead(t *testing.T) {
	for _, c := range []struct {
		name, class string
		setup       func(t *testing.T, path string)
	}{
		{"group writable", HolderIDsWritableByOthers, func(t *testing.T, path string) { chmod(t, path, 0o620) }},
		{"world writable", HolderIDsWritableByOthers, func(t *testing.T, path string) { chmod(t, path, 0o606) }},
		{"another owner", HolderIDsForeignOwner, func(t *testing.T, _ string) {
			t.Cleanup(fsutil.SwapOwnerUIDForTest(func(p string) (uint32, error) {
				if filepath.Base(p) == "holder-ids" {
					return uint32(os.Getuid()) + 1, nil
				}
				return uint32(os.Getuid()), nil
			}))
		}},
		{"a symbolic link", HolderIDsNotRegular, func(t *testing.T, path string) {
			real := filepath.Join(t.TempDir(), "elsewhere")
			if err := os.WriteFile(real, []byte(pairedA+" mayasecret\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(real, path); err != nil {
				t.Fatal(err)
			}
		}},
		{"not a regular file", HolderIDsNotRegular, func(t *testing.T, path string) {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{"past the size cap", HolderIDsTooLarge, func(t *testing.T, path string) {
			body := pairedA + " mayasecret\n" + strings.Repeat("# padding\n", maxHolderIDsBytes/10+1)
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{"a malformed line", HolderIDsMalformed, func(t *testing.T, path string) {
			if err := os.WriteFile(path, []byte(pairedA+" mayasecret quinnsecret\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			home := t.TempDir()
			keepHolderID(t, home, pairedB)
			path := declareHolderIDs(t, home, pairedA+" mayasecret\n")
			c.setup(t, path)
			r := ReaderSelf(t.TempDir(), Options{Home: home})
			if r.HolderIDsRefused != c.class || r.ID != pairedB || len(r.Declared) != 0 {
				t.Fatalf("a render's self = %+v; want the kept id, no declared id, and the refusal %q", r, c.class)
			}
			h, err := OwnSelf("octo", Options{Home: home})
			if err != nil || h.HolderIDsRefused != c.class || h.ID != pairedB || len(h.Declared) != 0 {
				t.Fatalf("--holder's self = %+v, %v; want the kept id, no declared id, and the refusal %q", h, err, c.class)
			}
			if _, err := ownSelf(home, "octo", false); err == nil {
				t.Fatal("an act's read of the refused declaration did not refuse")
			}
			line := HolderIDsRefusedLine(c.class)
			if !strings.HasPrefix(line, abcdhome.Display("holder-ids")+" is refused (") || !strings.Contains(line, "no declared id is read") ||
				strings.Contains(line, "secret") || strings.Contains(line, "maya") || strings.Contains(line, pairedA) || strings.Contains(line, home) {
				t.Fatalf("line %q; want the file and the class named, and nothing it holds", line)
			}
		})
	}
	t.Run("absent", func(t *testing.T) {
		home := t.TempDir()
		keepHolderID(t, home, pairedB)
		if r := ReaderSelf(t.TempDir(), Options{Home: home}); r.HolderIDsRefused != "" || HolderIDsRefusedLine("") != "" {
			t.Fatalf("an absent declaration gives the refusal %q; want none", r.HolderIDsRefused)
		}
	})
}

func chmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}
