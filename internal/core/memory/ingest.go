package memory

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/intentdriven/abcd/internal/urlguard"
)

// ingest.go — the ingest flow from 07-memory.md §1:
// probe -> licence-detect -> distil (injected seam) -> validate -> atomic write
// -> discard/keep-original. Every pre-dispatch failure raises before the single
// WritePages call — no orphan registry row, no partial page.

const ingestedBy = "abcd memory ingest"

const (
	maxFetchBytes       = 10 * 1024 * 1024
	fetchTimeoutSeconds = 30
	maxRedirects        = 5
)

var textContentTypes = map[string]bool{
	"application/json": true, "application/xml": true, "application/xhtml+xml": true,
}

var extRe = regexp.MustCompile(`^\.[a-z0-9]{1,10}$`)

// FetchedSource is the raw result of fetching a URL (the injectable Fetcher
// contract). Content-type / size / decode checks are applied uniformly by the
// ingest path after the fetcher returns.
type FetchedSource struct {
	FinalURL string
	Headers  map[string]string
	Body     []byte
}

// Fetcher fetches a URL; a nil Fetcher uses the bounded default fetch.
type Fetcher func(url string) (FetchedSource, error)

// PDFExtractor extracts text from PDF bytes; a nil extractor rejects with a
// clear error (never silently pulls in a parser dependency).
type PDFExtractor func(data []byte) (string, error)

// Distiller is the host-delegated seam: (normalisedText, sourceBlock) -> raw
// page maps. A map omitting "source" gets sourceBlock injected. The core
// validates every page.
type Distiller func(normalisedText string, sourceBlock map[string]any) ([]map[string]any, error)

// IngestRequest is the input to Ingest.
type IngestRequest struct {
	RepoRoot     string
	Source       string
	Distiller    Distiller
	KeepOriginal bool
	Fetcher      Fetcher
	PDFExtractor PDFExtractor
	Now          time.Time
}

// IngestResult is the structured result of one Ingest call.
type IngestResult struct {
	Status           string         `json:"status"`
	ContentHash      string         `json:"content_hash"`
	Licence          string         `json:"licence"`
	SourceTokenCount int            `json:"source_token_count"`
	Pages            []string       `json:"pages"`
	Citation         map[string]any `json:"citation"`
	KeptOriginal     string         `json:"kept_original"`
	// KeepOriginalError records a --keep-original copy failure that occurred
	// AFTER the pages and registry were durably written. The ingest itself
	// succeeded; only the best-effort original copy did not. Empty when
	// --keep-original was not requested or the copy succeeded.
	KeepOriginalError string       `json:"keep_original_error,omitempty"`
	Linked            [][2]string  `json:"linked"`
	Contradictions    [][2]string  `json:"contradictions"`
	WriteReport       *WriteReport `json:"write_report"`
}

type sourceMaterial struct {
	origin      string
	text        string
	rawBytes    []byte
	headers     map[string]string
	ext         string
	sourceClass string
	title       string
}

