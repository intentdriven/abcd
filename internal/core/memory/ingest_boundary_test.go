package memory

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ingest_boundary_test.go — iss-30 remainder: the ingest input boundary that was
// never exercised. Every case here drives one of the two injectable seams
// (Fetcher, PDFExtractor) or the material builders directly, so no test needs a
// network, a real PDF parser, or any dependency outside the stdlib. Hosts are
// RFC 2606 reserved documentation names.

const docURL = "https://docs.example.test/rotation"

// staticFetcher is the injected Fetcher seam: it records the URL it was asked
// for and returns a canned response. Nothing dials.
func staticFetcher(seen *string, fs FetchedSource) Fetcher {
	return func(u string) (FetchedSource, error) {
		if seen != nil {
			*seen = u
		}
		return fs, nil
	}
}

// textFetched is the common success-shape response body.
func textFetched(final, ctype, body string) FetchedSource {
	return FetchedSource{
		FinalURL: final,
		Headers:  map[string]string{"Content-Type": ctype},
		Body:     []byte(body),
	}
}

// ---------------------------------------------------------------------------
// URL-ingest success path (iss-30 instance 5)
// ---------------------------------------------------------------------------

// TestIngestFromURLSuccessPath drives a whole Ingest over the URL branch through
// the injected Fetcher. It pins the four things the URL path contributes that the
// local path cannot: the fetcher is asked for the REQUESTED url; the FINAL url
// (after redirects) is what lands in the citation and origin; the response
// headers reach licence detection (a `License:` header, which the local path has
// no equivalent of); and --keep-original derives its extension from the final
// URL's path.
func TestIngestFromURLSuccessPath(t *testing.T) {
	repo := t.TempDir()
	const final = "https://docs.example.test/rotation/index.html"
	body := "Token rotation policy: rotate tokens every 24 hours."

	var asked string
	fetched := textFetched(final, "text/html; charset=utf-8", body)
	fetched.Headers["License"] = "MIT"

	res, err := Ingest(IngestRequest{
		RepoRoot:     repo,
		Source:       docURL,
		Fetcher:      staticFetcher(&asked, fetched),
		KeepOriginal: true,
		Distiller:    oneTopicDistiller("topic", "auth", "tokens", "# Token rotation\nRotate tokens every 24 hours."),
		Now:          fixedNow,
	})
	if err != nil {
		t.Fatalf("URL ingest: %v", err)
	}
	if asked != docURL {
		t.Fatalf("fetcher asked for %q, want the requested URL %q", asked, docURL)
	}
	if res.Status != "ingested" {
		t.Fatalf("status = %q, want ingested", res.Status)
	}
	if res.ContentHash != sourceContentHash(body) {
		t.Fatalf("content hash = %q, want the hash of the fetched body", res.ContentHash)
	}
	// The HTTP License: header is the ONLY licence signal here (the body carries
	// no SPDX tag), so "unknown" would prove the headers never reached
	// detectLicence.
	if res.Licence != "MIT" {
		t.Fatalf("licence = %q, want MIT from the HTTP License: header", res.Licence)
	}
	if got := res.Citation["origin"]; got != final {
		t.Fatalf("citation origin = %v, want the FINAL url %q (redirect target, not the requested one)", got, final)
	}
	if got := res.Citation["title"]; got != final {
		t.Fatalf("citation title = %v, want %q", got, final)
	}
	if len(res.Pages) != 1 || res.Pages[0] != "topic_auth_tokens.md" {
		t.Fatalf("pages = %v", res.Pages)
	}
	if _, err := os.Stat(filepath.Join(Dir(repo), "topic_auth_tokens.md")); err != nil {
		t.Fatalf("page not written: %v", err)
	}
	reg, err := LoadRegistry(SourcesIndexPath(repo))
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	entry, ok := reg[res.ContentHash].(map[string]any)
	if !ok {
		t.Fatalf("registry has no entry for the fetched source")
	}
	if entry["origin"] != final {
		t.Fatalf("registry origin = %v, want %q", entry["origin"], final)
	}
	consumer := entry["consumers"].(map[string]any)["memory"].(map[string]any)
	if consumer["class"] != "external_article" {
		t.Fatalf("registry class = %v, want external_article", consumer["class"])
	}
	// --keep-original stores the fetched bytes under the extension taken from the
	// final URL's path (.html), not a blanket .txt.
	if !strings.HasSuffix(res.KeptOriginal, ".html") {
		t.Fatalf("kept original = %q, want the .html extension from the final URL path", res.KeptOriginal)
	}
	got, err := os.ReadFile(filepath.Join(repo, res.KeptOriginal))
	if err != nil {
		t.Fatalf("kept original not readable: %v", err)
	}
	if string(got) != body {
		t.Fatalf("kept original bytes = %q, want the fetched body", got)
	}
}

