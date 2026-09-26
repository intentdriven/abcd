package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomicCreatesWithPerm(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "f.txt") // parent dir does not exist yet
	if err := WriteFileAtomic(p, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("content = %q, want hello", got)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %o, want 600", fi.Mode().Perm())
	}
}

func TestWriteFileAtomicOverwrites(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := WriteFileAtomic(p, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(p, []byte("second"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "second" {
		t.Fatalf("content = %q, want second", got)
	}
	// No temp files linger after a successful write.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("expected exactly one file, got %d: %v", len(entries), entries)
	}
}

// TestWriteFileAtomicReplacesSymlink proves the leaf symlink is REPLACED, not
// written through: a pre-planted symlink at path must not clobber its target.
func TestWriteFileAtomicReplacesSymlink(t *testing.T) {
	dir := t.TempDir()
	victim := filepath.Join(dir, "victim.txt")
	if err := os.WriteFile(victim, []byte("do-not-touch"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(victim, link); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(link, []byte("new"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// The symlink is now a real file with the new content...
	fi, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("path is still a symlink; the write followed it")
	}
	// ...and the victim was not written through.
	got, _ := os.ReadFile(victim)
	if string(got) != "do-not-touch" {
		t.Fatalf("symlink target was clobbered: %q", got)
	}
}

func TestWriteFileAtomicPreserveMode(t *testing.T) {
	dir := t.TempDir()

	// New file defaults to 0644.
	fresh := filepath.Join(dir, "fresh.txt")
	if err := WriteFileAtomicPreserveMode(fresh, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(fresh); fi.Mode().Perm() != 0o644 {
		t.Fatalf("new file perm = %o, want 644", fi.Mode().Perm())
	}

	// Existing file keeps its mode across a rewrite.
	kept := filepath.Join(dir, "kept.txt")
	if err := WriteFileAtomic(kept, []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomicPreserveMode(kept, []byte("b")); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(kept); fi.Mode().Perm() != 0o600 {
		t.Fatalf("rewritten file perm = %o, want 600 (preserved)", fi.Mode().Perm())
	}
}

func TestIsRealDir(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "d")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "f")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkToDir := filepath.Join(dir, "ld")
	if err := os.Symlink(realDir, linkToDir); err != nil {
		t.Fatal(err)
	}

	if !IsRealDir(realDir) {
		t.Errorf("real dir reported as not-real")
	}
	if IsRealDir(file) {
		t.Errorf("file reported as real dir")
	}
	if IsRealDir(linkToDir) {
		t.Errorf("symlink-to-dir reported as real dir (must lstat, not follow)")
	}
	if IsRealDir(filepath.Join(dir, "missing")) {
		t.Errorf("missing path reported as real dir")
	}
}

// TestWriteFileAtomicAppliesPermViaDescriptor guards B16: the mode must be set on
// the open temp descriptor (fchmod), not chmod-by-name on the closed temp. The
// requested perm differs from CreateTemp's 0600 default, so a dropped/no-op chmod
// would surface here as the wrong final mode.
func TestWriteFileAtomicAppliesPermViaDescriptor(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := WriteFileAtomic(p, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o644 {
		t.Fatalf("perm = %o, want 644 — mode not applied via the descriptor", fi.Mode().Perm())
	}
	// No temp file may be left behind.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if len(e.Name()) >= 10 && e.Name()[:10] == ".abcd-tmp-" {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

// TestWriteFileAtomicPreserveModeFailsClosedOnStatFault guards B17: a real stat
// fault (here ELOOP from a self-referencing symlink at the target) is NOT
// absence, so PreserveMode must fail closed rather than silently default to 0644
// and write through — which would widen an existing restrictive mode.
func TestWriteFileAtomicPreserveModeFailsClosedOnStatFault(t *testing.T) {
	dir := t.TempDir()
	loop := filepath.Join(dir, "loop")
	if err := os.Symlink("loop", loop); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	// os.Stat(loop) now fails with ELOOP (a genuine fault, not not-exist).
	if err := WriteFileAtomicPreserveMode(loop, []byte("data")); err == nil {
		t.Fatal("want error on a real stat fault (ELOOP), got nil — mode would be silently reset to 0644")
	}
}

// TestReadGuardedRejectsSymlinkAndOversize proves the guarded read refuses a
// symlinked leaf (O_NOFOLLOW) and a file over the cap, and reads a normal file.
func TestReadGuardedRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(real, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Normal read.
	if b, err := ReadGuarded(real, 1024); err != nil || string(b) != "hello" {
		t.Fatalf("ReadGuarded(real) = %q, %v", b, err)
	}
	// Symlinked leaf is refused (O_NOFOLLOW -> ELOOP).
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("cannot symlink: %v", err)
	}
	if _, err := ReadGuarded(link, 1024); err == nil {
		t.Error("ReadGuarded followed a symlinked leaf")
	}
	// Oversize is refused with ErrTooBig.
	if _, err := ReadGuarded(real, 2); err != ErrTooBig {
		t.Errorf("ReadGuarded over cap = %v, want ErrTooBig", err)
	}
}

// TestEnsureRealDirCreatesAtThePermGiven holds the parameter that let three
// callers become one: the mode is the caller's, not the primitive's
// (iss-2609091128479544). A private store under the caller's home is 0o700 and a
// record directory in a shared worktree is 0o755, and a consolidation that
// silently picked one would have changed the other's directories on disk.
func TestEnsureRealDirCreatesAtThePermGiven(t *testing.T) {
	base := t.TempDir()
	for _, perm := range []os.FileMode{0o700, 0o755} {
		dir := filepath.Join(base, "d"+perm.String())
		if err := EnsureRealDir(dir, perm); err != nil {
			t.Fatalf("EnsureRealDir(%v): %v", perm, err)
		}
		fi, err := os.Lstat(dir)
		if err != nil {
			t.Fatal(err)
		}
		if got := fi.Mode().Perm(); got != perm {
			t.Errorf("created %v, want %v", got, perm)
		}
	}
}

// TestEnsureRealDirLeavesAnExistingModeAlone is the other half of that contract.
// Mkdir returns ErrExist without touching an existing directory, so re-running
// the primitive over a directory the caller made themselves must not widen or
// narrow it — otherwise every resolve would rewrite the operator's own choice.
func TestEnsureRealDirLeavesAnExistingModeAlone(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "existing")
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := EnsureRealDir(dir, 0o700); err != nil {
		t.Fatalf("EnsureRealDir over an existing directory: %v", err)
	}
	fi, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0o750 {
		t.Errorf("mode became %v; an existing directory must keep the mode it has", got)
	}
}

// TestEnsureRealDirRefusesASymlinkedLeaf is the proof step. os.Mkdir over a
// symlink returns EEXIST whether the link points at a directory or anywhere
// else, so the ErrExist branch alone would accept a planted redirect and every
// later write would land at the target.
func TestEnsureRealDirRefusesASymlinkedLeaf(t *testing.T) {
	base, elsewhere := t.TempDir(), t.TempDir()
	planted := filepath.Join(base, "store")
	if err := os.Symlink(elsewhere, planted); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	err := EnsureRealDir(planted, 0o700)
	if err == nil {
		t.Fatal("a symlinked leaf must be refused, not followed")
	}
	if !errors.Is(err, ErrNotRealDir) {
		t.Errorf("want ErrNotRealDir, got %T: %v", err, err)
	}
}

// TestEnsureRealDirRefusesAFileAtThePath covers the non-symlink half of the same
// refusal: a regular file occupying the name is EEXIST too.
func TestEnsureRealDirRefusesAFileAtThePath(t *testing.T) {
	p := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureRealDir(p, 0o700); !errors.Is(err, ErrNotRealDir) {
		t.Errorf("want ErrNotRealDir for a file at the path, got %v", err)
	}
}

// TestEnsureRealDirAllRefusesASymlinkedAncestor is the hardening the
// consolidation carries. os.MkdirAll follows a symlinked ancestor and creates
// the rest of the chain under its target — the hole one of the absorbed copies
// documented in its own doc comment — and walking level by level is what closes
// it. The target must stay empty: refusing after creating is not refusing.
func TestEnsureRealDirAllRefusesASymlinkedAncestor(t *testing.T) {
	base, elsewhere := t.TempDir(), t.TempDir()
	if err := os.Symlink(elsewhere, filepath.Join(base, "records")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	err := EnsureRealDirAll(base, "records/intents/drafts", 0o755)
	if err == nil {
		t.Fatal("a symlinked ancestor must be refused, not traversed")
	}
	if !errors.Is(err, ErrNotRealDir) {
		t.Errorf("want ErrNotRealDir, got %T: %v", err, err)
	}
	if entries, _ := os.ReadDir(elsewhere); len(entries) != 0 {
		t.Errorf("the walk created %d entr(ies) through the symlink", len(entries))
	}
}

// TestEnsureRealDirAllCreatesTheWholeChain is the ordinary path, and it also
// pins that every level lands at the caller's mode rather than only the leaf.
func TestEnsureRealDirAllCreatesTheWholeChain(t *testing.T) {
	base := t.TempDir()
	if err := EnsureRealDirAll(base, ".abcd/development/intents/drafts", 0o755); err != nil {
		t.Fatalf("EnsureRealDirAll: %v", err)
	}
	dir := base
	for _, seg := range []string{".abcd", "development", "intents", "drafts"} {
		dir = filepath.Join(dir, seg)
		fi, err := os.Lstat(dir)
		if err != nil {
			t.Fatalf("level %s: %v", dir, err)
		}
		if !fi.IsDir() {
			t.Fatalf("level %s is not a directory", dir)
		}
		if got := fi.Mode().Perm(); got != 0o755 {
			t.Errorf("level %s created %v, want 0755", dir, got)
		}
	}
}

// TestEnsureRealDirAllRefusesAnEscapingRel keeps the walk lexically bounded: a
// relative path is the only thing that may name levels under the caller's root,
// so an absolute path or a traversal is refused before a single Mkdir runs.
func TestEnsureRealDirAllRefusesAnEscapingRel(t *testing.T) {
	base := t.TempDir()
	for _, rel := range []string{"", ".", "/etc/abcd", "../escaped", "a/../../b"} {
		if err := EnsureRealDirAll(base, rel, 0o755); err == nil {
			t.Errorf("rel %q was accepted; a path that is not a clean relative path must be refused", rel)
		}
	}
	if entries, _ := os.ReadDir(base); len(entries) != 0 {
		t.Errorf("a refused rel created %d entr(ies)", len(entries))
	}
}

// TestEnsureRealDirAllRefusesAnUnrealBase closes the entry point: the walk
// proves every level it creates, so it must prove the one it starts from too.
func TestEnsureRealDirAllRefusesAnUnrealBase(t *testing.T) {
	base, elsewhere := t.TempDir(), t.TempDir()
	link := filepath.Join(base, "link")
	if err := os.Symlink(elsewhere, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := EnsureRealDirAll(link, "records", 0o755); !errors.Is(err, ErrNotRealDir) {
		t.Errorf("want ErrNotRealDir for a symlinked base, got %v", err)
	}
	if entries, _ := os.ReadDir(elsewhere); len(entries) != 0 {
		t.Errorf("the walk created %d entr(ies) through the symlinked base", len(entries))
	}
}

// TestProbeRealDirAllRefusesWhatEnsureRealDirAllRefuses: the read-only walk
// refuses the level the creating walk would refuse, naming it in the same
// *os.PathError, and creates nothing on any path. A missing level is not a
// refusal: there is nothing under it to read.
func TestProbeRealDirAllRefusesWhatEnsureRealDirAllRefuses(t *testing.T) {
	base, elsewhere := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "a", "b"), 0o700); err != nil {
		t.Fatal(err)
	}
	if ok, err := ProbeRealDirAll(base, "a/b"); !ok || err != nil {
		t.Errorf("a real chain = %v, %v; want true, nil", ok, err)
	}
	if ok, err := ProbeRealDirAll(base, "a/b/c/d"); ok || err != nil {
		t.Errorf("a missing level = %v, %v; want false, nil", ok, err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(base, "a", "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(elsewhere, "c"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "a", "file"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for rel, refused := range map[string]string{
		"a/link":   filepath.Join(base, "a", "link"),
		"a/link/c": filepath.Join(base, "a", "link"),
		"a/file":   filepath.Join(base, "a", "file"),
	} {
		ok, err := ProbeRealDirAll(base, rel)
		var pe *os.PathError
		if ok || !errors.Is(err, ErrNotRealDir) || !errors.As(err, &pe) || pe.Path != refused {
			t.Errorf("ProbeRealDirAll(%q) = %v, %v; want ErrNotRealDir naming %s", rel, ok, err, refused)
		}
		if werr := EnsureRealDirAll(base, rel, 0o700); !errors.As(werr, &pe) || pe.Path != refused {
			t.Errorf("EnsureRealDirAll(%q) = %v; the probe and the create disagree on %s", rel, werr, refused)
		}
	}
	link := filepath.Join(base, "a", "link")
	if ok, err := ProbeRealDirAll(link, "c"); ok || !errors.Is(err, ErrNotRealDir) {
		t.Errorf("a symlinked base = %v, %v; want ErrNotRealDir", ok, err)
	}
	for _, rel := range []string{"", "/etc", "../x"} {
		if ok, err := ProbeRealDirAll(base, rel); ok || err == nil {
			t.Errorf("rel %q = %v, %v; want a refusal", rel, ok, err)
		}
	}
	if entries, _ := os.ReadDir(filepath.Join(elsewhere, "c")); len(entries) != 0 {
		t.Errorf("a walk created %d entr(ies) through the link", len(entries))
	}
}
