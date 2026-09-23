package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lint_page_name_test.go — iss-2609090642035097. The write boundary refuses a
// page whose FILENAME carries a hard-fail secret (iss-2609020321100138), but a
// store written before that refusal existed is not repaired, and the read-side
// MR001 lint could not report it: it scanned stored text and the registry bytes,
// and a token embedded in a page name such as `topic_auth_ghp_…` has no word
// boundary after the `_`-joined type and domain prefix, so the anchored pattern
// never matched. The legacy dirty name then sat in the tree, in index.md, in
// log.md and in the registry back-link with no instrument reporting it.
//
// The fixture is a real store seeded through Ingest whose page file and registry
// back-link are then renamed by hand to the dirty name — the legacy shape the
// write side can no longer produce. The token is the same FAKE `ghp_` + forty
// 'A's the filename tests use. Nothing here is a live credential.

// renamePageInStore moves the seeded page to name and rewrites its registry
// back-link to match, returning the new page path.
func renamePageInStore(t *testing.T, repo, from, name string) string {
	t.Helper()
	mem := Dir(repo)
	to := filepath.Join(mem, name)
	if err := os.Rename(filepath.Join(mem, from), to); err != nil {
		t.Fatal(err)
	}
	plantResidue(t, SourcesIndexPath(repo), map[string]string{`"` + from + `"`: `"` + name + `"`})
	return to
}

func pageNameMR001(res LintResult) []Finding {
	var out []Finding
	for _, f := range res.Findings {
		if f.Code == "MR001" && strings.Contains(f.Message, "page name") {
			out = append(out, f)
		}
	}
	return out
}

func TestLintReportsASecretEmbeddedInAPageName(t *testing.T) {
	repo := t.TempDir()
	token, _ := x46mSpans(t)
	seedResidueStore(t, repo, false)
	page := renamePageInStore(t, repo, "topic_auth_tokens.md", secretSlugPage(token))

	res, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	var onPage, onRegistry bool
	for _, f := range pageNameMR001(res) {
		if f.Severity != "blocker" {
			t.Errorf("MR001 must be a blocker, got %q", f.Severity)
		}
		if !strings.Contains(f.Message, "github_pat") {
			t.Errorf("MR001 must name the kind, got %q", f.Message)
		}
		// The finding locates the page by its path, as the write-side refusal
		// names the page: the operator cannot repair a page nobody names. The
		// message and the suggestion carry the kind, never the span.
		if strings.Contains(f.Message, token) || strings.Contains(f.Suggestion, token) {
			t.Errorf("MR001 message carries the raw span: %q", f.Message)
		}
		switch f.File {
		case page:
			onPage = true
		case SourcesIndexPath(repo):
			onRegistry = true
			if f.Line <= 0 {
				t.Errorf("the registry back-link finding must locate its line, got %d", f.Line)
			}
		}
	}
	if !onPage || !onRegistry {
		t.Fatalf("iss-2609090642035097: a token embedded in a page name went unreported (page=%v registry back-link=%v):\n%+v", onPage, onRegistry, res.Findings)
	}
	if res.ExitCode != 1 {
		t.Errorf("exit = %d, want 1", res.ExitCode)
	}
}

// TestLintPageNameBarIsHardFailOnly is the read side of TestFilenameBarIsHardFailOnly:
// an ordinary name that net_device_hostname matches at warn severity must stay
// clean of the page-name rule, or every such page in an ordinary store would be
// a blocker.
func TestLintPageNameBarIsHardFailOnly(t *testing.T) {
	repo := t.TempDir()
	seedResidueStore(t, repo, false)
	renamePageInStore(t, repo, "topic_auth_tokens.md", "topic_home_migrating-off-the-nas.md")

	res, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	for _, f := range pageNameMR001(res) {
		t.Errorf("an ordinary page name drew MR001: %+v", f)
	}
}

// The registry back-link is judged even when it names no page on disk: an
// orphan back-link to a dirty name is the one place the name survives once its
// page is removed.
func TestLintReportsASecretInAnOrphanBackLink(t *testing.T) {
	repo := t.TempDir()
	token, _ := x46mSpans(t)
	seedResidueStore(t, repo, false)
	page := renamePageInStore(t, repo, "topic_auth_tokens.md", secretSlugPage(token))
	if err := os.Remove(page); err != nil {
		t.Fatal(err)
	}
	res, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	var onRegistry bool
	for _, f := range pageNameMR001(res) {
		if f.File == SourcesIndexPath(repo) {
			onRegistry = true
		}
	}
	if !onRegistry {
		t.Fatalf("an orphan registry back-link carrying a token went unreported:\n%+v", res.Findings)
	}
	// Sanity: the fixture really is an orphan back-link, not a stray span in text.
	raw, _ := os.ReadFile(SourcesIndexPath(repo))
	var reg map[string]any
	if json.Unmarshal(raw, &reg) != nil || !strings.Contains(string(raw), `"`+secretSlugPage(token)+`"`) {
		t.Fatalf("fixture drift: the registry does not carry the back-link")
	}
}

// TestLintOrdinaryBackLinkIsNotHostnameResidue — iss-2609230851376499. The
// registry's free-text scan held every page back-link to the BlockingResidual
// bar, so the ordinary name `topic_home_migrating-off-the-nas.md` matched
// net_device_hostname at warn on the hyphen boundary and an ordinary store,
// written by the ingest path without complaint, linted as a blocker. A back-link
// is judged by the page-name rule alone, as the write side's leaf walk excludes
// it; the rest of the registry is still scanned as text.
func TestLintOrdinaryBackLinkIsNotHostnameResidue(t *testing.T) {
	repo := t.TempDir()
	src := writeSource(t, repo, "notes.md", "Plan the move.\n")
	if _, err := Ingest(IngestRequest{
		RepoRoot: repo, Source: src, Now: fixedNow,
		Distiller: oneTopicDistiller("topic", "home", "migrating-off-the-nas", "# Move\nPlan the move.\n"),
	}); err != nil {
		t.Fatalf("an ordinary page must write: %v", err)
	}
	res, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	for _, f := range res.Findings {
		if f.Code == "MR001" {
			t.Errorf("iss-2609230851376499: an ordinary store drew MR001: %+v", f)
		}
	}
	if res.ExitCode != 0 {
		t.Errorf("exit = %d, want 0 for an ordinary store", res.ExitCode)
	}
}

// The mask covers only quoted strings byte-equal to a well-formed back-link,
// wherever they sit in the registry; the only findings it can hide are
// warn-level identity and network kinds on bytes the write side accepts as a
// page name, because pageNameResidue still reports a hard-fail span on the
// identical name. A free-text registry value that differs from every back-link,
// as the span planted here does, is still residue.
func TestLintStillScansTheRegistryTextBesideTheBackLinks(t *testing.T) {
	repo := t.TempDir()
	token, _ := x46mSpans(t)
	seedResidueStore(t, repo, false)
	plantResidue(t, SourcesIndexPath(repo), map[string]string{`"origin": "notes.md"`: `"origin": "https://example.com/?t=` + token + `"`})
	res, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	var text int
	for _, f := range residueFindingsFor(res, SourcesIndexPath(repo)) {
		if strings.Contains(f.Message, "stored text") {
			text++
		}
	}
	if text == 0 {
		t.Fatalf("the registry's free text was not scanned beside the back-links: %+v", res.Findings)
	}
}
