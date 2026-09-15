package mode_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// newRepo stands up a hermetic git checkout and returns the root as git itself
// names it. The resolved form matters: on macOS the temp area is reached through
// a symlink, so git's toplevel and t.TempDir() are different strings for the same
// directory, and a test comparing the two would fail on the platform rather than
// on the behaviour.
func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustGit(t, dir, "init")
	return gitTopLevel(t, dir)
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func gitTopLevel(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// makeTier creates the local-ephemeral tier, the way an adopted repository has
// it. Nothing in this package creates it, so every write test that expects to
// succeed calls this first.
func makeTier(t *testing.T, root string) string {
	t.Helper()
	tier := filepath.Join(root, filepath.FromSlash(mode.TierRelPath))
	if err := os.MkdirAll(tier, 0o700); err != nil {
		t.Fatal(err)
	}
	return tier
}

func storePath(root string) string {
	return filepath.Join(root, filepath.FromSlash(mode.FileRelPath))
}

// readRawStore returns the store's bytes and whether it exists, without going
// through the package under test — so a test asserting "nothing was written" is
// not asking the reader whose refusal it is checking.
func readRawStore(t *testing.T, root string) ([]byte, bool) {
	t.Helper()
	data, err := os.ReadFile(storePath(root))
	// ENOTDIR is an absence too: it is what the open reports when something other
	// than a directory occupies the tier, so there is no store there either.
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return nil, false
	}
	if err != nil {
		t.Fatal(err)
	}
	return data, true
}

// TestAbsentStoreReadsManaged pins the rule the whole managed-only property rests
// on at the reading end: no file means nobody is waiting. It holds at each of the
// three depths the absence can take — no `.abcd/` at all (a repository abcd does
// not manage), a tier but no file (a managed repository nobody has parked), and an
// `.abcd/` without the tier.
func TestAbsentStoreReadsManaged(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, root string)
	}{
		{"no .abcd at all", func(*testing.T, string) {}},
		{"an .abcd without the local tier", func(t *testing.T, root string) {
			if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"the tier but no file", func(t *testing.T, root string) { makeTier(t, root) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := newRepo(t)
			tc.setup(t, root)

			got, err := mode.ReadAt(root)
			if err != nil {
				t.Fatalf("ReadAt: unexpected error: %v", err)
			}
			if got != mode.Managed {
				t.Fatalf("ReadAt = %q, want %q", got, mode.Managed)
			}

			// The cwd door must answer identically, and from a subdirectory too:
			// the store is per-repository, not per-directory.
			sub := filepath.Join(root, "internal", "deep")
			if err := os.MkdirAll(sub, 0o755); err != nil {
				t.Fatal(err)
			}
			if got, err := mode.Read(sub); err != nil || got != mode.Managed {
				t.Fatalf("Read(subdir) = %q, %v; want %q, nil", got, err, mode.Managed)
			}
		})
	}
}

// TestStateRoundTrips pins each of the three states through the writer and back
// out of the reader, and pins the stored form: one word, one line. A stored form
// that drifted would be read by nothing else, and the file is the contract
// between two writers and two readers.
func TestStateRoundTrips(t *testing.T) {
	for _, want := range mode.States() {
		t.Run(string(want), func(t *testing.T) {
			root := newRepo(t)
			makeTier(t, root)

			if err := mode.SetAt(root, want); err != nil {
				t.Fatalf("SetAt(%q): %v", want, err)
			}
			got, err := mode.ReadAt(root)
			if err != nil {
				t.Fatalf("ReadAt: %v", err)
			}
			if got != want {
				t.Fatalf("round trip: set %q, read %q", want, got)
			}

			raw, ok := readRawStore(t, root)
			if !ok {
				t.Fatalf("SetAt(%q) wrote no file at %s", want, mode.FileRelPath)
			}
			if string(raw) != string(want)+"\n" {
				t.Fatalf("stored form = %q, want one line %q", raw, string(want)+"\n")
			}

			fi, err := os.Lstat(storePath(root))
			if err != nil {
				t.Fatal(err)
			}
			if fi.Mode().Perm() != 0o600 {
				t.Fatalf("stored mode = %v, want 0600", fi.Mode().Perm())
			}
		})
	}
}

