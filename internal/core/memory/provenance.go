package memory

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// provenance.go — the provenance/licence substrate (09-provenance-substrate.md):
// content hashing, the .sources_index.json registry (pure merge/serialise/load),
// SPDX licence detection, and citation generation. Pure helpers carry no I/O and
// no lock; WritePages owns the store lock.

// TokenCountVersion is the pinned deterministic tokenizer version: count of
// regex \w+ word tokens over the normalised text. The registry stores it so
// consumers read a stable basis.
const TokenCountVersion = 1

var wordRe = regexp.MustCompile(`\w+`)

// ---------------------------------------------------------------------------
// Store paths
// ---------------------------------------------------------------------------

// RelDir is the store's repo-relative directory, slash-separated. It is what a
// front door names when it has a checkout root and no store path yet — the
// stray-store walk a resolving front door makes reports a substrate sitting
// below the checkout root, and it has only the relative shape to look for.
const RelDir = ".abcd/memory"

// Dir returns the canonical store path <repoRoot>/.abcd/memory.
func Dir(repoRoot string) string { return filepath.Join(repoRoot, filepath.FromSlash(RelDir)) }

// SourcesIndexPath returns .abcd/memory/.sources_index.json.
func SourcesIndexPath(repoRoot string) string {
	return filepath.Join(Dir(repoRoot), ".sources_index.json")
}

// CoverageIndexPath returns .abcd/memory/.coverage_index.json.
func CoverageIndexPath(repoRoot string) string {
	return filepath.Join(Dir(repoRoot), coverageIndexName)
}

// ---------------------------------------------------------------------------
// Content hashing
// ---------------------------------------------------------------------------

// NormaliseSourceText normalises line endings (CRLF/CR to LF) and strips
// per-line trailing whitespace — applied before hashing so cosmetic transfer
// differences do not defeat dedup-by-hash.
func NormaliseSourceText(text string) string {
	unified := strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	parts := strings.Split(unified, "\n")
	for i, line := range parts {
		parts[i] = strings.TrimRight(line, " \t\v\f")
	}
	return strings.Join(parts, "\n")
}

// SourceContentHash is the sha256 hex digest of the normalised source text —
// the registry key.
func SourceContentHash(text string) string {
	sum := sha256.Sum256([]byte(NormaliseSourceText(text)))
	return hex.EncodeToString(sum[:])
}

// CountSourceTokens is the token_count_version 1 tokenizer: regex \w+ word
// tokens over the normalised text.
func CountSourceTokens(normalised string) int {
	return len(wordRe.FindAllString(normalised, -1))
}

func sha256Hex(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// ---------------------------------------------------------------------------
// Registry — pure helpers (no I/O, no lock)
// ---------------------------------------------------------------------------

var hex64Re = regexp.MustCompile(`^[0-9a-f]{64}$`)
var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// maxRegistryBytes caps the sources-index read. The registry lives inside the
// repo working tree — a trust boundary — so it is read with the guarded
// primitive (O_NOFOLLOW + regular-file + size cap): a committed symlink to
// /dev/zero or an endless device would otherwise OOM or hang the CLI, and the
// read also runs under the store lock, so a hang would wedge it. A few MiB is
// generous for a JSON index.
const maxRegistryBytes = 8 << 20 // 8 MiB

// LoadRegistry reads the registry JSON. A missing file yields an empty
// registry; a present-but-unparseable file raises RegistryFormatError — durable
// source metadata must fail loudly, never be silently replaced.
func LoadRegistry(path string) (map[string]any, error) {
	raw, err := fsutil.ReadGuarded(path, maxRegistryBytes)
	return decodeRegistry(raw, err, path)
}

// decodeRegistry is LoadRegistry's meaning applied to a read that already
// happened, by path or through a store handle: absent is an empty registry,
// anything unreadable or malformed a *RegistryFormatError naming path.
func decodeRegistry(raw []byte, err error, path string) (map[string]any, error) {
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		// A committed symlink where the index should be fails the O_NOFOLLOW open
		// with ELOOP (a raw *os.PathError), not ErrNotRegular. Classify it with the
		// other guarded refusals: a symlinked index is exactly "not a readable
		// regular file", so a caller branching on RegistryFormatError treats it
		// identically, and the raw ELOOP syscall detail never escapes.
		if errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, fsutil.ErrTooBig) || errors.Is(err, syscall.ELOOP) {
			return nil, &RegistryFormatError{Msg: fmt.Sprintf("sources index at %s is not a readable regular file within the size cap", path)}
		}
		return nil, err
	}
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, &RegistryFormatError{Msg: fmt.Sprintf("corrupt sources index at %s: %v", path, err)}
	}
	m, ok := data.(map[string]any)
	if !ok {
		return nil, &RegistryFormatError{Msg: fmt.Sprintf("sources index root must be a JSON object: %s", path)}
	}
	return m, nil
}

