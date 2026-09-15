package memory

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ingest_credential_test.go — a credential carried by the ingested URL must
// never reach the committed store, the returned result, or an error message.
//
// materialFromFetched used the fetched final URL verbatim as the page origin
// AND title, so `https://user:pass@host/doc` wrote the password into the page
// frontmatter's source.citation, into .sources_index.json (origin and
// consumers.memory.citation) and into IngestResult.Citation, which `ingest
// --json` prints. The write-time redactor cannot save it: an opaque basic-auth
// password matches no pattern. The six `fetch failed for %s` sites, the
// byte-cap refusal and the redirect-count refusal echoed the raw source the
// same way. A credential-shaped query key (`?token=…`) is the same leak
// through the other half of the URL.
//
// The password below is a FAKE fixture, not a live credential.

const fakeURLPassword = "s3cr3tPassw0rd"

// fakeQueryToken is the other half of the class: a bearer credential carried as
// a query value. Also FAKE.
const fakeQueryToken = "SUPERSECRETqueryToken"

// credentialPageDistiller copies the core's source block into the page, so the
// stored frontmatter carries the citation the core built from the origin.
func credentialPageDistiller(_ string, sourceBlock map[string]any) ([]map[string]any, error) {
	return []map[string]any{{
		"type": "topic", "domain": "auth", "slug": "tokens",
		"body": "# Token rotation\nRotate tokens every 24 hours.\n", "source": sourceBlock,
	}}, nil
}

func fetcherReturning(final string) Fetcher {
	return func(string) (FetchedSource, error) {
		return FetchedSource{
			FinalURL: final,
			Headers:  map[string]string{"Content-Type": "text/plain"},
			Body:     []byte("Rotate tokens every 24 hours.\n"),
		}, nil
	}
}