// TestAcquireSourceFetcherErrors proves the two fetch-failure shapes are kept
// distinct: an IngestError raised by the fetcher (the SSRF guard's own refusal)
// is surfaced verbatim, while any other error is wrapped with the URL so the
// operator can see which source failed.
func TestAcquireSourceFetcherErrors(t *testing.T) {
	guard := func(string) (FetchedSource, error) {
		return FetchedSource{}, newIngestError("refusing to fetch a blocked host")
	}
	_, err := acquireSource("", docURL, guard, nil)
	if err == nil || err.Error() != "refusing to fetch a blocked host" {
		t.Fatalf("an IngestError from the fetcher must pass through verbatim, got: %v", err)
	}
	var ie *IngestError
	if !errors.As(err, &ie) {
		t.Fatalf("fetch refusal is %T, want *IngestError", err)
	}

	transport := func(string) (FetchedSource, error) {
		return FetchedSource{}, errors.New("connection reset")
	}
	_, err = acquireSource("", docURL, transport, nil)
	if err == nil {
		t.Fatal("a transport error must fail the ingest")
	}
	if !strings.Contains(err.Error(), "fetch failed for "+docURL) || !strings.Contains(err.Error(), "connection reset") {
		t.Fatalf("transport error must be wrapped with the URL and the cause, got: %v", err)
	}
	if !errors.As(err, &ie) {
		t.Fatalf("wrapped fetch failure is %T, want *IngestError", err)
	}
}

// ---------------------------------------------------------------------------
// Content-type matrix (iss-30 instance 5)
// ---------------------------------------------------------------------------

// TestMaterialFromFetchedContentTypeMatrix walks every branch of the fetched
// content-type decision: the text/* prefix, the explicit non-text allowlist, the
// PDF branch, and the rejection that catches everything else. A near-miss type
// (application/jsonl) must NOT be admitted by the allowlist, and a lying
// text/plain header must still be rejected by the byte-level decode.
func TestMaterialFromFetchedContentTypeMatrix(t *testing.T) {
	const text = "Rotate tokens every 24 hours."
	cases := []struct {
		name      string
		ctype     string
		body      string
		wantClass string
		wantErr   string // substring; empty means the fetch must be accepted
	}{
		{name: "text_plain", ctype: "text/plain", body: text, wantClass: "external_article"},
		{name: "text_html_with_charset", ctype: "text/html; charset=utf-8", body: text, wantClass: "external_article"},
		{name: "text_markdown", ctype: "text/markdown", body: text, wantClass: "external_article"},
		{name: "text_uppercase", ctype: "TEXT/PLAIN", body: text, wantClass: "external_article"},
		{name: "allowlist_json", ctype: "application/json", body: `{"a":1}`, wantClass: "external_article"},
		{name: "allowlist_xml", ctype: "application/xml", body: "<a/>", wantClass: "external_article"},
		{name: "allowlist_xhtml", ctype: "application/xhtml+xml", body: "<html/>", wantClass: "external_article"},
		{name: "allowlist_padded", ctype: "  application/json ; charset=utf-8", body: `{"a":1}`, wantClass: "external_article"},
		{name: "pdf", ctype: "application/pdf", body: "%PDF-1.7 bytes", wantClass: "external_pdf"},
		{name: "pdf_with_parameter", ctype: "application/pdf; version=1.7", body: "%PDF-1.7 bytes", wantClass: "external_pdf"},

		// Everything not text/*, not on the allowlist, and not PDF is refused.
		{name: "octet_stream", ctype: "application/octet-stream", body: text, wantErr: `non-text content-type "application/octet-stream" rejected`},
		{name: "image", ctype: "image/png", body: text, wantErr: `non-text content-type "image/png" rejected`},
		{name: "allowlist_near_miss", ctype: "application/jsonl", body: text, wantErr: `non-text content-type "application/jsonl" rejected`},
		{name: "zip", ctype: "application/zip", body: text, wantErr: `non-text content-type "application/zip" rejected`},
		{name: "missing", ctype: "", body: text, wantErr: `non-text content-type "(missing)" rejected`},

		// A text/* header is not a promise: the bytes are still decoded, and
		// undecodable ones are refused rather than stored as knowledge.
		{name: "text_plain_with_nul", ctype: "text/plain", body: "head\x00tail", wantErr: "binary source rejected"},
		{name: "text_plain_invalid_utf8", ctype: "text/plain", body: "head\xfftail", wantErr: "not decodable text"},
	}
	pdf := func([]byte) (string, error) { return "extracted pdf text", nil }
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := FetchedSource{FinalURL: docURL, Body: []byte(tc.body)}
			if tc.ctype != "" {
				fs.Headers = map[string]string{"Content-Type": tc.ctype}
			}
			mat, err := materialFromFetched(docURL, fs, pdf)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("content-type %q must be rejected, got material %+v", tc.ctype, mat)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("content-type %q must be accepted: %v", tc.ctype, err)
			}
			if mat.sourceClass != tc.wantClass {
				t.Fatalf("source class = %q, want %q", mat.sourceClass, tc.wantClass)
			}
			if tc.wantClass == "external_pdf" {
				if mat.text != "extracted pdf text" {
					t.Fatalf("PDF text = %q, want the extractor's output", mat.text)
				}
				if mat.ext != ".pdf" {
					t.Fatalf("PDF ext = %q, want .pdf", mat.ext)
				}
			} else if mat.text != tc.body {
				t.Fatalf("text = %q, want the body verbatim %q", mat.text, tc.body)
			}
			if string(mat.rawBytes) != tc.body {
				t.Fatalf("rawBytes = %q, want the fetched bytes verbatim", mat.rawBytes)
			}
			if mat.origin != docURL {
				t.Fatalf("origin = %q, want %q", mat.origin, docURL)
			}
		})
	}
}