// Ingest runs the full ingest flow for one source (a local path or an https
// URL; a plaintext http source is refused — see isURL).
func Ingest(req IngestRequest) (IngestResult, error) {
	if req.Distiller == nil {
		return IngestResult{}, newIngestError("ingest requires a Distiller")
	}
	root := req.RepoRoot
	// Refuse a symlinked store DIRECTORY before the pre-write dedup reads
	// (pageHashSet / existingPageFrontmatter) touch it (GHSA-72rp): their leaf
	// O_NOFOLLOW guards do not contain a symlinked ancestor. The write path is
	// already refused at WritePages -> validatedMemoryDir, but that fires only
	// after these reads; guarding here closes the pre-write read. A missing store
	// is fine (present=false) — WritePages materialises it.
	if _, _, err := safeMemoryDir(root); err != nil {
		return IngestResult{}, err
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ingestedAt := now.Format("2006-01-02")

	material, err := acquireSource(root, req.Source, req.Fetcher, req.PDFExtractor)
	if err != nil {
		return IngestResult{}, err
	}
	// Build the write-time sanitiser before any store write. It fails closed on a
	// degraded scanner, so a broken per-repo pii.json refuses the whole ingest
	// before anything is acquired rather than committing acquired source text
	// with a weakened detector (GHSA-j5f5-phgm-9m73). The --keep-original copy
	// routes through it below; the page bodies are redacted by WritePages
	// itself, the one primitive every verb writes through.
	redactor, err := newStoreRedactor(root)
	if err != nil {
		return IngestResult{}, err
	}
	// The origin and title the core copies into every citation and registry
	// entry are acquired text too: a redirect-controlled final URL carries its
	// query string, a local path can sit under the caller's home. They are
	// judged here, once, before either path reads them, so the fast path's
	// fill-if-empty origin, the citation, the registry event and the returned
	// result all agree on the one redacted value (GHSA-x46m-mw9h-5jwj). The
	// WritePages leaf walk stays the fail-closed backstop behind this.
	if material.origin, _, err = redactor.redactText(material.origin, "citation.origin"); err != nil {
		return IngestResult{}, err
	}
	if material.title, _, err = redactor.redactText(material.title, "citation.title"); err != nil {
		return IngestResult{}, err
	}
	normalized := NormaliseSourceText(material.text)
	if strings.TrimSpace(normalized) == "" {
		return IngestResult{}, newIngestError("source has no text content: %s", material.origin)
	}
	contentHash := SourceContentHash(material.text)
	tokenCount := CountSourceTokens(normalized)

	registry, err := LoadRegistry(SourcesIndexPath(root))
	if err != nil {
		return IngestResult{}, err
	}
	entry, _ := registry[contentHash].(map[string]any)
	var memoryConsumer map[string]any
	if entry != nil {
		if consumers, ok := entry["consumers"].(map[string]any); ok {
			memoryConsumer, _ = consumers["memory"].(map[string]any)
		}
	}
	mem := Dir(root)

	// ---- Registry-hit fast path (validate BEFORE mutate) -------------------
	var validRecorded []string
	var recorded []string
	if memoryConsumer != nil {
		recorded = anyToStrings(memoryConsumer["pages"])
		allValid := len(recorded) > 0
		for _, pageName := range recorded {
			hashes, present := pageHashSet(mem, pageName)
			if present && contains(hashes, contentHash) {
				validRecorded = append(validRecorded, pageName)
			} else {
				allValid = false
			}
		}
		if allValid {
			sourceClass := material.sourceClass
			if c, ok := memoryConsumer["class"].(string); ok && c != "" {
				sourceClass = c
			}
			citation, _ := memoryConsumer["citation"].(map[string]any)
			if citation == nil {
				citation = map[string]any{}
			}
			origin := material.origin
			if o, ok := entry["origin"].(string); ok && o != "" {
				origin = o
			}
			licence := "unknown"
			if l, ok := entry["licence"].(string); ok && l != "" {
				licence = l
			}
			event := IngestEvent{
				ContentHash: contentHash, Consumer: "memory", SourceClass: sourceClass,
				Citation: citation, Origin: origin, Licence: licence, IngestedAt: ingestedAt,
				Pages: recorded, SourceTokenCount: tokenCount, TokenCountVersion: TokenCountVersion,
			}
			var newRegistry map[string]any
			report, err := WritePages(root, nil, func(current map[string]any) (map[string]any, error) {
				merged, err := MergeIngest(current, event)
				if err != nil {
					return nil, err
				}
				newRegistry = merged
				return merged, nil
			}, now)
			if err != nil {
				return IngestResult{}, err
			}
			// Best-effort keep-original: a failure after the durable write is
			// recorded, never reported as total failure (iss-30).
			kept, keepErr := "", ""
			if req.KeepOriginal {
				if k, serr := storeOriginal(root, material, contentHash, redactor); serr != nil {
					keepErr = keepOriginalErrorMessage(serr)
				} else {
					kept = k
				}
			}
			cachedEntry, _ := newRegistry[contentHash].(map[string]any)
			cachedConsumers, _ := cachedEntry["consumers"].(map[string]any)
			cached, _ := cachedConsumers["memory"].(map[string]any)
			cachedCitation, _ := cached["citation"].(map[string]any)
			resultLicence := "unknown"
			if l, ok := cachedEntry["licence"].(string); ok && l != "" {
				resultLicence = l
			}
			return IngestResult{
				Status: "registry_only", ContentHash: contentHash, Licence: resultLicence,
				SourceTokenCount: tokenCount, Pages: recorded, Citation: deepCopyMap(cachedCitation),
				KeptOriginal: kept, KeepOriginalError: keepErr, WriteReport: &report,
			}, nil
		}
	}

	repairing := memoryConsumer != nil

	// ---- Licence detect (sourceRoot="": SPDX header + HTTP License:) --------
	detection := DetectLicence(material.text, "", material.headers)
	// The licence is lifted verbatim from the source's own bytes (an SPDX line)
	// or from a response header, so it is judged before it is copied into the
	// source block, the registry event and IngestResult.Licence — the same
	// discipline as a body: redact, then refuse only a blocking residual, so a
	// fingerprinted token is stored as the honest "ghp_***" it is rather than
	// the whole ingest refused on a value the operator never wrote.
	licence, _, err := redactor.redactText(detection.Licence, "source.licence")
	if err != nil {
		return IngestResult{}, err
	}

	citation := BuildCitation("knowledge", material.origin, "unknown", material.title, now.Year(), ingestedAt, ingestedBy)
	sourceBlock, err := buildSingleSource(material.sourceClass, citation, licence, contentHash, ingestedAt)
	if err != nil {
		return IngestResult{}, err
	}

	// ---- Distil + validate BEFORE any write --------------------------------
	rawPages, err := req.Distiller(normalized, sourceBlock)
	if err != nil {
		return IngestResult{}, err
	}
	var distilled []DistilledPage
	for _, raw := range rawPages {
		if _, ok := raw["source"]; !ok {
			merged := map[string]any{}
			for k, v := range raw {
				merged[k] = v
			}
			merged["source"] = sourceBlock
			raw = merged
		}
		page, err := ValidateDistilledPage(raw)
		if err != nil {
			return IngestResult{}, err
		}
		distilled = append(distilled, page)
	}
	if len(distilled) == 0 {
		return IngestResult{}, newIngestError("distillation produced 0 pages for %s; nothing written", material.origin)
	}
	for _, page := range distilled {
		if !contains(SourceHashes(page.Source), contentHash) {
			return IngestResult{}, newIngestError("distilled page %s does not cite the ingested source hash %s; refusing to write an unattributable page", page.Filename(), contentHash)
		}
	}

	// ---- Existing pages + repair safety ------------------------------------
	existing := existingPageFrontmatter(mem)
	if repairing {
		for _, pageName := range recorded {
			if contains(validRecorded, pageName) {
				continue
			}
			hashes, present := pageHashSet(mem, pageName)
			if !present {
				continue // missing — re-distil writes fresh
			}
			if len(hashes) == 0 {
				delete(existing, pageName)
				continue
			}
			for _, h := range hashes {
				if _, ok := registry[h]; ok {
					return IngestResult{}, newIngestError("repair collision: recorded page %s now cites a different registry entry; operator resolution required, nothing overwritten", pageName)
				}
			}
		}
	}

	plan, err := ResolveDistilledPages(existing, distilled)
	if err != nil {
		return IngestResult{}, err
	}

	// ---- Build the COMPLETE new registry mapping ---------------------------
	ourPages := append([]string(nil), plan.RegistryPages[contentHash]...)
	for _, pageName := range validRecorded {
		if !contains(ourPages, pageName) {
			ourPages = append(ourPages, pageName)
		}
	}
	// Back-link invariant: the entry lists EXACTLY the live page set.
	dedupPages := dedupStrings(ourPages)
	event := IngestEvent{
		ContentHash: contentHash, Consumer: "memory", SourceClass: material.sourceClass,
		Citation: citation, Origin: material.origin, Licence: licence, IngestedAt: ingestedAt,
		Pages: ourPages, SourceTokenCount: tokenCount, TokenCountVersion: TokenCountVersion,
	}
	// The full registry mutation is recomputed under the store lock against the
	// freshly-read registry (lost-update fix): merge this event, pin the
	// consumer page set, and back-link the other cited hashes.
	merge := func(current map[string]any) (map[string]any, error) {
		newRegistry, err := MergeIngest(current, event)
		if err != nil {
			return nil, err
		}
		setConsumerPages(newRegistry, contentHash, dedupPages)
		backlinkOtherHashes(newRegistry, plan, contentHash, distilled, ingestedAt)
		return newRegistry, nil
	}

	// The distiller is host-delegated and may echo a secret or an absolute home
	// path from the source straight into a page body; WritePages redacts every
	// body before it lands — the core's own fail-closed gate, not a trust in
	// the host.
	var pageWrites []PageWrite
	for _, w := range plan.Writes {
		pageWrites = append(pageWrites, PageWrite{Filename: w.Filename, Frontmatter: w.Frontmatter, Body: w.Body})
	}
	report, err := WritePages(root, pageWrites, merge, now)
	if err != nil {
		return IngestResult{}, err
	}
	// storeOriginal runs AFTER the durable page + registry write. A failure
	// here does not un-ingest anything, so it must not be reported as total
	// failure — record it and return the successful result (iss-30).
	kept, keepErr := "", ""
	if req.KeepOriginal {
		if k, serr := storeOriginal(root, material, contentHash, redactor); serr != nil {
			keepErr = keepOriginalErrorMessage(serr)
		} else {
			kept = k
		}
	}

	status := "ingested"
	if repairing {
		status = "repaired"
	}
	return IngestResult{
		Status: status, ContentHash: contentHash, Licence: licence,
		SourceTokenCount: tokenCount, Pages: dedupPages, Citation: citation,
		KeptOriginal: kept, KeepOriginalError: keepErr,
		Linked: plan.Linked, Contradictions: plan.Contradictions,
		WriteReport: &report,
	}, nil
}

// ---------------------------------------------------------------------------
// Registry back-link helpers
// ---------------------------------------------------------------------------

func setConsumerPages(registry map[string]any, contentHash string, pages []string) {
	entry, _ := registry[contentHash].(map[string]any)
	consumers, _ := entry["consumers"].(map[string]any)
	memc, _ := consumers["memory"].(map[string]any)
	if memc != nil {
		memc["pages"] = toAnySlice(pages)
	}
}

func backlinkOtherHashes(registry map[string]any, plan WritePlan, contentHash string, distilled []DistilledPage, ingestedAt string) {
	sourceMeta := map[string]map[string]any{}
	for _, page := range distilled {
		var entries []map[string]any
		if _, ok := page.Source["class"]; ok {
			entries = []map[string]any{page.Source}
		} else if raw, ok := page.Source["sources"].([]any); ok {
			for _, e := range raw {
				if em, ok := e.(map[string]any); ok {
					entries = append(entries, em)
				}
			}
		}
		for _, se := range entries {
			if h, ok := se["source_hash"].(string); ok {
				if _, exists := sourceMeta[h]; !exists {
					sourceMeta[h] = se
				}
			}
		}
	}
	for otherHash, filenames := range plan.RegistryPages {
		if otherHash == contentHash {
			continue
		}
		entry, ok := registry[otherHash].(map[string]any)
		if !ok {
			continue
		}
		consumers, _ := entry["consumers"].(map[string]any)
		if consumers == nil {
			consumers = map[string]any{}
			entry["consumers"] = consumers
		}
		consumer, _ := consumers["memory"].(map[string]any)
		if consumer == nil {
			meta := sourceMeta[otherHash]
			class := "external_article"
			var citation map[string]any = map[string]any{}
			if meta != nil {
				if c, ok := meta["class"].(string); ok {
					class = c
				}
				if c, ok := meta["citation"].(map[string]any); ok {
					citation = deepCopyMap(c)
				}
			}
			consumer = map[string]any{
				"class": class, "citation": citation, "ingested_at": ingestedAt, "pages": []any{},
			}
			consumers["memory"] = consumer
		}
		pages := anyToStrings(consumer["pages"])
		for _, f := range filenames {
			if !contains(pages, f) {
				pages = append(pages, f)
			}
		}
		consumer["pages"] = toAnySlice(pages)
	}
}

// ---------------------------------------------------------------------------
// Registry-hit validation helpers
// ---------------------------------------------------------------------------

// maxMemoryPageBytes caps a single memory page read. Pages sit inside the repo
// working tree, so each is read with the guarded primitive (O_NOFOLLOW + size
// cap): an in-store page name can itself be a committed symlink to /dev/zero, and
// os.ReadFile would follow it and grow without bound. A page is a small distilled
// fact; a few MiB is far more than any real one.
const maxMemoryPageBytes = 4 << 20 // 4 MiB

func pageHashSet(mem, filename string) ([]string, bool) {
	if !IsMemoryPageName(filename) {
		return nil, false
	}
	path := filepath.Join(mem, filename)
	raw, err := fsutil.ReadGuarded(path, maxMemoryPageBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false
		}
		// A symlinked, oversize, or otherwise unreadable page is present-but-
		// unparseable: keep the name taken (empty hash set) rather than following
		// a hostile leaf or reading it unbounded.
		return []string{}, true
	}
	return SourceHashes(pageSourceBlock(string(raw))), true
}