// TestSetOverwritesAPreviousState pins that the store holds one state, not an
// accumulation: the second write replaces the first rather than appending to it.
func TestSetOverwritesAPreviousState(t *testing.T) {
	root := newRepo(t)
	makeTier(t, root)

	if err := mode.SetAt(root, mode.ProductThinker); err != nil {
		t.Fatal(err)
	}
	if err := mode.SetAt(root, mode.Managed); err != nil {
		t.Fatal(err)
	}
	got, err := mode.ReadAt(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != mode.Managed {
		t.Fatalf("ReadAt after overwrite = %q, want %q", got, mode.Managed)
	}
	raw, _ := readRawStore(t, root)
	if string(raw) != "managed\n" {
		t.Fatalf("stored form after overwrite = %q", raw)
	}
}

// TestSetRefusesUnknownStateAndWritesNothing pins both halves of the closed
// vocabulary on the writing side: the refusal names the three legal words, and
// the store is left exactly as it was. An error returned over a truncated file
// would satisfy the first half and lose the state.
func TestSetRefusesUnknownStateAndWritesNothing(t *testing.T) {
	bad := []mode.State{"", "waiting", "Facilitator", "product_thinker", "managed\nfacilitator"}

	t.Run("over an existing state", func(t *testing.T) {
		for _, s := range bad {
			root := newRepo(t)
			makeTier(t, root)
			if err := mode.SetAt(root, mode.Facilitator); err != nil {
				t.Fatal(err)
			}
			before, _ := readRawStore(t, root)

			err := mode.SetAt(root, s)
			if err == nil {
				t.Fatalf("SetAt(%q): want a refusal", s)
			}
			if !errors.Is(err, mode.ErrUnknownState) {
				t.Fatalf("SetAt(%q) error %v does not wrap ErrUnknownState", s, err)
			}
			for _, legal := range mode.States() {
				if !strings.Contains(err.Error(), string(legal)) {
					t.Fatalf("SetAt(%q) refusal %q does not name %q", s, err, legal)
				}
			}

			after, ok := readRawStore(t, root)
			if !ok {
				t.Fatalf("SetAt(%q) removed the store", s)
			}
			if string(after) != string(before) {
				t.Fatalf("SetAt(%q) changed the store: %q -> %q", s, before, after)
			}
			if got, err := mode.ReadAt(root); err != nil || got != mode.Facilitator {
				t.Fatalf("after a refused write ReadAt = %q, %v; want %q", got, err, mode.Facilitator)
			}
		}
	})

	t.Run("over no state at all", func(t *testing.T) {
		for _, s := range bad {
			root := newRepo(t)
			tier := makeTier(t, root)

			if err := mode.SetAt(root, s); err == nil {
				t.Fatalf("SetAt(%q): want a refusal", s)
			}
			if _, ok := readRawStore(t, root); ok {
				t.Fatalf("SetAt(%q) created the store", s)
			}
			assertNoLitter(t, tier)
		}
	})
}

// TestSetRefusesWithoutTheLocalTier pins the managed-only property at the writing
// end, and pins the half that makes it a property rather than a check: the tier is
// REQUIRED and never created. Creating it would mint an abcd namespace in a
// repository abcd does not manage, which is the one thing that would make the
// state readable outside a managed tree.
func TestSetRefusesWithoutTheLocalTier(t *testing.T) {
	cases := []struct {
		name string
		// setup arranges the tree and returns the directory a symlinked tier
		// would have redirected the write into, or "" where the case plants no
		// symlink.
		setup func(t *testing.T, root string) (decoy string)
		// readerRefuses distinguishes a genuine absence, which reads as Managed,
		// from something standing where the tier belongs. The second is somebody
		// having planted a thing, so the reader fails closed rather than reporting
		// that nobody is waiting.
		readerRefuses bool
	}{
		{name: "no .abcd at all", setup: func(*testing.T, string) string { return "" }},
		{name: "an .abcd without the tier", setup: func(t *testing.T, root string) string {
			if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
				t.Fatal(err)
			}
			return ""
		}},
		{name: "a regular file where the tier belongs", readerRefuses: true, setup: func(t *testing.T, root string) string {
			if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, ".abcd", ".work.local"), []byte("not a tier\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			return ""
		}},
		{name: "a symlink standing in for the tier", readerRefuses: true, setup: func(t *testing.T, root string) string {
			if err := os.MkdirAll(filepath.Join(root, ".abcd"), 0o755); err != nil {
				t.Fatal(err)
			}
			decoy := t.TempDir()
			if err := os.Symlink(decoy, filepath.Join(root, ".abcd", ".work.local")); err != nil {
				t.Fatal(err)
			}
			return decoy
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := newRepo(t)
			decoy := tc.setup(t, root)

			err := mode.SetAt(root, mode.Facilitator)
			if err == nil {
				t.Fatal("SetAt without the local tier: want a refusal")
			}
			if !errors.Is(err, mode.ErrNoLocalTier) {
				t.Fatalf("SetAt error %v does not wrap ErrNoLocalTier", err)
			}
			if !strings.Contains(err.Error(), mode.TierRelPath) {
				t.Fatalf("refusal %q does not name the tier %q", err, mode.TierRelPath)
			}

			// Nothing was created on the way to the refusal — neither the tier
			// nor the namespace above it.
			if fi, statErr := os.Lstat(filepath.Join(root, filepath.FromSlash(mode.TierRelPath))); statErr == nil && fi.IsDir() {
				t.Fatalf("SetAt created the tier at %s", mode.TierRelPath)
			}
			if _, ok := readRawStore(t, root); ok {
				t.Fatal("SetAt wrote a store")
			}
			if decoy != "" {
				entries, readErr := os.ReadDir(decoy)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if len(entries) != 0 {
					t.Fatalf("SetAt wrote through the symlinked tier into %s: %v", decoy, entries)
				}
			}

			// A genuine absence still reads as managed; something planted where
			// the tier belongs is refused rather than read as an absence.
			got, readErr := mode.ReadAt(root)
			if tc.readerRefuses {
				if readErr == nil {
					t.Fatalf("ReadAt over %s = %q, want a refusal", tc.name, got)
				}
			} else if readErr != nil || got != mode.Managed {
				t.Fatalf("ReadAt = %q, %v; want %q, nil", got, readErr, mode.Managed)
			}
		})
	}
}

