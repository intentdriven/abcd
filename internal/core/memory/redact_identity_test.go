package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestIngestRedactsEveryGitIdentity: with a global identity and a different
// repo-local one, a source naming both stores neither.
func TestIngestRedactsEveryGitIdentity(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	global, local := gittest.SplitIdentity(t, repo)

	stores := ingestEcho(t, repo, "Ask "+global.Name+" <"+global.Email+"> or "+local.Name+" <"+local.Email+"> about the rotation.\n")
	for label, content := range stores {
		for _, p := range []gittest.Person{global, local} {
			for _, v := range []string{p.Email, p.Name} {
				if strings.Contains(content, v) {
					t.Errorf("%s persists the git identity %q:\n%s", label, v, content)
				}
			}
		}
	}
}

func echoPage(normalised string, _ map[string]any) ([]map[string]any, error) {
	return []map[string]any{{"type": "topic", "domain": "auth", "slug": "keys", "body": normalised}}, nil
}

func ingestEcho(t *testing.T, repo, sourceBody string) map[string]string {
	t.Helper()
	src := writeSource(t, repo, "notes.md", sourceBody)
	res, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, KeepOriginal: true, Distiller: echoPage, Now: fixedNow})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if res.KeptOriginal == "" {
		t.Fatalf("keep-original requested but no copy path returned (keepErr=%q)", res.KeepOriginalError)
	}
	out := map[string]string{}
	for label, path := range map[string]string{
		"page body":     filepath.Join(Dir(repo), "topic_auth_keys.md"),
		"kept-original": filepath.Join(repo, res.KeptOriginal),
	} {
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			t.Fatalf("%s (%s): %v", label, path, rerr)
		}
		out[label] = string(raw)
	}
	return out
}