func existingPageFrontmatter(mem string) map[string]map[string]any {
	pages := map[string]map[string]any{}
	entries, err := os.ReadDir(mem)
	if err != nil {
		return pages
	}
	for _, e := range entries {
		if !e.Type().IsRegular() || !IsMemoryPageName(e.Name()) {
			continue
		}
		// ReadGuarded re-checks regular-file on the open fd (closing the ReadDir→
		// open symlink-swap TOCTOU) and caps the size, so a hostile page cannot
		// redirect the read or exhaust memory.
		raw, err := fsutil.ReadGuarded(filepath.Join(mem, e.Name()), maxMemoryPageBytes)
		if err != nil {
			pages[e.Name()] = map[string]any{}
			continue
		}
		fm, err := parseFrontmatter(string(raw))
		if err != nil {
			pages[e.Name()] = map[string]any{}
			continue
		}
		pages[e.Name()] = fm
	}
	return pages
}

// ---------------------------------------------------------------------------
// Source acquisition
// ---------------------------------------------------------------------------

// isURL reports whether source is one abcd will FETCH. Only https qualifies:
// the store copies a fetched source's text, and the SPDX line or License:
// header lifted out of it, verbatim into durable provenance, so a plaintext
// transport would let a man-in-the-middle choose what the store remembers and
// under what licence (GHSA-35fj-9w6f-7h62). This is the posture of `abcd
// update` and of the brief's HTTPS-only fetch invariant; unlike update's
// allowHTTP test seam there is no escape here, and the additive --allow-http
// alternative is recorded as an open observation.
func isURL(source string) bool {
	u, err := url.Parse(source)
	if err != nil {
		return false
	}
	return u.Scheme == "https"
}