// TestSetIsAtomic pins the rename-commit contract this store inherits from
// fsutil: no temp file survives a write, and a symlink pre-planted at the store's
// path is REPLACED by the real file rather than written through. The second half
// is what stops a planted link turning a mode write into a write to a file the
// repository chose.
func TestSetIsAtomic(t *testing.T) {
	t.Run("no temporary file survives", func(t *testing.T) {
		root := newRepo(t)
		tier := makeTier(t, root)
		for _, s := range mode.States() {
			if err := mode.SetAt(root, s); err != nil {
				t.Fatal(err)
			}
			assertNoLitter(t, tier)
		}
		entries, err := os.ReadDir(tier)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != "mode" {
			t.Fatalf("tier holds %v, want exactly the mode file", entries)
		}
	})

	t.Run("a planted symlink at the store is replaced, not followed", func(t *testing.T) {
		root := newRepo(t)
		makeTier(t, root)
		outside := filepath.Join(t.TempDir(), "victim")
		if err := os.WriteFile(outside, []byte("untouched\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, storePath(root)); err != nil {
			t.Fatal(err)
		}

		if err := mode.SetAt(root, mode.ProductThinker); err != nil {
			t.Fatalf("SetAt over a planted symlink: %v", err)
		}
		victim, err := os.ReadFile(outside)
		if err != nil {
			t.Fatal(err)
		}
		if string(victim) != "untouched\n" {
			t.Fatalf("the write followed the symlink: victim now %q", victim)
		}
		fi, err := os.Lstat(storePath(root))
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			t.Fatal("the store is still a symlink after the write")
		}
		if got, err := mode.ReadAt(root); err != nil || got != mode.ProductThinker {
			t.Fatalf("ReadAt = %q, %v; want %q", got, err, mode.ProductThinker)
		}
	})
}

