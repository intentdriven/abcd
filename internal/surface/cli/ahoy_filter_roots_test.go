package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestBareAhoyNamesTheCheckAFilterRootsFileFailed (iss-2610091920437492): the
// gap is report-only, so the text board is where a person meets it, and a
// title alone would hide which check failed and what to do. The board's
// `filters:` line carries both.
func TestBareAhoyNamesTheCheckAFilterRootsFileFailed(t *testing.T) {
	hermeticEnv(t)
	home := os.Getenv("HOME")
	dir := abcdhome.Path(home)
	if err := os.MkdirAll(dir, abcdhome.DirMode); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(dir, "filter-roots")
	if err := os.WriteFile(f, []byte("/somewhere\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(f, 0o620); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	out, err := runCLIErr(t, "ahoy")
	if err != nil {
		t.Fatalf("ahoy: %v\n%s", err, out)
	}
	var line string
	for _, l := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(l, "  filters:     ") {
			line = l
		}
	}
	if line == "" {
		t.Fatalf("bare ahoy carries no filters: line for an ignored filter-roots file:\n%s", out)
	}
	if !strings.Contains(line, "chmod 600") || !strings.Contains(line, "filter-roots") {
		t.Errorf("the filters line names neither the file nor its repair:\n%s", line)
	}
	if strings.Contains(line, home) {
		t.Errorf("the filters line names the home directory in full:\n%s", line)
	}
}
