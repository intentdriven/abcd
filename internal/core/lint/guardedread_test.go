package lint

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// readRepoFile refuses a path that leaves the repository lexically or through a
// link, reads an in-repo link through the path containment judged, and keeps a
// missing file's os.IsNotExist error for callers that treat absence as a state.
func TestReadRepoFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "real.md", "real")
	if err := os.Symlink(filepath.Join(root, "real.md"), filepath.Join(root, "bridge.md")); err != nil {
		t.Fatal(err)
	}
	symlinkOut(t, root, "leak.md", "secret")
	if err := os.Symlink("/dev/zero", filepath.Join(root, "zero.md")); err != nil {
		t.Fatal(err)
	}
	if b, err := readRepoFile(root, "bridge.md", 64); err != nil || string(b) != "real" {
		t.Errorf("in-repo link: got %q, %v", b, err)
	}
	for _, rel := range []string{"leak.md", "../x.md", "/etc/hosts"} {
		if _, err := readRepoFile(root, rel, 64); err == nil || !strings.Contains(err.Error(), "inside the repository") {
			t.Errorf("%s: want a containment refusal, got %v", rel, err)
		}
	}
	if _, err := readRepoFile(root, "zero.md", 64); err == nil {
		t.Error("a link to a device outside the repository must be refused")
	}
	if _, err := readRepoFile(root, "absent.md", 64); !os.IsNotExist(err) {
		t.Errorf("absent: want an IsNotExist error, got %v", err)
	}
}

// No read in the lint package goes around the guard: every production file reads
// through readRepoFile, readRepoAbs or fsutil.ReadGuarded, so a new rule that
// reaches for os.ReadFile is refused here rather than found by the next sweep.
func TestLintReadsNothingUnguarded(t *testing.T) {
	raw := regexp.MustCompile(`\bos\.(ReadFile|Open|OpenFile)\(`)
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			code, _, _ := strings.Cut(line, "//")
			if raw.MatchString(code) {
				t.Errorf("%s:%d reads without the guard (use readRepoFile, readRepoAbs or fsutil.ReadGuarded): %s", name, i+1, strings.TrimSpace(line))
			}
		}
	}
}