// assertNoLitter fails if the tier holds an abandoned atomic-write temp file.
func assertNoLitter(t *testing.T, tier string) {
	t.Helper()
	entries, err := os.ReadDir(tier)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".abcd-tmp-") {
			t.Fatalf("an atomic-write temp file survived: %s", e.Name())
		}
	}
}

// TestReadRefusesAMalformedStore pins that the reader fails closed rather than
// falling back to Managed. A word somebody wrote is not an absence, and reporting
// "nobody is waiting" over it would hide exactly the parked stop this store
// exists to make visible.
func TestReadRefusesAMalformedStore(t *testing.T) {
	for _, body := range []string{"", "\n", "waiting\n", "Managed\n", "managed facilitator\n", "managed\nfacilitator\n"} {
		t.Run(strings.ReplaceAll(body, "\n", `\n`), func(t *testing.T) {
			root := newRepo(t)
			makeTier(t, root)
			if err := os.WriteFile(storePath(root), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}

			got, err := mode.ReadAt(root)
			if err == nil {
				t.Fatalf("ReadAt of %q = %q, want a refusal", body, got)
			}
			if !errors.Is(err, mode.ErrUnknownState) {
				t.Fatalf("ReadAt error %v does not wrap ErrUnknownState", err)
			}
			if got != "" {
				t.Fatalf("ReadAt returned %q alongside its refusal", got)
			}
		})
	}
}