// remoteScheme returns the scheme of a source written in absolute-URL form
// ("<scheme>://…") and "" for anything else, which is how a refused scheme is
// told apart from a local path. Without it a plaintext http:// source would
// fall through to the file lookup and be reported as a missing path — a
// refusal must say what was wrong with the source the caller actually gave.
func remoteScheme(source string) string {
	u, err := url.Parse(source)
	if err != nil || u.Scheme == "" || !strings.HasPrefix(source[len(u.Scheme):], "://") {
		return ""
	}
	return u.Scheme
}

// credentialQueryKeys are the query parameter names whose value is, by
// convention, a bearer credential. A URL is the one piece of acquired text the
// store copies verbatim into a citation, and the write-time scanner cannot help
// here: an opaque token or password matches no pattern, so it has to be dropped
// structurally, by the name it travels under. The list is deliberately short and
// closed, and every entry has to name a secret in ESSENTIALLY EVERY usage, not
// merely in a credentialled one: dropping a parameter truncates the address the
// citation exists to reproduce, and a citation that no longer resolves is its
// own defect. `key` and `signature` fail that bar — `?key=section3` addresses a
// document section and `?signature=v2` names a content signature at least as
// often as either carries a secret — so they are not on the list. A key-shaped
// credential that travels under one of those names is out of reach here by
// design; it is the operator's to keep out of the URL.
var credentialQueryKeys = map[string]bool{
	"token":        true,
	"api_key":      true,
	"apikey":       true,
	"access_token": true,
	"password":     true,
	"secret":       true,
}

