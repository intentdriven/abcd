package banlist

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// docsLintAllowEscape is the per-line escape every entry in the family declares —
// the same regexp the hand-curated families use, so one comment suppresses any
// banned token on its line and a reader learns one convention, not two.
const docsLintAllowEscape = `(?i)<!--\s*docs-lint:\s*allow\b`

// defaultSuccessor is the machine-readable replacement a verb-written entry
// declares. The family's schema REQUIRES a successor (a ban with no successor left
// its replacement in prose only), so a default is what makes an argless add
// produce a valid entry rather than one the linter refuses to load.
const defaultSuccessor = "a generic term"

// PublicEntry is one entry of the public banned-token family. Public entries
// render IN FULL: they are committed, reviewable config, and the pattern is the
// reviewable part (AC6).
type PublicEntry struct {
	ID       string `json:"id"`
	Pattern  string `json:"pattern"`
	Severity string `json:"severity"`
	// Key is the id with the managed prefix stripped, empty for an entry outside
	// the verb-owned namespace.
	Key string `json:"key"`
	// Managed marks the entries these verbs own and may remove.
	Managed bool `json:"managed"`
}

// PublicReport is the public layer's state.
type PublicReport struct {
	Path    string        `json:"path"`
	Present bool          `json:"present"`
	Entries []PublicEntry `json:"entries"`
}

// PublicResult is the outcome of a public-layer mutation.
type PublicResult struct {
	Path  string      `json:"path"`
	Entry PublicEntry `json:"entry"`
}

// AddPublicRequest adds one public entry. Severity defaults to blocker; Successor
// defaults to defaultSuccessor.
type AddPublicRequest struct {
	RepoRoot  string
	Key       string
	Pattern   string
	Severity  string
	Successor string
}

// Report is both layers at once, for the read-only render.
type Report struct {
	Private PrivateReport `json:"private"`
	Public  PublicReport  `json:"public"`
	// Inherited is the private layer a LINKED git worktree inherits from its primary
	// checkout, and nil everywhere else — so a standalone checkout's envelope is
	// exactly what it always was. It is carried on the one read every status surface
	// makes, rather than left for each to remember to ask: the guard enforces that
	// layer in a worktree (itd-150), and a surface that forgot it would report a
	// worktree unprotected while every commit made in it is checked.
	Inherited *InheritedReport `json:"inherited,omitempty"`
}

// List reports both layers. It is the read-only status render's single call.
func List(repoRoot string) (Report, error) {
	priv, err := ListPrivate(repoRoot)
	if err != nil {
		return Report{}, err
	}
	pub, err := ListPublic(repoRoot)
	if err != nil {
		return Report{}, err
	}
	inherited, err := InheritedPrivate(repoRoot)
	if err != nil {
		return Report{}, err
	}
	return Report{Private: priv, Public: pub, Inherited: inherited}, nil
}

// publicPath resolves the docs-lint config's absolute path under repoRoot.
func publicPath(repoRoot string) string {
	return filepath.Join(repoRoot, filepath.FromSlash(PublicConfigRelPath))
}

// readPublic returns the config's bytes, or ErrNoStore when it does not exist.
func readPublic(repoRoot string) ([]byte, error) {
	data, err := fsutil.ReadGuarded(publicPath(repoRoot), maxStoreBytes)
	switch {
	case err == nil:
		return data, nil
	case os.IsNotExist(err):
		return nil, fmt.Errorf("%w: %s", ErrNoStore, PublicConfigRelPath)
	default:
		return nil, fmt.Errorf("%w: %s is unreadable", ErrMalformedStore, PublicConfigRelPath)
	}
}

// ListPublic reports the whole banned_tokens family — the hand-curated entries as
// well as the verb-managed ones. There is one primitive, so a list that hid the
// hand-curated half would misrepresent what gates the repo.
func ListPublic(repoRoot string) (PublicReport, error) {
	rep := PublicReport{Path: PublicConfigRelPath, Entries: []PublicEntry{}}
	data, err := readPublic(repoRoot)
	switch {
	case err == nil:
	case errors.Is(err, ErrNoStore):
		return rep, nil
	default:
		return PublicReport{}, err
	}
	rep.Present = true
	span, err := locateBannedTokens(data)
	if err != nil {
		return PublicReport{}, err
	}
	for _, el := range span.elems {
		var tok lint.BannedToken
		if err := json.Unmarshal(el.raw, &tok); err != nil {
			return PublicReport{}, fmt.Errorf("%w: %s has an unreadable banned_tokens entry", ErrMalformedStore, PublicConfigRelPath)
		}
		rep.Entries = append(rep.Entries, publicEntry(tok))
	}
	return rep, nil
}