// TestReadRefusesANonRegularStore pins the guarded read at the leaf: a symlink, a
// directory or an oversize file at the store's path is a refusal, never Managed.
func TestReadRefusesANonRegularStore(t *testing.T) {
	cases := []struct {
		name  string
		plant func(t *testing.T, root string)
	}{
		{"a symlink", func(t *testing.T, root string) {
			target := filepath.Join(t.TempDir(), "elsewhere")
			if err := os.WriteFile(target, []byte("facilitator\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, storePath(root)); err != nil {
				t.Fatal(err)
			}
		}},
		{"a directory", func(t *testing.T, root string) {
			if err := os.Mkdir(storePath(root), 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{"an oversize file", func(t *testing.T, root string) {
			if err := os.WriteFile(storePath(root), make([]byte, 1<<20), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := newRepo(t)
			makeTier(t, root)
			tc.plant(t, root)

			got, err := mode.ReadAt(root)
			if err == nil {
				t.Fatalf("ReadAt over %s = %q, want a refusal", tc.name, got)
			}
			if got != "" {
				t.Fatalf("ReadAt returned %q alongside its refusal", got)
			}
		})
	}
}

// TestRootIsTheCheckoutRootFromAnyDepth pins that the store is addressed
// per-repository: a caller standing anywhere inside the checkout resolves the same
// root, so a verb run from a subdirectory reads and writes the one store rather
// than minting a second one under the subdirectory.
func TestRootIsTheCheckoutRootFromAnyDepth(t *testing.T) {
	root := newRepo(t)
	makeTier(t, root)
	sub := filepath.Join(root, "internal", "core", "mode")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := mode.Root(sub)
	if err != nil {
		t.Fatalf("Root(subdir): %v", err)
	}
	if got != root {
		t.Fatalf("Root(subdir) = %q, want the checkout root %q", got, root)
	}

	if err := mode.Set(sub, mode.Facilitator); err != nil {
		t.Fatalf("Set(subdir): %v", err)
	}
	if _, ok := readRawStore(t, root); !ok {
		t.Fatalf("Set from a subdirectory wrote no store at the checkout root")
	}
	if _, err := os.Lstat(filepath.Join(sub, ".abcd")); err == nil {
		t.Fatal("Set from a subdirectory minted a second namespace under it")
	}
	if got, err := mode.Read(sub); err != nil || got != mode.Facilitator {
		t.Fatalf("Read(subdir) = %q, %v; want %q", got, err, mode.Facilitator)
	}
}

// TestRootRefusesOutsideARepository pins the first of the two root refusals: a
// directory that is no repository at all has no store to address, and the answer
// is an error rather than a silent fallback to the working directory.
func TestRootRefusesOutsideARepository(t *testing.T) {
	dir := t.TempDir()
	if shaped := gitutil.RepoShapedRoot(dir); shaped != "" {
		t.Skipf("the temp area sits inside a repository-shaped tree at %s", shaped)
	}

	_, err := mode.Root(dir)
	if err == nil {
		t.Fatal("Root outside a repository: want a refusal")
	}
	assertRootRefusal(t, err)
	if !strings.Contains(err.Error(), "not inside a git repository") {
		t.Fatalf("refusal %q does not say the directory is not in a repository", err)
	}

	if _, err := mode.Read(dir); err == nil {
		t.Fatal("Read outside a repository: want a refusal")
	} else {
		assertRootRefusal(t, err)
	}
	if err := mode.Set(dir, mode.Facilitator); err == nil {
		t.Fatal("Set outside a repository: want a refusal")
	} else {
		assertRootRefusal(t, err)
	}
	if entries, err := os.ReadDir(dir); err != nil {
		t.Fatal(err)
	} else if len(entries) != 0 {
		t.Fatalf("a refused verb wrote into a non-repository: %v", entries)
	}
}

// TestRootRefusesARepositoryShapedTreeGitCannotAnswerFor pins the second root
// refusal, which is the one a fallback would swallow: the tree carries a `.git`
// marker, so content here CAN belong to a checkout, but git cannot name the root.
// Guessing the working directory there writes a second store into a real
// repository and reports success.
func TestRootRefusesARepositoryShapedTreeGitCannotAnswerFor(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	makeTier(t, dir)

	_, err := mode.Root(dir)
	if err == nil {
		t.Fatal("Root in a repository-shaped tree git cannot answer for: want a refusal")
	}
	assertRootRefusal(t, err)
	if !strings.Contains(err.Error(), "git could not name the repository root") {
		t.Fatalf("refusal %q does not say git could not answer", err)
	}

	if err := mode.Set(dir, mode.Facilitator); err == nil {
		t.Fatal("Set in an unanswerable tree: want a refusal")
	} else {
		assertRootRefusal(t, err)
	}
	if _, ok := readRawStore(t, dir); ok {
		t.Fatal("a refused Set wrote a store into an unanswerable tree")
	}
}

// assertRootRefusal checks that err is the shared checkout-root refusal carrying
// this store's noun — one error a front door maps to one exit code, phrased for
// the store the caller was addressing.
func assertRootRefusal(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, gitutil.ErrNoCheckoutRoot) {
		t.Fatalf("error %v does not wrap gitutil.ErrNoCheckoutRoot", err)
	}
	if !strings.Contains(err.Error(), mode.StoreName) {
		t.Fatalf("refusal %q does not name %q", err, mode.StoreName)
	}
}

// TestSetRefusesUnknownStateBeforeResolvingTheRoot pins the ordering: the closed
// vocabulary is checked first, so an invalid word is refused as an invalid word
// even where the root could not have been resolved anyway. A caller that saw a
// root refusal for a typo would fix the wrong thing.
func TestSetRefusesUnknownStateBeforeResolvingTheRoot(t *testing.T) {
	dir := t.TempDir()
	err := mode.Set(dir, "waiting")
	if err == nil {
		t.Fatal("Set with an unknown state: want a refusal")
	}
	if !errors.Is(err, mode.ErrUnknownState) {
		t.Fatalf("error %v does not wrap ErrUnknownState", err)
	}
	if errors.Is(err, gitutil.ErrNoCheckoutRoot) {
		t.Fatalf("the root refusal masked the vocabulary refusal: %v", err)
	}
}
