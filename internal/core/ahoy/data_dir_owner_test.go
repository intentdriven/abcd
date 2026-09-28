package ahoy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestDataDirHazardRefusesAnyoneElsesDirectory is iss-2609260057111315: the
// shape check refused a world-writable data directory, but a group-writable one
// or one another account owns lets that group or account supply both the cache
// artefact and its recorded hash just the same. Each is refused, for the data
// directory and for its cache/ subdirectory, and a directory the caller alone
// can write still passes.
func TestDataDirHazardRefusesAnyoneElsesDirectory(t *testing.T) {
	repo := t.TempDir()
	fresh := func(t *testing.T) string {
		t.Helper()
		data := t.TempDir()
		if err := os.Mkdir(filepath.Join(data, "cache"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(data, 0o755); err != nil {
			t.Fatal(err)
		}
		return data
	}

	if h := dataDirHazard(fresh(t), repo); h != "" {
		t.Fatalf("precondition: a directory only the caller can write is refused: %s", h)
	}

	for _, sub := range []string{"", "cache"} {
		data := fresh(t)
		if err := os.Chmod(filepath.Join(data, sub), 0o775); err != nil {
			t.Fatal(err)
		}
		h := dataDirHazard(data, repo)
		if h == "" {
			t.Errorf("a group-writable %q under the data directory is not refused", "./"+sub)
		} else if !strings.Contains(h, "group") {
			t.Errorf("the group-writable refusal does not say so: %s", h)
		}
	}

	data := fresh(t)
	t.Cleanup(fsutil.SwapOwnerUIDForTest(func(string) (uint32, error) {
		return uint32(os.Getuid()) + 1, nil
	}))
	if h := dataDirHazard(data, repo); h == "" {
		t.Error("a data directory another account owns is not refused")
	} else if !strings.Contains(h, "another account") {
		t.Errorf("the foreign-owner refusal does not say so: %s", h)
	}
}

// TestInstallIgnoresGroupWritableDataCache is the install-level twin of the
// world-writable case: an attested cache in a group-writable directory is still
// not promoted, and the note names the data directory.
func TestInstallIgnoresGroupWritableDataCache(t *testing.T) {
	home, _ := setupUserScope(t)
	binDir := filepath.Join(home, ".local", "bin")
	t.Setenv("PATH", binDir)
	data := seedDataCache(t, cacheArtefact)
	if err := os.Chmod(filepath.Join(data, "cache"), 0o775); err != nil {
		t.Fatal(err)
	}

	res, err := Install(adoptableRepo(t), installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	assertCacheIgnored(t, filepath.Join(binDir, "abcd"), res)
}