// SerializeRegistry deterministically serialises the registry (sorted keys,
// 2-space indent, trailing newline).
func SerializeRegistry(registry map[string]any) string {
	return marshalIndentNoEscape(registry)
}

// IngestEvent is one ingest event to merge into the registry. A zero
// SourceTokenCount (or TokenCountVersion) is treated as "not provided" — the
// durable value is filled only when currently null, never overwritten.
type IngestEvent struct {
	ContentHash       string
	Consumer          string
	SourceClass       string
	Citation          map[string]any
	Origin            string
	Licence           string
	IngestedAt        string
	Pages             []string
	SourceTokenCount  int
	TokenCountVersion int
}

// MergeIngest merges one ingest event into registry and returns a NEW map (the
// input is never mutated). Pure — no I/O, no lock.
func MergeIngest(registry map[string]any, ev IngestEvent) (map[string]any, error) {
	if strings.TrimSpace(ev.Consumer) == "" {
		return nil, fmt.Errorf("consumer must be a non-empty string")
	}
	if !hex64Re.MatchString(ev.ContentHash) {
		return nil, fmt.Errorf("content_hash must be a sha256 hex digest, got %q", ev.ContentHash)
	}
	if !dateRe.MatchString(ev.IngestedAt) {
		return nil, fmt.Errorf("ingested_at must be YYYY-MM-DD, got %q", ev.IngestedAt)
	}

	merged := deepCopyMap(registry)
	citationCopy := deepCopyMap(ev.Citation)
	pageList := append([]string(nil), ev.Pages...)

	var tokenVal any
	if ev.SourceTokenCount > 0 {
		tokenVal = ev.SourceTokenCount
	}
	var tokenVerVal any
	if ev.TokenCountVersion > 0 {
		tokenVerVal = ev.TokenCountVersion
	}

	entryAny, exists := merged[ev.ContentHash]
	entry, _ := entryAny.(map[string]any)
	if !exists || entry == nil {
		merged[ev.ContentHash] = map[string]any{
			"origin":              ev.Origin,
			"licence":             ev.Licence,
			"source_token_count":  tokenVal,
			"token_count_version": tokenVerVal,
			"ingest_count":        1,
			"first_ingest":        ev.IngestedAt,
			"last_ingest":         ev.IngestedAt,
			"consumers": map[string]any{
				ev.Consumer: map[string]any{
					"class":       ev.SourceClass,
					"citation":    citationCopy,
					"ingested_at": ev.IngestedAt,
					"pages":       toAnySlice(pageList),
				},
			},
		}
		return merged, nil
	}

	entry["ingest_count"] = toInt(entry["ingest_count"]) + 1
	entry["last_ingest"] = ev.IngestedAt
	if _, ok := entry["first_ingest"]; !ok {
		entry["first_ingest"] = ev.IngestedAt
	}
	if s, _ := entry["origin"].(string); s == "" {
		entry["origin"] = ev.Origin
	}
	if s, _ := entry["licence"].(string); s == "" {
		entry["licence"] = ev.Licence
	}
	if entry["source_token_count"] == nil && tokenVal != nil {
		entry["source_token_count"] = tokenVal
	}
	if entry["token_count_version"] == nil && tokenVerVal != nil {
		entry["token_count_version"] = tokenVerVal
	}

	consumers, _ := entry["consumers"].(map[string]any)
	if consumers == nil {
		consumers = map[string]any{}
		entry["consumers"] = consumers
	}
	existing, _ := consumers[ev.Consumer].(map[string]any)
	if existing == nil {
		consumers[ev.Consumer] = map[string]any{
			"class":       ev.SourceClass,
			"citation":    citationCopy,
			"ingested_at": ev.IngestedAt,
			"pages":       toAnySlice(pageList),
		}
	} else {
		existing["ingested_at"] = ev.IngestedAt
		mergedPages := anyToStrings(existing["pages"])
		for _, p := range pageList {
			if !contains(mergedPages, p) {
				mergedPages = append(mergedPages, p)
			}
		}
		existing["pages"] = toAnySlice(mergedPages)
		// class + citation deliberately untouched: read-only after creation.
	}
	return merged, nil
}

