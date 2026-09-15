package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lint_residue_test.go — GHSA-xj89-cc2c-wgwr. The curator health-check ran
// source-class, licence and quotation checks and imported no scanner, so a
// store already carrying a secret or the caller's home — written before the
// write-time redactor existed, or planted by hand — produced zero findings and
// a clean exit. MR001 is the read side of the store redactor: every stored
// page, the sources registry and each text kept-original are scanned, and a
// span the write side would have refused or rewritten is a blocker. Lint
// reports and never rewrites the store (adr-13). The message names the kind
// and the line, never the span, so the report files stay clean too.
//
// Fixtures are seeded through Ingest with marker words and the secret is
// planted by editing the written files afterwards: the write path redacts
// every leaf it introduces (GHSA-x46m), so a dirty fixture cannot be written
// through it, and a hand-written page would not follow the store's own YAML.
// Both spans are FAKE. Nothing here is a live credential.

func seedResidueStore(t *testing.T, repo string, keep bool) IngestResult {
	t.Helper()
	src := writeSource(t, repo, "notes.md", "Rotate the MARKERTOKEN every day.\n")
	res, err := Ingest(IngestRequest{
		RepoRoot: repo, Source: src, KeepOriginal: keep, Now: fixedNow,
		Distiller: oneTopicDistiller("topic", "auth", "tokens", "# Tokens\nRotate MARKERTOKEN daily; notes at MARKERHOME.\n"),
	})
	if err != nil {
		t.Fatalf("seed ingest: %v", err)
	}
	return res
}

