package history

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// An earlier binary laid the transcript store out 0o755, and a directory that
// already exists keeps its mode (fsutil.EnsureRealDir never narrows). The store
// narrows its own records leaf on the next resolve, on the descriptor it proved,
// and touches nothing above it (iss-2609291610432030). Every assertion compares
// exact permission bits set with chmod or fchmod, so none depends on the umask.

// wideLegacyLayout lays the user-level store out the way an earlier binary did:
// every level 0o755. It returns the chain, ancestors first and the records leaf
// last.
func wideLegacyLayout(t *testing.T, home string) []string {
	t.Helper()
	abcd := filepath.Join(home, ".abcd")
	base := filepath.Join(abcd, "transcripts")
	lane := filepath.Join(base, testRootSHA)
	records := filepath.Join(lane, recordsDirName)
	chain := []string{abcd, base, lane, records}
	for _, d := range chain {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return chain
}

func permOf(t *testing.T, path string) os.FileMode {
	t.Helper()
	fi, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Mode().Perm()
}

// TestCaptureNarrowsAWideRecordsLeaf: the next write narrows a 0o755 records
// leaf to 0o700, and every ancestor keeps the mode it had.
func TestCaptureNarrowsAWideRecordsLeaf(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	chain := wideLegacyLayout(t, home)

	if _, err := Capture(t.TempDir(), testRootSHA, []byte("assistant: hi\n"), CaptureMeta{SessionID: "sess-leaf", Kind: "native"}); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	records := chain[len(chain)-1]
	if got := permOf(t, records); got != 0o700 {
		t.Errorf("records leaf is mode %#o, want 0o700: a leaf an earlier binary left 0o755 must be narrowed on the next write", got)
	}
	for _, d := range chain[:len(chain)-1] {
		if got := permOf(t, d); got != 0o755 {
			t.Errorf("ancestor %s is mode %#o, want 0o755 kept: only the records leaf is narrowed", d, got)
		}
	}
}

// TestResolveKeepsTheOwnerBitsOfTheRecordsLeaf: narrowing removes the group and
// other bits and nothing else, so it never widens what the owner had.
func TestResolveKeepsTheOwnerBitsOfTheRecordsLeaf(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	chain := wideLegacyLayout(t, home)
	records := chain[len(chain)-1]
	if err := os.Chmod(records, 0o775); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(t.TempDir(), testRootSHA); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := permOf(t, records); got != 0o700 {
		t.Errorf("records leaf is mode %#o, want 0o700", got)
	}
}

// TestResolveRefusesARecordsLeafAnotherAccountOwns: a leaf this account does not
// own is never changed, and the store refuses to write into it, because its
// owner can read it whatever its mode. The owner lookup is stubbed; the test
// never needs a second account.
func TestResolveRefusesARecordsLeafAnotherAccountOwns(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	chain := wideLegacyLayout(t, home)
	records := chain[len(chain)-1]

	restore := recordsLeafOwner
	recordsLeafOwner = func(os.FileInfo) (uint32, bool) { return uint32(os.Geteuid()) + 1, true }
	defer func() { recordsLeafOwner = restore }()

	_, err := Resolve(t.TempDir(), testRootSHA)
	var spe *StorePathError
	if !errors.As(err, &spe) || spe.Path != records {
		t.Fatalf("Resolve over a records leaf another account owns: got %v, want a *StorePathError naming %s", err, records)
	}
	for _, d := range chain {
		if got := permOf(t, d); got != 0o755 {
			t.Errorf("%s is mode %#o, want 0o755 untouched", d, got)
		}
	}

	recordsLeafOwner = func(os.FileInfo) (uint32, bool) { return 0, false }
	if _, err := Resolve(t.TempDir(), testRootSHA); !errors.As(err, &spe) {
		t.Fatalf("a records leaf whose owner cannot be read must be refused, got %v", err)
	}
	if got := permOf(t, records); got != 0o755 {
		t.Errorf("records leaf is mode %#o, want 0o755 untouched", got)
	}
}

// TestNarrowRecordsLeafNeverFollowsASymlink: a symlink where the leaf should be
// is refused, and the directory it points at keeps its mode.
func TestNarrowRecordsLeafNeverFollowsASymlink(t *testing.T) {
	scratch := t.TempDir()
	target := filepath.Join(scratch, "elsewhere")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(scratch, recordsDirName)
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	_, err := narrowRecordsLeaf(link)
	var spe *StorePathError
	if !errors.As(err, &spe) {
		t.Fatalf("narrowRecordsLeaf over a symlink: got %v, want a *StorePathError", err)
	}
	if got := permOf(t, target); got != 0o755 {
		t.Errorf("the symlink's target is mode %#o, want 0o755: the narrowing followed the link", got)
	}
}