// dropCredentialQuery removes every credential-shaped query parameter from u in
// place and reports whether it removed one. The surviving pairs keep their
// original encoding — the raw query is filtered textually rather than round-
// tripped through url.Values, which would re-order and re-escape an address the
// store means to reproduce faithfully.
func dropCredentialQuery(u *url.URL) bool {
	if u.RawQuery == "" {
		return false
	}
	dropped := false
	pairs := strings.Split(u.RawQuery, "&")
	kept := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		name := pair
		if i := strings.IndexByte(pair, '='); i >= 0 {
			name = pair[:i]
		}
		decoded, err := url.QueryUnescape(name)
		if err != nil {
			decoded = name
		}
		if credentialQueryKeys[strings.ToLower(strings.TrimSpace(decoded))] {
			dropped = true
			continue
		}
		kept = append(kept, pair)
	}
	if dropped {
		u.RawQuery = strings.Join(kept, "&")
	}
	return dropped
}

// sanitisedOrigin strips the credentials a URL can carry before it becomes the
// durable origin and title of a stored page. Two of them: basic-auth userinfo
// (dropped whole — a stored origin is a citation, not a way back in) and a
// credential-shaped query key. Nothing else is touched, and a URL carrying
// neither is returned byte-identical, so the address a citation reproduces is
// the fetched one wherever there is no credential to remove.
func sanitisedOrigin(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return maskUserinfo(raw)
	}
	changed := false
	if u.User != nil {
		u.User = nil
		changed = true
	}
	if dropCredentialQuery(u) {
		changed = true
	}
	if !changed {
		return raw
	}
	return u.String()
}

// redactedSource renders a URL for an error message with any userinfo masked
// and any credential-shaped query parameter dropped, so a refusal never echoes
// a credential embedded in the source back to the terminal or the run log. It
// keeps the username where the origin form drops it: a refusal is read by the
// operator who typed the URL, and the account name is how they recognise it.
func redactedSource(source string) string {
	u, err := url.Parse(source)
	if err != nil {
		return maskUserinfo(source)
	}
	dropCredentialQuery(u)
	return u.Redacted()
}

// maskUserinfo is the fallback for a string url.Parse refused: there is no
// structured userinfo to mask, so the "scheme://user:pass@" prefix is masked
// textually, in the shape url.URL.Redacted uses. A string with no such prefix
// is returned unchanged — masking is all this does; it never claims the string
// is a URL.
func maskUserinfo(source string) string {
	i := strings.Index(source, "://")
	if i < 0 {
		return source
	}
	rest := source[i+3:]
	at := strings.IndexByte(rest, '@')
	if at < 0 {
		return source
	}
	if slash := strings.IndexByte(rest, '/'); slash >= 0 && slash < at {
		return source
	}
	user := rest[:at]
	if colon := strings.IndexByte(user, ':'); colon >= 0 {
		user = user[:colon]
	}
	return source[:i+3] + user + ":xxxxx@" + rest[at+1:]
}

// transportCause renders the CAUSE of a failed fetch without the transport's
// own copy of the request URL. net/http wraps every client error in a
// *url.Error whose Error() re-prints the whole URL — masking only the
// basic-auth password (stripPassword), never the query — so interpolating the
// raw error with %v put a `?token=…` credential straight back into a message
// whose own URL argument had just been filtered. Unwrapping to the innermost
// non-*url.Error cause keeps what the operator needs (the reason the fetch
// failed) and drops the one part the wrapper re-adds.
func transportCause(err error) string {
	for {
		var ue *url.Error
		if !errors.As(err, &ue) || ue.Err == nil {
			return err.Error()
		}
		err = ue.Err
	}
}

func acquireSource(repoRoot, source string, fetcher Fetcher, pdf PDFExtractor) (sourceMaterial, error) {
	// The scheme gate stands ahead of the fetcher seam, not inside defaultFetch
	// alone: a caller-supplied Fetcher replaces defaultFetch entirely, and the
	// admission rule belongs to the core either way.
	if scheme := remoteScheme(source); scheme != "" && !isURL(source) {
		return sourceMaterial{}, newIngestError(
			"refusing to fetch %s: memory ingest requires https, not %s — a plaintext source, and the licence header the store copies into its provenance, can be rewritten in transit",
			redactedSource(source), scheme)
	}
	if isURL(source) {
		var fetched FetchedSource
		var err error
		if fetcher != nil {
			fetched, err = fetcher(source)
		} else {
			fetched, err = defaultFetch(source)
		}
		if err != nil {
			// The IngestError itself, never the *url.Error a client wrapped it
			// in: the wrapper re-prints the request URL, query credential and
			// all, in front of the message this package composed.
			var ie *IngestError
			if errors.As(err, &ie) {
				return sourceMaterial{}, ie
			}
			return sourceMaterial{}, newIngestError("fetch failed for %s: %s", redactedSource(source), transportCause(err))
		}
		return materialFromFetched(source, fetched, pdf)
	}
	return materialFromLocal(repoRoot, source, pdf)
}