// storeMustNotContain walks every byte the ingest made durable — every file
// under .abcd/memory, the sources registry included — and fails on a span.
func storeMustNotContain(t *testing.T, repo string, spans ...string) {
	t.Helper()
	roots := []string{Dir(repo), SourcesIndexPath(repo)}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			raw, rerr := os.ReadFile(path)
			if rerr != nil {
				return nil
			}
			for _, s := range spans {
				if bytes.Contains(raw, []byte(s)) {
					t.Errorf("the store file %s carries the URL credential %q:\n%s",
						filepath.Base(path), s, raw)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}

func TestIngestStripsURLCredential(t *testing.T) {
	t.Run("basic-auth userinfo never reaches the store or the result", func(t *testing.T) {
		repo := t.TempDir()
		source := "https://alice:" + fakeURLPassword + "@example.com/doc"
		res, err := Ingest(IngestRequest{
			RepoRoot:  repo,
			Source:    source,
			Distiller: credentialPageDistiller,
			Fetcher:   fetcherReturning(source),
			Now:       fixedNow,
		})
		if err != nil {
			t.Fatalf("ingest: %v", err)
		}
		storeMustNotContain(t, repo, fakeURLPassword)

		encoded, merr := json.Marshal(res)
		if merr != nil {
			t.Fatalf("marshal result: %v", merr)
		}
		if bytes.Contains(encoded, []byte(fakeURLPassword)) {
			t.Errorf("ingest --json returns the URL credential:\n%s", encoded)
		}
		// The legitimate address survives: only the credential is dropped.
		origin, _ := res.Citation["origin"].(string)
		if !strings.Contains(origin, "example.com/doc") {
			t.Fatalf("stripping the credential must not drop the address: %q", origin)
		}
	})

	t.Run("a credential-shaped query key is dropped from the stored origin", func(t *testing.T) {
		repo := t.TempDir()
		final := "https://example.com/doc?q=keep&token=" + fakeURLPassword + "&Api_Key=" + fakeURLPassword
		res, err := Ingest(IngestRequest{
			RepoRoot:  repo,
			Source:    "https://example.com/doc",
			Distiller: credentialPageDistiller,
			Fetcher:   fetcherReturning(final),
			Now:       fixedNow,
		})
		if err != nil {
			t.Fatalf("ingest: %v", err)
		}
		storeMustNotContain(t, repo, fakeURLPassword)
		origin, _ := res.Citation["origin"].(string)
		if strings.Contains(strings.ToLower(origin), "token=") || strings.Contains(strings.ToLower(origin), "api_key=") {
			t.Errorf("the stored origin keeps a credential-shaped query key: %q", origin)
		}
		if !strings.Contains(origin, "q=keep") {
			t.Errorf("the innocent query parameter was dropped too: %q", origin)
		}
	})

	t.Run("a failing fetch does not echo the credential", func(t *testing.T) {
		repo := t.TempDir()
		source := "https://alice:" + fakeURLPassword + "@example.com/doc"
		_, err := Ingest(IngestRequest{
			RepoRoot:  repo,
			Source:    source,
			Distiller: credentialPageDistiller,
			Fetcher:   func(string) (FetchedSource, error) { return FetchedSource{}, errors.New("connection reset") },
			Now:       fixedNow,
		})
		if err == nil {
			t.Fatalf("a failing fetch must be an error")
		}
		if strings.Contains(err.Error(), fakeURLPassword) {
			t.Errorf("the fetch failure echoes the URL credential: %v", err)
		}
	})

	t.Run("an addressing query key is not mistaken for a credential", func(t *testing.T) {
		repo := t.TempDir()
		// `key` and `signature` name a credential only sometimes; they name a
		// document section, a sort key or a content signature at least as
		// often, and dropping them truncates a legitimate address the citation
		// is supposed to reproduce.
		final := "https://example.com/doc?key=section3&signature=v2&token=" + fakeQueryToken
		res, err := Ingest(IngestRequest{
			RepoRoot:  repo,
			Source:    "https://example.com/doc",
			Distiller: credentialPageDistiller,
			Fetcher:   fetcherReturning(final),
			Now:       fixedNow,
		})
		if err != nil {
			t.Fatalf("ingest: %v", err)
		}
		origin, _ := res.Citation["origin"].(string)
		if !strings.Contains(origin, "key=section3") {
			t.Errorf("an addressing ?key= parameter was dropped from the stored origin: %q", origin)
		}
		if !strings.Contains(origin, "signature=v2") {
			t.Errorf("an addressing ?signature= parameter was dropped from the stored origin: %q", origin)
		}
		if strings.Contains(origin, fakeQueryToken) {
			t.Errorf("the stored origin keeps the bearer credential: %q", origin)
		}
	})

	t.Run("a transport error does not echo the credential it re-prints", func(t *testing.T) {
		repo := t.TempDir()
		source := "https://alice:" + fakeURLPassword + "@example.com/doc?token=" + fakeQueryToken
		_, err := Ingest(IngestRequest{
			RepoRoot:  repo,
			Source:    source,
			Distiller: credentialPageDistiller,
			// The shape net/http actually returns from client.Do: a *url.Error
			// whose message re-prints the whole request URL. It masks the
			// basic-auth password and nothing else, so the query credential
			// rides along inside the %v of the caller's own message.
			Fetcher: func(raw string) (FetchedSource, error) {
				return FetchedSource{Body: nil}, &url.Error{
					Op:  "Get",
					URL: raw,
					Err: errors.New("connection reset"),
				}
			},
			Now: fixedNow,
		})
		if err == nil {
			t.Fatalf("a failing fetch must be an error")
		}
		if strings.Contains(err.Error(), fakeQueryToken) {
			t.Errorf("the transport error echoes the URL query credential: %v", err)
		}
		if strings.Contains(err.Error(), fakeURLPassword) {
			t.Errorf("the transport error echoes the URL password: %v", err)
		}
	})

	t.Run("the redirect refusal does not echo the credential", func(t *testing.T) {
		policy := ingestRedirectPolicy("https://example.com/a")
		hop, perr := url.Parse("http://alice:" + fakeURLPassword + "@example.com/x?token=" + fakeQueryToken)
		if perr != nil {
			t.Fatalf("parse hop: %v", perr)
		}
		err := policy(&http.Request{URL: hop}, []*http.Request{{URL: hop}})
		if err == nil {
			t.Fatal("a redirect off https must be refused")
		}
		if strings.Contains(err.Error(), fakeQueryToken) {
			t.Errorf("the redirect refusal echoes the hop's query credential: %v", err)
		}
		if strings.Contains(err.Error(), fakeURLPassword) {
			t.Errorf("the redirect refusal echoes the hop's password: %v", err)
		}
	})

	t.Run("the plaintext refusal at the connection does not echo the credential", func(t *testing.T) {
		_, err := defaultFetch("http://alice:" + fakeURLPassword + "@example.com/x?token=" + fakeQueryToken)
		if err == nil {
			t.Fatal("a plaintext fetch must be refused")
		}
		if strings.Contains(err.Error(), fakeQueryToken) {
			t.Errorf("the plaintext refusal echoes the query credential: %v", err)
		}
		if strings.Contains(err.Error(), fakeURLPassword) {
			t.Errorf("the plaintext refusal echoes the password: %v", err)
		}
	})

	t.Run("the byte-cap refusal does not echo the credential", func(t *testing.T) {
		repo := t.TempDir()
		source := "https://alice:" + fakeURLPassword + "@example.com/doc"
		_, err := Ingest(IngestRequest{
			RepoRoot:  repo,
			Source:    source,
			Distiller: credentialPageDistiller,
			Fetcher: func(string) (FetchedSource, error) {
				return FetchedSource{
					FinalURL: source,
					Headers:  map[string]string{"Content-Type": "text/plain"},
					Body:     bytes.Repeat([]byte("a"), maxFetchBytes+1),
				}, nil
			},
			Now: fixedNow,
		})
		if err == nil {
			t.Fatalf("an over-cap body must be refused")
		}
		if strings.Contains(err.Error(), fakeURLPassword) {
			t.Errorf("the byte-cap refusal echoes the URL credential: %v", err)
		}
	})
}
