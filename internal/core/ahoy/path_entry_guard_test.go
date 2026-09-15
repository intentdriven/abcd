package ahoy

import (
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// ~/.abcd/path-entry decides which binary the hook shims execute, so of the three
// home-scoped declaration files it is the one where the consequence is code
// execution rather than a location choice. It is therefore read behind the same
// three-part guard as ~/.abcd/trusted-roots and ~/.abcd/local-transcript-roots:
// a regular file, not writable by group or other, owned by this session's uid.
// "A declaration this process does not own, or one anyone can write, is not the
// caller's word" — and a record that is not the caller's word vouches for
// nothing, so it reports not-ok exactly as a truncated record does.
//
// The gap this closes is not the accepted same-uid residual
// (iss-2609012039107700). A group- or other-writable record lets a DIFFERENT
// local uid name a binary of their choosing in a directory they own at mode 755:
// the record's own guards were the only thing standing there (iss-2609091927085132).

// vouchedPathEntry writes a well-formed record naming target — the shape
// writePathEntry produces, so every refusal below is provoked by the file's
// ownership or mode alone and never by its content.
func vouchedPathEntry(t *testing.T, target string) {
	t.Helper()
	writeUserPathEntry(t, "path="+target+"\nbinary_sha256="+strings.Repeat("a", 64)+"\n")
}

// TestPathEntryIsIgnoredUnlessThisSessionOwnsIt covers the record's second
// acceptance criterion (a foreign-owned record reports not-ok) and the mode half
// of its first.
func TestPathEntryIsIgnoredUnlessThisSessionOwnsIt(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode os.FileMode
		why  string
	}{
		{"group-writable", 0o664, "another member of the group can rewrite which binary the hooks execute"},
		{"other-writable", 0o646, "any local uid can rewrite which binary the hooks execute"},
		{"group-and-other-writable", 0o666, "any local uid can rewrite which binary the hooks execute"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupUserScope(t)
			target := t.TempDir() + "/abcd"
			vouchedPathEntry(t, target)
			if err := os.Chmod(userPathEntryPath(), tc.mode); err != nil {
				t.Fatal(err)
			}
			if rec, ok := readPathEntry(); ok {
				t.Errorf("readPathEntry honoured a %v record vouching for %q: %s", tc.mode, rec.path, tc.why)
			}
		})
	}

	t.Run("owned by another uid", func(t *testing.T) {
		setupUserScope(t)
		target := t.TempDir() + "/abcd"
		vouchedPathEntry(t, target)
		// A second uid cannot be created by a test process, so the owner lookup
		// is substituted — the established answer to a branch the host cannot
		// provoke (fsutil.caseFoldingFS, rules.ownedByAnother).
		restore := fsutil.SwapOwnerUIDForTest(func(string) (uint32, error) {
			return uint32(os.Getuid()) + 1, nil
		})
		t.Cleanup(restore)
		if rec, ok := readPathEntry(); ok {
			t.Errorf("readPathEntry honoured a foreign-owned record vouching for %q", rec.path)
		}
	})

	// "I could not learn who owns this" and "I own this" are different answers,
	// and a fail-closed gate must not spell them the same way.
	t.Run("owner unreadable", func(t *testing.T) {
		setupUserScope(t)
		target := t.TempDir() + "/abcd"
		vouchedPathEntry(t, target)
		restore := fsutil.SwapOwnerUIDForTest(func(string) (uint32, error) {
			return 0, os.ErrPermission
		})
		t.Cleanup(restore)
		if rec, ok := readPathEntry(); ok {
			t.Errorf("readPathEntry honoured a record whose owner could not be read, vouching for %q", rec.path)
		}
	})
}

// TestPathEntryCorrectlyOwnedIsUnchanged is the record's third acceptance
// criterion. The guard must refuse the two new shapes and NOTHING else: a
// correctly owned, correctly permissioned record still reads exactly as it did.
func TestPathEntryCorrectlyOwnedIsUnchanged(t *testing.T) {
	for _, mode := range []os.FileMode{0o600, 0o640, 0o644, 0o755} {
		t.Run(mode.String(), func(t *testing.T) {
			setupUserScope(t)
			target := t.TempDir() + "/abcd"
			vouchedPathEntry(t, target)
			if err := os.Chmod(userPathEntryPath(), mode); err != nil {
				t.Fatal(err)
			}
			rec, ok := readPathEntry()
			if !ok {
				t.Fatalf("readPathEntry refused a %v record this session owns; the guard is refusing more than the two unowned shapes", mode)
			}
			if rec.path != target {
				t.Errorf("readPathEntry read path=%q, want %q", rec.path, target)
			}
		})
	}
}