// guardFetchHost refuses a host the shared SSRF guard blocks, re-typed as an
// IngestError so every ingest failure a caller sees is one error type. The guard
// itself lives in internal/urlguard — the canonical home shared with the citation
// refresh's fetcher, so the two fetch paths cannot drift apart.
func guardFetchHost(host string) error {
	if err := urlguard.CheckHost(host); err != nil {
		return newIngestError("%s", err.Error())
	}
	return nil
}

// ingestRedirectPolicy is defaultFetch's CheckRedirect, lifted out so the
// scheme pin can be tested without a network (a loopback httptest server
// cannot stand in: urlguard.DialControl refuses loopback by design). Admission
// checking the scheme is not enough — a redirect chooses the next hop's scheme,
// so an https source can be walked down to plaintext one Location at a time.
// The pin per hop mirrors update.go's newUpdater, and it stands AHEAD of the
// host guard so the refusal names the scheme rather than the host.
func ingestRedirectPolicy(rawURL string) func(*http.Request, []*http.Request) error {
	return func(r *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return newIngestError("too many redirects fetching %s", redactedSource(rawURL))
		}
		if r.URL.Scheme != "https" {
			return newIngestError("redirect left https: %s", redactedSource(r.URL.String()))
		}
		return guardFetchHost(r.URL.Hostname())
	}
}

func defaultFetch(rawURL string) (FetchedSource, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return FetchedSource{}, newIngestError("fetch failed for %s: %s", redactedSource(rawURL), transportCause(err))
	}
	// acquireSource has already refused a non-https source; this is the same
	// rule stated where the connection is actually made, so the fetcher cannot
	// be reached over plaintext by a later caller (mirrors update.go's get).
	if parsed.Scheme != "https" {
		return FetchedSource{}, newIngestError("refusing to fetch %s: the source origin must be https", redactedSource(rawURL))
	}
	if err := guardFetchHost(parsed.Hostname()); err != nil {
		return FetchedSource{}, err
	}
	// A connect-time Control hook re-checks the ACTUAL resolved IP for every
	// dialled connection, closing the DNS-rebinding gap between the name-based
	// guard above and the transport's own resolution.
	dialer := &net.Dialer{
		Timeout: fetchTimeoutSeconds * time.Second,
		Control: urlguard.DialControl(urlguard.BlockedIP),
	}
	client := &http.Client{
		Timeout:       fetchTimeoutSeconds * time.Second,
		Transport:     &http.Transport{DialContext: dialer.DialContext},
		CheckRedirect: ingestRedirectPolicy(rawURL),
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return FetchedSource{}, newIngestError("fetch failed for %s: %s", redactedSource(rawURL), transportCause(err))
	}
	req.Header.Set("User-Agent", "abcd-memory-ingest")
	resp, err := client.Do(req)
	if err != nil {
		// A CheckRedirect refusal comes back wrapped in a *url.Error; return the
		// refusal this package composed, not the wrapper's re-print of the URL.
		var ie *IngestError
		if errors.As(err, &ie) {
			return FetchedSource{}, ie
		}
		return FetchedSource{}, newIngestError("fetch failed for %s: %s", redactedSource(rawURL), transportCause(err))
	}
	defer resp.Body.Close()
	return readFetchedResponse(rawURL, resp)
}

// readFetchedResponse validates the HTTP status and reads the size-capped body.
// A non-2xx response (a 404/500 error page) is an ingest ERROR, not source
// content — otherwise the error page's HTML would be stored as knowledge.
func readFetchedResponse(rawURL string, resp *http.Response) (FetchedSource, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return FetchedSource{}, newIngestError("fetch failed for %s: HTTP %d %s", redactedSource(rawURL), resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBytes+1))
	if err != nil {
		return FetchedSource{}, newIngestError("fetch failed for %s: %s", redactedSource(rawURL), transportCause(err))
	}
	headers := map[string]string{}
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}
	final := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		final = resp.Request.URL.String()
	}
	return FetchedSource{FinalURL: final, Headers: headers, Body: body}, nil
}

