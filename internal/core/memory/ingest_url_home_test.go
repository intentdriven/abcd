package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIngestWritesAndRedactsHomePathBehindAURLHost pins the store side of the
// URL case: a source naming the caller's home behind a URL host must be
// ingested — the page written — with the home redacted, neither committed
// verbatim nor refused as "home path survived redaction".
func TestIngestWritesAndRedactsHomePathBehindAURLHost(t *testing.T) {
	repo := t.TempDir()
	home := "/Users/zzhomeuser42" // abcd-audit:allow
	t.Setenv("HOME", home)

	body := "The build log is at https://ci.example.com" + home + "/build.log for review.\n"
	src := writeSource(t, repo, "notes.md", body)
	echo := func(normalised string, _ map[string]any) ([]map[string]any, error) {
		return []map[string]any{{"type": "topic", "domain": "ci", "slug": "logs", "body": normalised}}, nil
	}
	res, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: echo, Now: fixedNow})
	if err != nil {
		t.Fatalf("ingest refused the URL-shaped home: %v", err)
	}
	if res.Status != "ingested" {
		t.Fatalf("status = %q, want ingested", res.Status)
	}
	raw, err := os.ReadFile(filepath.Join(Dir(repo), "topic_ci_logs.md"))
	if err != nil {
		t.Fatalf("page not written: %v", err)
	}
	if strings.Contains(string(raw), home) {
		t.Errorf("the page carries the caller's home behind a URL host:\n%s", raw)
	}
	if !strings.Contains(string(raw), "for review") {
		t.Errorf("the page body was lost:\n%s", raw)
	}
}
