package history

import (
	"os"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestCaptureRedactsEveryGitIdentity: with a global identity and a different
// repo-local one, a transcript naming both stores neither.
func TestCaptureRedactsEveryGitIdentity(t *testing.T) {
	repoRoot, _ := setupStore(t)
	global, local := gittest.SplitIdentity(t, repoRoot)

	record := captureText(t, repoRoot, "sess-ident", "user: cc "+global.Email+" and "+local.Email+"\nassistant: "+global.Name+" and "+local.Name+" both signed off\n")
	for _, p := range []gittest.Person{global, local} {
		for _, v := range []string{p.Email, p.Name} {
			if strings.Contains(record, v) {
				t.Errorf("stored record carries the git identity %q:\n%s", v, record)
			}
		}
	}
}

func captureText(t *testing.T, repoRoot, session, text string) string {
	t.Helper()
	res, err := Capture(repoRoot, testRootSHA, session, []byte(text), "native")
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	onDisk, err := os.ReadFile(res.Record.Path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	return string(onDisk)
}