func materialFromLocal(repoRoot, source string, pdf PDFExtractor) (sourceMaterial, error) {
	expanded := source
	// Expand only a leading "~" or "~/…" to the home dir. A "~user" form is NOT the
	// caller's home and must not be mangled into home+"user" — leave it literal.
	if expanded == "~" || strings.HasPrefix(expanded, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			expanded = home + expanded[1:]
		}
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return sourceMaterial{}, newIngestError("source path is invalid: %q (%v)", fsutil.RepoRel(repoRoot, expanded), err)
	}
	resolved := abs
	if r, err := filepath.EvalSymlinks(abs); err == nil {
		resolved = r
	}
	// A source may lie outside the repo; render it repo-relative once and use that
	// everywhere its path reaches machine output or a persisted citation, so no
	// absolute developer-identity path is ever emitted (iss-81). The bare
	// filesystem-error reason is stripped too, so a *PathError cannot re-embed the
	// absolute path a repo-relative message deliberately dropped.
	//
	// Relativise `abs`, NOT the EvalSymlinks-resolved path: repoRoot comes from the
	// working directory in its logical (unresolved) form, so relativising a resolved
	// path would diverge under any symlinked prefix (macOS /var→/private/var) and
	// produce a "../../…/private/var/…" locator that re-embeds the absolute location.
	// `resolved` still drives the actual stat/read below.
	rel := fsutil.RepoRel(repoRoot, abs)
	st, err := os.Stat(resolved)
	if err != nil {
		return sourceMaterial{}, newIngestError("source path is invalid: %q (%v)", rel, bareErr(err))
	}
	if !st.Mode().IsRegular() {
		return sourceMaterial{}, newIngestError("source path is not a regular file (directories, devices and symlinks-to-special are rejected): %s", rel)
	}
	// Cap the local read the same as the URL path, so a huge local file cannot be
	// slurped whole into memory before any text/NUL sniffing.
	if st.Size() > maxFetchBytes {
		return sourceMaterial{}, newIngestError("source file exceeds the %d-byte cap: %s", maxFetchBytes, rel)
	}
	// Read through the shared guarded primitive, not os.ReadFile: the os.Stat
	// above and a bare ReadFile leave a swap window (a type/symlink swap between
	// stat and read) and an uncapped read (a regular file that GROWS past the cap
	// after the size check is read whole). ReadGuarded validates on the same fd it
	// reads and caps the bytes actually read. resolved is EvalSymlinks output, so
	// the leaf is the real file and O_NOFOLLOW does not break a legitimate
	// symlink source. The Stat pre-check stays as belt-and-suspenders (iss-109)
	// and keeps the distinct "source path is invalid" message for a missing path.
	raw, err := fsutil.ReadGuarded(resolved, maxFetchBytes)
	switch {
	case errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, syscall.ELOOP):
		return sourceMaterial{}, newIngestError("source path is not a regular file (directories, devices and symlinks-to-special are rejected): %s", rel)
	case errors.Is(err, fsutil.ErrTooBig):
		return sourceMaterial{}, newIngestError("source file exceeds the %d-byte cap: %s", maxFetchBytes, rel)
	case err != nil:
		return sourceMaterial{}, newIngestError("cannot read source: %s (%v)", rel, bareErr(err))
	}
	isPDF := strings.ToLower(filepath.Ext(resolved)) == ".pdf" || strings.HasPrefix(string(raw), "%PDF-")
	if isPDF {
		text, err := extractPDFText(raw, pdf)
		if err != nil {
			return sourceMaterial{}, err
		}
		return sourceMaterial{origin: rel, text: text, rawBytes: raw, ext: ".pdf", sourceClass: "external_pdf", title: filepath.Base(resolved)}, nil
	}
	text, err := decodeText(raw, rel)
	if err != nil {
		return sourceMaterial{}, err
	}
	ext := safeExt(filepath.Ext(resolved))
	if ext == "" {
		ext = ".txt"
	}
	return sourceMaterial{origin: rel, text: text, rawBytes: raw, ext: ext, sourceClass: "external_article", title: filepath.Base(resolved)}, nil
}

func materialFromFetched(rawURL string, fetched FetchedSource, pdf PDFExtractor) (sourceMaterial, error) {
	if len(fetched.Body) > maxFetchBytes {
		return sourceMaterial{}, newIngestError("fetched source exceeds the %d-byte cap: %s", maxFetchBytes, redactedSource(rawURL))
	}
	ctype := contentType(fetched.Headers)
	finalURL := fetched.FinalURL
	if finalURL == "" {
		finalURL = rawURL
	}
	// The final address is redirect-controlled, and net/url preserves raw non-ASCII
	// in the query, so C1/bidi/zero-width runes in a hostile Location would reach the
	// durable sources registry (JSON, which encoding/json does NOT escape for those
	// classes) and the returned citation verbatim — the same class the cite fetcher
	// fixed one door over (iss-359). Percent-encode the hidden runes losslessly here,
	// once, before finalURL becomes the stored origin/title (iss-357).
	finalURL = termsafe.EncodeHiddenRunes(finalURL)
	// The same address is also the one place a CREDENTIAL reaches the store
	// intact: `https://user:pass@host/doc` and `?token=…` are copied verbatim
	// into the page frontmatter's citation, the sources registry and the
	// returned result, and the write-time scanner cannot see an opaque password
	// — it matches no pattern. Strip both here, once, before finalURL becomes
	// the stored origin AND title. Hidden-rune encoding runs first so a URL
	// carrying a control byte parses at all: url.Parse refuses one, and an
	// unparseable URL is exactly where a credential would otherwise survive.
	finalURL = sanitisedOrigin(finalURL)
	if ctype == "application/pdf" {
		text, err := extractPDFText(fetched.Body, pdf)
		if err != nil {
			return sourceMaterial{}, err
		}
		return sourceMaterial{origin: finalURL, text: text, rawBytes: fetched.Body, headers: fetched.Headers, ext: ".pdf", sourceClass: "external_pdf", title: finalURL}, nil
	}
	if strings.HasPrefix(ctype, "text/") || textContentTypes[ctype] {
		text, err := decodeText(fetched.Body, finalURL)
		if err != nil {
			return sourceMaterial{}, err
		}
		ext := ""
		if u, err := url.Parse(finalURL); err == nil {
			ext = safeExt(filepath.Ext(u.Path))
		}
		if ext == "" {
			ext = ".txt"
		}
		return sourceMaterial{origin: finalURL, text: text, rawBytes: fetched.Body, headers: fetched.Headers, ext: ext, sourceClass: "external_article", title: finalURL}, nil
	}
	shown := ctype
	if shown == "" {
		shown = "(missing)"
	}
	return sourceMaterial{}, newIngestError("non-text content-type %q rejected for %s; nothing written", shown, finalURL)
}