// TestMaterialFromFetchedFinalURLFallback proves a fetcher that reports no final
// URL (no redirect information) falls back to the requested URL rather than
// recording an empty origin.
func TestMaterialFromFetchedFinalURLFallback(t *testing.T) {
	mat, err := materialFromFetched(docURL, textFetched("", "text/plain", "body"), nil)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if mat.origin != docURL || mat.title != docURL {
		t.Fatalf("origin/title = %q/%q, want the requested URL when FinalURL is empty", mat.origin, mat.title)
	}
	// A rejection message must name the requested URL too, never an empty string.
	_, err = materialFromFetched(docURL, FetchedSource{Headers: map[string]string{"Content-Type": "image/png"}}, nil)
	if err == nil || !strings.Contains(err.Error(), docURL) {
		t.Fatalf("rejection must name the requested URL, got: %v", err)
	}
}

// TestMaterialFromFetchedExtension pins where the kept-original extension comes
// from: the PATH component of the final URL, lowercased and shape-checked. A
// query string or fragment must not leak into it, and an implausible extension
// falls back to .txt.
func TestMaterialFromFetchedExtension(t *testing.T) {
	cases := []struct{ url, want string }{
		{"https://docs.example.test/a.json", ".json"},
		{"https://docs.example.test/a.JSON", ".json"},
		{"https://docs.example.test/archive.tar.gz", ".gz"},
		{"https://docs.example.test/a.html?v=1", ".html"},
		{"https://docs.example.test/a.html#top", ".html"},
		{"https://docs.example.test/rotation", ".txt"},
		{"https://docs.example.test/", ".txt"},
		{"https://docs.example.test/a.thisextensioniswaytoolong", ".txt"},
		{"https://docs.example.test/a.tar%2Egz", ".gz"},
	}
	for _, tc := range cases {
		t.Run(tc.url, func(t *testing.T) {
			mat, err := materialFromFetched(tc.url, textFetched(tc.url, "text/plain", "body"), nil)
			if err != nil {
				t.Fatalf("fetch: %v", err)
			}
			if mat.ext != tc.want {
				t.Fatalf("ext = %q, want %q", mat.ext, tc.want)
			}
		})
	}
}

