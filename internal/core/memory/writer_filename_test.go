package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writer_filename_test.go — iss-2609020321100138. The store redactor judged
// page BODIES, page FRONTMATTER leaves and the registry leaves a write
// introduces, but never the page FILENAME. A host distiller controls the slug,
// slugRe admits [A-Za-z0-9_-] and pageNameRe admits <type>_<domain>_<slug>.md,
// so a distiller returning the slug `ghp_<40 chars>` wrote the token into the
// committed tree four times over: as the file's own name, in index.md, in
// log.md, and as the registry back-link the pruneOrphans data-loss fix
// deliberately excludes from the leaf walk.
//
// The filename is judged at the write boundary and REFUSED, never rewritten —
// renaming a page is not a redaction, it is a different page, and the registry
// back-link that names it would then point at nothing.
//
// The bar is NARROWER than the leaf walk's. redactText and judgeKey refuse on
// scanner.BlockingResidual, which treats any identity-or-network span as
// blocking whatever its severity — and `migrating-off-the-nas` is an ordinary
// English slug that net_device_hostname matches at warn severity. A filename
// rule at that bar would refuse ordinary pages, so it runs on hard_fail
// findings only. The second subtest is what holds that line.
//
// The token is the same FAKE fixture the leaf tests use: `ghp_` and forty
// literal 'A's. Nothing here is a live credential.

// secretSlugPage is the page the refused write would have written.
func secretSlugPage(token string) string { return "topic_auth_" + token + ".md" }

// secretSlugDistiller returns a distiller whose one page carries token as its
// slug — the host-controlled field that reaches the filename verbatim.
func secretSlugDistiller(token string) Distiller {
	return func(_ string, sourceBlock map[string]any) ([]map[string]any, error) {
		return []map[string]any{{
			"type": "topic", "domain": "auth", "slug": token,
			"body": "# Token rotation\nRotate tokens every 24 hours.\n", "source": sourceBlock,
		}}, nil
	}
}

// mustNotCarry asserts that path either does not exist or does not hold span.
// A refused write never reaches the store lock, so the derived surfaces are
// commonly absent rather than clean; both are a pass, and the file being
// present with the token in it is the failure.
func mustNotCarry(t *testing.T, label, path, span string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("%s (%s): %v", label, path, err)
	}
	if strings.Contains(string(raw), span) {
		t.Errorf("%s carries the refused token:\n%s", label, raw)
	}
}

func TestWriteRefusesASecretShapedFilename(t *testing.T) {
	repo := t.TempDir()
	token, _ := x46mSpans(t)
	page := secretSlugPage(token)
	src := writeSource(t, repo, "notes.md", "Rotate tokens every 24 hours.\n")

	_, err := Ingest(IngestRequest{
		RepoRoot: repo, Source: src, Distiller: secretSlugDistiller(token), Now: fixedNow,
	})
	if err == nil {
		t.Fatalf("a token-shaped page FILENAME was accepted into the store")
	}
	// Unlike judgeKey, the refusal names the page: the filename is the write's
	// identity, and a batch refusal that withheld it would leave the operator
	// with no way to say which page to repair.
	if !strings.Contains(err.Error(), page) {
		t.Errorf("the refusal does not name the refused page %q: %v", page, err)
	}

	mem := Dir(repo)
	if _, statErr := os.Stat(filepath.Join(mem, page)); !os.IsNotExist(statErr) {
		t.Errorf("the page file was written despite the refusal (%v)", statErr)
	}
	// The other three places the slug lands.
	mustNotCarry(t, "index.md", filepath.Join(mem, "index.md"), token)
	mustNotCarry(t, "log.md", filepath.Join(mem, "log.md"), token)
	mustNotCarry(t, "sources registry", SourcesIndexPath(repo), token)
}

// TestFilenameBarIsHardFailOnly is the anti-vacuity guard. An implementation
// that reused scanner.BlockingResidual — the bar every other write-side rule
// holds — would refuse this ordinary page, because net_device_hostname matches
// `off-the-nas` at warn severity and BlockingResidual promotes any network span
// to blocking. If this test passes against that naive implementation it is not
// doing its job.
func TestFilenameBarIsHardFailOnly(t *testing.T) {
	repo := t.TempDir()
	src := writeSource(t, repo, "storage.md", "The array moved to the cloud.\n")

	res, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: nasDistiller, Now: fixedNow})
	if err != nil {
		t.Fatalf("an ordinary slug a warn-severity network pattern matches was refused: %v", err)
	}
	if len(res.Pages) != 1 || res.Pages[0] != nasPage {
		t.Fatalf("the ingest wrote %v, want [%s]", res.Pages, nasPage)
	}
	if _, serr := os.Stat(filepath.Join(Dir(repo), nasPage)); serr != nil {
		t.Fatalf("the page was not written under its own name: %v", serr)
	}
	links := registryBackLinks(t, repo, res.ContentHash)
	if len(links) != 1 || links[0] != nasPage {
		t.Errorf("the registry back-link %v does not name the written page %q", links, nasPage)
	}
}

// TestOrdinaryFilenamesStillWrite pins the third case: a plain slug is written,
// named in index.md and log.md, and back-linked, exactly as before.
func TestOrdinaryFilenamesStillWrite(t *testing.T) {
	repo := t.TempDir()
	src := writeSource(t, repo, "notes.md", "Rotate tokens every 24 hours.\n")

	res, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: fixedBodyDistiller(nil), Now: fixedNow})
	if err != nil {
		t.Fatalf("an ordinary page was refused: %v", err)
	}
	const page = "topic_auth_tokens.md"
	if len(res.Pages) != 1 || res.Pages[0] != page {
		t.Fatalf("the ingest wrote %v, want [%s]", res.Pages, page)
	}
	mem := Dir(repo)
	if _, serr := os.Stat(filepath.Join(mem, page)); serr != nil {
		t.Fatalf("the page was not written: %v", serr)
	}
	for _, sibling := range []string{"index.md", "log.md"} {
		raw, rerr := os.ReadFile(filepath.Join(mem, sibling))
		if rerr != nil {
			t.Fatalf("read %s: %v", sibling, rerr)
		}
		if !strings.Contains(string(raw), "tokens") {
			t.Errorf("%s does not name the written page:\n%s", sibling, raw)
		}
	}
	if links := registryBackLinks(t, repo, res.ContentHash); len(links) != 1 || links[0] != page {
		t.Errorf("the registry back-link %v does not name %q", links, page)
	}
}