func plantResidue(t *testing.T, path string, replacements map[string]string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for from, to := range replacements {
		if !strings.Contains(text, from) {
			t.Fatalf("fixture drift: %s no longer carries the marker %q:\n%s", path, from, text)
		}
		text = strings.ReplaceAll(text, from, to)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func residueFindingsFor(res LintResult, file string) []Finding {
	var out []Finding
	for _, f := range res.Findings {
		if f.Code == "MR001" && f.File == file {
			out = append(out, f)
		}
	}
	return out
}

func reportsMustBeClean(t *testing.T, res LintResult, spans ...string) {
	t.Helper()
	for _, name := range []string{"report.json", "report.md"} {
		raw, err := os.ReadFile(filepath.Join(res.ReportDir, name))
		if err != nil {
			t.Fatalf("lint must always write %s: %v", name, err)
		}
		for _, s := range spans {
			if strings.Contains(string(raw), s) {
				t.Errorf("GHSA-xj89: %s carries the raw span %q — the report must name the kind, never the value", name, s[:8])
			}
		}
	}
}

func TestLintReportsSecretResidueInStoredPages(t *testing.T) {
	t.Run("page frontmatter and body", func(t *testing.T) {
		repo := t.TempDir()
		token, homePath := x46mSpans(t)
		seedResidueStore(t, repo, false)
		page := filepath.Join(Dir(repo), "topic_auth_tokens.md")
		plantResidue(t, page, map[string]string{
			"title: notes.md": "title: keys " + token,
			"MARKERTOKEN":     token,
			"MARKERHOME":      homePath,
		})
		res, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
		if err != nil {
			t.Fatalf("lint: %v", err)
		}
		found := residueFindingsFor(res, page)
		if len(found) < 2 {
			t.Fatalf("GHSA-xj89: lint reported %d MR001 finding(s) for a page carrying a PAT in citation.title and the body plus a home path; want the frontmatter line and the body line:\n%+v", len(found), res.Findings)
		}
		var sawToken, sawHome bool
		for _, f := range found {
			if f.Severity != "blocker" {
				t.Errorf("MR001 must be a blocker, got %q", f.Severity)
			}
			if f.Line <= 0 {
				t.Errorf("MR001 must locate the line, got %d", f.Line)
			}
			if strings.Contains(f.Message, token) || strings.Contains(f.Message, homePath) || strings.Contains(f.Suggestion, token) {
				t.Errorf("MR001 message carries the raw span: %q", f.Message)
			}
			if strings.Contains(f.Message, "github_pat") {
				sawToken = true
			}
			if strings.Contains(f.Message, "home_path") {
				sawHome = true
			}
		}
		if !sawToken || !sawHome {
			t.Errorf("MR001 must name the kind: github_pat=%v home_path=%v in %+v", sawToken, sawHome, found)
		}
		if res.ExitCode != 1 || res.Summary.Blockers < 2 {
			t.Errorf("exit=%d blockers=%d, want a nonzero exit with the residue counted", res.ExitCode, res.Summary.Blockers)
		}
		reportsMustBeClean(t, res, token, homePath)
	})

	t.Run("sources index", func(t *testing.T) {
		repo := t.TempDir()
		token, _ := x46mSpans(t)
		seedResidueStore(t, repo, false)
		plantResidue(t, SourcesIndexPath(repo), map[string]string{`"origin": "notes.md"`: `"origin": "https://example.com/?t=` + token + `"`})
		res, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
		if err != nil {
			t.Fatalf("lint: %v", err)
		}
		// The marker sits on two registry lines (the entry's origin and the
		// consumer citation's origin), so every located finding must name the
		// kind and there must be at least one.
		found := residueFindingsFor(res, SourcesIndexPath(repo))
		if len(found) == 0 {
			t.Fatalf("GHSA-xj89: want MR001 on the sources index, got none: %+v", res.Findings)
		}
		for _, f := range found {
			if f.Line <= 0 || !strings.Contains(f.Message, "github_pat") {
				t.Errorf("GHSA-xj89: MR001 on the sources index must be located and name github_pat, got %+v", f)
			}
		}
		if res.ExitCode != 1 {
			t.Errorf("exit = %d, want 1", res.ExitCode)
		}
		reportsMustBeClean(t, res, token)
	})

	t.Run("kept original", func(t *testing.T) {
		repo := t.TempDir()
		token, _ := x46mSpans(t)
		seeded := seedResidueStore(t, repo, true)
		if seeded.KeptOriginal == "" {
			t.Fatalf("keep-original did not store a copy: %q", seeded.KeepOriginalError)
		}
		kept := filepath.Join(repo, seeded.KeptOriginal)
		plantResidue(t, kept, map[string]string{"MARKERTOKEN": token})
		res, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
		if err != nil {
			t.Fatalf("lint: %v", err)
		}
		found := residueFindingsFor(res, kept)
		if len(found) != 1 || !strings.Contains(found[0].Message, "github_pat") {
			t.Fatalf("GHSA-xj89: want one MR001 on the kept original naming github_pat, got %+v", found)
		}
		reportsMustBeClean(t, res, token)
	})

	t.Run("clean store has no residue finding", func(t *testing.T) {
		repo := t.TempDir()
		x46mSpans(t)
		seedResidueStore(t, repo, true)
		res, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
		if err != nil {
			t.Fatalf("lint: %v", err)
		}
		for _, f := range res.Findings {
			if f.Code == "MR001" {
				t.Errorf("a clean store must not report residue: %+v", f)
			}
		}
	})

	t.Run("degraded scanner is reported, not fatal", func(t *testing.T) {
		repo := t.TempDir()
		x46mSpans(t)
		seedResidueStore(t, repo, false)
		cfg := filepath.Join(repo, ".abcd", "config")
		if err := os.MkdirAll(cfg, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cfg, "pii.json"), []byte("{not json"), 0o644); err != nil {
			t.Fatal(err)
		}
		res, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
		if err != nil {
			t.Fatalf("lint must always crawl and write its report; a degraded scanner is a finding, got error: %v", err)
		}
		found := residueFindingsFor(res, res.StorePath)
		if len(found) != 1 || found[0].Severity != "blocker" || !strings.Contains(found[0].Message, "unavailable") {
			t.Fatalf("GHSA-xj89: want one blocker MR001 on the store path saying the scan is unavailable, got %+v", res.Findings)
		}
		if res.ExitCode != 1 {
			t.Errorf("exit = %d, want 1", res.ExitCode)
		}
		if _, err := os.Stat(filepath.Join(res.ReportDir, "report.json")); err != nil {
			t.Errorf("the report was not written: %v", err)
		}
	})
}
