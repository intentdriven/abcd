package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFileBackRedactsSecretsBeforeTheStoreWrite closes the door beside
// GHSA-j5f5-phgm-9m73: the store redactor stood at the Ingest call site only,
// and `ask --file-back` handed the host-delegated distiller's page body to
// WritePages unscanned, so a secret or an absolute home path echoed into it
// reached the committed .abcd/memory store — the page, and the index and log
// derived from it. Redaction belongs to the single write primitive, so every
// PageWrite lands scanned whichever verb built it.
//
// Both spans are FAKE (a ghp_ token shape of literal 'A's; a home path under a
// set-for-the-test $HOME). Nothing here is a live credential.
func TestFileBackRedactsSecretsBeforeTheStoreWrite(t *testing.T) {
	repo := t.TempDir()
	home := "/Users/testperson"
	t.Setenv("HOME", home)
	seedAskStore(t, repo)

	token := "ghp_" + strings.Repeat("A", 40)        // FAKE GitHub PAT shape
	homePath := home + "/private/rotation-notes.txt" // FAKE home path

	ask, err := Ask(AskRequest{
		RepoRoot: repo,
		Question: "how does token rotation work?",
		FileBackPage: map[string]any{
			"type": "topic", "domain": "auth", "slug": "rotation-summary",
			"body": "# Rotation summary\nRotate " + token + " daily; notes live at " + homePath + ".\n",
		},
		DecideFileBack: func(DistilledPage) bool { return true },
		Now:            fixedNow,
	})
	if err != nil {
		t.Fatalf("ask --file-back: %v", err)
	}
	if ask.FileBack == nil || ask.FileBack.Status != "written" {
		t.Fatalf("file-back did not write: %+v", ask.FileBack)
	}

	mem := Dir(repo)
	targets := map[string]string{
		"filed page":    filepath.Join(mem, "topic_auth_rotation-summary.md"),
		"derived index": filepath.Join(mem, "index.md"),
		"derived log":   filepath.Join(mem, "log.md"),
	}
	for label, path := range targets {
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			t.Fatalf("%s (%s): %v", label, path, rerr)
		}
		content := string(raw)
		if strings.Contains(content, token) {
			t.Errorf("%s persists the raw token span unredacted", label)
		}
		if strings.Contains(content, homePath) {
			t.Errorf("%s persists the raw home path unredacted", label)
		}
	}
}