// TestMaterialFromFetchedSizeCap proves an oversize fetched body is refused by
// the explicit cap — the same ceiling the local path applies — and names the cap
// so the refusal is not confused with a decode failure.
func TestMaterialFromFetchedSizeCap(t *testing.T) {
	body := strings.Repeat("a", maxFetchBytes+1)
	_, err := materialFromFetched(docURL, textFetched(docURL, "text/plain", body), nil)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("an over-cap fetched body must be rejected by the size cap, got: %v", err)
	}
	// One byte under the cap is still accepted, so the test pins a boundary and
	// not merely "large bodies fail".
	ok := strings.Repeat("a", maxFetchBytes)
	if _, err := materialFromFetched(docURL, textFetched(docURL, "text/plain", ok), nil); err != nil {
		t.Fatalf("a body exactly at the cap must be accepted: %v", err)
	}
}

// ---------------------------------------------------------------------------
// PDF extraction (iss-30 instance 5) — via the injectable PDFExtractor seam
// ---------------------------------------------------------------------------

// TestExtractPDFText walks the extractor contract directly: no extractor, a
// failing extractor, an extractor that yields nothing usable, and the success
// case. Each failure must be a distinct, self-explaining message — "no text"
// and "no parser installed" are different operator problems.
func TestExtractPDFText(t *testing.T) {
	cases := []struct {
		name    string
		pdf     PDFExtractor
		want    string
		wantErr string
	}{
		{
			name:    "nil_extractor",
			pdf:     nil,
			wantErr: "PDF extraction unavailable",
		},
		{
			name:    "extractor_error",
			pdf:     func([]byte) (string, error) { return "", errors.New("xref table corrupt") },
			wantErr: "PDF extraction failed: xref table corrupt",
		},
		{
			name:    "empty_text",
			pdf:     func([]byte) (string, error) { return "", nil },
			wantErr: "PDF has no extractable text",
		},
		{
			name:    "whitespace_only_text",
			pdf:     func([]byte) (string, error) { return "  \n\t \r\n ", nil },
			wantErr: "PDF has no extractable text",
		},
		{
			name: "success",
			pdf:  func([]byte) (string, error) { return "Rotate tokens every 24 hours.", nil },
			want: "Rotate tokens every 24 hours.",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractPDFText([]byte("%PDF-1.7"), tc.pdf)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("want an error containing %q, got text %q", tc.wantErr, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tc.wantErr)
				}
				var ie *IngestError
				if !errors.As(err, &ie) {
					t.Fatalf("PDF failure is %T, want *IngestError (a pre-write refusal)", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("extract: %v", err)
			}
			if got != tc.want {
				t.Fatalf("text = %q, want %q", got, tc.want)
			}
		})
	}
	// The extractor is handed the PDF bytes unchanged.
	var seen []byte
	raw := []byte("%PDF-1.7\x00binary")
	if _, err := extractPDFText(raw, func(b []byte) (string, error) { seen = b; return "text", nil }); err != nil {
		t.Fatalf("extract: %v", err)
	}
	if string(seen) != string(raw) {
		t.Fatalf("extractor received %q, want the raw bytes %q", seen, raw)
	}
}

