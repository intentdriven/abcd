package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// redact_identity_test.go — the intent store's leg of the identity-scope
// defect (GHSA-v826-5jf4-p8xg and siblings): the fourth committed sink on
// the shared scanner, fixed by the same probe change; this is the proof.
func TestCreateFromTextRedactsEveryGitIdentity(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	global, local := gittest.SplitIdentity(t, root)
	draft := createDraft(t, root, "I want "+global.Name+" <"+global.Email+"> and "+local.Name+" <"+local.Email+"> to be told when the build breaks")
	name := strings.ToLower(draft[strings.LastIndex(draft, "\n")+1:])
	for _, p := range []gittest.Person{global, local} {
		for _, v := range []string{p.Email, p.Name} {
			if strings.Contains(draft, v) {
				t.Errorf("committed draft carries the git identity %q:\n%s", v, draft)
			}
		}
		if strings.Contains(name, strings.SplitN(p.Email, "@", 2)[0]) {
			t.Errorf("draft filename carries the mailbox of %q: %s", p.Email, name)
		}
	}
}

func createDraft(t *testing.T, root, text string) string {
	t.Helper()
	it, err := CreateFromText(root, text, "", "")
	if err != nil {
		t.Fatalf("CreateFromText: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, it.Path))
	if err != nil {
		t.Fatalf("created file unreadable: %v", err)
	}
	return string(data) + "\n" + filepath.Base(it.Path)
}
