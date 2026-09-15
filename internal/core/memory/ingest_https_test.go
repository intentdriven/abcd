package memory

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// ingest_https_test.go — GHSA-35fj-9w6f-7h62 (CWE-319, CWE-311). Memory ingest
// admitted a plaintext http:// source and followed a redirect off https, so a
// man-in-the-middle could rewrite the fetched text and, with it, the SPDX line
// or the License: header the store copies verbatim into its durable provenance.
// `abcd update` already pins the scheme at admission and per hop; the memory
// fetcher now holds the same posture. No test here touches the network: the
// admission refusal precedes the fetcher seam, and the redirect policy is a
// pure function called with synthetic requests (a local httptest server cannot
// stand in — urlguard.DialControl refuses loopback by design).

func TestIngestRefusesPlaintextHTTP(t *testing.T) {
	// A fetcher that records being reached. The refusal must land BEFORE
	// acquisition, so the seam is gated in the core and not merely inside
	// defaultFetch, which a caller-supplied Fetcher replaces.
	newFetcher := func(called *bool) Fetcher {
		return func(string) (FetchedSource, error) {
			*called = true
			return FetchedSource{
				FinalURL: "https://example.com/a",
				Headers:  map[string]string{"Content-Type": "text/plain"},
				Body:     []byte("Rotate tokens every 24 hours.\n"),
			}, nil
		}
	}
	distiller := oneTopicDistiller("topic", "auth", "tokens", "# Tokens\nRotate tokens.\n")

	t.Run("plaintext http is refused by name", func(t *testing.T) {
		called := false
		_, err := Ingest(IngestRequest{
			RepoRoot: t.TempDir(), Source: "http://example.com/a", Now: fixedNow,
			Fetcher: newFetcher(&called), Distiller: distiller,
		})
		if err == nil {
			t.Fatal("GHSA-35fj: a plaintext http:// source was ingested; expected a scheme refusal")
		}
		var ie *IngestError
		if !errors.As(err, &ie) {
			t.Errorf("refusal is %T, want *IngestError so every ingest failure is one type: %v", err, err)
		}
		msg := err.Error()
		if !strings.Contains(msg, "https") || !strings.Contains(msg, "http") {
			t.Errorf("the refusal must name the scheme it wants and the scheme it got: %q", msg)
		}
		if called {
			t.Error("GHSA-35fj: the fetcher was reached; the refusal must precede acquisition")
		}
	})

	t.Run("the refusal does not echo userinfo", func(t *testing.T) {
		called := false
		_, err := Ingest(IngestRequest{
			RepoRoot: t.TempDir(), Source: "http://someone:hunter2@example.com/a", Now: fixedNow,
			Fetcher: newFetcher(&called), Distiller: distiller,
		})
		if err == nil {
			t.Fatal("GHSA-35fj: expected a scheme refusal")
		}
		if strings.Contains(err.Error(), "hunter2") {
			t.Errorf("the refusal echoes the source's password back to the terminal: %q", err.Error())
		}
	})

	t.Run("https is still admitted", func(t *testing.T) {
		called := false
		res, err := Ingest(IngestRequest{
			RepoRoot: t.TempDir(), Source: "https://example.com/a", Now: fixedNow,
			Fetcher: newFetcher(&called), Distiller: distiller,
		})
		if err != nil {
			t.Fatalf("an https source must still ingest: %v", err)
		}
		if !called || res.Status != "ingested" {
			t.Errorf("fetcher reached = %v, status = %q; want the https source fetched and ingested", called, res.Status)
		}
	})

	t.Run("a local path is still a local path", func(t *testing.T) {
		repo := t.TempDir()
		src := writeSource(t, repo, "notes.md", "Rotate tokens every 24 hours.\n")
		called := false
		res, err := Ingest(IngestRequest{
			RepoRoot: repo, Source: src, Now: fixedNow,
			Fetcher: newFetcher(&called), Distiller: distiller,
		})
		if err != nil {
			t.Fatalf("a local source must still ingest: %v", err)
		}
		if called {
			t.Error("a local path must not reach the fetcher")
		}
		if res.Status != "ingested" {
			t.Errorf("status = %q, want ingested", res.Status)
		}
	})
}

func TestIngestRedirectMustStayHTTPS(t *testing.T) {
	policy := ingestRedirectPolicy("https://example.com/a")
	req := func(raw string) *http.Request {
		t.Helper()
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		return &http.Request{URL: u}
	}
	via := []*http.Request{req("https://example.com/a")}

	t.Run("a hop off https is refused", func(t *testing.T) {
		err := policy(req("http://example.com/x"), via)
		if err == nil {
			t.Fatal("GHSA-35fj: a redirect from https to plaintext http was followed")
		}
		if !strings.Contains(err.Error(), "https") {
			t.Errorf("the refusal must say the redirect left https: %q", err.Error())
		}
	})

	t.Run("the refusal does not echo userinfo", func(t *testing.T) {
		err := policy(req("http://someone:hunter2@example.com/x"), via)
		if err == nil {
			t.Fatal("GHSA-35fj: expected a scheme refusal")
		}
		if strings.Contains(err.Error(), "hunter2") {
			t.Errorf("the refusal echoes the redirect target's password: %q", err.Error())
		}
	})

	t.Run("an https hop is followed", func(t *testing.T) {
		// A public IP literal (TEST-NET-3, RFC 5737) so the host guard answers
		// without a DNS lookup and the case stays offline.
		if err := policy(req("https://203.0.113.10/x"), via); err != nil {
			t.Errorf("an https redirect to a public host must be followed: %v", err)
		}
	})

	t.Run("the hop budget still binds", func(t *testing.T) {
		long := make([]*http.Request, maxRedirects)
		for i := range long {
			long[i] = req("https://example.com/a")
		}
		err := policy(req("https://example.com/x"), long)
		if err == nil || !strings.Contains(err.Error(), "too many redirects") {
			t.Errorf("the redirect budget must still bind: %v", err)
		}
	})

	t.Run("the SSRF host guard still binds", func(t *testing.T) {
		err := policy(req("https://169.254.169.254/latest/meta-data/"), via)
		if err == nil {
			t.Fatal("a redirect to the cloud-metadata endpoint must be refused")
		}
		if !strings.Contains(err.Error(), "refusing to") && !strings.Contains(err.Error(), "cannot resolve host") {
			t.Errorf("want the SSRF-guard refusal, got: %v", err)
		}
	})
}