// publicEntry renders one loaded token as a reportable entry.
func publicEntry(tok lint.BannedToken) PublicEntry {
	e := PublicEntry{ID: tok.ID, Pattern: tok.Pattern, Severity: tok.Severity}
	if key, ok := strings.CutPrefix(tok.ID, PublicIDPrefix); ok {
		e.Key, e.Managed = key, true
	}
	return e
}

// AddPublic inserts one entry into the banned_tokens family. The edit is byte
// surgery on the located array — one inserted line — so unrelated keys, entry
// order, and the file's formatting survive byte-for-byte and a review diff shows
// only the new entry.
func AddPublic(req AddPublicRequest) (PublicResult, error) {
	if !validKey(req.Key) {
		return PublicResult{}, fmt.Errorf("%w: %q (want [A-Za-z0-9][A-Za-z0-9._/-]*)", ErrInvalidKey, req.Key)
	}
	stored := storedPublicPattern(req.Pattern)
	if !validPublicPattern(stored) {
		return PublicResult{}, fmt.Errorf("%w for key %q: empty, or not a usable regular expression", ErrInvalidPattern, req.Key)
	}
	severity := req.Severity
	if severity == "" {
		severity = SeverityBlocker
	}
	if !slices.Contains(Severities(), severity) {
		return PublicResult{}, fmt.Errorf("%w: %q (want %s|%s)", ErrInvalidSeverity, req.Severity, SeverityBlocker, SeverityWarn)
	}
	successor := strings.TrimSpace(req.Successor)
	if successor == "" {
		successor = defaultSuccessor
	}

	id := PublicIDPrefix + req.Key
	if err := withPublicLock(req.RepoRoot, func() error {
		data, err := readPublic(req.RepoRoot)
		if err != nil {
			return err
		}
		span, err := locateBannedTokens(data)
		if err != nil {
			return err
		}
		for _, el := range span.elems {
			var tok lint.BannedToken
			if err := json.Unmarshal(el.raw, &tok); err == nil && tok.ID == id {
				return fmt.Errorf("%w: %q", ErrDuplicateKey, req.Key)
			}
		}

		entry, err := encodeEntry(id, stored, severity, successor)
		if err != nil {
			return err
		}

		var out bytes.Buffer
		if len(span.elems) == 0 {
			indent := lineIndent(data, span.openEnd) + "  "
			out.Write(data[:span.openEnd])
			out.WriteString("\n" + indent)
			out.Write(entry)
			out.WriteString("\n" + lineIndent(data, span.openEnd))
			out.Write(data[span.closeStart:])
		} else {
			last := span.elems[len(span.elems)-1]
			out.Write(data[:last.end])
			out.WriteString(",\n" + elementIndent(data, span, last))
			out.Write(entry)
			out.Write(data[last.end:])
		}

		return writePublic(req.RepoRoot, out.Bytes())
	}); err != nil {
		return PublicResult{}, err
	}
	return PublicResult{
		Path:  PublicConfigRelPath,
		Entry: PublicEntry{ID: id, Pattern: stored, Severity: severity, Key: req.Key, Managed: true},
	}, nil
}

// storedPublicPattern is the exact string an entry carries on disk. Every one of
// the config's hand-curated entries opens with (?i), and the docs promise the
// public layer matches case-insensitively — but the linter compiles the stored
// pattern RAW, so an entry without the flag is case-SENSITIVE and a mixed-case
// spelling of the banned name walks through CI. The prefix is added here, once, so
// a verb-written entry is enforced exactly as a hand-written one is; a pattern that
// already carries it is left alone rather than double-flagged.
func storedPublicPattern(pattern string) string {
	if pattern == "" || strings.HasPrefix(pattern, "(?i)") {
		return pattern
	}
	return "(?i)" + pattern
}

