package ahoy

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// iss-97: ahoy.Detect's marker/config reads used bare os.ReadFile — no size cap
// and, worse, a blocking open, so a FIFO planted at a marker/config path (cwd is
// attacker-influenced for any hook that calls Detect) would hang the read
// forever. The reads route through fsutil.ReadGuarded (O_NONBLOCK|O_NOFOLLOW,
// regular-file check, byte cap), so a FIFO returns promptly as not-regular and an
// oversized file is refused. These tests arm that: pre-fix each hangs on the FIFO
// (the select times out), post-fix each returns within the deadline.

// mkfifoOrSkip plants a FIFO at path, skipping on platforms without mkfifo.
func mkfifoOrSkip(t *testing.T, path string) {
	t.Helper()
	if err := syscall.Mkfifo(path, 0o644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
}

// withinDeadline runs fn and fails if it does not return within 3s (a blocking
// open on a FIFO would otherwise hang indefinitely).
func withinDeadline(t *testing.T, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		fn()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("%s hung on a FIFO (guarded open must not block)", what)
	}
}

func TestClassifyMarkerRejectsFifoPromptly(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "CLAUDE.md")
	mkfifoOrSkip(t, p)

	var got markerState
	withinDeadline(t, "classifyMarker", func() { got = classifyMarker(p) })
	if got != markerUnreadable {
		t.Fatalf("a FIFO marker must classify as unreadable, got %v", got)
	}
}

func TestReadConfigRejectsFifoPromptly(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	mkfifoOrSkip(t, configPath(dir))

	withinDeadline(t, "readConfig", func() { _, _ = readConfig(dir) })
}

// iss-109: the three residual manual-guard reads — verifyHookManifest (store.go)
// and the two gitignore.go read sites — did Lstat + IsRegular + size-cap and THEN
// a SEPARATE os.ReadFile, leaving a swap window between check and read on which no
// refusal is enforced on the SAME descriptor (the TOCTOU sibling of iss-97). The
// fix routes all three through fsutil.ReadGuarded, whose single O_NOFOLLOW|
// O_NONBLOCK open validates regular-file + size on the fd it reads. After the fix
// no bare os.ReadFile remains in ahoy's production source; every read is guarded.

// TestNoBareReadFileInAhoy is the structural detector: no non-test .go file in the
// ahoy package may call os.ReadFile — every read must route through
// fsutil.ReadGuarded so a type/symlink/size swap cannot slip between an Lstat check
// and the read. Before the iss-109 fix it flags gitignore.go (x2) and store.go's
// verifyHookManifest; after, it is empty.
func TestNoBareReadFileInAhoy(t *testing.T) {
	entries, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob ahoy sources: %v", err)
	}
	var offenders []string
	for _, path := range entries {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "os.ReadFile(") {
				offenders = append(offenders, path+":"+strconv.Itoa(i+1))
			}
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("bare os.ReadFile in ahoy (route through fsutil.ReadGuarded to close the Lstat->read TOCTOU):\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

// TestVerifyHookManifestRejectsFifo proves the structured reason string is
// preserved: a FIFO planted at pluginRoot/hooks/hooks.json is a non-regular leaf,
// so verifyHookManifest must return exactly "not a regular file" — promptly (the
// guarded open must not block on the FIFO), not a different string.
func TestVerifyHookManifestRejectsFifo(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	mkfifoOrSkip(t, filepath.Join(root, "hooks", "hooks.json"))

	var got string
	withinDeadline(t, "verifyHookManifest", func() { got = verifyHookManifest(root) })
	if got != "not a regular file" {
		t.Fatalf("FIFO hooks.json: reason = %q; want %q", got, "not a regular file")
	}
}

// TestVerifyHookManifestReportsAbsent preserves the "file absent" signal when
// hooks.json does not exist (os.IsNotExist path).
func TestVerifyHookManifestReportsAbsent(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := verifyHookManifest(root); got != "file absent" {
		t.Fatalf("missing hooks.json: reason = %q; want %q", got, "file absent")
	}
}

// TestVerifyHookManifestRejectsOversize preserves the size-cap reason string for a
// file past the 256KB cap.
func TestVerifyHookManifestRejectsOversize(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	big := make([]byte, 256*1024+1)
	if err := os.WriteFile(filepath.Join(root, "hooks", "hooks.json"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := verifyHookManifest(root); got != "file size exceeds 256KB cap" {
		t.Fatalf("oversize hooks.json: reason = %q; want %q", got, "file size exceeds 256KB cap")
	}
}

// TestApplyVisibilityBlockRefusesOversize preserves the write-path oversize
// refusal exactly: an oversize cwd/.gitignore must not be overwritten.
func TestApplyVisibilityBlockRefusesOversize(t *testing.T) {
	dir := t.TempDir()
	big := make([]byte, gitignoreMaxBytes+1)
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	wrote, err := applyVisibilityBlock(dir, "private")
	if wrote {
		t.Fatal("oversize .gitignore: wrote = true; want false (refusal preserves the file)")
	}
	if err == nil || err.Error() != "refusing to overwrite oversize .gitignore" {
		t.Fatalf("oversize .gitignore: err = %v; want \"refusing to overwrite oversize .gitignore\"", err)
	}
}

// TestGitignoreBlockDriftsFifoIsDrift proves the read-side bool is preserved: a
// FIFO at cwd/.gitignore is unsafe/unreadable, so drift is reported (fail-closed)
// promptly, never a hang.
func TestGitignoreBlockDriftsFifoIsDrift(t *testing.T) {
	dir := t.TempDir()
	mkfifoOrSkip(t, filepath.Join(dir, ".gitignore"))

	var got bool
	withinDeadline(t, "gitignoreBlockDrifts", func() { got = gitignoreBlockDrifts(dir, "private") })
	if !got {
		t.Fatal("FIFO .gitignore: drift = false; want true (fail-closed on unsafe shape)")
	}
}