// TestMaterialFromLocalPDFSniff proves both local PDF triggers — the .pdf
// extension and the %PDF- magic prefix — route to the extractor, that neither
// path decodes the bytes as text, and that an ordinary text file does NOT reach
// the extractor at all.
func TestMaterialFromLocalPDFSniff(t *testing.T) {
	dir := t.TempDir()
	// Binary content: were the PDF branch missed, decodeText would reject it, so
	// "no error" alone proves the sniff fired.
	pdfBytes := "%PDF-1.7\n\x00\x01binary object stream"

	cases := []struct {
		name    string
		file    string
		content string
		wantPDF bool
	}{
		// Distinct base names: a case-insensitive filesystem would collide on
		// paper.pdf/paper.PDF and silently test the same file twice.
		{name: "extension_lowercase", file: "lower.pdf", content: "plain text, PDF only by name", wantPDF: true},
		{name: "extension_uppercase", file: "upper.PDF", content: "plain text, PDF only by name", wantPDF: true},
		{name: "magic_prefix", file: "paper.bin", content: pdfBytes, wantPDF: true},
		{name: "plain_text_file", file: "notes.txt", content: "Rotate tokens every 24 hours.", wantPDF: false},
		{name: "magic_not_at_start", file: "notes2.txt", content: "intro %PDF-1.7", wantPDF: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeSource(t, dir, tc.file, tc.content)
			called := false
			var seen []byte
			pdf := func(b []byte) (string, error) {
				called, seen = true, b
				return "extracted pdf text", nil
			}
			mat, err := materialFromLocal(dir, path, pdf)
			if err != nil {
				t.Fatalf("materialFromLocal: %v", err)
			}
			if called != tc.wantPDF {
				t.Fatalf("extractor called = %v, want %v", called, tc.wantPDF)
			}
			if tc.wantPDF {
				if string(seen) != tc.content {
					t.Fatalf("extractor received %q, want the file bytes", seen)
				}
				if mat.text != "extracted pdf text" {
					t.Fatalf("text = %q, want the extractor output", mat.text)
				}
				if mat.sourceClass != "external_pdf" || mat.ext != ".pdf" {
					t.Fatalf("class/ext = %q/%q, want external_pdf/.pdf", mat.sourceClass, mat.ext)
				}
				// --keep-original must store the ORIGINAL bytes, not the text.
				if string(mat.rawBytes) != tc.content {
					t.Fatalf("rawBytes = %q, want the original file bytes", mat.rawBytes)
				}
				return
			}
			if mat.sourceClass != "external_article" {
				t.Fatalf("class = %q, want external_article", mat.sourceClass)
			}
			if mat.text != tc.content {
				t.Fatalf("text = %q, want the file content verbatim", mat.text)
			}
		})
	}
}