// RemovePublic drops the verb-managed entry with the given key. Entries outside the
// managed namespace are refused: the hand-curated families are edited in the config
// by a human, in a reviewable commit, never by a verb.
func RemovePublic(repoRoot, key string) (PublicResult, error) {
	id := PublicIDPrefix + strings.TrimPrefix(key, PublicIDPrefix)
	var removed lint.BannedToken
	if err := withPublicLock(repoRoot, func() error {
		data, err := readPublic(repoRoot)
		if err != nil {
			return err
		}
		span, err := locateBannedTokens(data)
		if err != nil {
			return err
		}

		matched, sawBareKey := false, false
		for _, el := range span.elems {
			var tok lint.BannedToken
			if err := json.Unmarshal(el.raw, &tok); err != nil {
				continue
			}
			switch {
			case tok.ID == id:
				matched, removed = true, tok
			case tok.ID == key:
				// Named exactly, but outside the namespace these verbs own. Recorded,
				// not returned: a managed `names/<key>` MAY also be present, and firing
				// the namespace refusal here — before the whole array is scanned — would
				// shadow it, refusing to remove a target the verb actually owns.
				sawBareKey = true
			}
		}
		if !matched {
			if sawBareKey {
				return fmt.Errorf("%w: %q is hand-curated; edit %s to change it", ErrNotManaged, key, PublicConfigRelPath)
			}
			return fmt.Errorf("%w: %q", ErrUnknownKey, key)
		}

		// Remove every element carrying the id, not just one. A hand-edited config can
		// hold duplicate `names/<key>` entries (AddPublic refuses to create them, a
		// human edit is not so constrained); deleting one and reporting success would
		// leave a ban enforcing under a key the report calls gone — the asymmetry the
		// private store was hardened against. Each pass re-locates and drops the first
		// match with the proven single-element surgery, so the single-match case (the
		// common one) stays byte-identical.
		data = removeAllManaged(data, id)
		return writePublic(repoRoot, data)
	}); err != nil {
		return PublicResult{}, err
	}
	return PublicResult{Path: PublicConfigRelPath, Entry: publicEntry(removed)}, nil
}

// removeAllManaged returns cfg with every banned_tokens element whose id equals id
// deleted. Each pass re-locates the array and drops the first match with the same
// bounded byte-surgery a single removal uses, so removing one entry is byte-identical
// to the pre-existing path and removing duplicates leaves none behind. cfg is known
// to hold the array (the caller matched at least once).
func removeAllManaged(cfg []byte, id string) []byte {
	for {
		span, err := locateBannedTokens(cfg)
		if err != nil {
			return cfg
		}
		idx := -1
		for i, el := range span.elems {
			var tok lint.BannedToken
			if err := json.Unmarshal(el.raw, &tok); err != nil {
				continue
			}
			if tok.ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			return cfg
		}
		var out bytes.Buffer
		switch {
		case len(span.elems) == 1:
			out.Write(cfg[:span.openEnd])
			out.Write(cfg[span.closeStart:])
		case idx == 0:
			out.Write(cfg[:span.elems[0].start])
			out.Write(cfg[span.elems[1].start:])
		default:
			out.Write(cfg[:span.elems[idx-1].end])
			out.Write(cfg[span.elems[idx].end:])
		}
		cfg = out.Bytes()
	}
}

// withPublicLock serialises a load-modify-write of the committed config on the same
// inter-process lock primitive the private store uses. Two adds that interleave
// would each write a body derived from the same read, so one entry — one ban — is
// silently lost. The lock file lives in the gitignored local tier, never beside the
// committed config it guards.
func withPublicLock(repoRoot string, fn func() error) error {
	if err := mkdirLocalTier(repoRoot); err != nil {
		return err
	}
	lockPath := filepath.Join(repoRoot, filepath.FromSlash(PrivateDirRelPath), publicLockFilename)
	err := fsutil.WithFileLock(lockPath, storeLockTimeout, fn)
	switch {
	case errors.Is(err, fsutil.ErrLockContention):
		return fmt.Errorf("%w: could not lock %s within %s", ErrStoreLocked, PublicConfigRelPath, storeLockTimeout)
	case errors.Is(err, fsutil.ErrLockPathUnsafe):
		return fmt.Errorf("%w: the lock path for %s is not a regular file", ErrMalformedStore, PublicConfigRelPath)
	}
	return err
}

