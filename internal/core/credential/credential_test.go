package credential

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const secretValue = "tok-0123456789-not-a-real-secret"

func writeStore(t *testing.T, home, body string, mode os.FileMode) string {
	t.Helper()
	dir := filepath.Join(home, ".abcd")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, StoreFileName)
	if err := os.WriteFile(p, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolveReadsAnOwnerOnlyStore(t *testing.T) {
	home := t.TempDir()
	writeStore(t, home, `{"hosting.cloudflare": "`+secretValue+`"}`, 0o600)
	got, err := Machine(home).Resolve("hosting.cloudflare")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != secretValue {
		t.Fatal("resolve returned a different value from the one stored")
	}
}

func TestAnAbsentStoreOrNameIsNotSet(t *testing.T) {
	home := t.TempDir()
	if _, err := Machine(home).Resolve("hosting.cloudflare"); !errors.Is(err, ErrNotSet) {
		t.Fatalf("no store: err = %v, want ErrNotSet", err)
	}
	writeStore(t, home, `{"other": "x"}`, 0o600)
	if _, err := Machine(home).Resolve("hosting.cloudflare"); !errors.Is(err, ErrNotSet) {
		t.Fatalf("absent name: err = %v, want ErrNotSet", err)
	}
	writeStore(t, home, `{"hosting.cloudflare": ""}`, 0o600)
	if _, err := Machine(home).Resolve("hosting.cloudflare"); !errors.Is(err, ErrNotSet) {
		t.Fatalf("empty value: err = %v, want ErrNotSet", err)
	}
}

// TestAStoreOthersCanReadIsRefused: a secret group or other can read is not
// kept, and it is refused loudly, never read and never treated as absent.
func TestAStoreOthersCanReadIsRefused(t *testing.T) {
	for _, mode := range []os.FileMode{0o640, 0o604, 0o644, 0o660} {
		home := t.TempDir()
		writeStore(t, home, `{"hosting.cloudflare": "`+secretValue+`"}`, mode)
		v, err := Machine(home).Resolve("hosting.cloudflare")
		if err == nil || errors.Is(err, ErrNotSet) {
			t.Fatalf("mode %o: err = %v, want a refusal", mode, err)
		}
		if v != "" || strings.Contains(err.Error(), secretValue) {
			t.Fatalf("mode %o: the refusal carries the value", mode)
		}
		if !strings.Contains(err.Error(), "0600") {
			t.Fatalf("mode %o: the refusal does not name the remedy: %v", mode, err)
		}
	}
}

func TestASymlinkedStoreIsRefused(t *testing.T) {
	home := t.TempDir()
	real := filepath.Join(t.TempDir(), "elsewhere.json")
	if err := os.WriteFile(real, []byte(`{"hosting.cloudflare": "`+secretValue+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(home, ".abcd", StoreFileName)); err != nil {
		t.Fatal(err)
	}
	if _, err := Machine(home).Resolve("hosting.cloudflare"); err == nil || errors.Is(err, ErrNotSet) {
		t.Fatalf("a symlinked store: err = %v, want a refusal", err)
	}
}

func TestAMalformedStoreNeverEchoesItsBytes(t *testing.T) {
	home := t.TempDir()
	writeStore(t, home, `{"hosting.cloudflare": "`+secretValue+`",,}`, 0o600)
	_, err := Machine(home).Resolve("hosting.cloudflare")
	if err == nil || errors.Is(err, ErrNotSet) {
		t.Fatalf("malformed: err = %v, want a refusal", err)
	}
	if strings.Contains(err.Error(), secretValue) {
		t.Fatal("the refusal echoes the store's contents")
	}
}

func TestAnUnsafeNameIsRefused(t *testing.T) {
	if _, err := Machine(t.TempDir()).Resolve("../x"); err == nil || errors.Is(err, ErrNotSet) {
		t.Fatalf("err = %v, want a refusal", err)
	}
}

func TestNoHomeIsNotSet(t *testing.T) {
	if _, err := Machine("").Resolve("hosting.cloudflare"); !errors.Is(err, ErrNotSet) {
		t.Fatalf("err = %v, want ErrNotSet", err)
	}
}

// TestAStoreNamingACredentialTwiceIsRefused: encoding/json reads a repeated key
// last-wins, and binds a case twin to the same entry, so a store naming one
// credential twice would resolve to whichever spelling came last, silently. It
// is refused, like every other store the reader cannot trust, and the refusal
// echoes neither value nor key (iss-2609260120380520).
func TestAStoreNamingACredentialTwiceIsRefused(t *testing.T) {
	const other = "tok-other-value-not-a-real-secret"
	for name, body := range map[string]string{
		"exact repeat":   `{"hosting.cloudflare": "` + secretValue + `", "hosting.cloudflare": "` + other + `"}`,
		"escaped repeat": `{"hosting.cloudflare": "` + secretValue + `", "hosting.cloudflare": "` + other + `"}`,
		"case twin":      `{"hosting.cloudflare": "` + secretValue + `", "Hosting.Cloudflare": "` + other + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			writeStore(t, home, body, 0o600)
			got, err := Machine(home).Resolve("hosting.cloudflare")
			if err == nil {
				t.Fatalf("a store naming a credential twice resolved (to the %s value)", map[bool]string{true: "first", false: "last"}[got == secretValue])
			}
			if errors.Is(err, ErrNotSet) {
				t.Fatalf("a store naming a credential twice must be refused, not read as unset: %v", err)
			}
			msg := err.Error()
			for _, leak := range []string{secretValue, other, "loudflare"} {
				if strings.Contains(msg, leak) {
					t.Fatalf("the refusal echoes %q: %s", leak, msg)
				}
			}
		})
	}
}