// TestMaterialFromLocalPDFFailures proves a local PDF whose text cannot be
// extracted is refused outright — it is never silently decoded as text, which
// for a real PDF would store binary object-stream noise as knowledge.
func TestMaterialFromLocalPDFFailures(t *testing.T) {
	dir := t.TempDir()
	path := writeSource(t, dir, "paper.pdf", "%PDF-1.7\nobject stream")
	cases := []struct {
		name    string
		pdf     PDFExtractor
		wantErr string
	}{
		{name: "nil_extractor", pdf: nil, wantErr: "PDF extraction unavailable"},
		{name: "extractor_error", pdf: func([]byte) (string, error) { return "", errors.New("encrypted") }, wantErr: "PDF extraction failed: encrypted"},
		{name: "no_text", pdf: func([]byte) (string, error) { return "   \n ", nil }, wantErr: "PDF has no extractable text"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mat, err := materialFromLocal(dir, path, tc.pdf)
			if err == nil {
				t.Fatalf("want a refusal, got material %+v", mat)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

// TestIngestPDFFromURL is the end-to-end PDF instance: an application/pdf
// response whose bytes are NOT decodable text is ingested through the injected
// extractor, classified external_pdf, hashed on the EXTRACTED text, and kept as
// the original PDF bytes.
func TestIngestPDFFromURL(t *testing.T) {
	repo := t.TempDir()
	const final = "https://docs.example.test/rotation.pdf"
	raw := "%PDF-1.7\n\x00\x01object stream"
	extracted := "Token rotation policy: rotate tokens every 24 hours."

	res, err := Ingest(IngestRequest{
		RepoRoot:     repo,
		Source:       final,
		Fetcher:      staticFetcher(nil, textFetched(final, "application/pdf", raw)),
		PDFExtractor: func([]byte) (string, error) { return extracted, nil },
		KeepOriginal: true,
		Distiller:    oneTopicDistiller("topic", "auth", "tokens", "# Token rotation\nRotate tokens every 24 hours."),
		Now:          fixedNow,
	})
	if err != nil {
		t.Fatalf("PDF ingest: %v", err)
	}
	if res.ContentHash != sourceContentHash(extracted) {
		t.Fatalf("content hash must be taken over the EXTRACTED text, not the PDF bytes")
	}
	reg, err := LoadRegistry(SourcesIndexPath(repo))
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	entry := reg[res.ContentHash].(map[string]any)
	consumer := entry["consumers"].(map[string]any)["memory"].(map[string]any)
	if consumer["class"] != "external_pdf" {
		t.Fatalf("registry class = %v, want external_pdf", consumer["class"])
	}
	if !strings.HasSuffix(res.KeptOriginal, ".pdf") {
		t.Fatalf("kept original = %q, want a .pdf extension", res.KeptOriginal)
	}
	kept, err := os.ReadFile(filepath.Join(repo, res.KeptOriginal))
	if err != nil {
		t.Fatalf("kept original: %v", err)
	}
	if string(kept) != raw {
		t.Fatalf("kept original must be the ORIGINAL PDF bytes, got %q", kept)
	}
}

// TestIngestDistilledControlCharsKeepProvenance is the ingest-level form of the
// writer/reader round-trip invariant: whatever control characters a distiller puts
// in a page field, the page it writes must read back with its provenance intact.
// A raw carriage return is the dangerous one — parseFrontmatter normalises it to a
// newline before splitting, so an unquoted \r followed by "---" ends the
// frontmatter early and the `source` block (with it every source hash) falls into
// the page body while Ingest still reports success.
func TestIngestDistilledControlCharsKeepProvenance(t *testing.T) {
	// `recall` sorts before `source` in the written frontmatter, so an injection
	// through it is exactly what would strip the source block below it.
	const injected = "harmless\r---\rleaked"
	repo := t.TempDir()
	body := "Token rotation policy: rotate tokens every 24 hours."

	res, err := Ingest(IngestRequest{
		RepoRoot: repo,
		Source:   docURL,
		Fetcher:  staticFetcher(nil, textFetched(docURL, "text/plain", body)),
		Distiller: func(string, map[string]any) ([]map[string]any, error) {
			return []map[string]any{{
				"type": "topic", "domain": "auth", "slug": "tokens",
				"body":   "# Token rotation\nRotate tokens every 24 hours.",
				"recall": []any{injected},
			}}, nil
		},
		Now: fixedNow,
	})
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if len(res.Pages) != 1 {
		t.Fatalf("pages = %v, want one page", res.Pages)
	}
	raw, err := os.ReadFile(filepath.Join(Dir(repo), res.Pages[0]))
	if err != nil {
		t.Fatalf("read page: %v", err)
	}
	text := string(raw)

	src := pageSourceBlock(text)
	if !contains(SourceHashes(src), res.ContentHash) {
		t.Fatalf("the page lost its provenance on reread: source block = %#v, want the ingested hash %q\npage:\n%q", src, res.ContentHash, text)
	}
	fm, err := parseFrontmatter(text)
	if err != nil {
		t.Fatalf("written page does not reparse: %v\npage:\n%q", err, text)
	}
	recall, ok := fm["recall"].([]any)
	if !ok || len(recall) != 1 || recall[0] != any(injected) {
		t.Fatalf("recall = %#v, want the distilled value verbatim %q", fm["recall"], injected)
	}
	if _, ok := fm["topic_hash"]; !ok {
		t.Fatalf("topic_hash was pushed out of the frontmatter: %#v", fm)
	}
}

// TestIngestRejectedSourceWritesNothing proves the content-type and PDF
// refusals are pre-dispatch: the distiller never runs and the store is left
// untouched — no memory page, no sources index.
func TestIngestRejectedSourceWritesNothing(t *testing.T) {
	cases := []struct {
		name    string
		ctype   string
		pdf     PDFExtractor
		wantErr string
	}{
		{name: "non_text", ctype: "image/png", wantErr: "non-text content-type"},
		{name: "pdf_without_extractor", ctype: "application/pdf", wantErr: "PDF extraction unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			_, err := Ingest(IngestRequest{
				RepoRoot: repo,
				Source:   docURL,
				Fetcher:  staticFetcher(nil, textFetched(docURL, tc.ctype, "%PDF-1.7 or png bytes")),
				Distiller: func(string, map[string]any) ([]map[string]any, error) {
					t.Fatal("distiller must not run for a refused source")
					return nil, nil
				},
				PDFExtractor: tc.pdf,
				Now:          fixedNow,
			})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tc.wantErr)
			}
			if _, err := os.Stat(SourcesIndexPath(repo)); !os.IsNotExist(err) {
				t.Fatalf("a refused source must not create the sources index (stat err = %v)", err)
			}
			// An ABSENT memory directory is itself a pass: a refusal happens before
			// any write, so the store is never created. Hence the read is only
			// asserted on when it succeeds — the two acceptable states are "no
			// directory" and "an empty one".
			if entries, err := os.ReadDir(Dir(repo)); err == nil && len(entries) > 0 {
				t.Fatalf("a refused source must not write into the store, found %d entries", len(entries))
			} else if err != nil && !os.IsNotExist(err) {
				t.Fatalf("reading the store: %v", err)
			}
		})
	}
}