// writePublic validates the edited bytes and commits them atomically. Validation
// before the write is the point: the config gates CI, so an edit that would leave
// it unloadable must fail with the file untouched.
func writePublic(repoRoot string, data []byte) error {
	var probe lint.Config
	if err := json.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("%w: the edit would leave %s unparseable", ErrMalformedStore, PublicConfigRelPath)
	}
	return fsutil.WriteFileAtomicPreserveMode(publicPath(repoRoot), data)
}

// encodeEntry renders one banned-token entry as a single compact line, matching
// the field order the config's existing entries use.
func encodeEntry(id, pattern, severity, successor string) ([]byte, error) {
	// A local ordered shape rather than lint.BannedToken: the struct's field order
	// is the on-disk key order, and the entry must read like its neighbours.
	entry := struct {
		ID           string   `json:"id"`
		Pattern      string   `json:"pattern"`
		Severity     string   `json:"severity"`
		Successor    string   `json:"successor"`
		AllowContext []string `json:"allow_context"`
		Message      string   `json:"message"`
	}{
		ID: id, Pattern: pattern, Severity: severity, Successor: successor,
		AllowContext: []string{docsLintAllowEscape},
		Message: "names a banned token in user-facing content (" + id + "); this repo's published surface does not name it. " +
			"Use " + successor + ", or add <!-- docs-lint: allow --> if naming it is genuinely necessary.",
	}
	// Compact, and with HTML escaping off: the patterns and the allow-context
	// regexp carry <, > and & (the escape is an HTML comment), which <-style
	// escaping would render unreadable beside the hand-written entries.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(entry); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// elemSpan is one array element's byte range: start is its first byte, end the
// byte after its last.
type elemSpan struct {
	start, end int
	raw        json.RawMessage
}

// arraySpan locates the banned_tokens array: the byte after its '[', the byte of
// its ']', and every element's range.
type arraySpan struct {
	openEnd    int
	closeStart int
	elems      []elemSpan
}

// locateBannedTokens finds the top-level banned_tokens array by streaming the
// document with the standard decoder and reading its input offsets. Nothing is
// re-marshalled: the caller edits the original bytes inside the located ranges,
// which is what keeps an edit to one entry from churning the whole file.
//
// Two things it must get exactly right, because the editor's every offset depends
// on them. It matches an object KEY, never a string VALUE that happens to spell
// banned_tokens — json.Decoder.Token does not distinguish the two, so the walker
// below tracks it. And a document with MORE THAN ONE top-level banned_tokens key is
// refused rather than resolved: encoding/json keeps the last such key and a byte
// editor finds the first, so the linter would enforce one array while the verb edited
// another — a ban reported as written and enforced by nobody. Neither reading is
// safe to prefer, so a human resolves it.
func locateBannedTokens(data []byte) (arraySpan, error) {
	malformed := func(detail string) (arraySpan, error) {
		return arraySpan{}, fmt.Errorf("%w: %s %s", ErrMalformedStore, PublicConfigRelPath, detail)
	}
	switch n := countBannedTokensKeys(data); {
	case n == 0:
		return malformed("has no top-level banned_tokens array")
	case n > 1:
		return malformed("declares the top-level banned_tokens key more than once; encoding/json would keep the last " +
			"and a byte editor the first, so no reading of it is safe — remove the duplicate")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	walker := keyWalker{dec: dec}
	for {
		tok, isKey, depth, err := walker.next()
		if err != nil {
			return malformed("has no top-level banned_tokens array")
		}
		if !isKey || depth != 1 || tok != "banned_tokens" {
			continue
		}
		open, err := dec.Token()
		if err != nil {
			return malformed("has an unreadable banned_tokens value")
		}
		if d, ok := open.(json.Delim); !ok || d != '[' {
			return malformed("has a non-array banned_tokens value")
		}
		span := arraySpan{openEnd: int(dec.InputOffset())}
		for dec.More() {
			before := int(dec.InputOffset())
			var raw json.RawMessage
			if err := dec.Decode(&raw); err != nil {
				return malformed("has an unreadable banned_tokens entry")
			}
			span.elems = append(span.elems, elemSpan{start: valueStart(data, before), end: int(dec.InputOffset()), raw: raw})
		}
		if _, err := dec.Token(); err != nil {
			return malformed("has an unterminated banned_tokens array")
		}
		span.closeStart = int(dec.InputOffset()) - 1
		return span, nil
	}
}

// countBannedTokensKeys counts the top-level banned_tokens OBJECT KEYS. A document
// the decoder cannot finish reading counts what it saw: locateBannedTokens reports
// the parse failure itself.
func countBannedTokensKeys(data []byte) int {
	walker := keyWalker{dec: json.NewDecoder(bytes.NewReader(data))}
	n := 0
	for {
		tok, isKey, depth, err := walker.next()
		if err != nil {
			return n
		}
		if isKey && depth == 1 && tok == "banned_tokens" {
			n++
		}
	}
}

// keyWalker streams json.Decoder tokens while tracking what the decoder does not
// report: whether a string token is an object KEY or a VALUE, and the depth of the
// container holding it. Inside an object, tokens alternate key, value, key, value —
// so one bit per frame is enough, and a container opening in a value position
// consumes its parent's value slot before it becomes a frame of its own.
type keyWalker struct {
	dec   *json.Decoder
	stack []jsonFrame
}

// jsonFrame is one open container: obj distinguishes {} from [], and wantKey says
// the next token in an object is a key.
type jsonFrame struct {
	obj     bool
	wantKey bool
}

// next returns the next token, whether it is an object key, and the depth of the
// container holding it (1 is the document's top-level object).
func (w *keyWalker) next() (tok json.Token, isKey bool, depth int, err error) {
	tok, err = w.dec.Token()
	if err != nil {
		return nil, false, 0, err
	}
	if d, ok := tok.(json.Delim); ok {
		if d == '{' || d == '[' {
			w.consumedValue()
			w.stack = append(w.stack, jsonFrame{obj: d == '{', wantKey: d == '{'})
			return tok, false, len(w.stack) - 1, nil
		}
		if len(w.stack) > 0 {
			w.stack = w.stack[:len(w.stack)-1]
		}
		return tok, false, len(w.stack), nil
	}
	depth = len(w.stack)
	if depth > 0 {
		if top := &w.stack[depth-1]; top.obj && top.wantKey {
			top.wantKey = false
			return tok, true, depth, nil
		}
		w.consumedValue()
	}
	return tok, false, depth, nil
}

// consumedValue records that the innermost object has just taken its value, so the
// next token there is a key again.
func (w *keyWalker) consumedValue() {
	if n := len(w.stack); n > 0 && w.stack[n-1].obj {
		w.stack[n-1].wantKey = true
	}
}

// valueStart advances past the separator whitespace and comma the decoder reports
// as part of an element's leading offset, landing on the value's first byte.
func valueStart(data []byte, at int) int {
	for at < len(data) {
		switch data[at] {
		case ' ', '\t', '\r', '\n', ',':
			at++
		default:
			return at
		}
	}
	return at
}

// elementIndent is the column an inserted entry belongs in: the same one the array's
// last element sits in. When that element shares its line with the array's opening
// bracket — a one-line config, or a compact `"banned_tokens": [{…}]` — the line's
// indent is the ARRAY's, not an element's, so the insertion would land flush against
// the margin (column zero for a one-line file). One level in from the array is the
// right answer for exactly that case, and it is the same rule the empty-array branch
// already uses.
func elementIndent(data []byte, span arraySpan, last elemSpan) string {
	if lineStart(data, last.start) == lineStart(data, span.openEnd) {
		return lineIndent(data, span.openEnd) + "  "
	}
	return lineIndent(data, last.start)
}

// lineStart is the offset of the first byte of the line containing at.
func lineStart(data []byte, at int) int {
	if at > len(data) {
		at = len(data)
	}
	return bytes.LastIndexByte(data[:at], '\n') + 1
}

// lineIndent returns the leading whitespace of the line containing at, so an
// inserted entry lands in the same column as the entries around it.
func lineIndent(data []byte, at int) string {
	if at > len(data) {
		at = len(data)
	}
	start := lineStart(data, at)
	indent := 0
	for start+indent < len(data) && (data[start+indent] == ' ' || data[start+indent] == '\t') {
		indent++
	}
	return string(data[start : start+indent])
}
