package credential

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// The write half of the interim store, for the one home it reads: the
// abcd-only file, owner-only (itd-2609081951381895 scope 5; the other homes
// are itd-2609221017023290's).

func TestSetMachineWritesAnOwnerOnlyStore(t *testing.T) {
	home := t.TempDir()
	changed, err := SetMachine(home, "openrouter", secretValue)
	if err != nil || !changed {
		t.Fatalf("SetMachine = %v, %v", changed, err)
	}
	p := filepath.Join(home, ".abcd", StoreFileName)
	fi, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 || !fi.Mode().IsRegular() {
		t.Fatalf("store mode = %v, want a regular file at 0600", fi.Mode())
	}
	di, err := os.Stat(filepath.Join(home, ".abcd"))
	if err != nil {
		t.Fatal(err)
	}
	if di.Mode().Perm()&0o077 != 0 {
		t.Fatalf("~/.abcd created at %o; a directory abcd creates for a secret is owner-only", di.Mode().Perm())
	}
	got, err := Machine(home).Resolve("openrouter")
	if err != nil || got != secretValue {
		t.Fatal("the value written does not resolve back")
	}
}

func TestSetMachineKeepsTheOtherEntries(t *testing.T) {
	home := t.TempDir()
	writeStore(t, home, `{"hosting.cloudflare": "cf-value"}`, 0o600)
	if _, err := SetMachine(home, "openrouter", secretValue); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".abcd", StoreFileName))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["hosting.cloudflare"] != "cf-value" || m["openrouter"] != secretValue || len(m) != 2 {
		t.Fatal("the store lost or changed an entry it was not asked to write")
	}
}

// TestSetMachineNeverOverwritesAStoredSecret: a name already holding another
// value is refused, and the refusal carries neither value; the same value is
// an unchanged no-op.
func TestSetMachineNeverOverwritesAStoredSecret(t *testing.T) {
	home := t.TempDir()
	writeStore(t, home, `{"openrouter": "old-value-0123456789"}`, 0o600)
	changed, err := SetMachine(home, "openrouter", secretValue)
	if err == nil || changed {
		t.Fatalf("SetMachine over a stored value = %v, %v; want a refusal", changed, err)
	}
	if strings.Contains(err.Error(), secretValue) || strings.Contains(err.Error(), "old-value") {
		t.Fatal("the refusal carries a value")
	}
	if got, _ := Machine(home).Resolve("openrouter"); got != "old-value-0123456789" {
		t.Fatal("the stored value was replaced")
	}
	changed, err = SetMachine(home, "openrouter", "old-value-0123456789")
	if err != nil || changed {
		t.Fatalf("SetMachine with the stored value = %v, %v; want unchanged", changed, err)
	}
}

// TestSetMachineRefusesWhatResolveRefuses: a store Resolve would refuse is not
// written over either, so a write never launders an unsafe file.
func TestSetMachineRefusesWhatResolveRefuses(t *testing.T) {
	home := t.TempDir()
	writeStore(t, home, `{}`, 0o644)
	if _, err := SetMachine(home, "openrouter", secretValue); err == nil {
		t.Fatal("wrote into a store others can read")
	}

	home = t.TempDir()
	real := filepath.Join(t.TempDir(), "elsewhere.json")
	if err := os.WriteFile(real, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(home, ".abcd", StoreFileName)); err != nil {
		t.Fatal(err)
	}
	if _, err := SetMachine(home, "openrouter", secretValue); err == nil {
		t.Fatal("wrote through a symlinked store")
	}
	if raw, _ := os.ReadFile(real); string(raw) != `{}` {
		t.Fatal("the symlink's target was written")
	}

	home = t.TempDir()
	writeStore(t, home, `{"a": ,}`, 0o600)
	if _, err := SetMachine(home, "openrouter", secretValue); err == nil {
		t.Fatal("wrote over a malformed store")
	}

	// A store naming one credential twice is one Resolve refuses
	// (iss-2609260120380520); rewriting it would keep one spelling and drop the
	// other unasked.
	home = t.TempDir()
	twice := `{"hosting.cloudflare": "a", "Hosting.Cloudflare": "b"}`
	p := writeStore(t, home, twice, 0o600)
	if _, err := SetMachine(home, "openrouter", secretValue); err == nil {
		t.Fatal("wrote over a store naming one credential twice")
	}
	if raw, _ := os.ReadFile(p); string(raw) != twice {
		t.Fatal("the store naming one credential twice was rewritten")
	}
}