func extractPDFText(data []byte, pdf PDFExtractor) (string, error) {
	if pdf == nil {
		return "", newIngestError("PDF extraction unavailable: no PDF text-extraction dependency is installed (supply a PDFExtractor)")
	}
	text, err := pdf(data)
	if err != nil {
		return "", newIngestError("PDF extraction failed: %v", err)
	}
	if strings.TrimSpace(text) == "" {
		return "", newIngestError("PDF has no extractable text; nothing to ingest")
	}
	return text, nil
}

func decodeText(data []byte, what string) (string, error) {
	for _, b := range data {
		if b == 0 {
			return "", newIngestError("binary source rejected: %s contains NUL bytes and no text-extraction path applies", what)
		}
	}
	if !utf8.Valid(data) {
		return "", newIngestError("binary source rejected: %s is not decodable text", what)
	}
	return string(data), nil
}

func contentType(headers map[string]string) string {
	for k, v := range headers {
		if strings.ToLower(k) == "content-type" {
			return strings.ToLower(strings.TrimSpace(strings.SplitN(v, ";", 2)[0]))
		}
	}
	return ""
}

func safeExt(ext string) string {
	ext = strings.ToLower(ext)
	if extRe.MatchString(ext) {
		return ext
	}
	return ""
}

// ---------------------------------------------------------------------------
// Original storage (--keep-original)
// ---------------------------------------------------------------------------

// sourcesRelPath is the repo-relative location of the kept-originals store,
// used in user-facing errors so no absolute path leaks into rendered output.
var sourcesRelPath = filepath.Join(".abcd", "memory", "sources")

// keepOriginalErrorMessage renders a --keep-original failure without leaking the
// absolute sources path: filesystem errors embed the full path(s), so report
// only their bare cause against the repo-relative store location (iss-30). Both
// *PathError (Lstat/MkdirAll/OpenFile/Write/Sync) and *LinkError (Rename, which
// carries TWO absolute paths) are stripped; the only other storeOriginal error
// already names the repo-relative sourcesRelPath.
func keepOriginalErrorMessage(err error) string {
	if pe := (*os.PathError)(nil); errors.As(err, &pe) {
		return fmt.Sprintf("could not store original under %s: %s", sourcesRelPath, pe.Err.Error())
	}
	if le := (*os.LinkError)(nil); errors.As(err, &le) {
		return fmt.Sprintf("could not store original under %s: %s", sourcesRelPath, le.Err.Error())
	}
	return err.Error()
}

func storeOriginal(repoRoot string, material sourceMaterial, contentHash string, redactor *storeRedactor) (string, error) {
	// The kept-original copy lands in the committed store, so it is sanitised
	// through the same detector as the page bodies before it is written — the raw
	// bytes verbatim were the zero-host-cooperation leak in GHSA-j5f5-phgm-9m73.
	payload, err := redactor.redactOriginalBytes(material)
	if err != nil {
		return "", err
	}
	sourcesDir := filepath.Join(Dir(repoRoot), "sources")
	if fi, err := os.Lstat(sourcesDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(sourcesDir, 0o755); err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	} else if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return "", newIngestError("sources dir is a symlink or non-directory: %s", sourcesRelPath)
	}
	// The sources dir is guaranteed a real directory by the guard above; route
	// the durable write through the canonical primitive (temp + fsync + chmod +
	// rename + parent-dir fsync) rather than an inline copy (iss-79 /
	// one-canonical-primitive). os.Rename does not follow a leaf symlink, so a
	// pre-planted target symlink is replaced, not written through.
	target := filepath.Join(sourcesDir, contentHash+material.ext)
	if err := fsutil.WriteFileAtomic(target, payload, 0o644); err != nil {
		return "", err
	}
	return filepath.Join(".abcd", "memory", "sources", contentHash+material.ext), nil
}

func dedupStrings(ss []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
