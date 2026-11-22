//go:build unix

package matrix

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// A copy controls everything in its clone, so the request is read from it as
// plain data: a link is never followed, a FIFO or device never waited on, and
// a file past the read bound never read; each is named in the manifest as
// withheld, never left out silently.
func TestACopysLinksFIFOsAndOversizeFilesAreWithheldUnread(t *testing.T) {
	src, frozen, store, clone := builtCopy(t)
	if err := os.Symlink("/etc/passwd", filepath.Join(clone, "link.go")); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(clone, "pipe.go"), 0o644); err != nil {
		t.Fatal(err)
	}
	big := strings.Repeat("// padding\n", maxReviewFileBytes/11+1)
	if err := os.WriteFile(filepath.Join(clone, "big.go"), []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(store), filepath.Join(clone, "out")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, man, err := assembleReview(ctx, reviewInputs{RepoRoot: src.Root(), Frozen: frozen, Dir: clone,
		Label: "copy-0a1b2c3d", Run: reviewRun, Store: store})
	if err != nil {
		t.Fatal(err)
	}
	withheld := map[string]string{}
	for _, w := range man.Withheld {
		withheld[w.Path] = w.Reason
	}
	for p, want := range map[string]string{
		"link.go": withheldLink, "pipe.go": withheldNotRegular, "big.go": withheldTooLarge, "out": withheldLink,
	} {
		if withheld[p] != want {
			t.Errorf("%s is withheld as %q, want %q (%v)", p, withheld[p], want, man.Withheld)
		}
	}
	data, _ := encodeReviewRequest(req)
	if strings.Contains(string(data), "/etc/passwd") || strings.Contains(string(data), "padding") {
		t.Fatal("nothing withheld reaches the request")
	}
}

// A directory of the copy that cannot be listed hides what is under it, so a
// frozen file there is withheld as unreadable, never shown to the reviewer as
// deleted.
func TestAFrozenFileUnderAnUnlistableDirectoryIsNotShownDeleted(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a directory's permissions do not bind the superuser")
	}
	src := gittest.NewRepo(t)
	src.Write("widget.go", "package widget\n")
	src.Write("pkg/a.go", "package pkg\n")
	src.Commit("frozen")
	frozen := src.Git("rev-parse", "HEAD")
	store := filepath.Join(t.TempDir(), "matrix-store")
	clone, err := makeClone(src.Root(), store, reviewRun, frozen, TrialKey{Task: "widget", Cell: "a", Repeat: 1}, keyAlias(TrialKey{Task: "widget", Cell: "a", Repeat: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(clone, "widget.go"), []byte("package widget\n\nconst B = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(clone, "pkg")
	if err := os.Chmod(pkg, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(pkg, 0o755) })
	_, man, err := assembleReview(context.Background(), reviewInputs{RepoRoot: src.Root(), Frozen: frozen, Dir: clone,
		Label: "copy-0a1b2c3d", Run: reviewRun, Store: store})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range man.Items {
		if it.Path == "pkg/a.go" {
			t.Fatalf("a file under an unlistable directory travelled as %s", it.Change)
		}
	}
	withheld := map[string]string{}
	for _, w := range man.Withheld {
		withheld[w.Path] = w.Reason
	}
	if withheld["pkg/a.go"] != withheldUnreadable || withheld["pkg"] != withheldUnreadable {
		t.Fatalf("the directory and the frozen file under it are withheld as unreadable: %v", man.Withheld)
	}
}
