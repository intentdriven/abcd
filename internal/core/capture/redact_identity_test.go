package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestCaptureRedactsEveryGitIdentity: with a global identity and a different
// repo-local one, a capture naming both commits neither — not in the body and
// not in the slug the filename is built from.
func TestCaptureRedactsEveryGitIdentity(t *testing.T) {
	repo, ir := ledger(t)
	t.Setenv("HOME", t.TempDir())
	global, local := gittest.SplitIdentity(t, repo)

	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir,
		Text:     "contact " + global.Email + " or " + local.Email + " about this flake; " + global.Name + " and " + local.Name + " both saw it",
		Severity: SeverityMinor, Category: "process", Source: "user-observation", FoundDuring: "t",
	})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	record := readLedgerFile(t, repo, res.Path)
	name := strings.ToLower(filepath.Base(res.Path))
	for _, p := range []gittest.Person{global, local} {
		for _, v := range []string{p.Email, p.Name} {
			if strings.Contains(record, v) {
				t.Errorf("record carries the git identity %q:\n%s", v, record)
			}
		}
		if strings.Contains(name, strings.SplitN(p.Email, "@", 2)[0]) {
			t.Errorf("record filename carries the mailbox of %q: %s", p.Email, res.Path)
		}
	}
}

func readLedgerFile(t *testing.T, repo, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, rel))
	if err != nil {
		data, err = os.ReadFile(rel)
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
	}
	return string(data)
}