// ---------------------------------------------------------------------------
// Citation
// ---------------------------------------------------------------------------

// BuildCitation builds a knowledge citation mapping in the locked shape from
// 09-provenance-substrate.md §2.
func BuildCitation(typ, origin, author, title string, year int, ingestedAt, ingestedBy string) map[string]any {
	return map[string]any{
		"type":        typ,
		"origin":      origin,
		"author":      author,
		"title":       title,
		"year":        year,
		"ingested_at": ingestedAt,
		"ingested_by": ingestedBy,
	}
}

// ---------------------------------------------------------------------------
// SPDX licence detection
// ---------------------------------------------------------------------------

// LicenceDetection is the result of DetectLicence.
type LicenceDetection struct {
	Licence     string // SPDX id, verbatim compound expression, or "unknown"
	Restrictive bool
	Source      string // where it was detected: spdx_header | manifest | licence_file | http_header | none
}

var canonicalIDs = map[string]string{
	"mit":          "MIT",
	"apache-2.0":   "Apache-2.0",
	"bsd-3-clause": "BSD-3-Clause",
	"gpl-3.0":      "GPL-3.0",
	"agpl-3.0":     "AGPL-3.0",
	"cc-by-4.0":    "CC-BY-4.0",
	"unknown":      "unknown",
}

var restrictiveCanonical = map[string]bool{"GPL-3.0": true, "AGPL-3.0": true}
var expressionOperators = map[string]bool{"AND": true, "OR": true, "WITH": true}

var spdxTagRe = regexp.MustCompile(`(?i)SPDX-License-Identifier:\s*([^\r\n]+)`)

var tagValueTrailers = []string{"*/", "-->", "#>", "}}"}

// DetectLicence detects the licence of a source. Priority: in-file SPDX header,
// then (only when sourceRoot != "") package.json / LICENSE-file SPDX, then the
// HTTP License: header, else explicit "unknown". Memory ingest passes
// sourceRoot="" so only steps 1 + 4 apply. (TOML manifests are not parsed in
// the Go port — no stdlib TOML — but memory never supplies a sourceRoot, so the
// manifest step is inert for this surface.)
func DetectLicence(text, sourceRoot string, httpHeaders map[string]string) LicenceDetection {
	raw := ""
	method := "none"

	if c := contentSPDXHeader(text); c != "" {
		raw, method = c, "spdx_header"
	}
	if raw == "" && sourceRoot != "" {
		if c := manifestLicence(sourceRoot); c != "" {
			raw, method = c, "manifest"
		}
	}
	if raw == "" && sourceRoot != "" {
		if c := licenceFileLicence(sourceRoot); c != "" {
			raw, method = c, "licence_file"
		}
	}
	if raw == "" && len(httpHeaders) > 0 {
		if c := httpHeaderLicence(httpHeaders); c != "" {
			raw, method = c, "http_header"
		}
	}
	if raw == "" {
		return LicenceDetection{Licence: "unknown", Restrictive: false, Source: "none"}
	}
	stored := normaliseLicenceValue(raw)
	return LicenceDetection{
		Licence:     stored,
		Restrictive: classifyLicenceExpression(stored) == "restrictive",
		Source:      method,
	}
}

