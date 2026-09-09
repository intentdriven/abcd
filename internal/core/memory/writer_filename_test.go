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

// ---------------------------------------------------------------------------
// The separator-straddling spelling
// ---------------------------------------------------------------------------

// The component pass above judges the three fields pageNameRe parses out of the
// name, and it closes the spelling it was written for — `topic_auth_ghp_<36>`,
// where the whole token sits inside the slug. But THE SPLIT IS ITSELF ON
// UNDERSCORE, and a credential prefix ends in one: `ghp_`, `sk_live_`. A name
// whose separator falls inside the token is therefore invisible to both passes
// at once. `topic_ghp_<36>.md` parses as type `topic`, domain `ghp`, slug
// `<36>`; the joined form has no word boundary before `ghp` because '_' is a
// word character, the domain alone is three letters, and the slug alone carries
// no prefix. Nothing matches, and the token reaches the committed tree as the
// file's own name, in index.md, and as the registry back-link.
//
// The tokens here are the same FAKE fixtures the leaf tests use — a prefix and
// a run of literal 'A's. Nothing here is a live credential.

const (
	a36 = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" // 36 — github_pat's {36,}
	a24 = "AAAAAAAAAAAAAAAAAAAAAAAA"             // 24 — stripe_live's {20,}
)

// componentDistiller returns a distiller whose one page carries the given
// component fields verbatim — each innocuous on its own, and the filename they
// compose to is not.
func componentDistiller(typ, domain, slug string) Distiller {
	return func(_ string, sourceBlock map[string]any) ([]map[string]any, error) {
		return []map[string]any{{
			"type": typ, "domain": domain, "slug": slug,
			"body": "# Token rotation\nRotate tokens every 24 hours.\n", "source": sourceBlock,
		}}, nil
	}
}

func TestWriteRefusesACredentialSplitAcrossTheSeparator(t *testing.T) {
	cases := []struct {
		name              string
		typ, domain, slug string
		token             string
	}{
		{
			// The prefix ends the domain and the body is the whole slug.
			name: "github pat straddling domain and slug",
			typ:  "topic", domain: "ghp", slug: a36,
			token: "ghp_" + a36,
		},
		{
			// `sk_live_` splits the other way: the domain takes `sk` and the
			// slug opens with the rest of the prefix.
			name: "stripe live key straddling domain and slug",
			typ:  "topic", domain: "sk", slug: "live_" + a24,
			token: "sk_live_" + a24,
		},
		{
			// slugRe admits '_', so the token can begin at an underscore INSIDE
			// the slug — a position no pair of parsed components starts at.
			// This is the case that separates scanning every underscore suffix
			// from merely re-joining adjacent components.
			name: "github pat beginning at an underscore inside the slug",
			typ:  "topic", domain: "auth", slug: "x_ghp_" + a36,
			token: "ghp_" + a36,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			t.Setenv("HOME", x46mHome)
			page := tc.typ + "_" + tc.domain + "_" + tc.slug + ".md"
			src := writeSource(t, repo, "notes.md", "Rotate tokens every 24 hours.\n")

			_, err := Ingest(IngestRequest{
				RepoRoot: repo, Source: src,
				Distiller: componentDistiller(tc.typ, tc.domain, tc.slug), Now: fixedNow,
			})
			if err == nil {
				t.Fatalf("a page FILENAME spelling %s across the separator was accepted into the store", tc.token)
			}
			if !strings.Contains(err.Error(), page) {
				t.Errorf("the refusal does not name the refused page %q: %v", page, err)
			}

			mem := Dir(repo)
			if _, statErr := os.Stat(filepath.Join(mem, page)); !os.IsNotExist(statErr) {
				t.Errorf("the page file was written despite the refusal (%v)", statErr)
			}
			mustNotCarry(t, "index.md", filepath.Join(mem, "index.md"), tc.token)
			mustNotCarry(t, "log.md", filepath.Join(mem, "log.md"), tc.token)
			mustNotCarry(t, "sources registry", SourcesIndexPath(repo), tc.token)
		})
	}
}

// TestShortDomainsAreNotCredentials is the anti-vacuity guard for the split
// rule, and it is load-bearing. `ghp` is three ordinary letters and `sk` is
// two; a rule that refused on the PREFIX rather than on the whole credential
// shape would refuse every one of these ordinary pages. The rule keys on the
// scanner's own patterns, which carry a length floor ({36,} for a github PAT,
// {20,} for a stripe key), so a short real word in the domain position writes
// exactly as it did before.
func TestShortDomainsAreNotCredentials(t *testing.T) {
	for _, tc := range []struct{ domain, slug string }{
		{"api", "rate-limits"},
		{"git", "rebasing"},
		{"sk", "notes"},
		{"ghp", "reference"},       // the bypass prefix itself, as an honest domain
		{"auth", "x_ghp_rotation"}, // an underscore-bearing slug that is not a token
	} {
		t.Run(tc.domain+"_"+tc.slug, func(t *testing.T) {
			repo := t.TempDir()
			t.Setenv("HOME", x46mHome)
			page := "topic_" + tc.domain + "_" + tc.slug + ".md"
			src := writeSource(t, repo, "notes.md", "Rotate tokens every 24 hours.\n")

			res, err := Ingest(IngestRequest{
				RepoRoot: repo, Source: src,
				Distiller: componentDistiller("topic", tc.domain, tc.slug), Now: fixedNow,
			})
			if err != nil {
				t.Fatalf("an ordinary page named %s was refused: %v", page, err)
			}
			if len(res.Pages) != 1 || res.Pages[0] != page {
				t.Fatalf("the ingest wrote %v, want [%s]", res.Pages, page)
			}
			if _, serr := os.Stat(filepath.Join(Dir(repo), page)); serr != nil {
				t.Fatalf("the page was not written under its own name: %v", serr)
			}
			if links := registryBackLinks(t, repo, res.ContentHash); len(links) != 1 || links[0] != page {
				t.Errorf("the registry back-link %v does not name %q", links, page)
			}
		})
	}
}
