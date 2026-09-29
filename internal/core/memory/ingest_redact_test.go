package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ingest_redact_test.go — GHSA-j5f5-phgm-9m73. Memory ingest writes acquired
// source text into the committed .abcd/memory/ store (page bodies, the
// --keep-original copy, and the derived index.md/log.md) with no secret/PII
// scanner, and .gitignore does not exclude that tree. A PAT or an absolute home
// path in the source therefore reached the repository unredacted.
//
// Both spans below are FAKE (a ghp_ token shape of literal 'A's; a home path
// under a set-for-the-test $HOME). Nothing here is a live credential.

// TestIngestRedactsSecretsFromEveryStoreWrite drives a full local-file ingest
// with --keep-original through a distiller that echoes the source text straight
// into the page body (a non-redacting host), and proves the raw secret span and
// the raw home path survive in NONE of the four store writes: the page body, the
// kept-original copy, index.md, and log.md.
func TestIngestRedactsSecretsFromEveryStoreWrite(t *testing.T) {
	repo := t.TempDir()

	// A set-for-the-test home so the scanner's identity probe (which reads $HOME)
	// flags the path as the caller's own — deterministic across platforms.
	home := "/Users/testperson" // abcd-lint:allow
	t.Setenv("HOME", home)

	token := "ghp_" + strings.Repeat("A", 40)          // FAKE GitHub PAT shape
	homePath := home + "/private/credentials-note.txt" // FAKE home path

	sourceBody := "Rotate the token " + token + " every 24 hours.\n" +
		"The working copy lives at " + homePath + " on this machine.\n"
	src := writeSource(t, repo, "notes.md", sourceBody)

	// A distiller that copies the (normalised) source text verbatim into the page
	// body — modelling a host that does not itself redact. The core must redact.
	echoDistiller := func(normalised string, _ map[string]any) ([]map[string]any, error) {
		return []map[string]any{{
			"type": "topic", "domain": "auth", "slug": "tokens", "body": normalised,
		}}, nil
	}

	res, err := Ingest(IngestRequest{
		RepoRoot:     repo,
		Source:       src,
		KeepOriginal: true,
		Distiller:    echoDistiller,
		Now:          fixedNow,
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if res.Status != "ingested" {
		t.Fatalf("status = %q, want ingested", res.Status)
	}
	if res.KeptOriginal == "" {
		t.Fatalf("keep-original requested but no copy path returned (keepErr=%q)", res.KeepOriginalError)
	}

	mem := Dir(repo)
	targets := map[string]string{
		"page body":     filepath.Join(mem, "topic_auth_tokens.md"),
		"kept-original": filepath.Join(repo, res.KeptOriginal),
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
			t.Errorf("%s (%s) persists the raw token span unredacted", label, path)
		}
		if strings.Contains(content, homePath) {
			t.Errorf("%s (%s) persists the raw home path unredacted", label, path)
		}
	}
}