func classifyLicenceExpression(expr string) string {
	flattened := strings.NewReplacer("(", " ", ")", " ").Replace(expr)
	var tokens []string
	for _, tok := range strings.Fields(flattened) {
		if !expressionOperators[strings.ToUpper(tok)] {
			tokens = append(tokens, tok)
		}
	}
	if len(tokens) == 0 {
		return "unknown"
	}
	canonical := make([]string, len(tokens))
	for i, tok := range tokens {
		canonical[i] = canonicalIDs[strings.ToLower(tok)]
	}
	for _, c := range canonical {
		if restrictiveCanonical[c] {
			return "restrictive"
		}
	}
	for _, c := range canonical {
		if c == "" {
			return "unrecognised"
		}
	}
	for _, c := range canonical {
		if c == "unknown" {
			return "unknown"
		}
	}
	return "permissive"
}

func normaliseLicenceValue(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, "\"'")
	s = strings.TrimSpace(s)
	if s == "" {
		return "unknown"
	}
	if isCompoundExpression(s) {
		return s
	}
	if canonical, ok := canonicalIDs[strings.ToLower(s)]; ok {
		return canonical
	}
	return s
}

func isCompoundExpression(s string) bool {
	if strings.ContainsAny(s, "()") {
		return true
	}
	for _, tok := range strings.Fields(s) {
		if expressionOperators[strings.ToUpper(tok)] {
			return true
		}
	}
	return false
}

func contentSPDXHeader(text string) string {
	m := spdxTagRe.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	value := strings.TrimSpace(m[1])
	for _, trailer := range tagValueTrailers {
		if strings.HasSuffix(value, trailer) {
			value = strings.TrimSpace(value[:len(value)-len(trailer)])
		}
	}
	return value
}

func manifestLicence(sourceRoot string) string {
	pkg := filepath.Join(sourceRoot, "package.json")
	// Guarded: sourceRoot is an arbitrary ingest source, outside the trusted
	// worktree — a symlinked manifest is refused.
	if raw, err := fsutil.ReadGuarded(pkg, maxRegistryBytes); err == nil {
		var data map[string]any
		if json.Unmarshal(raw, &data) == nil {
			if lic, ok := data["license"].(string); ok && strings.TrimSpace(lic) != "" {
				return strings.TrimSpace(lic)
			}
		}
	}
	return ""
}

var licenceFileNames = []string{"LICENSE", "LICENCE", "LICENSE.md", "LICENCE.md", "LICENSE.txt", "LICENCE.txt"}

func licenceFileLicence(sourceRoot string) string {
	for _, name := range licenceFileNames {
		path := filepath.Join(sourceRoot, name)
		raw, err := fsutil.ReadGuarded(path, maxRegistryBytes)
		if err != nil {
			continue
		}
		content := string(raw)
		if tagged := contentSPDXHeader(content); tagged != "" {
			return tagged
		}
		for _, line := range strings.Split(content, "\n") {
			t := strings.TrimSpace(line)
			if t == "" {
				continue
			}
			if canonical, ok := canonicalIDs[strings.ToLower(t)]; ok && canonical != "unknown" {
				return t
			}
			break
		}
	}
	return ""
}

func httpHeaderLicence(headers map[string]string) string {
	for k, v := range headers {
		if strings.ToLower(k) == "license" && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Small typed helpers
// ---------------------------------------------------------------------------

func marshalIndentNoEscape(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	// encoding/json sorts map keys; Encoder appends a trailing newline.
	_ = enc.Encode(v)
	return buf.String()
}

func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = deepCopyValue(v)
	}
	return out
}

func deepCopyValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return deepCopyMap(val)
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = deepCopyValue(item)
		}
		return out
	case []string:
		return toAnySlice(val)
	default:
		return val
	}
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func anyToStrings(v any) []string {
	var out []string
	if list, ok := v.([]any); ok {
		for _, item := range list {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