func TestSetMachineRefusesABadNameOrValue(t *testing.T) {
	for _, tc := range []struct{ name, value string }{
		{"../x", secretValue},
		{"Open Router", secretValue},
		{"openrouter", ""},
		{"openrouter", "two\nlines"},
		{"openrouter", "esc\x1b[31m"},
		{"openrouter", " padded"},
		{"openrouter", strings.Repeat("k", MaxValueBytes+1)},
	} {
		home := t.TempDir()
		_, err := SetMachine(home, tc.name, tc.value)
		if err == nil {
			t.Errorf("SetMachine(%q, …) wrote; want a refusal", tc.name)
			continue
		}
		if tc.value != "" && strings.Contains(err.Error(), tc.value) {
			t.Errorf("SetMachine(%q, …): the refusal carries the value", tc.name)
		}
		if _, statErr := os.Lstat(filepath.Join(home, ".abcd", StoreFileName)); !errors.Is(statErr, os.ErrNotExist) {
			t.Errorf("SetMachine(%q, …): a refused write left a store", tc.name)
		}
	}
	if _, err := SetMachine("", "openrouter", secretValue); err == nil {
		t.Fatal("SetMachine with no home wrote")
	}
}

// TestConcurrentSetsKeepEveryEntry: the store is read, changed and renamed
// into place, so writers that overlap must be serialised or all but one
// entry is lost while each reports it wrote. Every writer's entry survives.
func TestConcurrentSetsKeepEveryEntry(t *testing.T) {
	const writers = 8
	for round := 0; round < 3; round++ {
		home := t.TempDir()
		var wg sync.WaitGroup
		errs := make(chan error, writers)
		start := make(chan struct{})
		for i := 0; i < writers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				changed, err := SetMachine(home, fmt.Sprintf("provider-%02d", i), fmt.Sprintf("throwaway-value-%02d", i))
				if err == nil && !changed {
					err = fmt.Errorf("writer %d reported no change", i)
				}
				errs <- err
			}(i)
		}
		close(start)
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatalf("round %d: SetMachine: %v", round, err)
			}
		}
		for i := 0; i < writers; i++ {
			got, err := Machine(home).Resolve(fmt.Sprintf("provider-%02d", i))
			if err != nil || got != fmt.Sprintf("throwaway-value-%02d", i) {
				t.Fatalf("round %d: provider-%02d was lost by a concurrent writer (%v)", round, i, err)
			}
		}
	}
}

// TestSetMachineNamesAnUnsafeLockRatherThanContention: a lock beside the
// store that is a symlink is refused, and the refusal says so; it is not the
// contention message, because retrying cannot cure a symlink. The symlink's
// target is never created and nothing is written.
func TestSetMachineNamesAnUnsafeLockRatherThanContention(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".abcd")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.Symlink(target, filepath.Join(dir, storeLockFileName)); err != nil {
		t.Fatal(err)
	}
	_, err := SetMachine(home, "openrouter", secretValue)
	if err == nil {
		t.Fatal("SetMachine succeeded through a symlinked lock")
	}
	msg := err.Error()
	if strings.Contains(msg, "retry") || strings.Contains(msg, "another abcd") {
		t.Fatalf("err = %v, want the unsafe lock named, not contention", err)
	}
	if !strings.Contains(msg, "~/.abcd/"+storeLockFileName) || !strings.Contains(msg, "not a regular file") ||
		!strings.Contains(msg, "nothing was written") {
		t.Fatalf("err = %v, want it to name the lock, that it is not a regular file, and that nothing was written", err)
	}
	if strings.Contains(msg, home) {
		t.Fatalf("err = %v carries the home path", err)
	}
	for _, p := range []string{target, filepath.Join(dir, StoreFileName)} {
		if _, statErr := os.Lstat(p); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("%s was created", p)
		}
	}
}
